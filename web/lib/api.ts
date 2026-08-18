const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type RequestRecord = {
  id: string;
  created_at: string;
  provider: string;
  model: string;
  prompt: string;
  response: string;
  latency_ms: number;
  cache_hit: boolean;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
  status: "ok" | "error";
};

export type EvalRecord = {
  id: string;
  request_id: string;
  rubric: string;
  score: number;
  reasoning: string;
  created_at: string;
};

export type EvalSummary = {
  rolling_avg_score: number;
  sample_count: number;
  status: "ok" | "regressed";
};

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`${path} failed: ${res.status}`);
  }
  return res.json();
}

export const fetchRequests = () => get<RequestRecord[]>("/api/requests");
export const fetchEvalSummary = () => get<EvalSummary>("/api/evals/summary");
export const fetchRecentEvals = () => get<EvalRecord[]>("/api/evals/recent");
