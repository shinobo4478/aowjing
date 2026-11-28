"use client";

import { useState, type CSSProperties, type ReactNode } from "react";
import { App, Button, Card, Input, Modal, Popconfirm, Space, Typography } from "antd";
import SecondsField from "@/components/SecondsField";
import Tally from "@/components/Tally";
import { imageSrc, reviseShot, shotVoiceSrc } from "@/lib/api";
import { mobileModal } from "@/lib/ui";
import { formatUsd } from "@/lib/format";
import { estimateSpeechSeconds, narrationWarnings, secondsForAudio, speechChars } from "@/lib/narration";
import { hasThaiLetters, usePromptPair, useTyped } from "@/lib/promptPair";
import type { Shot } from "@/lib/types";
import PromptPairField from "./PromptPairField";
import ShotPlayer from "./ShotPlayer";
import { ZoomableImage } from "@/components/ZoomableImage";

/** The part of a shot's plan typed on the card. The picture is chosen in its
 *  own dialog and saved at once. Each prompt is a Thai/English pair: the owner
 *  edits the Thai; the English is what the models receive (translated from the
 *  Thai before anything is saved or paid for). */
export interface ShotDraft {
  /** The clip motion prompt (English) and its Thai. */
  prompt: string;
  promptTh: string;
  /** What the starting picture should show (Make with AI starts from it). */
  picturePrompt: string;
  picturePromptTh: string;
  /** What changes by the END picture (a pose or position only). */
  endPicturePrompt: string;
  endPicturePromptTh: string;
  /** What the narrator says over this shot (Thai). Never sent to a model. */
  narrationTh: string;
  seconds: number;
}

/**
 * One shot of a video: its prompt, length and starting picture (the plan), and
 * the newest attempt at making it. Prompt and length are edited here and saved
 * with the Save button, or automatically when Generate is pressed.
 */
