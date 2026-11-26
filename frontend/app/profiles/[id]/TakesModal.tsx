"use client";

import { useEffect, useState } from "react";
import { App, Button, Modal, Spin, Tag, Typography } from "antd";
import Tally from "@/components/Tally";
import { listShotAttempts, setShotTake } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { mobileModal } from "@/lib/ui";
import type { Shot, ShotAttempt } from "@/lib/types";

/**
 * Every attempt at a shot, newest first, side by side, to compare the takes and
 * pick the one to use. The take in use is what the shot shows and what the
 * download list links to; without a choice it is the newest successful one.
 */
export default function TakesModal({
  open,
  videoId,
  shot,
  onClose,
  onChanged,
}: {
  open: boolean;
  videoId: string;
  shot: Shot | null;
  onClose: () => void;
  /** The shot after a take was chosen (or the choice cleared). */
  onChanged: (shot: Shot) => void;
}) {
  return (
    <Modal
      title={shot ? `Takes of shot ${shot.position}` : "Takes"}
      open={open}
      onCancel={onClose}
      footer={null}
      width={760}
      destroyOnHidden
      {...mobileModal}
    >
      {open && shot && (
        <Takes videoId={videoId} shot={shot} onChanged={onChanged} />
      )}
    </Modal>
  );
}

function Takes({
  videoId,
  shot,
  onChanged,
}: {
  videoId: string;
  shot: Shot;
  onChanged: (shot: Shot) => void;
}) {
  const { message } = App.useApp();
  const [attempts, setAttempts] = useState<ShotAttempt[] | null>(null);
  const [busy, setBusy] = useState(false);
  // The take in use. Kept here so the tags follow a choice made in this dialog.
  const [inUse, setInUse] = useState(shot.take?.id ?? "");
  const [chosen, setChosen] = useState(shot.takeChosen);

  useEffect(() => {
    let cancelled = false;
    listShotAttempts(videoId, shot.id)
      .then((r) => {
        if (!cancelled) setAttempts(r.attempts);
      })
      .catch((err) => {
        message.error(err instanceof Error ? err.message : "Failed to load the takes.");
        if (!cancelled) setAttempts([]);
      });
    return () => {
      cancelled = true;
    };
  }, [videoId, shot.id, message]);

  async function choose(generationId: string) {
    setBusy(true);
    try {
      const next = await setShotTake(videoId, shot.id, generationId);
      setInUse(next.take?.id ?? "");
      setChosen(next.takeChosen);
      onChanged(next);
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to choose the take.");
    } finally {
      setBusy(false);
    }
  }

  if (attempts === null) {
    return (
      <div style={{ textAlign: "center", padding: 32 }}>
        <Spin />
      </div>
    );
  }

  return (
    <div>
      <Typography.Paragraph type="secondary">
        The take in use is the one this shot shows and the download list links to.
        {chosen
          ? " You chose it."
          : " Nothing is chosen, so the newest successful one is used."}
      </Typography.Paragraph>
      {chosen && (
        <Button size="small" disabled={busy} onClick={() => void choose("")} style={{ marginBottom: 12 }}>
          Go back to the newest successful take
        </Button>
      )}

      <div style={{ display: "flex", flexWrap: "wrap", gap: 16 }}>
        {attempts.map((a, i) => (
          <div key={a.id} style={{ width: 200 }}>
            <div style={{ marginBottom: 4 }}>
              <Tally status={a.status} />
              {a.id === inUse && (
                <Tag color="green" style={{ marginLeft: 8 }}>
                  in use
                </Tag>
              )}
            </div>
            {a.status === "succeeded" && a.output ? (
              <video
                controls
                preload="metadata"
                src={a.output}
                style={{ width: "100%", maxHeight: 340, borderRadius: 4, background: "#000" }}
              />
            ) : (
              <div
                style={{
                  height: 120,
                  border: "1px dashed rgba(255,255,255,0.25)",
                  borderRadius: 4,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  padding: 8,
                  textAlign: "center",
                }}
              >
                <Typography.Text type={a.status === "failed" ? "danger" : "secondary"} style={{ fontSize: 12 }}>
                  {a.status === "failed" ? a.error || "It failed." : "Still being made…"}
                </Typography.Text>
              </div>
            )}
            <div style={{ fontSize: 12, marginTop: 4 }}>
              Take {attempts.length - i} · {a.seconds} s · {formatDateTime(a.createdAt)}
            </div>
            {a.status === "succeeded" && a.id !== inUse && (
              <Button size="small" disabled={busy} onClick={() => void choose(a.id)} style={{ marginTop: 4 }}>
                Use this take
              </Button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
