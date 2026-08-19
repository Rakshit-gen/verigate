"use client";

import { motion } from "motion/react";

function Node({ label, sub, accent }: { label: string; sub: string; accent?: boolean }) {
  return (
    <motion.div
      className="rounded-xl border px-4 py-3 text-center"
      style={{
        borderColor: accent ? "var(--accent-border)" : "var(--border)",
        background: accent ? "var(--accent-soft)" : "var(--surface-2)",
      }}
      whileHover={{ y: -3, scale: 1.03 }}
      transition={{ type: "spring", stiffness: 300, damping: 20 }}
    >
      <div className="font-mono text-sm font-medium" style={{ color: accent ? "var(--accent)" : "var(--text-primary)" }}>
        {label}
      </div>
      <div className="mt-0.5 text-[11px] text-[color:var(--text-tertiary)]">{sub}</div>
    </motion.div>
  );
}

// A small dot animates along the connector to suggest live traffic moving
// through the gateway — cheap (opacity + position keyframes on one element
// per arrow), not a full path-morphing animation. Labels sit in the empty
// space beside/below the line (never above, where a node's box already
// is) so a long label can never render on top of a neighboring node.
function Arrow({ vertical = false, label, delay = 0 }: { vertical?: boolean; label?: string; delay?: number }) {
  return (
    <div className={`relative flex shrink-0 items-center justify-center ${vertical ? "h-10 flex-col" : "w-14 px-1"}`}>
      {label && (
        <span
          className={`absolute whitespace-nowrap text-[10px] text-[color:var(--text-tertiary)] ${
            vertical ? "left-full top-1/2 ml-2 -translate-y-1/2" : "left-1/2 top-full mt-1.5 -translate-x-1/2"
          }`}
        >
          {label}
        </span>
      )}
      <div className={`bg-[color:var(--border)] ${vertical ? "h-full w-px" : "h-px w-full"}`} />
      <motion.span
        className="absolute h-1.5 w-1.5 rounded-full"
        style={{ background: "var(--accent)" }}
        animate={
          vertical
            ? { top: ["0%", "90%"], opacity: [0, 1, 1, 0] }
            : { left: ["0%", "90%"], opacity: [0, 1, 1, 0] }
        }
        transition={{ duration: 1.6, repeat: Infinity, repeatDelay: 1.4, delay, ease: "easeInOut" }}
      />
      <span className={`absolute text-[color:var(--text-tertiary)] ${vertical ? "bottom-0" : "right-0"}`}>
        {vertical ? "▾" : "▸"}
      </span>
    </div>
  );
}

export default function FlowDiagram() {
  return (
    <div className="rounded-2xl border border-[color:var(--border)] bg-[color:var(--surface-1)] p-6 sm:p-8">
      <div className="flex flex-col items-center gap-6 sm:flex-row sm:justify-center sm:gap-0">
        <Node label="Your app" sub="OpenAI-shaped request" />
        <Arrow delay={0} />
        <Node label="Verigate" sub="cache · route · guard" accent />
        <Arrow label="miss" delay={0.4} />
        <Node label="Provider" sub="Groq · OpenAI · Anthropic" />
      </div>
      <div className="mt-8 flex flex-col items-center gap-2 sm:mt-10">
        <Arrow vertical label="sampled" delay={0.8} />
        <div className="flex flex-col items-center gap-6 sm:flex-row">
          <Node label="Judge model" sub="scores vs. rubric" />
          <Arrow delay={1.2} />
          <Node label="Dashboard" sub="regression + alerts" />
        </div>
      </div>
    </div>
  );
}
