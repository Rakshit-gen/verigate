package provider

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeProvider is a controllable Provider for exercising the router/breaker
// without any real network calls or API keys — every scenario below is
// pure logic, deterministic, and fast.
type fakeProvider struct {
	name     string
	mu       sync.Mutex
	calls    int
	behavior func(callN int) error // nil error = succeed on this call
	delay    time.Duration         // simulated call latency, for reordering tests
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) callN() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.calls
}

func (f *fakeProvider) ChatCompletion(ctx context.Context, rawBody []byte) (*ChatResponse, error) {
	n := f.callN()
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if err := f.behavior(n); err != nil {
		return nil, err
	}
	return &ChatResponse{Content: "ok", ProviderName: f.name}, nil
}

func (f *fakeProvider) StreamChatCompletion(ctx context.Context, rawBody []byte) (io.ReadCloser, error) {
	n := f.callN()
	if err := f.behavior(n); err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader("data: ok\n\n")), nil
}

func alwaysFail(n int) error    { return errors.New("simulated failure") }
func alwaysSucceed(n int) error { return nil }

func TestRouter_FallsForwardToSecondaryOnFailure(t *testing.T) {
	primary := &fakeProvider{name: "primary", behavior: alwaysFail}
	secondary := &fakeProvider{name: "secondary", behavior: alwaysSucceed}

	r := NewRouter(
		RouterEntry{Provider: primary, FailureThreshold: 3, Cooldown: time.Minute},
		RouterEntry{Provider: secondary, FailureThreshold: 3, Cooldown: time.Minute},
	)

	resp, err := r.ChatCompletion(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if resp.ProviderName != "secondary" {
		t.Errorf("expected the response to be attributed to secondary, got %q", resp.ProviderName)
	}
}

func TestRouter_OpensBreakerAfterThreshold_ThenSkipsPrimary(t *testing.T) {
	primary := &fakeProvider{name: "primary", behavior: alwaysFail}
	secondary := &fakeProvider{name: "secondary", behavior: alwaysSucceed}

	r := NewRouter(
		RouterEntry{Provider: primary, FailureThreshold: 2, Cooldown: time.Hour}, // long cooldown: won't accidentally half-open mid-test
		RouterEntry{Provider: secondary, FailureThreshold: 2, Cooldown: time.Hour},
	)

	// First two calls: primary is tried and fails both times (threshold=2),
	// opening its breaker.
	for i := 0; i < 2; i++ {
		if _, err := r.ChatCompletion(context.Background(), nil); err != nil {
			t.Fatalf("call %d: expected fallback to secondary to succeed, got %v", i, err)
		}
	}
	if primary.calls != 2 {
		t.Fatalf("expected primary to have been tried twice, got %d", primary.calls)
	}

	// Third call: primary's breaker should now be open, so it must be
	// skipped entirely (call count stays at 2) and go straight to secondary.
	if _, err := r.ChatCompletion(context.Background(), nil); err != nil {
		t.Fatalf("expected third call to succeed via secondary, got %v", err)
	}
	if primary.calls != 2 {
		t.Errorf("expected primary to be skipped once its breaker opened, but it was called again (calls=%d)", primary.calls)
	}
}

func TestRouter_HalfOpenRecoversAfterCooldown(t *testing.T) {
	var primaryHealthy bool
	primary := &fakeProvider{name: "primary", behavior: func(n int) error {
		if primaryHealthy {
			return nil
		}
		return errors.New("still down")
	}}
	secondary := &fakeProvider{name: "secondary", behavior: alwaysSucceed}

	shortCooldown := 20 * time.Millisecond
	r := NewRouter(
		RouterEntry{Provider: primary, FailureThreshold: 1, Cooldown: shortCooldown},
		RouterEntry{Provider: secondary, FailureThreshold: 1, Cooldown: time.Hour},
	)

	// Primary fails once, opening its breaker immediately (threshold=1).
	resp, err := r.ChatCompletion(context.Background(), nil)
	if err != nil || resp.ProviderName != "secondary" {
		t.Fatalf("expected fallback to secondary, got resp=%v err=%v", resp, err)
	}

	// Simulate the provider recovering, and wait out the cooldown.
	primaryHealthy = true
	time.Sleep(shortCooldown * 2)

	// Breaker should now be half-open and allow exactly one probe call to
	// primary, which succeeds and closes the breaker again.
	resp, err = r.ChatCompletion(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected the half-open probe to succeed, got %v", err)
	}
	if resp.ProviderName != "primary" {
		t.Errorf("expected the recovered primary to serve this request, got %q", resp.ProviderName)
	}
}

func TestRouter_AllProvidersFailingReturnsError(t *testing.T) {
	a := &fakeProvider{name: "a", behavior: alwaysFail}
	b := &fakeProvider{name: "b", behavior: alwaysFail}
	r := NewRouter(
		RouterEntry{Provider: a, FailureThreshold: 5, Cooldown: time.Hour},
		RouterEntry{Provider: b, FailureThreshold: 5, Cooldown: time.Hour},
	)

	_, err := r.ChatCompletion(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error when every provider in the chain fails")
	}
}

func TestRouter_StreamChatCompletion_FallsForward(t *testing.T) {
	primary := &fakeProvider{name: "primary", behavior: alwaysFail}
	secondary := &fakeProvider{name: "secondary", behavior: alwaysSucceed}
	r := NewRouter(
		RouterEntry{Provider: primary, FailureThreshold: 3, Cooldown: time.Minute},
		RouterEntry{Provider: secondary, FailureThreshold: 3, Cooldown: time.Minute},
	)

	rc, err := r.StreamChatCompletion(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected streaming fallback to succeed, got %v", err)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)
	if string(body) != "data: ok\n\n" {
		t.Errorf("expected the secondary's stream body, got %q", body)
	}
}

func TestRouter_PrefersMeasuredFasterProviderOnceBothHaveData(t *testing.T) {
	slow := &fakeProvider{name: "slow", behavior: alwaysSucceed, delay: 40 * time.Millisecond}
	fast := &fakeProvider{name: "fast", behavior: alwaysSucceed, delay: 5 * time.Millisecond}

	// Declared order deliberately puts the SLOW one first — proves the
	// router is actually reordering by measured latency, not just
	// following config order.
	r := NewRouter(
		RouterEntry{Provider: slow, FailureThreshold: 3, Cooldown: time.Minute},
		RouterEntry{Provider: fast, FailureThreshold: 3, Cooldown: time.Minute},
	)

	// Seed both trackers with the measurements a real round of successful
	// calls would have produced, without depending on attempt order to get
	// there — the router only returns from ChatCompletion on the FIRST
	// success, so a single real call can't warm up both entries. This
	// tests the ordering logic in isolation from the warm-up mechanics.
	for i := range r.chain {
		switch r.chain[i].provider.Name() {
		case "slow":
			r.chain[i].latency.Record(40 * time.Millisecond)
		case "fast":
			r.chain[i].latency.Record(5 * time.Millisecond)
		}
	}

	callsBefore := fast.calls
	if _, err := r.ChatCompletion(context.Background(), nil); err != nil {
		t.Fatalf("expected router call to succeed, got %v", err)
	}
	if fast.calls != callsBefore+1 {
		t.Errorf("expected the measured-faster provider to be tried first once both have data, but it was not called (fast.calls=%d)", fast.calls)
	}
}
