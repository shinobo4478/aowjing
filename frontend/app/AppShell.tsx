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
      <Layout.Header>
        <SiteHeader />
      </Layout.Header>
      <Layout.Content style={{ padding: "32px 24px 64px" }}>
        <div style={{ maxWidth: 820, margin: "0 auto" }}>{children}</div>
      </Layout.Content>
    </Layout>
  );
}
