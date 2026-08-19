package router

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/rakshit-gen/verigate/internal/auth"
	"github.com/rakshit-gen/verigate/internal/cache"
	"github.com/rakshit-gen/verigate/internal/config"
	"github.com/rakshit-gen/verigate/internal/eval"
	"github.com/rakshit-gen/verigate/internal/guardrails"
	"github.com/rakshit-gen/verigate/internal/otelx"
	"github.com/rakshit-gen/verigate/internal/provider"
	"github.com/rakshit-gen/verigate/internal/replay"
	"github.com/rakshit-gen/verigate/internal/store"
)

type Deps struct {
	Cfg      config.Config
	Store    *store.Store
	Cache    *cache.Cache
	Provider provider.Provider
	Judge    *eval.Judge
	Sampler  *eval.Sampler
	Otel     *otelx.Providers
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// The gateway route: this is what client apps point their base_url at.
	r.Group(func(gr chi.Router) {
		gr.Use(auth.Middleware(d.Cfg.VerigateAPIKey, d.Store))
		gr.Post("/v1/chat/completions", handleChatCompletions(d))
	})

	// Dashboard read API stays open — this is a public demo/portfolio
	// deployment, and the read surface only ever shows this single
	// deployment's own demo traffic (see README's honest-gaps note on
	// per-tenant read scoping being a real, separate gap from this one).
	// Write/admin operations are a different story: creating a tenant or
	// triggering a replay run (which spends real provider credits) must
	// not be something anyone with the URL can do, so those two require
	// the operator's own VERIGATE_API_KEY specifically — not a tenant key.
	r.Route("/api", func(ar chi.Router) {
		ar.Get("/requests", handleListRequests(d))
		ar.Get("/evals/summary", handleEvalSummary(d))
		ar.Get("/evals/recent", handleRecentEvals(d))
		ar.Get("/tenants", handleListTenants(d))
		ar.Get("/providers", handleProviderStatus(d))

		ar.Group(func(admin chi.Router) {
			admin.Use(auth.AdminOnly(d.Cfg.VerigateAPIKey))
			admin.Post("/tenants", handleCreateTenant(d))
			admin.Post("/replay", handleReplay(d))
		})
	})

	return r
}

