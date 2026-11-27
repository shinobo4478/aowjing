"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getRunning } from "@/lib/api";
import type { RunningItem } from "@/lib/types";
import { refreshCredits } from "./CreditsBadge";

/** Fired right after something is queued (lib/api.ts), so what shows "rendering"
 *  asks at once instead of at the next poll. */
export const RUNNING_REFRESH_EVENT = "running:refresh";

// Ask often while something renders (a clip takes minutes, and the list should
// empty soon after it finishes); rarely when nothing does.
const BUSY_MS = 5_000;
const IDLE_MS = 20_000;

/**
 * Everything being made right now (clips and picture requests), kept fresh by
 * polling GET /dashboard/running. Used by the menu bar's indicator and the
 * Profiles page. The credit balance is refreshed when the list shrinks, since
 * something finishing is what moves it.
 */
export function useRunning(): RunningItem[] {
  const [items, setItems] = useState<RunningItem[]>([]);
  const previous = useRef(0);

  const load = useCallback(() => {
    getRunning()
      .then((list) => {
        if (previous.current > list.length) refreshCredits();
        previous.current = list.length;
        setItems(list);
      })
      .catch(() => {
        /* a convenience: keep what it showed */
      });
  }, []);

  const busy = items.length > 0;
  useEffect(() => {
    load();
    const iv = setInterval(load, busy ? BUSY_MS : IDLE_MS);
    window.addEventListener(RUNNING_REFRESH_EVENT, load);
    return () => {
      clearInterval(iv);
      window.removeEventListener(RUNNING_REFRESH_EVENT, load);
    };
  }, [load, busy]);

  return items;
}

/** How many clips each profile has rendering right now, by profile id. Picture
 *  requests are left out: they finish in seconds and are not in the Generations tab. */
export function clipsByProfile(items: RunningItem[]): Record<string, number> {
  const out: Record<string, number> = {};
  for (const it of items) {
    if (it.kind === "clip") out[it.profileId] = (out[it.profileId] ?? 0) + 1;
  }
  return out;
}
