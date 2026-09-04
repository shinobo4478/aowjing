"use client";

import { AntdRegistry } from "@ant-design/nextjs-registry";
import { App, ConfigProvider, theme } from "antd";
import { FONT_FAMILY } from "@/lib/theme";

/**
 * Client-side providers for the whole app.
 *
 * - AntdRegistry: extracts antd's CSS-in-JS on the server so RSC pages render
 *   styled on first paint (no flash).
 * - ConfigProvider: the app's light theme — an indigo primary, softer radii,
 *   a tinted page background so white cards and tables lift off it.
 * - App: gives components access to the static message/notification/modal APIs
 *   via `App.useApp()`.
 */
export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AntdRegistry>
      <ConfigProvider
        theme={{
          algorithm: theme.defaultAlgorithm,
          token: {
            colorPrimary: "#4f46e5",
            colorInfo: "#4f46e5",
            colorBgLayout: "#f6f7f9",
            borderRadius: 8,
            borderRadiusLG: 12,
            controlHeight: 36,
            fontFamily: FONT_FAMILY,
          },
          components: {
            Layout: {
              headerBg: "#ffffff",
              headerHeight: 56,
            },
            Menu: {
              itemBg: "transparent",
              activeBarHeight: 2,
            },
            Card: {
              paddingLG: 24,
            },
            Table: {
              headerBg: "#fafafa",
              borderColor: "#f0f0f0",
              rowHoverBg: "#f7f8fa",
            },
          },
        }}
      >
        <App>{children}</App>
      </ConfigProvider>
    </AntdRegistry>
  );
}
