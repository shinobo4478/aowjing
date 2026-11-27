"use client";

import { useCallback, useState } from "react";
import { translatePrompt, type PromptKind } from "./api";

/** True when s holds Thai letters. An English box with Thai in it is a mistake:
 *  the models are only promised to understand English. */
export const hasThaiLetters = (s: string) => /[฀-๿]/.test(s);

/**
 * A text that was typed on screen on top of a saved value. The saved value can
 * also change elsewhere (a dialog saves a prompt, a translation lands), so what
 * was typed only counts while the saved value is still the one it was typed
 * over; otherwise the saved value shows.
 */
function useTyped(saved: string): [string, (text: string) => void] {
  const [typed, setTyped] = useState<{ text: string; base: string } | null>(null);
  const value = typed && typed.base === saved ? typed.text : saved;
  return [value, (text) => setTyped({ text, base: saved })];
}

/** The state of one prompt that exists twice: the Thai the owner edits and the
 *  English the models receive. */
export interface PromptPair {
  th: string;
  en: string;
  setTh: (text: string) => void;
  setEn: (text: string) => void;
  /** The English is missing, or the Thai was changed after the English was made
   *  from it, so the English no longer says what the Thai says. */
  stale: boolean;
  /** A translation is running. */
  busy: boolean;
  /** What the last translation cost, if there was one. */
  lastCostUsd: number | null;
  /** Translates the Thai into English now; returns the English. */
  toEnglish: () => Promise<string>;
  /** Translates the English into Thai now; returns the Thai. */
  toThai: () => Promise<string>;
  /** The English to use: translated first if the Thai changed since. Rejects if
   *  the translation fails. */
  ensureEnglish: () => Promise<string>;
}

/**
 * One prompt as a Thai/English pair. `savedTh` and `savedEn` are the stored
 * values; typing and translating change what is shown until the next save.
 *
 * The English counts as made from the Thai when the page loads (a shot saved by
 * this screen always has its English made from its Thai); it becomes stale as
 * soon as the Thai is edited, and is brought up to date by ensureEnglish before
 * anything is saved or paid for.
 */
export function usePromptPair(kind: PromptKind, savedTh: string, savedEn: string): PromptPair {
  const [th, setTh] = useTyped(savedTh);
  const [en, setEn] = useTyped(savedEn);
  // The Thai the English was last made from. Like the typed texts it only counts
  // while the saved Thai is the one it was recorded over.
  const [made, setMade] = useState<{ text: string; base: string } | null>(null);
  const madeFrom = made && made.base === savedTh ? made.text : savedTh;
  const [busy, setBusy] = useState(false);
  const [lastCostUsd, setLastCostUsd] = useState<number | null>(null);

  // The English is out of date (the Thai was edited since) or missing.
  const stale = th.trim() !== "" && (th.trim() !== madeFrom.trim() || en.trim() === "");

  const toEnglish = useCallback(async () => {
    setBusy(true);
    try {
      const r = await translatePrompt({ kind, direction: "toEnglish", text: th });
      setEn(r.text);
      setMade({ text: th, base: savedTh });
      setLastCostUsd(r.costUsd);
      return r.text;
    } finally {
      setBusy(false);
    }
    // setEn is recreated on each render but only closes over `savedEn`.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, th, savedTh, savedEn]);

  const toThai = useCallback(async () => {
    setBusy(true);
    try {
      const r = await translatePrompt({ kind, direction: "toThai", text: en });
      setTh(r.text);
      // The English is what this Thai was made from, so the pair is in step.
      setMade({ text: r.text, base: savedTh });
      setLastCostUsd(r.costUsd);
      return r.text;
    } finally {
      setBusy(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, en, savedTh]);

  const ensureEnglish = useCallback(async () => {
    if (stale) return toEnglish();
    return en;
  }, [stale, toEnglish, en]);

  return { th, en, setTh, setEn, stale, busy, lastCostUsd, toEnglish, toThai, ensureEnglish };
}

/** A prompt as the two texts it is kept in: the Thai the owner edits and the
 *  English the models receive. */
export interface PromptTexts {
  english: string;
  thai: string;
}
