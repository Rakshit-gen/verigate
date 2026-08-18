package auth

import (
	"sync"

	"golang.org/x/time/rate"
)

// tenantLimiters lazily creates and caches one token-bucket limiter per
// tenant, sized to that tenant's own configured requests-per-minute — so
// one tenant's traffic can never starve another's, which is the whole
// point of per-tenant limits in a shared gateway.
type tenantLimiters struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func newTenantLimiters() *tenantLimiters {
	return &tenantLimiters{limiters: make(map[string]*rate.Limiter)}
}

func (t *tenantLimiters) Allow(tenantID string, rpm int) bool {
	t.mu.Lock()
	limiter, ok := t.limiters[tenantID]
	if !ok {
		// burst = rpm allows a tenant to use up its whole minute's budget
		// in a quick burst rather than being forced into a perfectly even
		// drip — the common, expected shape of real traffic.
		limiter = rate.NewLimiter(rate.Limit(float64(rpm)/60.0), rpm)
		t.limiters[tenantID] = limiter
	}
	t.mu.Unlock()
	return limiter.Allow()
}
