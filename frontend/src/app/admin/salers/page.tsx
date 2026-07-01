"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { formatVnd } from "@/lib/format";
import { Badge, Button, Card, Spinner, Stat } from "@/components/ui";

// Duyệt yêu cầu trở thành Cộng tác viên bán hàng (khách hàng → saler).
function AffiliateRequests() {
  const qc = useQueryClient();
  const { data: reqs, isLoading } = useQuery({ queryKey: ["admin-affiliate-requests"], queryFn: adminApi.affiliateRequests });
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["admin-affiliate-requests"] });
    qc.invalidateQueries({ queryKey: ["admin-saler-stats"] });
  };
  const approve = useMutation({ mutationFn: (id: string) => adminApi.approveAffiliate(id), onSuccess: refresh });
  const reject = useMutation({ mutationFn: (id: string) => adminApi.rejectAffiliate(id), onSuccess: refresh });

  if (isLoading) return <Spinner />;
  if (!reqs || reqs.length === 0) return null;

  return (
    <Card className="space-y-3">
      <div className="flex items-center gap-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">Yêu cầu làm Cộng tác viên</h2>
        <Badge tone="yellow">{reqs.length} chờ duyệt</Badge>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="text-left">Người yêu cầu</th>
              <th className="text-left">SĐT</th>
              <th className="text-right">Thao tác</th>
            </tr>
          </thead>
          <tbody>
            {reqs.map((r) => (
              <tr key={r.user_id}>
                <td>
                  <p className="font-medium text-cream">{r.full_name}</p>
                  <p className="text-xs text-cream/45">{r.email}</p>
                </td>
                <td className="text-cream/70">{r.phone || "—"}</td>
                <td className="text-right whitespace-nowrap">
                  <Button onClick={() => approve.mutate(r.user_id)} disabled={approve.isPending}>Duyệt</Button>
                  <button onClick={() => window.confirm(`Từ chối yêu cầu của ${r.full_name}?`) && reject.mutate(r.user_id)} className="ml-3 text-red-300 hover:underline">Từ chối</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

function SalersInner() {
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({ queryKey: ["admin-saler-stats"], queryFn: adminApi.salerStats });
  const delSaler = useMutation({
    mutationFn: (id: string) => adminApi.deleteUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-saler-stats"] }),
  });

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
      <AffiliateRequests />
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
                <th className="text-right">Thao tác</th>
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
                  <td className="text-right">
                    <button
                      onClick={() => window.confirm(`Xoá tài khoản CTV "${s.full_name}"? Không thể hoàn tác.`) && delSaler.mutate(s.seller_id)}
                      className="text-red-300 hover:underline"
                    >
                      Xoá
                    </button>
                  </td>
                </tr>
              ))}
              {data?.length === 0 && (
                <tr><td colSpan={6} className="py-6 text-center text-cream/45">Chưa có nhân viên bán hàng nào.</td></tr>
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
