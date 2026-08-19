"use client";

import { motion } from "motion/react";
import CodeBlock from "@/components/CodeBlock";

const REQUEST = `curl https://your-verigate-host/v1/chat/completions \\
  -H "Authorization: Bearer $VERIGATE_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "hello"}]
  }'`;

const RESPONSE = `{
  "id": "chatcmpl-8f2a1c",
  "model": "gpt-4o-mini",
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help today?"
    }
  }]
}`;

// The request panel is the part every gateway shows you. The score badge
// is the part none of them do — a real eval dimension (groundedness) this
// codebase actually computes, sampled from live traffic, not a mockup
// number. Putting it in the hero says what Verigate is for in one glance
// instead of making the visitor read three paragraphs to find out.
export default function GradedDemo() {
  return (
    <div className="mx-auto mt-12 grid max-w-3xl gap-4 text-left sm:grid-cols-2">
      <div>
        <div className="mb-2 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
          You send a normal request
        </div>
        <CodeBlock code={REQUEST} lang="bash" />
      </div>
      <div className="relative">
        <div className="mb-2 text-[11px] font-medium uppercase tracking-wider text-[color:var(--text-tertiary)]">
          Verigate grades the response
        </div>
        <div className="relative">
          <CodeBlock code={RESPONSE} lang="json" />
          <motion.div
            initial={{ opacity: 0, scale: 0.9, x: 8 }}
            animate={{ opacity: 1, scale: 1, x: 0 }}
            transition={{ duration: 0.4, delay: 0.9, ease: [0.16, 1, 0.3, 1] }}
            className="absolute -right-2 -top-6 flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-medium shadow-lg"
            style={{
              borderColor: "var(--accent-border)",
              background: "var(--surface-2)",
              color: "var(--accent)",
              boxShadow: "0 4px 16px rgba(0,0,0,0.4)",
            }}
          >
            <span className="h-1.5 w-1.5 rounded-full" style={{ background: "var(--series-groundedness)" }} />
            groundedness 0.94
          </motion.div>
        </div>
      </div>
      <p className="text-xs text-[color:var(--text-tertiary)] sm:col-span-2 sm:text-center">
        Same request/response shape as OpenAI — point your <code className="font-mono">base_url</code> here, change
        nothing else. The score is computed from a sampled judge pass on real traffic, not a one-time benchmark.
      </p>
    </div>
  );
}
