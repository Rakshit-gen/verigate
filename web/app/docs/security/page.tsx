import { H1, Lead, H2, P, UL, LI, IC, Callout, Table } from "@/components/docs/Prose";
import CodeBlock from "@/components/CodeBlock";

export default function SecurityPage() {
  return (
    <article>
      <H1>Guardrails &amp; multi-tenancy</H1>
      <Lead>What protects Verigate&apos;s own data, and how more than one consumer shares a single deployment.</Lead>

      <H2>Guardrails</H2>
      <P>
        Guardrails protect what Verigate <em className="not-italic">stores</em> — they never touch what&apos;s forwarded to
        the actual provider or returned to the caller. Redacting a live prompt before sending it to the model would
        silently corrupt the request the caller asked for; that&apos;s a different, unbuilt feature.
      </P>
      <Table
        head={["Signal", "What it catches", "Where it's applied"]}
        rows={[
          ["PII/secret redaction", "Email, credit card, SSN, phone, common API-key prefixes (regex-based)", "Prompt & response text before the requests table INSERT"],
          ["Injection-risk score", "Weighted pattern match — “ignore previous instructions,” jailbreak phrasing, etc.", "0–1 score stored on every request, not blocked or altered"],
        ]}
      />
      <P>
        This is a heuristic first pass, not a compliance-grade DLP system — favor missing something over mangling
        normal text. A determined attacker can phrase around any fixed pattern list; the point is a cheap signal
        worth logging and alerting on for near-zero cost, not a hard block.
      </P>

      <H2>Multi-tenancy</H2>
      <P>
        Each tenant gets an independent API key and rate limit. Keys are generated as{" "}
        <IC>vg_&lt;64 hex chars&gt;</IC>, and only their SHA-256 hash is ever persisted — the plaintext is shown
        exactly once, at creation, with no way to recover it later.
      </P>
      <CodeBlock lang="bash" code={`go run ./cmd/tenant create --name acme --rpm 60`} />
      <P>Or from the dashboard at <IC>/tenants</IC> — same underlying endpoint either way.</P>

      <P>
        Rate limiting is a token bucket per tenant (<IC>golang.org/x/time/rate</IC>), sized to that tenant&apos;s own
        configured requests-per-minute, so one tenant&apos;s traffic can never starve another&apos;s. The static{" "}
        <IC>VERIGATE_API_KEY</IC> keeps working unchanged for single-tenant/local use and is never rate-limited — it&apos;s
        the operator&apos;s own key, not a customer&apos;s.
      </P>

      <H2>What&apos;s tenant-scoped today</H2>
      <UL>
        <LI>
          <IC>GET /api/requests?tenant_id=&lt;id&gt;</IC> — scoped.
        </LI>
        <LI>Eval summary and recent evals — not yet tenant-scoped; they report across all tenants.</LI>
      </UL>

      <Callout>
        The dashboard&apos;s <IC>/api</IC> routes are unauthenticated by design — a local-only demo surface, not a route
        real client traffic hits. Tenant creation in particular is an admin operation: add auth there before deploying
        anywhere reachable off localhost.
      </Callout>
    </article>
  );
}
