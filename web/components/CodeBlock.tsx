"use client";

import { useState } from "react";

export default function CodeBlock({ code, lang }: { code: string; lang?: string }) {
  const [copied, setCopied] = useState(false);

  function copy() {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1600);
  }

  return (
    <div className="group relative overflow-hidden rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)]">
      <div className="flex items-center justify-between border-b border-[color:var(--border-soft)] px-4 py-2">
        <div className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-[color:var(--status-critical)] opacity-40" />
          <span className="h-2.5 w-2.5 rounded-full bg-[color:var(--status-warning)] opacity-40" />
          <span className="h-2.5 w-2.5 rounded-full bg-[color:var(--status-good)] opacity-40" />
        </div>
        {lang && <span className="text-[10px] uppercase tracking-wider text-[color:var(--text-tertiary)]">{lang}</span>}
        <button
          onClick={copy}
          className="text-[11px] text-[color:var(--text-tertiary)] transition-colors hover:text-[color:var(--text-primary)]"
        >
          {copied ? "Copied ✓" : "Copy"}
        </button>
      </div>
      <pre className="overflow-x-auto px-4 py-3.5 text-[13px] leading-relaxed">
        <code className="font-mono text-[color:var(--text-secondary)]">{code}</code>
      </pre>
    </div>
  );
}