func handleChatCompletions(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Span name follows the GenAI convention's "{operation} {model}"
		// pattern (finalized once we know the model, via span.SetName is
		// not available in the stable API, so we start it eagerly and
		// carry the model on as an attribute instead — the convention
		// itself acknowledges the model isn't always known before parsing
		// the body).
		ctx, span := d.Otel.Tracer.Start(r.Context(), "chat_completions",
			trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()

		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			span.SetStatus(codes.Error, "failed to read request body")
			http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
			return
		}

		var parsed struct {
			Model    string `json:"model"`
			Stream   bool   `json:"stream"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(rawBody, &parsed)
		lastUserPrompt := extractLastUserMessage(parsed.Messages)
		tenantID := auth.TenantIDFromContext(r.Context())

		span.SetAttributes(
			otelx.AttrGenAISystem.String(d.Provider.Name()),
			otelx.AttrGenAIOperationName.String("chat"),
			otelx.AttrGenAIRequestModel.String(parsed.Model),
		)

		key := cache.Key(rawBody)

		if parsed.Stream {
			handleStreamingChat(ctx, d, w, span, rawBody, key, parsed.Model, lastUserPrompt, tenantID, start)
			return
		}

		metricAttrs := metric.WithAttributes(
			otelx.AttrGenAISystem.String(d.Provider.Name()),
			otelx.AttrGenAIRequestModel.String(parsed.Model),
		)

		cached, cacheType, hit := lookupCache(ctx, d.Cache, key, lastUserPrompt)
		if hit {
			latency := time.Since(start)
			span.SetAttributes(
				otelx.AttrVerigateCacheHit.Bool(true),
				otelx.AttrVerigateCacheType.String(cacheType),
			)
			d.Otel.RequestCounter.Add(ctx, 1, metricAttrs)
			d.Otel.LatencyHist.Record(ctx, float64(latency.Milliseconds()), metricAttrs)

			w.Header().Set("X-Verigate-Cache", cacheType) // "exact" or "semantic" — more informative than a bare "hit"
			w.Header().Set("Content-Type", "application/json")
			w.Write(cached)

			gPrompt, gResponse, piiRedacted, injScore := guardedFields(lastUserPrompt, extractContentFromRaw(cached))
			id, err := d.Store.InsertRequest(ctx, store.RequestRecord{
				Provider:       d.Provider.Name(),
				Model:          parsed.Model,
				Prompt:         gPrompt,
				Response:       gResponse,
				LatencyMS:      int(latency.Milliseconds()),
				CacheHit:       true,
				Status:         "ok",
				PIIRedacted:    piiRedacted,
				InjectionScore: injScore,
				ToolCalls:      extractToolCallsFromRaw(cached),
				TenantID:       tenantID,
			})
			if err != nil {
				log.Printf("failed to log cached request: %v", err)
			} else {
				span.SetAttributes(otelx.AttrVerigateRequestID.String(id))
				d.Sampler.MaybeSample(id)
			}
			return
		}
		span.SetAttributes(
			otelx.AttrVerigateCacheHit.Bool(false),
			otelx.AttrVerigateCacheType.String("miss"),
		)

		resp, err := d.Provider.ChatCompletion(ctx, rawBody)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Printf("provider error: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			gPrompt, _, piiRedacted, injScore := guardedFields(lastUserPrompt, "")
			d.Store.InsertRequest(ctx, store.RequestRecord{
				Provider: d.Provider.Name(), Model: parsed.Model, Prompt: gPrompt,
				Response: "", LatencyMS: int(time.Since(start).Milliseconds()), Status: "error",
				PIIRedacted: piiRedacted, InjectionScore: injScore, TenantID: tenantID,
			})
			return
		}

		// Prefer the response's own ProviderName over d.Provider.Name(): when
		// d.Provider is a fallback-chain Router, Name() can only report the
		// chain's declared primary — the request may actually have been
		// served by whichever provider's circuit was open at the time.
		servedBy := resp.ProviderName
		if servedBy == "" {
			servedBy = d.Provider.Name()
		}

		latency := time.Since(start)
		span.SetAttributes(
			otelx.AttrGenAISystem.String(servedBy),
			otelx.AttrGenAIUsageInputTokens.Int(resp.TokensIn),
			otelx.AttrGenAIUsageOutputTokens.Int(resp.TokensOut),
			otelx.AttrVerigateCostUSD.Float64(resp.CostUSD),
		)
		d.Otel.RequestCounter.Add(ctx, 1, metricAttrs)
		d.Otel.LatencyHist.Record(ctx, float64(latency.Milliseconds()), metricAttrs)
		d.Otel.TokenCounter.Add(ctx, int64(resp.TokensIn), metric.WithAttributes(
			otelx.AttrGenAISystem.String(servedBy), otelx.AttrGenAITokenType.String(string(otelx.TokenInput))))
		d.Otel.TokenCounter.Add(ctx, int64(resp.TokensOut), metric.WithAttributes(
			otelx.AttrGenAISystem.String(servedBy), otelx.AttrGenAITokenType.String(string(otelx.TokenOutput))))

		w.Header().Set("X-Verigate-Cache", "miss")
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp.RawJSON)

		if err := d.Cache.Set(ctx, key, resp.RawJSON); err != nil {
			log.Printf("cache set failed: %v", err)
		} else {
			d.Cache.IndexForSemanticLookup(ctx, lastUserPrompt, key)
		}

		gPrompt, gResponse, piiRedacted, injScore := guardedFields(lastUserPrompt, resp.Content)
		id, err := d.Store.InsertRequest(ctx, store.RequestRecord{
			Provider:       servedBy,
			Model:          parsed.Model,
			Prompt:         gPrompt,
			Response:       gResponse,
			LatencyMS:      int(latency.Milliseconds()),
			CacheHit:       false,
			TokensIn:       resp.TokensIn,
			TokensOut:      resp.TokensOut,
			CostUSD:        resp.CostUSD,
			Status:         "ok",
			PIIRedacted:    piiRedacted,
			InjectionScore: injScore,
			ToolCalls:      extractToolCallsFromRaw(resp.RawJSON),
			TenantID:       tenantID,
		})
		if err != nil {
			log.Printf("failed to log request: %v", err)
			return
		}
		span.SetAttributes(otelx.AttrVerigateRequestID.String(id))
		d.Sampler.MaybeSample(id)
	}
}

func handleListRequests(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ?tenant_id= scopes the view to one tenant's traffic; omitted (the
		// default dashboard view) shows every tenant's requests together.
		reqs, err := d.Store.ListRequests(r.Context(), r.URL.Query().Get("tenant_id"), 50)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if reqs == nil {
			reqs = []store.RequestRecord{}
		}
		writeJSON(w, http.StatusOK, reqs)
	}
}

func handleEvalSummary(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sum, err := d.Store.RegressionSummary(r.Context(), "", // "" = across all rubrics, for the dashboard's overall banner
			d.Cfg.RegressionWindow, d.Cfg.RegressionBaselineWindow,
			d.Cfg.RegressionZThreshold, d.Cfg.RegressionMinScore)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		status := "ok"
		if sum.Regressed {
			status = "regressed"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"rolling_avg_score": sum.RollingAvgScore,
			"sample_count":      sum.SampleCount,
			"baseline_avg":      sum.BaselineAvg,
			"baseline_stddev":   sum.BaselineStddev,
			"baseline_count":    sum.BaselineCount,
			"z_score":           sum.ZScore,
			"method":            sum.Method,
			"status":            status,
		})
	}
}

func handleRecentEvals(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		evals, err := d.Store.RecentEvals(r.Context(), 30)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if evals == nil {
			evals = []store.EvalRecord{}
		}
		writeJSON(w, http.StatusOK, evals)
	}
}

func handleListTenants(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenants, err := d.Store.ListTenants(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if tenants == nil {
			tenants = []store.Tenant{}
		}
		writeJSON(w, http.StatusOK, tenants)
	}
}

// handleCreateTenant is the ONLY place the plaintext API key is ever
// visible after creation — the response includes it once, the same
// "shown once, save it now" contract as the CLI (cmd/tenant). Nothing
// else, including this same dashboard, can ever retrieve it again.
func handleCreateTenant(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name         string `json:"name"`
			RateLimitRPM int    `json:"rate_limit_rpm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}
		if body.RateLimitRPM <= 0 {
			body.RateLimitRPM = 60
		}

		tenant, apiKey, err := d.Store.CreateTenant(r.Context(), body.Name, body.RateLimitRPM)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"id":             tenant.ID,
			"name":           tenant.Name,
			"rate_limit_rpm": tenant.RateLimitRPM,
			"created_at":     tenant.CreatedAt,
			"api_key":        apiKey,
		})
	}
}

