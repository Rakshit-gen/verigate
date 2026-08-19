export default function StatTile({
  label,
  value,
  unit,
  accent = "neutral",
  sub,
}: {
  label: string;
  value: string;
  unit?: string;
  accent?: "neutral" | "good" | "warning" | "critical";
  sub?: string;
}) {
  const valueColor =
    accent === "good"
      ? "text-[color:var(--status-good)]"
      : accent === "warning"
        ? "text-[color:var(--status-warning)]"
        : accent === "critical"
          ? "text-[color:var(--status-critical)]"
          : "text-[color:var(--text-primary)]";

  return (
    <div className="rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] px-5 py-4">
      <div className="text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
        {label}
      </div>
      <div className="mt-1.5 flex items-baseline gap-1.5">
        <span className={`font-mono text-2xl font-semibold tabular-nums ${valueColor}`}>{value}</span>
        {unit && <span className="text-xs text-[color:var(--text-secondary)]">{unit}</span>}
      </div>
      {sub && <div className="mt-1 text-xs text-[color:var(--text-tertiary)]">{sub}</div>}
    </div>
  );
}
