"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import AppHeader from "@/components/AppHeader";
import { useSession } from "@/lib/session-context";

export default function LoginPage() {
  const router = useRouter();
  const { login } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await login(email.trim(), password);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="mx-auto min-h-screen max-w-lg px-6 py-10">
      <AppHeader title="Log in" subtitle="See your own dashboard — only your requests, evals, and quality trend." />

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
          <label className="text-xs text-[color:var(--text-tertiary)]">Password</label>
          <input
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            className="rounded-lg border border-[color:var(--border)] bg-[color:var(--bg)] px-2.5 py-1.5 text-sm text-[color:var(--text-primary)] outline-none transition-colors focus:border-[color:var(--accent)]"
          />
        </div>
        <button
          type="submit"
          disabled={submitting || !email.trim() || !password}
          className="rounded-lg px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-40"
          style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
        >
          {submitting ? "Logging in…" : "Log in"}
        </button>
        {error && (
          <p className="text-xs" style={{ color: "var(--status-critical)" }}>
            {error}
          </p>
        )}
        <p className="text-xs text-[color:var(--text-tertiary)]">
          Don&apos;t have an account?{" "}
          <Link href="/signup" className="text-[color:var(--accent)] hover:underline">
            Sign up
          </Link>
        </p>
      </form>
    </main>
  );
}
