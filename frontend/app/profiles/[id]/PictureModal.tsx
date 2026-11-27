"use client";

import { useEffect, useRef, useState } from "react";
import { App, Button, Input, Modal, Segmented, Typography, Upload } from "antd";
import { ApiError, uploadImage } from "@/lib/api";
import { mobileModal } from "@/lib/ui";
import type { PromptTexts } from "@/lib/promptPair";
import { imageFileError, imageUrlError } from "./GenerateModal";
import KeyframePicker from "./KeyframePicker";
import PastPictures from "./PastPictures";

/** The picture the owner chose: a stored one (uploaded, made with AI, or reused)
 *  by id, or a public link. */
export type PictureChoice = { imageId: string } | { imageUrl: string };

type Source = "ai" | "past" | "file" | "link";

/**
 * Chooses a shot's starting picture, from the same four places as the Generate
 * dialog: make one with AI, reuse one an earlier clip used, upload one, or paste
 * a link. `onPick` is awaited before the dialog closes, so a picture made here
 * is attached to the shot before the dialog throws its unused candidates away.
 */
export default function PictureModal({
  open,
  profileId,
  scenePictures,
  end = false,
  initialPrompt = { english: "", thai: "" },
  onClose,
  onPick,
}: {
  open: boolean;
  profileId: string;
  /** Pictures to make this one match: the video's other shots, or (for an end
   *  picture) the shot's own starting picture. */
  scenePictures: { imageId: string; label: string }[];
  /** Choosing the END picture of a shot: the clip flows from the starting
   *  picture to this one, so a picture made here keeps the camera, room and
   *  light exactly and changes only the pose; a link is not accepted. */
  end?: boolean;
  /** The shot's saved picture prompt: Make with AI starts from it, and what is
   *  typed there is handed back with the chosen picture to be saved. (Not used
   *  for an end picture, whose prompt only says what changes.) */
  initialPrompt?: PromptTexts;
  onClose: () => void;
  onPick: (choice: PictureChoice, prompts?: PromptTexts) => Promise<void>;
}) {
  return (
    <Modal
      title={end ? "End picture" : "Starting picture"}
      open={open}
      onCancel={onClose}
      footer={null}
      width={760}
      destroyOnHidden
      {...mobileModal}
    >
      {open && (
        <Chooser
          profileId={profileId}
          scenePictures={scenePictures}
          end={end}
          initialPrompt={initialPrompt}
          onPick={onPick}
        />
      )}
    </Modal>
  );
}

