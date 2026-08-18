package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rakshit-gen/verigate/internal/provider"
)

type Judge struct {
	provider provider.Provider
	model    string
}

func NewJudge(p provider.Provider, model string) *Judge {
	return &Judge{provider: p, model: model}
}

type verdict struct {
	Score     float64 `json:"score"`
	Reasoning string  `json:"reasoning"`
}

// Score asks the judge model to grade one (prompt, response) pair against
// a rubric and returns a 0-1 score plus a short reasoning string. It builds
// its own OpenAI-shaped request body and reuses the same Provider interface
// the gateway itself talks to — the judge is just another chat completion.
func (j *Judge) Score(ctx context.Context, r Rubric, userPrompt, assistantResponse string) (verdict, error) {
	body := map[string]any{
		"model": j.model,
		"messages": []map[string]string{
			{"role": "system", "content": r.SystemPrompt},
			{"role": "user", "content": fmt.Sprintf("PROMPT:\n%s\n\nRESPONSE:\n%s", userPrompt, assistantResponse)},
		},
		"temperature": 0,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return verdict{}, err
	}

	resp, err := j.provider.ChatCompletion(ctx, raw)
	if err != nil {
		return verdict{}, err
	}

	var v verdict
	content := bytes.TrimSpace([]byte(resp.Content))
	if err := json.Unmarshal(content, &v); err != nil {
		// judge didn't return clean JSON — fail safe rather than crash the
		// worker pool; a 0.5 with a visible reasoning string is easy to spot
		// and debug in the dashboard instead of silently dropping the sample.
		return verdict{Score: 0.5, Reasoning: "judge returned non-JSON output"}, nil
	}
	return v, nil
}
