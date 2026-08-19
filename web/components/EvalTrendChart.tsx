"use client";

import {
  Area,
  AreaChart,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
  Legend,
} from "recharts";
import type { EvalRecord } from "@/lib/api";

const REGRESSION_THRESHOLD = 0.6; // mirrors REGRESSION_MIN_SCORE default in the backend

// Mirrors the CSS custom properties in globals.css (--series-groundedness /
// --series-format) — hardcoded here because Recharts renders these as raw
// SVG fill/stroke attributes, which can't resolve CSS custom properties.
// Validated with the dataviz skill's palette checker against this app's
// dark surface: lightness band + CVD separation + contrast all pass.
const SERIES = {
  groundedness: "#0d9488",
  format_compliance: "#6366f1",
};

function CustomTooltip({ active, payload, label }: { active?: boolean; payload?: Array<{ name: string; value: number; color: string }>; label?: string }) {
  if (!active || !payload || payload.length === 0) return null;
  return (
    <div className="rounded-lg border border-[color:var(--border)] bg-[color:var(--surface-2)] px-3 py-2 text-xs shadow-xl">
      <div className="mb-1 font-mono text-[color:var(--text-tertiary)]">{label}</div>
      {payload.map((p) => (
        <div key={p.name} className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: p.color }} />
          <span className="text-[color:var(--text-secondary)]">{p.name}</span>
          <span className="ml-auto font-mono tabular-nums text-[color:var(--text-primary)]">
            {(p.value * 100).toFixed(0)}%
          </span>
        </div>
      ))}
    </div>
  );
}

export default function EvalTrendChart({ evals }: { evals: EvalRecord[] }) {
  // evals arrive newest-first from the API; the chart reads left-to-right
  // as oldest-to-newest, and rubrics are split into separate series so a
  // format-compliance dip doesn't visually cancel out a groundedness dip.
  const chronological = [...evals].reverse();

  // Each request gets one eval row per rubric, so group by request_id to
  // land both scores on the same point instead of alternating x-positions.
  const byRequest = new Map<string, Record<string, number | string>>();
  for (const e of chronological) {
    const point = byRequest.get(e.request_id) ?? {
      request_id: e.request_id,
      time: new Date(e.created_at).toLocaleTimeString(),
    };
    point[e.rubric] = e.score;
    byRequest.set(e.request_id, point);
  }
  const data = Array.from(byRequest.values());

  if (data.length === 0) {
    return (
      <div className="flex h-64 items-center justify-center rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] text-sm text-[color:var(--text-secondary)]">
        Eval score trend will appear here once the judge has scored a few samples.
      </div>
    );
  }

  return (
    <div className="h-64 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-4">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 16, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="fillGroundedness" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={SERIES.groundedness} stopOpacity={0.28} />
              <stop offset="100%" stopColor={SERIES.groundedness} stopOpacity={0} />
            </linearGradient>
            <linearGradient id="fillFormat" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={SERIES.format_compliance} stopOpacity={0.24} />
              <stop offset="100%" stopColor={SERIES.format_compliance} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--border-soft)" vertical={false} />
          <XAxis dataKey="time" tick={{ fill: "var(--text-tertiary)", fontSize: 11 }} axisLine={{ stroke: "var(--border)" }} tickLine={false} />
          <YAxis
            domain={[0, 1]}
            tickFormatter={(v) => `${Math.round(v * 100)}%`}
            tick={{ fill: "var(--text-tertiary)", fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            width={44}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend
            wrapperStyle={{ fontSize: 12, color: "var(--text-secondary)" }}
            formatter={(value) => <span style={{ color: "var(--text-secondary)" }}>{value}</span>}
          />
          <ReferenceLine
            y={REGRESSION_THRESHOLD}
            stroke="var(--status-critical)"
            strokeDasharray="4 4"
            strokeOpacity={0.6}
            label={{ value: "threshold", fill: "var(--status-critical)", fontSize: 10, position: "insideTopLeft" }}
          />
          <Area
            type="monotone"
            dataKey="groundedness"
            stroke={SERIES.groundedness}
            strokeWidth={2}
            fill="url(#fillGroundedness)"
            dot={{ r: 2.5, fill: SERIES.groundedness, strokeWidth: 0 }}
            activeDot={{ r: 4.5, strokeWidth: 0 }}
            connectNulls
          />
          <Area
            type="monotone"
            dataKey="format_compliance"
            stroke={SERIES.format_compliance}
            strokeWidth={2}
            fill="url(#fillFormat)"
            dot={{ r: 2.5, fill: SERIES.format_compliance, strokeWidth: 0 }}
            activeDot={{ r: 4.5, strokeWidth: 0 }}
            connectNulls
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
