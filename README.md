# Verigate

An LLM gateway that routes your AI traffic **and** continuously grades it for quality — in one open-source service, instead of stitching together a router (LiteLLM) and a separate observability tool (Helicone/Langfuse) that never talk to each other.

Point your app's `base_url` at Verigate instead of OpenAI/Groq directly. It caches repeat requests, logs every call, samples a slice of live traffic, has a second model grade each sample against a rubric, and flips a dashboard banner red the moment the rolling quality score drops — before your users tell you something's wrong.

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full design and build plan.

## Quickstart

**Requirements:** Go 1.25+, Node 20+, Postgres, Redis (all running locally is fine — no Docker needed).

```bash
# 1. database
createdb verigate
for f in migrations/*.sql; do psql verigate -f "$f"; done

# 2. backend
cp .env.example .env
# edit .env — at minimum set OPENAI_API_KEY (or use the Groq/Ollama block instead)
go run ./cmd/gateway
# -> verigate listening on :8080

# 3. frontend, in a second terminal
cd web
cp .env.local.example .env.local
npm install
npm run dev
# -> http://localhost:3000
```

**See it catch a regression on purpose:**

```bash
go run ./scripts/seed_demo_traffic.go
```

This sends real traffic through the gateway, then injects a batch of deliberately bad responses straight into the database so the eval worker scores them low — the dashboard's banner should flip red within a few seconds of refreshing.

**Point any existing app at it with zero code changes:**

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer dev-local-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'
```

Same request/response shape as OpenAI's `/v1/chat/completions` — this is a drop-in proxy, not a new SDK to learn, **even when the request is actually served by Anthropic** (`PROVIDER_NAME=anthropic` — see `.env.example`): Verigate translates the request in and the response back out, so the client never has to know. Add `"stream": true` and it proxies real server-sent events chunk-by-chunk (`curl -N ... -d '{"model":"...","stream":true,"messages":[...]}'`) — the full response still gets logged and eval-sampled exactly like a buffered request once the stream ends.

**Caching:** exact-match by default, and shared between streaming and non-streaming requests — a prompt cached by a streaming call can be served to a later non-streaming call and vice versa. Add an `EMBEDDING_API_KEY` (see `.env.example`) to also enable semantic caching — a paraphrased prompt within `SEMANTIC_CACHE_THRESHOLD` similarity of a previous one reuses its cached response instead of calling the provider again.

**Automatic failover:** if you configure credentials for *both* an OpenAI-compatible backend and Anthropic, Verigate chains them behind a circuit breaker — `PROVIDER_NAME`'s choice is tried first, and traffic automatically fails over to the other on repeated errors, healing back once the failing provider's cooldown elapses and a health-check call succeeds. Once both providers have measured latency data, the chain also tries the currently-faster one first — a static declared order is just the starting point, not the whole policy. Nothing to configure beyond having both credential sets present; with only one, behavior is unchanged.

**Regression detection** compares the recent eval-score window against a trailing baseline with a z-test rather than a fixed cutoff, so it catches drift relative to what's actually normal for your deployment — falls back to a fixed floor automatically until there's enough history.

**Model migration decisions:** `go run ./cmd/replay --candidate-model <model> --limit 5` replays recent real prompts through a candidate model and scores both the original and candidate responses side by side — an actual answer to "should we switch models," not a guess.

**Guardrails:** every stored prompt/response is scanned for recognizable PII/secrets (email, credit card, SSN, phone, common API-key prefixes) and redacted before it hits the database — the live call to the actual provider is never touched, only what Verigate logs. Prompts also get a heuristic 0-1 injection-risk score (`injection_score` on each request) from pattern matching against common jailbreak/override phrasing.

**Tool-call evaluation:** when a model's turn is a tool call rather than text, Verigate captures it properly (a pure tool-call response has empty `content`, so naive logging misses it entirely) and grades it with its own rubric — did the assistant call the right tool with sensible arguments — instead of running text-quality rubrics against an empty string.

**Multi-tenancy:** `go run ./cmd/tenant create --name acme --rpm 60` creates a tenant with its own API key (shown once, SHA-256-hashed at rest — never stored in plaintext) and its own independent rate limit; one tenant's traffic can't starve another's. The static `VERIGATE_API_KEY` keeps working unchanged for single-tenant/local use. `GET /api/requests?tenant_id=<id>` scopes the dashboard to one tenant's traffic.

**Observability:** every request and every eval grading pass is a real OpenTelemetry span, with attribute names following the OTel GenAI semantic conventions (`gen_ai.system`, `gen_ai.request.model`, `gen_ai.usage.input_tokens`, etc.), plus `verigate.eval.score` and `gen_ai.server.time_to_first_token` metrics. By default (no `OTEL_EXPORTER_OTLP_ENDPOINT` set) these pretty-print to stdout — no external infra needed to see it work. Point it at a real backend with zero code changes:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318   # or any OTLP/HTTP endpoint — Grafana, Honeycomb, Datadog...
go run ./cmd/gateway
```

## Tests

```bash
go test ./...
```

Covers: statistical regression detection and tenant creation/lookup (real integration tests against local Postgres), the Anthropic request/response/streaming translation (against a mock server shaped like Anthropic's real API), semantic caching (pure cosine-similarity/index logic, plus full `Cache`-level plumbing against a mock embeddings server and real local Redis), cache-key normalization (streaming and non-streaming requests sharing an entry), the circuit breaker/fallback chain including dynamic latency-based reordering (using controllable fake providers, no external keys needed), guardrails (PII redaction patterns, injection-risk scoring), and the multi-tenant auth/rate-limit middleware (fake tenant lookup, no DB needed).

## What's here vs. what's next

This is a real, running system: a gateway with exact-match + semantic caching shared across streaming and non-streaming traffic; request logging with PII redaction and injection-risk scoring; async LLM-judge evaluation with statistical regression detection and tool-call grading; a live dashboard; real OTel instrumentation; two genuinely different provider adapters (OpenAI-compatible and Anthropic) automatically chained behind a circuit breaker with dynamic latency-based reordering when both are configured; per-tenant API keys, rate limits, and scoped dashboard queries; and a replay/diff tool for model-migration decisions.

Honest gaps worth knowing about: **live verification of the Anthropic adapter and semantic caching used mocks / a deliberately-invalid key, not a real Anthropic API key or a real embeddings key** (neither configured in the environment this was built in) — the logic is tested and correct, including a live test of the fallback *path* using a genuine invalid-key failure against the real Anthropic API, but a real Anthropic response has not been observed. Streaming tool-call capture isn't implemented (OpenAI's streaming format sends function-call fragments incrementally — reassembling them is real, separate work from the non-streaming case, which is done). Tool-call grading works from the judge's general knowledge, not the tool's formal JSON schema (Verigate doesn't currently capture the `tools` array from the request). Full roadmap and verification detail for every feature: `docs/ARCHITECTURE.md` §10.

## License

MIT — see [`LICENSE`](LICENSE).
