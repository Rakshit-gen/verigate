// Package replay answers a real question teams actually have: "should we
// switch models?" It re-sends historical prompts to a candidate model and
// scores both the original and candidate responses with the same judge
// rubrics, so the comparison is apples-to-apples. Shared by cmd/replay
// (CLI) and the dashboard's POST /api/replay — one implementation, two
// front ends.
package replay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rakshit-gen/verigate/internal/eval"
	"github.com/rakshit-gen/verigate/internal/provider"
	"github.com/rakshit-gen/verigate/internal/store"
)

type Result struct {
	Prompt          string             `json:"prompt"`
	OriginalModel   string             `json:"original_model"`
	CandidateModel  string             `json:"candidate_model"`
	OriginalScores  map[string]float64 `json:"original_scores"`
	CandidateScores map[string]float64 `json:"candidate_scores"`
}

// SelectTargets returns the requests to replay: one specific request by ID,
// or the `limit` most recent successful, non-cached requests (cache hits
// and errors aren't meaningful to replay — there's no real generation to
// compare against).
func SelectTargets(ctx context.Context, st *store.Store, requestID string, limit int) ([]store.RequestRecord, error) {
	if requestID != "" {
		r, err := st.GetRequest(ctx, requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to load request %s: %w", requestID, err)
		}
		return []store.RequestRecord{*r}, nil
	}

	all, err := st.ListRequests(ctx, store.Scope{All: true}, 200) // admin operation — over-fetch across all tenants, then filter down to eligible ones below
	if err != nil {
		return nil, fmt.Errorf("failed to list requests: %w", err)
	}
	var targets []store.RequestRecord
	for _, r := range all {
		if r.Status != "ok" || r.CacheHit {
			continue
		}
		targets = append(targets, r)
		if len(targets) >= limit {
			break
		}
	}
	return targets, nil
}

// One replays a single request through candidateModel and scores both the
// original and candidate responses against every rubric in eval.Rubrics.
func One(ctx context.Context, chatProvider provider.Provider, judge *eval.Judge, target store.RequestRecord, candidateModel string) (Result, error) {
	body := map[string]any{
		"model":    candidateModel,
		"messages": []map[string]string{{"role": "user", "content": target.Prompt}},
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}

	resp, err := chatProvider.ChatCompletion(ctx, rawBody)
	if err != nil {
		return Result{}, fmt.Errorf("candidate call failed: %w", err)
	}

	res := Result{
		Prompt:          target.Prompt,
		OriginalModel:   target.Model,
		CandidateModel:  candidateModel,
		OriginalScores:  map[string]float64{},
		CandidateScores: map[string]float64{},
	}
	for _, rubric := range eval.Rubrics {
		if v, err := judge.Score(ctx, rubric, target.Prompt, target.Response); err == nil {
			res.OriginalScores[rubric.Name] = v.Score
		}
		if v, err := judge.Score(ctx, rubric, target.Prompt, resp.Content); err == nil {
			res.CandidateScores[rubric.Name] = v.Score
		}
	}
	return res, nil
}
