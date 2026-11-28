// Checks on the ENGLISH of a shot's prompts: the mistakes the first real
// short-film draft made (CLAUDE.md, "First short-film test"). They run on every
// prompt as it is typed, so they catch a draft's slips and the owner's own edits.
// A check only warns; nothing is blocked. No imports, so it can be run on its own.

/** What a prompt is for; mirrors PromptKind in lib/api.ts. */
export type LintKind = "motion" | "picture" | "endPicture";

export interface LintIssue {
  /** Stable key for the list. */
  id: string;
  /** What is wrong and why, in Thai (the owner's language). */
  message: string;
  /** Offered when the problem has one safe automatic fix. */
  fix?: (text: string) => string;
  fixLabel?: string;
}

// Role words are the studio's own labels for cast members. The clip model sees
// only the starting picture and the text, so it cannot know what they mean.
const MOTION_ROLE_WORDS = /\b(co-?stars?|heroe?s?|lead characters?|main characters?|other characters?)\b/gi;
// In a picture prompt "the lead character" / "the other character" are the words
// the picture step uses when it names the reference pictures; the others are not.
const PICTURE_ROLE_WORDS = /\b(co-?stars?|heroe?s?)\b/gi;

const CAMERA_MOVES = /\b(zoom(?:s|ing)?\s+(?:in|out)|pulls?\s+back|pull-back|pulling\s+back|pans?|dolly|dollies|tracking|tilts?\s+(?:up|down))\b/i;

/** Replaces a role word, with the article before it, by "the person"; keeps the
 *  capital when it starts a sentence. */
function plainNoun(text: string): string {
  const withArticle = new RegExp(`\\b((?:the|a)\\s+)?(?:${MOTION_ROLE_WORDS.source})`, "gi");
  return text.replace(withArticle, (match: string, article: string | undefined, ...rest: unknown[]) => {
    const offset = rest.find((r) => typeof r === "number") as number;
    const whole = rest.find((r) => typeof r === "string") as string;
    const startsSentence = offset === 0 || /[.!?]\s*$/.test(whole.slice(0, offset));
    const capital = article ? article[0] === article[0].toUpperCase() : startsSentence;
    return capital ? "The person" : "the person";
  });
}

function lintMotion(text: string): LintIssue[] {
  const out: LintIssue[] = [];
  if (new RegExp(MOTION_ROLE_WORDS.source, "i").test(text)) {
    out.push({
      id: "motion-role-word",
      message:
        "ใช้คำบทบาท (costar, hero, lead/main/other character) โมเดลวิดีโอเห็นแค่ภาพต้นช็อตกับข้อความนี้ ไม่รู้ว่าหมายถึงใคร ใช้คำสามัญ เช่น “the person”, “the player”",
      fix: plainNoun,
      fixLabel: "แทนด้วย “the person”",
    });
  }
  // "Static camera" together with a camera movement says two things at once.
  if (/static camera/i.test(text) && CAMERA_MOVES.test(text.replace(/no zoom/gi, ""))) {
    out.push({
      id: "motion-camera-conflict",
      message:
        "คำสั่งกล้องขัดกัน: มีทั้ง “Static camera” และการเคลื่อนกล้อง (ซูม/ถอย/แพน) เลือกอย่างเดียว ปิดท้ายด้วย “Static camera, no zoom.” หรือบอกการเคลื่อนกล้องช้าๆ แค่ครั้งเดียว",
    });
  }
  if (/\breveal/i.test(text)) {
    out.push({
      id: "motion-reveal",
      message:
        "“reveal” สิ่งที่ไม่อยู่ในภาพต้นช็อต โมเดลต้องวาดเอง ให้ช็อตถัดไปเป็นภาพกว้างกว่าแล้วตัดไปแทน",
    });
  }
  return out;
}

function lintPicture(text: string, kind: LintKind): LintIssue[] {
  const out: LintIssue[] = [];
  if (new RegExp(PICTURE_ROLE_WORDS.source, "i").test(text)) {
    out.push({
      id: "picture-role-word",
      message:
        "ใช้คำบทบาท “costar/hero” ซึ่งระบบไม่ได้ใช้ตอนระบุรูปอ้างอิง ใช้ “the lead character” (พระเอก) หรือ “the other character” (ตัวประกอบ) หรือคำสามัญ",
    });
  }
  // The scene line is copied in front of every picture prompt; it must name a
  // place, not a camera framing. "showing the full ..." / "A wide view" give it away.
  if (kind === "picture" && /^\s*(?:an?\s+)?(?:wide|full)\s+(?:view|shot)\b|\bshowing the full\b/i.test(text)) {
    out.push({
      id: "picture-scene-framing",
      message:
        "ประโยคสถานที่ที่นำหน้ามีกรอบภาพ (“A wide view…”, “showing the full…”) ซึ่งขัดกับกรอบของช็อตนี้ ให้ประโยคสถานที่บอกแค่สถานที่ ไม่บอกกรอบภาพ (ประโยคนี้ถูกคัดลอกไปทุกช็อต ต้องแก้ทุกช็อต)",
    });
  }
  // Image models draw what is named: "nothing visible around his feet" invites it.
  const negative = text.match(/\b(nothing|without|no one|nobody|empty-handed|absent|not visible|none)\b|\bno\s+(?!text|logo|writing|sign)\w+/i);
  if (negative) {
    out.push({
      id: "picture-negative",
      message: `มีคำปฏิเสธ (“${negative[0]}”) โมเดลภาพมักวาดสิ่งที่ถูกเอ่ยถึง ถ้าไม่อยากให้เห็นอะไร ให้ตัดกรอบออก (เช่น “framed from the chest up”) หรือไม่ต้องเอ่ยถึง`,
    });
  }
  return out;
}

/**
 * The warnings for one English prompt. `kind` says which rules apply. An empty
 * text has none (the box is just empty).
 */
export function lintPrompt(kind: LintKind, english: string): LintIssue[] {
  const text = english.trim();
  if (text === "") return [];
  const issues = kind === "motion" ? lintMotion(text) : lintPicture(text, kind);
  if (kind === "endPicture" && !/^only\b/i.test(text)) {
    issues.push({
      id: "end-only",
      message:
        "ภาพจบควรขึ้นต้นด้วย “Only the …” และบอกเฉพาะสิ่งที่เปลี่ยน (ท่าหรือตำแหน่ง) เพราะคลิปจะบิดทุกอย่างที่ต่างกันระหว่างภาพเริ่มกับภาพจบ",
    });
  }
  return issues;
}
