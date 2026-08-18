"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  { href: "/", label: "Dashboard" },
  { href: "/tenants", label: "Tenants" },
  { href: "/replay", label: "Replay" },
  { href: "/providers", label: "Providers" },
];

export default function Nav() {
  const pathname = usePathname();

  return (
    <nav className="flex items-center gap-4">
      {links.map((l) => {
        const active = pathname === l.href;
        return (
          <Link
            key={l.href}
            href={l.href}
            className={`text-xs ${active ? "text-teal-300" : "text-slate-500 hover:text-teal-300"}`}
          >
            {l.label}
          </Link>
        );
      })}
    </nav>
  );
}
