"use client";

import { useState } from "react";
import { runReplay, getAdminKey, setAdminKey, type ReplayResult } from "@/lib/api";
import ReplayResults from "@/components/ReplayResults";
import AppHeader from "@/components/AppHeader";

export default function ReplayPage() {
  const [adminKey, setAdminKeyState] = useState(() => getAdminKey() ?? "");
  const [unlocked, setUnlocked] = useState(() => Boolean(getAdminKey()));
  const [candidateModel, setCandidateModel] = useState("");
  const [limit, setLimit] = useState(3);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [results, setResults] = useState<ReplayResult[] | null>(null);
  const [skipped, setSkipped] = useState<string[]>([]);

  function handleUnlock(e: React.FormEvent) {
    e.preventDefault();
    setAdminKey(adminKey.trim() || null);
    setUnlocked(Boolean(adminKey.trim()));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!candidateModel.trim()) return;
    setRunning(true);
    setError(null);
    setResults(null);
    try {
      const resp = await runReplay(candidateModel.trim(), limit);
      setResults(resp.results);
      setSkipped(resp.skipped);
    } catch (err) {
      setError(err instanceof Error ? err.message : "replay failed");
    } finally {
      setRunning(false);
    }
  }

  return (
    <main className="mx-auto min-h-screen max-w-6xl px-6 py-10">
      <AppHeader
        title="Replay & diff"
        subtitle="Re-run recent real prompts through a candidate model and compare eval scores side by side."
      />

      {!unlocked && (
        <form
          onSubmit={handleUnlock}
          className="mb-8 flex flex-wrap items-end gap-3 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-4"
        >
          <div className="flex flex-1 min-w-64 flex-col gap-1">
            <label className="text-xs text-[color:var(--text-tertiary)]">
              Admin API key required — replay spends real provider credits
            </label>
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
      )}

      {unlocked && (
        <>
          <form
            onSubmit={handleSubmit}
            className="mb-8 flex flex-wrap items-end gap-3 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-4"
          >
            <div className="flex flex-col gap-1">
              <label className="text-xs text-[color:var(--text-tertiary)]">Candidate model</label>
              <input
                value={candidateModel}
                onChange={(e) => setCandidateModel(e.target.value)}
                placeholder="openai/gpt-oss-120b"
                className="w-64 rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 font-mono text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-[color:var(--text-tertiary)]">Recent requests (max 10)</label>
              <input
                type="number"
                min={1}
                max={10}
                value={limit}
                onChange={(e) => setLimit(Number(e.target.value))}
                className="w-24 rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
              />
            </div>
            <button
              type="submit"
              disabled={running || !candidateModel.trim()}
              className="flex items-center gap-2 rounded-lg px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-40"
              style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
            >
              {running && (
                <span className="h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
              )}
              {running ? "Replaying — real judge calls, a few seconds" : "Replay & compare"}
            </button>
            {error && (
              <p className="w-full text-xs" style={{ color: "var(--status-critical)" }}>
                {error}
              </p>
            )}
          </form>

          {skipped.length > 0 && (
            <p className="mb-4 text-xs" style={{ color: "var(--status-warning)" }}>
              {skipped.length} request{skipped.length === 1 ? "" : "s"} skipped: {skipped.join("; ")}
            </p>
          )}

          {results && results.length === 0 && !error && (
            <div className="flex h-32 items-center justify-center rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] text-sm text-[color:var(--text-secondary)]">
              No eligible requests found — need at least one non-cached, successful request logged first.
            </div>
          )}

          {results && results.length > 0 && (
            <div className="animate-fade-in-up">
              <ReplayResults results={results} />
            </div>
          )}
        </>
      )}
    </main>
  );
}