export default function ShotCard({
  shot,
  count,
  perSecond,
  secondsOptions,
  busy,
  onSave,
  onChoosePicture,
  onRemovePicture,
  hasPreviousEnd,
  onStartFromPreviousEnd,
  onOpenTakes,
  canUseNextAsEnd,
  onChooseEnd,
  onUseNextAsEnd,
  onRemoveEnd,
  narrator,
  onMakeVoice,
  onRemoveVoice,
  onGenerate,
  onMove,
  onDelete,
}: {
  shot: Shot;
  /** How many shots the video has, to disable moving the first/last. */
  count: number;
  /** Price per second for this video's style; undefined = not known. */
  perSecond: number | undefined;
  /** The clip lengths the profile's video model takes. */
  secondsOptions: number[];
  /** Something is being saved or queued for this video. */
  busy: boolean;
  onSave: (draft: ShotDraft) => Promise<void>;
  onChoosePicture: (draft: ShotDraft) => void;
  onRemovePicture: (draft: ShotDraft) => Promise<void>;
  /** The previous shot has an end picture that can be this shot's start (so a
   *  shot begins on the pose the one before it ends on). */
  hasPreviousEnd: boolean;
  onStartFromPreviousEnd: (draft: ShotDraft) => void;
  /** Opens the dialog to compare this shot's takes and pick one. */
  onOpenTakes: () => void;
  /** The next shot has a stored starting picture that can be this one's end. */
  canUseNextAsEnd: boolean;
  onChooseEnd: (draft: ShotDraft) => void;
  onUseNextAsEnd: (draft: ShotDraft) => void;
  onRemoveEnd: (draft: ShotDraft) => Promise<void>;
  /** The narrator voice chosen in Settings (null when none is) and what it costs. */
  narrator: { label: string; pricePer1kChars: number } | null;
  /** Speaks the narration (the screen confirms the price and saves it first). */
  onMakeVoice: (draft: ShotDraft) => void;
  onRemoveVoice: () => Promise<void>;
  onGenerate: (draft: ShotDraft) => void;
  onMove: (direction: -1 | 1) => void;
  onDelete: () => void;
}) {
  const { message } = App.useApp();
  // Each prompt is a Thai/English pair (see lib/promptPair.ts). They can also
  // change outside this card (a dialog saves what was typed there), which the
  // hook handles: what was typed here only counts while the saved value is
  // still the one it was typed over.
  const motion = usePromptPair("motion", shot.promptTh, shot.prompt);
  const picture = usePromptPair("picture", shot.picturePromptTh, shot.picturePrompt);
  const endPic = usePromptPair("endPicture", shot.endPicturePromptTh, shot.endPicturePrompt);
  const [seconds, setSeconds] = useState(shot.seconds);
  const [narration, setNarration] = useTyped(shot.narrationTh);
  const translating = motion.busy || picture.busy || endPic.busy;

  // "ให้ AI แก้ช็อตนี้": the owner says in a sentence what the shot should show and
  // the script helper rewrites the three prompts. The result goes into the boxes
  // (nothing is saved until Save), and what was there is kept to undo it.
  const [reviseOpen, setReviseOpen] = useState(false);
  const [instruction, setInstruction] = useState("");
  const [revising, setRevising] = useState(false);
  const [beforeRevision, setBeforeRevision] = useState<{
    motion: [string, string];
    picture: [string, string];
    endPic: [string, string];
  } | null>(null);

  async function revise() {
    setRevising(true);
    try {
      const r = await reviseShot(shot.videoId, shot.id, instruction);
      setBeforeRevision({
        motion: [motion.th, motion.en],
        picture: [picture.th, picture.en],
        endPic: [endPic.th, endPic.en],
      });
      motion.setBoth(r.promptTh, r.prompt);
      picture.setBoth(r.picturePromptTh, r.picturePrompt);
      endPic.setBoth(r.endPicturePromptTh, r.endPicturePrompt);
      setReviseOpen(false);
      setInstruction("");
      message.success(
        `AI เขียนใหม่แล้ว (≈ ${formatUsd(r.costUsd)}) ตรวจแล้วกด Save · ภาพที่สร้างไว้แล้วไม่เปลี่ยน กด Change เพื่อสร้างภาพใหม่จากพรอมต์นี้`,
        8,
      );
    } catch (err) {
      message.error(err instanceof Error ? err.message : "The revision failed.");
    } finally {
      setRevising(false);
    }
  }

  function undoRevision() {
    if (!beforeRevision) return;
    motion.setBoth(...beforeRevision.motion);
    picture.setBoth(...beforeRevision.picture);
    endPic.setBoth(...beforeRevision.endPic);
    setBeforeRevision(null);
  }

  const draft: ShotDraft = {
    prompt: motion.en,
    promptTh: motion.th,
    picturePrompt: picture.en,
    picturePromptTh: picture.th,
    endPicturePrompt: endPic.en,
    endPicturePromptTh: endPic.th,
    narrationTh: narration,
    seconds,
  };
  const dirty =
    motion.en.trim() !== shot.prompt ||
    motion.th.trim() !== shot.promptTh ||
    picture.en.trim() !== shot.picturePrompt ||
    picture.th.trim() !== shot.picturePromptTh ||
    endPic.en.trim() !== shot.endPicturePrompt ||
    endPic.th.trim() !== shot.endPicturePromptTh ||
    narration.trim() !== shot.narrationTh ||
    seconds !== shot.seconds;

  /** The draft with every English prompt brought up to date: a Thai prompt that
   *  was edited (or has no English yet) is translated first, one at a time. A
   *  failed translation stops the action, with the reason shown. */
  async function fresh(): Promise<ShotDraft | null> {
    try {
      const prompt = await motion.ensureEnglish();
      const picturePrompt = await picture.ensureEnglish();
      const endPicturePrompt = await endPic.ensureEnglish();
      return { ...draft, prompt, picturePrompt, endPicturePrompt };
    } catch (err) {
      message.error(err instanceof Error ? err.message : "The translation failed.");
      return null;
    }
  }
  /** Wraps an action that takes the draft so it gets the fresh one. */
  const withFresh = (fn: (d: ShotDraft) => void | Promise<void>) => async () => {
    const d = await fresh();
    if (d) await fn(d);
  };

  // How long the narration takes to say: the measured length of its audio when it
  // has one that matches the text, else an estimate from the characters.
  const speechCharsAll = [...narration.trim()].length;
  const estimateSeconds =
    shot.voice?.current && narration.trim() === shot.voice.text
      ? shot.voice.durationMs / 1000
      : estimateSpeechSeconds(narration);
  const narrationTooLong = estimateSeconds > seconds;
  // The clip length that holds the real audio, offered as one click.
  const voiceFit = shot.voice ? secondsForAudio(secondsOptions, shot.voice.durationMs) : null;

  const hasPicture = shot.imageId !== "" || shot.imageUrl !== "";
  const pending = shot.latest?.status === "pending";
  const price = perSecond === undefined ? undefined : perSecond * seconds;
  // A clip needs a prompt (the Thai counts: it is translated when pressed) and
  // an English box without Thai in it.
  const englishHasThai = hasThaiLetters(motion.en);
  const canGenerate =
    (motion.th.trim() !== "" || motion.en.trim() !== "") &&
    !englishHasThai &&
    hasPicture &&
    !pending &&
    !busy &&
    !translating;


  const voiceSrc = shot.voice ? shotVoiceSrc(shot.videoId, shot.id, shot.voice.createdAt) : undefined;

  return (
    <Card
      id={`shot-${shot.id}`}
      size="small"
      style={{ scrollMarginTop: 72 }}
      title={`Shot ${shot.position}${shot.approved ? " · OK ✓" : ""}`}
      extra={
        <Space size={4}>
          <Button
            size="small"
            type="text"
            aria-label="Move shot up"
            disabled={shot.position === 1 || busy}
            onClick={() => onMove(-1)}
          >
            ↑
          </Button>
          <Button
            size="small"
            type="text"
            aria-label="Move shot down"
            disabled={shot.position === count || busy}
            onClick={() => onMove(1)}
          >
            ↓
          </Button>
          <Popconfirm
            title="Delete this shot?"
            description="Clips already made from it stay in the Generations tab."
            okText="Delete"
            okButtonProps={{ danger: true }}
            onConfirm={onDelete}
          >
            <Button size="small" type="text" danger disabled={busy}>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      }
    >
      {/* Four sections, in the order the work goes: pictures, what they say and
          do (prompts), what the narrator says, then the clip itself. */}
      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "stretch" }}>
          <Section n={1} title="ภาพ · Pictures" subtitle="ภาพเริ่มต้น และภาพจบ (ไม่บังคับ)" style={{ flex: "0 0 auto" }}>
            <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
              {/* Starting picture */}
              <div style={{ width: 108 }}>
                <Typography.Text type="secondary" style={{ display: "block", fontSize: 12, marginBottom: 4 }}>
                  เริ่มต้น · Start
                </Typography.Text>
                {hasPicture ? (
                  <ZoomableImage
                    src={shot.imageId ? imageSrc(shot.imageId) : shot.imageUrl}
                    alt={`Starting picture of shot ${shot.position}`}
                    width={108}
                    height={192}
                  />
                ) : (
                  <EmptyPicture>No starting picture yet</EmptyPicture>
                )}
                <Space size={4} style={{ marginTop: 4 }} wrap>
                  <Button size="small" disabled={busy} onClick={() => void withFresh(onChoosePicture)()}>
                    {hasPicture ? "Change" : "Choose…"}
                  </Button>
                  {hasPicture && (
                    <Button size="small" type="text" disabled={busy} onClick={() => void withFresh(onRemovePicture)()}>
                      Remove
                    </Button>
                  )}
                  {hasPreviousEnd && (
                    <Button
                      size="small"
                      type="text"
                      disabled={busy}
                      onClick={() => void withFresh(onStartFromPreviousEnd)()}
                    >
                      Use previous end
                    </Button>
                  )}
                </Space>
              </div>

              {/* End picture: the clip flows from the starting picture to this one. */}
              <div style={{ width: 108 }}>
                <Typography.Text type="secondary" style={{ display: "block", fontSize: 12, marginBottom: 4 }}>
                  จบ · End
                </Typography.Text>
                {shot.endImageId ? (
                  <ZoomableImage
                    src={imageSrc(shot.endImageId)}
                    alt={`End picture of shot ${shot.position}`}
                    width={108}
                    height={192}
                  />
                ) : (
                  <EmptyPicture>
                    {hasPicture
                      ? "No end picture (optional): the clip decides how it ends"
                      : "Choose a starting picture first"}
                  </EmptyPicture>
                )}
                <Space size={4} style={{ marginTop: 4 }} wrap>
                  <Button
                    size="small"
                    disabled={busy || !hasPicture}
                    onClick={() => void withFresh(onChooseEnd)()}
                  >
                    {shot.endImageId ? "Change end" : "End picture…"}
                  </Button>
                  {canUseNextAsEnd && (
                    <Button
                      size="small"
                      type="text"
                      disabled={busy || !hasPicture}
                      onClick={() => void withFresh(onUseNextAsEnd)()}
                    >
                      Use next shot&apos;s
                    </Button>
                  )}
                  {shot.endImageId && (
                    <Button size="small" type="text" disabled={busy} onClick={() => void withFresh(onRemoveEnd)()}>
                      Remove
                    </Button>
                  )}
                </Space>
              </div>
            </div>
          </Section>

          <Section
            n={2}
            title="พรอมต์ · Prompts"
            subtitle="พิมพ์ไทย ระบบแปลเป็นอังกฤษให้โมเดลก่อนสร้าง"
            style={{ flex: "1 1 360px", minWidth: 0 }}
          >
            <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
              <Button
                size="small"
                disabled={busy || translating || pending}
                onClick={() => setReviseOpen(true)}
              >
                ✨ ให้ AI แก้ช็อตนี้
              </Button>
              {beforeRevision && (
                <Button size="small" type="text" onClick={undoRevision}>
                  ย้อนกลับ
                </Button>
              )}
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                พิมพ์ประโยคเดียวว่าอยากให้ภาพเป็นอย่างไร AI เขียนพรอมต์ใหม่ให้ครบทั้ง 3 ช่อง
              </Typography.Text>
            </div>
            <PromptPairField
              pair={motion}
              kind="motion"
              title="การเคลื่อนไหวของคลิป · Motion"
              hint="บอกว่าอะไรขยับ กล้องทำอะไร และจบอย่างไร (ไม่ต้องบรรยายหน้าตา ภาพกำหนดไว้แล้ว)"
              placeholder="เช่น แมวส่ายก้น กระโจนใส่ถุง ตีลังกาแล้วลุกขึ้น กล้องนิ่ง"
              minRows={3}
              maxRows={10}
            />
            <PromptPairField
              pair={picture}
              kind="picture"
              title="ภาพเริ่มต้น · Picture"
              hint="ใครอยู่ในภาพ ทำอะไร มุมกล้อง บรรยากาศ (หน้าต่าง Make with AI เริ่มจากข้อความนี้)"
              placeholder="เช่น ภาพระยะกลางของแมวนั่งตัวตรงกลางพรม มองกล้อง มีถุงพลาสติกสีขาวลอยลงมา"
            />
            {(hasPicture || endPic.th.trim() !== "" || endPic.en.trim() !== "") && (
              <PromptPairField
                pair={endPic}
                kind="endPicture"
                title="ภาพจบ · End picture"
                hint="เฉพาะสิ่งที่เปลี่ยนเมื่อจบช็อต (ท่าหรือตำแหน่ง) กล้องและห้องเหมือนเดิม เว้นว่างถ้าช็อตนี้ไม่มีภาพจบ"
                placeholder="เช่น เปลี่ยนเฉพาะท่าของแมว: ก้มตัวต่ำ หางตั้ง ถุงยุบอยู่บนพรมตรงหน้าจมูก"
                maxRows={6}
              />
            )}
          </Section>
        </div>

        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "stretch" }}>
          {/* The narration and its voice-over. The narration is plain Thai: the voice
              reads it directly, so there is no English twin to keep. */}
          <Section
            n={3}
            title="บทพากย์ · Narration"
            subtitle="สิ่งที่คนเล่าเรื่องพูดทับช็อตนี้ พิมพ์ไทยตรงๆ ไม่ส่งให้โมเดลภาพหรือคลิป ใช้สร้างเสียงเท่านั้น"
            style={{ flex: "1 1 340px", minWidth: 0 }}
          >
            <Input.TextArea
              value={narration}
              onChange={(e) => setNarration(e.target.value)}
              autoSize={{ minRows: 2, maxRows: 8 }}
              maxLength={1500}
              placeholder="เช่น หลายคนอ่านหนังสือรวดเดียวก่อนสอบ แต่สมองของเรากลับลืมเร็วกว่าที่คิด"
            />
            <Typography.Text
              type={narrationTooLong ? "danger" : "secondary"}
              style={{ display: "block", fontSize: 12, marginTop: 2 }}
            >
              {speechChars(narration) === 0
                ? "ยังไม่มีบทพากย์"
                : `${speechChars(narration)} ตัวอักษร ≈ ${estimateSeconds.toFixed(1)} วินาที (ประมาณจากจำนวนตัวอักษร)`}
              {narrationTooLong
                ? ` · ยาวกว่าช็อต ${seconds} วินาที ย่อบทหรือเพิ่มความยาวช็อต`
                : ""}
            </Typography.Text>
            {narrationWarnings(narration).map((w) => (
              <Typography.Text key={w} type="warning" style={{ display: "block", fontSize: 12, marginTop: 2 }}>
                ⚠ {w}
              </Typography.Text>
            ))}

            <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap", marginTop: 8 }}>
              <Button
                size="small"
                disabled={!narrator || narration.trim() === "" || busy || translating}
                onClick={() => void onMakeVoice(draft)}
              >
                {shot.voice ? "สร้างเสียงใหม่" : "สร้างเสียง"}
                {narrator && narration.trim() !== ""
                  ? ` (≈ ${formatUsd((narrator.pricePer1kChars * speechCharsAll) / 1000)})`
                  : ""}
              </Button>
              {narrator ? (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  เสียง: {narrator.label}
                </Typography.Text>
              ) : (
                <Typography.Text type="warning" style={{ fontSize: 12 }}>
                  ยังไม่ได้เลือกเสียงพากย์ (Settings → Thai voice samples → “ใช้เป็นเสียงพากย์”)
                </Typography.Text>
              )}
            </div>

            {shot.voice && (
              <div style={{ marginTop: 8 }}>
                <audio controls preload="none" src={voiceSrc} style={{ display: "block", width: "100%" }} />
                <Typography.Text
                  type={shot.voice.current ? "secondary" : "warning"}
                  style={{ fontSize: 12, display: "block", marginTop: 2 }}
                >
                  เสียงจริง {(shot.voice.durationMs / 1000).toFixed(1)} วินาที (±0.1)
                  {shot.voice.current ? "" : " · บทเปลี่ยนไปหลังสร้างเสียงนี้ สร้างเสียงใหม่เพื่อให้ตรงกัน"}
                </Typography.Text>
                <Space size={8} wrap style={{ marginTop: 4 }}>
                  {voiceFit && voiceFit.seconds !== seconds && (
                    <Button size="small" onClick={() => setSeconds(voiceFit.seconds)}>
                      ตั้งความยาวช็อตเป็น {voiceFit.seconds} วินาที ตามเสียง
                    </Button>
                  )}
                  {voiceFit && !voiceFit.fits && (
                    <Typography.Text type="danger" style={{ fontSize: 12 }}>
                      เสียงยาวกว่าช็อตที่ยาวสุดของโมเดลนี้ ย่อบทพากย์
                    </Typography.Text>
                  )}
                  <Popconfirm
                    title="Remove this voice-over?"
                    description="The narration text stays."
                    okText="Remove"
                    okButtonProps={{ danger: true }}
                    onConfirm={() => void onRemoveVoice()}
                  >
                    <Button size="small" type="text" danger>
                      ลบเสียง
                    </Button>
                  </Popconfirm>
                </Space>
              </div>
            )}
          </Section>

          {/* The clip: its length and price, the button that spends, and what came out. */}
          <Section
            n={4}
            title="คลิป · Clip"
            subtitle="ความยาว ราคา และผลที่ได้ กดเล่นเพื่อฟังพร้อมเสียงพากย์"
            style={{ flex: "1 1 340px", minWidth: 0 }}
          >
            <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
              <Typography.Text type="secondary">Length</Typography.Text>
              <SecondsField options={secondsOptions} value={seconds} onChange={setSeconds} />
              <Typography.Text type="secondary">
                {price !== undefined ? `≈ ${formatUsd(price)} per attempt` : "price not known"}
              </Typography.Text>
            </div>
            <Space style={{ marginTop: 8 }} wrap>
              <Button type="primary" disabled={!canGenerate} onClick={() => void withFresh(onGenerate)()}>
                {shot.attempts > 0 ? "Generate again" : "Generate"}
                {price !== undefined ? ` (≈ ${formatUsd(price)})` : ""}
              </Button>
            </Space>
            {!hasPicture && (
              <Typography.Paragraph type="secondary" style={{ margin: "8px 0 0" }}>
                Choose a starting picture to generate this shot.
              </Typography.Paragraph>
            )}

            {/* The newest attempt's state, and the take in use */}
            <div style={{ borderTop: "1px solid var(--acmp-rule-soft)", marginTop: 12, paddingTop: 12 }}>
              {shot.latest ? (
                <>
                  <Tally status={shot.latest.status} />
                  <Typography.Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
                    {shot.attempts} attempt{shot.attempts === 1 ? "" : "s"}
                  </Typography.Text>
                  {shot.latest.status === "failed" && (
                    <Typography.Paragraph type="danger" style={{ margin: "8px 0 0", fontSize: 12 }}>
                      {shot.latest.error || "It failed."}
                    </Typography.Paragraph>
                  )}
                  {shot.take?.output && (
                    <div style={{ marginTop: 8, maxWidth: 260 }}>
                      <ShotPlayer
                        videoSrc={shot.take.output}
                        voiceSrc={voiceSrc}
                        voiceCurrent={shot.voice?.current}
                      />
                      <Typography.Text type="secondary" style={{ fontSize: 12, display: "block", marginTop: 4 }}>
                        {shot.takeChosen ? "Your chosen take" : "Newest take"}
                        {shot.latest.status === "pending" ? " (a new one is being made)" : ""}
                        {shot.takeChosen &&
                        shot.latest.status === "succeeded" &&
                        shot.latest.id !== shot.take.id
                          ? " · a newer take exists"
                          : ""}
                      </Typography.Text>
                      <Space size={8} wrap>
                        <a href={shot.take.output} target="_blank" rel="noreferrer">
                          Open clip
                        </a>
                        {shot.attempts > 1 && (
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={onOpenTakes}>
                            Compare takes ({shot.attempts})
                          </Button>
                        )}
                      </Space>
                      <Typography.Paragraph type="secondary" style={{ margin: "4px 0 0", fontSize: 12 }}>
                        fal keeps the file at least 7 days. Download what you want to keep.
                      </Typography.Paragraph>
                    </div>
                  )}
                </>
              ) : (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  Not generated yet.
                </Typography.Text>
              )}
            </div>
          </Section>
        </div>

        {/* Saves everything typed above (prompts, narration, length). */}
        <div style={{ display: "flex", justifyContent: "flex-end", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
          <Typography.Text type={dirty ? "warning" : "secondary"} style={{ fontSize: 12 }}>
            {dirty ? "มีการแก้ไขที่ยังไม่ได้บันทึก" : "บันทึกแล้ว"}
          </Typography.Text>
          <Modal
            title={`ให้ AI แก้ Shot ${shot.position}`}
            open={reviseOpen}
            onCancel={() => !revising && setReviseOpen(false)}
            okText="เขียนใหม่ (ไม่กี่เซ็นต์ที่สุด)"
            cancelText="ยกเลิก"
            okButtonProps={{ disabled: instruction.trim().length < 3 }}
            confirmLoading={revising}
            onOk={() => void revise()}
            destroyOnHidden
            {...mobileModal}
          >
            <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
              บอกว่าช็อตนี้ควรเห็นอะไร AI จะดูบทพากย์ พรอมต์เดิม และช็อตข้างเคียง แล้วเขียนภาพเริ่มต้น ภาพจบ
              และการเคลื่อนไหวใหม่ ผลจะใส่ในกล่อง ยังไม่บันทึกจนกว่าคุณกด Save (ย้อนกลับได้)
            </Typography.Paragraph>
            <Input.TextArea
              value={instruction}
              onChange={(e) => setInstruction(e.target.value)}
              autoSize={{ minRows: 3, maxRows: 8 }}
              maxLength={600}
              placeholder="เช่น ตัวการ์ตูนยืนอยู่บนตาชั่ง มีรูปกระเพาะอยู่บนตัว มีไอคอนอาหารอยู่บนกระเพาะ แล้วลูกศรพาอาหารออกไป"
            />
            <Typography.Text type="secondary" style={{ display: "block", fontSize: 12, marginTop: 6 }}>
              ค่าใช้จ่ายเป็นค่า token ของ AI เขียนข้อความ ประมาณไม่กี่เซ็นต์ที่สุด (ยังไม่นับในเพดานรายวัน)
              ไม่สร้างภาพและไม่สร้างคลิป
            </Typography.Text>
          </Modal>
          <Button disabled={!dirty || busy || translating} loading={translating} onClick={() => void withFresh(onSave)()}>
            Save{dirty ? "" : "d"}
          </Button>
        </div>
      </div>
    </Card>
  );
}

