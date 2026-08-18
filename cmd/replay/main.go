// Command replay is the CLI front end for internal/replay — see that
// package's doc comment for what it does and why.
//
// Usage:
//
//	go run ./cmd/replay --candidate-model openai/gpt-oss-120b --limit 5
//	go run ./cmd/replay --candidate-model openai/gpt-oss-120b --request <request-id>
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/joho/godotenv"

	"github.com/rakshit-gen/verigate/internal/config"
	"github.com/rakshit-gen/verigate/internal/eval"
	"github.com/rakshit-gen/verigate/internal/provider"
	"github.com/rakshit-gen/verigate/internal/replay"
	"github.com/rakshit-gen/verigate/internal/store"
)

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

	targets, err := replay.SelectTargets(ctx, st, *requestID, *limit)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(targets) == 0 {
		fmt.Println("no eligible requests found to replay (need at least one non-cached, successful request in the DB)")
		return
	}

	fmt.Printf("Replaying %d request(s) through %s...\n\n", len(targets), *candidateModel)

	var results []replay.Result
	for _, target := range targets {
		res, err := replay.One(ctx, chatProvider, judge, target, *candidateModel)
		if err != nil {
			fmt.Printf("  [skip] %q -> %v\n", truncate(target.Prompt, 50), err)
			continue
		}
		results = append(results, res)
		fmt.Printf("  scored: %q\n", truncate(target.Prompt, 60))
	}

	printReport(results)
}

func printReport(results []replay.Result) {
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
		fmt.Printf("%-60s\n", truncate(res.Prompt, 60))
		fmt.Printf("  %-58s", "  "+res.OriginalModel+" (original)")
		for _, name := range rubricNames {
			fmt.Printf("  %-22.2f", res.OriginalScores[name])
			originalTotals[name] += res.OriginalScores[name]
		}
		fmt.Println()
		fmt.Printf("  %-58s", "  "+res.CandidateModel+" (candidate)")
		for _, name := range rubricNames {
			score := res.CandidateScores[name]
			delta := score - res.OriginalScores[name]
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
