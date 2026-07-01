"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { Modal } from "@/components/Modal";
import { adminApi, publicApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatDate, formatPct, formatVnd } from "@/lib/format";
import {
  Badge,
  Button,
  Card,
  ErrorText,
  Input,
  Select,
  Spinner,
  statusTone,
  statusLabel,
} from "@/components/ui";

// Danh sách kỳ để CHỌN (thay vì gõ tay): 8 quý gần nhất, mới nhất trước.
function quarterOptions(): string[] {
  const now = new Date();
  let y = now.getFullYear();
  let q = Math.floor(now.getMonth() / 3) + 1;
  const out: string[] = [];
  for (let i = 0; i < 8; i++) {
    out.push(`${y}-Q${q}`);
    q -= 1;
    if (q < 1) { q = 4; y -= 1; }
  }
  return out;
}

function DividendsInner() {
  const qc = useQueryClient();
  const [period, setPeriod] = useState("");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [openId, setOpenId] = useState<string | null>(null);

  const { data: dividends, isLoading } = useQuery({
    queryKey: ["admin-dividends"],
    queryFn: () => adminApi.dividends(),
  });
  const { data: pool } = useQuery({ queryKey: ["pool"], queryFn: publicApi.pool });

  // Hệ thống tự tính số cổ tức GỢI Ý = pool nhà đầu tư luỹ kế − đã công bố (tránh nhập tay sai).
  const declaredSum = (dividends ?? []).reduce((a, d) => a + (d.total_amount || 0), 0);
  const suggested = Math.max(0, (pool?.total_investor_pool_vnd ?? 0) - declaredSum);

  // Payouts for the expanded dividend.
  const { data: payouts, isLoading: payoutsLoading } = useQuery({
    queryKey: ["admin-dividend-payouts", openId],
    queryFn: () => adminApi.dividendPayouts(openId as string),
    enabled: !!openId,
  });

  const declare = useMutation({
    mutationFn: (v: { period: string; total_amount: number; note: string }) =>
      adminApi.declareDividend(v),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-dividends"] });
      setPeriod("");
      setAmount("");
      setNote("");
    },
    onError: (e) => setError((e as ApiException).message),
  });

  const pay = useMutation({
    mutationFn: (payoutId: string) => adminApi.payPayout(payoutId),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ["admin-dividend-payouts", openId] }),
    onError: (e) => setError((e as ApiException).message),
  });

  // Duyệt 1 lần → chi trả tất cả cổ đông của đợt (hệ thống tính sẵn theo cổ phần).
  const payAll = useMutation({
    mutationFn: (id: string) => adminApi.payAllDividend(id),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["admin-dividend-payouts", openId] });
      window.alert(`Đã chi trả ${res.paid} cổ đông. Tiền đã vào ví cổ tức của từng người.`);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  // Xoá tay 1 đợt cổ tức — gỡ luôn các khoản chi trả (payouts) và bản ghi phân bổ liên quan.
  const del = useMutation({
    mutationFn: (id: string) => adminApi.deleteDividend(id),
    onSuccess: (_d, id) => {
      qc.invalidateQueries({ queryKey: ["admin-dividends"] });
      qc.invalidateQueries({ queryKey: ["admin-distributions"] });
      qc.invalidateQueries({ queryKey: ["pool"] });
      if (openId === id) setOpenId(null);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  function handleDelete(id: string, period: string) {
    if (!window.confirm(`Xoá đợt cổ tức kỳ "${period}"?\n\nXoá cả bảng chi trả cho cổ đông của đợt này. KHÔNG THỂ hoàn tác.`)) return;
    setError(null);
    del.mutate(id);
  }

  function handleDeclare(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const total = parseInt(amount.replace(/\D/g, ""), 10);
    if (!period || !total || Number.isNaN(total)) {
      setError("Vui lòng nhập kỳ và tổng số tiền hợp lệ (VND nguyên).");
      return;
    }
    declare.mutate({ period, total_amount: total, note });
  }

  return (
    <div className="space-y-8">
      <h1 className="text-xl font-bold text-cream">Quản lý cổ tức</h1>
      <p className="text-sm text-cream/55">
        Cổ tức chỉ tồn tại khi admin công bố — không có tiến trình tự động nào
        sinh cổ tức. Số tiền chia là cổ tức <strong>thực chi</strong>.
      </p>

      <ErrorText>{error}</ErrorText>

      <Card className="max-w-lg">
        <h2 className="mb-4 text-sm font-semibold text-cream/85">
          Công bố cổ tức mới
        </h2>
        <form onSubmit={handleDeclare} className="space-y-4">
          <Select
            label="Kỳ chia cổ tức (chọn quý)"
            value={period}
            onChange={(e) => {
              setPeriod(e.target.value);
              if (e.target.value && !amount && suggested > 0) setAmount(String(suggested));
            }}
          >
            <option value="">— Chọn quý —</option>
            {quarterOptions().map((p) => (
              <option key={p} value={p}>{p.replace("-Q", " · Quý ")}</option>
            ))}
          </Select>
          <Input
            label="Tổng số tiền cổ tức (VND nguyên)"
            value={amount}
            inputMode="numeric"
            onChange={(e) => setAmount(e.target.value)}
            placeholder="VD: 100000000"
          />
          {suggested > 0 && (
            <p className="text-xs text-cream/55">
              Hệ thống gợi ý theo pool khả dụng:{" "}
              <button type="button" onClick={() => setAmount(String(suggested))} className="font-mono font-semibold text-gold-300 hover:underline">
                {formatVnd(suggested)}
              </button>{" "}
              (pool luỹ kế − đã công bố). Bấm để điền.
            </p>
          )}
          <Input
            label="Ghi chú"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          <Button type="submit" disabled={declare.isPending}>
            {declare.isPending ? "Đang công bố..." : "Công bố cổ tức"}
          </Button>
        </form>
      </Card>

      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">
          Danh sách đợt cổ tức
        </h2>
        {isLoading ? (
          <Spinner />
        ) : (
          <Card className="overflow-x-auto p-0">
            <table>
              <thead>
                <tr>
                  <th>Kỳ</th>
                  <th>Tổng chia</th>
                  <th>Ngày công bố</th>
                  <th>Ghi chú</th>
                  <th>Chi tiết</th>
                </tr>
              </thead>
              <tbody>
                {dividends?.map((d) => (
                  <tr key={d.id}>
                    <td className="font-medium text-cream">{d.period}</td>
                    <td>{formatVnd(d.total_amount)}</td>
                    <td>{formatDate(d.declared_at)}</td>
                    <td className="whitespace-normal text-cream/55">
                      {d.note}
                    </td>
                    <td>
                      <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                        <button
                          onClick={() => setOpenId(d.id)}
                          className="text-xs font-medium text-gold-300 transition hover:text-gold-200"
                        >
                          Xem chi trả
                        </button>
                        <button
                          onClick={() => handleDelete(d.id, d.period)}
                          disabled={del.isPending}
                          className="text-xs font-medium text-red-400/80 transition hover:text-red-300 disabled:opacity-40"
                        >
                          Xoá
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {dividends?.length === 0 && (
                  <tr>
                    <td colSpan={5} className="text-center text-cream/45">
                      Chưa có cổ tức nào.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </Card>
        )}
      </section>

      <Modal open={!!openId} onClose={() => setOpenId(null)} title="Bảng chi trả cổ tức" size="lg">
        <div>
          <p className="mb-3 max-w-3xl text-xs text-cream/55">
            Chi tiết cấu thành cổ tức theo <strong>phần ĐẦU TƯ</strong>: mỗi người ={" "}
            <strong className="text-cream/80">Đồng chia</strong> (9% doanh thu, cào bằng cho mọi cổ đông) +{" "}
            <strong className="text-cream/80">Bonus</strong> (6% doanh thu, chia theo hạng vốn đầu tư).
            Đây <strong>KHÔNG</strong> bao gồm hoa hồng % bán hàng theo đơn — đó là khoản riêng.
          </p>
          {payoutsLoading ? (
            <Spinner />
          ) : (
            <>
              {(payouts?.some((p) => !p.paid_at) ?? false) && (
                <div className="mb-3 flex flex-wrap items-center gap-3 rounded-lg border border-gold-500/30 bg-gold-500/10 p-3">
                  <span className="text-sm text-cream/80">
                    Hệ thống đã tính sẵn theo cổ phần. Duyệt 1 lần để chi trả toàn bộ:
                  </span>
                  <Button
                    onClick={() => window.confirm("Chi trả TẤT CẢ cổ đông của đợt này? Tiền sẽ vào ví cổ tức của từng người.") && payAll.mutate(openId as string)}
                    disabled={payAll.isPending}
                  >
                    {payAll.isPending ? "Đang chi trả..." : `Duyệt & chi trả tất cả (${payouts?.filter((p) => !p.paid_at).length})`}
                  </Button>
                </div>
              )}
            <Card className="overflow-x-auto p-0">
              <table>
                <thead>
                  <tr>
                    <th>Cổ đông</th>
                    <th>Vốn đầu tư</th>
                    <th>Hạng</th>
                    <th>Đồng chia (9%)</th>
                    <th>Bonus (hạng)</th>
                    <th>Tổng nhận</th>
                    <th>Trạng thái</th>
                    <th>Hành động</th>
                  </tr>
                </thead>
                <tbody>
                  {payouts?.map((p) => {
                    const status = p.paid_at ? "paid" : "pending";
                    const tiered = !!p.band; // cổ tức tiered có hạng; pro-rata thì rỗng
                    return (
                      <tr key={p.id}>
                        <td className="font-medium text-cream">
                          {p.full_name}
                          <span className="block text-xs text-cream/45">
                            {p.email}
                          </span>
                        </td>
                        <td>{p.invested_vnd > 0 ? formatVnd(p.invested_vnd) : "—"}</td>
                        <td>
                          {tiered ? (
                            <span className="whitespace-nowrap">
                              {p.band}
                              <span className="ml-1 text-xs text-gold-300">
                                {formatPct(p.band_rate * 100)}
                              </span>
                            </span>
                          ) : (
                            "—"
                          )}
                        </td>
                        <td>{tiered ? formatVnd(p.equal_share) : "—"}</td>
                        <td className="text-gold-300">
                          {tiered ? formatVnd(p.bonus) : "—"}
                        </td>
                        <td className="font-semibold text-cream">{formatVnd(p.amount)}</td>
                        <td>
                          <Badge tone={statusTone(status)}>{statusLabel(status)}</Badge>
                        </td>
                        <td>
                          {!p.paid_at ? (
                            <Button
                              variant="secondary"
                              onClick={() => {
                                setError(null);
                                pay.mutate(p.id);
                              }}
                              disabled={pay.isPending}
                            >
                              Đánh dấu đã chi
                            </Button>
                          ) : (
                            <span className="text-xs text-cream/45">
                              {formatDate(p.paid_at)}
                            </span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                  {payouts?.length === 0 && (
                    <tr>
                      <td colSpan={8} className="text-center text-cream/45">
                        Chưa có bản ghi chi trả.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </Card>
            </>
          )}
        </div>
      </Modal>
    </div>
  );
}

export default function AdminDividendsPage() {
  return (
    <Guard requireRole="admin">
      <DividendsInner />
    </Guard>
  );
}
