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
  pii_redacted: boolean;
  injection_score: number;
  tool_calls: string;
  tenant_id: string;
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
  baseline_avg: number;
  baseline_stddev: number;
  baseline_count: number;
  z_score: number;
  method: "statistical" | "fixed_threshold_bootstrap";
  status: "ok" | "regressed";
};

export type Tenant = {
  id: string;
  name: string;
  rate_limit_rpm: number;
  created_at: string;
};

export type CreatedTenant = Tenant & { api_key: string };

export type ProviderEntryStatus = {
  name: string;
  declared_order: number;
  breaker_state: "closed" | "open" | "half_open";
  consecutive_failures: number;
  measured_latency_ms: number;
};

export type ProviderStatusResponse = {
  mode: "single" | "chain";
  providers: ProviderEntryStatus[];
};

export type ReplayResult = {
  prompt: string;
  original_model: string;
  candidate_model: string;
  original_scores: Record<string, number>;
  candidate_scores: Record<string, number>;
};

export type ReplayResponse = {
  results: ReplayResult[];
  skipped: string[];
};

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`${path} failed: ${res.status}`);
  }
  return res.json();
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? `${path} failed: ${res.status}`);
  }
  return res.json();
}

export const fetchRequests = () => get<RequestRecord[]>("/api/requests");
export const fetchEvalSummary = () => get<EvalSummary>("/api/evals/summary");
export const fetchRecentEvals = () => get<EvalRecord[]>("/api/evals/recent");
export const fetchTenants = () => get<Tenant[]>("/api/tenants");
export const createTenant = (name: string, rateLimitRPM: number) =>
  post<CreatedTenant>("/api/tenants", { name, rate_limit_rpm: rateLimitRPM });
export const fetchProviderStatus = () => get<ProviderStatusResponse>("/api/providers");
export const runReplay = (candidateModel: string, limit: number, requestId?: string) =>
  post<ReplayResponse>("/api/replay", {
    candidate_model: candidateModel,
    limit,
    request_id: requestId ?? "",
  });
