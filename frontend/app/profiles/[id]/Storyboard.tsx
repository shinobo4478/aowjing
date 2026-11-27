"use client";

import { Button, Checkbox, Typography } from "antd";
import { imageSrc } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import type { Shot } from "@/lib/types";

/** A shot that would be made by "Generate approved": approved, not already made
 *  and not being made now. */
export function isReadyToMake(s: Shot): boolean {
  return s.approved && s.latest?.status !== "succeeded" && s.latest?.status !== "pending";
}

/**
 * The video's pictures side by side, in order — a contact sheet. Look at the
 * whole sequence here (it costs cents) and approve the shots that are right;
 * then make all the approved ones with one confirmation, instead of paying for
 * each clip to find out the sequence does not hold together.
 */
export default function Storyboard({
  shots,
  perSecond,
  busy,
  onApprove,
  onMakeApproved,
}: {
  shots: Shot[];
  perSecond: number | undefined;
  busy: boolean;
  onApprove: (shot: Shot, approved: boolean) => void;
  onMakeApproved: () => void;
}) {
  const ready = shots.filter(isReadyToMake);
  const readySeconds = ready.reduce((n, s) => n + s.seconds, 0);
  const readyPrice = perSecond === undefined ? undefined : perSecond * readySeconds;

  return (
    <div style={{ margin: "8px 0 20px" }}>
      <Typography.Text strong>Storyboard</Typography.Text>
      <div style={{ display: "flex", gap: 12, overflowX: "auto", padding: "8px 0" }}>
        {shots.map((s) => {
          const hasPicture = s.imageId !== "" || s.imageUrl !== "";
          const canApprove = s.prompt.trim() !== "" && hasPicture;
          return (
            <div key={s.id} style={{ flex: "0 0 auto", width: 84 }}>
              <button
                type="button"
                aria-label={`Go to shot ${s.position}`}
                onClick={() =>
                  document
                    .getElementById(`shot-${s.id}`)
                    ?.scrollIntoView({ behavior: "smooth", block: "start" })
                }
                style={{
                  padding: 0,
                  background: "none",
                  border: s.approved ? "2px solid var(--acmp-succeeded)" : "2px solid transparent",
                  borderRadius: 5,
                  cursor: "pointer",
                  display: "block",
                }}
              >
                {hasPicture ? (
                  // Served by the API with the session cookie (or a pasted link),
                  // so next/image cannot optimize it.
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={s.imageId ? imageSrc(s.imageId) : s.imageUrl}
                    alt={`Shot ${s.position}`}
                    style={{ width: 80, height: 142, objectFit: "cover", display: "block", borderRadius: 3 }}
                  />
                ) : (
                  <div
                    style={{
                      width: 80,
                      height: 142,
                      border: "1px dashed var(--acmp-dashed)",
                      borderRadius: 3,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  >
                    <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                      no picture
                    </Typography.Text>
                  </div>
                )}
              </button>
              <div style={{ fontSize: 12, marginTop: 2 }}>
                {s.position} · {s.seconds}s
                {s.endImageId !== "" && " · →end"}
                {s.latest?.status === "succeeded" && " · made"}
                {s.latest?.status === "pending" && " · …"}
              </div>
              <Checkbox
                checked={s.approved}
                disabled={busy || (!canApprove && !s.approved)}
                onChange={(e) => onApprove(s, e.target.checked)}
              >
                <span style={{ fontSize: 12 }}>OK</span>
              </Checkbox>
            </div>
          );
        })}
      </div>

      <Button
        type="primary"
        disabled={busy || ready.length === 0}
        onClick={onMakeApproved}
      >
        {ready.length === 0
          ? "Generate approved shots"
          : `Generate ${ready.length} approved shot${ready.length === 1 ? "" : "s"} (${readySeconds} s${
              readyPrice !== undefined ? `, ≈ ${formatUsd(readyPrice)}` : ""
            })`}
      </Button>
      <Typography.Text type="secondary" style={{ marginLeft: 12, fontSize: 12 }}>
        Tick OK on the shots whose picture and prompt are right. Changing a shot
        clears its OK.
      </Typography.Text>
    </div>
  );
}
