"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button, Drawer, Typography } from "antd";
import { useAuth } from "./AuthGate";

// Hand-drawn 24px icons (stroke = currentColor); no icon package to add.
const svg = {
  width: 24,
  height: 24,
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

const HomeIcon = () => (
  <svg {...svg}>
    <path d="M4 11l8-7 8 7" />
    <path d="M6 10v10h12V10" />
  </svg>
);
const DashboardIcon = () => (
  <svg {...svg}>
    <path d="M5 20V11M12 20V4M19 20v-6" />
  </svg>
);
const ProfilesIcon = () => (
  <svg {...svg}>
    <rect x="3" y="4" width="18" height="16" rx="2" />
    <path d="M3 9h18M8 4v5M16 4v5" />
  </svg>
);
const ProductsIcon = () => (
  <svg {...svg}>
    <path d="M6 7h12l1 13H5L6 7z" />
    <path d="M9 7a3 3 0 0 1 6 0" />
  </svg>
);
const AccountIcon = () => (
  <svg {...svg}>
    <circle cx="12" cy="8" r="4" />
    <path d="M4 21a8 8 0 0 1 16 0" />
  </svg>
);

const tabs = [
  { href: "/", label: "Home", icon: <HomeIcon /> },
  { href: "/dashboard", label: "Dashboard", icon: <DashboardIcon /> },
  { href: "/profiles", label: "Profiles", icon: <ProfilesIcon /> },
  { href: "/products", label: "Products", icon: <ProductsIcon /> },
];

/**
 * Phone navigation: a tab bar fixed to the bottom of the screen, where a thumb
 * reaches it, instead of a menu behind a button. The last tab is the account
 * (name, Settings, sign out). Pads for the home-indicator safe area.
 */
export default function BottomNav() {
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(false);

  return (
    <>
      <nav className="acmp-tabbar" aria-label="Main">
        {tabs.map((t) => {
          // "/" would prefix-match every page, so Home is matched exactly.
          const active = t.href === "/" ? pathname === "/" : pathname.startsWith(t.href);
          return (
            <Link
              key={t.href}
              href={t.href}
              className={active ? "acmp-tab is-active" : "acmp-tab"}
              aria-current={active ? "page" : undefined}
            >
              {t.icon}
              <span>{t.label}</span>
            </Link>
          );
        })}
        <button
          type="button"
          className={open ? "acmp-tab is-active" : "acmp-tab"}
          onClick={() => setOpen(true)}
        >
          <AccountIcon />
          <span>Account</span>
        </button>
      </nav>

      <Drawer
        open={open}
        onClose={() => setOpen(false)}
        placement="bottom"
        size="auto"
        title="Account"
        styles={{ body: { paddingBottom: "calc(24px + env(safe-area-inset-bottom))" } }}
      >
        <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
          Signed in as <strong>{user.username}</strong>
        </Typography.Paragraph>
        <Link href="/settings" onClick={() => setOpen(false)}>
          <Button block size="large" style={{ marginBottom: 12 }}>
            Settings
          </Button>
        </Link>
        <Button
          block
          size="large"
          onClick={() => {
            setOpen(false);
            logout();
          }}
        >
          Sign out
        </Button>
      </Drawer>
    </>
  );
}
