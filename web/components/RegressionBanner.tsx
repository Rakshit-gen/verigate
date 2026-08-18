import type { EvalSummary } from "@/lib/api";

export default function RegressionBanner({ summary }: { summary?: EvalSummary }) {
  if (!summary || summary.sample_count === 0) {
    return (
      <div className="flex items-center gap-3 rounded-lg border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-slate-400">
        <span className="h-2.5 w-2.5 rounded-full bg-slate-600" />
        No eval samples yet — send some traffic through the gateway to see a quality signal here.
      </div>
    );
  }

  const regressed = summary.status === "regressed";
  const pct = Math.round(summary.rolling_avg_score * 100);
  const statistical = summary.method === "statistical";

  return (
    <div
      className={`rounded-lg border px-4 py-3 text-sm ${
        regressed
          ? "border-red-500/40 bg-red-500/10 text-red-300"
          : "border-emerald-500/30 bg-emerald-500/10 text-emerald-300"
      }`}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className={`h-2.5 w-2.5 rounded-full ${regressed ? "bg-red-400 animate-pulse" : "bg-emerald-400"}`} />
          <span className="font-medium">
            {regressed ? "Quality regression detected" : "All systems nominal"}
          </span>
          <span className="text-slate-400">
            rolling avg score {pct}% over last {summary.sample_count} sample{summary.sample_count === 1 ? "" : "s"}
          </span>
        </div>
        <span className="font-mono text-xs text-slate-500">status: {summary.status}</span>
      </div>

      <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-white/5 pt-2 text-xs text-slate-500">
        <span>
          method:{" "}
          <span className="font-mono text-slate-400">
            {statistical ? "z-test vs. baseline" : "fixed threshold (bootstrap)"}
          </span>
        </span>
        {statistical ? (
          <>
            <span>
              baseline avg: <span className="font-mono tabular-nums text-slate-400">{(summary.baseline_avg * 100).toFixed(0)}%</span>{" "}
              (n={summary.baseline_count})
            </span>
            <span>
              z-score: <span className="font-mono tabular-nums text-slate-400">{summary.z_score.toFixed(2)}</span>
            </span>
          </>
        ) : (
          <span>collecting baseline history — statistical z-test kicks in once there&apos;s enough data</span>
        )}
      </div>
    </div>
  );
}
