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
  auth: "gateway key" | "scoped" | "admin key" | "public";
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
        Every route Verigate exposes. All auth is <IC>Authorization: Bearer &lt;token&gt;</IC> — a tenant API key, a
        session token from <IC>/api/auth/login</IC>, or the operator&apos;s <IC>VERIGATE_API_KEY</IC>, depending on
        the route. <IC>scoped</IC> routes below work with or without a token — see the note at the bottom.
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

      <H2>Account</H2>
      <Endpoint method="POST" path="/api/auth/signup" auth="public">
        <P>
          Create an account: <IC>{`{"email": "...", "password": "..."}`}</IC> (password min 8 chars). Creates a
          user, an owned tenant, and a session together. Response includes the plaintext API key (shown once) and a{" "}
          <IC>session_token</IC> for the browser to use going forward.
        </P>
      </Endpoint>
      <Endpoint method="POST" path="/api/auth/login" auth="public">
        <P>
          <IC>{`{"email": "...", "password": "..."}`}</IC> → a fresh <IC>session_token</IC>, valid 30 days.
        </P>
      </Endpoint>
      <Endpoint method="POST" path="/api/auth/logout" auth="scoped">
        <P>Invalidates the session token sent in <IC>Authorization</IC>. Idempotent.</P>
      </Endpoint>
      <Endpoint method="GET" path="/api/me" auth="scoped">
        <P>Who the current token resolves to — <IC>{`{"tenant": {...} | null}`}</IC>.</P>
      </Endpoint>
      <Endpoint method="POST" path="/api/tenant/regenerate-key" auth="scoped">
        <P>
          Issues a fresh API key for the caller&apos;s own tenant and invalidates the old one — the recovery path for
          a lost key. Requires a tenant key or session token (rejects an anonymous or admin-only caller, since
          neither has a tenant of their own to regenerate).
        </P>
      </Endpoint>

      <H2>Dashboard reads</H2>
      <P>
        These resolve scope from whatever <IC>Authorization</IC> is presented instead of trusting a client-supplied{" "}
        <IC>tenant_id</IC>: a tenant/session token sees only that tenant&apos;s own data, no token sees only public
        demo traffic, and <IC>VERIGATE_API_KEY</IC> sees everything (and may still pass <IC>?tenant_id=</IC> to
        target one tenant explicitly).
      </P>
      <Endpoint method="GET" path="/api/requests" auth="scoped">
        <P>Recent requests, newest first, scoped per the rules above.</P>
      </Endpoint>
      <Endpoint method="GET" path="/api/evals/summary" auth="scoped">
        <P>
          Rolling regression status, scoped to the caller&apos;s own tenant: <IC>rolling_avg_score</IC>,{" "}
          <IC>baseline_avg</IC>, <IC>z_score</IC>, <IC>method</IC> (<IC>statistical</IC> or{" "}
          <IC>fixed_threshold_bootstrap</IC>), and <IC>status</IC>.
        </P>
      </Endpoint>
      <Endpoint method="GET" path="/api/evals/recent" auth="scoped">
        <P>The most recent individual eval scores, with per-rubric reasoning from the judge — scoped.</P>
      </Endpoint>
      <Endpoint method="GET" path="/api/tenants" auth="scoped">
        <P>
          The admin key lists every tenant (name, rate limit, creation time — never an API key); a tenant/session
          token sees only its own tenant; an anonymous caller sees an empty list.
        </P>
      </Endpoint>
      <Endpoint method="GET" path="/api/providers" auth="public">
        <P>
          Live provider status: <IC>mode</IC> (<IC>single</IC> or <IC>chain</IC>) and, per provider,{" "}
          <IC>breaker_state</IC>, <IC>consecutive_failures</IC>, and <IC>measured_latency_ms</IC>.
        </P>
      </Endpoint>

      <H2>Admin</H2>
      <Endpoint method="POST" path="/api/tenants" auth="admin key">
        <P>
          Manually create a tenant: <IC>{`{"name": "...", "rate_limit_rpm": 60}`}</IC>. The response includes the
          plaintext API key — the only time it&apos;s ever visible. Most callers should use{" "}
          <IC>/api/auth/signup</IC> instead; this exists for the operator&apos;s own ad hoc/demo tenants.
        </P>
      </Endpoint>
      <Endpoint method="POST" path="/api/replay" auth="admin key">
        <P>
          Replay recent (or one specific) requests through a candidate model:{" "}
          <IC>{`{"candidate_model": "...", "limit": 5, "request_id": ""}`}</IC>. Synchronous — a few seconds for real
          judge calls. <IC>limit</IC> is capped at 10.
        </P>
      </Endpoint>

      <Callout>
        Every route above requires <IC>VERIGATE_API_KEY</IC> specifically — not a tenant key or session token —
        because both spend real provider credits or create billable resources. This is the same reasoning that
        governs which reads are public vs. scoped: see the <IC>/docs/security</IC> page.
      </Callout>
    </article>
  );
}
