"use client";

import { useState } from "react";
import { App, Button, Modal, Tag, Typography } from "antd";
import { reviewVideo, type ReviewFinding } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import { mobileModal } from "@/lib/ui";

/** What each kind of finding means, in a few Thai words. */
const KIND_LABEL: Record<string, string> = {
  "picture-shows-action": "ภาพแสดงการกระทำไปแล้ว",
  "too-many-beats": "การกระทำเยอะเกินความยาวช็อต",
  "stray-object": "มีสิ่งของที่ชนกับภาพ",
  "picture-vs-narration": "ภาพไม่ตรงกับบทพากย์",
  chain: "ภาพจบไม่ต่อกับช็อตถัดไป",
  camera: "คำสั่งกล้องขัดกัน",
  character: "ตัวละครไม่คงที่",
  "narration-loanword": "บทขึ้นต้นด้วยคำทับศัพท์",
  "narration-claim": "ข้อความที่ควรตรวจข้อเท็จจริง",
};

/**
 * "ตรวจด้วย AI": a strict reviewer reads the whole drafted video and lists what it
 * is sure is wrong (the kinds of slip a pattern check cannot see). It changes
 * nothing. A finding a rewrite can fix has "แก้ช็อตนี้ด้วย AI", which opens that
 * shot's rewrite dialog with the suggestion filled in (the owner still reads it
 * and decides); narration findings are for the owner to edit.
 */
export default function ReviewPanel({
  videoId,
  shotCount,
  disabled,
  onFix,
}: {
  videoId: string;
  shotCount: number;
  disabled: boolean;
  /** Opens the rewrite dialog of a shot with an instruction already typed. */
  onFix: (shotId: string, instruction: string) => void;
}) {
  const { message } = App.useApp();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<{ findings: ReviewFinding[]; costUsd: number } | null>(null);

  async function run() {
    setBusy(true);
    try {
      const r = await reviewVideo(videoId);
      setResult({ findings: r.findings, costUsd: r.costUsd });
      setOpen(true);
    } catch (err) {
      message.error(err instanceof Error ? err.message : "The review failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap", margin: "16px 0" }}>
        <Button loading={busy} disabled={disabled || busy || shotCount === 0} onClick={() => void run()}>
          🔍 ตรวจด้วย AI
        </Button>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          AI อ่านทุกช็อตแล้วบอกจุดที่น่าจะผิด (ภาพแสดงการกระทำไปแล้ว, สิ่งของที่ชนกัน, บทขึ้นต้นด้วยคำทับศัพท์ …) ไม่แก้อะไรเอง
          · ไม่กี่เซ็นต์ที่สุด
        </Typography.Text>
      </div>

      <Modal
        title="ผลตรวจด้วย AI"
        open={open}
        onCancel={() => setOpen(false)}
        footer={null}
        width={680}
        destroyOnHidden
        {...mobileModal}
      >
        {result && result.findings.length === 0 && (
          <Typography.Paragraph>
            AI ไม่พบปัญหาที่มั่นใจ (ไม่ได้แปลว่าไม่มีเลย ดูภาพและอ่านบทเองอีกครั้งก่อนสร้างคลิป)
          </Typography.Paragraph>
        )}
        {result?.findings.map((f, i) => (
          <div
            key={`${f.shotId}-${f.kind}-${i}`}
            style={{ border: "1px solid var(--acmp-rule)", borderRadius: 12, padding: 12, marginBottom: 8 }}
          >
            <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
              <Typography.Text strong>Shot {f.shot}</Typography.Text>
              <Tag color={f.fixTh ? "gold" : "default"}>{KIND_LABEL[f.kind] ?? f.kind}</Tag>
            </div>
            <Typography.Paragraph style={{ margin: "6px 0" }}>{f.messageTh}</Typography.Paragraph>
            {f.fixTh ? (
              <>
                <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
                  ที่ AI เสนอ: {f.fixTh}
                </Typography.Text>
                <Button
                  size="small"
                  style={{ marginTop: 6 }}
                  onClick={() => {
                    setOpen(false);
                    onFix(f.shotId, f.fixTh);
                  }}
                >
                  แก้ช็อตนี้ด้วย AI
                </Button>
              </>
            ) : (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                แก้ที่ช่อง “บทพากย์” ของช็อตนี้เอง
              </Typography.Text>
            )}
          </div>
        ))}
        {result && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            ค่าตรวจครั้งนี้ ≈ {formatUsd(result.costUsd)} · ผลเป็นความเห็นของ AI อาจพลาดได้
          </Typography.Text>
        )}
      </Modal>
    </>
  );
}
