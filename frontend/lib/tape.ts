// The app's colour system, in one place.
//
// Colour carries meaning, never decoration:
//   - TUNGSTEN marks the primary action, the current page and keyboard focus.
//   - A profile's TAPE colour is its identity — like the spike tape a crew
//     lays on a stage floor to mark a position. It is derived from the
//     profile id, so a profile keeps the same colour everywhere, forever.
//   - STATUS colours are the tally lights of a generation.

export type ThemeMode = "light" | "dark";

/**
 * The real colour values per mode. antd's theme tokens need literal colours
 * (its algorithms derive shades from them), so providers.tsx reads these. The
 * same values are published as CSS variables in app/globals.css, which is what
 * everything we draw ourselves uses (INK / SURFACE / STATUS below), so a mode
 * switch repaints our own elements without a re-render.
 *
 * Keep the two in step: a change here needs the matching line in globals.css.
 */
export const PALETTE = {
  dark: {
    tungsten: "#FFC933",
    // Tungsten used as text or a thin line on the canvas.
    accent: "#FFC933",
    chalk: "#ECEAE4",
    ash: "#9B988F",
    dim: "#6E6B64",
    faint: "#4A4843",
    void: "#000000",
    panel: "#0D0D0F",
    raised: "#151518",
    inset: "#0A0A0B",
    rule: "#2A2A2F",
    ruleSoft: "#1C1C20",
    ruleStrong: "#3A3A40",
    ruleHover: "#55555C",
    succeeded: "#4FE3A8",
    failed: "#FF5D5D",
    info: "#46D2F2",
    mask: "rgba(0, 0, 0, 0.7)",
    shadow: "rgba(0, 0, 0, 0.6)",
  },
  light: {
    tungsten: "#FFC933",
    // Bright yellow text is unreadable on paper; a deep amber is not.
    accent: "#8A5A00",
    chalk: "#1B1A17",
    ash: "#5E5B53",
    dim: "#8A867C",
    faint: "#B5B1A6",
    void: "#F4F2ED",
    panel: "#FFFFFF",
    raised: "#FFFFFF",
    inset: "#FAF9F6",
    rule: "#D9D5CB",
    ruleSoft: "#E8E4DA",
    ruleStrong: "#C4BFB2",
    ruleHover: "#9C978A",
    succeeded: "#0E8A5A",
    failed: "#D12F2F",
    info: "#0B7C99",
    mask: "rgba(20, 18, 14, 0.45)",
    shadow: "rgba(40, 34, 20, 0.18)",
  },
} as const satisfies Record<ThemeMode, Record<string, string>>;

// Everything below is a CSS variable reference, safe to use in inline styles
// and in either mode. (Defined in app/globals.css.)
export const TUNGSTEN = "var(--acmp-tungsten)";

export const INK = {
  chalk: "var(--acmp-chalk)",
  ash: "var(--acmp-ash)",
  dim: "var(--acmp-dim)",
} as const;

export const SURFACE = {
  void: "var(--acmp-void)",
  panel: "var(--acmp-panel)",
  raised: "var(--acmp-raised)",
  rule: "var(--acmp-rule)",
  ruleSoft: "var(--acmp-rule-soft)",
} as const;

export const STATUS = {
  pending: "var(--acmp-tungsten)",
  succeeded: "var(--acmp-succeeded)",
  failed: "var(--acmp-failed)",
} as const;

const TAPE = [
  "#46D2F2",
  "#FF9248",
  "#A98BFF",
  "#4FE3A8",
  "#F07BC8",
  "#6F8CFF",
] as const;

/** Stable tape colour for any string key (usually a profile id). */
export function tapeColor(key: string): string {
  let h = 5381;
  for (let i = 0; i < key.length; i++) h = ((h << 5) + h + key.charCodeAt(i)) | 0;
  return TAPE[Math.abs(h) % TAPE.length];
}
