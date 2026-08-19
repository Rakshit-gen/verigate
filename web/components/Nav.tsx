"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useSession } from "@/lib/session-context";

const links = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/tenants", label: "Tenants" },
  { href: "/replay", label: "Replay" },
  { href: "/providers", label: "Providers" },
  { href: "/docs", label: "Docs" },
];

export default function Nav() {
  const pathname = usePathname();
  const router = useRouter();
  const { tenant, loading, logout } = useSession();

  return (
    <div className="flex items-center gap-3">
      <nav className="flex items-center gap-1 rounded-full border border-[color:var(--border)] bg-[color:var(--surface-1)] p-1">
        {links.map((l) => {
          const active = pathname === l.href || pathname.startsWith(l.href + "/");
          return (
            <Link
              key={l.href}
              href={l.href}
              className={`rounded-full px-3 py-1.5 text-xs font-medium transition-colors ${
                active
                  ? "bg-[color:var(--accent-soft)] text-[color:var(--accent)]"
                  : "text-[color:var(--text-secondary)] hover:text-[color:var(--text-primary)]"
              }`}
            >
              {l.label}
            </Link>
          );
        })}
      </nav>

      {!loading &&
        (tenant ? (
          <button
            onClick={() => {
              logout();
              router.push("/");
            }}
            title={tenant.name}
            className="rounded-full px-3 py-1.5 text-xs font-medium text-[color:var(--text-secondary)] transition-colors hover:text-[color:var(--text-primary)]"
          >
            Log out
          </button>
        ) : (
          <div className="flex items-center gap-1">
            <Link
              href="/login"
              className="rounded-full px-3 py-1.5 text-xs font-medium text-[color:var(--text-secondary)] transition-colors hover:text-[color:var(--text-primary)]"
            >
              Log in
            </Link>
            <Link
              href="/signup"
              className="rounded-full px-3 py-1.5 text-xs font-medium transition-colors"
              style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
            >
              Sign up
            </Link>
          </div>
        ))}
    </div>
  );
}
