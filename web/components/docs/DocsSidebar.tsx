"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const sections = [
  {
    heading: "Getting started",
    links: [
      { href: "/docs", label: "Overview" },
      { href: "/docs/quickstart", label: "Quickstart" },
    ],
  },
  {
    heading: "Concepts",
    links: [
      { href: "/docs/architecture", label: "Architecture" },
      { href: "/docs/security", label: "Guardrails & multi-tenancy" },
    ],
  },
  {
    heading: "Reference",
    links: [{ href: "/docs/api", label: "API reference" }],
  },
];

export default function DocsSidebar() {
  const pathname = usePathname();

  return (
    <nav className="flex flex-col gap-6">
      {sections.map((section) => (
        <div key={section.heading}>
          <div className="mb-2 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
            {section.heading}
          </div>
          <div className="flex flex-col gap-0.5">
            {section.links.map((l) => {
              const active = pathname === l.href;
              return (
                <Link
                  key={l.href}
                  href={l.href}
                  className={`rounded-lg px-2.5 py-1.5 text-sm transition-colors ${
                    active
                      ? "bg-[color:var(--accent-soft)] text-[color:var(--accent)]"
                      : "text-[color:var(--text-secondary)] hover:bg-[color:var(--surface-2)] hover:text-[color:var(--text-primary)]"
                  }`}
                >
                  {l.label}
                </Link>
              );
            })}
          </div>
        </div>
      ))}
    </nav>
  );
}
