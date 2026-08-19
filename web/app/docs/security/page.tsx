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
        exactly once, at creation, with no way to recover it later (a lost key can be replaced from the dashboard,
        which invalidates the old one).
      </P>
      <P>Three ways to get a tenant + key:</P>
      <UL>
        <LI>
          Self-serve at <IC>/signup</IC> (email + password) — creates a user, an owned tenant, and a logged-in
          session together. This is how everyone other than the operator should get a key.
        </LI>
        <LI>
          The operator&apos;s dashboard at <IC>/tenants</IC>, gated behind <IC>VERIGATE_API_KEY</IC> — for manual/ad
          hoc tenants.
        </LI>
        <LI>
          <CodeBlock lang="bash" code={`go run ./cmd/tenant create --name acme --rpm 60`} /> — same admin-only path,
          from the CLI.
        </LI>
      </UL>

      <P>
        Rate limiting is a token bucket per tenant (<IC>golang.org/x/time/rate</IC>), sized to that tenant&apos;s own
        configured requests-per-minute, so one tenant&apos;s traffic can never starve another&apos;s. The static{" "}
        <IC>VERIGATE_API_KEY</IC> keeps working unchanged for single-tenant/local use and is never rate-limited — it&apos;s
        the operator&apos;s own key, not a customer&apos;s.
      </P>

      <H2>Dashboard scoping</H2>
      <P>
        <IC>GET /api/requests</IC>, <IC>/api/evals/summary</IC>, <IC>/api/evals/recent</IC>, and{" "}
        <IC>/api/tenants</IC> resolve <em className="not-italic">who&apos;s calling</em> from the{" "}
        <IC>Authorization</IC> header — a tenant API key or a session token from <IC>/login</IC> — and force the
        response to that caller&apos;s own tenant only, regardless of any <IC>tenant_id</IC> query param passed in. A
        request with no <IC>Authorization</IC> header sees only this deployment&apos;s own public demo traffic
        (rows with no tenant at all) — that&apos;s what powers the logged-out <IC>/dashboard</IC> anyone visiting
        this site sees. The operator&apos;s <IC>VERIGATE_API_KEY</IC> is the only identity with unscoped, full
        visibility, and the only one that may still target an arbitrary <IC>tenant_id</IC> explicitly.
      </P>

      <Callout>
        Write/admin operations — creating a tenant manually and triggering a replay run, which spends real provider
        credits — require <IC>VERIGATE_API_KEY</IC> specifically, never a tenant key or session token. Self-serve
        signup bypasses this by calling the store layer directly rather than going through that admin route.
      </Callout>
    </article>
  );
}
