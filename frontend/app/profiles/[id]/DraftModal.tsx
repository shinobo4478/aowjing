"use client";

import { useState } from "react";
import { App, Button, Input, InputNumber, Modal, Typography } from "antd";
import { ApiError, draftVideo } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import type { Video } from "@/lib/types";
import { mobileModal } from "@/lib/ui";
import { refreshCredits } from "../../CreditsBadge";

/**
 * Asks the script helper to draft a whole video: the owner gives the topic and
 * how many shots of how many seconds, and gets a new Video with the scene and
 * each shot's picture prompt, motion prompt and length filled in, to read and
 * change. It makes no pictures and no clips, so the only cost is the model's
 * reply; fal bills per token, so the exact cost is shown afterwards.
 */
export default function DraftModal({
  open,
  profileId,
  onClose,
  onDrafted,
}: {
  open: boolean;
  profileId: string;
  onClose: () => void;
  onDrafted: (video: Video) => void;
}) {
  return (
    <Modal
      title="Draft a video with AI"
      open={open}
      onCancel={onClose}
      footer={null}
      width={640}
      destroyOnHidden
      {...mobileModal}
    >
      {open && <DraftForm profileId={profileId} onDrafted={onDrafted} />}
    </Modal>
  );
}

function DraftForm({
  profileId,
  onDrafted,
}: {
  profileId: string;
  onDrafted: (video: Video) => void;
}) {
  const { message, modal } = App.useApp();
  const [topic, setTopic] = useState("");
  const [shots, setShots] = useState(4);
  const [seconds, setSeconds] = useState(4);
  const [title, setTitle] = useState("");
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const totalSeconds = shots * seconds;

  // The draft costs a few cents (fal bills per token), so ask first.
  function confirmDraft() {
    modal.confirm({
      title: "Draft this video?",
      content: (
        <>
          <p style={{ margin: "0 0 8px" }}>
            The AI writes the shot list for {shots} shot{shots === 1 ? "" : "s"} of{" "}
            {seconds} s ({totalSeconds} s in all). It costs a few cents at most,
            charged to your fal.ai credits; fal bills per token, so the exact cost
            is shown once it is done.
          </p>
          <p style={{ margin: 0 }}>
            It makes no pictures and no clips: you read and change the shots first.
          </p>
        </>
      ),
      okText: "Draft it",
      cancelText: "Cancel",
      onOk: () => draft(),
    });
  }

  async function draft() {
    setBusy(true);
    setErrors({});
    try {
      const r = await draftVideo({
        profileId,
        topic,
        shots,
        seconds,
        title: title.trim() || undefined,
      });
      refreshCredits();
      message.success(`Drafted with ${r.model}. It cost ${formatUsd(r.costUsd)}.`);
      onDrafted(r.video);
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        setErrors(err.fieldErrors as Record<string, string>);
      } else {
        message.error(err instanceof Error ? err.message : "The draft failed.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <Typography.Paragraph type="secondary">
        Say what the video is about. The AI writes the scene and, for each shot, what
        the starting picture shows and how the clip moves. You can write the topic in
        Thai or English; the prompts it writes are in English.
      </Typography.Paragraph>

      <Typography.Text>Topic</Typography.Text>
      <Input.TextArea
        value={topic}
        onChange={(e) => setTopic(e.target.value)}
        autoSize={{ minRows: 4, maxRows: 10 }}
        maxLength={1000}
        showCount
        status={errors.topic ? "error" : undefined}
        placeholder="e.g. A cat is startled by a plastic bag, plays with it, and ends up proud of it. Funny, quick, loops."
      />
      {errors.topic && (
        <Typography.Text type="danger" style={{ display: "block", marginTop: 4 }}>
          {errors.topic}
        </Typography.Text>
      )}

      <div style={{ display: "flex", gap: 24, flexWrap: "wrap", marginTop: 28 }}>
        <div>
          <Typography.Text>Shots</Typography.Text>
          <div>
            <InputNumber min={1} max={12} precision={0} value={shots} onChange={(v) => setShots(typeof v === "number" ? v : 4)} />
          </div>
          {errors.shots && <Typography.Text type="danger">{errors.shots}</Typography.Text>}
        </div>
        <div>
          <Typography.Text>Seconds per shot</Typography.Text>
          <div>
            <InputNumber min={3} max={15} precision={0} value={seconds} onChange={(v) => setSeconds(typeof v === "number" ? v : 4)} suffix="s" />
          </div>
          {errors.seconds && <Typography.Text type="danger">{errors.seconds}</Typography.Text>}
        </div>
        <div style={{ alignSelf: "end" }}>
          <Typography.Text type="secondary">{totalSeconds} seconds in all</Typography.Text>
        </div>
      </div>

      <div style={{ marginTop: 16 }}>
        <Typography.Text>Title (optional)</Typography.Text>
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          maxLength={120}
          status={errors.title ? "error" : undefined}
          placeholder="The AI suggests one if you leave it empty"
        />
        {errors.title && <Typography.Text type="danger">{errors.title}</Typography.Text>}
      </div>

      <Button
        type="primary"
        style={{ marginTop: 20 }}
        loading={busy}
        disabled={topic.trim().length < 3}
        onClick={confirmDraft}
      >
        Draft the video
      </Button>
      {busy && (
        <Typography.Text type="secondary" style={{ marginLeft: 12 }}>
          Writing the draft… this can take up to a minute.
        </Typography.Text>
      )}
    </div>
  );
}
