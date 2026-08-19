import Link from "next/link";
import { H1, Lead, H2, P, UL, LI, IC } from "@/components/docs/Prose";

export default function DocsOverviewPage() {
  return (
    <article>
      <H1>Overview</H1>
      <Lead>
        Verigate is an LLM gateway that routes your traffic and continuously grades it for quality — one open-source
        service, instead of a router and a separate observability tool stitched together.
      </Lead>

      <H2>The gap it fills</H2>
      <P>
        Every major LLM gateway picks one lane. Routers (LiteLLM) cache and route but have no idea whether the
        responses are any good. Observability tools (Langfuse, Helicone) grade quality after the fact, but sit
        outside the request path entirely. The common pattern teams settle for is stitching both together — a router
        for traffic, a separate tool for logging, a separate service again for guardrails.
      </P>
      <P>
        Verigate does the routing and the grading in the same request path: a slice of live traffic is sampled and
        scored by a second model as it flows through, with statistical regression detection comparing the recent
        window against a trailing baseline — so a quality drop shows up as an alert, not as something a user reports
        three weeks later.
      </P>

      <H2>What&apos;s in it</H2>
      <UL>
        <LI>An OpenAI-compatible passthrough gateway with exact-match and semantic caching, shared across streaming and non-streaming traffic.</LI>
        <LI>A second provider adapter (Anthropic) with real request/response translation — the client never knows which provider actually served it.</LI>
        <LI>A circuit breaker with latency-aware reordering across providers, for automatic failover.</LI>
        <LI>Continuous LLM-judge evaluation with statistical (z-test) regression detection and tool-call grading.</LI>
        <LI>Guardrails: PII/secret redaction and injection-risk scoring on everything stored.</LI>
        <LI>Multi-tenancy: per-tenant API keys, hashed at rest, with independent rate limits.</LI>
        <LI>Real OpenTelemetry instrumentation using the GenAI semantic conventions.</LI>
        <LI>A replay/diff tool for model-migration decisions, and a live dashboard for all of it.</LI>
      </UL>

      <H2>Where to go next</H2>
      <P>
        <Link href="/docs/quickstart" className="text-[color:var(--accent)] hover:underline">
          Quickstart
        </Link>{" "}
        gets a local instance running end to end.{" "}
        <Link href="/docs/architecture" className="text-[color:var(--accent)] hover:underline">
          Architecture
        </Link>{" "}
        explains how the pieces fit together and what tradeoffs each one made.{" "}
        <Link href="/docs/api" className="text-[color:var(--accent)] hover:underline">
          API reference
        </Link>{" "}
        covers every endpoint.
      </P>
      <P>
        The gateway route itself is deliberately boring: <IC>POST /v1/chat/completions</IC>, same shape as OpenAI&apos;s
        own API. Point your existing client&apos;s <IC>base_url</IC> at Verigate and change nothing else.
      </P>
    </article>
  );
}
