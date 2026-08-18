package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rakshit-gen/verigate/internal/embeddings"
)

type Cache struct {
	rdb *redis.Client
	ttl time.Duration

	// embedder is nil unless semantic caching is configured (needs a real
	// embeddings-capable key — see config.EmbeddingAPIKey). A nil embedder
	// means the cache behaves exactly like exact-match-only tonight's
	// version did — semantic caching degrades to "off", not to an error.
	embedder    embeddings.Embedder
	semantic    *SemanticIndex
	semanticMin float64
}

func New(addr string) *Cache {
	return &Cache{
		rdb: redis.NewClient(&redis.Options{Addr: addr}),
		ttl: 15 * time.Minute,
	}
}

// EnableSemanticCache turns on the embedding-based similarity layer on top
// of the existing exact-match cache. Call it after New; leaving it unused
// (the common case on providers without an embeddings API, like Groq) is
// the supported default, not a degraded mode.
func (c *Cache) EnableSemanticCache(embedder embeddings.Embedder, threshold float64, maxEntries int) {
	c.embedder = embedder
	c.semantic = NewSemanticIndex(maxEntries)
	c.semanticMin = threshold
}

func (c *Cache) SemanticEnabled() bool { return c.embedder != nil }

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Key hashes a normalized request body: identical model+messages+params
// produce the same key regardless of transport-only differences. Stripping
// `stream`/`stream_options` before hashing is what lets a streaming request
// and a non-streaming request for the same underlying prompt share one
// cache entry — the response content doesn't depend on how it was
// delivered, so the cache key shouldn't either. Go's json.Marshal sorts
// map keys alphabetically, so this also makes the key independent of the
// client's original field ordering, which the previous verbatim-hash
// version didn't.
func Key(rawBody []byte) string {
	var m map[string]any
	if err := json.Unmarshal(rawBody, &m); err != nil {
		// not JSON we can normalize — fall back to hashing verbatim rather
		// than failing the request over a body we can't parse.
		sum := sha256.Sum256(rawBody)
		return "verigate:chat:" + hex.EncodeToString(sum[:])
	}
	delete(m, "stream")
	delete(m, "stream_options")
	normalized, err := json.Marshal(m)
	if err != nil {
		sum := sha256.Sum256(rawBody)
		return "verigate:chat:" + hex.EncodeToString(sum[:])
	}
	sum := sha256.Sum256(normalized)
	return "verigate:chat:" + hex.EncodeToString(sum[:])
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return val, true
}

func (c *Cache) Set(ctx context.Context, key string, value []byte) error {
	return c.rdb.Set(ctx, key, value, c.ttl).Err()
}

// SemanticLookup embeds prompt and searches the in-process similarity
// index for a close-enough previous prompt, returning the exact-match
// Redis key that holds ITS response — callers fetch the actual bytes via
// the normal Get, so there is exactly one code path that reads cached
// response bytes regardless of how the match was found. Returns
// found=false immediately (no embedding call) when semantic caching isn't
// enabled, or when the embedding call itself fails — a slow/broken
// embeddings backend degrades to "cache miss, call the provider", never
// to a hard error on the request path.
func (c *Cache) SemanticLookup(ctx context.Context, prompt string) (matchedKey string, similarity float64, found bool) {
	if !c.SemanticEnabled() || prompt == "" {
		return "", 0, false
	}
	vec, err := c.embedder.Embed(ctx, prompt)
	if err != nil {
		log.Printf("semantic cache: embedding lookup failed, falling back to exact-match only: %v", err)
		return "", 0, false
	}
	return c.semantic.BestMatch(vec, c.semanticMin)
}

// IndexForSemanticLookup embeds prompt and records it against exactKey so
// future semantically-similar prompts can find this response. Called
// after a successful exact-match Set. Best-effort: an embedding failure
// here only costs a future semantic-cache opportunity, not the current
// request.
func (c *Cache) IndexForSemanticLookup(ctx context.Context, prompt, exactKey string) {
	if !c.SemanticEnabled() || prompt == "" {
		return
	}
	vec, err := c.embedder.Embed(ctx, prompt)
	if err != nil {
		log.Printf("semantic cache: failed to index prompt for future semantic hits: %v", err)
		return
	}
	c.semantic.Add(vec, exactKey, c.ttl)
}
