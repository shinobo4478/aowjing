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
const SettingsIcon = () => (
  <svg {...svg}>
    <path d="M4 7h10M18 7h2M4 17h2M10 17h10" />
    <circle cx="16" cy="7" r="2" />
    <circle cx="8" cy="17" r="2" />
  </svg>
);
const AccountIcon = () => (
  <svg {...svg}>
    <circle cx="12" cy="8" r="4" />
    <path d="M4 21a8 8 0 0 1 16 0" />
  </svg>
);

const tabs = [
  { href: "/profiles", label: "Profiles", icon: <ProfilesIcon /> },
  { href: "/products", label: "Products", icon: <ProductsIcon /> },
  { href: "/settings", label: "Settings", icon: <SettingsIcon /> },
];

/**
 * Phone navigation: a tab bar fixed to the bottom of the screen, where a thumb
 * reaches it, instead of a menu behind a button. The last tab is the account
 * (name + sign out). Pads for the home-indicator safe area.
 */
export default function BottomNav() {
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const [open, setOpen] = useState(false);

  return (
    <>
      <nav className="acmp-tabbar" aria-label="Main">
        {tabs.map((t) => {
          const active = pathname.startsWith(t.href);
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
