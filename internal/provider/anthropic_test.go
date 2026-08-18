package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests run against an httptest server shaped exactly like
// Anthropic's real Messages API (verified against Anthropic's published
// request/response/SSE examples), not against api.anthropic.com — proving
// the translation logic is correct without needing a live API key. Live
// end-to-end verification against the real endpoint still needs
// ANTHROPIC_API_KEY set and hasn't been run in this environment.

func TestAnthropic_ChatCompletion_TranslatesBothDirections(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Errorf("expected path /messages, got %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("expected x-api-key header, got %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicAPIVersion {
			t.Errorf("expected anthropic-version %q, got %q", anthropicAPIVersion, got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode translated request body: %v", err)
		}
		if body["system"] != "Be terse." {
			t.Errorf("expected system message to be extracted to top-level `system`, got %v", body["system"])
		}
		if mt, ok := body["max_tokens"].(float64); !ok || mt != defaultMaxTokens {
			t.Errorf("expected max_tokens to default to %d when caller omits it, got %v", defaultMaxTokens, body["max_tokens"])
		}
		msgs, ok := body["messages"].([]any)
		if !ok || len(msgs) != 1 {
			t.Fatalf("expected exactly one non-system message forwarded, got %v", body["messages"])
		}

		fmt.Fprint(w, `{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Hello there!"}],
			"model": "claude-test",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`)
	}))
	defer srv.Close()

	a := NewAnthropic("test-key", srv.URL)
	openAIRequestBody := []byte(`{
		"model": "claude-test",
		"messages": [
			{"role": "system", "content": "Be terse."},
			{"role": "user", "content": "Say hi."}
		]
	}`)

	resp, err := a.ChatCompletion(context.Background(), openAIRequestBody)
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Content != "Hello there!" {
		t.Errorf("expected translated content %q, got %q", "Hello there!", resp.Content)
	}
	if resp.TokensIn != 10 || resp.TokensOut != 5 {
		t.Errorf("expected tokens 10/5, got %d/%d", resp.TokensIn, resp.TokensOut)
	}

	// The whole point: a real OpenAI-compatible client parsing Verigate's
	// RawJSON should get a normal chat.completion response, with no idea
	// the actual backend was Anthropic.
	var openAIShaped struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(resp.RawJSON, &openAIShaped); err != nil {
		t.Fatalf("RawJSON did not parse as OpenAI-shaped JSON: %v", err)
	}
	if len(openAIShaped.Choices) != 1 || openAIShaped.Choices[0].Message.Content != "Hello there!" {
		t.Errorf("RawJSON's choices[0].message.content did not round-trip correctly: %s", resp.RawJSON)
	}
}

func TestAnthropic_StreamChatCompletion_TranslatesSSEEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		events := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-test","usage":{"input_tokens":12,"output_tokens":1}}}

`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}

`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" there"}}

`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}

`,
			`event: message_stop
data: {"type":"message_stop"}

`,
		}
		for _, e := range events {
			fmt.Fprint(w, e)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	a := NewAnthropic("test-key", srv.URL)
	openAIRequestBody := []byte(`{"model":"claude-test","stream":true,"messages":[{"role":"user","content":"Say hi."}]}`)

	rc, err := a.StreamChatCompletion(context.Background(), openAIRequestBody)
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}
	defer rc.Close()

	content, tokensIn, tokensOut, sawDone := readTranslatedOpenAISSE(t, rc)
	if content != "Hi there" {
		t.Errorf("expected accumulated content %q, got %q", "Hi there", content)
	}
	if tokensIn != 12 || tokensOut != 2 {
		t.Errorf("expected tokens 12/2, got %d/%d", tokensIn, tokensOut)
	}
	if !sawDone {
		t.Error("expected the translated stream to end with data: [DONE]")
	}
}

// readTranslatedOpenAISSE is a minimal, test-local OpenAI SSE reader —
// deliberately not importing internal/router (which already imports this
// package, so importing it back here would be a cycle) to parse
// translateAnthropicStreamToOpenAI's output the same way a real client
// would.
func readTranslatedOpenAISSE(t *testing.T, r io.Reader) (content string, tokensIn, tokensOut int, sawDone bool) {
	t.Helper()
	var sb strings.Builder
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			sawDone = true
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("failed to parse translated chunk %q: %v", payload, err)
		}
		if len(chunk.Choices) > 0 {
			sb.WriteString(chunk.Choices[0].Delta.Content)
		}
		if chunk.Usage != nil {
			tokensIn, tokensOut = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	return sb.String(), tokensIn, tokensOut, sawDone
}