// handleProviderStatus reports which provider(s) are actually configured
// and, when it's a fallback chain, each entry's live breaker state and
// measured latency — otherwise there is no visibility at all into which
// provider is serving traffic or why one might have failed over.
func handleProviderStatus(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if chainRouter, ok := d.Provider.(*provider.Router); ok {
			writeJSON(w, http.StatusOK, map[string]any{
				"mode":      "chain",
				"providers": chainRouter.Status(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"mode": "single",
			"providers": []provider.EntryStatus{
				{Name: d.Provider.Name(), BreakerState: "closed"},
			},
		})
	}
}

// handleReplay is the HTTP front end for internal/replay — same logic
// cmd/replay uses, synchronous because this is a low-traffic admin
// operation (a handful of judge calls, a few seconds), not something that
// needs a job queue at today's scale. limit is capped to keep one request
// from accidentally kicking off dozens of judge calls.
func handleReplay(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RequestID      string `json:"request_id"`
			Limit          int    `json:"limit"`
			CandidateModel string `json:"candidate_model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.CandidateModel == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "candidate_model is required"})
			return
		}
		if body.Limit <= 0 {
			body.Limit = 3
		}
		if body.Limit > 10 {
			body.Limit = 10
		}

		targets, err := replay.SelectTargets(r.Context(), d.Store, body.RequestID, body.Limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if len(targets) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{"results": []replay.Result{}, "skipped": []string{}})
			return
		}

		var results []replay.Result
		var skipped []string
		for _, target := range targets {
			res, err := replay.One(r.Context(), d.Provider, d.Judge, target, body.CandidateModel)
			if err != nil {
				skipped = append(skipped, err.Error())
				continue
			}
			results = append(results, res)
		}
		if results == nil {
			results = []replay.Result{}
		}
		if skipped == nil {
			skipped = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": results, "skipped": skipped})
	}
}

// lookupCache tries exact-match first (free, precise) and only falls back
// to a semantic (embedding-similarity) lookup on a miss — semantic search
// costs a real embedding API call, so it's never on the fast path when an
// exact hit is available. Returns cacheType "exact" | "semantic" for
// telemetry even though the caller only needs hit/miss to decide what to
// do next.
func lookupCache(ctx context.Context, c *cache.Cache, exactKey, prompt string) (value []byte, cacheType string, hit bool) {
	if v, ok := c.Get(ctx, exactKey); ok {
		return v, "exact", true
	}
	if matchedKey, _, found := c.SemanticLookup(ctx, prompt); found {
		if v, ok := c.Get(ctx, matchedKey); ok {
			return v, "semantic", true
		}
	}
	return nil, "", false
}

func extractLastUserMessage(messages []struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

func extractContentFromRaw(raw []byte) string {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return ""
	}
	return parsed.Choices[0].Message.Content
}

// extractToolCallsFromRaw pulls the raw tool_calls JSON array out of an
// OpenAI-shaped chat.completion response, if the model's turn was a tool
// call rather than (or in addition to) text content. Returns "" when
// there are none — the common case.
//
// Streaming responses are NOT covered here: OpenAI's streaming tool-call
// format sends the function name and arguments as incremental fragments
// spread across multiple chunks, and reassembling that correctly is
// meaningfully more work than the buffered case — an honest, scoped gap
// (tracked in docs/ARCHITECTURE.md), not an oversight.
func extractToolCallsFromRaw(raw []byte) string {
	var parsed struct {
		Choices []struct {
			Message struct {
				ToolCalls json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return ""
	}
	tc := parsed.Choices[0].Message.ToolCalls
	if len(tc) == 0 || string(tc) == "null" {
		return ""
	}
	return string(tc)
}

func extractUsageFromRaw(raw []byte) (tokensIn, tokensOut int) {
	var parsed struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return 0, 0
	}
	return parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens
}

// guardedFields redacts recognizable PII/secrets out of prompt/response
// text and scores the prompt for prompt-injection heuristics — applied to
// what Verigate STORES (the requests table), never to what's actually
// forwarded to the provider or returned to the caller. Truncation happens
// after redaction/scoring so the guardrails see the full text, not a
// pre-cut fragment.
func guardedFields(prompt, response string) (redactedPrompt, redactedResponse string, piiRedacted bool, injectionScore float64) {
	injectionScore, _ = guardrails.PromptInjectionScore(prompt)
	var promptFound, responseFound bool
	redactedPrompt, promptFound = guardrails.RedactPII(prompt)
	redactedResponse, responseFound = guardrails.RedactPII(response)
	redactedPrompt = truncate(redactedPrompt, 2000)
	redactedResponse = truncate(redactedResponse, 2000)
	return redactedPrompt, redactedResponse, promptFound || responseFound, injectionScore
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
