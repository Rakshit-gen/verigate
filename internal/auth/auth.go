package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/rakshit-gen/verigate/internal/store"
)

type contextKey string

const (
	tenantIDKey contextKey = "verigate-tenant-id"
	scopeKey    contextKey = "verigate-scope"
)

// TenantIDFromContext returns the authenticated tenant's ID, or "" for a
// request authenticated with the static VERIGATE_API_KEY (single-tenant
// mode — no tenant record at all).
func TenantIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(tenantIDKey).(string)
	return id
}

// ScopeFromContext returns the dashboard-read scope Identify resolved for
// this request — the zero value (anonymous/demo) if Identify was never
// applied or resolved no identity.
func ScopeFromContext(ctx context.Context) store.Scope {
	s, _ := ctx.Value(scopeKey).(store.Scope)
	return s
}

// TenantLookup is the subset of *store.Store the middleware needs —
// defined as an interface so tests can supply a fake without a real
// database.
type TenantLookup interface {
	GetTenantByAPIKey(ctx context.Context, plaintextKey string) (*store.Tenant, error)
}

// SessionLookup resolves a browser session token to its owning tenant —
// the session equivalent of TenantLookup.
type SessionLookup interface {
	GetSessionOwner(ctx context.Context, token string) (*store.User, *store.Tenant, error)
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
			if key == "" || !constantTimeEqual(key, expectedKey) {
				http.Error(w, `{"error":"invalid or missing admin API key"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Identify resolves whatever identity a dashboard read request carries —
// unlike Middleware/AdminOnly, it never rejects the request; an
// unrecognized or missing key just resolves to the anonymous/demo scope.
// Handlers read the result via ScopeFromContext and use it to decide what
// the caller is allowed to see, instead of trusting a client-supplied
// tenant_id query param (the fix for a real bug: previously any caller
// could view any tenant's data by guessing/passing that tenant's ID).
//
// Resolution order: static admin key (full visibility) -> tenant API key
// (scoped to that tenant) -> session token (scoped to the session's own
// tenant) -> anonymous (scoped to tenant_id IS NULL, today's public demo
// traffic).
func Identify(expectedKey string, tenants TenantLookup, sessions SessionLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

			var scope store.Scope
			switch {
			case key == "":
				// anonymous — zero-value Scope (demo traffic only)
			case constantTimeEqual(key, expectedKey):
				scope = store.Scope{All: true}
			default:
				if tenant, err := tenants.GetTenantByAPIKey(r.Context(), key); err == nil {
					scope = store.Scope{TenantID: tenant.ID}
				} else if _, tenant, err := sessions.GetSessionOwner(r.Context(), key); err == nil {
					scope = store.Scope{TenantID: tenant.ID}
				}
				// no match — falls through to the anonymous zero value,
				// same as a request with no Authorization header at all.
			}

			ctx := context.WithValue(r.Context(), scopeKey, scope)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
