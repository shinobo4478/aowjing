"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
} from "react";
import type { ThemeMode } from "@/lib/tape";
import { THEME_STORAGE_KEY } from "@/lib/themeScript";

/** What the person chose. "system" follows the OS and changes with it. */
export type ThemePreference = ThemeMode | "system";

function readPreference(): ThemePreference {
  try {
    const p = localStorage.getItem(THEME_STORAGE_KEY);
    if (p === "light" || p === "dark") return p;
  } catch {
    // Storage can be blocked (private mode); fall through to "system".
  }
  return "system";
}

function resolveMode(p: ThemePreference): ThemeMode {
  if (p !== "system") return p;
  return window.matchMedia("(prefers-color-scheme: light)").matches
    ? "light"
    : "dark";
}

type Ctx = {
  /** The mode actually on screen. */
  mode: ThemeMode;
  preference: ThemePreference;
  setPreference: (p: ThemePreference) => void;
};

const ThemeCtx = createContext<Ctx>({
  mode: "dark",
  preference: "system",
  setPreference: () => {},
});

export function useThemeMode(): Ctx {
  return useContext(ThemeCtx);
}

export function ThemeModeProvider({ children }: { children: React.ReactNode }) {
  // The server cannot know the person's choice, so it renders dark (the CSS
  // default). The real value is read right after hydration, before paint.
  const [preference, setPreferenceState] = useState<ThemePreference>("system");
  const [mode, setMode] = useState<ThemeMode>("dark");

  useLayoutEffect(() => {
    // Reading browser-only state (localStorage / matchMedia) can only happen
    // after hydration; doing it here, not in useEffect, avoids a visible flash.
    const p = readPreference();
    setPreferenceState(p); // eslint-disable-line react-hooks/set-state-in-effect
    setMode(resolveMode(p));
  }, []);

  // Keep <html data-theme> in step. React's dev Strict Mode remount can clear
  // the attribute the head script set, so this also re-applies it.
  useLayoutEffect(() => {
    document.documentElement.setAttribute("data-theme", mode);
  }, [mode]);

  // While on "system", follow the OS when it flips (e.g. sunset auto-switch).
  useEffect(() => {
    if (preference !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: light)");
    const onChange = () => setMode(mq.matches ? "light" : "dark");
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [preference]);

  const setPreference = useCallback((p: ThemePreference) => {
    try {
      if (p === "system") localStorage.removeItem(THEME_STORAGE_KEY);
      else localStorage.setItem(THEME_STORAGE_KEY, p);
    } catch {
      // Not persisted, still applied for this visit.
    }
    setPreferenceState(p);
    setMode(resolveMode(p));
  }, []);

  const value = useMemo(
    () => ({ mode, preference, setPreference }),
    [mode, preference, setPreference],
  );
  return <ThemeCtx.Provider value={value}>{children}</ThemeCtx.Provider>;
}
