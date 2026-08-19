package store

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// ResetDemoData wipes request/eval history so a demo script can seed a
// clean before/after state — without it, repeated seed runs accumulate
// history that drags the regression baseline down until the banner stops
// flipping. Not exposed over HTTP; callers are local dev tooling only.
func (s *Store) ResetDemoData(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "TRUNCATE TABLE evals, requests RESTART IDENTITY CASCADE")
	return err
}

type RequestRecord struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	Provider       string    `json:"provider"`
	Model          string    `json:"model"`
	Prompt         string    `json:"prompt"`
	Response       string    `json:"response"`
	LatencyMS      int       `json:"latency_ms"`
	CacheHit       bool      `json:"cache_hit"`
	TokensIn       int       `json:"tokens_in"`
	TokensOut      int       `json:"tokens_out"`
	CostUSD        float64   `json:"cost_usd"`
	Status         string    `json:"status"`
	PIIRedacted    bool      `json:"pii_redacted"`
	InjectionScore float64   `json:"injection_score"`
	// ToolCalls holds the raw JSON array of tool/function calls the model
	// made, when the response was a tool call rather than (or alongside)
	// text content — OpenAI-shaped responses put these in a separate
	// `tool_calls` field on the message, distinct from `content`, which is
	// often empty for a pure tool-call turn. Empty string means no tool
	// calls were present.
	ToolCalls string `json:"tool_calls"`
	// TenantID is "" for requests authenticated with the static
	// VERIGATE_API_KEY (the default, single-tenant mode) — only requests
	// authenticated with a per-tenant key carry one.
	TenantID string `json:"tenant_id"`
}

func (s *Store) InsertRequest(ctx context.Context, r RequestRecord) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO requests (provider, model, prompt, response, latency_ms, cache_hit, tokens_in, tokens_out, cost_usd, status, pii_redacted, injection_score, tool_calls, tenant_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id
	`, r.Provider, r.Model, r.Prompt, r.Response, r.LatencyMS, r.CacheHit, r.TokensIn, r.TokensOut, r.CostUSD, r.Status, r.PIIRedacted, r.InjectionScore, r.ToolCalls, nullableUUID(r.TenantID)).Scan(&id)
	return id, err
}

// nullableUUID converts an empty string to a real SQL NULL rather than
// letting pgx try to parse "" as a UUID (which errors) — used everywhere
// TenantID is optional.
func nullableUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

// ListRequests, when tenantID is "", returns requests across ALL tenants
// (the default dashboard view / single-tenant mode). Pass a specific
// tenant ID to scope the view to just that tenant's traffic.
func (s *Store) ListRequests(ctx context.Context, tenantID string, limit int) ([]RequestRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, created_at, provider, model, prompt, response, latency_ms, cache_hit,
		       COALESCE(tokens_in,0), COALESCE(tokens_out,0), COALESCE(cost_usd,0), status,
		       pii_redacted, injection_score, tool_calls, COALESCE(tenant_id::text, '')
		FROM requests
		WHERE ($1 = '' OR tenant_id::text = $1)
		ORDER BY created_at DESC LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RequestRecord
	for rows.Next() {
		var r RequestRecord
		if err := rows.Scan(&r.ID, &r.CreatedAt, &r.Provider, &r.Model, &r.Prompt, &r.Response,
			&r.LatencyMS, &r.CacheHit, &r.TokensIn, &r.TokensOut, &r.CostUSD, &r.Status,
			&r.PIIRedacted, &r.InjectionScore, &r.ToolCalls, &r.TenantID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetRequest(ctx context.Context, id string) (*RequestRecord, error) {
	var r RequestRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, created_at, provider, model, prompt, response, latency_ms, cache_hit,
		       COALESCE(tokens_in,0), COALESCE(tokens_out,0), COALESCE(cost_usd,0), status,
		       pii_redacted, injection_score, tool_calls, COALESCE(tenant_id::text, '')
		FROM requests WHERE id = $1
	`, id).Scan(&r.ID, &r.CreatedAt, &r.Provider, &r.Model, &r.Prompt, &r.Response,
		&r.LatencyMS, &r.CacheHit, &r.TokensIn, &r.TokensOut, &r.CostUSD, &r.Status,
		&r.PIIRedacted, &r.InjectionScore, &r.ToolCalls, &r.TenantID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

type EvalRecord struct {
	ID        string    `json:"id"`
	RequestID string    `json:"request_id"`
	Rubric    string    `json:"rubric"`
	Score     float64   `json:"score"`
	Reasoning string    `json:"reasoning"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) InsertEval(ctx context.Context, e EvalRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO evals (request_id, rubric, score, reasoning) VALUES ($1,$2,$3,$4)
	`, e.RequestID, e.Rubric, e.Score, e.Reasoning)
	return err
}

