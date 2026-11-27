"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Button, Drawer, Grid, Popover, Spin, Typography } from "antd";
import type { RunningItem } from "@/lib/types";
import { useRunning } from "./useRunning";

function elapsed(since: string, now: number): string {
  const s = Math.max(0, Math.floor((now - new Date(since).getTime()) / 1000));
  const m = Math.floor(s / 60);
  return m > 0 ? `${m} min ${String(s % 60).padStart(2, "0")} s` : `${s} s`;
}

function Row({ item, now, onNavigate }: { item: RunningItem; now: number; onNavigate: () => void }) {
  const tab = item.kind === "clip" ? "?tab=generations" : "";
  const detail = [
    item.profileName,
    item.kind === "clip" && item.model ? `${item.model}${item.seconds ? `, ${item.seconds} s` : ""}` : null,
  ]
    .filter(Boolean)
    .join(" · ");
  return (
    <Link
      href={`/profiles/${item.profileId}${tab}`}
      onClick={onNavigate}
      style={{ display: "flex", gap: 12, alignItems: "center", padding: "8px 0", color: "inherit" }}
    >
      <Spin size="small" />
      <div style={{ minWidth: 0, flex: 1 }}>
        <div style={{ fontWeight: 600, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
          {item.label}
        </div>
        <Typography.Text type="secondary" style={{ fontSize: 13 }}>
          {detail}
        </Typography.Text>
      </div>
      <Typography.Text type="secondary" style={{ fontSize: 13, whiteSpace: "nowrap" }}>
        {elapsed(item.createdAt, now)}
      </Typography.Text>
    </Link>
  );
}

/**
 * The "rendering" indicator in the menu bar. It shows nothing while nothing is
 * being made; otherwise a pill with a spinner and the count, and a click opens a
 * popover listing every clip and picture request in progress (what, which
 * profile, how long) with a link to where it is. On a phone that list is a sheet
 * that slides up from the bottom instead (a popover would not fit the screen).
 */
export default function RunningBadge() {
  const items = useRunning();
  const [open, setOpen] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  // On a phone the top bar is crowded: just the spinner and the count.
  const compact = Grid.useBreakpoint().md === false;

  const busy = items.length > 0;

  // The elapsed times tick once a second while something is rendering (only
  // this small component re-renders).
  useEffect(() => {
    if (!busy) return;
    const iv = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(iv);
  }, [busy]);

  if (!busy) return null;

  const content = (
    <div style={{ width: compact ? "100%" : 320 }}>
      {items.map((item) => (
        <Row key={item.id} item={item} now={now} onNavigate={() => setOpen(false)} />
      ))}
      <Typography.Text type="secondary" style={{ display: "block", fontSize: 12, marginTop: 6 }}>
        Clips usually take several minutes. This list updates by itself.
      </Typography.Text>
    </div>
  );

  const title = `Rendering now (${items.length})`;
  const pill = (
    <Button
      size="small"
      shape="round"
      aria-label={`${items.length} rendering. Show details`}
      onClick={compact ? () => setOpen(true) : undefined}
      style={{ display: "inline-flex", alignItems: "center", gap: 8 }}
    >
      <Spin size="small" />
      <span style={{ whiteSpace: "nowrap" }}>{compact ? items.length : `${items.length} rendering`}</span>
    </Button>
  );

  if (compact) {
    return (
      <>
        {pill}
        <Drawer
          placement="bottom"
          title={title}
          open={open}
          onClose={() => setOpen(false)}
          size="auto"
          styles={{ body: { paddingBottom: "calc(16px + env(safe-area-inset-bottom))" } }}
        >
          {content}
        </Drawer>
      </>
    );
  }

  return (
    <Popover
      trigger="click"
      placement="bottomRight"
      arrow={false}
      title={title}
      content={content}
      open={open}
      onOpenChange={setOpen}
    >
      {pill}
    </Popover>
  );
}
