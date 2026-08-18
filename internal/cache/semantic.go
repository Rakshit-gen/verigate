package cache

import (
	"math"
	"sync"
	"time"
)

// SemanticIndex is a bounded, in-process, brute-force nearest-neighbor
// index over cached prompts' embeddings. That's a deliberate scoping
// choice, not an oversight: a real ANN index (pgvector, an HNSW library)
// earns its complexity at tens of thousands of entries — at the size of a
// short-TTL response cache (hundreds of entries, expiring in minutes),
// comparing against every live entry is sub-millisecond and needs zero
// extra infrastructure. Revisit if MaxEntries needs to grow by orders of
// magnitude.
type SemanticIndex struct {
	mu         sync.Mutex
	entries    []semanticEntry
	maxEntries int
}

type semanticEntry struct {
	embedding []float32
	cacheKey  string
	expiresAt time.Time
}

func NewSemanticIndex(maxEntries int) *SemanticIndex {
	return &SemanticIndex{maxEntries: maxEntries}
}

// Add records an embedding against the exact-match Redis key that holds
// its response, so a future semantic hit can fetch the value through the
// same Get path a cache miss would have populated.
func (idx *SemanticIndex) Add(embedding []float32, cacheKey string, ttl time.Duration) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.entries = append(idx.entries, semanticEntry{
		embedding: embedding,
		cacheKey:  cacheKey,
		expiresAt: time.Now().Add(ttl),
	})
	if over := len(idx.entries) - idx.maxEntries; over > 0 {
		idx.entries = idx.entries[over:] // drop oldest — insertion order doubles as a rough LRU
	}
}

// BestMatch returns the cache key of the closest embedding at or above
// threshold, pruning expired entries as it scans so the index doesn't
// silently outlive its own TTL'd Redis values.
func (idx *SemanticIndex) BestMatch(embedding []float32, threshold float64) (cacheKey string, similarity float64, found bool) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	now := time.Now()
	live := idx.entries[:0]
	bestSim := -1.0
	for _, e := range idx.entries {
		if now.After(e.expiresAt) {
			continue // pruned
		}
		live = append(live, e)

		sim := CosineSimilarity(embedding, e.embedding)
		if sim > bestSim {
			bestSim = sim
			cacheKey = e.cacheKey
		}
	}
	idx.entries = live

	if bestSim >= threshold {
		return cacheKey, bestSim, true
	}
	return "", bestSim, false
}

// CosineSimilarity returns a value in [-1, 1]; 1 means identical direction.
// Text embeddings from a real model are never the zero vector, so no
// divide-by-zero guard is needed for that case — it's guarded anyway
// because a bug elsewhere producing a zero vector should degrade to "no
// match" rather than panic.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
