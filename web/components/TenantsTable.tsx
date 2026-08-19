import type { Tenant } from "@/lib/api";

export default function TenantsTable({ tenants }: { tenants: Tenant[] }) {
  if (tenants.length === 0) {
    return (
      <div className="flex h-32 items-center justify-center rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] text-sm text-[color:var(--text-secondary)]">
        No tenants yet — create one above to get a per-tenant API key.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-[color:var(--border)]">
      <table className="w-full text-left text-sm">
        <thead className="bg-[color:var(--surface-2)] text-[11px] uppercase tracking-wider text-[color:var(--text-tertiary)]">
          <tr>
            <th className="px-4 py-2.5 font-medium">Name</th>
            <th className="px-4 py-2.5 font-medium text-right">Rate limit</th>
            <th className="px-4 py-2.5 font-medium">Created</th>
            <th className="px-4 py-2.5 font-medium">Tenant ID</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[color:var(--border-soft)]">
          {tenants.map((t) => (
            <tr key={t.id} className="bg-[color:var(--surface-1)] transition-colors hover:bg-[color:var(--surface-3)]">
              <td className="px-4 py-2.5 font-medium text-[color:var(--text-primary)]">{t.name}</td>
              <td className="whitespace-nowrap px-4 py-2.5 text-right font-mono text-xs tabular-nums text-[color:var(--text-secondary)]">
                {t.rate_limit_rpm} req/min
              </td>
              <td className="whitespace-nowrap px-4 py-2.5 font-mono text-xs text-[color:var(--text-tertiary)]">
                {new Date(t.created_at).toLocaleString()}
              </td>
              <td className="px-4 py-2.5 font-mono text-xs text-[color:var(--text-tertiary)]">{t.id}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
