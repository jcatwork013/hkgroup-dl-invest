"use client";

import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { formatVnd } from "@/lib/format";
import { Card, Spinner, Stat } from "@/components/ui";

function SalersInner() {
  const { data, isLoading } = useQuery({ queryKey: ["admin-saler-stats"], queryFn: adminApi.salerStats });

  const totals = (data ?? []).reduce(
    (acc, s) => ({
      revenue: acc.revenue + s.revenue_vnd,
      commission: acc.commission + s.commission_net_vnd,
      orders: acc.orders + s.paid_orders,
    }),
    { revenue: 0, commission: 0, orders: 0 }
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-serif text-2xl text-cream">Giám sát nhân viên bán hàng</h1>
        <p className="mt-1 text-sm text-cream/55">Doanh số, số đơn và hoa hồng theo từng Saler.</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Stat label="Tổng doanh số (đã TT)" value={formatVnd(totals.revenue)} />
        <Stat label="Tổng hoa hồng (net)" value={formatVnd(totals.commission)} />
        <Stat label="Tổng đơn đã thanh toán" value={totals.orders.toLocaleString("vi-VN")} />
      </div>

      {isLoading ? (
        <Spinner />
      ) : (
        <Card className="overflow-x-auto p-0">
          <table className="w-full text-sm">
            <thead>
              <tr>
                <th className="text-left">Nhân viên</th>
                <th className="text-right">Đơn đã TT</th>
                <th className="text-right">Đơn chờ</th>
                <th className="text-right">Doanh số</th>
                <th className="text-right">Hoa hồng (net)</th>
              </tr>
            </thead>
            <tbody>
              {data?.map((s) => (
                <tr key={s.seller_id}>
                  <td>
                    <p className="font-medium text-cream">{s.full_name}</p>
                    <p className="text-xs text-cream/45">{s.email}</p>
                  </td>
                  <td className="text-right font-mono text-cream">{s.paid_orders}</td>
                  <td className="text-right font-mono text-cream/50">{s.pending_orders}</td>
                  <td className="text-right font-mono text-cream">{formatVnd(s.revenue_vnd)}</td>
                  <td className="text-right font-mono font-medium text-gold-300">{formatVnd(s.commission_net_vnd)}</td>
                </tr>
              ))}
              {data?.length === 0 && (
                <tr><td colSpan={5} className="py-6 text-center text-cream/45">Chưa có nhân viên bán hàng nào.</td></tr>
              )}
            </tbody>
          </table>
        </Card>
      )}
    </div>
  );
}

export default function SalersPage() {
  return (
    <Guard requireRole="admin">
      <SalersInner />
    </Guard>
  );
}
