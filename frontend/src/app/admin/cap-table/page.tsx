"use client";

import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { formatNumber, formatPct } from "@/lib/format";
import { Card, Spinner } from "@/components/ui";
import { Donut, DonutLegend, DONUT_PALETTE } from "@/components/Donut";
import type { DonutSegment } from "@/components/Donut";

function CapTableInner() {
  const capTable = useQuery({
    queryKey: ["cap-table"],
    queryFn: adminApi.capTable,
  });
  const integrity = useQuery({
    queryKey: ["integrity-check"],
    queryFn: adminApi.integrityCheck,
  });

  const mismatches = integrity.data?.mismatches ?? [];
  const mismatchCount = Array.isArray(mismatches) ? mismatches.length : 0;

  // Build the ownership pie: top holders by % + a "Khác" slice + the unsold/company remainder.
  const rows = [...(capTable.data ?? [])].sort(
    (a, b) => b.ownership_pct - a.ownership_pct
  );
  const TOP = 7;
  const top = rows.slice(0, TOP);
  const rest = rows.slice(TOP);
  const restPct = rest.reduce((a, r) => a + r.ownership_pct, 0);
  const heldPct = rows.reduce((a, r) => a + r.ownership_pct, 0);
  const segments: DonutSegment[] = top.map((r, i) => ({
    label: r.full_name,
    value: r.ownership_pct,
    color: DONUT_PALETTE[i % DONUT_PALETTE.length],
  }));
  if (restPct > 0)
    segments.push({ label: `Cổ đông khác (${rest.length})`, value: restPct, color: "#5c8a8a" });
  const remaining = Math.max(0, 100 - heldPct);
  if (remaining > 0)
    segments.push({
      label: "Chưa phát hành / công ty",
      value: remaining,
      color: "rgba(255,255,255,0.12)",
    });

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-bold text-cream">Cơ cấu sở hữu (Cap Table)</h1>

      {integrity.data &&
        (integrity.data.healthy ? (
          <div className="rounded-lg border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-800">
            Kiểm tra toàn vẹn dữ liệu: <strong>khỏe mạnh</strong>. Không phát
            hiện sai lệch.
          </div>
        ) : (
          <div className="rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800">
            Kiểm tra toàn vẹn dữ liệu: <strong>phát hiện sai lệch</strong> (
            {mismatchCount}). Cần rà soát.
          </div>
        ))}

      {!capTable.isLoading && segments.length > 0 && (
        <Card className="flex flex-col items-center gap-8 lg:flex-row lg:items-center lg:gap-12">
          <Donut
            segments={segments}
            size={200}
            thickness={26}
            centerLabel={formatPct(heldPct)}
            centerSub="Đã phát hành"
          />
          <div className="w-full lg:max-w-md">
            <p className="mb-3 text-sm font-semibold text-cream/85">
              Cơ cấu cổ đông
            </p>
            <DonutLegend segments={segments} format={(v) => formatPct(v)} />
          </div>
        </Card>
      )}

      {capTable.isLoading ? (
        <Spinner />
      ) : (
        <Card className="overflow-x-auto p-0">
          <table>
            <thead>
              <tr>
                <th>Cổ đông</th>
                <th>Email</th>
                <th>Cổ phần</th>
                <th>Tỷ lệ sở hữu</th>
              </tr>
            </thead>
            <tbody>
              {capTable.data?.map((row) => (
                <tr key={row.user_id}>
                  <td className="font-medium text-cream">{row.full_name}</td>
                  <td className="text-cream/55">{row.email}</td>
                  <td>{formatNumber(row.shares)}</td>
                  <td>{formatPct(row.ownership_pct)}</td>
                </tr>
              ))}
              {capTable.data?.length === 0 && (
                <tr>
                  <td colSpan={4} className="text-center text-cream/45">
                    Chưa có cổ đông.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </Card>
      )}
    </div>
  );
}

export default function CapTablePage() {
  return (
    <Guard requireRole="admin">
      <CapTableInner />
    </Guard>
  );
}
