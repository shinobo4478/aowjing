"use client";

import { Segmented } from "antd";
import { useThemeMode, type ThemePreference } from "./ThemeMode";

const OPTIONS: { value: ThemePreference; label: string }[] = [
  { value: "light", label: "Light" },
  { value: "system", label: "Auto" },
  { value: "dark", label: "Dark" },
];

/** Light / Auto / Dark. Auto follows the operating system. */
export default function ThemeSwitch({ block }: { block?: boolean }) {
  const { preference, setPreference } = useThemeMode();
  return (
    <Segmented<ThemePreference>
      size="small"
      block={block}
      aria-label="Colour theme"
      options={OPTIONS}
      value={preference}
      onChange={setPreference}
    />
  );
}
