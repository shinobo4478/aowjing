"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button, Menu, Space, Typography, theme } from "antd";
import { useAuth } from "./AuthGate";

const items = [
  { key: "/profiles", label: <Link href="/profiles">Profiles</Link> },
  { key: "/settings", label: <Link href="/settings">Settings</Link> },
];

export default function SiteHeader() {
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const { token } = theme.useToken();
  const selected = items
    .filter((i) => pathname.startsWith(i.key))
    .map((i) => i.key);

  return (
    <div
      style={{ display: "flex", alignItems: "center", gap: 24, height: "100%" }}
    >
      <Link
        href="/"
        style={{
          fontWeight: 700,
          fontSize: 16,
          letterSpacing: 0.3,
          color: token.colorPrimary,
        }}
      >
        ACMP
      </Link>
      <Menu
        mode="horizontal"
        theme="light"
        selectedKeys={selected}
        items={items}
        style={{
          flex: 1,
          minWidth: 0,
          background: "transparent",
          borderBottom: "none",
        }}
      />
      <Space size="small">
        <Typography.Text type="secondary">{user.username}</Typography.Text>
        <Button size="small" onClick={logout}>
          Sign out
        </Button>
      </Space>
    </div>
  );
}
