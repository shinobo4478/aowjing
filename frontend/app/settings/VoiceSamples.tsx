"use client";

import { useEffect, useState } from "react";
import { Alert, App, Button, Checkbox, Input, Typography } from "antd";
import { ApiError, getVoiceCandidates, makeVoiceSamples } from "@/lib/api";
import { formatUsd } from "@/lib/format";
import type { VoiceCandidates, VoiceSampleResult } from "@/lib/types";

/** A Thai paragraph of about 300 characters to listen with: plain narration with
 *  numbers, a list and a question, which is what educational clips are made of.
 *  The owner can replace it with a real script. */
const DEFAULT_TEXT =
  "สวัสดี วันนี้เรามาเรียนรู้เรื่องง่ายๆ ที่ช่วยให้จำอะไรได้นานขึ้น นั่นคือการทบทวนแบบเว้นระยะ " +
  "หลายคนอ่านหนังสือรวดเดียวก่อนสอบ แต่สมองของเรากลับลืมเร็วกว่าที่คิด " +
  "วิธีที่ได้ผลคือทบทวนซ้ำเมื่อผ่านไปหนึ่งวัน สามวัน และเจ็ดวัน " +
  "ทุกครั้งที่ทบทวน ความจำจะแข็งแรงขึ้น ลองทำดูสักสัปดาห์ แล้วคุณจะรู้สึกได้เลยว่าจำได้แม่นขึ้นมาก";

/** The voices ticked at the start: both MiniMax HD voices, the cheaper model with
 *  one of them (to hear what is lost), and ElevenLabs v3 (whose Thai is the open
 *  question). */
const DEFAULT_PICKS = ["minimax-hd-woman", "minimax-hd-man", "minimax-turbo-woman", "eleven-v3"];

/**
 * Listen before choosing: the SAME Thai text is spoken by each candidate voice.
 * There is no independent Thai ranking, so the choice is made by ear. Costs fal
 * credits per character (a few cents in all), so it asks first. Nothing is saved.
 */
