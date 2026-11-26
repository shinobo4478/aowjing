"use client";

import { App, Button, Typography } from "antd";
import { formatDateTime } from "@/lib/format";
import type { Shot } from "@/lib/types";

/**
 * The clips to take into the editor, in the video's order: for each shot the
 * take in use (the one chosen, else the newest successful). A fal link is kept
 * at least 7 days, so download what is wanted soon. Shots not made yet are
 * listed so a gap is visible.
 */
export default function ClipList({ shots }: { shots: Shot[] }) {
  const { message } = App.useApp();
  const ready = shots.filter((s) => s.take?.output);
  const totalSeconds = ready.reduce((n, s) => n + (s.take?.seconds ?? 0), 0);

  async function copyLinks() {
    const text = ready.map((s) => s.take!.output).join("\n");
    try {
      await navigator.clipboard.writeText(text);
      message.success("Links copied, one per line, in order.");
    } catch {
      message.error("Couldn't copy — your browser blocked clipboard access.");
    }
  }

  return (
    <div style={{ marginTop: 24 }}>
      <Typography.Text strong>Clips for your editor</Typography.Text>
      <Typography.Paragraph type="secondary" style={{ margin: "4px 0 8px" }}>
        {ready.length} of {shots.length} shot{shots.length === 1 ? "" : "s"} made
        {ready.length > 0 ? ` · ${totalSeconds} seconds` : ""}. fal keeps each file
        at least 7 days: open a clip and save it before then.
      </Typography.Paragraph>

      <ol style={{ margin: 0, paddingLeft: 20 }}>
        {shots.map((s) => (
          <li key={s.id} style={{ margin: "4px 0" }}>
            Shot {s.position} · {s.seconds} s ·{" "}
            {s.take?.output ? (
              <>
                <a href={s.take.output} target="_blank" rel="noreferrer">
                  Open clip
                </a>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {" "}
                  ({s.takeChosen ? "your chosen take" : "newest take"},{" "}
                  {formatDateTime(s.take.createdAt)})
                </Typography.Text>
              </>
            ) : (
              <Typography.Text type="secondary">not made yet</Typography.Text>
            )}
          </li>
        ))}
      </ol>

      <Button size="small" style={{ marginTop: 8 }} disabled={ready.length === 0} onClick={() => void copyLinks()}>
        Copy all links
      </Button>
    </div>
  );
}
