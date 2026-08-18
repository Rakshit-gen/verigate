package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Anthropic's Messages API is not OpenAI-wire-compatible: different
// endpoint, different auth header, a separate top-level `system` field
// instead of a system-role message, a required `max_tokens`, and a
// completely different response and SSE event shape. Verigate's public
// contract is "point your base_url here, it's OpenAI-shaped" — so this
// adapter's whole job is translating both directions, not just forwarding
// bytes with a different URL like OpenAICompat does. That translation is
// the actual proof the Provider interface abstracts something real.
type Anthropic struct {
	apiKey       string
	baseURL      string
	version      string
	client       *http.Client
	streamClient *http.Client
}

const anthropicAPIVersion = "2023-06-01" // stable, documented header value — not a model name, so no "did this get deprecated" risk

func NewAnthropic(apiKey, baseURL string) *Anthropic {
	return &Anthropic{
		apiKey:       apiKey,
		baseURL:      baseURL,
		version:      anthropicAPIVersion,
		client:       &http.Client{Timeout: 60 * time.Second},
		streamClient: &http.Client{},
	}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) ChatCompletion(ctx context.Context, rawBody []byte) (*ChatResponse, error) {
	anthropicBody, model, err := translateRequestToAnthropic(rawBody, false)
	if err != nil {
		return nil, fmt.Errorf("anthropic: translating request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/messages", bytes.NewReader(anthropicBody))
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provider anthropic: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("provider anthropic returned %d: %s", resp.StatusCode, string(body))
	}

	openAIBody, content, tokensIn, tokensOut, err := translateResponseToOpenAI(body, model)
	if err != nil {
		// same fail-open contract as OpenAICompat: don't break the caller
		// over a response shape we didn't expect, just skip enrichment.
		return &ChatResponse{RawJSON: body, ProviderName: a.Name()}, nil
	}

	return &ChatResponse{
		RawJSON:      openAIBody,
		Content:      content,
		TokensIn:     tokensIn,
		TokensOut:    tokensOut,
		CostUSD:      EstimateCost(model, tokensIn, tokensOut),
		ProviderName: a.Name(),
	}, nil
}

func (a *Anthropic) StreamChatCompletion(ctx context.Context, rawBody []byte) (io.ReadCloser, error) {
	anthropicBody, model, err := translateRequestToAnthropic(rawBody, true)
	if err != nil {
		return nil, fmt.Errorf("anthropic: translating request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/messages", bytes.NewReader(anthropicBody))
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := a.streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provider anthropic: %w", err)
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider anthropic returned %d: %s", resp.StatusCode, string(body))
	}

	// The router only knows how to forward and parse OpenAI-shaped SSE
	// chunks, so translation has to happen inline with the stream, not
	// after it — a goroutine reads Anthropic's events off resp.Body and
	// writes OpenAI-shaped "data: ..." lines into the pipe as they arrive,
	// which the router then reads exactly like it reads OpenAICompat's
	// stream. Neither router.go nor streaming.go needed to change for this.
	pr, pw := io.Pipe()
	go func() {
		defer resp.Body.Close()
		err := translateAnthropicStreamToOpenAI(resp.Body, pw, model)
		pw.CloseWithError(err) //nolint:errcheck // CloseWithError(nil) is the normal-completion case
	}()
	return pr, nil
}

func (a *Anthropic) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", a.version)
}

// --- request translation: OpenAI-shaped in, Anthropic-shaped out ---

type openAIRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature *float64        `json:"temperature"`
	Messages    []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

const defaultMaxTokens = 1024 // Anthropic requires max_tokens; OpenAI callers often omit it

func translateRequestToAnthropic(rawBody []byte, stream bool) (out []byte, model string, err error) {
	var req openAIRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		return nil, "", err
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	var systemParts []string
	var messages []openAIMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, m.Content)
			continue
		}
		messages = append(messages, m)
	}

	body := map[string]any{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"messages":   messages,
		"stream":     stream,
	}
	if len(systemParts) > 0 {
		body["system"] = strings.Join(systemParts, "\n\n")
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}

	out, err = json.Marshal(body)
	return out, req.Model, err
}

// --- non-streaming response translation: Anthropic-shaped in, OpenAI-shaped out ---

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func translateResponseToOpenAI(body []byte, model string) (openAIJSON []byte, content string, tokensIn, tokensOut int, err error) {
	var ar anthropicResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, "", 0, 0, err
	}

	var text strings.Builder
	for _, block := range ar.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	content = text.String()

	out := map[string]any{
		"id":      "verigate-anthropic-translated",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       map[string]string{"role": "assistant", "content": content},
				"finish_reason": mapStopReason(ar.StopReason),
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     ar.Usage.InputTokens,
			"completion_tokens": ar.Usage.OutputTokens,
			"total_tokens":      ar.Usage.InputTokens + ar.Usage.OutputTokens,
		},
	}
	openAIJSON, err = json.Marshal(out)
	return openAIJSON, content, ar.Usage.InputTokens, ar.Usage.OutputTokens, err
}

func mapStopReason(anthropicReason string) string {
	switch anthropicReason {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		return "stop"
	}
}

// --- streaming translation: Anthropic SSE events in, OpenAI-shaped SSE chunks out ---

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"` // "text_delta" for content, present on content_block_delta
		Text string `json:"text"`
	} `json:"delta"`
	Message struct {
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func translateAnthropicStreamToOpenAI(upstream io.Reader, w io.Writer, model string) error {
	scanner := bufio.NewScanner(upstream)
	var tokensIn, tokensOut int

	writeChunk := func(deltaContent string, usage map[string]int) error {
		chunk := map[string]any{
			"id":      "verigate-anthropic-translated",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]any{
				{"index": 0, "delta": map[string]string{"content": deltaContent}, "finish_reason": nil},
			},
		}
		if usage != nil {
			chunk["usage"] = usage
		}
		payload, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue // Anthropic also sends "event: <type>" lines; the data line carries everything needed
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

		var ev anthropicStreamEvent
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue // skip anything we can't parse rather than aborting the whole stream
		}

		switch ev.Type {
		case "message_start":
			tokensIn = ev.Message.Usage.InputTokens
		case "content_block_delta":
			if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
				if err := writeChunk(ev.Delta.Text, nil); err != nil {
					return err
				}
			}
		case "message_delta":
			if ev.Usage.OutputTokens > 0 {
				tokensOut = ev.Usage.OutputTokens
			}
		case "message_stop":
			if err := writeChunk("", map[string]int{
				"prompt_tokens":     tokensIn,
				"completion_tokens": tokensOut,
				"total_tokens":      tokensIn + tokensOut,
			}); err != nil {
				return err
			}
			_, err := fmt.Fprint(w, "data: [DONE]\n\n")
			return err
		}
	}
	return scanner.Err()
}
