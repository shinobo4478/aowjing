"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Tooltip, Typography } from "antd";
import { getFalCredits } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import type { FalCredits } from "@/lib/types";

/** Fired after something that spends fal credits finishes (pictures made, a
 *  clip done), so the balance in the menu bar catches up. */
export const CREDITS_REFRESH_EVENT = "credits:refresh";

export function refreshCredits() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(CREDITS_REFRESH_EVENT));
  }
}

const POLL_MS = 2 * 60 * 1000;

/**
 * The fal.ai credit balance, for the menu bar. It needs its own optional key on
 * the BILLING preset (Settings); until that is saved it shows a quiet link to
 * set it up, and if fal refuses the key it says so in a tooltip instead of
 * breaking the page.
 */
export default function CreditsBadge() {
  const [credits, setCredits] = useState<FalCredits | null>(null);

  const load = useCallback((refresh: boolean) => {
    getFalCredits(refresh)
      .then(setCredits)
      .catch(() => {
        /* the badge is a convenience; stay as it was */
      });
  }, []);

  useEffect(() => {
    load(false);
    const iv = setInterval(() => load(false), POLL_MS);
    const onRefresh = () => load(true);
    window.addEventListener(CREDITS_REFRESH_EVENT, onRefresh);
    return () => {
      clearInterval(iv);
      window.removeEventListener(CREDITS_REFRESH_EVENT, onRefresh);
    };
  }, [load]);

  if (!credits) return null;

  if (!credits.configured) {
    return (
      <Tooltip title="Add a fal.ai billing key in Settings to see your remaining credits here.">
        <Link href="/settings" style={{ fontSize: 13 }}>
          <Typography.Text type="secondary">fal credits: set up</Typography.Text>
        </Link>
      </Tooltip>
    );
  }

  if (credits.balance === undefined) {
    return (
      <Tooltip title={credits.error ?? "Could not read the balance."}>
        <Link href="/settings" style={{ fontSize: 13 }}>
          <Typography.Text type="warning">fal credits: unavailable</Typography.Text>
        </Link>
      </Tooltip>
    );
  }

  return (
    <Tooltip title="Remaining fal.ai credit (updates after each clip or picture)">
      <Typography.Text style={{ fontSize: 13, whiteSpace: "nowrap" }}>
        fal {credits.currency === "USD" || !credits.currency
          ? formatUsd(credits.balance)
          : `${credits.balance} ${credits.currency}`}
      </Typography.Text>
    </Tooltip>
  );
}
