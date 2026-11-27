"use client";

import { InputNumber, Select } from "antd";

/**
 * A clip length in seconds. A model that takes a continuous range (Kling: 3 to
 * 15) gets a number box; one that takes a few fixed lengths (Veo: 4, 6 or 8)
 * gets a list, so a length the model would refuse cannot be typed.
 */
export default function SecondsField({
  options,
  value,
  onChange,
  width = 90,
}: {
  options: number[];
  value: number;
  onChange: (n: number) => void;
  width?: number;
}) {
  const min = Math.min(...options);
  const max = Math.max(...options);
  if (max - min + 1 === options.length) {
    return (
      <InputNumber
        min={min}
        max={max}
        precision={0}
        value={value}
        onChange={(v) => onChange(typeof v === "number" ? v : value)}
        suffix="s"
        style={{ width }}
      />
    );
  }
  return (
    <Select
      value={value}
      onChange={onChange}
      options={options.map((n) => ({ value: n, label: `${n} s` }))}
      style={{ width: width + 10 }}
    />
  );
}
