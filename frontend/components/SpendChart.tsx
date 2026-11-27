import { formatUsd } from "@/lib/format";

/**
 * Daily spend as bars. Plain SVG (no chart package): 30 bars, a baseline, the
 * highest day labelled, first / last dates under it. Each bar carries a
 * <title>, so hovering shows the day, the cost and the number of clips.
 */
export default function SpendChart({
  days,
}: {
  days: { date: string; cost: number; clips: number }[];
}) {
  const W = 600;
  const H = 150;
  const top = 26; // room for the max label
  const bottom = 28; // room for the dates
  const plotH = H - top - bottom;
  const max = Math.max(...days.map((d) => d.cost), 0);
  const slot = W / days.length;
  const bar = slot * 0.66;

  if (max === 0) {
    return (
      <div
        style={{
          height: 150,
          display: "grid",
          placeItems: "center",
          color: "var(--acmp-ash)",
        }}
      >
        No billed spend in the last {days.length} days.
      </div>
    );
  }

  const fmtDay = (iso: string) =>
    new Date(iso + "T00:00:00").toLocaleDateString("en-GB", {
      day: "numeric",
      month: "short",
    });

  return (
    <svg
      viewBox={`0 0 ${W} ${H}`}
      role="img"
      aria-label={
        max > 0
          ? `Daily fal spend over the last ${days.length} days, highest ${formatUsd(max)}`
          : "No billed spend in the last 30 days"
      }
      style={{ width: "100%", height: "auto", display: "block" }}
    >
      <line
        x1={0}
        x2={W}
        y1={top + plotH}
        y2={top + plotH}
        stroke="var(--acmp-rule)"
      />
      {days.map((d, i) => {
        const h = max > 0 ? (d.cost / max) * plotH : 0;
        const x = i * slot + (slot - bar) / 2;
        return (
          <g key={d.date}>
            <title>{`${fmtDay(d.date)}: ${formatUsd(d.cost)} · ${d.clips} ${d.clips === 1 ? "clip" : "clips"}`}</title>
            {/* A full-height invisible hit area so thin or empty days still show a tooltip. */}
            <rect x={i * slot} y={top} width={slot} height={plotH} fill="transparent" />
            {d.cost > 0 && (
              <rect
                x={x}
                y={top + plotH - Math.max(h, 2)}
                width={bar}
                height={Math.max(h, 2)}
                rx={2}
                fill="var(--acmp-tungsten)"
              />
            )}
          </g>
        );
      })}
      {max > 0 && (
        <text x={0} y={16} fontSize={16} fill="var(--acmp-ash)">
          {formatUsd(max)} / day at most
        </text>
      )}
      <text x={0} y={H - 6} fontSize={16} fill="var(--acmp-ash)">
        {fmtDay(days[0].date)}
      </text>
      <text x={W} y={H - 6} fontSize={16} fill="var(--acmp-ash)" textAnchor="end">
        {fmtDay(days[days.length - 1].date)}
      </text>
    </svg>
  );
}
