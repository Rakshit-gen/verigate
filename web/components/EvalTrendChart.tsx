"use client";

import {
  Line,
  LineChart,
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
      <div className="flex h-64 items-center justify-center rounded-lg border border-slate-800 bg-slate-900/50 text-sm text-slate-500">
        Eval score trend will appear here once the judge has scored a few samples.
      </div>
    );
  }

  return (
    <div className="h-64 rounded-lg border border-slate-800 bg-slate-900/50 p-4">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 16, left: -16, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
          <XAxis dataKey="time" tick={{ fill: "#64748b", fontSize: 11 }} />
          <YAxis domain={[0, 1]} tick={{ fill: "#64748b", fontSize: 11 }} />
          <Tooltip
            contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8, fontSize: 12 }}
          />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <ReferenceLine y={REGRESSION_THRESHOLD} stroke="#f87171" strokeDasharray="4 4" label={{ value: "threshold", fill: "#f87171", fontSize: 10, position: "insideTopLeft" }} />
          <Line type="monotone" dataKey="groundedness" stroke="#2dd4bf" strokeWidth={2} dot={{ r: 3 }} connectNulls />
          <Line type="monotone" dataKey="format_compliance" stroke="#818cf8" strokeWidth={2} dot={{ r: 3 }} connectNulls />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
