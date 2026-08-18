package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakshit-gen/verigate/internal/store"
)

// fakeTenants is a controllable TenantLookup — no database needed to test
// the middleware's auth/rate-limit logic in isolation.
type fakeTenants struct {
	byKey map[string]*store.Tenant
}

func (f *fakeTenants) GetTenantByAPIKey(ctx context.Context, key string) (*store.Tenant, error) {
	t, ok := f.byKey[key]
	if !ok {
		return nil, store.ErrTenantNotFound
	}
	return t, nil
}

func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func doRequest(h http.Handler, bearerToken string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestMiddleware_StaticKeyAlwaysAllowed(t *testing.T) {
	tenants := &fakeTenants{byKey: map[string]*store.Tenant{}}
	h := Middleware("dev-local-key", tenants)(testHandler())

	rec := doRequest(h, "dev-local-key")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for the static key, got %d", rec.Code)
	}
}

func TestMiddleware_MissingOrWrongKeyRejected(t *testing.T) {
	tenants := &fakeTenants{byKey: map[string]*store.Tenant{}}
	h := Middleware("dev-local-key", tenants)(testHandler())

	for _, key := range []string{"", "totally-wrong-key"} {
		rec := doRequest(h, key)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("key %q: expected 401, got %d", key, rec.Code)
		}
	}
}

func TestMiddleware_ValidTenantKeyAllowedAndAttachesTenantID(t *testing.T) {
	tenant := &store.Tenant{ID: "tenant-123", Name: "acme", RateLimitRPM: 60}
	tenants := &fakeTenants{byKey: map[string]*store.Tenant{"vg_acmekey": tenant}}

	var capturedTenantID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenantID = TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware("dev-local-key", tenants)(handler)

	rec := doRequest(h, "vg_acmekey")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a valid tenant key, got %d", rec.Code)
	}
	if capturedTenantID != "tenant-123" {
		t.Errorf("expected tenant ID to be attached to the request context, got %q", capturedTenantID)
	}
}

func TestMiddleware_RateLimitsPerTenant(t *testing.T) {
	// RPM=1 with burst=rpm (see ratelimit.go) allows exactly one request,
	// then the very next one should be rejected before the bucket refills.
	tenant := &store.Tenant{ID: "tenant-limited", Name: "limited", RateLimitRPM: 1}
	tenants := &fakeTenants{byKey: map[string]*store.Tenant{"vg_limitedkey": tenant}}
	h := Middleware("dev-local-key", tenants)(testHandler())

	first := doRequest(h, "vg_limitedkey")
	if first.Code != http.StatusOK {
		t.Fatalf("expected the first request to succeed, got %d", first.Code)
	}
	second := doRequest(h, "vg_limitedkey")
	if second.Code != http.StatusTooManyRequests {
		t.Errorf("expected the second immediate request to be rate-limited (429), got %d", second.Code)
	}
}

func TestMiddleware_DifferentTenantsHaveIndependentLimits(t *testing.T) {
	tenantA := &store.Tenant{ID: "a", Name: "a", RateLimitRPM: 1}
	tenantB := &store.Tenant{ID: "b", Name: "b", RateLimitRPM: 1}
	tenants := &fakeTenants{byKey: map[string]*store.Tenant{"vg_a": tenantA, "vg_b": tenantB}}
	h := Middleware("dev-local-key", tenants)(testHandler())

	// Exhaust tenant A's budget.
	doRequest(h, "vg_a")
	// Tenant B should be completely unaffected by tenant A's usage.
	rec := doRequest(h, "vg_b")
	if rec.Code != http.StatusOK {
		t.Errorf("expected tenant B's first request to succeed regardless of tenant A's usage, got %d", rec.Code)
	}
}
