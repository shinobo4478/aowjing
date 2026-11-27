"use client";

import { Segmented } from "antd";
import { useThemeMode } from "./ThemeMode";

// Hand-drawn 16px icons (stroke = currentColor), so there is no icon package
// to add. Sun: a disc with eight rays. Moon: a crescent.
function SunIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      aria-hidden
      style={{ display: "block" }}
    >
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      style={{ display: "block" }}
    >
      <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
    </svg>
  );
}

const OPTIONS = [
  {
    value: "light",
    label: <SunIcon />,
    title: "Light mode",
    "aria-label": "Light mode",
  },
  {
    value: "dark",
    label: <MoonIcon />,
    title: "Dark mode",
    "aria-label": "Dark mode",
  },
] as const;

/**
 * Sun / moon switch. It shows the mode that is on screen; tapping the other
 * icon sets it. Until the person picks one, the app follows the operating
 * system (see ThemeMode), and the icon for the mode in use is highlighted.
 */
export default function ThemeSwitch({ block }: { block?: boolean }) {
  const { mode, setPreference } = useThemeMode();
  // The wrapper resets line-height: inside the 60px-high header the inline
  // switch would otherwise sit on a 60px text baseline, off-centre.
  return (
    <div style={{ display: "flex", alignItems: "center", lineHeight: 1, width: block ? "100%" : undefined }}>
      <Segmented
        size="small"
        block={block}
        aria-label="Colour theme"
        options={[...OPTIONS]}
        value={mode}
        onChange={(v) => setPreference(v as "light" | "dark")}
      />
    </div>
  );
}
