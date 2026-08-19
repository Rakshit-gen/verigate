import { Fragment } from "react";
import type { ReplayResult } from "@/lib/api";

function delta(candidate: number, original: number) {
  const d = candidate - original;
  if (d > 0.05) return { symbol: "▲", color: "var(--status-good)" };
  if (d < -0.05) return { symbol: "▼", color: "var(--status-critical)" };
  return { symbol: "≈", color: "var(--text-tertiary)" };
}

export default function ReplayResults({ results }: { results: ReplayResult[] }) {
  if (results.length === 0) return null;

  const rubrics = Array.from(new Set(results.flatMap((r) => Object.keys(r.original_scores)))).sort();

  const averages = rubrics.map((rubric) => {
    const orig = results.reduce((sum, r) => sum + (r.original_scores[rubric] ?? 0), 0) / results.length;
    const cand = results.reduce((sum, r) => sum + (r.candidate_scores[rubric] ?? 0), 0) / results.length;
    return { rubric, orig, cand };
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-4">
        <h3 className="mb-3 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
          Averages across {results.length} replayed request{results.length === 1 ? "" : "s"}
        </h3>
        <div className="flex flex-wrap gap-6">
          {averages.map((a) => {
            const d = delta(a.cand, a.orig);
            return (
              <div key={a.rubric} className="text-sm">
                <div className="text-xs text-[color:var(--text-tertiary)]">{a.rubric}</div>
                <div className="font-mono tabular-nums text-[color:var(--text-primary)]">
                  {(a.orig * 100).toFixed(0)}% → {(a.cand * 100).toFixed(0)}%{" "}
                  <span style={{ color: d.color }}>{d.symbol}</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="overflow-x-auto rounded-xl border border-[color:var(--border)]">
        <table className="w-full text-left text-sm">
          <thead className="bg-[color:var(--surface-2)] text-[11px] uppercase tracking-wider text-[color:var(--text-tertiary)]">
            <tr>
              <th className="px-4 py-2.5 font-medium">Prompt</th>
              <th className="px-4 py-2.5 font-medium">Model</th>
              {rubrics.map((r) => (
                <th key={r} className="px-4 py-2.5 font-medium text-right">
                  {r}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-[color:var(--border-soft)]">
            {results.map((r, i) => (
              <Fragment key={i}>
                <tr className="bg-[color:var(--surface-1)] transition-colors hover:bg-[color:var(--surface-3)]">
                  <td rowSpan={2} className="px-4 py-2.5 align-top text-[color:var(--text-secondary)]">
                    {r.prompt.length > 70 ? r.prompt.slice(0, 70) + "…" : r.prompt}
                  </td>
                  <td className="whitespace-nowrap px-4 py-2.5 font-mono text-xs text-[color:var(--text-secondary)]">
                    {r.original_model} <span className="text-[color:var(--text-tertiary)]">(original)</span>
                  </td>
                  {rubrics.map((rubric) => (
                    <td
                      key={rubric}
                      className="px-4 py-2.5 text-right font-mono text-xs tabular-nums text-[color:var(--text-secondary)]"
                    >
                      {((r.original_scores[rubric] ?? 0) * 100).toFixed(0)}%
                    </td>
                  ))}
                </tr>
                <tr className="bg-[color:var(--surface-1)] transition-colors hover:bg-[color:var(--surface-3)]">
                  <td className="whitespace-nowrap px-4 py-2.5 font-mono text-xs text-[color:var(--accent)]">
                    {r.candidate_model} <span className="text-[color:var(--text-tertiary)]">(candidate)</span>
                  </td>
                  {rubrics.map((rubric) => {
                    const d = delta(r.candidate_scores[rubric] ?? 0, r.original_scores[rubric] ?? 0);
                    return (
                      <td
                        key={rubric}
                        className="px-4 py-2.5 text-right font-mono text-xs tabular-nums text-[color:var(--text-primary)]"
                      >
                        {((r.candidate_scores[rubric] ?? 0) * 100).toFixed(0)}%{" "}
                        <span style={{ color: d.color }}>{d.symbol}</span>
                      </td>
                    );
                  })}
                </tr>
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