function Chooser({
  profileId,
  scenePictures,
  end,
  initialPrompt,
  onPick,
}: {
  profileId: string;
  end: boolean;
  initialPrompt: PromptTexts;
  scenePictures: { imageId: string; label: string }[];
  onPick: (choice: PictureChoice, prompts?: PromptTexts) => Promise<void>;
}) {
  const { message } = App.useApp();
  const [source, setSource] = useState<Source>("ai");
  const [madeId, setMadeId] = useState("");
  // The picture prompt as it stands in the box, kept to save with the shot.
  const [madePrompt, setMadePrompt] = useState<PromptTexts>(initialPrompt);
  const [pastId, setPastId] = useState("");
  const [link, setLink] = useState("");
  const [picked, setPicked] = useState<{ file: File; preview: string } | null>(null);
  const [busy, setBusy] = useState(false);

  // A preview URL holds the file in memory until revoked (see GenerateModal).
  const pickedRef = useRef(picked);
  useEffect(() => {
    pickedRef.current = picked;
  }, [picked]);
  useEffect(
    () => () => {
      if (pickedRef.current) URL.revokeObjectURL(pickedRef.current.preview);
    },
    [],
  );
  function pickFile(file: File | null) {
    if (picked) URL.revokeObjectURL(picked.preview);
    setPicked(file ? { file, preview: URL.createObjectURL(file) } : null);
  }

  const linkError = source === "link" ? imageUrlError(link) : null;
  const ready =
    (source === "ai" && madeId !== "") ||
    (source === "past" && pastId !== "") ||
    (source === "file" && picked !== null) ||
    (source === "link" && link.trim() !== "" && !linkError);

  async function use() {
    setBusy(true);
    try {
      if (source === "ai") await onPick({ imageId: madeId }, madePrompt);
      else if (source === "past") await onPick({ imageId: pastId });
      else if (source === "link") await onPick({ imageUrl: link.trim() });
      else {
        // Upload inside the dialog so a refused picture (too small, …) shows
        // its message here and keeps the choice.
        let id: string;
        try {
          id = (await uploadImage(picked!.file)).id;
        } catch (err) {
          const field =
            err instanceof ApiError && err.fieldErrors
              ? Object.values(err.fieldErrors)[0]
              : undefined;
          message.error(
            field ?? (err instanceof Error ? err.message : "Upload failed."),
          );
          return;
        }
        await onPick({ imageId: id });
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Segmented<Source>
        size="small"
        value={source}
        onChange={setSource}
        options={[
          { value: "ai", label: "Make with AI" },
          { value: "past", label: "From an earlier clip" },
          { value: "file", label: "From this computer" },
          // An end picture must be a stored one (the backend takes only ids).
          ...(end ? [] : [{ value: "link" as const, label: "Link" }]),
        ]}
      />

      {source === "ai" && (
        <KeyframePicker
          profileId={profileId}
          value={madeId}
          onChange={setMadeId}
          scenePictures={scenePictures}
          sameCamera={end}
          initialPrompt={initialPrompt}
          onPromptChange={setMadePrompt}
        />
      )}
      {source === "past" && (
        <PastPictures profileId={profileId} value={pastId} onChange={setPastId} />
      )}
      {source === "file" && (
        <div style={{ marginTop: 8, display: "flex", gap: 12, alignItems: "center" }}>
          <Upload
            accept="image/png,image/jpeg"
            maxCount={1}
            showUploadList={false}
            beforeUpload={(file) => {
              const problem = imageFileError(file);
              if (problem) message.error(problem);
              else pickFile(file);
              return false; // it is uploaded when the picture is used
            }}
          >
            <Button>{picked ? "Choose another…" : "Choose a file…"}</Button>
          </Upload>
          {picked && (
            <>
              {/* A local blob: URL, which next/image cannot optimize. */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={picked.preview}
                alt="Chosen starting picture"
                style={{ width: 56, height: 56, objectFit: "cover", borderRadius: 10 }}
              />
              <Typography.Text type="secondary" ellipsis style={{ minWidth: 0, flex: 1 }}>
                {picked.file.name}
              </Typography.Text>
              <Button type="text" size="small" onClick={() => pickFile(null)}>
                Remove
              </Button>
            </>
          )}
        </div>
      )}
      {source === "link" && (
        <Input
          value={link}
          onChange={(e) => setLink(e.target.value)}
          placeholder="https://…"
          status={linkError ? "error" : undefined}
          allowClear
          style={{ marginTop: 8 }}
        />
      )}

      <Typography.Paragraph
        type={linkError ? "danger" : "secondary"}
        style={{ margin: "8px 0 0" }}
      >
        {linkError ??
          (source === "ai"
            ? "Pictures cost credits; you are asked before they are made. Pick one to use."
            : source === "past"
              ? "Reuse a picture this profile's earlier clips started from. Nothing is charged."
              : source === "file"
                ? "A JPG or PNG, at least 300×300 px, up to 8 MB. Nothing is charged."
                : "A public link, at least 300×300 px. Nothing is charged.")}
      </Typography.Paragraph>

      <Button
        type="primary"
        style={{ marginTop: 16 }}
        disabled={!ready}
        loading={busy}
        onClick={use}
      >
        Use this picture
      </Button>
    </>
  );
}
