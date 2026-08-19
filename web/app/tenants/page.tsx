"use client";

import { useState } from "react";
import useSWR from "swr";
import { fetchTenantsAsAdmin, getAdminKey, setAdminKey } from "@/lib/api";
import CreateTenantForm from "@/components/CreateTenantForm";
import TenantsTable from "@/components/TenantsTable";
import AppHeader from "@/components/AppHeader";

// This page is the operator's manual tenant-management tool — distinct
// from self-serve /signup, which is how everyone else gets a tenant.
// Manual creation still exists for ad hoc/demo tenants and stays gated
// behind the admin key (POST /api/tenants requires it server-side
// regardless of what this page sends), so this is a local convenience
// prompt, not itself a security boundary.
export default function TenantsPage() {
  const [adminKey, setAdminKeyState] = useState(() => getAdminKey() ?? "");
  const [unlocked, setUnlocked] = useState(() => Boolean(getAdminKey()));
  const { data: tenants, mutate } = useSWR(unlocked ? "tenants-admin" : null, fetchTenantsAsAdmin);

  function handleUnlock(e: React.FormEvent) {
    e.preventDefault();
    setAdminKey(adminKey.trim() || null);
    setUnlocked(Boolean(adminKey.trim()));
  }

  return (
    <main className="mx-auto min-h-screen max-w-6xl px-6 py-10">
      <AppHeader
        title="Tenants"
        subtitle="Operator tool for manually creating tenants — most people should sign up at /signup instead."
      />

      {!unlocked ? (
        <form
          onSubmit={handleUnlock}
          className="animate-fade-in-up flex flex-wrap items-end gap-3 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-4"
        >
          <div className="flex flex-1 min-w-64 flex-col gap-1">
            <label className="text-xs text-[color:var(--text-tertiary)]">Admin API key (VERIGATE_API_KEY)</label>
            <input
              type="password"
              value={adminKey}
              onChange={(e) => setAdminKeyState(e.target.value)}
              placeholder="vg_… or your configured admin key"
              className="rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
            />
          </div>
          <button
            type="submit"
            className="rounded-lg px-4 py-1.5 text-sm font-medium transition-colors"
            style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
          >
            Unlock
          </button>
        </form>
      ) : (
        <>
          <section className="animate-fade-in-up mb-8">
            <CreateTenantForm onCreated={() => mutate()} />
            <button
              onClick={() => {
                setAdminKey(null);
                setAdminKeyState("");
                setUnlocked(false);
              }}
              className="mt-3 text-xs text-[color:var(--text-tertiary)] underline decoration-dotted underline-offset-2 hover:text-[color:var(--text-secondary)]"
            >
              Forget admin key
            </button>
          </section>

          <section className="animate-fade-in-up" style={{ animationDelay: "80ms" }}>
            <h2 className="mb-3 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
              Existing tenants ({tenants?.length ?? 0})
            </h2>
            <TenantsTable tenants={tenants ?? []} />
          </section>
        </>
      )}
    </main>
  );
}
