"use client";

import { useEffect, useState } from "react";
import { Input, Select } from "antd";
import { getVoiceCandidates } from "@/lib/api";
import type { VoiceCandidate } from "@/lib/types";

const NONE = "";
const OTHER = "__other__";
const CUSTOM_PREFIX = "custom:";

/**
 * The narrator voice of a profile, for an Ant Design Form.Item: `value` is "" (none),
 * a voice key from the list, or "custom:<name>" for a voice the owner named (a name
 * from fal's examples, or an ElevenLabs voice id). The list is the voices the owner
 * kept after listening (Settings → Thai voice samples), and the choice can be
 * changed at any time: shots already spoken keep their audio until spoken again.
 */
export default function NarratorField({
  value = "",
  onChange,
}: {
  value?: string;
  onChange?: (v: string) => void;
}) {
  const [candidates, setCandidates] = useState<VoiceCandidate[]>([]);
  useEffect(() => {
    let cancelled = false;
    getVoiceCandidates()
      .then((c) => {
        if (!cancelled) setCandidates(c.candidates);
      })
      .catch(() => {
        /* the list stays empty: only "none" and "other" are offered */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const isCustom = value.startsWith(CUSTOM_PREFIX);
  // A stored voice that is no longer in the list (it was dropped) still shows,
  // so the form does not silently change it.
  const known = value === NONE || isCustom || candidates.some((c) => c.key === value);
  const selectValue = isCustom ? OTHER : value;

  return (
    <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
      <Select
        value={selectValue}
        onChange={(v) => onChange?.(v === OTHER ? CUSTOM_PREFIX : v)}
        style={{ minWidth: 260 }}
        options={[
          { value: NONE, label: "ยังไม่เลือก · None" },
          ...candidates.map((c) => ({ value: c.key, label: c.label })),
          ...(known ? [] : [{ value, label: `${value} (no longer in the list)` }]),
          { value: OTHER, label: "เสียงอื่น · Other voice…" },
        ]}
      />
      {isCustom && (
        <Input
          value={value.slice(CUSTOM_PREFIX.length)}
          onChange={(e) => onChange?.(CUSTOM_PREFIX + e.target.value.trim())}
          placeholder="ชื่อหรือ ID ของเสียง ElevenLabs"
          maxLength={64}
          style={{ maxWidth: 300 }}
        />
      )}
    </div>
  );
}
