import type { ProviderStatusResponse } from "@/lib/api";

const stateTone: Record<string, { fg: string; bg: string; border: string; label: string }> = {
  closed: { fg: "var(--status-good)", bg: "var(--status-good-bg)", border: "var(--status-good-border)", label: "healthy" },
  half_open: {
    fg: "var(--status-warning)",
    bg: "var(--status-warning-bg)",
    border: "var(--status-warning-border)",
    label: "probing recovery",
  },
  open: {
    fg: "var(--status-critical)",
    bg: "var(--status-critical-bg)",
    border: "var(--status-critical-border)",
    label: "failing over",
  },
};

export default function ProvidersStatus({ data }: { data?: ProviderStatusResponse }) {
  if (!data) {
    return (
      <div className="grid gap-3 sm:grid-cols-2">
        {[0, 1].map((i) => (
          <div key={i} className="h-32 animate-pulse rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)]" />
        ))}
      </div>
    );
  }

  const maxLatency = Math.max(1, ...data.providers.map((p) => p.measured_latency_ms));

  return (
    <div>
      <p className="mb-4 text-sm text-[color:var(--text-secondary)]">
        {data.mode === "chain"
          ? "Fallback chain configured — requests try each provider in order below, skipping any with an open breaker, and prefer whichever is currently measured fastest."
          : "Single provider configured — no fallback chain. Set both an OpenAI-compatible and an Anthropic credential to enable one."}
      </p>
      <div className="grid gap-3 sm:grid-cols-2">
        {data.providers.map((p) => {
          const tone = stateTone[p.breaker_state] ?? stateTone.closed;
          const latencyPct = p.measured_latency_ms > 0 ? Math.max(4, (p.measured_latency_ms / maxLatency) * 100) : 0;
          return (
            <div
              key={p.name}
              className="animate-fade-in-up rounded-xl border p-4"
              style={{ borderColor: tone.border, background: tone.bg }}
            >
              <div className="flex items-center justify-between">
                <span className="font-mono text-sm font-medium" style={{ color: "var(--text-primary)" }}>
                  {p.name}
                </span>
                <span
                  className="flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium"
                  style={{ background: "var(--bg)", color: tone.fg }}
                >
                  <span className={`h-1.5 w-1.5 rounded-full ${p.breaker_state === "open" ? "animate-pulse" : ""}`} style={{ background: tone.fg }} />
                  {tone.label}
                </span>
              </div>

              <div className="mt-4 grid grid-cols-2 gap-3 text-xs">
                <div>
                  <div className="text-[10px] uppercase tracking-wider text-[color:var(--text-tertiary)]">Priority</div>
                  <div className="font-mono tabular-nums text-[color:var(--text-primary)]">#{p.declared_order + 1}</div>
                </div>
                <div>
                  <div className="text-[10px] uppercase tracking-wider text-[color:var(--text-tertiary)]">Consec. failures</div>
                  <div className="font-mono tabular-nums text-[color:var(--text-primary)]">{p.consecutive_failures}</div>
                </div>
              </div>

              <div className="mt-3">
                <div className="mb-1 flex items-baseline justify-between text-[10px] uppercase tracking-wider text-[color:var(--text-tertiary)]">
                  <span>Measured latency</span>
                  <span className="font-mono normal-case tracking-normal text-[color:var(--text-secondary)]">
                    {p.measured_latency_ms > 0 ? `${p.measured_latency_ms.toFixed(0)}ms` : "no data yet"}
                  </span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-[color:var(--bg)]">
                  <div
                    className="h-full rounded-full transition-all"
                    style={{ width: `${latencyPct}%`, background: tone.fg }}
                  />
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
