import { H1, Lead, H2, P, IC, Callout } from "@/components/docs/Prose";
import CodeBlock from "@/components/CodeBlock";

function Endpoint({
  method,
  path,
  auth,
  children,
}: {
  method: "GET" | "POST";
  path: string;
  auth: "gateway key" | "none";
  children: React.ReactNode;
}) {
  const methodColor = method === "GET" ? "var(--status-info)" : "var(--accent)";
  return (
    <div className="mb-8 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-5">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span
          className="rounded px-2 py-0.5 font-mono text-xs font-semibold"
          style={{ background: "var(--surface-2)", color: methodColor }}
        >
          {method}
        </span>
        <code className="font-mono text-sm text-[color:var(--text-primary)]">{path}</code>
        <span className="ml-auto rounded-full bg-[color:var(--surface-2)] px-2 py-0.5 text-[10px] uppercase tracking-wider text-[color:var(--text-tertiary)]">
          {auth}
        </span>
      </div>
      <div className="text-sm leading-relaxed text-[color:var(--text-secondary)]">{children}</div>
    </div>
  );
}

export default function ApiReferencePage() {
  return (
    <article>
      <H1>API reference</H1>
      <Lead>
        Every route Verigate exposes. Gateway routes require{" "}
        <IC>Authorization: Bearer &lt;key&gt;</IC> (the static key or a tenant key); dashboard routes are
        unauthenticated by design — see the note at the bottom.
      </Lead>

      <H2>Gateway</H2>
      <Endpoint method="POST" path="/v1/chat/completions" auth="gateway key">
        <P>
          OpenAI-shaped passthrough. Same request/response contract as OpenAI&apos;s own API, including{" "}
          <IC>&quot;stream&quot;: true</IC> for real server-sent events. Cached, evaluated, and logged identically
          regardless of which provider actually serves it.
        </P>
        <CodeBlock
          lang="bash"
          code={`curl .../v1/chat/completions \\
  -H "Authorization: Bearer $KEY" \\
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}'`}
        />
      </Endpoint>

      <H2>Dashboard</H2>
      <Endpoint method="GET" path="/api/requests?tenant_id=" auth="none">
        <P>Recent requests, newest first. <IC>tenant_id</IC> is optional — omit it to see traffic across all tenants.</P>
      </Endpoint>
      <Endpoint method="GET" path="/api/evals/summary" auth="none">
        <P>
          Rolling regression status: <IC>rolling_avg_score</IC>, <IC>baseline_avg</IC>, <IC>z_score</IC>,{" "}
          <IC>method</IC> (<IC>statistical</IC> or <IC>fixed_threshold_bootstrap</IC>), and <IC>status</IC>.
        </P>
      </Endpoint>
      <Endpoint method="GET" path="/api/evals/recent" auth="none">
        <P>The most recent individual eval scores, with per-rubric reasoning from the judge.</P>
      </Endpoint>
      <Endpoint method="GET" path="/api/tenants" auth="none">
        <P>List tenants — name, rate limit, creation time. Never returns an API key.</P>
      </Endpoint>
      <Endpoint method="POST" path="/api/tenants" auth="none">
        <P>
          Create a tenant: <IC>{`{"name": "...", "rate_limit_rpm": 60}`}</IC>. The response includes the plaintext
          API key — the only time it&apos;s ever visible.
        </P>
      </Endpoint>
      <Endpoint method="GET" path="/api/providers" auth="none">
        <P>
          Live provider status: <IC>mode</IC> (<IC>single</IC> or <IC>chain</IC>) and, per provider,{" "}
          <IC>breaker_state</IC>, <IC>consecutive_failures</IC>, and <IC>measured_latency_ms</IC>.
        </P>
      </Endpoint>
      <Endpoint method="POST" path="/api/replay" auth="none">
        <P>
          Replay recent (or one specific) requests through a candidate model:{" "}
          <IC>{`{"candidate_model": "...", "limit": 5, "request_id": ""}`}</IC>. Synchronous — a few seconds for real
          judge calls. <IC>limit</IC> is capped at 10.
        </P>
      </Endpoint>

      <Callout tone="warning">
        Dashboard routes have no auth today — a local-only surface, not something real client traffic hits. Tenant
        creation is an admin operation: add auth before deploying anywhere reachable off localhost.
      </Callout>
    </article>
  );
}
