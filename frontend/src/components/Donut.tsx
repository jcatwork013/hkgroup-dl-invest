"use client";

import { ReactNode } from "react";

export type DonutSegment = {
  label: string;
  value: number;
  color: string;
};

// Theme-aligned palette (gold-forward) for chart slices.
export const DONUT_PALETTE = [
  "#c9a24a", // gold-500
  "#e3c987", // gold-300
  "#7fa37a", // brand/forest green
  "#b8743f", // amber
  "#6b8cae", // muted blue
  "#9c6b9e", // muted purple
  "#5c8a8a", // teal
  "#b5563f", // brick
];

const MUTED = "rgba(255,255,255,0.10)";

// Donut renders a dependency-free SVG ring chart. Pass segments (value need not sum to 100 — it is
// normalised). centerLabel/centerSub render in the hole. Slices < 0 are ignored.
export function Donut({
  segments,
  size = 168,
  thickness = 20,
  centerLabel,
  centerSub,
}: {
  segments: DonutSegment[];
  size?: number;
  thickness?: number;
  centerLabel?: ReactNode;
  centerSub?: ReactNode;
}) {
  const clean = segments.filter((s) => s.value > 0);
  const total = clean.reduce((a, s) => a + s.value, 0);
  const r = (size - thickness) / 2;
  const c = 2 * Math.PI * r;
  const cx = size / 2;

  let offset = 0;
  const arcs =
    total > 0
      ? clean.map((s, i) => {
          const frac = s.value / total;
          const dash = frac * c;
          const el = (
            <circle
              key={i}
              cx={cx}
              cy={cx}
              r={r}
              fill="none"
              stroke={s.color}
              strokeWidth={thickness}
              strokeDasharray={`${dash} ${c - dash}`}
              strokeDashoffset={-offset}
              strokeLinecap="butt"
            />
          );
          offset += dash;
          return el;
        })
      : null;

  return (
    <div className="relative inline-flex items-center justify-center">
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        className="-rotate-90"
      >
        {/* track */}
        <circle
          cx={cx}
          cy={cx}
          r={r}
          fill="none"
          stroke={MUTED}
          strokeWidth={thickness}
        />
        {arcs}
      </svg>
      {(centerLabel || centerSub) && (
        <div className="absolute inset-0 flex flex-col items-center justify-center text-center">
          {centerLabel && (
            <span className="font-serif text-2xl font-semibold text-cream">
              {centerLabel}
            </span>
          )}
          {centerSub && (
            <span className="text-xs text-cream/55">{centerSub}</span>
          )}
        </div>
      )}
    </div>
  );
}

// Legend renders a compact key for the segments.
export function DonutLegend({
  segments,
  format,
}: {
  segments: DonutSegment[];
  format?: (v: number) => string;
}) {
  return (
    <ul className="space-y-2 text-sm">
      {segments.map((s, i) => (
        <li key={i} className="flex items-center justify-between gap-3">
          <span className="flex min-w-0 items-center gap-2">
            <span
              className="h-3 w-3 shrink-0 rounded-sm"
              style={{ backgroundColor: s.color }}
            />
            <span className="truncate text-cream/75">{s.label}</span>
          </span>
          <span className="shrink-0 font-medium text-cream">
            {format ? format(s.value) : s.value}
          </span>
        </li>
      ))}
    </ul>
  );
}
