"use client";

import { AntdRegistry } from "@ant-design/nextjs-registry";
import { App, ConfigProvider, theme } from "antd";
import { FONT_FAMILY } from "@/lib/theme";

/**
 * Client-side providers for the whole app.
 *
 * - AntdRegistry: extracts antd's CSS-in-JS on the server so RSC pages render
 *   styled on first paint (no flash).
 * - ConfigProvider: a refined OLED dark theme. The page canvas is true black
 *   (`#000` — zero-emission pixels on OLED, least eye strain at night), and
 *   surfaces lift off it in small, deliberate steps: containers ~#0d, elevated
 *   layers (modals, dropdowns) ~#17. Text is never pure white — the brightest
 *   ink is white at 88% so long reading sessions stay comfortable, and
 *   secondary text drops to 55% for a calm hierarchy. The accent is a soft
 *   periwinkle that reads clearly on black without glare.
 * - App: gives components the static message/notification/modal APIs via
 *   `App.useApp()`.
 */

// One place to tune the darkness ramp. Each step is a small lift so depth
// reads as depth, not as a hard edge.
const surface = {
  canvas: "#000000", // Layout background — true black for OLED
  container: "#0d0d0f", // Cards, tables, inputs
  containerAlt: "#111114", // Table header, inset panels
  elevated: "#171719", // Modals, dropdowns, popovers, tooltips
  borderStrong: "#2a2a30", // Visible dividers, input outlines
  borderSubtle: "#1e1e22", // Hairlines between rows
};

const ink = {
  primary: "rgba(255, 255, 255, 0.88)", // headings, body — softer than #fff
  secondary: "rgba(255, 255, 255, 0.55)", // captions, metadata — reduced contrast
  tertiary: "rgba(255, 255, 255, 0.38)", // placeholders, disabled hints
  quaternary: "rgba(255, 255, 255, 0.24)",
};

// Soft periwinkle. Bright enough to guide the eye on true black, never a glare.
const accent = "#818cf8";

export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AntdRegistry>
      <ConfigProvider
        theme={{
          algorithm: theme.darkAlgorithm,
          token: {
            colorPrimary: accent,
            colorInfo: accent,
            colorLink: accent,

            colorBgLayout: surface.canvas,
            colorBgContainer: surface.container,
            colorBgElevated: surface.elevated,
            // Masks over true black want to be lighter, not darker.
            colorBgMask: "rgba(0, 0, 0, 0.65)",

            colorBorder: surface.borderStrong,
            colorBorderSecondary: surface.borderSubtle,

            colorText: ink.primary,
            colorTextSecondary: ink.secondary,
            colorTextTertiary: ink.tertiary,
            colorTextQuaternary: ink.quaternary,

            borderRadius: 8,
            borderRadiusLG: 12,
            controlHeight: 36,
            fontFamily: FONT_FAMILY,

            // Shadows tuned for a dark canvas: deep, soft, no glow.
            boxShadow:
              "0 6px 16px rgba(0, 0, 0, 0.55), 0 3px 6px rgba(0, 0, 0, 0.5)",
            boxShadowSecondary:
              "0 4px 12px rgba(0, 0, 0, 0.5), 0 2px 4px rgba(0, 0, 0, 0.45)",
          },
          components: {
            Layout: {
              headerBg: "#0a0a0b",
              bodyBg: surface.canvas,
              headerHeight: 56,
            },
            Menu: {
              itemBg: "transparent",
              activeBarHeight: 2,
              itemHoverBg: "rgba(255, 255, 255, 0.05)",
              itemSelectedBg: "rgba(129, 140, 248, 0.14)",
              itemSelectedColor: accent,
            },
            Card: {
              paddingLG: 24,
              colorBgContainer: surface.container,
            },
            Table: {
              headerBg: surface.containerAlt,
              headerColor: ink.secondary,
              borderColor: surface.borderSubtle,
              rowHoverBg: "rgba(255, 255, 255, 0.035)",
              colorBgContainer: surface.container,
            },
            Modal: {
              contentBg: surface.elevated,
              headerBg: surface.elevated,
            },
            Input: {
              colorBgContainer: surface.containerAlt,
              activeShadow: "0 0 0 2px rgba(129, 140, 248, 0.2)",
            },
            Button: {
              defaultBg: surface.containerAlt,
              defaultBorderColor: surface.borderStrong,
            },
            Tag: {
              defaultBg: "rgba(255, 255, 255, 0.06)",
            },
          },
        }}
      >
        <App>{children}</App>
      </ConfigProvider>
    </AntdRegistry>
  );
}
