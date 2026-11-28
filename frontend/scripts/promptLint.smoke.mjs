// Smoke test for lib/promptLint.ts, using the real texts of the owner's first
// short-film draft. No test framework is installed, so it runs on plain node:
//   node --experimental-strip-types scripts/promptLint.smoke.mjs
// Exit code 1 when a case fails.
import { lintPrompt } from "../lib/promptLint.ts";


const cases = [
  { name: "owner shot 1 motion (costar)", kind: "motion",
    text: "The costar tilts the bottle up, drinks slowly, then lowers it back down, settling into the same relaxed pose. Static camera, no zoom.",
    want: ["motion-role-word"], fixed: "The person tilts the bottle up, drinks slowly, then lowers it back down, settling into the same relaxed pose. Static camera, no zoom." },
  { name: "owner shot 2 motion (costar, static + zoom, reveal)", kind: "motion",
    text: "The costar keeps drinking from the new bottle without reacting to the empty bottles beside his feet, while the camera slowly pulls back a little to reveal more of the floor. Static camera, slight slow zoom out.",
    want: ["motion-role-word", "motion-camera-conflict", "motion-reveal"] },
  { name: "owner shot 3 motion (lead character)", kind: "motion",
    text: "The lead character drinks from the green bottle, lowers it, and looks straight at the camera with a refreshed, satisfied expression. Static camera, no zoom.",
    want: ["motion-role-word"], fixed: "The person drinks from the green bottle, lowers it, and looks straight at the camera with a refreshed, satisfied expression. Static camera, no zoom." },
  { name: "clean motion", kind: "motion",
    text: "The person on the bench raises the clear plastic bottle, drinks slowly, then lowers it. Static camera, no zoom.", want: [] },
  { name: "mid-sentence role word", kind: "motion", text: "She hands the bottle to the hero. Static camera, no zoom.", want: ["motion-role-word"], fixed: "She hands the bottle to the person. Static camera, no zoom." },
  { name: "owner shot 1 picture (framing in scene + negative)", kind: "picture",
    text: "A wide view of an indoor badminton court showing the full court, green wooden flooring. A medium-close shot shows the uncle badminton player sitting on the bench, with nothing visible around his feet yet.",
    want: ["picture-scene-framing", "picture-negative"] },
  { name: "fixed picture", kind: "picture",
    text: "An indoor badminton court with a green wooden floor. Medium-close shot, framed from the chest up, of the other character sitting upright on the bench, holding a clear plastic bottle at chest height.", want: [] },
  { name: "no text is allowed", kind: "picture", text: "A sign with no text on the wall; the lead character smiles.", want: [] },
  { name: "end picture without Only", kind: "endPicture", text: "The cat crouches low.", want: ["end-only"] },
  { name: "end picture ok", kind: "endPicture", text: "Only the cat's pose changes: it crouches low.", want: [] },
  { name: "empty", kind: "picture", text: "   ", want: [] },
];

let bad = 0;
for (const c of cases) {
  const issues = lintPrompt(c.kind, c.text);
  const got = issues.map((i) => i.id).sort();
  const want = [...c.want].sort();
  let ok = JSON.stringify(got) === JSON.stringify(want);
  let extra = "";
  if (ok && c.fixed !== undefined) {
    const fix = issues.find((i) => i.fix)?.fix;
    const out = fix ? fix(c.text) : "(no fix)";
    if (out !== c.fixed) { ok = false; extra = `\n   fix gave: ${out}`; }
  }
  if (!ok) bad++;
  console.log(`${ok ? "ok  " : "FAIL"} ${c.name}${ok ? "" : ` got=${got} want=${want}${extra}`}`);
}
process.exit(bad ? 1 : 0);
