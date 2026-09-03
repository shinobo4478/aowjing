"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Menu } from "antd";

const items = [
  { key: "/profiles", label: <Link href="/profiles">Profiles</Link> },
];

export default function SiteHeader() {
  const pathname = usePathname();
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
    </div>
  );
}
