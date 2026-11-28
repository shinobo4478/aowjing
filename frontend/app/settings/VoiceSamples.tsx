"use client";

import { useEffect, useState } from "react";
import { Alert, App, Button, Checkbox, Input, Tag, Typography } from "antd";
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

/** The voices ticked at the start: Rachel (heard 2026-11-28: no mistakes, accent OK) and
 *  a woman and two men beside her, to choose a narrator by ear. MiniMax read Thai with
 *  many mistakes and is no longer offered. */
const DEFAULT_PICKS = ["eleven-v3-rachel", "eleven-v3-sarah", "eleven-v3-roger", "eleven-v3-daniel"];

/** Names from fal's own examples that are not in the list above, offered as one
 *  click to try. Whether each reads Thai well is for the ear to say (Aria and
 *  Brian read Thai but not well, which is why they are not listed). */
const SUGGESTED_NAMES = ["Alice", "Matilda", "Jessica", "Lily", "Laura", "Eric", "Will", "Liam", "Chris", "Bill", "Charlie", "Callum", "River"];

/** ElevenLabs v3 on fal, per 1000 characters. Mirrors studio.CustomVoice; the
 *  server's own figure is the one charged. */
const CUSTOM_PRICE_PER_1K = 0.1;

const NOTES_KEY = "acmp.voiceNotes";

/** What the owner wrote about each voice, kept in this browser only. */
function loadNotes(): Record<string, string> {
  try {
    return JSON.parse(localStorage.getItem(NOTES_KEY) ?? "{}") as Record<string, string>;
  } catch {
    return {};
  }
}

function saveNotes(notes: Record<string, string>) {
  try {
    localStorage.setItem(NOTES_KEY, JSON.stringify(notes));
  } catch {
    /* private window or blocked storage: the notes just are not kept */
  }
}

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
  // Voices the owner typed in (a name or an ElevenLabs voice id), and what they
  // wrote about each voice they heard. Notes live in this browser only.
  const [custom, setCustom] = useState<string[]>([]);
  const [customDraft, setCustomDraft] = useState("");
  const [notes, setNotes] = useState<Record<string, string>>({});

  useEffect(() => {
    // Read once on the client (localStorage does not exist on the server).
    setNotes(loadNotes()); // eslint-disable-line react-hooks/set-state-in-effect
  }, []);

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
  const voiceCount = chosen.length + custom.length;
  const total =
    chosen.reduce((sum, c) => sum + priceOf(c.pricePer1kChars), 0) +
    custom.length * priceOf(CUSTOM_PRICE_PER_1K);
  const tooShort = !!info && chars < info.minChars;
  const tooLong = !!info && chars > info.maxChars;

  // Adds a voice the owner named (once; the server checks the characters too).
  function addCustom(raw: string) {
    const name = raw.trim();
    if (name === "") return;
    setCustom((cur) => (cur.includes(name) ? cur : [...cur, name]));
    setCustomDraft("");
  }

  function setNote(key: string, text: string) {
    setNotes((cur) => {
      const next = { ...cur, [key]: text };
      saveNotes(next);
      return next;
    });
  }

  function toggle(key: string, on: boolean) {
    setPicked((cur) => (on ? [...cur, key] : cur.filter((k) => k !== key)));
  }

  // Speaking costs fal credits per character, so ask first, with the price.
  function confirmMake() {
    modal.confirm({
      title: `Make ${voiceCount} voice sample${voiceCount === 1 ? "" : "s"}?`,
      content: (
        <>
          <p style={{ margin: "0 0 8px" }}>
            {chars} characters × {voiceCount} voice{voiceCount === 1 ? "" : "s"} ≈{" "}
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
      const r = await makeVoiceSamples({ text, candidates: picked, customVoices: custom });
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
              disabled={!picked.includes(c.key) && picked.length + custom.length >= (info?.maxVoices ?? 12)}
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

      <div style={{ marginTop: 20 }}>
        <Typography.Text>ลองเสียงอื่น · Try another voice</Typography.Text>
        <Typography.Paragraph type="secondary" style={{ fontSize: 12, margin: "2px 0 8px" }}>
          พิมพ์ชื่อหรือ ID ของเสียง ElevenLabs (อ่านด้วยรุ่น v3 เหมือนกัน) หรือกดชื่อด้านล่าง
          เสียงมาตรฐาน (Default) ของ ElevenLabs จะหมดอายุ 31 ธ.ค. 2026 ตามเอกสารของเขา และถูกแทนด้วยเสียง
          “Infinity” (มีสัญลักษณ์ ∞ ใน Voice Library) ที่เอกสารไม่บอกชื่อหรือ ID ถ้าคุณมีเสียงแบบนั้นในบัญชี
          ลองใส่ ID ที่นี่เพื่อฟังว่า fal รับหรือไม่ (ไม่รับ = ขึ้นข้อผิดพลาด ไม่เสียเงิน)
        </Typography.Paragraph>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <Input
            value={customDraft}
            onChange={(e) => setCustomDraft(e.target.value)}
            onPressEnter={() => addCustom(customDraft)}
            placeholder="เช่น Alice หรือ ID ของเสียง"
            maxLength={64}
            style={{ maxWidth: 320 }}
            status={errors.customVoices ? "error" : undefined}
          />
          <Button onClick={() => addCustom(customDraft)} disabled={customDraft.trim() === ""}>
            เพิ่ม
          </Button>
        </div>
        {errors.customVoices && (
          <Typography.Text type="danger" style={{ display: "block", marginTop: 4 }}>
            {errors.customVoices}
          </Typography.Text>
        )}
        {custom.length > 0 && (
          <div style={{ marginTop: 8, display: "flex", gap: 6, flexWrap: "wrap" }}>
            {custom.map((n) => (
              <Tag key={n} closable onClose={() => setCustom((cur) => cur.filter((x) => x !== n))}>
                {n} · ≈ {formatUsd(priceOf(CUSTOM_PRICE_PER_1K))}
              </Tag>
            ))}
          </div>
        )}
        <div style={{ marginTop: 8, display: "flex", gap: 6, flexWrap: "wrap" }}>
          {SUGGESTED_NAMES.filter((n) => !custom.includes(n)).map((n) => (
            <Tag key={n} style={{ cursor: "pointer" }} onClick={() => addCustom(n)}>
              + {n}
            </Tag>
          ))}
        </div>
      </div>

      <div style={{ marginTop: 20, display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
        <Button
          type="primary"
          loading={busy}
          disabled={!info || voiceCount === 0 || tooShort || tooLong}
          onClick={confirmMake}
        >
          Make samples
        </Button>
        {voiceCount > 0 && (
          <Typography.Text type="secondary">
            ≈ {formatUsd(total)} for {voiceCount} voice{voiceCount === 1 ? "" : "s"}
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
                <Input
                  size="small"
                  value={notes[r.key] ?? ""}
                  onChange={(e) => setNote(r.key, e.target.value)}
                  placeholder="โน้ตของคุณเกี่ยวกับเสียงนี้ (เก็บในเบราว์เซอร์นี้เท่านั้น)"
                  maxLength={200}
                  style={{ marginTop: 6 }}
                />
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
