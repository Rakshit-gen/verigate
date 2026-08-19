# Verigate — Architecture & Build Plan

*An LLM gateway that routes your AI traffic and continuously grades it for quality, in one service.*

Working name: **Verigate** (verify + gate). Rename freely — nothing below depends on it.

---

## 1. The pitch, one paragraph

Verigate sits between your app and OpenAI/Anthropic/local models the way a normal API gateway sits in front of microservices: it routes, caches, rate-limits, and logs. What it does that LiteLLM/Portkey/Helicone don't do in one place: it also samples a slice of live traffic, has a second LLM grade each sampled response against a rubric (groundedness, hallucination, format compliance), and raises an alert the moment the rolling quality score drops — turning "did our AI feature get worse?" from a question you find out about from angry users into something a dashboard tells you first.

---

## 2. Tonight's scope — what "done" means by morning

Time-boxed to roughly 5–6 hours. This is deliberately smaller than the full 3-week vision — the goal tonight is a **real, running vertical slice**, not a mockup.

**In scope tonight:**
- Go gateway service, OpenAI-compatible passthrough (`POST /v1/chat/completions`) — any existing app can point its `base_url` at Verigate with zero code changes.
- One provider adapter (OpenAI; works against a local Ollama model too since it's OpenAI-compatible — useful for testing without burning API credits).
- API-key auth (single static key, header-based).
- Exact-match response caching in Redis.
- Every request logged to Postgres: prompt, response, model, latency, cache hit/miss, tokens, cost.
- An async eval worker: samples a configurable % of requests, calls a judge model with a rubric prompt, writes a score back.
- A regression signal: rolling average score over the last N evals, flips a status flag when it drops below a threshold.
- Dashboard API (same Go binary, separate routes) serving requests + eval summary as JSON.
- Next.js frontend: a live requests table, an eval-score trend chart, and a regression banner that turns red on trigger.
- A seed/demo script that fires realistic + deliberately-bad traffic through the gateway, so the regression banner has something to catch on demand — this is what makes the README GIF actually prove the claim instead of asserting it.

**Explicitly out of scope tonight** (real, tracked as the post-tonight roadmap in §8 — don't drift into building these before the slice above works end to end):
- Semantic/embedding-based caching
- Multiple providers + cost-based routing + circuit breakers/fallback chains
- Real OpenTelemetry GenAI-semantic-convention spans exported to a collector/Grafana (tonight: structured logs with the same field names, so swapping in real OTel later is a rename, not a rewrite)
- Guardrails (PII redaction, prompt-injection detection)
- Multi-tenancy
- Auth beyond one static key

---

## 3. System architecture

```
                 ┌──────────────────────────────────────────────┐
                 │                 Verigate (Go)                 │
  Client app ───▶│  Auth ──▶ Router ──▶ Cache (Redis) ──▶ Adapter │───▶ OpenAI / Ollama
  (points its    │              │            │               │    │
  base_url here) │              ▼            ▼               ▼    │
                 │         Request Logger ──────────────▶ Postgres │
                 │              │                                  │
                 │              ▼                                  │
                 │       Eval Sampler (channel) ──▶ Eval Worker(s)  │
                 │                                       │          │
                 │                                       ▼          │
                 │                              Judge model call    │
                 │                                       │          │
                 │                                       ▼          │
                 │                          writes score ──▶ Postgres│
                 └──────────────────────────────────────────────┘
                                       ▲
                                       │  JSON over HTTP (polling)
                                       │
                              ┌─────────────────┐
                              │  Next.js Dashboard│
                              │  - requests table │
                              │  - eval trend chart│
                              │  - regression banner│
                              └─────────────────┘
```

Everything above runs as **one Go binary** tonight (gateway + dashboard API + eval worker as a goroutine pool) plus **one Next.js dev server** plus **Postgres and Redis running locally via Homebrew** (already installed — no Docker needed tonight, which removes a whole category of "why won't the container start" risk on a deadline).

---

## 4. Data model (Postgres)

```sql
CREATE TABLE requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    provider      TEXT NOT NULL,           -- 'openai' | 'ollama'
    model         TEXT NOT NULL,
    prompt        TEXT NOT NULL,           -- last user message, truncated
    response      TEXT NOT NULL,           -- truncated
    latency_ms    INT NOT NULL,
    cache_hit     BOOLEAN NOT NULL DEFAULT false,
    tokens_in     INT,
    tokens_out    INT,
    cost_usd      NUMERIC(10,6),
    status        TEXT NOT NULL            -- 'ok' | 'error'
);

CREATE TABLE evals (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id    UUID NOT NULL REFERENCES requests(id),
    rubric        TEXT NOT NULL,           -- 'groundedness' | 'format_compliance'
    score         NUMERIC(3,2) NOT NULL,   -- 0.00 - 1.00
    reasoning     TEXT,                    -- judge's explanation, short
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_requests_created_at ON requests (created_at DESC);
CREATE INDEX idx_evals_created_at ON evals (created_at DESC);
```

Rolling average is a read-time query (`AVG(score) OVER last N rows`), not a separate table — no need to maintain a rollup tonight.

---

## 5. Tech stack

| Layer | Choice | Why |
|---|---|---|
| Gateway/API | Go, `net/http` + `chi` router | You already know this cold from VantageEdge; no framework tax |
| Cache | Redis (`go-redis`) | Same as VantageEdge |
| DB | Postgres (`pgx`) | Same as VantageEdge/HSV |
| Async worker | Go channels + goroutine pool | A queue library (Kafka/BullMQ-equivalent) is overkill for tonight's volume — upgrade path noted in roadmap |
| Frontend | Next.js (App Router) + TypeScript + Tailwind | Matches your resume's strongest FE stack; Tailwind is fastest to style under time pressure |
| Charts | `recharts` | Minimal setup, good enough for one trend line |
| Data fetching | Plain `fetch` + polling (5s interval) | A WebSocket is a nice-to-have, not worth the setup time tonight |
| Infra tonight | Local Homebrew Postgres + Redis, no Docker | Both already installed on this machine — skip container overhead entirely |

---

## 6. API contract

```
POST /v1/chat/completions
  Headers: Authorization: Bearer <VERIGATE_API_KEY>
  Body: OpenAI chat-completions JSON (unchanged) — full passthrough
  Response: OpenAI chat-completions JSON (unchanged), + response header X-Verigate-Cache: hit|miss

GET  /api/requests?limit=50&before=<cursor>
  → [{ id, created_at, provider, model, latency_ms, cache_hit, tokens_in, tokens_out, cost_usd, status }]

GET  /api/evals/summary?window=1h
  → { rolling_avg_score, sample_count, status: "ok" | "regressed", by_rubric: {...} }

GET  /api/evals/recent?limit=20
  → [{ request_id, rubric, score, reasoning, created_at }]
```

---

## 7. Folder structure

```
verigate/
  cmd/
    gateway/main.go            # entrypoint: wires config, DB, redis, starts HTTP server + worker pool
  internal/
    auth/                      # API key middleware
    router/                    # chi routes, handlers
    cache/                     # redis get/set, key hashing
    provider/                  # provider interface + openai adapter (+ ollama for local testing)
    store/                     # postgres queries (requests, evals)
    eval/
      worker.go                # goroutine pool, consumes sample channel
      rubrics.go                # rubric prompts (groundedness, format_compliance)
      judge.go                  # calls judge model, parses score
  migrations/
    001_init.sql
  scripts/
    seed_demo_traffic.go       # fires realistic + deliberately-bad requests for the demo GIF
  web/                          # Next.js app
    app/
      page.tsx                 # dashboard: table + chart + banner
      api/                      # (optional) proxy routes if needed for CORS
    components/
      RequestsTable.tsx
      EvalTrendChart.tsx
      RegressionBanner.tsx
    lib/api.ts                  # typed fetch client for the Go API
  docs/
    ARCHITECTURE.md             # this file
  .env.example
  README.md
```

---

## 8. Tonight's hour-by-hour checklist

- **Hour 1 — scaffold & data layer.** Go module init, folder structure, Postgres schema applied locally, Redis reachable, `chi` server with `/healthz`, config loading from `.env` (API keys, DB/Redis URLs, sample rate).
- **Hour 2 — the passthrough.** `POST /v1/chat/completions` handler → auth middleware → OpenAI adapter (real HTTP call) → log to `requests` table → return response unchanged. Verify with `curl` against a real OpenAI key or local Ollama.
- **Hour 3 — caching + eval plumbing.** Redis exact-match cache keyed on `hash(model + messages)`; cache hit short-circuits the provider call and is logged as `cache_hit=true`. Eval sampler: on each logged request, roll the sample rate and push the request ID onto a channel. Write the two rubric prompts.
- **Hour 4 — eval worker + dashboard API.** Goroutine pool reads from the channel, fetches the request, calls the judge model, writes to `evals`. `GET /api/requests` and `GET /api/evals/summary` handlers with the rolling-average query.
- **Hour 5 — frontend.** `npx create-next-app`, `RequestsTable` (polls `/api/requests`), `EvalTrendChart` (polls `/api/evals/summary`, recharts line), `RegressionBanner` (red when `status: "regressed"`).
- **Hour 6 — the proof.** Write `seed_demo_traffic.go`: fires ~30 normal requests, then deliberately injects a few off-topic/garbage "model responses" (bypass the real provider, just log fake bad responses) so the eval worker scores them low and the banner flips red on camera. Record the GIF. Write the README with the architecture diagram from §3 and the GIF at the top — before any install instructions.

---

## 9. Why the demo script matters as much as the code

The single thing that makes this project read as "understands production AI systems" instead of "followed a tutorial" is that the regression banner turning red is **provoked on demand, on camera, in under 30 seconds**. That's the whole differentiator claim from the pitch, proven visually instead of asserted in a README paragraph. Build the seed script early enough tonight that there's time to actually capture it.

---

## 10. Post-tonight roadmap

- **Delivered — real OpenTelemetry instrumentation.** `internal/otelx` wires a `TracerProvider` and `MeterProvider` with the GenAI semantic-convention attribute names (`gen_ai.system`, `gen_ai.operation.name`, `gen_ai.request.model`, `gen_ai.usage.input_tokens`/`output_tokens`), plus a `verigate.eval.score` metric as an explicit extension (there's no official convention yet for continuous quality grading — namespaced accordingly rather than squatting on `gen_ai.*`). Every `/v1/chat/completions` call gets a `chat_completions` span, every judge grading pass gets an `eval.grade_request` span with an `eval.scored` event per rubric. Exporter is env-driven like any standard OTel service: no `OTEL_EXPORTER_OTLP_ENDPOINT` set → spans/metrics pretty-print to stdout (provable with zero infra, which is how this was verified — see below); set that env var → the exact same instrumentation ships to a real OTel Collector, Grafana, Honeycomb, or Datadog over OTLP/HTTP. Verified by running a live request through a throwaway instance and confirming the stdout export: `gen_ai.system: "groq"`, real `gen_ai.usage.input_tokens`/`output_tokens` values, and the `gen_ai.client.token.usage` metric with the correct `gen_ai.token.type` attribute all matched the spec exactly.
- **Delivered — streaming (`stream: true`) support.** `internal/router/streaming.go` proxies SSE chunk-by-chunk with an immediate `Write`+`Flush` per line (real streaming, not buffer-then-send), while a parallel accumulator reassembles the full text so streamed requests get the exact same logging, caching-key hashing, and eval sampling as buffered ones — no second-class path. Two decisions worth knowing about: (1) caching is intentionally skipped for streamed requests — replaying a cached response as a faithful synthetic stream is a real feature, not something to fake, and is tracked separately below; (2) the gateway transparently opts every streaming request into OpenAI's `stream_options.include_usage` (a spec field standard SDKs already ignore safely) so real token counts and cost are available even though the client never asked for them. Also added `gen_ai.server.time_to_first_token` — a real GenAI semantic-convention metric name that only means something once you're actually proxying a stream. Verified against live Groq traffic: correct SSE forwarding, accumulated content matched the model's actual answer, real token counts via the injected usage chunk, and the TTFT metric exported with the right attributes.
- **Delivered — a second real provider adapter (Anthropic).** `internal/provider/anthropic.go` translates both directions: OpenAI-shaped requests in (system-role messages extracted to Anthropic's top-level `system` field, `max_tokens` defaulted since Anthropic requires it, everything else passed through) and Anthropic-shaped responses back into genuine OpenAI-shaped JSON, so a client pointed at Verigate gets identical response shapes regardless of which provider is actually configured. Streaming is translated too — a goroutine reads Anthropic's `message_start`/`content_block_delta`/`message_delta`/`message_stop` SSE events off an `io.Pipe` and writes OpenAI-shaped chunks out the other end, so `internal/router/streaming.go` needed zero changes to support it. **Verified against a mock server shaped like Anthropic's real documented API** (`internal/provider/anthropic_test.go`), not against `api.anthropic.com` — no live Anthropic key is configured in this environment, so live end-to-end verification is still open.
- **Delivered — statistical regression detection.** `Store.RegressionSummary` replaced the fixed-threshold rolling average with a z-test: the mean of the last `REGRESSION_WINDOW` eval scores compared against the mean of the `REGRESSION_BASELINE_WINDOW` scores before that, scoped per-rubric so different rubrics' scores don't get averaged together into one meaningless number. Falls back to the old fixed-floor check automatically during cold start (fewer than 10 baseline points, or zero baseline variance) — a brand-new deployment has no baseline to test against, so silence would be worse than a blunt floor. **Verified with real integration tests against local Postgres** (`internal/store/store_test.go`): confirmed a genuine drop gets flagged with a positive z-score, a noise-level (~equal) change does not, and cold start correctly falls back to the bootstrap method — one test run also caught a real bug in the *test data* (zero-variance synthetic scores triggering the production stddev guard), fixed by seeding realistic jitter instead.
- **Delivered — semantic caching.** Exact-match Redis caching (unchanged) is now backed by an optional embedding-similarity layer: on an exact-match miss, `Cache.SemanticLookup` embeds the prompt and searches an in-process, TTL-bounded nearest-neighbor index (`internal/cache/semantic.go`) for a close-enough previous prompt; a hit re-fetches the original response through the same `Get` path a normal cache hit uses. Deliberately brute-force rather than a real ANN index (pgvector, HNSW) — at hundreds of TTL'd entries that's the correctly-scoped choice, not a shortcut; revisit if `SEMANTIC_CACHE_MAX_ENTRIES` needs to grow by orders of magnitude. Disabled by default and fails open: no `EMBEDDING_API_KEY` configured (the common case on Groq, which doesn't serve embeddings) means exact-match-only, unchanged from before. **Verified two ways**: pure logic (cosine similarity, index eviction, TTL pruning) with no external dependencies, and full `Cache`-level plumbing (embedding call → index → real Redis round-trip) against a mock embeddings server and the actual local Redis — a paraphrased prompt correctly hits the cached response, an unrelated prompt correctly doesn't. Live verification against a real embeddings API (OpenAI) is still open, same caveat as the Anthropic adapter.
- **Delivered — cache streamed responses.** `cache.Key` now normalizes the request body (strips `stream`/`stream_options` before hashing, and — as a side effect — sorts JSON keys via Go's `map[string]any` marshaling, so field ordering no longer affects the key either) so a streaming and a non-streaming request for the same prompt genuinely share one cache entry. A streaming request now checks the cache first, same as non-streaming, and on a hit synthesizes an SSE reply (one content chunk, one usage chunk, `[DONE]`) from the cached OpenAI-shaped completion rather than calling the provider — a deliberate, documented trade-off: a cache hit is already near-instant, so replaying the *original* chunk boundaries would add real complexity for no observable benefit. On a miss, the accumulated streamed response gets cached in the same OpenAI-shaped format non-streaming responses use, so a *later* request in either style can hit it. **Verified live, both directions**, against real Groq traffic: sent a prompt non-streaming, then the identical prompt streaming — served instantly from cache (`id: "verigate-cached"` in the synthesized chunks); then a new prompt streaming first, then the same prompt non-streaming — correctly hit the cache streaming had populated.
- **Delivered — circuit breaker + fallback chain across providers.** `internal/provider/circuitrouter.go` implements a standard three-state breaker (closed/open/half-open) per provider and a `Router` that tries a priority-ordered chain, skipping any provider whose breaker is open and falling forward on failure. This is the honestly-scoped version of "cost/latency-aware routing": the chain's declared order **is** the routing policy (put the cheaper/faster provider first) rather than the router dynamically re-ranking by measured latency, which would need a decay window and real scheduling and is a genuine further step, not this pass's job. Wired automatically in `provider.New`: if credentials for *both* an OpenAI-compatible backend and Anthropic are present, they're chained (`PROVIDER_NAME`'s choice first); with only one credential set configured — the common case — nothing changes. `ChatResponse.ProviderName` was added so the request log and telemetry attribute the response to whichever provider *actually* served it, not just the chain's primary. **Verified with real integration tests** using controllable fake providers (`internal/provider/circuitrouter_test.go`: fallover on failure, breaker opens and skips the failing provider after threshold, half-open recovery after cooldown, all-providers-fail returns an error) — no external keys needed for that. **Then verified live against real network calls**: primary configured as Anthropic with a deliberately invalid key (real 401s against `api.anthropic.com`), fallback the real working Groq credentials. All requests succeeded via fallback with correct content; the third and fourth requests were measurably faster (~0.5–0.8s vs ~0.85–1.05s) than the first two, matching the breaker skipping the failing Anthropic call entirely once its threshold (2) was hit; the DB correctly attributed every request to `openai` (the actual server), not `anthropic` (the declared primary).
- **Delivered — dynamic latency-aware reordering.** `Router` now keeps an exponential moving average of each chain entry's successful-call latency (`internal/provider/circuitrouter.go`'s `latencyTracker`) and, among the providers currently allowed by their breaker, tries the measured-fastest one first — an untested provider (no data yet) sorts as fastest so it gets a chance to prove itself rather than starting permanently last, and declared config order is the tie-break, not the whole policy anymore. **Verified with a real test**: two fake providers with genuinely different simulated latencies, declared in the WRONG order (slow first) — once both have measured data, the router correctly tries the faster one first regardless of declaration order.
- **Delivered — guardrails.** `internal/guardrails` redacts recognizable PII/secrets (email, credit card, SSN, phone, common API-key prefixes) and scores prompts for injection-heuristic risk (0-1, weighted pattern matching) — applied to what Verigate **stores** (the `requests` table), never to what's forwarded to the provider or returned to the caller; conflating those would silently corrupt the live request. New `pii_redacted`/`injection_score` columns (migration `002_guardrails.sql`). **Verified two ways**: real unit tests for both the redaction patterns and the injection scorer (including a test-driven fix — "tell me your system prompt" wasn't originally caught, added as a real pattern gap, not just a test tweak), and a live request through Groq with real fake PII and an injection phrase — confirmed the DB stored `[REDACTED_EMAIL]`/`[REDACTED_CARD]` while the actual model call still received the real, unredacted text (it organically refused the injection attempt).
- **Delivered — tool-call evaluation.** Judge-based grading of whether an assistant's tool call was an appropriate response to the request — genuinely unclaimed territory in current eval tooling, scoped honestly (see `eval.ToolCallRubric`'s doc comment: it grades plausibility from the judge's general knowledge, not against the tool's formal JSON schema, which Verigate doesn't currently capture from the request — a real next step). Required a capture-layer fix first: pure tool-call responses have empty `content`, so nothing was being stored for them at all before this; new `tool_calls` column (migration `003_tool_calls.sql`) fixes that for the non-streaming path (streaming tool-call capture needs reassembling incremental function-call fragments across chunks — a real follow-up, not attempted here). The eval worker now skips the text rubrics for a pure tool-call turn (grading empty text against "groundedness" was quietly poisoning the regression baseline) and runs `ToolCallRubric` whenever tool calls are present — both can apply to the same turn if a model emits commentary alongside a call. **Verified live**: sent a real request with a `get_weather` tool definition, the model correctly called it with `{"location":"Paris"}`, Verigate captured it, and the judge scored it 0.99 with sensible reasoning — and only the tool-call rubric ran, not the (correctly skipped) text rubrics.
- **Delivered — multi-tenancy.** Per-tenant API keys (SHA-256 hashed, never stored in plaintext — `internal/store/tenant.go`), independent per-tenant rate limiting (`golang.org/x/time/rate` token buckets, one tenant's traffic can't starve another's), and tenant-scoped dashboard queries (`GET /api/requests?tenant_id=`). The static `VERIGATE_API_KEY` still works unchanged and is never rate-limited (it's the operator's own key). A `cmd/tenant` CLI creates/lists tenants (`go run ./cmd/tenant create --name acme --rpm 60` — the plaintext key is shown exactly once). **Verified two ways**: real tests for the auth middleware and rate limiter (fake tenant lookup, no DB needed — static key always allowed, invalid keys rejected, per-tenant limits independent) and real integration tests for tenant creation/lookup against Postgres (including confirming the plaintext key is genuinely never stored). **Then verified fully live**: created a real tenant via the CLI, made real Groq calls with its key, confirmed a wrong key gets 401, confirmed the RPM limit actually triggers 429s against real traffic (and correctly persists across separate requests — an earlier warm-up call correctly counted against the same budget), and confirmed every request landed in the DB tagged with the real tenant ID and the scoped dashboard query returned exactly that tenant's requests.
- **Reach, still open:** streaming tool-call capture (see above), tool-schema-aware grading (see above), replaying the *original* chunk sequence on a streaming cache hit rather than a synthesized one, eval-summary/RecentEvals tenant scoping (only `ListRequests` is tenant-scoped today), and live verification of the Anthropic adapter / semantic caching against real API keys (both verified against mocks or a deliberately-invalid-key live failure path — see their entries above — but never a real successful Anthropic or OpenAI-embeddings call, since neither key is configured in this environment).
- **Launch:** MIT `LICENSE` added, git repo initialized, two clean local commits. README leads with setup + a demo before install instructions; a recorded GIF of the regression banner and the fallback-chain/rate-limit behavior would strengthen it further but hasn't been captured. Not yet done: pushing to a real GitHub remote (a real, visible action under a real account — worth confirming before doing it) and the Show HN / r/LocalLLaMA post.

## 11. Dashboard UI — closing the gap between backend and what's actually visible

Backend features kept outpacing what the dashboard could show — several were captured in the database and returned by the API with no page rendering them at all. This pass closed that gap rather than adding more backend surface:

- **Two new endpoints, both backing real logic that already existed elsewhere, not new business logic.** `GET /api/providers` exposes `Router.Status()` (added to `internal/provider/circuitrouter.go` — breaker/latency state was previously only visible by reading Go source, never over HTTP) and falls back to a single-entry response when no fallback chain is configured. `POST /api/replay` is `internal/replay` (the CLI's core logic, extracted from `cmd/replay/main.go` into a shared package so the CLI and the API call the exact same code) — synchronous, since replay is a low-traffic admin operation, capped at 10 requests per call so one form submission can't accidentally trigger dozens of judge calls. `Deps` gained a `Judge` field so the router can drive replay directly.
- **`/providers` page** — live breaker state (closed/open/half-open, color-coded) and measured latency per provider, polling every 3s. **Verified live**: single-provider mode showed the real configured provider; chain mode (deliberately-broken Anthropic + real Groq) showed `anthropic` as `open` with 1 recorded failure and `openai` as `closed` with real measured latency (623ms) immediately after a real fallover.
- **`/replay` page** — a form (candidate model, how many recent requests) replacing the CLI-only workflow, rendering the same original-vs-candidate comparison as a real table with per-rubric deltas instead of a terminal printout. **Verified live** against real Groq + judge traffic through the exact `/api/replay` endpoint the form calls.
- **`RequestsTable` now surfaces guardrails and tool-call data that existed in the API response but was never rendered**: a "Flags" column shows a PII-redacted badge, a color-graded injection-risk badge, and parsed tool-call-name badges (parsed client-side from the `tool_calls` JSON) — previously a tool-call turn just looked like a broken empty row.
- **`RegressionBanner` now shows the statistical method, baseline average, and z-score** the backend has been computing since §10's regression-detection work, not just the old rolling-average/status pair.
- **Shared `Nav` component** across all four pages (`/`, `/tenants`, `/replay`, `/providers`) — added once the page count stopped being "one, maybe two."

## 12. Visual design pass — from "generic Tailwind dashboard" to an actual design system

Functionally complete but visually generic: default slate palette, no real typographic hierarchy, no KPIs at a glance, chart colors picked by eye. This pass replaced that with an actual system, not a re-skin.

- **Palette chosen and validated, not eyeballed.** A custom dark ground (`#0a0e14`, deliberately blue-black rather than default Tailwind slate) with tokenized surfaces/text/status colors in `web/app/globals.css`. The two eval-chart series colors were run through the dataviz-skill palette validator against this exact surface — `#2dd4bf`/`#818cf8` (the original picks, Tailwind's teal-400/indigo-400) failed the dark-mode lightness band; stepped down to `#0d9488`/`#6366f1`, which pass lightness band, CVD separation (ΔE 18.9 deutan, 6.3 tritan — floor band, mitigated by the chart's always-on legend), normal-vision floor (23.3), and contrast. Status colors (good/warning/critical) use the skill's own reserved, contrast-validated scale rather than reusing the categorical hues, so a status badge never impersonates a chart series.
- **KPI stat tiles** (`StatTiles`/`StatTile`) computed entirely client-side from data the dashboard already fetches (request count, cache hit rate, avg latency, error count) — the "at a glance" numbers a screenshot needs, with zero new backend surface.
- **`EvalTrendChart` rebuilt**: area fill under each line (gradient to transparent), a custom crosshair tooltip matching the design tokens, recessive gridlines, and a real bug caught by actually looking at it — the Y-axis percentage labels were being clipped (`100%` rendering as `0%`, `75%` as `5%`) by a leftover negative chart margin from the original quick version. **Caught via a live browser screenshot with `claude-in-chrome`, not just `npm run build`** — the build and typecheck both pass with the bug present, since it's a layout/rendering issue, not a type error. Fixed by correcting the margin and axis width, then re-verified with another screenshot plus a hover interaction confirming the tooltip and crosshair render correctly.
- **`ProvidersStatus` cards** gained a latency bar (relative to the slowest configured provider) and consistent status-token coloring instead of ad-hoc Tailwind color classes.
- **Shared `Logo` + `AppHeader`** — a small inline-SVG brand mark (two gate-posts with a signal trace between them) and one header component used identically across all four pages, replacing four slightly-different hand-rolled headers.
- **Verified visually, not just by build passing**: navigated all four pages with `claude-in-chrome`, screenshotted each, zoomed into the chart to catch the axis-label bug at pixel level, and confirmed the fix with a follow-up screenshot and a hover-triggered tooltip check — the actual rendered pixels were inspected, not just "the build succeeded."
