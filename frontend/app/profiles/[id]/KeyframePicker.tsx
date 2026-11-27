"use client";

import { useEffect, useRef, useState } from "react";
import { Alert, App, Button, Checkbox, Segmented, Select, Typography } from "antd";
import {
  ApiError,
  createKeyframe,
  deleteKeyframe,
  getKeyframe,
  getKeyframeModels,
  imageSrc,
  listReferences,
} from "@/lib/api";
import { formatUsd } from "@/lib/format";
import { hasThaiLetters, usePromptPair, type PromptPair, type PromptTexts } from "@/lib/promptPair";
import type { Keyframe, KeyframeModels } from "@/lib/types";
import { refreshCredits } from "../../CreditsBadge";
import PromptPairField from "./PromptPairField";

const POLL_EVERY_MS = 3000;
const POLL_MAX_TRIES = 120; // 6 minutes; an image takes seconds to a minute

/** Most same-scene pictures one request may carry. Mirrors studio.MaxSceneReferences. */
const MAX_SCENE_PICTURES = 2;

/**
 * Makes the starting picture of a clip: describe the scene, choose the image
 * model and how many candidates, then pick one. The profile's reference
 * pictures go along (unless switched off) so the character stays the same.
 *
 * `value` is the picked picture's id ("" = none yet). Every candidate that no
 * clip uses is discarded when this goes away (the dialog closes), so the
 * unpicked ones do not pile up in the database.
 */
