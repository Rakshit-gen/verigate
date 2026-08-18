import type { ProviderStatusResponse } from "@/lib/api";

const stateStyle: Record<string, string> = {
  closed: "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
  half_open: "border-amber-500/30 bg-amber-500/10 text-amber-300",
  open: "border-red-500/40 bg-red-500/10 text-red-300",
};

const stateLabel: Record<string, string> = {
  closed: "healthy",
  half_open: "probing recovery",
  open: "failing over",
};

export default function ProvidersStatus({ data }: { data?: ProviderStatusResponse }) {
  if (!data) {
    return <div className="h-32 animate-pulse rounded-lg border border-slate-800 bg-slate-900/50" />;
  }

  return (
    <div>
      <p className="mb-4 text-sm text-slate-500">
        {data.mode === "chain"
          ? "Fallback chain configured — requests try each provider in order below, skipping any with an open breaker."
          : "Single provider configured — no fallback chain. Set both an OpenAI-compatible and an Anthropic credential to enable one."}
      </p>
      <div className="grid gap-3 sm:grid-cols-2">
        {data.providers.map((p) => (
          <div key={p.name} className={`rounded-lg border p-4 ${stateStyle[p.breaker_state] ?? "border-slate-800 bg-slate-900/50 text-slate-300"}`}>
            <div className="flex items-center justify-between">
              <span className="font-mono text-sm font-medium">{p.name}</span>
              <span className="rounded-full bg-black/20 px-2 py-0.5 text-xs">
                {stateLabel[p.breaker_state] ?? p.breaker_state}
              </span>
            </div>
            <div className="mt-3 grid grid-cols-3 gap-2 text-xs opacity-80">
              <div>
                <div className="text-[10px] uppercase tracking-wide opacity-60">Priority</div>
                <div className="font-mono tabular-nums">#{p.declared_order + 1}</div>
              </div>
              <div>
                <div className="text-[10px] uppercase tracking-wide opacity-60">Consec. failures</div>
                <div className="font-mono tabular-nums">{p.consecutive_failures}</div>
              </div>
              <div>
                <div className="text-[10px] uppercase tracking-wide opacity-60">Measured latency</div>
                <div className="font-mono tabular-nums">
                  {p.measured_latency_ms > 0 ? `${p.measured_latency_ms.toFixed(0)}ms` : "no data yet"}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
