"use client";

import { useState } from "react";
import { App, Button, Card, Input, Popconfirm, Space, Typography } from "antd";
import SecondsField from "@/components/SecondsField";
import Tally from "@/components/Tally";
import { imageSrc, shotVoiceSrc } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import { estimateSpeechSeconds, narrationWarnings, secondsForAudio, speechChars } from "@/lib/narration";
import { hasThaiLetters, usePromptPair, useTyped } from "@/lib/promptPair";
import type { Shot } from "@/lib/types";
import PromptPairField from "./PromptPairField";
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
      <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
        {/* Starting picture */}
        <div style={{ width: 108 }}>
          {hasPicture ? (
            <ZoomableImage
              src={shot.imageId ? imageSrc(shot.imageId) : shot.imageUrl}
              alt={`Starting picture of shot ${shot.position}`}
              width={108}
              height={192}
            />
          ) : (
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
                No starting picture yet
              </Typography.Text>
            </div>
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
              <Button size="small" type="text" disabled={busy} onClick={() => void withFresh(onStartFromPreviousEnd)()}>
                Use previous end
              </Button>
            )}
          </Space>
        </div>

        {/* End picture: the clip flows from the starting picture to this one. */}
        <div style={{ width: 108 }}>
          {shot.endImageId ? (
            <ZoomableImage
              src={imageSrc(shot.endImageId)}
              alt={`End picture of shot ${shot.position}`}
              width={108}
              height={192}
            />
          ) : (
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
                {hasPicture
                  ? "No end picture (optional): the clip decides how it ends"
                  : "Choose a starting picture first"}
              </Typography.Text>
            </div>
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

        {/* Plan */}
        <div style={{ flex: "1 1 320px", minWidth: 0 }}>
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
          {/* The narration and its voice-over. The narration is plain Thai: the voice
              reads it directly, so there is no English twin to keep. */}
          <div style={{ marginTop: 12 }}>
            <Typography.Text strong style={{ fontSize: 13 }}>
              บทพากย์ · Narration
            </Typography.Text>
            <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
              สิ่งที่คนเล่าเรื่องพูดทับช็อตนี้ พิมพ์ไทยตรงๆ ไม่ส่งให้โมเดลภาพหรือคลิป ใช้สร้างเสียงเท่านั้น
            </Typography.Text>
            <Input.TextArea
              value={narration}
              onChange={(e) => setNarration(e.target.value)}
              autoSize={{ minRows: 2, maxRows: 8 }}
              maxLength={1500}
              placeholder="เช่น หลายคนอ่านหนังสือรวดเดียวก่อนสอบ แต่สมองของเรากลับลืมเร็วกว่าที่คิด"
              style={{ marginTop: 4 }}
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

            <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap", marginTop: 6 }}>
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
                <audio
                  controls
                  preload="none"
                  src={shotVoiceSrc(shot.videoId, shot.id, shot.voice.createdAt)}
                  style={{ display: "block", width: "100%" }}
                />
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
          </div>

          <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap", marginTop: 8 }}>
            <Typography.Text type="secondary">Length</Typography.Text>
            <SecondsField options={secondsOptions} value={seconds} onChange={setSeconds} />
            <Typography.Text type="secondary">
              {price !== undefined
                ? `≈ ${formatUsd(price)} per attempt`
                : "price not known"}
            </Typography.Text>
          </div>

          <Space style={{ marginTop: 12 }} wrap>
            <Button
              disabled={!dirty || busy || translating}
              loading={translating}
              onClick={() => void withFresh(onSave)()}
            >
              Save{dirty ? "" : "d"}
            </Button>
            <Button
              type="primary"
              disabled={!canGenerate}
              onClick={() => void withFresh(onGenerate)()}
            >
              {shot.attempts > 0 ? "Generate again" : "Generate"}
              {price !== undefined ? ` (≈ ${formatUsd(price)})` : ""}
            </Button>
          </Space>
          {!hasPicture && (
            <Typography.Paragraph type="secondary" style={{ margin: "8px 0 0" }}>
              Choose a starting picture to generate this shot.
            </Typography.Paragraph>
          )}
        </div>

        {/* The newest attempt's state, and the take in use */}
        <div style={{ flex: "0 1 220px", minWidth: 160 }}>
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
                <div style={{ marginTop: 8 }}>
                  <video
                    controls
                    preload="metadata"
                    src={shot.take.output}
                    style={{ width: "100%", maxHeight: 260, borderRadius: 10, background: "#000" }}
                  />
                  <Typography.Text type="secondary" style={{ fontSize: 12, display: "block" }}>
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
      </div>
    </Card>
  );
}
