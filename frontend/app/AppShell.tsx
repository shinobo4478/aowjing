"use client";

import { Layout } from "antd";
import { SURFACE } from "@/lib/tape";
import SiteHeader from "./SiteHeader";

/**
 * Client-side app chrome. antd's `Layout.*` compound components can't be
 * rendered from a Server Component (the sub-components resolve to undefined
 * across the RSC boundary), so the shell lives here.
 */
export default function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Layout.Header
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
      <Layout.Content
        style={{
          padding: "clamp(28px, 5vw, 64px) clamp(16px, 4vw, 32px) 112px",
        }}
      >
        <div style={{ maxWidth: 1040, margin: "0 auto" }}>{children}</div>
      </Layout.Content>
    </Layout>
  );
}
