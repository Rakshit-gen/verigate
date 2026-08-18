package provider

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// breakerState mirrors the standard three-state circuit breaker: closed
// (normal operation), open (failing, reject fast), half-open (cooldown
// elapsed, let exactly the next call through as a health probe).
type breakerState int

const (
	closed breakerState = iota
	open
	halfOpen
)

type circuitBreaker struct {
	mu                  sync.Mutex
	state               breakerState
	consecutiveFailures int
	openedAt            time.Time
	failureThreshold    int
	cooldown            time.Duration
}

func newCircuitBreaker(failureThreshold int, cooldown time.Duration) *circuitBreaker {
	return &circuitBreaker{failureThreshold: failureThreshold, cooldown: cooldown}
}

// Allow reports whether a call should be attempted right now, transitioning
// open -> half-open once the cooldown has elapsed.
func (b *circuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case closed, halfOpen:
		return true
	case open:
		if time.Since(b.openedAt) >= b.cooldown {
			b.state = halfOpen
			return true
		}
		return false
	}
	return true
}

func (b *circuitBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = closed
	b.consecutiveFailures = 0
}

func (b *circuitBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.consecutiveFailures++
	if b.state == halfOpen || b.consecutiveFailures >= b.failureThreshold {
		b.state = open
		b.openedAt = time.Now()
	}
}

// latencyTracker keeps an exponential moving average of a provider's
// successful-call latency. A provider with no recorded calls yet reads as
// 0, which sorts first (see Router's ordering) — an untested provider gets
// tried before a provider with a known-bad measured latency, rather than
// being permanently stuck at the back of a static list.
type latencyTracker struct {
	mu   sync.Mutex
	ewma float64 // milliseconds; 0 = no data yet
}

const latencyEWMAAlpha = 0.3 // weight on the newest sample; higher = adapts faster, noisier

func (l *latencyTracker) Record(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ms := float64(d.Milliseconds())
	if l.ewma == 0 {
		l.ewma = ms
		return
	}
	l.ewma = latencyEWMAAlpha*ms + (1-latencyEWMAAlpha)*l.ewma
}

func (l *latencyTracker) Value() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ewma
}

type chainEntry struct {
	provider Provider
	breaker  *circuitBreaker
	latency  *latencyTracker
	// declaredOrder preserves the config-specified priority as the tie
	// break when latency data is equal or absent — dynamic reordering
	// augments the declared policy, it doesn't replace it outright.
	declaredOrder int
}

// Router tries a chain of providers, skipping any whose circuit breaker is
// open, and — among the ones currently allowed — trying the
// measured-fastest one first via a live exponential-moving-average of each
// provider's successful-call latency. A provider with no successful calls
// yet is treated as un-measured and tried before any provider with a known
// latency, so a fresh entry gets a chance to prove itself instead of
// starting permanently last. Declared config order is the tie-break, not
// the whole policy — this is the dynamic half of "cost/latency-aware
// routing"; the static half (declare the presumed-cheaper provider first)
// still applies as the starting point and the tie-break.
//
// On any hard failure, the router falls forward to the next entry in the
// (re-ordered) chain, and RecordFailure/RecordSuccess drive the same
// breaker behavior as before.
type Router struct {
	chain []chainEntry
}

// RouterEntry pairs a Provider with the failure count that opens its
// breaker and how long it stays open before a health-check retry.
type RouterEntry struct {
	Provider         Provider
	FailureThreshold int           // e.g. 3
	Cooldown         time.Duration // e.g. 30 * time.Second
}

func NewRouter(entries ...RouterEntry) *Router {
	r := &Router{}
	for i, e := range entries {
		r.chain = append(r.chain, chainEntry{
			provider:      e.Provider,
			breaker:       newCircuitBreaker(e.FailureThreshold, e.Cooldown),
			latency:       &latencyTracker{},
			declaredOrder: i,
		})
	}
	return r
}

