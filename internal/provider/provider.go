// Package provider defines the interface Verigate uses to talk to any
// OpenAI-compatible chat completions endpoint (OpenAI itself, or a local
// Ollama/vLLM server run with --openai-compat).
package provider

import (
	"context"
	"io"
)

type ChatRequest struct {
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Extra    map[string]any `json:"-"` // passthrough fields we don't need to inspect
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	RawJSON   []byte // the untouched provider response body, returned to the client as-is
	Content   string // extracted assistant text, for logging/eval only
	TokensIn  int
	TokensOut int
	CostUSD   float64
	// ProviderName identifies which adapter actually produced this
	// response. Set by every ChatCompletion implementation to its own
	// Name(); needed because Router.Name() can't say in advance which
	// backend in the chain will end up serving a given request — the
	// answer only exists after the call succeeds.
	ProviderName string
}

type Provider interface {
	Name() string
	ChatCompletion(ctx context.Context, rawBody []byte) (*ChatResponse, error)

	// StreamChatCompletion sends a `stream: true` request and returns the
	// live upstream response body for the caller to forward chunk-by-chunk
	// (server-sent events) — it does NOT buffer the stream, unlike
	// ChatCompletion. The caller owns closing the returned ReadCloser.
	StreamChatCompletion(ctx context.Context, rawBody []byte) (io.ReadCloser, error)
}
