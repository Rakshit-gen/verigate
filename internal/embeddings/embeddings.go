// Package embeddings provides the text->vector step semantic caching needs.
// It's a separate package (not folded into internal/provider) because it
// isn't every provider's job — Groq, for instance, serves chat completions
// but not embeddings, so a Verigate deployment on Groq needs a distinct,
// independently-configured embeddings backend (typically OpenAI's, even
// while chat traffic goes elsewhere). See config.EmbeddingAPIKey.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type OpenAICompat struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewOpenAICompat(apiKey, baseURL, model string) *OpenAICompat {
	return &OpenAICompat{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (o *OpenAICompat) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(map[string]string{"model": o.model, "input": text})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embeddings request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("embeddings endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("embeddings response had no data")
	}
	return parsed.Data[0].Embedding, nil
}
