"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import AppHeader from "@/components/AppHeader";
import { useSession } from "@/lib/session-context";

export default function SignupPage() {
  const router = useRouter();
  const { signUp } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const { apiKey } = await signUp(email.trim(), password);
      setApiKey(apiKey);
    } catch (err) {
      setError(err instanceof Error ? err.message : "sign up failed");
    } finally {
      setSubmitting(false);
    }
  }

  function copyKey() {
    if (!apiKey) return;
    navigator.clipboard.writeText(apiKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <main className="mx-auto min-h-screen max-w-lg px-6 py-10">
      <AppHeader title="Sign up" subtitle="Get your own API key and a dashboard scoped to just your own traffic." />

      {apiKey ? (
        <div
          className="animate-fade-in-up rounded-xl border p-4"
          style={{ borderColor: "var(--status-warning-border)", background: "var(--status-warning-bg)" }}
        >
          <p className="mb-1 text-sm font-medium" style={{ color: "var(--status-warning)" }}>
            Account created — save this key now
          </p>
          <p className="mb-3 text-xs text-[color:var(--text-secondary)]">
            This is the only time your API key is shown. It&apos;s stored as a hash — there is no way to recover it
            later (you can generate a new one from the dashboard if you lose it, but the old one stops working).
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 overflow-x-auto rounded-lg bg-[color:var(--bg)] px-3 py-2 font-mono text-xs text-[color:var(--accent)]">
              {apiKey}
            </code>
            <button
              onClick={copyKey}
              className="shrink-0 rounded-lg bg-[color:var(--surface-3)] px-3 py-2 text-xs text-[color:var(--text-secondary)] transition-colors hover:text-[color:var(--text-primary)]"
            >
              {copied ? "Copied ✓" : "Copy"}
            </button>
          </div>
          <button
            onClick={() => router.push("/dashboard")}
            className="mt-4 rounded-lg px-4 py-1.5 text-sm font-medium transition-colors"
            style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
          >
            Go to dashboard
          </button>
        </div>
      ) : (
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-3 rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-4"
        >
          <div className="flex flex-col gap-1">
            <label className="text-xs text-[color:var(--text-tertiary)]">Email</label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-[color:var(--text-tertiary)]">Password (min 8 characters)</label>
            <input
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
            />
          </div>
          <button
            type="submit"
            disabled={submitting || !email.trim() || password.length < 8}
            className="rounded-lg px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-40"
            style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
          >
            {submitting ? "Creating account…" : "Sign up"}
          </button>
          {error && (
            <p className="text-xs" style={{ color: "var(--status-critical)" }}>
              {error}
            </p>
          )}
          <p className="text-xs text-[color:var(--text-tertiary)]">
            Already have an account?{" "}
            <Link href="/login" className="text-[color:var(--accent)] hover:underline">
              Log in
            </Link>
          </p>
        </form>
      )}
    </main>
  );
}
