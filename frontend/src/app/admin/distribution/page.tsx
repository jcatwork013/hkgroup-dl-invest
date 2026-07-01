"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi, publicApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatDate, formatNumber, formatPct, formatVnd } from "@/lib/format";
import { Button, Card, ErrorText, Input, Spinner } from "@/components/ui";
import { Donut, DonutLegend, DONUT_PALETTE } from "@/components/Donut";
import type { DonutSegment } from "@/components/Donut";
import type { TieredPlan } from "@/lib/types";

function DistInner() {
  const qc = useQueryClient();
  const [period, setPeriod] = useState("");
  const [revenue, setRevenue] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [plan, setPlan] = useState<TieredPlan | null>(null);

  const { data: pool } = useQuery({ queryKey: ["pool"], queryFn: publicApi.pool });
  const { data: settings } = useQuery({ queryKey: ["admin-settings"], queryFn: adminApi.settings });
  const { data: dists, isLoading } = useQuery({ queryKey: ["admin-distributions"], queryFn: adminApi.distributions });

  const showPool = settings?.show_pool_public === "on";

  const toggle = useMutation({
    mutationFn: (on: boolean) => adminApi.updateSettings({ show_pool_public: on ? "on" : "off" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-settings"] });
      qc.invalidateQueries({ queryKey: ["settings"] });
    },
    onError: (e) => setError((e as ApiException).message),
  });

  const distribute = useMutation({
    mutationFn: (v: { period: string; revenue: number }) => adminApi.distributeTiered(v.period, v.revenue),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-distributions"] });
      qc.invalidateQueries({ queryKey: ["pool"] });
      setPeriod(""); setRevenue("");
    },
    onError: (e) => setError((e as ApiException).message),
  });

  const preview = useMutation({
    mutationFn: (v: { period: string; revenue: number }) =>
      adminApi.previewTiered(v.period || "preview", v.revenue),
    onSuccess: (p) => setPlan(p as TieredPlan),
    onError: (e) => setError((e as ApiException).message),
  });

  // Xoá tay 1 lần phân bổ — gỡ luôn đợt cổ tức + payouts gắn kèm; trả pool về đúng.
  const delDist = useMutation({
    mutationFn: (id: string) => adminApi.deleteDistribution(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-distributions"] });
      qc.invalidateQueries({ queryKey: ["admin-dividends"] });
      qc.invalidateQueries({ queryKey: ["pool"] });
    },
    onError: (e) => setError((e as ApiException).message),
  });

  function handleDeleteDist(id: string, period: string) {
    if (!window.confirm(`Xoá lần phân bổ kỳ "${period}"?\n\nXoá cả đợt cổ tức + các khoản chia cho cổ đông của lần này. KHÔNG THỂ hoàn tác.`)) return;
    setError(null);
    delDist.mutate(id);
  }

  const revNum = parseInt(revenue.replace(/\D/g, ""), 10) || 0;
  // Toàn bộ Pool Cổ Đông (pool_rate) chia thẳng cho nhà đầu tư — không nhân 49%.
  const previewPool = pool ? Math.floor(revNum * pool.pool_rate) : 0;

  // Build the "đồng chia + bonus" pie from the tiered plan preview.
  const planSegments: DonutSegment[] = plan
    ? [
        { label: "Đồng chia (cào bằng)", value: plan.equal_pool, color: "#c9a24a" },
        ...plan.bands
          .filter((b) => b.bonus_total > 0)
          .map((b, i) => ({
            label: `Bonus ${b.label}`,
            value: b.bonus_total,
            color: DONUT_PALETTE[(i + 2) % DONUT_PALETTE.length],
          })),
        ...(plan.residual > 0
          ? [{ label: "Chưa phân bổ", value: plan.residual, color: "rgba(255,255,255,0.12)" }]
          : []),
      ]
    : [];

  function handleDistribute(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!period || !revNum) { setError("Nhập kỳ và doanh thu hợp lệ."); return; }
    distribute.mutate({ period, revenue: revNum });
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="font-serif text-2xl text-cream">Pool &amp; Phân bổ doanh thu</h1>
        <p className="mt-1 text-sm text-cream/55">
          Nhập doanh thu kỳ → hệ thống tính Pool Cổ Đông và chia cho nhà đầu tư
          theo Đồng chia (cào bằng) + Bonus theo hạng đầu tư (cổ tức thực, biến động).
        </p>
      </div>

      <ErrorText>{error}</ErrorText>

      {/* POOL CỔ PHẦN ĐƠN HÀNG */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <PoolStat label="Định giá hệ sinh thái" value={pool ? formatVnd(pool.valuation_vnd) : "—"} />
        <PoolStat label="Cổ phần đơn hàng ký gửi (49%)" value={pool ? formatVnd(pool.pool_target_vnd) : "—"} />
        <PoolStat label="Đã bán" value={pool ? formatVnd(pool.raised_vnd) : "—"} accent />
        <PoolStat label="Còn lại" value={pool ? formatVnd(pool.remaining_vnd) : "—"} />
      </div>
      {pool && (
        <Card>
          <div className="mb-1.5 flex justify-between text-xs text-cream/55">
            <span>Tiến độ cổ phần đơn hàng ký gửi</span>
            <span className="text-gold-300">{pool.progress_pct.toFixed(1)}%</span>
          </div>
          <div className="h-3 w-full overflow-hidden rounded-full bg-white/10">
            <div className="h-full rounded-full bg-gradient-to-r from-gold-500 to-gold-300"
              style={{ width: `${Math.min(100, pool.progress_pct)}%` }} />
          </div>
          <div className="mt-4 grid gap-4 text-sm sm:grid-cols-3">
            <Mini label="Tỷ lệ Pool Cổ Đông" value={formatPct(pool.pool_rate * 100)} />
            <Mini label="NĐT hưởng" value="100% Pool" />
            <Mini label="Đã phân bổ cho NĐT" value={formatVnd(pool.total_investor_pool_vnd)} />
          </div>
        </Card>
      )}

      {/* SWITCH hiển thị Pool ngoài invest */}
      <Card className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="font-semibold text-cream">Hiển thị Pool ngoài trang Đầu tư</p>
          <p className="text-sm text-cream/55">
            Bật để khách thấy tiến độ Pool trên trang invest công khai.
          </p>
        </div>
        <button
          type="button"
          onClick={() => toggle.mutate(!showPool)}
          disabled={toggle.isPending}
          className={`relative h-7 w-12 rounded-full transition ${showPool ? "bg-gold-500" : "bg-white/15"}`}
          aria-pressed={showPool}
        >
          <span className={`absolute top-1 h-5 w-5 rounded-full bg-white transition-all ${showPool ? "left-6" : "left-1"}`} />
        </button>
      </Card>

      {/* PHÂN BỔ */}
      <Card className="max-w-xl space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
          Nhập doanh thu &amp; phân bổ
        </h2>
        <form onSubmit={handleDistribute} className="space-y-4">
          <Input label="Kỳ (period)" value={period} onChange={(e) => setPeriod(e.target.value)} placeholder="VD: 2026-06" />
          <Input
            label="Tổng doanh thu bán hàng của kỳ (VNĐ)"
            inputMode="numeric"
            value={revNum > 0 ? formatNumber(revNum) : revenue}
            onChange={(e) => setRevenue(e.target.value.replace(/\D/g, ""))}
            placeholder="VD: 100.000.000"
          />
          <p className="rounded-lg bg-white/5 px-3 py-2 text-xs leading-relaxed text-cream/55">
            Đây là <strong className="text-cream/80">tổng doanh thu bán hàng của hệ sinh thái trong kỳ</strong>.
            Hệ thống trích {pool ? formatPct(pool.pool_rate * 100) : "15%"} làm{" "}
            <strong className="text-cream/80">Pool cổ đông</strong> để chia cổ tức cho nhà đầu tư
            (đồng chia + bonus theo hạng). <strong>Không phải</strong> hoa hồng % bán hàng theo đơn — đó là khoản riêng.
          </p>
          {revNum > 0 && pool && (
            <p className="text-sm text-cream/70">
              → Pool Cổ Đông {formatPct(pool.pool_rate * 100)} ={" "}
              <strong className="text-gold-300">{formatVnd(previewPool)}</strong> chia theo Đồng chia + Bonus theo hạng.
              Bấm “Xem cơ cấu chia” để xem chi tiết trước khi chốt.
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            <Button type="submit" disabled={distribute.isPending}>
              {distribute.isPending ? "Đang phân bổ..." : "Phân bổ cho nhà đầu tư"}
            </Button>
            <Button
              type="button"
              variant="secondary"
              disabled={!revNum || preview.isPending}
              onClick={() => {
                setError(null);
                preview.mutate({ period, revenue: revNum });
              }}
            >
              {preview.isPending ? "Đang tính..." : "Xem cơ cấu chia (đồng chia + bonus)"}
            </Button>
          </div>
        </form>
      </Card>

      {plan && planSegments.length > 0 && (
        <Card className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
              Cơ cấu chia Pool · đồng chia {formatPct(plan.equal_rate * 100)} + bonus{" "}
              {formatPct(plan.bonus_rate * 100)}
            </h2>
            <span className="text-xs text-cream/45">{plan.accounts} tài khoản</span>
          </div>
          <div className="flex flex-col items-center gap-8 lg:flex-row lg:items-center lg:gap-12">
            <Donut
              segments={planSegments}
              size={200}
              thickness={26}
              centerLabel={formatVnd(plan.distributed)}
              centerSub="Tổng chia"
            />
            <div className="w-full lg:max-w-md">
              <DonutLegend segments={planSegments} format={(v) => formatVnd(v)} />
              {plan.scaled && (
                <p className="mt-3 text-xs text-amber-300/80">
                  ⚠ Tổng bonus vượt pool — đã co tỉ lệ cho khít.
                </p>
              )}
            </div>
          </div>
        </Card>
      )}

      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">Lịch sử phân bổ</h2>
        {isLoading ? <Spinner /> : (
          <Card className="overflow-x-auto p-0">
            <table>
              <thead>
                <tr><th>Kỳ</th><th>Doanh thu bán hàng</th><th>Pool NĐT (cổ tức)</th><th>Ngày</th><th>Hành động</th></tr>
              </thead>
              <tbody>
                {dists?.map((d) => (
                  <tr key={d.id}>
                    <td className="font-medium text-cream">{d.period}</td>
                    <td>{formatVnd(d.total_revenue)}</td>
                    <td className="text-gold-300">{formatVnd(d.investor_pool)}</td>
                    <td>{formatDate(d.created_at)}</td>
                    <td>
                      <button
                        onClick={() => handleDeleteDist(d.id, d.period)}
                        disabled={delDist.isPending}
                        className="text-xs font-medium text-red-400/80 transition hover:text-red-300 disabled:opacity-40"
                      >
                        Xoá
                      </button>
                    </td>
                  </tr>
                ))}
                {dists?.length === 0 && (
                  <tr><td colSpan={5} className="text-center text-cream/45">Chưa có phân bổ nào.</td></tr>
                )}
              </tbody>
            </table>
          </Card>
        )}
      </section>
    </div>
  );
}

function PoolStat({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <Card>
      <p className="text-xs uppercase tracking-wide text-cream/45">{label}</p>
      <p className={`mt-2 font-serif text-xl font-semibold ${accent ? "text-gold-400" : "text-cream"}`}>{value}</p>
    </Card>
  );
}
function Mini({ label, value }: { label: string; value: string }) {
  return <div><p className="text-cream/45">{label}</p><p className="mt-0.5 font-medium text-cream">{value}</p></div>;
}

export default function AdminDistributionPage() {
  return <Guard requireRole="admin"><DistInner /></Guard>;
}