export default function VoiceSamples() {
  const { message, modal } = App.useApp();
  const [info, setInfo] = useState<VoiceCandidates | null>(null);
  const [text, setText] = useState(DEFAULT_TEXT);
  const [picked, setPicked] = useState<string[]>(DEFAULT_PICKS);
  const [busy, setBusy] = useState(false);
  const [results, setResults] = useState<VoiceSampleResult[] | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    let cancelled = false;
    getVoiceCandidates()
      .then((c) => {
        if (!cancelled) setInfo(c);
      })
      .catch((err) =>
        message.error(err instanceof Error ? err.message : "Failed to load the voices."),
      );
    return () => {
      cancelled = true;
    };
  }, [message]);

  const chars = [...text.trim()].length;
  const priceOf = (per1k: number) => (per1k * chars) / 1000;
  const chosen = (info?.candidates ?? []).filter((c) => picked.includes(c.key));
  const total = chosen.reduce((sum, c) => sum + priceOf(c.pricePer1kChars), 0);
  const tooShort = !!info && chars < info.minChars;
  const tooLong = !!info && chars > info.maxChars;

  function toggle(key: string, on: boolean) {
    setPicked((cur) => (on ? [...cur, key] : cur.filter((k) => k !== key)));
  }

  // Speaking costs fal credits per character, so ask first, with the price.
  function confirmMake() {
    modal.confirm({
      title: `Make ${chosen.length} voice sample${chosen.length === 1 ? "" : "s"}?`,
      content: (
        <>
          <p style={{ margin: "0 0 8px" }}>
            {chars} characters × {chosen.length} voice{chosen.length === 1 ? "" : "s"} ≈{" "}
            <strong>{formatUsd(total)}</strong> at fal&apos;s list prices, charged to your fal.ai
            credits. A voice that fails costs nothing.
          </p>
          <p style={{ margin: 0 }}>
            Nothing is saved in the app; fal keeps each audio file for at least 7 days.
          </p>
        </>
      ),
      okText: `Make samples (≈ ${formatUsd(total)})`,
      cancelText: "Cancel",
      onOk: () => make(),
    });
  }

  async function make() {
    setBusy(true);
    setErrors({});
    setResults(null);
    try {
      const r = await makeVoiceSamples({ text, candidates: picked });
      setResults(r.results);
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors) {
        setErrors(err.fieldErrors as Record<string, string>);
      } else {
        message.error(err instanceof Error ? err.message : "The samples failed.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ marginTop: 56, maxWidth: 640 }}>
      <Typography.Title level={4} style={{ marginTop: 0, marginBottom: 8 }}>
        Thai voice samples
      </Typography.Title>
      <Typography.Paragraph type="secondary">
        ฟังเสียงพากย์ไทยก่อนเลือก แต่ละเสียงอ่านข้อความเดียวกัน ยังไม่มีการจัดอันดับเสียงไทยที่เป็นกลาง
        จึงเลือกด้วยหู ถ้าเสียงไหนขึ้นข้อผิดพลาดหรือสำเนียงเพี้ยน ก็ตัดทิ้งได้เลย
        เสียงไม่ถูกบันทึกในแอป (fal เก็บไฟล์อย่างน้อย 7 วัน)
      </Typography.Paragraph>

      <Typography.Text>ข้อความที่ให้อ่าน · Text to speak</Typography.Text>
      <Input.TextArea
        value={text}
        onChange={(e) => setText(e.target.value)}
        autoSize={{ minRows: 5, maxRows: 12 }}
        status={errors.text || tooShort || tooLong ? "error" : undefined}
        style={{ marginTop: 4 }}
      />
      <Typography.Text type={tooShort || tooLong ? "danger" : "secondary"} style={{ fontSize: 12 }}>
        {chars} characters
        {info ? ` (${info.minChars}–${info.maxChars})` : ""}. ใส่บทพากย์จริงของคุณแทนได้
      </Typography.Text>
      {errors.text && (
        <Typography.Text type="danger" style={{ display: "block" }}>
          {errors.text}
        </Typography.Text>
      )}

      <div style={{ marginTop: 20 }}>
        <Typography.Text>เสียง · Voices</Typography.Text>
        <div style={{ display: "flex", flexDirection: "column", gap: 10, marginTop: 6 }}>
          {(info?.candidates ?? []).map((c) => (
            <Checkbox
              key={c.key}
              checked={picked.includes(c.key)}
              disabled={!picked.includes(c.key) && picked.length >= (info?.maxVoices ?? 7)}
              onChange={(e) => toggle(c.key, e.target.checked)}
            >
              <span>
                {c.label}{" "}
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  ≈ {formatUsd(priceOf(c.pricePer1kChars))} for this text
                </Typography.Text>
              </span>
              <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
                {c.note}
              </Typography.Text>
            </Checkbox>
          ))}
        </div>
        {errors.candidates && (
          <Typography.Text type="danger" style={{ display: "block", marginTop: 4 }}>
            {errors.candidates}
          </Typography.Text>
        )}
      </div>

      <div style={{ marginTop: 20, display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
        <Button
          type="primary"
          loading={busy}
          disabled={!info || chosen.length === 0 || tooShort || tooLong}
          onClick={confirmMake}
        >
          Make samples
        </Button>
        {chosen.length > 0 && (
          <Typography.Text type="secondary">
            ≈ {formatUsd(total)} for {chosen.length} voice{chosen.length === 1 ? "" : "s"}
          </Typography.Text>
        )}
        {busy && (
          <Typography.Text type="secondary">
            Speaking… this can take up to a minute or two.
          </Typography.Text>
        )}
      </div>

      {results && (
        <div style={{ marginTop: 24, display: "flex", flexDirection: "column", gap: 16 }}>
          {results.map((r) =>
            r.ok && r.audioUrl ? (
              <div key={r.key}>
                <Typography.Text strong>{r.label}</Typography.Text>{" "}
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {r.durationMs > 0 ? `${(r.durationMs / 1000).toFixed(1)} s · ` : ""}≈ {formatUsd(r.costUsd)}
                </Typography.Text>
                <audio controls preload="none" src={r.audioUrl} style={{ display: "block", width: "100%", marginTop: 4 }} />
              </div>
            ) : (
              <Alert
                key={r.key}
                type="warning"
                showIcon
                title={r.label}
                description={r.error ?? "No audio came back."}
              />
            ),
          )}
        </div>
      )}
    </div>
  );
}