// Name reports the highest-priority (declared-order) provider's name as a
// static label for pre-call telemetry — the actual serving provider for a
// given request may differ once breaker state and measured latency are
// taken into account, and is carried on ChatResponse.ProviderName instead.
func (r *Router) Name() string {
	if len(r.chain) == 0 {
		return "router-empty"
	}
	return r.chain[0].provider.Name()
}

// EntryStatus is a point-in-time snapshot of one chain entry — what a
// dashboard needs to answer "which provider is actually serving traffic
// right now, and why."
type EntryStatus struct {
	Name                string  `json:"name"`
	DeclaredOrder       int     `json:"declared_order"`
	BreakerState        string  `json:"breaker_state"` // "closed" | "open" | "half_open"
	ConsecutiveFailures int     `json:"consecutive_failures"`
	MeasuredLatencyMS   float64 `json:"measured_latency_ms"` // 0 = no successful call recorded yet
}

// Status reports every chain entry's current breaker state and measured
// latency, in declared order — this is the only place that state is
// observable from outside the package, since chainEntry/circuitBreaker are
// both unexported.
func (r *Router) Status() []EntryStatus {
	out := make([]EntryStatus, 0, len(r.chain))
	for _, e := range r.chain {
		e.breaker.mu.Lock()
		state := e.breaker.state
		failures := e.breaker.consecutiveFailures
		e.breaker.mu.Unlock()

		out = append(out, EntryStatus{
			Name:                e.provider.Name(),
			DeclaredOrder:       e.declaredOrder,
			BreakerState:        state.String(),
			ConsecutiveFailures: failures,
			MeasuredLatencyMS:   e.latency.Value(),
		})
	}
	return out
}

func (s breakerState) String() string {
	switch s {
	case closed:
		return "closed"
	case open:
		return "open"
	case halfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// attemptOrder returns the chain entries currently allowed to be tried,
// sorted fastest-measured-first (untested entries sort as fastest, per
// latencyTracker's zero-value semantics), falling back to declared order
// on a tie.
func (r *Router) attemptOrder() []chainEntry {
	var candidates []chainEntry
	for _, e := range r.chain {
		if e.breaker.Allow() {
			candidates = append(candidates, e)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		li, lj := candidates[i].latency.Value(), candidates[j].latency.Value()
		if li != lj {
			return li < lj
		}
		return candidates[i].declaredOrder < candidates[j].declaredOrder
	})
	return candidates
}

func (r *Router) ChatCompletion(ctx context.Context, rawBody []byte) (*ChatResponse, error) {
	var lastErr error
	attempted := r.attemptOrder()
	for _, entry := range attempted {
		callStart := time.Now()
		resp, err := entry.provider.ChatCompletion(ctx, rawBody)
		if err != nil {
			entry.breaker.RecordFailure()
			lastErr = err
			continue
		}
		entry.breaker.RecordSuccess()
		entry.latency.Record(time.Since(callStart))
		return resp, nil
	}
	if len(attempted) == 0 {
		return nil, fmt.Errorf("router: every provider in the chain has an open circuit breaker")
	}
	return nil, fmt.Errorf("router: all %d attempted provider(s) failed, last error: %w", len(attempted), lastErr)
}

func (r *Router) StreamChatCompletion(ctx context.Context, rawBody []byte) (io.ReadCloser, error) {
	var lastErr error
	attempted := r.attemptOrder()
	for _, entry := range attempted {
		callStart := time.Now()
		rc, err := entry.provider.StreamChatCompletion(ctx, rawBody)
		if err != nil {
			entry.breaker.RecordFailure()
			lastErr = err
			continue
		}
		// Streaming success/latency is recorded optimistically at connection
		// time — mid-stream failures aren't visible here without buffering
		// (which would defeat the point of streaming), so this measures
		// "how fast did the provider start responding," which is exactly
		// the number a latency-aware router should be ranking on anyway.
		entry.breaker.RecordSuccess()
		entry.latency.Record(time.Since(callStart))
		return rc, nil
	}
	if len(attempted) == 0 {
		return nil, fmt.Errorf("router: every provider in the chain has an open circuit breaker")
	}
	return nil, fmt.Errorf("router: all %d attempted provider(s) failed, last error: %w", len(attempted), lastErr)
}
