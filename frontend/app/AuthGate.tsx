"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { usePathname, useRouter } from "next/navigation";
import { Spin } from "antd";
import { getMe, logout as apiLogout, type User } from "@/lib/auth";
import AppShell from "./AppShell";

interface AuthValue {
  user: User;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthValue | null>(null);

/** Available to any component rendered inside the authenticated shell. */
export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthGate");
  return ctx;
}

type Status = "loading" | "authed" | "anon";

const LOGIN_PATH = "/login";

/**
 * Gates the whole app on a live backend session.
 *
 * The session cookie is HttpOnly and set on the API origin, so the only way to
 * know if we're signed in is to call /auth/me. While that's in flight we show a
 * spinner; on 401 we bounce to /login. The login page renders outside the gate.
 */
export default function AuthGate({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const onLoginPage = pathname === LOGIN_PATH;

  const [status, setStatus] = useState<Status>("loading");
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    if (onLoginPage) return;
    let cancelled = false;
    getMe()
      .then((u) => {
        if (!cancelled) {
          setUser(u);
          setStatus("authed");
        }
      })
      .catch(() => {
        if (!cancelled) setStatus("anon");
      });
    return () => {
      cancelled = true;
    };
  }, [onLoginPage, pathname]);

  useEffect(() => {
    if (status === "anon" && !onLoginPage) router.replace(LOGIN_PATH);
  }, [status, onLoginPage, router]);

  // A lapsed session on any API call (dispatched from lib/api.ts).
  useEffect(() => {
    const onExpired = () => {
      setUser(null);
      setStatus("anon");
    };
    window.addEventListener("auth:expired", onExpired);
    return () => window.removeEventListener("auth:expired", onExpired);
  }, []);

  // The login page manages its own layout.
  if (onLoginPage) return <>{children}</>;

  if (status !== "authed" || !user) {
    return (
      <div style={{ display: "grid", placeItems: "center", minHeight: "100vh" }}>
        <Spin size="large" />
      </div>
    );
  }

  const logout = async () => {
    await apiLogout().catch(() => {});
    router.replace(LOGIN_PATH);
  };

  return (
    <AuthContext.Provider value={{ user, logout }}>
      <AppShell>{children}</AppShell>
    </AuthContext.Provider>
  );
}
