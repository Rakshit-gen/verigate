"use client";

import Link from "next/link";
import { motion } from "motion/react";
import Logo from "@/components/Logo";

export default function LandingNav() {
  return (
    <nav className="mx-auto flex max-w-6xl items-center justify-between px-6 py-6">
      <Link href="/" className="flex items-center gap-2.5">
        <Logo size={20} />
        <span className="text-sm font-semibold tracking-tight text-[color:var(--text-primary)]">Verigate</span>
      </Link>
      <div className="flex items-center gap-6">
        <Link href="/docs" className="text-sm text-[color:var(--text-secondary)] transition-colors hover:text-[color:var(--text-primary)]">
          Docs
        </Link>
        <Link href="/signup" className="text-sm text-[color:var(--text-secondary)] transition-colors hover:text-[color:var(--text-primary)]">
          Sign up
        </Link>
        <motion.div whileHover={{ scale: 1.04 }} whileTap={{ scale: 0.97 }}>
          <Link
            href="/dashboard"
            className="rounded-full px-4 py-1.5 text-sm font-medium transition-colors"
            style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
          >
            Live dashboard →
          </Link>
        </motion.div>
      </div>
    </nav>
  );
}
