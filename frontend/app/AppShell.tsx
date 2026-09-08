"use client";

import { Layout } from "antd";
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
          paddingInline: "clamp(12px, 4vw, 24px)",
          borderBottom: "1px solid rgba(255, 255, 255, 0.06)",
          boxShadow: "0 1px 0 rgba(255, 255, 255, 0.03), 0 4px 16px rgba(0, 0, 0, 0.6)",
        }}
      >
        <SiteHeader />
      </Layout.Header>
      <Layout.Content
        style={{ padding: "clamp(16px, 4vw, 32px) clamp(12px, 4vw, 24px) 64px" }}
      >
        <div style={{ maxWidth: 900, margin: "0 auto" }}>{children}</div>
      </Layout.Content>
    </Layout>
  );
}
