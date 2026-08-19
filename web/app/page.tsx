"use client";

import Link from "next/link";
import { motion } from "motion/react";
import LandingNav from "@/components/landing/LandingNav";
import FlowDiagram from "@/components/landing/FlowDiagram";
import FeatureCard from "@/components/landing/FeatureCard";
import Reveal from "@/components/landing/Reveal";
import LiveBadge from "@/components/landing/LiveBadge";
import AnimatedBackground from "@/components/landing/AnimatedBackground";
import GradedDemo from "@/components/landing/GradedDemo";
import { IconEval, IconCache, IconShield, IconLock, IconUsers, IconPulse } from "@/components/landing/Icons";

// Each hero element animates independently with its own explicit delay,
// rather than through a parent/children `variants` chain — that pattern
// stopped propagating once a plain (non-motion) component like LiveBadge
// sat between a variant-driven parent and its motion.h1/motion.p
// descendants, leaving the whole hero stuck at its hidden state. Repeating
// initial/animate/transition per element is more verbose but each element
// animates on its own terms — nothing to silently break.
const EASE = [0.16, 1, 0.3, 1] as const;
function heroFade(delay: number) {
  return {
    initial: { opacity: 0, y: 18 },
    animate: { opacity: 1, y: 0 },
    transition: { duration: 0.55, delay, ease: EASE },
  };
}

