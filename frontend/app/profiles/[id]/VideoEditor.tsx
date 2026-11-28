"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Alert, App, Button, Popconfirm, Space, Spin, Typography } from "antd";
import {
  addShot,
  ApiError,
  deleteShot,
  deleteShotVoice,
  deleteVideo,
  generateShot,
  generateShots,
  getVoiceCandidates,
  getStudioPricing,
  getVideo,
  makeShotVoice,
  renameVideo,
  reorderShots,
  setShotApproved,
  updateShot,
} from "@/lib/api";
import { formatUsd } from "@/lib/format";
import { narratorInfo } from "@/lib/narration";
import type { PromptTexts } from "@/lib/promptPair";
import {
  MAX_SHOTS,
  perSecondUsd,
  priceCaveat,
  pricingFor,
  type Shot,
  type StudioPricing,
  type Video,
} from "@/lib/types";
import { refreshCredits } from "../../CreditsBadge";
import PictureModal, { type PictureChoice } from "./PictureModal";
import ClipList from "./ClipList";
import ReviewPanel from "./ReviewPanel";
import ShotCard, { type ShotDraft } from "./ShotCard";
import Storyboard, { isReadyToMake } from "./Storyboard";
import TakesModal from "./TakesModal";

const POLL_EVERY_MS = 5000;
const POLL_MAX_TRIES = 240; // 20 minutes; a real clip took about 8

/** The first message of a failed request: a 422 carries one per field. */
function errorText(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.fieldErrors) {
    const first = Object.values(err.fieldErrors)[0];
    if (first) return first;
  }
  return err instanceof Error ? err.message : fallback;
}

/**
 * A video: its shots in order, each with its own prompt, length and starting
 * picture, and the newest attempt at making it. Each Generate spends fal
 * credits and asks first, with the price.
 */
