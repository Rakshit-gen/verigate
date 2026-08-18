import type { Tenant } from "@/lib/api";

export default function TenantsTable({ tenants }: { tenants: Tenant[] }) {
  if (tenants.length === 0) {
    return (
      <div className="flex h-32 items-center justify-center rounded-lg border border-slate-800 bg-slate-900/50 text-sm text-slate-500">
        No tenants yet — create one above to get a per-tenant API key.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-800">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-900 text-xs uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-2 font-medium">Name</th>
            <th className="px-4 py-2 font-medium text-right">Rate limit</th>
            <th className="px-4 py-2 font-medium">Created</th>
            <th className="px-4 py-2 font-medium">Tenant ID</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {tenants.map((t) => (
            <tr key={t.id} className="hover:bg-slate-900/60">
              <td className="px-4 py-2 text-slate-200">{t.name}</td>
              <td className="whitespace-nowrap px-4 py-2 text-right font-mono text-xs tabular-nums text-slate-400">
                {t.rate_limit_rpm} req/min
              </td>
              <td className="whitespace-nowrap px-4 py-2 font-mono text-xs text-slate-500">
                {new Date(t.created_at).toLocaleString()}
              </td>
              <td className="px-4 py-2 font-mono text-xs text-slate-600">{t.id}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
