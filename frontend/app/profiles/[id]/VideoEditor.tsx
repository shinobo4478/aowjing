"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Alert, App, Button, Popconfirm, Space, Spin, Typography } from "antd";
import {
  addShot,
  ApiError,
  deleteShot,
  deleteVideo,
  generateShot,
  getStudioPricing,
  getVideo,
  renameVideo,
  reorderShots,
  updateShot,
} from "@/lib/api";
import { formatUsd } from "@/lib/format";
import { MAX_SHOTS, type Shot, type StudioPricing, type Video } from "@/lib/types";
import { refreshCredits } from "../../CreditsBadge";
import PictureModal, { type PictureChoice } from "./PictureModal";
import ShotCard, { type ShotDraft } from "./ShotCard";

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
  videoStyle,
  onBack,
}: {
  videoId: string;
  profileId: string;
  videoStyle: string;
  onBack: () => void;
}) {
  const { message, modal } = App.useApp();
  const [video, setVideo] = useState<Video | null>(null);
  const [shots, setShots] = useState<Shot[]>([]);
  const [pricing, setPricing] = useState<StudioPricing | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  // The shot whose starting picture is being chosen, with what was typed on its
  // card (saved together with the picture).
  const [choosing, setChoosing] = useState<{ shot: Shot; draft: ShotDraft } | null>(null);

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

  const perSecond = pricing?.imageUsdPerSecond[videoStyle];
  const minSeconds = pricing?.minSeconds ?? 3;
  const maxSeconds = pricing?.maxSeconds ?? 15;

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

  /** Saves a shot's whole plan: what was typed plus its current picture. */
  function plan(shot: Shot, draft: ShotDraft, picture?: PictureChoice | "none") {
    // Without `picture` the shot keeps the one it has; a new one replaces it
    // (a shot has an image id or a link, never both).
    let imageId = shot.imageId;
    let imageUrl = shot.imageUrl;
    if (picture === "none") {
      imageId = "";
      imageUrl = "";
    } else if (picture && "imageId" in picture) {
      imageId = picture.imageId;
      imageUrl = "";
    } else if (picture) {
      imageUrl = picture.imageUrl;
      imageId = "";
    }
    return updateShot(videoId, shot.id, {
      prompt: draft.prompt,
      seconds: draft.seconds,
      imageUrl,
      imageId,
    });
  }

  async function save(shot: Shot, draft: ShotDraft) {
    await run(async () => replaceShot(await plan(shot, draft)), "Failed to save.");
  }

  async function pickPicture(choice: PictureChoice) {
    if (!choosing) return;
    const { shot, draft } = choosing;
    const ok = await run(async () => replaceShot(await plan(shot, draft, choice)), "Failed to use the picture.");
    if (ok) setChoosing(null);
  }

  async function removePicture(shot: Shot, draft: ShotDraft) {
    await run(async () => replaceShot(await plan(shot, draft, "none")), "Failed to remove the picture.");
  }

  // Generating spends credits, so ask first, with the price.
  function confirmGenerate(shot: Shot, draft: ShotDraft) {
    const price = perSecond === undefined ? undefined : perSecond * draft.seconds;
    modal.confirm({
      title: `Generate shot ${shot.position}?`,
      content:
        price !== undefined ? (
          <p style={{ margin: 0 }}>
            A {draft.seconds}-second silent clip from the starting picture:{" "}
            <strong>≈ {formatUsd(price)}</strong>, charged to your fal.ai credits.
            {shot.attempts > 0
              ? ` This is attempt ${shot.attempts + 1}; the earlier ones stay in the Generations tab.`
              : ""}
          </p>
        ) : (
          <p style={{ margin: 0 }}>
            A {draft.seconds}-second silent clip. The price could not be loaded,
            so it is not shown here; it is charged to your fal.ai credits.
          </p>
        ),
      okText: price !== undefined ? `Generate (≈ ${formatUsd(price)})` : "Generate",
      cancelText: "Cancel",
      onOk: async () => {
        await run(async () => {
          // Save what was typed first, so the attempt uses exactly what is on screen.
          if (draft.prompt.trim() !== shot.prompt || draft.seconds !== shot.seconds) {
            replaceShot(await plan(shot, draft));
          }
          await generateShot(videoId, shot.id);
          message.success(`Shot ${shot.position} queued.`);
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

      <Space orientation="vertical" size={16} style={{ width: "100%" }}>
        {shots.map((shot) => (
          <ShotCard
            key={shot.id}
            shot={shot}
            count={shots.length}
            perSecond={perSecond}
            minSeconds={minSeconds}
            maxSeconds={maxSeconds}
            busy={busy}
            onSave={(d) => save(shot, d)}
            onChoosePicture={(d) => setChoosing({ shot, draft: d })}
            onRemovePicture={(d) => removePicture(shot, d)}
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

      <PictureModal
        open={choosing !== null}
        profileId={profileId}
        onClose={() => setChoosing(null)}
        onPick={pickPicture}
      />
    </div>
  );
}