/** One boxed part of the shot card: a numbered title and what it is for, so the
 *  parts of a long card read as separate steps. */
function Section({
  n,
  title,
  subtitle,
  style,
  children,
}: {
  n: number;
  title: string;
  subtitle: string;
  style?: CSSProperties;
  children: ReactNode;
}) {
  return (
    <section
      style={{
        border: "1px solid var(--acmp-rule)",
        background: "var(--acmp-inset)",
        borderRadius: 12,
        padding: 12,
        ...style,
      }}
    >
      <div
        style={{
          display: "flex",
          gap: 8,
          alignItems: "flex-start",
          paddingBottom: 8,
          marginBottom: 10,
          borderBottom: "1px solid var(--acmp-rule-soft)",
        }}
      >
        <span
          aria-hidden
          style={{
            flex: "0 0 auto",
            width: 22,
            height: 22,
            borderRadius: "50%",
            background: "var(--acmp-tungsten)",
            color: "#000",
            fontSize: 12,
            fontWeight: 700,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          {n}
        </span>
        <div style={{ minWidth: 0 }}>
          <Typography.Text strong style={{ fontSize: 14, display: "block" }}>
            {title}
          </Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12, display: "block" }}>
            {subtitle}
          </Typography.Text>
        </div>
      </div>
      {children}
    </section>
  );
}

/** The dashed box shown where a picture is not set yet. */
function EmptyPicture({ children }: { children: ReactNode }) {
  return (
    <div
      style={{
        width: 108,
        height: 192,
        border: "1px dashed var(--acmp-dashed)",
        borderRadius: 10,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        textAlign: "center",
        padding: 8,
      }}
    >
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {children}
      </Typography.Text>
    </div>
  );
}