export default function KeyframePicker({
  profileId,
  value,
  onChange,
  scenePictures = [],
  sameCamera = false,
  initialPrompt,
  onPromptChange,
}: {
  profileId: string;
  value: string;
  onChange: (imageId: string) => void;
  /** Pictures of the same scene this one should match: the other shots of a
   *  video, in order. The first is chosen at the start (it is the "master"). */
  scenePictures?: { imageId: string; label: string }[];
  /** Making an END picture: the clip flows from the starting picture to this
   *  one, so the camera, room and light must stay exactly as they are and only
   *  the pose may change. */
  sameCamera?: boolean;
  /** The picture prompt to start from (a shot's saved one): the Thai the owner
   *  edits and the English the image model receives. */
  initialPrompt?: PromptTexts;
  /** Called whenever either text changes (typing or a translation), so the
   *  caller can keep both with the shot. */
  onPromptChange?: (prompt: PromptTexts) => void;
}) {
  const { message, modal } = App.useApp();
  const [info, setInfo] = useState<KeyframeModels | null>(null);
  const [refCount, setRefCount] = useState(0);
  // The prompt is a Thai/English pair: the owner writes Thai; the English is what
  // the image model receives (translated before pictures are paid for).
  const raw = usePromptPair(
    sameCamera ? "endPicture" : "picture",
    initialPrompt?.thai ?? "",
    initialPrompt?.english ?? "",
  );
  // The same pair, telling the caller about every change.
  const pair: PromptPair = {
    ...raw,
    setTh: (text) => {
      raw.setTh(text);
      onPromptChange?.({ english: raw.en, thai: text });
    },
    setEn: (text) => {
      raw.setEn(text);
      onPromptChange?.({ english: text, thai: raw.th });
    },
    toEnglish: async () => {
      const english = await raw.toEnglish();
      onPromptChange?.({ english, thai: raw.th });
      return english;
    },
    toThai: async () => {
      const thai = await raw.toThai();
      onPromptChange?.({ english: raw.en, thai });
      return thai;
    },
    ensureEnglish: async () => {
      const english = await raw.ensureEnglish();
      onPromptChange?.({ english, thai: raw.th });
      return english;
    },
  };
  const [model, setModel] = useState("");
  const [count, setCount] = useState(2);
  const [useRefs, setUseRefs] = useState(true);
  // The same-scene pictures to send along. Starts with the first (the master).
  const [sceneIds, setSceneIds] = useState<string[]>(() =>
    scenePictures.slice(0, 1).map((p) => p.imageId),
  );
  const [keyframe, setKeyframe] = useState<Keyframe | null>(null);
  const [busy, setBusy] = useState(false);

  // The newest request's id, kept for the unmount cleanup below.
  const keyframeIdRef = useRef<string | null>(null);
  useEffect(() => {
    keyframeIdRef.current = keyframe?.id ?? null;
  }, [keyframe]);
  useEffect(
    () => () => {
      const id = keyframeIdRef.current;
      if (id) void deleteKeyframe(id).catch(() => {});
    },
    [],
  );

  useEffect(() => {
    let cancelled = false;
    getKeyframeModels()
      .then((m) => {
        if (cancelled) return;
        setInfo(m);
        setModel((cur) => cur || m.models[0]?.key || "");
      })
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load the picture models.",
        ),
      );
    listReferences(profileId)
      .then((r) => {
        if (!cancelled) setRefCount(r.length);
      })
      .catch(() => {
        /* non-fatal: the checkbox just stays hidden */
      });
    return () => {
      cancelled = true;
    };
  }, [profileId, message]);

  // While a request is pending, poll for the worker to finish it.
  const pendingId = keyframe?.status === "pending" ? keyframe.id : null;
  useEffect(() => {
    if (!pendingId) return;
    let tries = 0;
    const iv = setInterval(() => {
      tries += 1;
      getKeyframe(pendingId)
        .then((k) => {
          setKeyframe(k);
          if (k.status !== "pending") refreshCredits(); // the pictures were paid for
        })
        .catch(() => clearInterval(iv));
      if (tries >= POLL_MAX_TRIES) clearInterval(iv);
    }, POLL_EVERY_MS);
    return () => clearInterval(iv);
  }, [pendingId]);

  const chosenModel = info?.models.find((m) => m.key === model);
  const effectiveRefs = (useRefs ? refCount : 0) + sceneIds.length;
  // A picture needs a description in either language (the Thai is translated
  // when the pictures are made).
  const promptTooShort = Math.max(pair.th.trim().length, pair.en.trim().length) < 3;

  function toggleScene(imageId: string, on: boolean) {
    setSceneIds((cur) =>
      on ? [...cur, imageId].slice(0, MAX_SCENE_PICTURES) : cur.filter((id) => id !== imageId),
    );
  }

  // Making pictures spends fal.ai credits, so ask first, with the price. The
  // Thai is translated first (a fraction of a cent) so the confirmation can show
  // the English the image model will receive.
  async function confirmMake() {
    if (!chosenModel) return;
    let english = pair.en;
    if (pair.stale) {
      try {
        english = await pair.ensureEnglish();
      } catch (err) {
        message.error(err instanceof Error ? err.message : "The translation failed.");
        return;
      }
    }
    if (english.trim() === "") return;
    if (hasThaiLetters(english)) {
      message.error("The English box contains Thai. Press “แปลเป็นอังกฤษ” first.");
      return;
    }
    const total = chosenModel.priceUsd * count;
    modal.confirm({
      title: `Make ${count} picture${count === 1 ? "" : "s"}?`,
      content: (
        <>
          <p style={{ margin: "0 0 8px" }}>
            {chosenModel.label}: {count} × {formatUsd(chosenModel.priceUsd)} ≈{" "}
            <strong>{formatUsd(total)}</strong>, charged to your fal.ai credits.
          </p>
          <p style={{ margin: "0 0 8px", fontSize: 13 }}>
            <strong>English sent to the AI:</strong> {english}
          </p>
          <p style={{ margin: 0 }}>
            {sceneIds.length > 0
              ? sameCamera
                ? "The starting picture will be kept exactly, except for the pose. "
                : `The scene of ${sceneIds.length} other shot${sceneIds.length === 1 ? "" : "s"} will be kept. `
              : ""}
            {effectiveRefs > 0
              ? `${effectiveRefs} reference picture${effectiveRefs === 1 ? "" : "s"} in all will be sent along. `
              : "No reference pictures will be sent. "}
            {keyframe ? "The pictures below will be replaced." : ""}
          </p>
        </>
      ),
      okText: `Make pictures (≈ ${formatUsd(total)})`,
      cancelText: "Cancel",
      onOk: () => make(english),
    });
  }

  async function make(english: string) {
    setBusy(true);
    try {
      // A new request replaces the old one; drop the old pictures (the ones no
      // clip uses) so they do not pile up.
      if (keyframe) void deleteKeyframe(keyframe.id).catch(() => {});
      onChange("");
      setKeyframe(
        await createKeyframe({
          profileId,
          prompt: english.trim(),
          model,
          count,
          useReferences: useRefs,
          sceneImageIds: sceneIds,
          sameCamera: sameCamera && sceneIds.length > 0,
        }),
      );
    } catch (err) {
      const field =
        err instanceof ApiError && err.fieldErrors
          ? Object.values(err.fieldErrors)[0]
          : undefined;
      message.error(
        field ?? (err instanceof Error ? err.message : "Could not start."),
      );
      setKeyframe(null);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ marginTop: 8 }}>
      <PromptPairField
        pair={pair}
        title={sameCamera ? "ภาพจบ · End picture" : "ภาพ · Picture"}
        hint={
          sameCamera
            ? "เฉพาะสิ่งที่เปลี่ยน (ท่าหรือตำแหน่ง) กล้อง ห้อง และแสงเหมือนภาพเริ่มต้นทุกอย่าง"
            : "ใครอยู่ในภาพ อยู่ที่ไหน แสงและบรรยากาศ พิมพ์เป็นภาษาไทยได้ ระบบแปลเป็นอังกฤษให้ตอนกดสร้างภาพ"
        }
        placeholder={
          sameCamera
            ? "เช่น เปลี่ยนเฉพาะท่าของแมว: ก้มตัวต่ำ หางตั้ง ถุงยุบอยู่บนพรมตรงหน้าจมูก"
            : "เช่น แมวนั่งตัวตรงกลางพรมในห้องนั่งเล่นที่แดดส่อง มองกล้อง ถุงพลาสติกสีขาวลอยลงมา"
        }
        minRows={4}
        maxRows={12}
      />

      <Select
        value={model || undefined}
        onChange={setModel}
        loading={!info}
        style={{ width: "100%", marginTop: 12 }}
        options={(info?.models ?? []).map((m) => ({
          value: m.key,
          label: `${m.label} — ${formatUsd(m.priceUsd)} per picture`,
        }))}
      />
      <div style={{ display: "flex", gap: 8, alignItems: "center", marginTop: 8 }}>
        <Typography.Text type="secondary">How many pictures</Typography.Text>
        <Segmented<number>
          value={count}
          onChange={setCount}
          options={Array.from({ length: info?.maxImages ?? 4 }, (_, i) => ({
            value: i + 1,
            label: `${i + 1}`,
          }))}
        />
      </div>

      {scenePictures.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <Typography.Text>
            {sameCamera
              ? "Start from (the camera, room, light and everything but the pose stay exactly the same):"
              : "Keep the same scene as (the room, light and character stay, the camera angle or moment changes):"}
          </Typography.Text>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 12, marginTop: 6 }}>
            {scenePictures.map((p) => {
              const on = sceneIds.includes(p.imageId);
              const full = !on && sceneIds.length >= MAX_SCENE_PICTURES;
              return (
                <Checkbox
                  key={p.imageId}
                  checked={on}
                  disabled={full}
                  onChange={(e) => toggleScene(p.imageId, e.target.checked)}
                >
                  {/* Served by the API with the session cookie, so next/image cannot optimize it. */}
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={imageSrc(p.imageId)}
                    alt={p.label}
                    style={{ width: 48, height: 85, objectFit: "cover", borderRadius: 8, verticalAlign: "middle", marginRight: 6 }}
                  />
                  {p.label}
                </Checkbox>
              );
            })}
          </div>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {sameCamera
              ? "Describe only the pose or action that differs."
              : `Up to ${MAX_SCENE_PICTURES}. Describe only what is different in this shot.`}
          </Typography.Text>
        </div>
      )}

      {refCount > 0 ? (
        <Checkbox
          checked={useRefs}
          onChange={(e) => setUseRefs(e.target.checked)}
          style={{ marginTop: 8 }}
        >
          Keep this profile&apos;s look ({refCount} reference picture
          {refCount === 1 ? "" : "s"})
        </Checkbox>
      ) : (
        <Typography.Paragraph type="secondary" style={{ margin: "8px 0 0" }}>
          This profile has no reference pictures, so the picture is made from the
          description alone. Add some on the profile&apos;s Look tab to keep the
          same character.
        </Typography.Paragraph>
      )}

      <div style={{ marginTop: 12, display: "flex", gap: 12, alignItems: "center" }}>
        <Button
          onClick={confirmMake}
          loading={busy || pair.busy || keyframe?.status === "pending"}
          disabled={!chosenModel || promptTooShort}
        >
          {keyframe ? "Make new pictures" : "Make pictures"}
        </Button>
        {chosenModel && (
          <Typography.Text type="secondary">
            ≈ {formatUsd(chosenModel.priceUsd * count)} for {count} picture
            {count === 1 ? "" : "s"}
            {effectiveRefs > 0 ? `, using ${effectiveRefs} reference` : ""}
          </Typography.Text>
        )}
      </div>

      {keyframe?.status === "failed" && (
        <Alert
          type="error"
          showIcon
          style={{ marginTop: 12 }}
          title="The pictures could not be made"
          description={keyframe.error}
        />
      )}

      {keyframe?.status === "succeeded" && (
        <div style={{ marginTop: 12 }}>
          <Typography.Text>Pick the one to start the clip from:</Typography.Text>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 8 }}>
            {keyframe.imageIds.map((id) => {
              const selected = id === value;
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => onChange(id)}
                  aria-pressed={selected}
                  aria-label={selected ? "Chosen picture" : "Choose this picture"}
                  style={{
                    padding: 0,
                    background: "none",
                    cursor: "pointer",
                    borderRadius: 14,
                    border: selected
                      ? "3px solid var(--ant-color-primary, #1677ff)"
                      : "3px solid transparent",
                  }}
                >
                  {/* Served by the API with the session cookie, so next/image cannot optimize it. */}
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={imageSrc(id)}
                    alt="Candidate starting picture"
                    style={{ width: 120, height: 213, objectFit: "cover", display: "block", borderRadius: 8 }}
                  />
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
