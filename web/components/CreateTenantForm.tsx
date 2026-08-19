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
      <div
        className="animate-fade-in-up rounded-xl border p-4"
        style={{ borderColor: "var(--status-warning-border)", background: "var(--status-warning-bg)" }}
      >
        <p className="mb-1 text-sm font-medium" style={{ color: "var(--status-warning)" }}>
          Tenant &ldquo;{revealed.name}&rdquo; created — save this key now
        </p>
        <p className="mb-3 text-xs text-[color:var(--text-secondary)]">
          This is the only time the key is shown. It&apos;s stored as a hash — there is no way to recover it later.
        </p>
        <div className="flex items-center gap-2">
          <code className="flex-1 overflow-x-auto rounded-lg bg-[color:var(--bg)] px-3 py-2 font-mono text-xs text-[color:var(--accent)]">
            {revealed.api_key}
          </code>
          <button
            onClick={copyKey}
            className="shrink-0 rounded-lg bg-[color:var(--surface-3)] px-3 py-2 text-xs text-[color:var(--text-secondary)] transition-colors hover:text-[color:var(--text-primary)]"
          >
            {copied ? "Copied ✓" : "Copy"}
          </button>
        </div>
        <button
          onClick={() => setRevealed(null)}
          className="mt-3 text-xs text-[color:var(--text-tertiary)] underline decoration-dotted underline-offset-2 hover:text-[color:var(--text-secondary)]"
        >
          Done — create another tenant
        </button>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-wrap items-end gap-3 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-4"
    >
      <div className="flex flex-col gap-1">
        <label className="text-xs text-[color:var(--text-tertiary)]">Name</label>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="acme-corp"
          className="rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className="text-xs text-[color:var(--text-tertiary)]">Requests / min</label>
        <input
          type="number"
          min={1}
          value={rpm}
          onChange={(e) => setRpm(Number(e.target.value))}
          className="w-28 rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
        />
      </div>
      <button
        type="submit"
        disabled={submitting || !name.trim()}
        className="rounded-lg px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-40"
        style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
      >
        {submitting ? "Creating…" : "Create tenant"}
      </button>
      {error && (
        <p className="w-full text-xs" style={{ color: "var(--status-critical)" }}>
          {error}
        </p>
      )}
    </form>
  );
}
