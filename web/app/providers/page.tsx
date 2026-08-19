"use client";

import useSWR from "swr";
import { fetchProviderStatus } from "@/lib/api";
import ProvidersStatus from "@/components/ProvidersStatus";
import AppHeader from "@/components/AppHeader";

export default function ProvidersPage() {
  const { data } = useSWR("providers", fetchProviderStatus, { refreshInterval: 3000 });

  return (
    <main className="mx-auto min-h-screen max-w-6xl px-6 py-10">
      <AppHeader
        title="Providers"
        subtitle="Live circuit-breaker state and measured latency for every configured provider."
      />
      <div className="animate-fade-in-up">
        <ProvidersStatus data={data} />
      </div>
    </main>
  );
}
