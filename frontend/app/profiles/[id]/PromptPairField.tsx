"use client";

import { App, Button, Collapse, Input, Popconfirm, Typography } from "antd";
import { formatUsd } from "@/lib/format";
import { lintPrompt, type LintKind } from "@/lib/promptLint";
import { hasThaiLetters, type PromptPair } from "@/lib/promptPair";

const shorten = (s: string, n: number) => {
  const flat = s.replace(/\s+/g, " ").trim();
  return flat.length > n ? `${flat.slice(0, n - 1)}…` : flat;
};

/**
 * One prompt of a shot, written in Thai: a Thai box (what the owner edits) and,
 * folded under it, the English that the picture and video models receive, with
 * buttons to translate either way. The English box shows a preview even when
 * folded, so what will be sent is never hidden.
 */
export default function PromptPairField({
  pair,
  kind,
  title,
  hint,
  placeholder,
  minRows = 2,
  maxRows = 8,
}: {
  pair: PromptPair;
  /** What the prompt is for: the English is checked against that kind's rules
   *  (lib/promptLint.ts) and slips are shown as warnings. */
  kind?: LintKind;
  /** What the prompt is, in a few words. */
  title: string;
  /** One line on what to write in it. */
  hint?: string;
  placeholder: string;
  minRows?: number;
  maxRows?: number;
}) {
  const { message } = App.useApp();

  async function run(fn: () => Promise<unknown>, fallback: string) {
    try {
      await fn();
    } catch (err) {
      message.error(err instanceof Error ? err.message : fallback);
    }
  }

  const englishHasThai = hasThaiLetters(pair.en);
  // Warnings on the English. Skipped while the Thai is ahead of it (the English is
  // about to be translated again, so what it says now does not matter).
  const issues = kind && !pair.stale ? lintPrompt(kind, pair.en) : [];

  return (
    <div style={{ marginTop: 12 }}>
      <Typography.Text strong style={{ fontSize: 13 }}>
        {title}
      </Typography.Text>
      {hint && (
        <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
          {hint}
        </Typography.Text>
      )}
      <Input.TextArea
        value={pair.th}
        onChange={(e) => pair.setTh(e.target.value)}
        autoSize={{ minRows, maxRows }}
        maxLength={2000}
        placeholder={placeholder}
        style={{ marginTop: 4 }}
      />

      <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap", margin: "6px 0" }}>
        <Button
          size="small"
          loading={pair.busy}
          disabled={pair.busy || pair.th.trim() === ""}
          onClick={() => void run(pair.toEnglish, "The translation failed.")}
        >
          แปลเป็นอังกฤษ
        </Button>
        {pair.en.trim() !== "" &&
          (pair.th.trim() === "" ? (
            <Button
              size="small"
              type="text"
              disabled={pair.busy}
              onClick={() => void run(pair.toThai, "The translation failed.")}
            >
              ดูความหมายเป็นไทย
            </Button>
          ) : (
            <Popconfirm
              title="แทนที่ข้อความไทยด้วยคำแปลจากอังกฤษ?"
              description="ข้อความไทยที่คุณพิมพ์จะถูกเขียนทับ"
              okText="แทนที่"
              cancelText="ยกเลิก"
              onConfirm={() => void run(pair.toThai, "The translation failed.")}
            >
              <Button size="small" type="text" disabled={pair.busy}>
                ดูความหมายเป็นไทย
              </Button>
            </Popconfirm>
          ))}
        {pair.stale && (
          <Typography.Text type="warning" style={{ fontSize: 12 }}>
            ยังไม่ได้แปลหลังแก้ไข (จะแปลให้อัตโนมัติเมื่อบันทึกหรือสร้าง)
          </Typography.Text>
        )}
        {!pair.stale && pair.lastCostUsd !== null && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            แปลแล้ว ≈ {formatUsd(pair.lastCostUsd)}
          </Typography.Text>
        )}
      </div>

      {issues.map((issue) => (
        <div key={issue.id} style={{ fontSize: 12, margin: "0 0 4px" }}>
          <Typography.Text type="warning" style={{ fontSize: 12 }}>
            ⚠ {issue.message}
          </Typography.Text>
          {issue.fix && (
            <Button
              size="small"
              type="link"
              onClick={() => pair.setEn(issue.fix!(pair.en))}
              style={{ fontSize: 12 }}
            >
              {issue.fixLabel ?? "แก้ให้"}
            </Button>
          )}
        </div>
      ))}

      <Collapse
        size="small"
        ghost
        items={[
          {
            key: "en",
            label: (
              <span style={{ fontSize: 12 }}>
                English sent to the AI ·{" "}
                <Typography.Text type={englishHasThai ? "danger" : "secondary"} style={{ fontSize: 12 }}>
                  {pair.en.trim() === "" ? "(none yet)" : shorten(pair.en, 80)}
                </Typography.Text>
              </span>
            ),
            children: (
              <>
                <Input.TextArea
                  value={pair.en}
                  onChange={(e) => pair.setEn(e.target.value)}
                  autoSize={{ minRows: 2, maxRows: 10 }}
                  maxLength={2000}
                  status={englishHasThai ? "error" : undefined}
                  placeholder="The English the picture and video models receive."
                />
                {englishHasThai && (
                  <Typography.Text type="danger" style={{ fontSize: 12 }}>
                    กล่องนี้ควรเป็นภาษาอังกฤษ: กด &quot;แปลเป็นอังกฤษ&quot; ที่ช่องไทยด้านบน
                  </Typography.Text>
                )}
              </>
            ),
          },
        ]}
      />
    </div>
  );
}
