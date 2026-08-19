package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/rakshit-gen/verigate/internal/store"
)

type contextKey string

const tenantIDKey contextKey = "verigate-tenant-id"

// TenantIDFromContext returns the authenticated tenant's ID, or "" for a
// request authenticated with the static VERIGATE_API_KEY (single-tenant
// mode — no tenant record at all).
func TenantIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(tenantIDKey).(string)
	return id
}

// TenantLookup is the subset of *store.Store the middleware needs —
// defined as an interface so tests can supply a fake without a real
// database.
type TenantLookup interface {
	GetTenantByAPIKey(ctx context.Context, plaintextKey string) (*store.Tenant, error)
}

// Middleware accepts either the static VERIGATE_API_KEY (single-tenant
// mode, unchanged from before multi-tenancy existed) or a per-tenant API
// key looked up against the tenants table. A matched tenant key is also
// rate-limited to that tenant's own configured requests-per-minute; the
// static key is never rate-limited (it's the operator's own key, not a
// customer's).
func Middleware(expectedKey string, tenants TenantLookup) func(http.Handler) http.Handler {
	limiters := newTenantLimiters()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			key := strings.TrimPrefix(header, "Bearer ")
			if key == "" {
				http.Error(w, `{"error":"invalid or missing API key"}`, http.StatusUnauthorized)
				return
			}

			if key == expectedKey {
				next.ServeHTTP(w, r)
				return
			}

			tenant, err := tenants.GetTenantByAPIKey(r.Context(), key)
			if err != nil {
				http.Error(w, `{"error":"invalid or missing API key"}`, http.StatusUnauthorized)
				return
			}

			if !limiters.Allow(tenant.ID, tenant.RateLimitRPM) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"rate limit exceeded for this API key"}`, http.StatusTooManyRequests)
				return
			}

			ctx := context.WithValue(r.Context(), tenantIDKey, tenant.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly accepts only the static VERIGATE_API_KEY — unlike Middleware,
// it does not accept per-tenant keys. Use it for operator-level actions
// (creating tenants, triggering a replay run) that must not be something
// any customer's own API key can do.
func AdminOnly(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			key := strings.TrimPrefix(header, "Bearer ")
			if key == "" || key != expectedKey {
				http.Error(w, `{"error":"invalid or missing admin API key"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
