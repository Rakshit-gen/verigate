"use client";

import useSWR from "swr";
import { fetchRequests, fetchEvalSummary, fetchRecentEvals } from "@/lib/api";
import RegressionBanner from "@/components/RegressionBanner";
import EvalTrendChart from "@/components/EvalTrendChart";
import RequestsTable from "@/components/RequestsTable";

const POLL_MS = 4000;

export default function DashboardPage() {
  const { data: requests } = useSWR("requests", fetchRequests, { refreshInterval: POLL_MS });
  const { data: summary } = useSWR("eval-summary", fetchEvalSummary, { refreshInterval: POLL_MS });
  const { data: evals } = useSWR("recent-evals", fetchRecentEvals, { refreshInterval: POLL_MS });

  return (
    <main className="mx-auto min-h-screen max-w-5xl px-6 py-10">
      <header className="mb-8 flex items-baseline justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-slate-100">
            verigate<span className="text-teal-400">.</span>
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            An LLM gateway that grades its own traffic as it flows through.
          </p>
        </div>
        <span className="font-mono text-xs text-slate-600">localhost:8080</span>
      </header>

      <section className="mb-6">
        <RegressionBanner summary={summary} />
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-500">
          Eval score trend (last {evals?.length ?? 0} samples)
        </h2>
        <EvalTrendChart evals={evals ?? []} />
      </section>

      <section>
        <h2 className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-500">
          Recent requests
        </h2>
        <RequestsTable requests={requests ?? []} />
      </section>
    </main>
  );
}
