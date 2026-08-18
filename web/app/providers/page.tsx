"use client";

import useSWR from "swr";
import { fetchProviderStatus } from "@/lib/api";
import ProvidersStatus from "@/components/ProvidersStatus";
import Nav from "@/components/Nav";

export default function ProvidersPage() {
  const { data } = useSWR("providers", fetchProviderStatus, { refreshInterval: 3000 });

  return (
    <main className="mx-auto min-h-screen max-w-5xl px-6 py-10">
      <header className="mb-8 flex items-baseline justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-slate-100">
            Providers<span className="text-teal-400">.</span>
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Live circuit-breaker state and measured latency for every configured provider.
          </p>
        </div>
        <Nav />
      </header>

      <ProvidersStatus data={data} />
    </main>
  );
}
