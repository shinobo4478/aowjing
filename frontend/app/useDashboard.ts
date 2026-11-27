"use client";

import { useCallback, useEffect, useState } from "react";
import { getDashboard, getFalCredits } from "@/lib/api";
import type { Dashboard, FalCredits } from "@/lib/types";
import { CREDITS_REFRESH_EVENT } from "./CreditsBadge";
import { REFRESH_MS } from "./dashboardParts";

/**
 * The dashboard numbers plus the credit balance, kept fresh while a page that
 * shows them is open: every 30 s, and at once after something spends credits
 * (the same event the menu-bar balance listens to).
 */
export function useDashboard() {
  const [data, setData] = useState<Dashboard | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [credits, setCredits] = useState<FalCredits | null>(null);

  const load = useCallback(() => {
    getDashboard()
      .then((d) => {
        setData(d);
        setError(null);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Could not load the dashboard."));
    getFalCredits()
      .then(setCredits)
      .catch(() => {
        /* the credits tile just stays empty */
      });
  }, []);

  useEffect(() => {
    load();
    const iv = setInterval(load, REFRESH_MS);
    window.addEventListener(CREDITS_REFRESH_EVENT, load);
    return () => {
      clearInterval(iv);
      window.removeEventListener(CREDITS_REFRESH_EVENT, load);
    };
  }, [load]);

  return { data, error, credits };
}
