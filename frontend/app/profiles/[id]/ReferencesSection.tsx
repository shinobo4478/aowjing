"use client";

import { useCallback, useEffect, useState } from "react";
import { Alert, App, Button, Empty, Popconfirm, Typography, Upload } from "antd";
import {
  addReference,
  ApiError,
  imageSrc,
  listReferences,
  MAX_REFERENCE_BYTES,
  removeReference,
} from "@/lib/api";
import type { ProviderKey, ReferencePicture } from "@/lib/types";

/** The most reference pictures a profile keeps. Mirrors studio.MaxReferences. */
const MAX_REFERENCES = 4;

/** Checks a picked file before uploading it; null = fine. The backend checks
 *  the same things, plus the pixel size. */
function referenceFileError(file: File): string | null {
  if (file.type !== "image/jpeg" && file.type !== "image/png") {
    return "Use a JPG or PNG image.";
  }
  if (file.size > MAX_REFERENCE_BYTES) {
    return `The picture is too large; a reference picture can be at most ${MAX_REFERENCE_BYTES / 1024 / 1024} MB.`;
  }
  return null;
}

/**
 * The pictures that give this profile its own look — its character, mascot or
 * style. Whenever the profile makes a keyframe picture, they go to the image
 * model so the new picture keeps the same look.
 */
export default function ReferencesSection({
  profileId,
  provider,
}: {
  profileId: string;
  provider: ProviderKey;
}) {
  const { message } = App.useApp();
  const [refs, setRefs] = useState<ReferencePicture[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);

  const load = useCallback(() => {
    listReferences(profileId)
      .then(setRefs)
      .catch((err) =>
        message.error(
          err instanceof Error ? err.message : "Failed to load the pictures.",
        ),
      )
      .finally(() => setLoading(false));
  }, [profileId, message]);

  useEffect(() => {
    load();
  }, [load]);

  async function upload(file: File) {
    const problem = referenceFileError(file);
    if (problem) {
      message.error(problem);
      return;
    }
    setUploading(true);
    try {
      await addReference(profileId, file);
      load();
    } catch (err) {
      const field =
        err instanceof ApiError && err.fieldErrors
          ? Object.values(err.fieldErrors)[0]
          : undefined;
      message.error(
        field ?? (err instanceof Error ? err.message : "Upload failed."),
      );
    } finally {
      setUploading(false);
    }
  }

  async function remove(imageId: string) {
    try {
      await removeReference(profileId, imageId);
      setRefs((rs) => rs.filter((r) => r.imageId !== imageId));
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Failed to remove.");
    }
  }

  const full = refs.length >= MAX_REFERENCES;

  return (
    <div>
      <Typography.Paragraph type="secondary" style={{ maxWidth: 640 }}>
        Pictures of this profile&apos;s character, mascot or style. When you make a
        starting picture for a clip, they are sent to the image model so the
        result keeps the same look. Up to {MAX_REFERENCES} pictures, JPG or PNG,
        at least 300×300 px, up to {MAX_REFERENCE_BYTES / 1024 / 1024} MB each.
      </Typography.Paragraph>

      {provider !== "fal" && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16, maxWidth: 640 }}
          title="This profile uses the text generator, so these pictures are not used yet. Switch its generator to fal.ai video to make pictures and clips."
        />
      )}

      {!loading && refs.length === 0 ? (
        <Empty description="No reference pictures yet" style={{ margin: "24px 0" }} />
      ) : (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 16, margin: "16px 0" }}>
          {refs.map((r) => (
            <div key={r.imageId} style={{ width: 132 }}>
              {/* Served by the API with the session cookie, so next/image cannot optimize it. */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={imageSrc(r.imageId)}
                alt={`Reference picture ${r.position}`}
                style={{
                  width: 132,
                  height: 176,
                  objectFit: "cover",
                  borderRadius: 10,
                  display: "block",
                }}
              />
              <Popconfirm
                title="Remove this picture?"
                okText="Remove"
                okButtonProps={{ danger: true }}
                onConfirm={() => remove(r.imageId)}
              >
                <Button type="text" size="small" danger style={{ marginTop: 4 }}>
                  Remove
                </Button>
              </Popconfirm>
            </div>
          ))}
        </div>
      )}

      <Upload
        accept="image/png,image/jpeg"
        showUploadList={false}
        disabled={full || uploading}
        beforeUpload={(file) => {
          void upload(file);
          return false; // we upload ourselves, not antd
        }}
      >
        <Button disabled={full} loading={uploading}>
          {full ? `All ${MAX_REFERENCES} slots used` : "Add a picture…"}
        </Button>
      </Upload>
      <Typography.Text type="secondary" style={{ marginLeft: 12 }}>
        {refs.length} of {MAX_REFERENCES}
      </Typography.Text>
    </div>
  );
}