export default function VideoEditor({
  videoId,
  profileId,
  videoModel,
  videoAudio,
  narratorVoice,
  onBack,
}: {
  videoId: string;
  profileId: string;
  /** The profile's video model and sound setting: the lengths and the price. */
  videoModel: string;
  videoAudio: boolean;
  /** The profile's narrator voice key ("" = none): who speaks the narration. */
  narratorVoice: string;
  onBack: () => void;
}) {
  const { message, modal } = App.useApp();
  const [video, setVideo] = useState<Video | null>(null);
  const [shots, setShots] = useState<Shot[]>([]);
  const [pricing, setPricing] = useState<StudioPricing | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  // The shot whose starting (or, with `end`, ending) picture is being chosen,
  // with what was typed on its card (saved together with the picture).
  const [choosing, setChoosing] = useState<{
    shot: Shot;
    draft: ShotDraft;
    end: boolean;
  } | null>(null);
  // The shot whose takes are being compared.
  const [comparing, setComparing] = useState<Shot | null>(null);
  // A fix suggested by the AI review: opens that shot's rewrite dialog, typed in.
  const [pendingRevise, setPendingRevise] = useState<{ shotId: string; instruction: string; key: number } | null>(null);
  // The narrator voice chosen in Settings, with its label and price (null: none).
  const [narrator, setNarrator] = useState<{ label: string; pricePer1kChars: number } | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await getVideo(videoId);
      setVideo(r.video);
      setShots(r.shots);
    } catch (err) {
      setError(errorText(err, "Failed to load the video."));
    }
  }, [videoId]);

  // The first load. setState happens after the request returns, not in the
  // effect body itself (which the react-hooks lint rule forbids).
  useEffect(() => {
    let cancelled = false;
    getVideo(videoId)
      .then((r) => {
        if (cancelled) return;
        setVideo(r.video);
        setShots(r.shots);
      })
      .catch((err) => {
        if (!cancelled) setError(errorText(err, "Failed to load the video."));
      });
    getStudioPricing()
      .then(setPricing)
      .catch(() => {
        /* the cards say the price is not known */
      });
    return () => {
      cancelled = true;
    };
  }, [videoId]);

  // The profile's narrator voice: its label and price (null: none chosen). Runs
  // again when the profile's voice is changed, so the change shows without a
  // reload (the video itself is not reloaded).
  useEffect(() => {
    let cancelled = false;
    getVoiceCandidates()
      .then((c) => {
        if (!cancelled) setNarrator(narratorInfo(narratorVoice, c.candidates));
      })
      .catch(() => {
        /* no narrator shown */
      });
    return () => {
      cancelled = true;
    };
  }, [narratorVoice]);

  // While a shot is being made, poll for the worker to finish it; when the last
  // one finishes fal has charged for it, so refresh the balance in the menu.
  const anyPending = shots.some((s) => s.latest?.status === "pending");
  const hadPending = useRef(false);
  useEffect(() => {
    if (hadPending.current && !anyPending) refreshCredits();
    hadPending.current = anyPending;
  }, [anyPending]);
  useEffect(() => {
    if (!anyPending) return;
    let tries = 0;
    const iv = setInterval(() => {
      tries += 1;
      void load();
      if (tries >= POLL_MAX_TRIES) clearInterval(iv);
    }, POLL_EVERY_MS);
    return () => clearInterval(iv);
  }, [anyPending, load]);

  const model = pricingFor(pricing, videoModel);
  const perSecond = perSecondUsd(model, videoAudio);
  // The lengths the model takes; until the list loads, Kling's range.
  const secondsOptions = model?.seconds ?? Array.from({ length: 13 }, (_, i) => i + 3);
  const caveat = priceCaveat(model, videoAudio);
  const silent = videoAudio ? "" : "silent ";

  /** Runs one change to the video, showing why it failed. Returns whether it worked. */
  async function run(fn: () => Promise<unknown>, fallback: string): Promise<boolean> {
    setBusy(true);
    try {
      await fn();
      return true;
    } catch (err) {
      message.error(errorText(err, fallback));
      return false;
    } finally {
      setBusy(false);
    }
  }

  const replaceShot = (next: Shot) =>
    setShots((all) => all.map((s) => (s.id === next.id ? next : s)));

  /** Whether what is on the card differs from what is saved for the shot. */
  const planChanged = (shot: Shot, d: ShotDraft) =>
    d.prompt.trim() !== shot.prompt ||
    d.promptTh.trim() !== shot.promptTh ||
    d.picturePrompt.trim() !== shot.picturePrompt ||
    d.picturePromptTh.trim() !== shot.picturePromptTh ||
    d.endPicturePrompt.trim() !== shot.endPicturePrompt ||
    d.endPicturePromptTh.trim() !== shot.endPicturePromptTh ||
    d.narrationTh.trim() !== shot.narrationTh ||
    d.seconds !== shot.seconds;

  /** Saves a shot's whole plan: what was typed plus its current picture. */
  function plan(
    shot: Shot,
    draft: ShotDraft,
    change: { start?: PictureChoice | "none"; end?: string | "none" } = {},
  ) {
    // Without a change the shot keeps the pictures it has; a new start replaces
    // the old (a shot has an image id or a link, never both).
    let imageId = shot.imageId;
    let imageUrl = shot.imageUrl;
    let endImageId = shot.endImageId;
    const { start, end } = change;
    if (start === "none") {
      imageId = "";
      imageUrl = "";
      endImageId = ""; // an end picture needs a starting picture
    } else if (start && "imageId" in start) {
      imageId = start.imageId;
      imageUrl = "";
    } else if (start) {
      imageUrl = start.imageUrl;
      imageId = "";
    }
    if (end === "none") endImageId = "";
    else if (end) endImageId = end;
    return updateShot(videoId, shot.id, {
      prompt: draft.prompt,
      picturePrompt: draft.picturePrompt,
      endPicturePrompt: draft.endPicturePrompt,
      promptTh: draft.promptTh,
      picturePromptTh: draft.picturePromptTh,
      endPicturePromptTh: draft.endPicturePromptTh,
      narrationTh: draft.narrationTh,
      seconds: draft.seconds,
      imageUrl,
      imageId,
      endImageId,
    });
  }

  async function save(shot: Shot, draft: ShotDraft) {
    await run(async () => replaceShot(await plan(shot, draft)), "Failed to save.");
  }

  async function pickPicture(choice: PictureChoice, prompts?: PromptTexts) {
    if (!choosing) return;
    const { shot, end } = choosing;
    // What was typed in the dialog (the Thai and the English made from it)
    // replaces the card's picture prompt (or, for an end picture, its end
    // picture prompt), so it is kept with the shot.
    const draft: ShotDraft = !prompts
      ? choosing.draft
      : end
        ? { ...choosing.draft, endPicturePrompt: prompts.english, endPicturePromptTh: prompts.thai }
        : { ...choosing.draft, picturePrompt: prompts.english, picturePromptTh: prompts.thai };
    const change = end
      ? { end: "imageId" in choice ? choice.imageId : "none" } // an end picture is never a link
      : { start: choice };
    const ok = await run(async () => replaceShot(await plan(shot, draft, change)), "Failed to use the picture.");
    if (ok) setChoosing(null);
  }

  async function removePicture(shot: Shot, draft: ShotDraft) {
    await run(async () => replaceShot(await plan(shot, draft, { start: "none" })), "Failed to remove the picture.");
  }

  async function removeEndPicture(shot: Shot, draft: ShotDraft) {
    await run(async () => replaceShot(await plan(shot, draft, { end: "none" })), "Failed to remove the end picture.");
  }

  /** The end picture is the next shot's starting picture: one click, so the
   *  clip ends on the pose the next one starts from. */
  async function endOnNextPicture(shot: Shot, draft: ShotDraft) {
    const next = shots.find((s) => s.position === shot.position + 1);
    if (!next || next.imageId === "") return;
    await run(
      async () => replaceShot(await plan(shot, draft, { end: next.imageId })),
      "Failed to set the end picture.",
    );
  }

  /** The starting picture is the previous shot's end picture: one click, so the
   *  shot begins on the pose the one before it ends on (the same stored picture
   *  serves as both; it is removed only when neither uses it any more). */
  async function startFromPreviousEnd(shot: Shot, draft: ShotDraft) {
    const prev = shots.find((s) => s.position === shot.position - 1);
    if (!prev || prev.endImageId === "") return;
    await run(
      async () => replaceShot(await plan(shot, draft, { start: { imageId: prev.endImageId } })),
      "Failed to set the starting picture.",
    );
  }

  // Generating spends credits, so ask first, with the price.
  function confirmGenerate(shot: Shot, draft: ShotDraft) {
    const price = perSecond === undefined ? undefined : perSecond * draft.seconds;
    const flow = shot.endImageId
      ? "from the starting picture to the end picture"
      : "from the starting picture";
    modal.confirm({
      title: `Generate shot ${shot.position}?`,
      content: (
        <>
          {price !== undefined ? (
            <p style={{ margin: 0 }}>
              A {draft.seconds}-second {silent}clip {flow}:{" "}
              <strong>≈ {formatUsd(price)}</strong>, charged to your fal.ai credits.{caveat}
              {shot.endImageId
                ? " (The price with an end picture has not been confirmed against a bill.)"
                : ""}
              {shot.attempts > 0
                ? ` This is attempt ${shot.attempts + 1}; the earlier ones stay in the Generations tab.`
                : ""}
            </p>
          ) : (
            <p style={{ margin: 0 }}>
              A {draft.seconds}-second {silent}clip. The price of this model with
              these settings is not known (or could not be loaded), so it is not
              shown here; it is charged to your fal.ai credits.
            </p>
          )}
          {/* What the clip model will actually receive: the English made from the
              owner's Thai, shown so it can be checked before the money is spent. */}
          <p style={{ margin: "12px 0 0", fontSize: 13 }}>
            <strong>English sent to the AI:</strong> {draft.prompt}
          </p>
        </>
      ),
      okText: price !== undefined ? `Generate (≈ ${formatUsd(price)})` : "Generate",
      cancelText: "Cancel",
      onOk: async () => {
        await run(async () => {
          // Save what was typed first, so the attempt uses exactly what is on screen.
          if (planChanged(shot, draft)) {
            replaceShot(await plan(shot, draft));
          }
          await generateShot(videoId, shot.id);
          message.success(`Shot ${shot.position} queued.`);
          await load();
        }, "Failed to generate.");
      },
    });
  }

  // Speaking the narration costs fal credits per character, so ask first, with the
  // price. The narration on screen is saved first, so the voice says what is shown.
  function confirmMakeVoice(shot: Shot, draft: ShotDraft) {
    if (!narrator) return;
    const text = draft.narrationTh.trim();
    const chars = [...text].length;
    const price = (narrator.pricePer1kChars * chars) / 1000;
    modal.confirm({
      title: `Make the voice for shot ${shot.position}?`,
      content: (
        <>
          <p style={{ margin: "0 0 8px" }}>
            {chars} characters spoken by <strong>{narrator.label}</strong>:{" "}
            <strong>≈ {formatUsd(price)}</strong>, charged to your fal.ai credits.
            {shot.voice ? " The earlier voice-over of this shot is replaced." : ""}
          </p>
          <p style={{ margin: 0, fontSize: 13 }}>{text}</p>
        </>
      ),
      okText: `Make the voice (≈ ${formatUsd(price)})`,
      cancelText: "Cancel",
      onOk: async () => {
        await run(async () => {
          // Save what was typed first, so the voice says exactly what is on screen.
          if (planChanged(shot, draft)) {
            replaceShot(await plan(shot, draft));
          }
          replaceShot(await makeShotVoice(videoId, shot.id));
          refreshCredits();
          message.success(`Voice for shot ${shot.position} is ready.`);
        }, "Failed to make the voice.");
      },
    });
  }

  async function removeVoice(shot: Shot) {
    await run(async () => {
      await deleteShotVoice(videoId, shot.id);
      replaceShot({ ...shot, voice: null });
    }, "Failed to remove the voice-over.");
  }

  async function approve(shot: Shot, approved: boolean) {
    await run(async () => replaceShot(await setShotApproved(videoId, shot.id, approved)), "Failed to approve.");
  }

  // Making the approved shots in one go spends credits, so ask once, with the
  // total and the list.
  function confirmMakeApproved() {
    const list = shots.filter(isReadyToMake);
    if (list.length === 0) return;
    const seconds = list.reduce((n, s) => n + s.seconds, 0);
    const price = perSecond === undefined ? undefined : perSecond * seconds;
    modal.confirm({
      title: `Generate ${list.length} shot${list.length === 1 ? "" : "s"}?`,
      content: (
        <>
          <p style={{ margin: "0 0 8px" }}>
            Shots {list.map((s) => s.position).join(", ")}: {seconds} seconds of
            {silent}clips
            {price !== undefined ? (
              <>
                , <strong>≈ {formatUsd(price)}</strong> in all
              </>
            ) : (
              " (the price is not known for this model with these settings)"
            )}
            , charged to your fal.ai credits. They are made at the same time.
          </p>
          <p style={{ margin: 0 }}>
            Shots already made are not included; a retake costs again.
          </p>
        </>
      ),
      okText: price !== undefined ? `Generate (≈ ${formatUsd(price)})` : "Generate",
      cancelText: "Cancel",
      onOk: async () => {
        await run(async () => {
          const r = await generateShots(videoId, list.map((s) => s.id));
          if (r.queued.length > 0) {
            message.success(`${r.queued.length} shot${r.queued.length === 1 ? "" : "s"} queued.`);
          }
          if (r.skipped.length > 0) {
            message.warning(`${r.skipped.length} skipped: ${r.skipped[0].reason}`);
          }
          await load();
        }, "Failed to generate.");
      },
    });
  }

  async function move(shot: Shot, direction: -1 | 1) {
    const ids = shots.map((s) => s.id);
    const i = ids.indexOf(shot.id);
    const j = i + direction;
    if (j < 0 || j >= ids.length) return;
    [ids[i], ids[j]] = [ids[j], ids[i]];
    await run(async () => {
      await reorderShots(videoId, ids);
      await load();
    }, "Failed to reorder.");
  }

  async function remove(shot: Shot) {
    await run(async () => {
      await deleteShot(videoId, shot.id);
      await load();
    }, "Failed to delete the shot.");
  }

  async function add() {
    await run(async () => {
      await addShot(videoId);
      await load();
    }, "Failed to add a shot.");
  }

  async function rename(title: string) {
    if (!video || title.trim() === video.title) return;
    await run(async () => {
      const v = await renameVideo(videoId, title);
      setVideo((cur) => (cur ? { ...cur, title: v.title } : cur));
    }, "Failed to rename.");
  }

  async function removeVideo() {
    const ok = await run(() => deleteVideo(videoId), "Failed to delete the video.");
    if (ok) {
      message.success("Video deleted.");
      onBack();
    }
  }

  if (error) {
    return (
      <Alert
        type="error"
        showIcon
        title={error}
        action={<Button size="small" onClick={onBack}>Back to videos</Button>}
      />
    );
  }
  if (!video) {
    return (
      <div style={{ textAlign: "center", padding: 48 }}>
        <Spin />
      </div>
    );
  }

  const totalSeconds = shots.reduce((n, s) => n + s.seconds, 0);
  const totalPrice = perSecond === undefined ? undefined : perSecond * totalSeconds;

  return (
    <div>
      <Button type="link" onClick={onBack} style={{ padding: 0 }}>
        ← All videos
      </Button>

      <div style={{ display: "flex", justifyContent: "space-between", gap: 16, flexWrap: "wrap", alignItems: "flex-start" }}>
        <Typography.Title
          level={3}
          editable={{ onChange: (t) => void rename(t), tooltip: "Rename" }}
          style={{ margin: "8px 0", minWidth: 0 }}
        >
          {video.title}
        </Typography.Title>
        <Popconfirm
          title="Delete this video?"
          description="Its shots are deleted. Clips already made stay in the Generations tab."
          okText="Delete video"
          okButtonProps={{ danger: true }}
          onConfirm={() => void removeVideo()}
        >
          <Button danger disabled={busy}>Delete video</Button>
        </Popconfirm>
      </div>

      <Typography.Paragraph type="secondary">
        {shots.length} shot{shots.length === 1 ? "" : "s"} · {totalSeconds} seconds
        {totalPrice !== undefined
          ? ` · ≈ ${formatUsd(totalPrice)} if every shot is made once (retakes cost again)`
          : ""}
        . Generate each shot when its prompt and picture are ready; join the clips
        in your editor afterwards.
      </Typography.Paragraph>

      <Storyboard
        shots={shots}
        perSecond={perSecond}
        busy={busy}
        onApprove={(s, a) => void approve(s, a)}
        onMakeApproved={confirmMakeApproved}
      />

      <ReviewPanel
        videoId={videoId}
        shotCount={shots.length}
        disabled={busy}
        onFix={(shotId, instruction) => {
          setPendingRevise({ shotId, instruction, key: Date.now() });
          document.getElementById(`shot-${shotId}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
        }}
      />

      <Space orientation="vertical" size={16} style={{ width: "100%" }}>
        {shots.map((shot) => (
          <ShotCard
            key={shot.id}
            pendingRevise={pendingRevise?.shotId === shot.id ? pendingRevise : null}
            onPendingReviseDone={() => setPendingRevise(null)}
            shot={shot}
            count={shots.length}
            perSecond={perSecond}
            secondsOptions={secondsOptions}
            busy={busy}
            onSave={(d) => save(shot, d)}
            onChoosePicture={(d) => setChoosing({ shot, draft: d, end: false })}
            onRemovePicture={(d) => removePicture(shot, d)}
            canUseNextAsEnd={shots.some(
              (s) => s.position === shot.position + 1 && s.imageId !== "",
            )}
            onChooseEnd={(d) => setChoosing({ shot, draft: d, end: true })}
            hasPreviousEnd={shots.some(
              (s) => s.position === shot.position - 1 && s.endImageId !== "",
            )}
            onStartFromPreviousEnd={(d) => void startFromPreviousEnd(shot, d)}
            onOpenTakes={() => setComparing(shot)}
            onUseNextAsEnd={(d) => void endOnNextPicture(shot, d)}
            onRemoveEnd={(d) => removeEndPicture(shot, d)}
            narrator={narrator}
            onMakeVoice={(d) => confirmMakeVoice(shot, d)}
            onRemoveVoice={() => removeVoice(shot)}
            onGenerate={(d) => confirmGenerate(shot, d)}
            onMove={(dir) => void move(shot, dir)}
            onDelete={() => void remove(shot)}
          />
        ))}
      </Space>

      <Button
        style={{ marginTop: 16 }}
        disabled={busy || shots.length >= MAX_SHOTS}
        onClick={() => void add()}
      >
        {shots.length >= MAX_SHOTS ? `At most ${MAX_SHOTS} shots` : "Add a shot"}
      </Button>

      <ClipList shots={shots} />

      <TakesModal
        open={comparing !== null}
        videoId={videoId}
        shot={comparing}
        onClose={() => setComparing(null)}
        onChanged={(next) => {
          replaceShot(next);
          setComparing(next);
        }}
      />
      <PictureModal
        open={choosing !== null}
        profileId={profileId}
        end={choosing?.end ?? false}
        context={choosing ? `shot:${choosing.shot.id}:${choosing.end ? "end" : "start"}` : undefined}
        initialPrompt={
          choosing
            ? choosing.end
              ? { english: choosing.draft.endPicturePrompt, thai: choosing.draft.endPicturePromptTh }
              : { english: choosing.draft.picturePrompt, thai: choosing.draft.picturePromptTh }
            : { english: "", thai: "" }
        }
        // A starting picture can match the other shots' scene; an END picture is
        // made from the shot's own starting picture (same camera, only the pose
        // changes), since the clip morphs whatever differs between the two.
        scenePictures={
          choosing?.end
            ? choosing.shot.imageId !== ""
              ? [{ imageId: choosing.shot.imageId, label: "Starting picture" }]
              : []
            : shots
                .filter((s) => s.imageId !== "" && s.id !== choosing?.shot.id)
                .map((s) => ({ imageId: s.imageId, label: `Shot ${s.position}` }))
        }
        onClose={() => setChoosing(null)}
        onPick={pickPicture}
      />
    </div>
  );
}
