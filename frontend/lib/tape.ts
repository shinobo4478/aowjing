// The app's colour system, in one place.
//
// Colour carries meaning, never decoration:
//   - TUNGSTEN marks the primary action, the current page and keyboard focus.
//   - A profile's TAPE colour is its identity — like the spike tape a crew
//     lays on a stage floor to mark a position. It is derived from the
//     profile id, so a profile keeps the same colour everywhere, forever.
//   - STATUS colours are the tally lights of a generation.

export const TUNGSTEN = "#FFC933";

export const INK = {
  chalk: "#ECEAE4",
  ash: "#9B988F",
  dim: "#6E6B64",
} as const;

export const SURFACE = {
  void: "#000000",
  panel: "#0D0D0F",
  raised: "#151518",
  rule: "#2A2A2F",
  ruleSoft: "#1C1C20",
} as const;

export const STATUS = {
  pending: TUNGSTEN,
  succeeded: "#4FE3A8",
  failed: "#FF5D5D",
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
