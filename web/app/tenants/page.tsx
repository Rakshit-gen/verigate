"use client";

import Link from "next/link";
import useSWR from "swr";
import { fetchTenants } from "@/lib/api";
import CreateTenantForm from "@/components/CreateTenantForm";
import TenantsTable from "@/components/TenantsTable";

export default function TenantsPage() {
  const { data: tenants, mutate } = useSWR("tenants", fetchTenants);

  return (
    <main className="mx-auto min-h-screen max-w-5xl px-6 py-10">
      <header className="mb-8 flex items-baseline justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-slate-100">
            Tenants<span className="text-teal-400">.</span>
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Per-tenant API keys and rate limits — each key is independent, none can starve another.
          </p>
        </div>
        <Link href="/" className="text-xs text-slate-500 hover:text-teal-300">
          ← Dashboard
        </Link>
      </header>

      <section className="mb-8">
        <CreateTenantForm onCreated={() => mutate()} />
      </section>

      <section>
        <h2 className="mb-3 text-xs font-medium uppercase tracking-wide text-slate-500">
          Existing tenants ({tenants?.length ?? 0})
        </h2>
        <TenantsTable tenants={tenants ?? []} />
      </section>
    </main>
  );
}
