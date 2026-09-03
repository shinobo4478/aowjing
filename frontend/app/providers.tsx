"use client";

import { AntdRegistry } from "@ant-design/nextjs-registry";
import { App, ConfigProvider, theme } from "antd";
import { FONT_FAMILY } from "@/lib/theme";

/**
 * Client-side providers for the whole app.
 *
 * - AntdRegistry: extracts antd's CSS-in-JS on the server so RSC pages render
 *   styled on first paint (no flash). Required per the project stack notes.
 * - ConfigProvider: dark theme + shared token overrides.
 * - App: gives components access to the static message/notification/modal APIs
 *   via `App.useApp()` instead of the context-less top-level imports.
 */
export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AntdRegistry>
      <ConfigProvider
        theme={{
          algorithm: theme.darkAlgorithm,
          token: {
            colorPrimary: "#6ea8fe",
            borderRadius: 8,
            fontFamily: FONT_FAMILY,
          },
        }}
      >
        <App>{children}</App>
      </ConfigProvider>
    </AntdRegistry>
  );
}
