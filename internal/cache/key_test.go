package cache

import "testing"

func TestKey_StreamingAndNonStreamingShareACacheEntry(t *testing.T) {
	nonStreaming := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`)
	streaming := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}],"stream":true,"stream_options":{"include_usage":true}}`)

	if Key(nonStreaming) != Key(streaming) {
		t.Error("expected a streaming request and its non-streaming equivalent to normalize to the same cache key")
	}
}

func TestKey_DifferentFieldOrderSameKey(t *testing.T) {
	a := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`)
	b := []byte(`{"messages":[{"role":"user","content":"hi"}],"model":"gpt-4o-mini"}`)

	if Key(a) != Key(b) {
		t.Error("expected key derivation to be independent of the client's JSON field ordering")
	}
}

func TestKey_DifferentContentDifferentKey(t *testing.T) {
	a := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`)
	b := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"bye"}]}`)

	if Key(a) == Key(b) {
		t.Error("expected genuinely different prompts to produce different keys")
	}
}

func TestKey_UnparsableBodyFallsBackRatherThanPanicking(t *testing.T) {
	notJSON := []byte("not json at all")
	if Key(notJSON) == "" {
		t.Error("expected a non-empty fallback key for an unparsable body")
	}
}
