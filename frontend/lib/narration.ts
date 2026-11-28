// Helpers for a shot's narration: how long Thai speech takes, and which clip
// length fits an audio. No imports, so it can be run on its own (see
// scripts/narration.smoke.mjs).

/**
 * How many Thai characters a narrator speaks per second. A GUESS, used only to
 * warn before any audio exists: Thai is written without spaces and a syllable is
 * about two characters, and narration runs near five syllables a second. Once a
 * shot has real audio its measured length is used instead; adjust this number
 * after a few real ones.
 */
export const SPEECH_CHARS_PER_SECOND = 11;

/** Characters that are spoken: everything but white space. */
export function speechChars(text: string): number {
  return [...text.replace(/\s+/g, "")].length;
}

/** About how many seconds the narration takes to say. 0 for no text. */
export function estimateSpeechSeconds(text: string): number {
  return speechChars(text) / SPEECH_CHARS_PER_SECOND;
}

/** A little silence after the last word, so a clip does not cut the voice off. */
export const TAIL_SECONDS = 0.5;

/**
 * The shortest clip length (from the lengths the video model takes) that holds an
 * audio of `durationMs` plus a short tail. `fits` is false when even the longest
 * is too short: the longest is returned and the narration should be shortened.
 */
export function secondsForAudio(
  options: number[],
  durationMs: number,
): { seconds: number; fits: boolean } {
  const sorted = [...options].sort((a, b) => a - b);
  const needed = durationMs / 1000 + TAIL_SECONDS;
  const fit = sorted.find((s) => s >= needed);
  if (fit !== undefined) return { seconds: fit, fits: true };
  return { seconds: sorted[sorted.length - 1] ?? 0, fits: false };
}

/** What the narrator voice chosen in Settings is called and costs. A voice the
 *  owner named ("custom:<name>") is ElevenLabs v3, $0.10 per 1000 characters
 *  (mirrors studio.CustomVoice). Null when none is chosen or the key is unknown. */
export function narratorInfo(
  key: string,
  candidates: { key: string; label: string; pricePer1kChars: number }[],
): { label: string; pricePer1kChars: number } | null {
  if (!key) return null;
  const listed = candidates.find((c) => c.key === key);
  if (listed) return { label: listed.label, pricePer1kChars: listed.pricePer1kChars };
  if (key.startsWith("custom:") && key.length > "custom:".length) {
    return { label: `ElevenLabs v3 · ${key.slice("custom:".length)} (your voice)`, pricePer1kChars: 0.1 };
  }
  return null;
}

/**
 * Things in a narration that make the Thai voice read it badly. The voice takes
 * its accent from the first words, and drifts into English sounds on Latin
 * letters; digits may be read in either language. Returns Thai messages, empty
 * when the text is fine. A loanword written in Thai letters (แคลอรี่) cannot be
 * told apart from Thai, so a line that starts with one is not caught: start
 * with Thai words first. Mirrors scriptwriter.narrationProblem on the backend.
 */
export function narrationWarnings(text: string): string[] {
  const t = text.trim();
  if (!t) return [];
  const out: string[] = [];
  const first = [...t][0];
  const isLatin = (c: string) => /[A-Za-z]/.test(c);
  const isDigit = (c: string) => /[0-9๐-๙]/.test(c);
  if (isLatin(first) || isDigit(first)) {
    out.push("ขึ้นต้นด้วยตัวอักษรอังกฤษหรือตัวเลข เสียงอาจอ่านด้วยสำเนียงอังกฤษ ขึ้นต้นด้วยคำไทยก่อนจะชัดกว่า");
  }
  if (!isLatin(first) && [...t].some(isLatin)) {
    out.push("มีตัวอักษรอังกฤษในบท เขียนเป็นตัวไทยตามที่ออกเสียง (เช่น ที ดี อี อี)");
  }
  if (!isDigit(first) && [...t].some(isDigit)) {
    out.push("มีตัวเลขในบท เขียนเป็นคำอ่านไทยจะออกเสียงสม่ำเสมอกว่า (เช่น เจ็ดพันเจ็ดร้อย)");
  }
  return out;
}
