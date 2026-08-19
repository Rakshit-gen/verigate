package store

import (
	"context"
	"os"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/verigate?sslmode=disable"
	}
	s, err := New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("no local postgres available at %s: %v", dbURL, err)
	}
	return s
}

// seedRequestWithEvals inserts one throwaway request and n evals under the
// given rubric, returning the request ID so the caller can clean up. Each
// test uses its own rubric name so RegressionSummary's rubric filter fully
// isolates it from both the other tests and from whatever real demo data
// already lives in this DB.
//
// Scores get deterministic small jitter around the given center rather
// than being identical — a real judge model's scores always have some
// noise, and a zero-variance baseline is a degenerate case the production
// code correctly refuses to run a z-test against (see the BaselineStddev
// == 0 guard in RegressionSummary), so seeding constants here would be
// testing a case that can't happen in practice.
func seedRequestWithEvals(t *testing.T, ctx context.Context, s *Store, rubric string, n int, center float64) string {
	t.Helper()
	reqID, err := s.InsertRequest(ctx, RequestRecord{
		Provider: "test", Model: "test-model", Prompt: "p", Response: "r",
		LatencyMS: 1, Status: "ok",
	})
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}
	jitters := []float64{0.00, 0.02, -0.02, 0.01, -0.01, 0.03, -0.03}
	for i := 0; i < n; i++ {
		score := center + jitters[i%len(jitters)]
		if err := s.InsertEval(ctx, EvalRecord{RequestID: reqID, Rubric: rubric, Score: score}); err != nil {
			t.Fatalf("insert eval: %v", err)
		}
	}
	return reqID
}

func TestRegressionSummary_FlagsARealDrop(t *testing.T) {
	s := testStore(t)
	defer s.Close()
	ctx := context.Background()

	// baseline first (so it's OFFSET-past the recent window once queried
	// DESC), then a clearly worse recent batch.
	baselineReq := seedRequestWithEvals(t, ctx, s, "test-drop", 15, 0.85)
	defer s.pool.Exec(ctx, "DELETE FROM requests WHERE id = $1", baselineReq)
	recentReq := seedRequestWithEvals(t, ctx, s, "test-drop", 5, 0.55)
	defer s.pool.Exec(ctx, "DELETE FROM requests WHERE id = $1", recentReq)

	sum, err := s.RegressionSummary(ctx, Scope{All: true}, "test-drop", 5, 15, 2.0, 0.6)
	if err != nil {
		t.Fatalf("RegressionSummary: %v", err)
	}
	if sum.Method != "statistical" {
		t.Fatalf("expected statistical method with 15 baseline points, got %q", sum.Method)
	}
	if !sum.Regressed {
		t.Fatalf("expected regression flagged: recent=%.2f baseline=%.2f z=%.2f",
			sum.RollingAvgScore, sum.BaselineAvg, sum.ZScore)
	}
	if sum.ZScore <= 0 {
		t.Fatalf("expected a positive z-score for a genuine drop, got %.2f", sum.ZScore)
	}
}

func TestRegressionSummary_StableHistoryNotFlagged(t *testing.T) {
	s := testStore(t)
	defer s.Close()
	ctx := context.Background()

	baselineReq := seedRequestWithEvals(t, ctx, s, "test-stable", 15, 0.80)
	defer s.pool.Exec(ctx, "DELETE FROM requests WHERE id = $1", baselineReq)
	recentReq := seedRequestWithEvals(t, ctx, s, "test-stable", 5, 0.82) // noise-level difference, not a regression
	defer s.pool.Exec(ctx, "DELETE FROM requests WHERE id = $1", recentReq)

	sum, err := s.RegressionSummary(ctx, Scope{All: true}, "test-stable", 5, 15, 2.0, 0.6)
	if err != nil {
		t.Fatalf("RegressionSummary: %v", err)
	}
	if sum.Regressed {
		t.Fatalf("did not expect regression for a stable history: recent=%.2f baseline=%.2f z=%.2f",
			sum.RollingAvgScore, sum.BaselineAvg, sum.ZScore)
	}
}

func TestRegressionSummary_BootstrapFallbackOnColdStart(t *testing.T) {
	s := testStore(t)
	defer s.Close()
	ctx := context.Background()

	// Only 3 evals total — nowhere near the bootstrapMinBaseline (10), so
	// this must fall back to the fixed-threshold check rather than attempt
	// (and fail) a statistical test on almost no data.
	reqID := seedRequestWithEvals(t, ctx, s, "test-bootstrap", 3, 0.55)
	defer s.pool.Exec(ctx, "DELETE FROM requests WHERE id = $1", reqID)

	sum, err := s.RegressionSummary(ctx, Scope{All: true}, "test-bootstrap", 5, 15, 2.0, 0.6)
	if err != nil {
		t.Fatalf("RegressionSummary: %v", err)
	}
	if sum.Method != "fixed_threshold_bootstrap" {
		t.Fatalf("expected bootstrap fallback with insufficient baseline data, got %q", sum.Method)
	}
	if !sum.Regressed { // 0.55 avg < 0.6 fixed floor
		t.Fatalf("expected the fixed floor to flag 0.55 avg score as regressed")
	}
}
