// Command replay answers a real question teams actually have: "should we
// switch models?" It takes N historical requests, re-sends their exact
// prompts to a candidate model, and scores BOTH the original response and
// the candidate's response with the same judge rubrics — so the comparison
// is apples-to-apples instead of "the new model felt better in a few
// manual tries."
//
// Usage:
//
//	go run ./cmd/replay --candidate-model openai/gpt-oss-120b --limit 5
//	go run ./cmd/replay --candidate-model openai/gpt-oss-120b --request <request-id>
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/joho/godotenv"

	"github.com/rakshit-gen/verigate/internal/config"
	"github.com/rakshit-gen/verigate/internal/eval"
	"github.com/rakshit-gen/verigate/internal/provider"
	"github.com/rakshit-gen/verigate/internal/store"
)

type replayResult struct {
	prompt                          string
	originalModel, candidateModel   string
	originalScores, candidateScores map[string]float64
}

func main() {
	candidateModel := flag.String("candidate-model", "", "model to replay prompts through (required)")
	requestID := flag.String("request", "", "replay one specific request by ID instead of --limit most recent")
	limit := flag.Int("limit", 5, "number of recent successful, non-cached requests to replay")
	flag.Parse()

	if *candidateModel == "" {
		fmt.Fprintln(os.Stderr, "error: --candidate-model is required")
		flag.Usage()
		os.Exit(1)
	}

	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer st.Close()

	chatProvider := provider.New(cfg, "")
	judgeProvider := provider.New(cfg, "-judge")
	judge := eval.NewJudge(judgeProvider, cfg.JudgeModel)

	targets, err := selectTargets(ctx, st, *requestID, *limit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(targets) == 0 {
		fmt.Println("no eligible requests found to replay (need at least one non-cached, successful request in the DB)")
		return
	}

	fmt.Printf("Replaying %d request(s) through %s...\n\n", len(targets), *candidateModel)

	var results []replayResult
	for _, target := range targets {
		res, err := replayOne(ctx, chatProvider, judge, target, *candidateModel)
		if err != nil {
			fmt.Printf("  [skip] %q -> %v\n", truncate(target.Prompt, 50), err)
			continue
		}
		results = append(results, res)
		fmt.Printf("  scored: %q\n", truncate(target.Prompt, 60))
	}

	printReport(results)
}

func selectTargets(ctx context.Context, st *store.Store, requestID string, limit int) ([]store.RequestRecord, error) {
	if requestID != "" {
		r, err := st.GetRequest(ctx, requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to load request %s: %w", requestID, err)
		}
		return []store.RequestRecord{*r}, nil
	}

	all, err := st.ListRequests(ctx, "", 200) // "" = across all tenants; over-fetch, then filter down to eligible ones below
	if err != nil {
		return nil, fmt.Errorf("failed to list requests: %w", err)
	}
	var targets []store.RequestRecord
	for _, r := range all {
		if r.Status != "ok" || r.CacheHit {
			continue // only real, successful generations are meaningful to replay
		}
		targets = append(targets, r)
		if len(targets) >= limit {
			break
		}
	}
	return targets, nil
}

func replayOne(ctx context.Context, chatProvider provider.Provider, judge *eval.Judge, target store.RequestRecord, candidateModel string) (replayResult, error) {
	body := map[string]any{
		"model":    candidateModel,
		"messages": []map[string]string{{"role": "user", "content": target.Prompt}},
	}
	rawBody, _ := json.Marshal(body)

	resp, err := chatProvider.ChatCompletion(ctx, rawBody)
	if err != nil {
		return replayResult{}, fmt.Errorf("candidate call failed: %w", err)
	}

	res := replayResult{
		prompt:          target.Prompt,
		originalModel:   target.Model,
		candidateModel:  candidateModel,
		originalScores:  map[string]float64{},
		candidateScores: map[string]float64{},
	}
	for _, rubric := range eval.Rubrics {
		if v, err := judge.Score(ctx, rubric, target.Prompt, target.Response); err == nil {
			res.originalScores[rubric.Name] = v.Score
		}
		if v, err := judge.Score(ctx, rubric, target.Prompt, resp.Content); err == nil {
			res.candidateScores[rubric.Name] = v.Score
		}
	}
	return res, nil
}

func printReport(results []replayResult) {
	if len(results) == 0 {
		fmt.Println("\nnothing to report — every replay attempt failed.")
		return
	}

	rubricNames := make([]string, 0, len(eval.Rubrics))
	for _, r := range eval.Rubrics {
		rubricNames = append(rubricNames, r.Name)
	}
	sort.Strings(rubricNames)

	fmt.Printf("\n%-60s", "PROMPT")
	for _, name := range rubricNames {
		fmt.Printf("  %-22s", name)
	}
	fmt.Println()

	originalTotals := map[string]float64{}
	candidateTotals := map[string]float64{}
	n := 0

	for _, res := range results {
		fmt.Printf("%-60s\n", truncate(res.prompt, 60))
		fmt.Printf("  %-58s", "  "+res.originalModel+" (original)")
		for _, name := range rubricNames {
			fmt.Printf("  %-22.2f", res.originalScores[name])
			originalTotals[name] += res.originalScores[name]
		}
		fmt.Println()
		fmt.Printf("  %-58s", "  "+res.candidateModel+" (candidate)")
		for _, name := range rubricNames {
			score := res.candidateScores[name]
			delta := score - res.originalScores[name]
			marker := " "
			if delta > 0.05 {
				marker = "+"
			} else if delta < -0.05 {
				marker = "-"
			}
			fmt.Printf("  %-22s", fmt.Sprintf("%.2f %s", score, marker))
			candidateTotals[name] += score
		}
		fmt.Print("\n\n")
		n++
	}

	fmt.Println("── averages ─────────────────────────────────────────────")
	for _, name := range rubricNames {
		origAvg := originalTotals[name] / float64(n)
		candAvg := candidateTotals[name] / float64(n)
		verdict := "≈ same"
		if candAvg-origAvg > 0.03 {
			verdict = "▲ candidate better"
		} else if origAvg-candAvg > 0.03 {
			verdict = "▼ candidate worse"
		}
		fmt.Printf("%-22s original %.2f  →  candidate %.2f   %s\n", name, origAvg, candAvg, verdict)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
