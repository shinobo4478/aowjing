import { Button, Space, Typography } from "antd";
import type { Placement, YouTubeUpload } from "@/lib/types";

/**
 * What's left to do by hand after publishing a clip that has pinned products:
 * tag them on the video in YouTube Studio. The description already carries the
 * product links; this is the shopping tag itself. Copy gives you the plain
 * Shopee page to search or paste in Studio's product picker.
 */
export default function StudioChecklist({
  uploads,
  pinned,
  onCopy,
}: {
  uploads: YouTubeUpload[];
  /** In pin order. */
  pinned: Placement[];
  onCopy: (text: string) => void;
}) {
  return (
    <div style={{ display: "grid", gap: 6, margin: "6px 0" }}>
      <Typography.Text type="secondary">Tag in YouTube Studio</Typography.Text>
      <Typography.Text type="secondary" style={{ fontSize: 13, maxWidth: "62ch" }}>
        The links are already in the description. To show the shopping tag on
        the video, open it in Studio and add each product below.
      </Typography.Text>
      {uploads.map((u) => (
        <a
          key={u.id}
          href={`https://studio.youtube.com/video/${u.youtubeVideoId}/edit`}
          target="_blank"
          rel="noreferrer"
        >
          Open &ldquo;{u.title}&rdquo; in Studio
        </a>
      ))}
      {pinned.map((p) => (
        <Space key={p.product.id} wrap>
          <span>
            {p.position}. {p.product.name}
          </span>
          <Button
            size="small"
            type="text"
            onClick={() => onCopy(p.product.shopeeUrl)}
          >
            Copy Shopee link
          </Button>
        </Space>
      ))}
    </div>
  );
}
