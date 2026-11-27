"use client";

import { AntdRegistry } from "@ant-design/nextjs-registry";
import { App, ConfigProvider, Grid, theme } from "antd";
import { FONT_FAMILY } from "@/lib/theme";
import { PALETTE } from "@/lib/tape";
import { ThemeModeProvider, useThemeMode } from "./ThemeMode";

/**
 * Client-side providers for the whole app.
 *
 * - AntdRegistry: extracts antd's CSS-in-JS on the server so RSC pages render
 *   styled on first paint (no flash).
 * - ThemeModeProvider: light / dark / follow-the-system (app/ThemeMode.tsx).
 * - ConfigProvider: the "slate" theme. Dark is a true-black canvas (OLED) with
 *   warm-chalk ink; light is warm paper with near-black ink. Tungsten is the
 *   only brand colour in both — reserved for the primary action, the current
 *   page and focus. Colour elsewhere means identity or state (see
 *   lib/tape.ts), never decoration.
 *   Radii go by role: pill buttons, 12px inputs, 8px chips, 16px panels.
 * - App: gives components the static message/notification/modal APIs via
 *   `App.useApp()`.
 */
export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <AntdRegistry>
      <ThemeModeProvider>
        <Themed>{children}</Themed>
      </ThemeModeProvider>
    </AntdRegistry>
  );
}

function Themed({ children }: { children: React.ReactNode }) {
  const { mode } = useThemeMode();
  // antd derives shades from these, so they must be literal colours, not CSS
  // variables (those are for the things we draw ourselves; see lib/tape.ts).
  const c = PALETTE[mode];
  const dark = mode === "dark";
  // Phones: 16px text (iOS zooms the page into any smaller input) and 44px
  // controls, the minimum comfortable touch target.
  const phone = Grid.useBreakpoint().md === false;
  const focusRing = dark ? "rgba(255, 201, 51, 0.18)" : "rgba(217, 154, 0, 0.25)";

  return (
    <ConfigProvider
      theme={{
        algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: {
          colorPrimary: c.tungsten,
          colorInfo: c.info,
          colorSuccess: c.succeeded,
          colorWarning: c.tungsten,
          colorError: c.failed,
          colorLink: c.chalk,
          colorLinkHover: c.accent,
          // Text drawn on a tungsten fill (primary buttons) is black.
          colorTextLightSolid: "#000000",

          colorBgLayout: c.void,
          colorBgContainer: c.panel,
          colorBgElevated: c.raised,
          colorBgMask: c.mask,
          colorBorder: c.rule,
          colorBorderSecondary: c.ruleSoft,

          colorText: c.chalk,
          colorTextSecondary: c.ash,
          colorTextTertiary: c.dim,
          colorTextQuaternary: c.faint,
          colorTextPlaceholder: c.dim,

          // Soft all round, to go with the pill buttons: 12 on inputs and
          // selects, 8 on small chips, 16 on panels (cards, modals, menus).
          borderRadius: 12,
          borderRadiusSM: 8,
          borderRadiusLG: 16,
          controlHeight: phone ? 44 : 40,
          fontFamily: FONT_FAMILY,
          fontSize: phone ? 16 : 15,
          lineHeight: 1.65,
          fontWeightStrong: 600,

          // Shadows barely register on true black; keep them deep and short.
          // On paper they are what lifts a modal off the page.
          boxShadow: `0 8px 24px ${c.shadow}`,
          boxShadowSecondary: `0 6px 18px ${c.shadow}`,
        },
        components: {
          Layout: {
            headerBg: c.void,
            headerHeight: 60,
            headerPadding: "0 24px",
            bodyBg: c.void,
          },
          Menu: {
            darkItemBg: "transparent",
            darkItemColor: c.ash,
            darkItemHoverColor: c.chalk,
            darkItemHoverBg: "transparent",
            darkItemSelectedColor: c.accent,
            darkItemSelectedBg: "transparent",
            itemBg: "transparent",
            itemColor: c.ash,
            itemHoverColor: c.chalk,
            itemHoverBg: "transparent",
            itemSelectedColor: c.accent,
            itemSelectedBg: "transparent",
            horizontalItemSelectedColor: c.accent,
            horizontalItemHoverColor: c.chalk,
            activeBarHeight: 2,
            activeBarBorderWidth: 0,
            itemBorderRadius: 2,
            fontSize: 15,
          },
          Tabs: {
            itemColor: c.ash,
            itemHoverColor: c.chalk,
            itemActiveColor: c.chalk,
            itemSelectedColor: c.chalk,
            inkBarColor: dark ? c.tungsten : "#D99A00",
            titleFontSize: 17,
            horizontalItemGutter: 36,
            horizontalMargin: "0 0 20px 0",
          },
          Table: {
            headerBg: "transparent",
            headerColor: c.ash,
            headerSplitColor: "transparent",
            borderColor: c.ruleSoft,
            rowHoverBg: dark ? c.raised : c.inset,
            rowExpandedBg: c.inset,
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
            defaultBorderColor: c.ruleStrong,
            defaultColor: c.chalk,
            // Pills: fully rounded ends at every size.
            borderRadius: 999,
            borderRadiusSM: 999,
            borderRadiusLG: 999,
            paddingInline: 22,
            paddingInlineSM: 16,
          },
          Input: {
            activeBorderColor: dark ? c.tungsten : "#D99A00",
            hoverBorderColor: c.ruleHover,
            activeShadow: `0 0 0 2px ${focusRing}`,
            colorBgContainer: c.inset,
          },
          Select: {
            activeBorderColor: dark ? c.tungsten : "#D99A00",
            hoverBorderColor: c.ruleHover,
            activeOutlineColor: focusRing,
            colorBgContainer: c.inset,
          },
          Modal: {
            contentBg: c.raised,
            headerBg: c.raised,
            titleFontSize: 19,
          },
          // The sun/moon switch is a pill, like the buttons.
          Segmented: {
            borderRadius: 999,
            borderRadiusSM: 999,
            borderRadiusLG: 999,
          },
          Tag: {
            defaultBg: "transparent",
            defaultColor: c.chalk,
          },
          // Room for the error message and the helper text to stack without
          // touching the next field's label.
          Form: {
            itemMarginBottom: 28,
          },
          Breadcrumb: {
            linkColor: c.ash,
            linkHoverColor: c.chalk,
            lastItemColor: c.chalk,
            separatorColor: c.dim,
            itemColor: c.ash,
          },
          Drawer: {
            colorBgElevated: c.panel,
          },
          Empty: {
            colorTextDescription: c.ash,
          },
          Popover: { colorBgElevated: c.raised },
        },
      }}
    >
      <App>{children}</App>
    </ConfigProvider>
  );
}
