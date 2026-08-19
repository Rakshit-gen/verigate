"use client";

import useSWR from "swr";
import { fetchTenants } from "@/lib/api";
import CreateTenantForm from "@/components/CreateTenantForm";
import TenantsTable from "@/components/TenantsTable";
import AppHeader from "@/components/AppHeader";

export default function TenantsPage() {
  const { data: tenants, mutate } = useSWR("tenants", fetchTenants);

  return (
    <main className="mx-auto min-h-screen max-w-6xl px-6 py-10">
      <AppHeader
        title="Tenants"
        subtitle="Per-tenant API keys and rate limits — each key is independent, none can starve another."
      />

      <section className="animate-fade-in-up mb-8">
        <CreateTenantForm onCreated={() => mutate()} />
      </section>

      <section className="animate-fade-in-up" style={{ animationDelay: "80ms" }}>
        <h2 className="mb-3 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
          Existing tenants ({tenants?.length ?? 0})
        </h2>
        <TenantsTable tenants={tenants ?? []} />
      </section>
    </main>
  );
}
