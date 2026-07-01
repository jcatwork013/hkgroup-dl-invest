"use client";

import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { salesApi } from "@/lib/endpoints";
import { formatVnd } from "@/lib/format";
import { Badge, Card, Spinner, statusLabel, statusTone } from "@/components/ui";
import WalletPanel from "@/components/WalletPanel";

function CommissionsInner() {
  const { data: commissions, isLoading } = useQuery({ queryKey: ["my-sales-commissions"], queryFn: salesApi.myCommissions });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-serif text-2xl text-cream">Hoa hồng & Ví</h1>
        <p className="mt-1 text-sm text-cream/55">Hoa hồng bán hàng (đã trừ 10% TNCN) cộng vào ví chung, rút vào ngày rút khi admin duyệt.</p>
      </div>

      {/* Ví chung + lập lệnh rút (gồm cả hoa hồng chưa duyệt + nhắc admin duyệt). */}
      <WalletPanel />

      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">Hoa hồng bán hàng</h2>
        {isLoading ? (
          <Spinner />
        ) : (
          <Card className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="text-left">Loại</th>
                  <th className="text-right">Cơ sở</th>
                  <th className="text-right">Tỷ lệ</th>
                  <th className="text-right">Hoa hồng</th>
                  <th className="text-right">Thuế</th>
                  <th className="text-right">Thực nhận</th>
                  <th className="text-left">Trạng thái</th>
                </tr>
              </thead>
              <tbody>
                {commissions?.map((c) => (
                  <tr key={c.id}>
                    <td><Badge tone="blue">{c.kind === "seller" ? "Người bán" : "Affiliate"}</Badge></td>
                    <td className="text-right font-mono text-cream/70">{formatVnd(c.base_amount)}</td>
                    <td className="text-right font-mono">{(c.rate * 100).toFixed(0)}%</td>
                    <td className="text-right font-mono">{formatVnd(c.amount)}</td>
                    <td className="text-right font-mono text-red-300">−{formatVnd(c.tax_pit)}</td>
                    <td className="text-right font-mono font-medium text-cream">{formatVnd(c.net_amount)}</td>
                    <td><Badge tone={statusTone(c.status)}>{statusLabel(c.status)}</Badge></td>
                  </tr>
                ))}
                {commissions?.length === 0 && (
                  <tr><td colSpan={7} className="py-6 text-center text-cream/45">Chưa có hoa hồng nào.</td></tr>
                )}
              </tbody>
            </table>
          </Card>
        )}
      </section>
    </div>
  );
}

export default function SalerCommissionsPage() {
  return (
    <Guard requireRole="saler">
      <CommissionsInner />
    </Guard>
  );
}
