"use client";

import { useEffect, useRef, useState } from "react";
import { Alert, App, Button, Checkbox, Input, Segmented, Select, Typography } from "antd";
import {
  ApiError,
  createKeyframe,
  deleteKeyframe,
  getKeyframe,
  getKeyframeModels,
  imageSrc,
  listReferences,
} from "@/lib/api";
import type { Keyframe, KeyframeModels } from "@/lib/types";

const POLL_EVERY_MS = 3000;
const POLL_MAX_TRIES = 120; // 6 minutes; an image takes seconds to a minute

/** Dollars to at most 4 decimals, so a price like $0.0398 is not shown as $0.04. */
const usd = (n: number) => `$${Number(n.toFixed(4))}`;

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
}: {
  profileId: string;
  value: string;
  onChange: (imageId: string) => void;
}) {
  const { message } = App.useApp();
  const [info, setInfo] = useState<KeyframeModels | null>(null);
  const [refCount, setRefCount] = useState(0);
  const [prompt, setPrompt] = useState("");
  const [model, setModel] = useState("");
  const [count, setCount] = useState(2);
  const [useRefs, setUseRefs] = useState(true);
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
        .then(setKeyframe)
        .catch(() => clearInterval(iv));
      if (tries >= POLL_MAX_TRIES) clearInterval(iv);
    }, POLL_EVERY_MS);
    return () => clearInterval(iv);
  }, [pendingId]);

  const chosenModel = info?.models.find((m) => m.key === model);
  const effectiveRefs = useRefs ? refCount : 0;
  const promptTooShort = prompt.trim().length < 3;

  async function make() {
    setBusy(true);
    try {
      // A new request replaces the old one; drop the old pictures (the ones no
      // clip uses) so they do not pile up.
      if (keyframe) void deleteKeyframe(keyframe.id).catch(() => {});
      onChange("");
      setKeyframe(
        await createKeyframe({
          profileId,
          prompt: prompt.trim(),
          model,
          count,
          useReferences: useRefs,
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
      <Input.TextArea
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        rows={3}
        maxLength={1500}
        placeholder="Describe the scene, in English: who is in it, where, the light, the mood…"
      />

      <Select
        value={model || undefined}
        onChange={setModel}
        loading={!info}
        style={{ width: "100%", marginTop: 8 }}
        options={(info?.models ?? []).map((m) => ({
          value: m.key,
          label: `${m.label} — ${usd(m.priceUsd)} per picture`,
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
          onClick={make}
          loading={busy || keyframe?.status === "pending"}
          disabled={!chosenModel || promptTooShort}
        >
          {keyframe ? "Make new pictures" : "Make pictures"}
        </Button>
        {chosenModel && (
          <Typography.Text type="secondary">
            ≈ {usd(chosenModel.priceUsd * count)} for {count} picture
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
                    borderRadius: 6,
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
                    style={{ width: 120, height: 213, objectFit: "cover", display: "block", borderRadius: 3 }}
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
