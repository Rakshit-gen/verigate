// Command seed_demo_traffic fires realistic traffic through a running
// Verigate instance, then deliberately injects a batch of garbage
// "responses" straight into the database (bypassing the real provider —
// no need to spend API credits to prove the point) so the eval worker
// scores them low and the dashboard's regression banner flips red on
// camera. Run this after `go run ./cmd/gateway` is up.
//
// Usage: go run ./scripts/seed_demo_traffic.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/rakshit-gen/verigate/internal/store"
)

var goodPrompts = []string{
	"What's the capital of France?",
	"Explain what a hash map is in one sentence.",
	"Summarize the plot of Romeo and Juliet in two sentences.",
	"What does HTTP stand for?",
	"Give me a simple recipe for scrambled eggs.",
	"What's the time complexity of binary search?",
	"Name three primary colors.",
	"What year did World War II end?",
}

func main() {
	_ = godotenv.Load() // same .env the gateway itself reads — keeps model/keys in sync

	baseURL := getenv("VERIGATE_URL", "http://localhost:8080")
	apiKey := getenv("VERIGATE_API_KEY", "dev-local-key")
	dbURL := getenv("DATABASE_URL", "postgres://localhost:5432/verigate?sslmode=disable")
	// Must match whatever CHAT_MODEL_DEFAULT / provider you configured in
	// .env — a model name valid for OpenAI isn't valid for Groq/Ollama and
	// vice versa, so this can't be hardcoded.
	chatModel := getenv("CHAT_MODEL_DEFAULT", "gpt-4o-mini")

	fmt.Printf("phase 1/2: sending normal traffic through the real gateway (model=%s)...\n", chatModel)
	client := &http.Client{Timeout: 30 * time.Second}
	for _, p := range goodPrompts {
		body, _ := json.Marshal(map[string]any{
			"model":    chatModel,
			"messages": []map[string]string{{"role": "user", "content": p}},
		})
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  request failed (is the gateway running?): %v\n", err)
			continue
		}
		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("  sent: %q -> %s: %s\n", p, resp.Status, string(respBody))
			continue
		}
		resp.Body.Close()
		fmt.Printf("  sent: %q -> %s\n", p, resp.Status)
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println("phase 2/2: injecting bad responses directly to trigger the regression banner...")
	ctx := context.Background()
	st, err := store.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("  could not connect to postgres directly: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	badResponses := []struct{ prompt, response string }{
		{"What's the capital of France?", "Bananas are a good source of potassium and grow on trees in tropical climates."},
		{"Explain what a hash map is.", "asdkjalksjd 12312 !!! hash map hash map hash map error error"},
		{"What does HTTP stand for?", "I cannot answer this question because the moon is made of cheese."},
		{"Name three primary colors.", ""},
		{"What year did WWII end?", "The stock market closed higher on Tuesday amid inflation concerns."},
	}
	for _, b := range badResponses {
		id, err := st.InsertRequest(ctx, store.RequestRecord{
			Provider: "openai", Model: "gpt-4o-mini",
			Prompt: b.prompt, Response: b.response,
			LatencyMS: 400, CacheHit: false, Status: "ok",
		})
		if err != nil {
			fmt.Printf("  insert failed: %v\n", err)
			continue
		}
		// force a low score directly rather than waiting on the async judge —
		// this is the "provoke it on camera in under 30 seconds" trick.
		st.InsertEval(ctx, store.EvalRecord{RequestID: id, Rubric: "groundedness", Score: 0.05, Reasoning: "fabricated/off-topic (seeded demo data)"})
		st.InsertEval(ctx, store.EvalRecord{RequestID: id, Rubric: "format_compliance", Score: 0.10, Reasoning: "garbled or empty (seeded demo data)"})
		fmt.Printf("  injected low-quality eval for: %q\n", b.prompt)
	}

	fmt.Println("done. open the dashboard — the regression banner should be red within a few seconds.")
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
