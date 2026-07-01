"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { investorApi } from "@/lib/endpoints";
import { formatDate, formatNumber, formatPct, formatVnd } from "@/lib/format";
import {
  Badge,
  Button,
  Card,
  Spinner,
  Stat,
  statusTone,
  statusLabel,
} from "@/components/ui";
import RiskDisclaimer from "@/components/RiskDisclaimer";
import WalletPanel from "@/components/WalletPanel";
import { Donut, DonutLegend } from "@/components/Donut";

function DashboardInner() {
  const dash = useQuery({
    queryKey: ["dashboard"],
    queryFn: investorApi.dashboard,
  });
  const investments = useQuery({
    queryKey: ["my-investments"],
    queryFn: investorApi.investments,
  });
  const dividends = useQuery({
    queryKey: ["my-dividends"],
    queryFn: investorApi.dividends,
  });

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-cream">Bảng điều khiển</h1>
        <Link href="/invest">
          <Button>Đầu tư thêm</Button>
        </Link>
      </div>

      {dash.isLoading ? (
        <Spinner />
      ) : dash.isError ? (
        <Card>
          <p className="text-sm text-red-600">
            Không tải được dữ liệu: {(dash.error as Error)?.message}
          </p>
        </Card>
      ) : (
        dash.data && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Stat
              label="Vốn đã góp"
              value={formatVnd(dash.data.capital_contributed_vnd)}
            />
            <Stat label="Số cổ phần" value={formatNumber(dash.data.shares)} />
            <Stat
              label="Tỷ lệ sở hữu"
              value={formatPct(dash.data.ownership_pct)}
            />
            <Stat
              label="Cổ tức đã nhận"
              value={formatVnd(dash.data.dividend_received_vnd)}
              hint="Cổ tức thực tế đã chi trả"
            />
          </div>
        )
      )}

      {dash.data && dash.data.ownership_pct > 0 && (
        <Card className="flex flex-col items-center gap-6 sm:flex-row sm:items-center sm:gap-10">
          <Donut
            segments={[
              { label: "Sở hữu của bạn", value: dash.data.ownership_pct, color: "#c9a24a" },
              {
                label: "Phần còn lại của công ty",
                value: Math.max(0, 100 - dash.data.ownership_pct),
                color: "rgba(255,255,255,0.12)",
              },
            ]}
            centerLabel={formatPct(dash.data.ownership_pct)}
            centerSub="Tỷ lệ sở hữu"
          />
          <div className="w-full max-w-sm space-y-3">
            <p className="text-sm font-semibold text-cream/85">
              Cơ cấu sở hữu của bạn
            </p>
            <DonutLegend
              segments={[
                { label: "Sở hữu của bạn", value: dash.data.ownership_pct, color: "#c9a24a" },
                {
                  label: "Phần còn lại của công ty",
                  value: Math.max(0, 100 - dash.data.ownership_pct),
                  color: "rgba(255,255,255,0.12)",
                },
              ]}
              format={(v) => formatPct(v)}
            />
            <p className="text-xs text-cream/45">
              Tương ứng {formatNumber(dash.data.shares)} cổ phần trên tổng vốn cổ
              phần của công ty.
            </p>
          </div>
        </Card>
      )}

      {/* Ví hoa hồng & rút tiền — số dư có thể rút + lập lệnh rút (kể cả hoa hồng chưa duyệt). */}
      <WalletPanel />

      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">
          Khoản đầu tư của tôi
        </h2>
        <Card className="overflow-x-auto p-0">
          <table>
            <thead>
              <tr>
                <th>Mã</th>
                <th>Số tiền</th>
                <th>Cổ phần</th>
                <th>Trạng thái</th>
                <th>Ngày tạo</th>
                <th>Hợp đồng</th>
              </tr>
            </thead>
            <tbody>
              {investments.data?.map((inv) => (
                <tr key={inv.id}>
                  <td className="font-medium text-cream">{inv.code}</td>
                  <td>{formatVnd(inv.amount_vnd)}</td>
                  <td>{formatNumber(inv.shares)}</td>
                  <td>
                    <Badge tone={statusTone(inv.status)}>{statusLabel(inv.status)}</Badge>
                  </td>
                  <td>{formatDate(inv.created_at)}</td>
                  <td>
                    <span
                      title="Sẽ có khi được phê duyệt"
                      className="cursor-not-allowed text-xs text-cream/45 underline decoration-dotted"
                    >
                      Tải hợp đồng
                    </span>
                  </td>
                </tr>
              ))}
              {investments.data?.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center text-cream/45">
                    Chưa có khoản đầu tư nào.
                  </td>
                </tr>
              )}
              {investments.isLoading && (
                <tr>
                  <td colSpan={6} className="text-center text-cream/45">
                    Đang tải...
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </Card>
      </section>

      <section className="space-y-4">
        <h2 className="text-lg font-semibold text-cream">Cổ tức</h2>
        {/* Ví cổ tức — số dư rút riêng (giống ví hoa hồng): rút 1 phần hoặc rút hết. */}
        <WalletPanel kind="dividend" />
        <h3 className="pt-2 text-sm font-semibold uppercase tracking-wide text-cream/70">
          Lịch sử cổ tức được chia
        </h3>
        <Card className="overflow-x-auto p-0">
          <table>
            <thead>
              <tr>
                <th>Kỳ</th>
                <th>Số tiền</th>
                <th>Cổ phần</th>
                <th>Ngày chi trả</th>
                <th>Ghi chú</th>
              </tr>
            </thead>
            <tbody>
              {dividends.data?.map((d, i) => (
                <tr key={`${d.period}-${i}`}>
                  <td className="font-medium text-cream">{d.period}</td>
                  <td>{formatVnd(d.amount)}</td>
                  <td>{formatNumber(d.shares)}</td>
                  <td>{formatDate(d.paid_at)}</td>
                  <td className="whitespace-normal text-cream/55">{d.note}</td>
                </tr>
              ))}
              {dividends.data?.length === 0 && (
                <tr>
                  <td colSpan={5} className="text-center text-cream/45">
                    Chưa có cổ tức nào được chi trả.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </Card>
      </section>

      <RiskDisclaimer />
    </div>
  );
}

export default function DashboardPage() {
  return (
    <Guard requireRole="investor">
      <DashboardInner />
    </Guard>
  );
}
