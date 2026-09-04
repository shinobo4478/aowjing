"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button, Drawer, Grid, Menu, Space, Typography, theme } from "antd";
import { useAuth } from "./AuthGate";

const links = [
  { key: "/profiles", label: "Profiles" },
  { key: "/settings", label: "Settings" },
];

export default function SiteHeader() {
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const { token } = theme.useToken();
  const screens = Grid.useBreakpoint();
  const [drawerOpen, setDrawerOpen] = useState(false);

  // Assume desktop until we know otherwise — avoids a hamburger flash on load.
  const isDesktop = screens.md !== false;
  const selected = links
    .filter((l) => pathname.startsWith(l.key))
    .map((l) => l.key);

  const menuItems = links.map((l) => ({
    key: l.key,
    label: <Link href={l.key}>{l.label}</Link>,
  }));

  const brand = (
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
  );

  if (isDesktop) {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 24,
          height: "100%",
        }}
      >
        {brand}
        <Menu
          mode="horizontal"
          theme="light"
          selectedKeys={selected}
          items={menuItems}
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

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        height: "100%",
      }}
    >
      {brand}
      <Button
        type="text"
        aria-label="Open menu"
        onClick={() => setDrawerOpen(true)}
        style={{ fontSize: 20, lineHeight: 1 }}
      >
        ☰
      </Button>
      <Drawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        placement="right"
        size={240}
        title={user.username}
        styles={{ body: { padding: 0 } }}
      >
        <Menu
          mode="inline"
          selectedKeys={selected}
          items={menuItems}
          onClick={() => setDrawerOpen(false)}
          style={{ borderInlineEnd: "none" }}
        />
        <div style={{ padding: 16 }}>
          <Button
            block
            onClick={() => {
              setDrawerOpen(false);
              logout();
            }}
          >
            Sign out
          </Button>
        </div>
      </Drawer>
    </div>
  );
}
