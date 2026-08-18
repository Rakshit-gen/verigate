package cache

import (
	"testing"
	"time"
)

func TestCosineSimilarity(t *testing.T) {
	cases := []struct {
		name     string
		a, b     []float32
		wantNear float64
	}{
		{"identical vectors", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal vectors", []float32{1, 0}, []float32{0, 1}, 0.0},
		{"opposite vectors", []float32{1, 0}, []float32{-1, 0}, -1.0},
		{"scaled but same direction", []float32{2, 0}, []float32{10, 0}, 1.0},
		{"mismatched lengths", []float32{1, 2, 3}, []float32{1, 2}, 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CosineSimilarity(tc.a, tc.b)
			if diff := got - tc.wantNear; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("CosineSimilarity(%v, %v) = %v, want ~%v", tc.a, tc.b, got, tc.wantNear)
			}
		})
	}
}

func TestSemanticIndex_FindsMatchAboveThreshold(t *testing.T) {
	idx := NewSemanticIndex(100)
	idx.Add([]float32{1, 0, 0}, "key-a", time.Minute)
	idx.Add([]float32{0, 1, 0}, "key-b", time.Minute)

	// A query nearly identical to "key-a"'s embedding should match it, not "key-b".
	key, sim, found := idx.BestMatch([]float32{0.98, 0.02, 0}, 0.9)
	if !found {
		t.Fatalf("expected a match above threshold, got none (sim=%v)", sim)
	}
	if key != "key-a" {
		t.Errorf("expected best match to be key-a, got %q (sim=%v)", key, sim)
	}
}

func TestSemanticIndex_NoMatchBelowThreshold(t *testing.T) {
	idx := NewSemanticIndex(100)
	idx.Add([]float32{1, 0, 0}, "key-a", time.Minute)

	// Orthogonal query — nothing should be considered similar enough.
	_, _, found := idx.BestMatch([]float32{0, 1, 0}, 0.9)
	if found {
		t.Fatal("expected no match for an orthogonal (unrelated) query")
	}
}

func TestSemanticIndex_ExpiredEntriesArePruned(t *testing.T) {
	idx := NewSemanticIndex(100)
	idx.Add([]float32{1, 0, 0}, "expired-key", -time.Second) // already expired
	idx.Add([]float32{1, 0, 0}, "live-key", time.Minute)

	key, _, found := idx.BestMatch([]float32{1, 0, 0}, 0.5)
	if !found {
		t.Fatal("expected the live entry to still match")
	}
	if key != "live-key" {
		t.Errorf("expected the expired entry to be skipped in favor of the live one, got %q", key)
	}
	if len(idx.entries) != 1 {
		t.Errorf("expected the expired entry to be pruned from the index, %d entries remain", len(idx.entries))
	}
}

func TestSemanticIndex_EvictsOldestBeyondMaxEntries(t *testing.T) {
	idx := NewSemanticIndex(2)
	idx.Add([]float32{1, 0}, "first", time.Minute)
	idx.Add([]float32{1, 0}, "second", time.Minute)
	idx.Add([]float32{1, 0}, "third", time.Minute)

	if len(idx.entries) != 2 {
		t.Fatalf("expected index capped at 2 entries, got %d", len(idx.entries))
	}
	if idx.entries[0].cacheKey != "second" || idx.entries[1].cacheKey != "third" {
		t.Errorf("expected oldest entry evicted, kept %q and %q", idx.entries[0].cacheKey, idx.entries[1].cacheKey)
	}
}
