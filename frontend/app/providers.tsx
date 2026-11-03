"use client";

import { AntdRegistry } from "@ant-design/nextjs-registry";
import { App, ConfigProvider, theme } from "antd";
import { FONT_FAMILY } from "@/lib/theme";
import { INK, STATUS, SURFACE, TUNGSTEN } from "@/lib/tape";

/**
 * Client-side providers for the whole app.
 *
 * - AntdRegistry: extracts antd's CSS-in-JS on the server so RSC pages render
 *   styled on first paint (no flash).
 * - ConfigProvider: the "slate" theme. True-black canvas (OLED), warm-chalk
 *   ink, and tungsten as the only brand colour — reserved for the primary
 *   action, the current page and focus. Colour elsewhere means identity or
 *   state (see lib/tape.ts), never decoration.
 *   Radii differ by role on purpose: 2px structure, 4px controls, 6px overlays.
 * - App: gives components the static message/notification/modal APIs via
 *   `App.useApp()`.
 */
export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AntdRegistry>
      <ConfigProvider
        theme={{
          algorithm: theme.darkAlgorithm,
          token: {
            colorPrimary: TUNGSTEN,
            colorInfo: "#46D2F2",
            colorSuccess: STATUS.succeeded,
            colorWarning: TUNGSTEN,
            colorError: STATUS.failed,
            colorLink: INK.chalk,
            colorLinkHover: TUNGSTEN,
            // Text drawn on a tungsten fill (primary buttons) is black.
            colorTextLightSolid: "#000000",

            colorBgLayout: SURFACE.void,
            colorBgContainer: SURFACE.panel,
            colorBgElevated: SURFACE.raised,
            colorBgMask: "rgba(0, 0, 0, 0.7)",
            colorBorder: SURFACE.rule,
            colorBorderSecondary: SURFACE.ruleSoft,

            colorText: INK.chalk,
            colorTextSecondary: INK.ash,
            colorTextTertiary: INK.dim,
            colorTextQuaternary: "#4A4843",
            colorTextPlaceholder: INK.dim,

            borderRadius: 4,
            borderRadiusSM: 2,
            borderRadiusLG: 6,
            controlHeight: 40,
            fontFamily: FONT_FAMILY,
            fontSize: 15,
            lineHeight: 1.65,
            fontWeightStrong: 600,

            // Shadows barely register on true black; keep them deep and short.
            boxShadow: "0 8px 24px rgba(0, 0, 0, 0.6)",
            boxShadowSecondary: "0 6px 18px rgba(0, 0, 0, 0.6)",
          },
          components: {
            Layout: {
              headerBg: SURFACE.void,
              headerHeight: 60,
              headerPadding: "0 24px",
              bodyBg: SURFACE.void,
            },
            Menu: {
              darkItemBg: "transparent",
              darkItemColor: INK.ash,
              darkItemHoverColor: INK.chalk,
              darkItemHoverBg: "transparent",
              darkItemSelectedColor: TUNGSTEN,
              darkItemSelectedBg: "transparent",
              horizontalItemSelectedColor: TUNGSTEN,
              horizontalItemHoverColor: INK.chalk,
              activeBarHeight: 2,
              activeBarBorderWidth: 0,
              itemBorderRadius: 2,
              fontSize: 15,
            },
            Tabs: {
              itemColor: INK.ash,
              itemHoverColor: INK.chalk,
              itemActiveColor: INK.chalk,
              itemSelectedColor: INK.chalk,
              inkBarColor: TUNGSTEN,
              titleFontSize: 17,
              horizontalItemGutter: 36,
              horizontalMargin: "0 0 20px 0",
            },
            Table: {
              headerBg: "transparent",
              headerColor: INK.ash,
              headerSplitColor: "transparent",
              borderColor: SURFACE.ruleSoft,
              rowHoverBg: SURFACE.raised,
              rowExpandedBg: "#0A0A0B",
              cellPaddingBlock: 16,
              cellPaddingInline: 16,
              colorBgContainer: "transparent",
            },
            Button: {
              fontWeight: 600,
              primaryShadow: "none",
              defaultShadow: "none",
              dangerShadow: "none",
              defaultBg: "transparent",
              defaultBorderColor: "#3A3A40",
              defaultColor: INK.chalk,
            },
            Input: {
              activeBorderColor: TUNGSTEN,
              hoverBorderColor: "#55555C",
              activeShadow: "0 0 0 2px rgba(255, 201, 51, 0.18)",
              colorBgContainer: "#0A0A0B",
            },
            Select: {
              activeBorderColor: TUNGSTEN,
              hoverBorderColor: "#55555C",
              activeOutlineColor: "rgba(255, 201, 51, 0.18)",
              colorBgContainer: "#0A0A0B",
            },
            Modal: {
              contentBg: SURFACE.raised,
              headerBg: SURFACE.raised,
              titleFontSize: 19,
            },
            Tag: {
              defaultBg: "transparent",
              defaultColor: INK.chalk,
            },
            // Room for the error message and the helper text to stack without
            // touching the next field's label.
            Form: {
              itemMarginBottom: 28,
            },
            Breadcrumb: {
              linkColor: INK.ash,
              linkHoverColor: INK.chalk,
              lastItemColor: INK.chalk,
              separatorColor: INK.dim,
              itemColor: INK.ash,
            },
            Drawer: {
              colorBgElevated: SURFACE.panel,
            },
            Empty: {
              colorTextDescription: INK.ash,
            },
            Popover: { colorBgElevated: SURFACE.raised },
          },
        }}
      >
        <App>{children}</App>
      </ConfigProvider>
    </AntdRegistry>
  );
}
