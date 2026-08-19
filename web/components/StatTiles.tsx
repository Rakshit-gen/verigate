import type { RequestRecord } from "@/lib/api";
import StatTile from "./StatTile";

// Derived entirely client-side from data the dashboard already fetches —
// no new endpoint needed for "at a glance" numbers.
export default function StatTiles({ requests }: { requests: RequestRecord[] }) {
  const n = requests.length;
  const cacheHits = requests.filter((r) => r.cache_hit).length;
  const cacheHitRate = n > 0 ? (cacheHits / n) * 100 : 0;
  const avgLatency = n > 0 ? requests.reduce((sum, r) => sum + r.latency_ms, 0) / n : 0;
  const totalTokens = requests.reduce((sum, r) => sum + r.tokens_in + r.tokens_out, 0);
  const errorCount = requests.filter((r) => r.status === "error").length;

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <StatTile label="Requests" value={String(n)} sub={n === 50 ? "most recent 50" : "logged so far"} />
      <StatTile
        label="Cache hit rate"
        value={cacheHitRate.toFixed(0)}
        unit="%"
        accent={cacheHitRate >= 30 ? "good" : "neutral"}
        sub={`${cacheHits} of ${n} served from cache`}
      />
      <StatTile
        label="Avg latency"
        value={n > 0 ? avgLatency.toFixed(0) : "—"}
        unit="ms"
        sub="across recent traffic"
      />
      <StatTile
        label="Errors"
        value={String(errorCount)}
        accent={errorCount > 0 ? "critical" : "good"}
        sub={totalTokens > 0 ? `${(totalTokens / 1000).toFixed(1)}k tokens processed` : "no traffic yet"}
      />
    </div>
  );
}
