// Money is integer VND. Always format with thousands separators, no decimals.

export function formatVnd(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return "0 ₫";
  const n = Math.round(Number(value));
  return new Intl.NumberFormat("vi-VN").format(n) + " ₫";
}

export function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return "0";
  return new Intl.NumberFormat("vi-VN").format(Math.round(Number(value)));
}

export function formatPct(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return "0%";
  // Backend always returns ownership_pct as an already-computed percent number
  // (e.g. 1.0 = 1%, 49.0 = 49%). Do NOT rescale — a 1% holder must read "1%", not "100%".
  const pct = Number(value);
  return `${pct.toLocaleString("vi-VN", { maximumFractionDigits: 4 })}%`;
}

// Định dạng chỉ NGÀY (không giờ) từ "YYYY-MM-DD" -> "DD/MM/YYYY". Dùng cho lịch rút tiền.
export function formatDay(value: string | null | undefined): string {
  if (!value) return "-";
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (m) return `${m[3]}/${m[2]}/${m[1]}`;
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleDateString("vi-VN");
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return "-";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString("vi-VN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}
