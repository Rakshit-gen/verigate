"use client";

import useSWR from "swr";
import { fetchRequests, fetchEvalSummary, fetchRecentEvals } from "@/lib/api";
import RegressionBanner from "@/components/RegressionBanner";
import EvalTrendChart from "@/components/EvalTrendChart";
import RequestsTable from "@/components/RequestsTable";
import StatTiles from "@/components/StatTiles";
import AppHeader from "@/components/AppHeader";

const POLL_MS = 4000;

export default function DashboardPage() {
  const { data: requests } = useSWR("requests", fetchRequests, { refreshInterval: POLL_MS });
  const { data: summary } = useSWR("eval-summary", fetchEvalSummary, { refreshInterval: POLL_MS });
  const { data: evals } = useSWR("recent-evals", fetchRecentEvals, { refreshInterval: POLL_MS });

  return (
    <main className="mx-auto min-h-screen max-w-6xl px-6 py-10">
      <AppHeader title="Verigate" subtitle="An LLM gateway that grades its own traffic as it flows through." />

      <section className="animate-fade-in-up mb-6">
        <StatTiles requests={requests ?? []} />
      </section>

      <section className="animate-fade-in-up mb-6" style={{ animationDelay: "60ms" }}>
        <RegressionBanner summary={summary} />
      </section>

      <section className="animate-fade-in-up mb-8" style={{ animationDelay: "120ms" }}>
        <h2 className="mb-3 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
          Eval score trend — last {evals?.length ?? 0} samples
        </h2>
        <EvalTrendChart evals={evals ?? []} />
      </section>

      <section className="animate-fade-in-up" style={{ animationDelay: "180ms" }}>
        <h2 className="mb-3 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
          Recent requests
        </h2>
        <RequestsTable requests={requests ?? []} />
      </section>
    </main>
  );
}
