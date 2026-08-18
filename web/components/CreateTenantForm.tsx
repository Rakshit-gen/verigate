"use client";

import { useState } from "react";
import { createTenant, type CreatedTenant } from "@/lib/api";

export default function CreateTenantForm({ onCreated }: { onCreated: () => void }) {
  const [name, setName] = useState("");
  const [rpm, setRpm] = useState(60);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [revealed, setRevealed] = useState<CreatedTenant | null>(null);
  const [copied, setCopied] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      const tenant = await createTenant(name.trim(), rpm);
      setRevealed(tenant);
      setName("");
      setRpm(60);
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to create tenant");
    } finally {
      setSubmitting(false);
    }
  }

  function copyKey() {
    if (!revealed) return;
    navigator.clipboard.writeText(revealed.api_key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  if (revealed) {
    return (
      <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-4">
        <p className="mb-1 text-sm font-medium text-amber-300">
          Tenant &ldquo;{revealed.name}&rdquo; created — save this key now
        </p>
        <p className="mb-3 text-xs text-amber-400/80">
          This is the only time the key is shown. It&apos;s stored as a hash — there is no way to recover it later.
        </p>
        <div className="flex items-center gap-2">
          <code className="flex-1 overflow-x-auto rounded bg-slate-950 px-3 py-2 font-mono text-xs text-teal-300">
            {revealed.api_key}
          </code>
          <button
            onClick={copyKey}
            className="shrink-0 rounded bg-slate-800 px-3 py-2 text-xs text-slate-300 hover:bg-slate-700"
          >
            {copied ? "Copied" : "Copy"}
          </button>
        </div>
        <button
          onClick={() => setRevealed(null)}
          className="mt-3 text-xs text-slate-500 underline hover:text-slate-300"
        >
          Done — create another tenant
        </button>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-3 rounded-lg border border-slate-800 bg-slate-900/50 p-4">
      <div className="flex flex-col gap-1">
        <label className="text-xs text-slate-500">Name</label>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="acme-corp"
          className="rounded border border-slate-700 bg-slate-950 px-2 py-1.5 text-sm text-slate-200 outline-none focus:border-teal-500"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className="text-xs text-slate-500">Requests / min</label>
        <input
          type="number"
          min={1}
          value={rpm}
          onChange={(e) => setRpm(Number(e.target.value))}
          className="w-28 rounded border border-slate-700 bg-slate-950 px-2 py-1.5 text-sm text-slate-200 outline-none focus:border-teal-500"
        />
      </div>
      <button
        type="submit"
        disabled={submitting || !name.trim()}
        className="rounded bg-teal-500/15 px-4 py-1.5 text-sm font-medium text-teal-300 hover:bg-teal-500/25 disabled:opacity-40"
      >
        {submitting ? "Creating…" : "Create tenant"}
      </button>
      {error && <p className="w-full text-xs text-red-400">{error}</p>}
    </form>
  );
}
