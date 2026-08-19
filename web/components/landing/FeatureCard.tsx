"use client";

import { motion } from "motion/react";

export default function FeatureCard({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <motion.div
      className="rounded-xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-5"
      whileHover={{ y: -4, borderColor: "var(--accent-border)" }}
      transition={{ type: "spring", stiffness: 300, damping: 24 }}
    >
      <motion.div
        className="mb-3 flex h-9 w-9 items-center justify-center rounded-lg"
        style={{ background: "var(--accent-soft)", color: "var(--accent)" }}
        whileHover={{ scale: 1.1, rotate: -4 }}
        transition={{ type: "spring", stiffness: 400, damping: 15 }}
      >
        {icon}
      </motion.div>
      <h3 className="mb-1.5 text-sm font-semibold text-[color:var(--text-primary)]">{title}</h3>
      <p className="text-sm leading-relaxed text-[color:var(--text-secondary)]">{children}</p>
    </motion.div>
  );
}
