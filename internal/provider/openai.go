package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAICompat talks to any server implementing the OpenAI chat/completions
// contract — OpenAI itself, or a local Ollama/vLLM instance pointed at via
// a different BaseURL. That's the only thing that changes between "real"
// and "local/free" mode.
type OpenAICompat struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
	// streamClient has no fixed Timeout — http.Client's Timeout covers the
	// entire round trip including reading the body, which would cut off a
	// long-running SSE stream. Streaming requests instead live and die with
	// the inbound request's own context (cancelled when the client
	// disconnects), which is the correct lifetime for a proxy.
	streamClient *http.Client
}

func NewOpenAICompat(name, baseURL, apiKey string) *OpenAICompat {
	return &OpenAICompat{
		name:         name,
		baseURL:      baseURL,
		apiKey:       apiKey,
		client:       &http.Client{Timeout: 60 * time.Second},
		streamClient: &http.Client{},
	}
}

func (o *OpenAICompat) Name() string { return o.name }

// per-1M-token USD pricing, deliberately small and easy to extend.
// unknown models cost 0 rather than erroring — cost tracking degrades
// gracefully instead of blocking the request.
var pricePerMillion = map[string][2]float64{
	"gpt-4o-mini": {0.15, 0.60},
	"gpt-4o":      {2.50, 10.00},
}

// EstimateCost is shared by the buffered and streaming code paths so cost
// tracking behaves identically regardless of how the response arrived.
func EstimateCost(model string, tokensIn, tokensOut int) float64 {
	price, ok := pricePerMillion[model]
	if !ok {
		return 0
	}
	return float64(tokensIn)/1_000_000*price[0] + float64(tokensOut)/1_000_000*price[1]
}

func (o *OpenAICompat) ChatCompletion(ctx context.Context, rawBody []byte) (*ChatResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", o.name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("provider %s returned %d: %s", o.name, resp.StatusCode, string(body))
	}

	var parsed struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		// still return the raw body to the client — logging/eval is best-effort,
		// the passthrough contract to the caller is not allowed to break.
		return &ChatResponse{RawJSON: body, ProviderName: o.name}, nil
	}

	content := ""
	if len(parsed.Choices) > 0 {
		content = parsed.Choices[0].Message.Content
	}

	cost := EstimateCost(parsed.Model, parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens)

	return &ChatResponse{
		RawJSON:      body,
		Content:      content,
		ProviderName: o.name,
		TokensIn:     parsed.Usage.PromptTokens,
		TokensOut:    parsed.Usage.CompletionTokens,
		CostUSD:      cost,
	}, nil
}

func (o *OpenAICompat) StreamChatCompletion(ctx context.Context, rawBody []byte) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", o.name, err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider %s returned %d: %s", o.name, resp.StatusCode, string(body))
	}

	return resp.Body, nil
}
