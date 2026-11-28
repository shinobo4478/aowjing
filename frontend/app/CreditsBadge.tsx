"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Button, Popover, Space, Tooltip, Typography } from "antd";
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

/** fal's billing page, where credits are added (fal's docs and FAQ send people
 *  there to top up). Nothing deeper is known to exist: no top-up dialog has a
 *  documented link, and a signed-out visitor is asked to sign in by fal itself. */
const FAL_BILLING_URL = "https://fal.ai/dashboard/billing";
const FAL_HOME_URL = "https://fal.ai/dashboard";

/**
 * The fal.ai credit balance, for the menu bar. It needs its own optional key on
 * the BILLING preset (Settings); until that is saved it shows a quiet link to
 * set it up, and if fal refuses the key it says so in a tooltip instead of
 * breaking the page. Clicking the balance opens a small popover that links to
 * fal's billing page (to add credits) and to fal's dashboard.
 */
export default function CreditsBadge() {
  const [credits, setCredits] = useState<FalCredits | null>(null);
  const [open, setOpen] = useState(false);

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

  const balance =
    credits.currency === "USD" || !credits.currency
      ? formatUsd(credits.balance)
      : `${credits.balance} ${credits.currency}`;

  return (
    <Popover
      trigger="click"
      placement="bottomRight"
      open={open}
      onOpenChange={setOpen}
      title="เครดิต fal.ai"
      content={
        <div style={{ maxWidth: 260 }}>
          <Typography.Text style={{ display: "block", fontSize: 22, fontWeight: 600 }}>{balance}</Typography.Text>
          <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
            ยอดคงเหลือ (อัปเดตหลังสร้างคลิปหรือภาพเสร็จ)
          </Typography.Text>
          <Space orientation="vertical" size={8} style={{ width: "100%", marginTop: 12 }}>
            <Button
              type="primary"
              block
              href={FAL_BILLING_URL}
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => setOpen(false)}
            >
              เติมเครดิตที่ fal ↗
            </Button>
            <Button
              block
              href={FAL_HOME_URL}
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => setOpen(false)}
            >
              เปิดเว็บ fal.ai ↗
            </Button>
            <Tooltip title="ถามยอดล่าสุดจาก fal ตอนนี้ (ปกติอัปเดตเองทุก 2 นาที)">
              <Button block type="text" onClick={() => load(true)}>
                รีเฟรชยอด
              </Button>
            </Tooltip>
          </Space>
          <Typography.Text type="secondary" style={{ display: "block", fontSize: 12, marginTop: 8 }}>
            ปุ่มแรกเปิดหน้า Billing ของ fal ถ้ายังไม่ได้ล็อกอิน fal จะให้ล็อกอินก่อน
          </Typography.Text>
        </div>
      }
    >
      <button
        type="button"
        aria-label={`fal.ai credit ${balance}. Open the credit menu.`}
        style={{
          font: "inherit",
          fontSize: 13,
          whiteSpace: "nowrap",
          background: "transparent",
          border: 0,
          padding: "4px 6px",
          borderRadius: 8,
          color: "inherit",
          cursor: "pointer",
        }}
      >
        fal {balance}
      </button>
    </Popover>
  );
}
