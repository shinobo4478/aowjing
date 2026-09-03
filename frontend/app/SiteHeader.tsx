"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button, Menu, Typography } from "antd";
import { useAuth } from "./AuthGate";

const items = [
  { key: "/profiles", label: <Link href="/profiles">Profiles</Link> },
  { key: "/settings", label: <Link href="/settings">Settings</Link> },
];

export default function SiteHeader() {
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const selected = items
    .filter((i) => pathname.startsWith(i.key))
    .map((i) => i.key);

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 24 }}>
      <Link
        href="/"
        style={{ fontWeight: 700, letterSpacing: 0.5, color: "#fff" }}
      >
        ACMP
      </Link>
      <Menu
        mode="horizontal"
        theme="dark"
        selectedKeys={selected}
        items={items}
        style={{ flex: 1, minWidth: 0, background: "transparent" }}
      />
      <Typography.Text style={{ color: "rgba(255,255,255,0.65)" }}>
        {user.username}
      </Typography.Text>
      <Button size="small" onClick={logout}>
        Sign out
      </Button>
    </div>
  );
}