func (s *Store) RecentEvals(ctx context.Context, limit int) ([]EvalRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, request_id, rubric, score, COALESCE(reasoning,''), created_at
		FROM evals ORDER BY created_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EvalRecord
	for rows.Next() {
		var e EvalRecord
		if err := rows.Scan(&e.ID, &e.RequestID, &e.Rubric, &e.Score, &e.Reasoning, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type EvalSummary struct {
	RollingAvgScore float64
	SampleCount     int
	BaselineAvg     float64
	BaselineStddev  float64
	BaselineCount   int
	ZScore          float64
	Regressed       bool
	// Method is "statistical" once there's enough baseline history to run a
	// real z-test, or "fixed_threshold_bootstrap" during cold start when
	// there isn't yet — a brand-new deployment has no baseline to compare
	// against, so a fixed floor is the only sane thing to check.
	Method string
}

// RegressionSummary compares the mean of the most recent `recentWindow`
// eval scores against the mean of the `baselineWindow` scores immediately
// before that, using a one-sample z-test against the baseline's own
// distribution: z = (baselineAvg - recentAvg) / (baselineStddev /
// sqrt(recentN)). This catches gradual drift relative to what's actually
// normal for THIS deployment and THIS judge model, not just an absolute
// number picked in advance — a deployment whose honest baseline quality is
// 0.75 regressing to 0.65 is a real regression that a fixed threshold of
// 0.6 would miss entirely.
//
// Falls back to the fixed-threshold check (bootstrapMinScore) when there
// isn't yet enough baseline data (bootstrapMinBaseline) to compute a
// meaningful stddev — the statistical test is not reliable on a handful of
// points, and silence during cold start is worse than a blunt floor.
// rubric, when non-empty, scopes the whole comparison to one rubric (e.g.
// "groundedness") rather than mixing every rubric's scores into one
// average — pass "" for the all-rubrics dashboard summary.
func (s *Store) RegressionSummary(ctx context.Context, rubric string, recentWindow, baselineWindow int, zThreshold, bootstrapMinScore float64) (EvalSummary, error) {
	const bootstrapMinBaseline = 10

	var sum EvalSummary
	err := s.pool.QueryRow(ctx, `
		WITH recent AS (
			SELECT score FROM evals WHERE ($3 = '' OR rubric = $3) ORDER BY created_at DESC LIMIT $1
		), baseline AS (
			SELECT score FROM evals WHERE ($3 = '' OR rubric = $3) ORDER BY created_at DESC OFFSET $1 LIMIT $2
		)
		SELECT
			(SELECT COALESCE(AVG(score), 1.0) FROM recent),
			(SELECT COUNT(*) FROM recent),
			(SELECT COALESCE(AVG(score), 1.0) FROM baseline),
			(SELECT COALESCE(STDDEV(score), 0) FROM baseline),
			(SELECT COUNT(*) FROM baseline)
	`, recentWindow, baselineWindow, rubric).Scan(
		&sum.RollingAvgScore, &sum.SampleCount,
		&sum.BaselineAvg, &sum.BaselineStddev, &sum.BaselineCount,
	)
	if err != nil {
		return sum, err
	}

	if sum.SampleCount == 0 {
		sum.Method = "fixed_threshold_bootstrap"
		return sum, nil
	}

	if sum.BaselineCount < bootstrapMinBaseline || sum.BaselineStddev == 0 {
		sum.Method = "fixed_threshold_bootstrap"
		sum.Regressed = sum.RollingAvgScore < bootstrapMinScore
		return sum, nil
	}

	sum.Method = "statistical"
	standardError := sum.BaselineStddev / math.Sqrt(float64(sum.SampleCount))
	sum.ZScore = (sum.BaselineAvg - sum.RollingAvgScore) / standardError
	sum.Regressed = sum.ZScore > zThreshold
	return sum, nil
}
