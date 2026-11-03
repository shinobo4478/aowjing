"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button, Drawer, Grid, Menu, Space, Typography } from "antd";
import Stripe from "@/components/Stripe";
import { INK } from "@/lib/tape";
import { useAuth } from "./AuthGate";

const links = [
  { key: "/profiles", label: "Profiles" },
  { key: "/products", label: "Products" },
  { key: "/settings", label: "Settings" },
];

function Brand() {
  return (
    <Link
      href="/"
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 12,
        color: INK.chalk,
        fontWeight: 700,
        fontSize: 18,
        letterSpacing: "0.04em",
      }}
    >
      <Stripe
        color={INK.chalk}
        size={4}
        style={{ width: 22, height: 22, borderRadius: 2 }}
      />
      ACMP
    </Link>
  );
}

export default function SiteHeader() {
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const screens = Grid.useBreakpoint();
  const [drawerOpen, setDrawerOpen] = useState(false);

  // Assume desktop until we know otherwise — avoids a menu-button flash on load.
  const isDesktop = screens.md !== false;
  const selected = links
    .filter((l) => pathname.startsWith(l.key))
    .map((l) => l.key);

  const menuItems = links.map((l) => ({
    key: l.key,
    label: <Link href={l.key}>{l.label}</Link>,
  }));

  if (isDesktop) {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 40,
          height: "100%",
        }}
      >
        <Brand />
        <Menu
          mode="horizontal"
          theme="dark"
          selectedKeys={selected}
          items={menuItems}
          style={{
            flex: 1,
            minWidth: 0,
            background: "transparent",
            borderBottom: "none",
            lineHeight: "58px",
          }}
        />
        <Space size="middle">
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
      <Brand />
      <Button type="text" onClick={() => setDrawerOpen(true)}>
        Menu
      </Button>
      <Drawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        placement="right"
        size={260}
        title={user.username}
        styles={{ body: { padding: 0 } }}
      >
        <Menu
          mode="inline"
          selectedKeys={selected}
          items={menuItems}
          onClick={() => setDrawerOpen(false)}
          style={{ borderInlineEnd: "none", fontSize: 18 }}
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
