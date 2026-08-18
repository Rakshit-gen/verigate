"use client";

import { useState } from "react";
import { runReplay, type ReplayResult } from "@/lib/api";
import ReplayResults from "@/components/ReplayResults";
import Nav from "@/components/Nav";

export default function ReplayPage() {
  const [candidateModel, setCandidateModel] = useState("");
  const [limit, setLimit] = useState(3);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [results, setResults] = useState<ReplayResult[] | null>(null);
  const [skipped, setSkipped] = useState<string[]>([]);

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
    <main className="mx-auto min-h-screen max-w-5xl px-6 py-10">
      <header className="mb-8 flex items-baseline justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-slate-100">
            Replay &amp; diff<span className="text-teal-400">.</span>
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Re-run recent real prompts through a candidate model and compare eval scores side by side.
          </p>
        </div>
        <Nav />
      </header>

      <form onSubmit={handleSubmit} className="mb-8 flex flex-wrap items-end gap-3 rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex flex-col gap-1">
          <label className="text-xs text-slate-500">Candidate model</label>
          <input
            value={candidateModel}
            onChange={(e) => setCandidateModel(e.target.value)}
            placeholder="openai/gpt-oss-120b"
            className="w-64 rounded border border-slate-700 bg-slate-950 px-2 py-1.5 font-mono text-sm text-slate-200 outline-none focus:border-teal-500"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs text-slate-500">Recent requests (max 10)</label>
          <input
            type="number"
            min={1}
            max={10}
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="w-24 rounded border border-slate-700 bg-slate-950 px-2 py-1.5 text-sm text-slate-200 outline-none focus:border-teal-500"
          />
        </div>
        <button
          type="submit"
          disabled={running || !candidateModel.trim()}
          className="rounded bg-teal-500/15 px-4 py-1.5 text-sm font-medium text-teal-300 hover:bg-teal-500/25 disabled:opacity-40"
        >
          {running ? "Replaying… (real judge calls, a few seconds)" : "Replay & compare"}
        </button>
        {error && <p className="w-full text-xs text-red-400">{error}</p>}
      </form>

      {skipped.length > 0 && (
        <p className="mb-4 text-xs text-amber-400">
          {skipped.length} request{skipped.length === 1 ? "" : "s"} skipped: {skipped.join("; ")}
        </p>
      )}

      {results && results.length === 0 && !error && (
        <div className="flex h-32 items-center justify-center rounded-lg border border-slate-800 bg-slate-900/50 text-sm text-slate-500">
          No eligible requests found — need at least one non-cached, successful request logged first.
        </div>
      )}

      {results && results.length > 0 && <ReplayResults results={results} />}
    </main>
  );
}
