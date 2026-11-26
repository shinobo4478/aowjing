"use client";

import { useState } from "react";
import { Button, Card, Input, InputNumber, Popconfirm, Space, Typography } from "antd";
import Tally from "@/components/Tally";
import { imageSrc } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import type { Shot } from "@/lib/types";

/** The part of a shot's plan typed on the card. The picture is chosen in its
 *  own dialog and saved at once. */
export interface ShotDraft {
  prompt: string;
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
  minSeconds,
  maxSeconds,
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
  onGenerate,
  onMove,
  onDelete,
}: {
  shot: Shot;
  /** How many shots the video has, to disable moving the first/last. */
  count: number;
  /** Price per second for this video's style; undefined = not known. */
  perSecond: number | undefined;
  minSeconds: number;
  maxSeconds: number;
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
  onGenerate: (draft: ShotDraft) => void;
  onMove: (direction: -1 | 1) => void;
  onDelete: () => void;
}) {
  const [prompt, setPrompt] = useState(shot.prompt);
  const [seconds, setSeconds] = useState(shot.seconds);
  const draft: ShotDraft = { prompt, seconds };
  const dirty = prompt.trim() !== shot.prompt || seconds !== shot.seconds;

  const hasPicture = shot.imageId !== "" || shot.imageUrl !== "";
  const pending = shot.latest?.status === "pending";
  const price = perSecond === undefined ? undefined : perSecond * seconds;
  const canGenerate = prompt.trim() !== "" && hasPicture && !pending && !busy;

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
            // Served by the API with the session cookie (or a pasted link), so
            // next/image cannot optimize it.
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={shot.imageId ? imageSrc(shot.imageId) : shot.imageUrl}
              alt={`Starting picture of shot ${shot.position}`}
              style={{ width: 108, height: 192, objectFit: "cover", borderRadius: 4, display: "block" }}
            />
          ) : (
            <div
              style={{
                width: 108,
                height: 192,
                border: "1px dashed rgba(255,255,255,0.25)",
                borderRadius: 4,
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
            <Button size="small" disabled={busy} onClick={() => onChoosePicture(draft)}>
              {hasPicture ? "Change" : "Choose…"}
            </Button>
            {hasPicture && (
              <Button size="small" type="text" disabled={busy} onClick={() => void onRemovePicture(draft)}>
                Remove
              </Button>
            )}
            {hasPreviousEnd && (
              <Button size="small" type="text" disabled={busy} onClick={() => onStartFromPreviousEnd(draft)}>
                Use previous end
              </Button>
            )}
          </Space>
        </div>

        {/* End picture: the clip flows from the starting picture to this one. */}
        <div style={{ width: 108 }}>
          {shot.endImageId ? (
            // Served by the API with the session cookie, so next/image cannot optimize it.
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={imageSrc(shot.endImageId)}
              alt={`End picture of shot ${shot.position}`}
              style={{ width: 108, height: 192, objectFit: "cover", borderRadius: 4, display: "block" }}
            />
          ) : (
            <div
              style={{
                width: 108,
                height: 192,
                border: "1px dashed rgba(255,255,255,0.25)",
                borderRadius: 4,
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
              onClick={() => onChooseEnd(draft)}
            >
              {shot.endImageId ? "Change end" : "End picture…"}
            </Button>
            {canUseNextAsEnd && (
              <Button
                size="small"
                type="text"
                disabled={busy || !hasPicture}
                onClick={() => onUseNextAsEnd(draft)}
              >
                Use next shot&apos;s
              </Button>
            )}
            {shot.endImageId && (
              <Button size="small" type="text" disabled={busy} onClick={() => void onRemoveEnd(draft)}>
                Remove
              </Button>
            )}
          </Space>
        </div>

        {/* Plan */}
        <div style={{ flex: "1 1 320px", minWidth: 0 }}>
          <Input.TextArea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            autoSize={{ minRows: 4, maxRows: 12 }}
            maxLength={2000}
            placeholder="What moves in this shot, and how the camera moves, in English. The picture already sets how it looks."
          />
          <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap", marginTop: 8 }}>
            <Typography.Text type="secondary">Length</Typography.Text>
            <InputNumber
              min={minSeconds}
              max={maxSeconds}
              precision={0}
              value={seconds}
              onChange={(v) => setSeconds(typeof v === "number" ? v : shot.seconds)}
              suffix="s"
              style={{ width: 90 }}
            />
            <Typography.Text type="secondary">
              {price !== undefined
                ? `≈ ${formatUsd(price)} per attempt`
                : "price not known"}
            </Typography.Text>
          </div>

          <Space style={{ marginTop: 12 }} wrap>
            <Button
              disabled={!dirty || busy}
              onClick={() => void onSave(draft)}
            >
              Save{dirty ? "" : "d"}
            </Button>
            <Button
              type="primary"
              disabled={!canGenerate}
              onClick={() => onGenerate(draft)}
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
                    style={{ width: "100%", maxHeight: 260, borderRadius: 4, background: "#000" }}
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
