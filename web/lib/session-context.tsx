"use client";

import { createContext, useCallback, useContext, useSyncExternalStore } from "react";
import useSWR, { mutate as globalMutate } from "swr";
import {
  fetchMe,
  getSessionToken,
  getSessionTokenServerSnapshot,
  login as apiLogin,
  logout as apiLogout,
  setSessionToken,
  signUp as apiSignUp,
  subscribeSessionToken,
  type Tenant,
} from "./api";

const ME_KEY = "me";

type SessionState = {
  tenant: Tenant | null;
  loading: boolean;
  signUp: (email: string, password: string) => Promise<{ apiKey: string }>;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const SessionContext = createContext<SessionState | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  // useSyncExternalStore (not useState+effect) reads localStorage safely
  // across SSR/hydration — the server and first client paint both use
  // getSessionTokenServerSnapshot (null), so there's nothing to mismatch.
  const sessionToken = useSyncExternalStore(subscribeSessionToken, getSessionToken, getSessionTokenServerSnapshot);
  const { data, isLoading } = useSWR(sessionToken ? ME_KEY : null, fetchMe);
  const tenant = data?.tenant ?? null;

  const signUp = useCallback(async (email: string, password: string) => {
    const res = await apiSignUp(email, password);
    // Seed the "me" cache before flipping the token — by the time
    // useSyncExternalStore re-renders this provider with the new token and
    // useSWR switches its key to "me", the data's already there.
    await globalMutate(ME_KEY, { tenant: res.tenant }, false);
    setSessionToken(res.session_token);
    return { apiKey: res.api_key! };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const res = await apiLogin(email, password);
    await globalMutate(ME_KEY, { tenant: res.tenant }, false);
    setSessionToken(res.session_token);
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiLogout();
    } finally {
      setSessionToken(null);
      await globalMutate(ME_KEY, undefined, false);
    }
  }, []);

  return (
    <SessionContext.Provider value={{ tenant, loading: isLoading, signUp, login, logout }}>
      {children}
    </SessionContext.Provider>
  );
}

export function useSession() {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error("useSession must be used within a SessionProvider");
  return ctx;
}
