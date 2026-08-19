"use client";

import { motion } from "motion/react";

// Two slow-drifting radial blobs plus a scanning line behind the hero.
// The blobs are subtle, low-chroma, never competing with the accent color
// used for real UI. The scan line is the one deliberate brand touch: a
// thin accent sweep top-to-bottom, standing in for what Verigate actually
// does — reading every response as it passes through. Sized and positioned
// relative to the full-width <section>, not the narrow max-w-4xl content
// column, so the glow reads as ambient rather than a clipped box. Respects
// prefers-reduced-motion via Motion's own handling of transform-only
// animations plus a CSS media-query fallback that halts them outright.
export default function AnimatedBackground() {
  return (
    <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden motion-reduce:hidden">
      <motion.div
        className="absolute -left-60 -top-60 h-[640px] w-[640px] rounded-full blur-3xl"
        style={{ background: "radial-gradient(circle, rgba(45,212,191,0.09), transparent 72%)" }}
        animate={{ x: [0, 40, 0], y: [0, 30, 0] }}
        transition={{ duration: 18, repeat: Infinity, ease: "easeInOut" }}
      />
      <motion.div
        className="absolute -right-60 -top-20 h-[640px] w-[640px] rounded-full blur-3xl"
        style={{ background: "radial-gradient(circle, rgba(99,102,241,0.08), transparent 72%)" }}
        animate={{ x: [0, -30, 0], y: [0, 40, 0] }}
        transition={{ duration: 22, repeat: Infinity, ease: "easeInOut" }}
      />
      <motion.div
        className="absolute inset-x-0 h-px"
        style={{
          top: 0,
          background: "linear-gradient(90deg, transparent, var(--accent) 45%, var(--accent) 55%, transparent)",
        }}
        animate={{ top: ["0%", "92%", "0%"], opacity: [0, 0.35, 0.35, 0] }}
        transition={{ duration: 7, repeat: Infinity, ease: "easeInOut", times: [0, 0.5, 0.85, 1] }}
      />
    </div>
  );
}