export default function LandingPage() {
  return (
    <div className="min-h-screen">
      <LandingNav />

      {/* Hero */}
      <section className="relative overflow-hidden">
        <AnimatedBackground />
        <div className="relative mx-auto max-w-3xl px-6 pb-20 pt-12 text-center sm:pt-20">
          <motion.div {...heroFade(0)}>
            <LiveBadge />
          </motion.div>
          <motion.h1
            {...heroFade(0.08)}
            className="text-balance text-4xl font-semibold leading-[1.1] tracking-tight text-[color:var(--text-primary)] sm:text-6xl"
          >
            The gateway that knows
            <br className="hidden sm:block" /> when it&apos;s lying.
          </motion.h1>
          <motion.p
            {...heroFade(0.16)}
            className="mx-auto mt-5 max-w-2xl text-balance text-base leading-relaxed text-[color:var(--text-secondary)] sm:text-lg"
          >
            Verigate routes your LLM traffic <em className="not-italic text-[color:var(--text-primary)]">and</em>{" "}
            continuously grades it for quality — one open-source gateway, instead of stitching together a router and
            a separate observability tool that never talk to each other.
          </motion.p>
          <motion.div {...heroFade(0.24)} className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <motion.div whileHover={{ scale: 1.03 }} whileTap={{ scale: 0.97 }}>
              <Link
                href="/dashboard"
                className="block rounded-full px-5 py-2.5 text-sm font-medium"
                style={{ background: "var(--accent)", color: "var(--bg)" }}
              >
                View live dashboard
              </Link>
            </motion.div>
            <motion.div whileHover={{ scale: 1.03 }} whileTap={{ scale: 0.97 }}>
              <Link
                href="/docs"
                className="block rounded-full border border-[color:var(--border)] px-5 py-2.5 text-sm font-medium text-[color:var(--text-primary)] transition-colors hover:border-[color:var(--accent-border)]"
              >
                Read the docs
              </Link>
            </motion.div>
          </motion.div>

          <motion.div {...heroFade(0.32)}>
            <GradedDemo />
          </motion.div>
        </div>
      </section>

      {/* How it works */}
      <section className="mx-auto max-w-5xl px-6 pb-20">
        <Reveal className="mb-8 text-center">
          <h2 className="mb-2 text-2xl font-semibold tracking-tight text-[color:var(--text-primary)]">How it works</h2>
          <p className="mx-auto max-w-lg text-sm text-[color:var(--text-secondary)]">
            Every request is cached, routed, and guarded on the way through. A slice of live traffic is sampled and
            graded by a second model — continuously, not as a one-time benchmark.
          </p>
        </Reveal>
        <Reveal delay={0.1}>
          <FlowDiagram />
        </Reveal>
      </section>

      {/* The gap */}
      <section className="mx-auto max-w-5xl px-6 pb-20">
        <Reveal className="mb-8">
          <h2 className="text-center text-2xl font-semibold tracking-tight text-[color:var(--text-primary)]">
            Every other option makes you choose
          </h2>
        </Reveal>
        <div className="grid gap-4 sm:grid-cols-3">
          <Reveal delay={0}>
            <div className="h-full rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-5">
              <div className="text-xs font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">Gateway only</div>
              <div className="mt-1 text-sm text-[color:var(--text-secondary)]">Routes and caches. Has no idea if the answers are any good.</div>
            </div>
          </Reveal>
          <Reveal delay={0.1}>
            <div className="h-full rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-5">
              <div className="text-xs font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">Eval tool only</div>
              <div className="mt-1 text-sm text-[color:var(--text-secondary)]">Grades quality after the fact. Isn&apos;t in the request path at all.</div>
            </div>
          </Reveal>
          <Reveal delay={0.2}>
            <div className="h-full rounded-xl border p-5" style={{ borderColor: "var(--accent-border)", background: "var(--accent-soft)" }}>
              <div className="text-xs font-medium uppercase tracking-wider" style={{ color: "var(--accent)" }}>
                Verigate
              </div>
              <div className="mt-1 text-sm text-[color:var(--text-primary)]">
                Both, in one service — the same traffic gets routed <em className="not-italic">and</em> graded.
              </div>
            </div>
          </Reveal>
        </div>
      </section>

      {/* Features */}
      <section className="mx-auto max-w-5xl px-6 pb-20">
        <Reveal className="mb-8 text-center">
          <h2 className="text-2xl font-semibold tracking-tight text-[color:var(--text-primary)]">What&apos;s actually in it</h2>
        </Reveal>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Reveal delay={0}>
            <FeatureCard icon={<IconEval />} title="Continuous evaluation">
              A second model grades sampled responses against rubrics. Statistical regression detection compares the
              recent window to a trailing baseline — catches drift, not just a one-off benchmark.
            </FeatureCard>
          </Reveal>
          <Reveal delay={0.06}>
            <FeatureCard icon={<IconCache />} title="Smart caching">
              Exact-match and semantic (embedding-similarity) caching, sharing one entry across streaming and
              non-streaming requests for the same prompt.
            </FeatureCard>
          </Reveal>
          <Reveal delay={0.12}>
            <FeatureCard icon={<IconShield />} title="Automatic failover">
              A circuit breaker with latency-aware reordering across providers — OpenAI-compatible and Anthropic,
              translated transparently so the client never knows which served the response.
            </FeatureCard>
          </Reveal>
          <Reveal delay={0.18}>
            <FeatureCard icon={<IconLock />} title="Guardrails">
              PII and secrets redacted before anything hits the database, plus heuristic injection-risk scoring —
              applied to what Verigate stores, never to what&apos;s forwarded to the model.
            </FeatureCard>
          </Reveal>
          <Reveal delay={0.24}>
            <FeatureCard icon={<IconUsers />} title="Multi-tenancy">
              Per-tenant API keys, hashed at rest, each with its own independent rate limit — one tenant&apos;s traffic
              can&apos;t starve another&apos;s.
            </FeatureCard>
          </Reveal>
          <Reveal delay={0.3}>
            <FeatureCard icon={<IconPulse />} title="Real OpenTelemetry">
              GenAI semantic-convention spans and metrics out of the box — stdout by default, any OTLP backend with
              one environment variable.
            </FeatureCard>
          </Reveal>
        </div>
      </section>

      {/* Credibility strip */}
      <section className="border-y border-[color:var(--border)] bg-[color:var(--surface-1)] py-10">
        <Reveal className="mx-auto grid max-w-4xl grid-cols-2 gap-6 px-6 text-center sm:grid-cols-4">
          <div>
            <div className="font-mono text-2xl font-semibold text-[color:var(--text-primary)]">11</div>
            <div className="mt-1 text-xs text-[color:var(--text-tertiary)]">Go packages</div>
          </div>
          <div>
            <div className="font-mono text-2xl font-semibold text-[color:var(--text-primary)]">33</div>
            <div className="mt-1 text-xs text-[color:var(--text-tertiary)]">test functions</div>
          </div>
          <div>
            <div className="font-mono text-2xl font-semibold text-[color:var(--text-primary)]">~4.7k</div>
            <div className="mt-1 text-xs text-[color:var(--text-tertiary)]">lines of Go</div>
          </div>
          <div>
            <div className="font-mono text-2xl font-semibold text-[color:var(--text-primary)]">4</div>
            <div className="mt-1 text-xs text-[color:var(--text-tertiary)]">dashboard pages</div>
          </div>
        </Reveal>
      </section>

      <footer className="mx-auto max-w-5xl px-6 py-10 text-center text-xs text-[color:var(--text-tertiary)]">
        MIT licensed. Built with Go, Next.js, PostgreSQL, Redis, and OpenTelemetry.{" "}
        <Link href="/docs" className="underline decoration-dotted underline-offset-2 hover:text-[color:var(--text-secondary)]">
          Read the docs
        </Link>
        .
      </footer>
    </div>
  );
}
