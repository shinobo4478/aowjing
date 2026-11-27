"use client";

import { Grid, Layout } from "antd";
import { SURFACE } from "@/lib/tape";
import BottomNav from "./BottomNav";
import SiteHeader from "./SiteHeader";

/**
 * Client-side app chrome. antd's `Layout.*` compound components can't be
 * rendered from a Server Component (the sub-components resolve to undefined
 * across the RSC boundary), so the shell lives here.
 */
export default function AppShell({ children }: { children: React.ReactNode }) {
  // Assume desktop until we know otherwise — avoids a tab-bar flash on load.
  const isPhone = Grid.useBreakpoint().md === false;
  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Layout.Header
        className="acmp-header"
        style={{
          position: "sticky",
          top: 0,
          zIndex: 10,
          paddingInline: "clamp(16px, 4vw, 32px)",
          borderBottom: `1px solid ${SURFACE.ruleSoft}`,
        }}
      >
        <SiteHeader />
      </Layout.Header>
      <Layout.Content className="acmp-content">
        <div style={{ maxWidth: 1040, margin: "0 auto" }}>{children}</div>
      </Layout.Content>
      {isPhone && <BottomNav />}
    </Layout>
  );
}
