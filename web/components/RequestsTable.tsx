import type { RequestRecord } from "@/lib/api";

function truncate(s: string, max = 60) {
  return s.length > max ? s.slice(0, max) + "…" : s;
}

export default function RequestsTable({ requests }: { requests: RequestRecord[] }) {
  if (requests.length === 0) {
    return (
      <div className="flex h-40 items-center justify-center rounded-lg border border-slate-800 bg-slate-900/50 text-sm text-slate-500">
        No requests logged yet. Point a client at{" "}
        <code className="mx-1 rounded bg-slate-800 px-1.5 py-0.5 font-mono text-xs">/v1/chat/completions</code>
        to see traffic here.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-800">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-900 text-xs uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-2 font-medium">Time</th>
            <th className="px-4 py-2 font-medium">Model</th>
            <th className="px-4 py-2 font-medium">Prompt</th>
            <th className="px-4 py-2 font-medium text-right">Latency</th>
            <th className="px-4 py-2 font-medium text-center">Cache</th>
            <th className="px-4 py-2 font-medium text-right">Tokens</th>
            <th className="px-4 py-2 font-medium text-right">Cost</th>
            <th className="px-4 py-2 font-medium text-center">Status</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {requests.map((r) => (
            <tr key={r.id} className="hover:bg-slate-900/60">
              <td className="whitespace-nowrap px-4 py-2 font-mono text-xs text-slate-500">
                {new Date(r.created_at).toLocaleTimeString()}
              </td>
              <td className="whitespace-nowrap px-4 py-2 font-mono text-xs text-teal-300">{r.model}</td>
              <td className="px-4 py-2 text-slate-300">{truncate(r.prompt)}</td>
              <td className="whitespace-nowrap px-4 py-2 text-right font-mono text-xs tabular-nums text-slate-400">
                {r.latency_ms}ms
              </td>
              <td className="px-4 py-2 text-center">
                <span
                  className={`rounded-full px-2 py-0.5 text-xs ${
                    r.cache_hit ? "bg-teal-500/15 text-teal-300" : "bg-slate-800 text-slate-500"
                  }`}
                >
                  {r.cache_hit ? "hit" : "miss"}
                </span>
              </td>
              <td className="whitespace-nowrap px-4 py-2 text-right font-mono text-xs tabular-nums text-slate-400">
                {r.tokens_in}→{r.tokens_out}
              </td>
              <td className="whitespace-nowrap px-4 py-2 text-right font-mono text-xs tabular-nums text-slate-400">
                ${r.cost_usd.toFixed(5)}
              </td>
              <td className="px-4 py-2 text-center">
                <span
                  className={`rounded-full px-2 py-0.5 text-xs ${
                    r.status === "ok" ? "bg-slate-800 text-slate-400" : "bg-red-500/15 text-red-300"
                  }`}
                >
                  {r.status}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
