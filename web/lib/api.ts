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

export type User = {
  id: string;
  email: string;
  created_at: string;
};

export type AuthResponse = {
  user: User;
  tenant: Tenant;
  session_token: string;
  api_key?: string; // only present on signup — shown once
};

export type MeResponse = {
  tenant: Tenant | null;
};

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

const SESSION_KEY = "verigate_session";
const ADMIN_KEY = "verigate_admin_key";

// Session token (from signup/login) and admin key (the operator's own
// VERIGATE_API_KEY, entered manually) are deliberately separate concepts
// stored under separate keys — a signed-in customer's session must never
// double as admin access, and vice versa.
export function getSessionToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(SESSION_KEY);
}
export function setSessionToken(token: string | null) {
  if (typeof window === "undefined") return;
  if (token) localStorage.setItem(SESSION_KEY, token);
  else localStorage.removeItem(SESSION_KEY);
  sessionTokenListeners.forEach((l) => l());
}

// useSyncExternalStore plumbing for session-context.tsx: reading
// localStorage directly during render (e.g. a useState initializer)
// mismatches between server (no localStorage) and client first paint,
// causing a hydration error — useSyncExternalStore is React's sanctioned
// way to read external mutable state like this without that mismatch.
type Listener = () => void;
const sessionTokenListeners = new Set<Listener>();
export function subscribeSessionToken(listener: Listener) {
  sessionTokenListeners.add(listener);
  return () => sessionTokenListeners.delete(listener);
}
export function getSessionTokenServerSnapshot(): string | null {
  return null;
}
export function getAdminKey(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ADMIN_KEY);
}
export function setAdminKey(key: string | null) {
  if (typeof window === "undefined") return;
  if (key) localStorage.setItem(ADMIN_KEY, key);
  else localStorage.removeItem(ADMIN_KEY);
}

function authHeaders(token?: string | null): HeadersInit {
  const auth = token === undefined ? getSessionToken() : token;
  return auth ? { Authorization: `Bearer ${auth}` } : {};
}

// token: explicit bearer token to send, or omit to use the stored session
// token, or pass null to send no Authorization header at all.
async function get<T>(path: string, token?: string | null): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    cache: "no-store",
    headers: authHeaders(token),
  });
  if (!res.ok) {
    throw new Error(`${path} failed: ${res.status}`);
  }
  return res.json();
}

async function post<T>(path: string, body: unknown, token?: string | null): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders(token) },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? `${path} failed: ${res.status}`);
  }
  return res.json();
}

// Dashboard reads: no token argument, so they automatically pick up the
// signed-in user's session token when one exists, and fall back to the
// anonymous/demo scope when it doesn't — scoping happens server-side.
export const fetchRequests = () => get<RequestRecord[]>("/api/requests");
export const fetchEvalSummary = () => get<EvalSummary>("/api/evals/summary");
export const fetchRecentEvals = () => get<EvalRecord[]>("/api/evals/recent");
export const fetchTenants = () => get<Tenant[]>("/api/tenants");
// Operator view (app/tenants page): sends the admin key explicitly so it
// lists every tenant, rather than whatever the signed-in user's own
// session would scope it to.
export const fetchTenantsAsAdmin = () => get<Tenant[]>("/api/tenants", getAdminKey());
export const fetchProviderStatus = () => get<ProviderStatusResponse>("/api/providers");
export const fetchMe = () => get<MeResponse>("/api/me");

// Self-serve account. signUp/login return a session token the caller
// should persist with setSessionToken.
export const signUp = (email: string, password: string) =>
  post<AuthResponse>("/api/auth/signup", { email, password });
export const login = (email: string, password: string) =>
  post<AuthResponse>("/api/auth/login", { email, password });
export const logout = () => post<{ status: string }>("/api/auth/logout", {});
export const regenerateKey = () => post<{ api_key: string }>("/api/tenant/regenerate-key", {});

// Admin-gated operations use the operator's own key (stored separately,
// see getAdminKey/setAdminKey), never a customer session — the backend
// rejects anything else with 401 regardless of what's passed here.
export const createTenant = (name: string, rateLimitRPM: number) =>
  post<CreatedTenant>("/api/tenants", { name, rate_limit_rpm: rateLimitRPM }, getAdminKey());
export const runReplay = (candidateModel: string, limit: number, requestId?: string) =>
  post<ReplayResponse>(
    "/api/replay",
    { candidate_model: candidateModel, limit, request_id: requestId ?? "" },
    getAdminKey(),
  );
