"use client";

import { useEffect, useState } from "react";
import { App, Spin, Tooltip, Typography } from "antd";
import { imageSrc, listGenerations } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import type { Generation } from "@/lib/types";

/**
 * Pictures this profile has already used as a clip's starting image, newest
 * first, so one can be used again — to try another prompt on the same picture —
 * without paying for a new one. Only this profile's own clips are listed.
 *
 * `value` is the chosen picture's id ("" = none yet).
 */
export default function PastPictures({
  profileId,
  value,
  onChange,
}: {
  profileId: string;
  value: string;
  onChange: (imageId: string) => void;
}) {
  const { message } = App.useApp();
  const [clips, setClips] = useState<Generation[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    listGenerations(profileId)
      .then((all) => {
        if (cancelled) return;
        // The list is newest first; keep the first clip of each picture.
        const seen = new Set<string>();
        setClips(
          all.filter((g) => {
            if (!g.imageId || seen.has(g.imageId)) return false;
            seen.add(g.imageId);
            return true;
          }),
        );
      })
      .catch((err) => {
        message.error(
          err instanceof Error ? err.message : "Failed to load earlier clips.",
        );
        if (!cancelled) setClips([]);
      });
    return () => {
      cancelled = true;
    };
  }, [profileId, message]);

  if (clips === null) {
    return (
      <div style={{ padding: 16, textAlign: "center" }}>
        <Spin />
      </div>
    );
  }

  if (clips.length === 0) {
    return (
      <Typography.Paragraph type="secondary" style={{ margin: "8px 0 0" }}>
        No earlier clip of this profile has a picture yet. Make one with AI, or
        upload one from your computer.
      </Typography.Paragraph>
    );
  }

  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 8 }}>
      {clips.map((g) => {
        const selected = g.imageId === value;
        return (
          <Tooltip
            key={g.imageId}
            title={`${g.templateName || "Deleted template"} · ${formatDateTime(g.createdAt)}`}
          >
            <button
              type="button"
              onClick={() => onChange(g.imageId)}
              aria-pressed={selected}
              aria-label={selected ? "Chosen picture" : "Use this picture"}
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
                src={imageSrc(g.imageId)}
                alt="Picture from an earlier clip"
                style={{ width: 96, height: 170, objectFit: "cover", display: "block", borderRadius: 3 }}
              />
            </button>
          </Tooltip>
        );
      })}
    </div>
  );
}
