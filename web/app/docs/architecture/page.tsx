import { H1, Lead, H2, H3, P, UL, LI, IC, Callout, Table } from "@/components/docs/Prose";

export default function ArchitecturePage() {
  return (
    <article>
      <H1>Architecture</H1>
      <Lead>How a request actually moves through Verigate, and the real tradeoffs behind each piece.</Lead>

      <H2>Request path</H2>
      <P>
        A chat completion checks the cache first (exact-match, then semantic if configured), then the provider
        chain — a single provider, or a priority-ordered fallback chain if both an OpenAI-compatible and an
        Anthropic credential are configured. Every request is logged, redacted for PII/secrets before storage, scored
        for injection risk, and sampled for evaluation regardless of which path served it.
      </P>

      <H3>Caching</H3>
      <P>
        The cache key is a hash of the request body <em className="not-italic">normalized</em> — <IC>stream</IC> and{" "}
        <IC>stream_options</IC> stripped before hashing, and JSON keys sorted by Go&apos;s own marshaling. That means a
        streaming request and a non-streaming request for the identical prompt share one cache entry: a streaming hit
        synthesizes an SSE reply from the cached completion instead of replaying the original chunk boundaries — a
        cache hit is already near-instant, so reconstructing exact chunk timing wasn&apos;t worth the complexity.
      </P>
      <P>
        Semantic caching is optional and off by default — it needs a real embeddings-capable key (Groq, for one,
        doesn&apos;t serve embeddings). When enabled, a miss on the exact-match layer falls through to an in-process,
        TTL-bounded nearest-neighbor index over embedded prompts. Deliberately brute-force rather than a real ANN
        index: at hundreds of TTL&apos;d entries that&apos;s the correctly-scoped choice, not a shortcut.
      </P>

      <H3>Provider chain</H3>
      <P>
        A standard three-state circuit breaker (closed / open / half-open) per provider. A provider that fails
        <IC>FAILURE_THRESHOLD</IC> consecutive times gets skipped automatically; traffic fails over to the next entry
        until the cooldown elapses and a single health-check call proves recovery. Among providers currently allowed,
        the chain tries whichever is <em className="not-italic">measured</em> fastest first — an exponential moving
        average of real call latency, not just the declared config order.
      </P>

      <H3>Evaluation & regression detection</H3>
      <P>
        A sampler rolls a configurable rate on every logged request and hands sampled IDs to a worker pool. Each
        worker grades the (prompt, response) pair against every rubric with a second model, via strict-JSON prompts.
        A pure tool-call turn — empty text content, a populated <IC>tool_calls</IC> field — skips the text rubrics
        entirely and runs a dedicated tool-call-correctness rubric instead; grading empty text against
        &quot;groundedness&quot; would just poison the baseline.
      </P>
      <P>
        Regression detection compares the mean of the recent window against the mean of a trailing baseline window
        via a one-sample z-test, scoped per-rubric. It falls back to a fixed-floor check automatically when there
        isn&apos;t yet enough baseline history (fewer than 10 points, or zero variance) — a brand-new deployment has
        no baseline to test against, so a blunt floor beats silence.
      </P>

      <H2>What each piece deliberately doesn&apos;t do</H2>
      <Table
        head={["Piece", "Scoped out (for now)", "Why"]}
        rows={[
          ["Streaming", "Tool-call capture mid-stream", "OpenAI's streaming format sends function-call fragments incrementally — reassembling them is separate work from the buffered case."],
          ["Guardrails", "Blocking/filtering live traffic", "Redaction applies to what Verigate stores, never to what's forwarded to the model — altering the live request would silently corrupt it."],
          ["Provider chain", "Dynamic cost prediction", "The chain's declared order is the routing policy; true per-call cost isn't known until after the call completes."],
          ["Tool-call grading", "Tool-schema-aware scoring", "Verigate doesn't currently capture the tools array from the original request — grading works from the judge's general knowledge instead."],
        ]}
      />

      <Callout tone="warning">
        This documents real, current scope — not a promise. Anything listed here as &quot;not done&quot; is genuinely not
        done, tracked in the repository&apos;s architecture notes rather than glossed over.
      </Callout>
    </article>
  );
}
