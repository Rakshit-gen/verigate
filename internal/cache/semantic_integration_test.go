package cache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rakshit-gen/verigate/internal/embeddings"
)

// This test exercises the full Cache-level plumbing — embedding call,
// SemanticIndex, and the real local Redis — not just the pure
// CosineSimilarity/SemanticIndex logic covered in semantic_test.go. The
// mock embeddings server below produces a crude bag-of-words vector (word
// presence across a small fixed vocabulary), which is enough to make
// "similar wording -> similar vector -> real cosine similarity" true
// without needing a real embedding model — that's what makes this a wiring
// test, not a claim about real semantic quality. A live OpenAI embeddings
// key would replace the mock but wouldn't change what this test proves
// about Verigate's own code.
func mockEmbeddingsServer(t *testing.T) *httptest.Server {
	t.Helper()
	vocab := []string{"capital", "france", "paris", "bread", "bake", "recipe", "weather", "today"}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lower := strings.ToLower(body.Input)
		vec := make([]float32, len(vocab))
		for i, word := range vocab {
			if strings.Contains(lower, word) {
				vec[i] = 1
			}
		}
		resp := map[string]any{
			"data": []map[string]any{{"embedding": vec}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestCache_SemanticLookup_FullPlumbing(t *testing.T) {
	if err := New("localhost:6379").Ping(context.Background()); err != nil {
		t.Skipf("no local redis available: %v", err)
	}

	srv := mockEmbeddingsServer(t)
	defer srv.Close()

	c := New("localhost:6379")
	embedder := embeddings.NewOpenAICompat("test-key", srv.URL, "test-embed-model")
	c.EnableSemanticCache(embedder, 0.7, 100)

	ctx := context.Background()
	exactKey := "verigate:test:semantic-plumbing:" + t.Name()
	responseBytes := []byte(`{"choices":[{"message":{"content":"Paris is the capital of France."}}]}`)
	defer c.rdb.Del(ctx, exactKey)

	if err := c.Set(ctx, exactKey, responseBytes); err != nil {
		t.Fatalf("Set: %v", err)
	}
	c.IndexForSemanticLookup(ctx, "What is the capital of France?", exactKey)

	t.Run("paraphrased prompt hits the semantic cache", func(t *testing.T) {
		matchedKey, sim, found := c.SemanticLookup(ctx, "capital of France")
		if !found {
			t.Fatalf("expected a semantic match, got none (sim=%v)", sim)
		}
		if matchedKey != exactKey {
			t.Errorf("expected matched key %q, got %q", exactKey, matchedKey)
		}

		val, ok := c.Get(ctx, matchedKey)
		if !ok {
			t.Fatal("expected Get on the matched key to succeed")
		}
		if string(val) != string(responseBytes) {
			t.Errorf("fetched value did not match what was cached: %s", val)
		}
	})

	t.Run("unrelated prompt does not hit the semantic cache", func(t *testing.T) {
		_, _, found := c.SemanticLookup(ctx, "How do I bake bread?")
		if found {
			t.Error("did not expect an unrelated prompt to match")
		}
	})
}

func TestCache_SemanticDisabled_ByDefault(t *testing.T) {
	c := New("localhost:6379")
	if c.SemanticEnabled() {
		t.Error("expected semantic caching to be disabled until EnableSemanticCache is called")
	}
	_, _, found := c.SemanticLookup(context.Background(), "anything")
	if found {
		t.Error("expected SemanticLookup to no-op (found=false) when semantic caching is disabled")
	}
}
