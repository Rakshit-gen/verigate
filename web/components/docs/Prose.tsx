export function H1({ children }: { children: React.ReactNode }) {
  return <h1 className="mb-2 text-3xl font-semibold tracking-tight text-[color:var(--text-primary)]">{children}</h1>;
}

export function Lead({ children }: { children: React.ReactNode }) {
  return <p className="mb-8 text-base leading-relaxed text-[color:var(--text-secondary)]">{children}</p>;
}

export function H2({ children, id }: { children: React.ReactNode; id?: string }) {
  return (
    <h2 id={id} className="mb-3 mt-10 scroll-mt-6 text-xl font-semibold tracking-tight text-[color:var(--text-primary)] first:mt-0">
      {children}
    </h2>
  );
}

export function H3({ children }: { children: React.ReactNode }) {
  return <h3 className="mb-2 mt-6 text-base font-semibold text-[color:var(--text-primary)]">{children}</h3>;
}

export function P({ children }: { children: React.ReactNode }) {
  return <p className="mb-4 text-sm leading-relaxed text-[color:var(--text-secondary)]">{children}</p>;
}

export function UL({ children }: { children: React.ReactNode }) {
  return <ul className="mb-4 flex flex-col gap-1.5 text-sm leading-relaxed text-[color:var(--text-secondary)]">{children}</ul>;
}

export function LI({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex gap-2 pl-0.5">
      <span className="mt-2 h-1 w-1 shrink-0 rounded-full bg-[color:var(--text-tertiary)]" />
      <span>{children}</span>
    </li>
  );
}

export function IC({ children }: { children: React.ReactNode }) {
  // inline code
  return (
    <code className="rounded bg-[color:var(--surface-2)] px-1.5 py-0.5 font-mono text-[0.85em] text-[color:var(--accent)]">
      {children}
    </code>
  );
}

export function Callout({ tone = "info", children }: { tone?: "info" | "warning"; children: React.ReactNode }) {
  const styles =
    tone === "warning"
      ? { border: "var(--status-warning-border)", bg: "var(--status-warning-bg)", fg: "var(--status-warning)" }
      : { border: "var(--status-info-border)", bg: "var(--status-info-bg)", fg: "var(--status-info)" };
  return (
    <div
      className="mb-4 rounded-lg border px-4 py-3 text-sm leading-relaxed"
      style={{ borderColor: styles.border, background: styles.bg, color: "var(--text-secondary)" }}
    >
      <span className="font-medium" style={{ color: styles.fg }}>
        {tone === "warning" ? "Note — " : "Tip — "}
      </span>
      {children}
    </div>
  );
}

export function Table({ head, rows }: { head: string[]; rows: React.ReactNode[][] }) {
  return (
    <div className="mb-4 overflow-x-auto rounded-lg border border-[color:var(--border)]">
      <table className="w-full text-left text-sm">
        <thead className="bg-[color:var(--surface-2)] text-[11px] uppercase tracking-wider text-[color:var(--text-tertiary)]">
          <tr>
            {head.map((h) => (
              <th key={h} className="px-3 py-2 font-medium">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-[color:var(--border-soft)]">
          {rows.map((row, i) => (
            <tr key={i} className="bg-[color:var(--surface-1)]">
              {row.map((cell, j) => (
                <td key={j} className="px-3 py-2 align-top text-[color:var(--text-secondary)]">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
