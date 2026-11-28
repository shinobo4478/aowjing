// Smoke test for lib/narration.ts. No test framework is installed, so it runs on
// plain node:   node --experimental-strip-types scripts/narration.smoke.mjs
// Exit code 1 when a case fails.
import { estimateSpeechSeconds, narratorInfo, secondsForAudio, speechChars } from "../lib/narration.ts";

let bad = 0;
const check = (name, ok, detail = "") => {
  if (!ok) bad++;
  console.log(`${ok ? "ok  " : "FAIL"} ${name}${ok ? "" : ` ${detail}`}`);
};

check("white space is not spoken", speechChars("สวัสดี ครับ\nทุกคน") === 15, speechChars("สวัสดี ครับ\nทุกคน"));
check("empty text is 0 s", estimateSpeechSeconds("  ") === 0);
check("110 characters is about 10 s", Math.abs(estimateSpeechSeconds("ก".repeat(110)) - 10) < 1e-9);

const kling = [3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15];
const veo = [4, 6, 8];
const cases = [
  ["4.2 s of audio on Kling needs 5 s (4.7 needed)", kling, 4200, 5, true],
  ["4.5 s exactly: 5.0 needed, 5 fits", kling, 4500, 5, true],
  ["4.6 s: 5.1 needed, so 6", kling, 4600, 6, true],
  ["unsorted options are fine", [8, 4, 6], 5200, 6, true],
  ["Veo only has 4, 6, 8: 5.6 s needs 6.1, so 8", veo, 5600, 8, true],
  ["too long for every option: the longest, and it does not fit", veo, 9000, 8, false],
  ["a very short audio still gets the shortest option", kling, 300, 3, true],
];
for (const [name, options, ms, wantSeconds, wantFits] of cases) {
  const got = secondsForAudio(options, ms);
  check(name, got.seconds === wantSeconds && got.fits === wantFits, JSON.stringify(got));
}

const list = [{ key: "eleven-v3-rachel", label: "ElevenLabs v3 · Rachel", pricePer1kChars: 0.1 }];
check("a listed voice", narratorInfo("eleven-v3-rachel", list)?.label === "ElevenLabs v3 · Rachel");
check("a custom voice is v3 at 0.10", narratorInfo("custom:Alice", list)?.pricePer1kChars === 0.1);
check("none chosen", narratorInfo("", list) === null);
check("unknown key", narratorInfo("eleven-v3-aria", list) === null);
check("custom with no name", narratorInfo("custom:", list) === null);
process.exit(bad ? 1 : 0);
