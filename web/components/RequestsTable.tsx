import type { RequestRecord } from "@/lib/api";

function truncate(s: string, max = 60) {
  return s.length > max ? s.slice(0, max) + "…" : s;
}

function toolCallNames(toolCallsJSON: string): string[] {
  if (!toolCallsJSON) return [];
  try {
    const calls = JSON.parse(toolCallsJSON) as Array<{ function?: { name?: string } }>;
    return calls.map((c) => c.function?.name).filter((n): n is string => Boolean(n));
  } catch {
    return [];
  }
}

function Badge({ children, tone }: { children: React.ReactNode; tone: "info" | "warning" | "critical" | "accent" }) {
  const styles = {
    info: { bg: "var(--status-info-bg)", border: "var(--status-info-border)", fg: "var(--status-info)" },
    warning: { bg: "var(--status-warning-bg)", border: "var(--status-warning-border)", fg: "var(--status-warning)" },
    critical: { bg: "var(--status-critical-bg)", border: "var(--status-critical-border)", fg: "var(--status-critical)" },
    accent: { bg: "var(--accent-soft)", border: "var(--accent-border)", fg: "var(--accent)" },
  }[tone];
  return (
    <span
      className="whitespace-nowrap rounded-full border px-2 py-0.5 text-[11px] font-medium"
      style={{ background: styles.bg, borderColor: styles.border, color: styles.fg }}
    >
      {children}
    </span>
  );
}

function Flags({ r }: { r: RequestRecord }) {
  const names = toolCallNames(r.tool_calls);
  const badges: React.ReactNode[] = [];

  if (r.pii_redacted) {
    badges.push(
      <Badge key="pii" tone="info">
        PII redacted
      </Badge>
    );
  }
  if (r.injection_score >= 0.5) {
    badges.push(
      <Badge key="inj" tone="critical">
        injection {r.injection_score.toFixed(2)}
      </Badge>
    );
  } else if (r.injection_score > 0) {
    badges.push(
      <Badge key="inj" tone="warning">
        injection {r.injection_score.toFixed(2)}
      </Badge>
    );
  }
  for (const name of names) {
    badges.push(
      <Badge key={"tool-" + name} tone="accent">
        <span className="font-mono">⚙ {name}</span>
      </Badge>
    );
  }

  if (badges.length === 0) return <span className="text-xs text-[color:var(--text-tertiary)]">—</span>;
  return <div className="flex flex-wrap gap-1">{badges}</div>;
}

export default function RequestsTable({ requests }: { requests: RequestRecord[] }) {
  if (requests.length === 0) {
    return (
      <div className="flex h-40 items-center justify-center rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] text-sm text-[color:var(--text-secondary)]">
        No requests logged yet. Point a client at{" "}
        <code className="mx-1 rounded bg-[color:var(--surface-2)] px-1.5 py-0.5 font-mono text-xs">
          /v1/chat/completions
        </code>
        to see traffic here.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-[color:var(--border)]">
      <table className="w-full text-left text-sm">
        <thead className="bg-[color:var(--surface-2)] text-[11px] uppercase tracking-wider text-[color:var(--text-tertiary)]">
          <tr>
            <th className="px-4 py-2.5 font-medium">Time</th>
            <th className="px-4 py-2.5 font-medium">Model</th>
            <th className="px-4 py-2.5 font-medium">Prompt</th>
            <th className="px-4 py-2.5 font-medium">Flags</th>
            <th className="px-4 py-2.5 font-medium text-right">Latency</th>
            <th className="px-4 py-2.5 font-medium text-center">Cache</th>
            <th className="px-4 py-2.5 font-medium text-right">Tokens</th>
            <th className="px-4 py-2.5 font-medium text-right">Cost</th>
            <th className="px-4 py-2.5 font-medium text-center">Status</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[color:var(--border-soft)]">
          {requests.map((r) => (
            <tr key={r.id} className="bg-[color:var(--surface-1)] transition-colors hover:bg-[color:var(--surface-3)]">
              <td className="whitespace-nowrap px-4 py-2.5 font-mono text-xs text-[color:var(--text-tertiary)]">
                {new Date(r.created_at).toLocaleTimeString()}
              </td>
              <td className="whitespace-nowrap px-4 py-2.5 font-mono text-xs text-[color:var(--accent)]">{r.model}</td>
              <td className="px-4 py-2.5 text-[color:var(--text-secondary)]">{truncate(r.prompt)}</td>
              <td className="px-4 py-2.5">
                <Flags r={r} />
              </td>
              <td className="whitespace-nowrap px-4 py-2.5 text-right font-mono text-xs tabular-nums text-[color:var(--text-secondary)]">
                {r.latency_ms}ms
              </td>
              <td className="px-4 py-2.5 text-center">
                <span
                  className="rounded-full px-2 py-0.5 text-[11px] font-medium"
                  style={
                    r.cache_hit
                      ? { background: "var(--accent-soft)", color: "var(--accent)" }
                      : { background: "var(--surface-3)", color: "var(--text-tertiary)" }
                  }
                >
                  {r.cache_hit ? "hit" : "miss"}
                </span>
              </td>
              <td className="whitespace-nowrap px-4 py-2.5 text-right font-mono text-xs tabular-nums text-[color:var(--text-secondary)]">
                {r.tokens_in}→{r.tokens_out}
              </td>
              <td className="whitespace-nowrap px-4 py-2.5 text-right font-mono text-xs tabular-nums text-[color:var(--text-secondary)]">
                ${r.cost_usd.toFixed(5)}
              </td>
              <td className="px-4 py-2.5 text-center">
                <span
                  className="rounded-full px-2 py-0.5 text-[11px] font-medium"
                  style={
                    r.status === "ok"
                      ? { background: "var(--surface-3)", color: "var(--text-secondary)" }
                      : { background: "var(--status-critical-bg)", color: "var(--status-critical)" }
                  }
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
