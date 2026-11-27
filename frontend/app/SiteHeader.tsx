"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button, Grid, Menu, Space, Typography } from "antd";
import Stripe from "@/components/Stripe";
import { INK } from "@/lib/tape";
import { useAuth } from "./AuthGate";
import CreditsBadge from "./CreditsBadge";
import { useThemeMode } from "./ThemeMode";
import ThemeSwitch from "./ThemeSwitch";

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
  const { mode } = useThemeMode();

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
          theme={mode}
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
          <CreditsBadge />
          <ThemeSwitch />
          <Typography.Text type="secondary">{user.username}</Typography.Text>
          <Button size="small" onClick={logout}>
            Sign out
          </Button>
        </Space>
      </div>
    );
  }

  // Phones: a slim top bar (brand, balance, theme). Navigation is the tab bar
  // at the bottom (BottomNav), where a thumb reaches it.
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 12,
        height: "100%",
      }}
    >
      <Brand />
      <Space size={12}>
        <CreditsBadge />
        <ThemeSwitch />
      </Space>
    </div>
  );
}
