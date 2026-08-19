"use client";

import useSWR from "swr";
import { motion } from "motion/react";
import { fetchRequests, fetchProviderStatus } from "@/lib/api";

// Genuinely live, not decorative — polls the same endpoints the dashboard
// does. Fails quietly to a static "open source" badge if the backend isn't
// reachable, so the landing page never looks broken when Verigate itself
// isn't running.
export default function LiveBadge() {
  const { data: requests } = useSWR("landing-requests", fetchRequests, { refreshInterval: 5000, shouldRetryOnError: false });
  const { data: providers } = useSWR("landing-providers", fetchProviderStatus, { refreshInterval: 5000, shouldRetryOnError: false });

  const live = Boolean(requests && providers);
  const healthy = providers?.providers.every((p) => p.breaker_state !== "open") ?? true;

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.4 }}
      className="mx-auto mb-6 inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium"
      style={{
        borderColor: live ? "var(--accent-border)" : "var(--border)",
        background: live ? "var(--accent-soft)" : "var(--surface-1)",
        color: live ? "var(--accent)" : "var(--text-secondary)",
      }}
    >
      <span className="relative flex h-1.5 w-1.5">
        {live && (
          <span
            className="absolute inline-flex h-full w-full animate-ping rounded-full opacity-75"
            style={{ background: healthy ? "var(--status-good)" : "var(--status-warning)" }}
          />
        )}
        <span
          className="relative inline-flex h-1.5 w-1.5 rounded-full"
          style={{ background: live ? (healthy ? "var(--status-good)" : "var(--status-warning)") : "currentColor" }}
        />
      </span>
      {live ? (
        <span>
          Live — {requests!.length} requests logged, {providers!.providers.length} provider
          {providers!.providers.length === 1 ? "" : "s"} {healthy ? "healthy" : "degraded"}
        </span>
      ) : (
        <span>Open source · MIT licensed</span>
      )}
    </motion.div>
  );
}
