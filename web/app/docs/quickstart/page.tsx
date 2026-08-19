import { H1, Lead, H2, P, UL, LI, IC, Callout } from "@/components/docs/Prose";
import CodeBlock from "@/components/CodeBlock";

export default function QuickstartPage() {
  return (
    <article>
      <H1>Quickstart</H1>
      <Lead>Go 1.25+, Node 20+, Postgres, and Redis — all running locally is fine, no Docker required.</Lead>

      <H2>1. Database</H2>
      <CodeBlock
        lang="bash"
        code={`createdb verigate
for f in migrations/*.sql; do psql verigate -f "$f"; done`}
      />

      <H2>2. Backend</H2>
      <CodeBlock
        lang="bash"
        code={`cp .env.example .env
# edit .env — at minimum set a provider key (OpenAI, Groq, or Ollama block)
go run ./cmd/gateway
# -> verigate listening on :8080`}
      />
      <Callout>
        Groq is the fastest way to get a real key with no card required — the OpenAI-compatible block in{" "}
        <IC>.env.example</IC> already has it filled in, just paste your key.
      </Callout>

      <H2>3. Frontend</H2>
      <CodeBlock
        lang="bash"
        code={`cd web
cp .env.local.example .env.local
npm install
npm run dev
# -> http://localhost:3000`}
      />

      <H2>See it catch a regression on purpose</H2>
      <P>
        Sends real traffic through the gateway, then injects a batch of deliberately bad responses so the eval
        worker scores them low — the dashboard&apos;s regression banner should flip within a few seconds.
      </P>
      <CodeBlock lang="bash" code="go run ./scripts/seed_demo_traffic.go" />

      <H2>Point an existing app at it</H2>
      <P>Same request/response shape as OpenAI&apos;s own API — a drop-in proxy, not a new SDK to learn:</P>
      <CodeBlock
        lang="bash"
        code={`curl http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer dev-local-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}'`}
      />

      <H2>Run the tests</H2>
      <CodeBlock lang="bash" code="go test ./..." />
      <P>
        Covers statistical regression detection and tenant lookups against real local Postgres, the Anthropic
        translation against a mock shaped like Anthropic&apos;s real API, semantic caching against a mock embeddings
        server and real local Redis, and the circuit breaker/fallback chain with controllable fake providers — no
        external keys needed for most of it.
      </P>

      <H2>Everything else</H2>
      <UL>
        <LI>
          <IC>go run ./cmd/tenant create --name acme --rpm 60</IC> — create a per-tenant API key from the CLI (or use
          the dashboard at <IC>/tenants</IC>).
        </LI>
        <LI>
          <IC>go run ./cmd/replay --candidate-model &lt;model&gt; --limit 5</IC> — replay recent prompts through a
          candidate model (or use <IC>/replay</IC>).
        </LI>
        <LI>
          <IC>OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/gateway</IC> — ship spans/metrics to a
          real OTel Collector instead of stdout.
        </LI>
      </UL>
    </article>
  );
}
