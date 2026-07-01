"use client";

import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { formatNumber, formatVnd } from "@/lib/format";
import { Card, Spinner, Stat } from "@/components/ui";

function AdminInner() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-dashboard"],
    queryFn: adminApi.dashboard,
  });

  return (
    <div className="space-y-8">
      <h1 className="text-xl font-bold text-cream">Bảng điều khiển quản trị</h1>

      {isLoading ? (
        <Spinner />
      ) : isError ? (
        <Card>
          <p className="text-sm text-red-600">
            Không tải được: {(error as Error)?.message}
          </p>
        </Card>
      ) : (
        data && (
          <>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <Stat
                label="Tổng vốn đã đối soát"
                value={formatVnd(data.stats.total_capital_reconciled)}
              />
              <Stat
                label="Số cổ đông"
                value={formatNumber(data.stats.shareholder_count)}
              />
              <Stat
                label="Cổ phần đã bán"
                value={formatNumber(data.stats.shares_sold)}
              />
              <Stat
                label="Cổ phần chào bán"
                value={formatNumber(data.stats.shares_for_sale)}
              />
            </div>

            <section>
              <h2 className="mb-3 text-lg font-semibold text-cream">
                Hoa hồng theo loại
              </h2>
              <div className="grid gap-4 md:grid-cols-2">
                <Card title="Khách hàng (customer)">
                  <dl className="space-y-2 text-sm">
                    <Row label="Tổng (gross)" value={formatVnd(data.customer_commission_gross_vnd)} />
                    <Row label="Thuế TNCN" value={formatVnd(data.customer_commission_tax_vnd)} />
                    <Row label="Thực chi (net)" value={formatVnd(data.customer_commission_net_vnd)} strong />
                  </dl>
                </Card>
                <Card title="Nhà đầu tư (investor)">
                  <dl className="space-y-2 text-sm">
                    <Row
                      label="Tổng (gross)"
                      value={formatVnd(data.investor_commission_gross_vnd)}
                      strong
                    />
                  </dl>
                </Card>
              </div>
            </section>
          </>
        )
      )}
    </div>
  );
}

function Row({
  label,
  value,
  strong,
}: {
  label: string;
  value: string;
  strong?: boolean;
}) {
  return (
    <div className="flex items-center justify-between">
      <dt className="text-cream/55">{label}</dt>
      <dd className={strong ? "font-semibold text-cream" : "text-cream/85"}>
        {value}
      </dd>
    </div>
  );
}

export default function AdminDashboardPage() {
  return (
    <Guard requireRole="admin">
      <AdminInner />
    </Guard>
  );
}
