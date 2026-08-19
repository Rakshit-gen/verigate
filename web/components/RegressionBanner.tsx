import type { EvalSummary } from "@/lib/api";

export default function RegressionBanner({ summary }: { summary?: EvalSummary }) {
  if (!summary || summary.sample_count === 0) {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] px-4 py-3 text-sm text-[color:var(--text-secondary)]">
        <span className="h-2.5 w-2.5 rounded-full bg-[color:var(--text-tertiary)]" />
        No eval samples yet — send some traffic through the gateway to see a quality signal here.
      </div>
    );
  }

  const regressed = summary.status === "regressed";
  const pct = Math.round(summary.rolling_avg_score * 100);
  const statistical = summary.method === "statistical";

  return (
    <div
      className="rounded-xl border px-4 py-3 text-sm"
      style={{
        borderColor: regressed ? "var(--status-critical-border)" : "var(--status-good-border)",
        background: regressed ? "var(--status-critical-bg)" : "var(--status-good-bg)",
        color: regressed ? "var(--status-critical)" : "var(--status-good)",
      }}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-3">
          <span
            className={`h-2.5 w-2.5 rounded-full ${regressed ? "animate-pulse" : ""}`}
            style={{ background: regressed ? "var(--status-critical)" : "var(--status-good)" }}
          />
          <span className="font-medium">{regressed ? "Quality regression detected" : "All systems nominal"}</span>
          <span className="text-[color:var(--text-secondary)]">
            rolling avg score {pct}% over last {summary.sample_count} sample{summary.sample_count === 1 ? "" : "s"}
          </span>
        </div>
        <span className="font-mono text-xs text-[color:var(--text-tertiary)]">status: {summary.status}</span>
      </div>

      <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-white/5 pt-2 text-xs text-[color:var(--text-secondary)]">
        <span>
          method:{" "}
          <span className="font-mono text-[color:var(--text-primary)]">
            {statistical ? "z-test vs. baseline" : "fixed threshold (bootstrap)"}
          </span>
        </span>
        {statistical ? (
          <>
            <span>
              baseline avg:{" "}
              <span className="font-mono tabular-nums text-[color:var(--text-primary)]">
                {(summary.baseline_avg * 100).toFixed(0)}%
              </span>{" "}
              (n={summary.baseline_count})
            </span>
            <span>
              z-score:{" "}
              <span className="font-mono tabular-nums text-[color:var(--text-primary)]">{summary.z_score.toFixed(2)}</span>
            </span>
          </>
        ) : (
          <span>collecting baseline history — statistical z-test kicks in once there&apos;s enough data</span>
        )}
      </div>
    </div>
  );
}
