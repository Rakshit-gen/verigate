import { Fragment } from "react";
import type { ReplayResult } from "@/lib/api";

function delta(candidate: number, original: number) {
  const d = candidate - original;
  if (d > 0.05) return { symbol: "▲", cls: "text-emerald-400" };
  if (d < -0.05) return { symbol: "▼", cls: "text-red-400" };
  return { symbol: "≈", cls: "text-slate-500" };
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
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-500">Averages across {results.length} replayed request{results.length === 1 ? "" : "s"}</h3>
        <div className="flex flex-wrap gap-6">
          {averages.map((a) => {
            const d = delta(a.cand, a.orig);
            return (
              <div key={a.rubric} className="text-sm">
                <div className="text-xs text-slate-500">{a.rubric}</div>
                <div className="font-mono tabular-nums">
                  {(a.orig * 100).toFixed(0)}% → {(a.cand * 100).toFixed(0)}%{" "}
                  <span className={d.cls}>{d.symbol}</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="overflow-x-auto rounded-lg border border-slate-800">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-900 text-xs uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-2 font-medium">Prompt</th>
              <th className="px-4 py-2 font-medium">Model</th>
              {rubrics.map((r) => (
                <th key={r} className="px-4 py-2 font-medium text-right">{r}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800">
            {results.map((r, i) => (
              <Fragment key={i}>
                <tr className="hover:bg-slate-900/60">
                  <td rowSpan={2} className="px-4 py-2 align-top text-slate-300">
                    {r.prompt.length > 70 ? r.prompt.slice(0, 70) + "…" : r.prompt}
                  </td>
                  <td className="whitespace-nowrap px-4 py-2 font-mono text-xs text-slate-400">
                    {r.original_model} <span className="text-slate-600">(original)</span>
                  </td>
                  {rubrics.map((rubric) => (
                    <td key={rubric} className="px-4 py-2 text-right font-mono text-xs tabular-nums text-slate-400">
                      {((r.original_scores[rubric] ?? 0) * 100).toFixed(0)}%
                    </td>
                  ))}
                </tr>
                <tr className="hover:bg-slate-900/60">
                  <td className="whitespace-nowrap px-4 py-2 font-mono text-xs text-teal-300">
                    {r.candidate_model} <span className="text-slate-600">(candidate)</span>
                  </td>
                  {rubrics.map((rubric) => {
                    const d = delta(r.candidate_scores[rubric] ?? 0, r.original_scores[rubric] ?? 0);
                    return (
                      <td key={rubric} className="px-4 py-2 text-right font-mono text-xs tabular-nums text-slate-300">
                        {((r.candidate_scores[rubric] ?? 0) * 100).toFixed(0)}% <span className={d.cls}>{d.symbol}</span>
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
