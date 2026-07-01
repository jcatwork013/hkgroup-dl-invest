"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { investorApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatDate, formatDay, formatNumber, formatVnd } from "@/lib/format";
import { Badge, Button, Card, ErrorText, Input, statusLabel, statusTone } from "@/components/ui";
import { Modal } from "@/components/Modal";
import type { WithdrawalWindow } from "@/lib/types";

// "ngày 15 và 30" / "ngày 15, 30 và 31"
function joinDays(ds: number[]): string {
  if (!ds || ds.length === 0) return "ngày rút cố định";
  if (ds.length === 1) return `ngày ${ds[0]}`;
  return `ngày ${ds.slice(0, -1).join(", ")} và ${ds[ds.length - 1]}`;
}

// Banner lịch rút tiền — báo cửa sổ rút đang mở hay kỳ kế tiếp.
function WithdrawalSchedule({ win }: { win?: WithdrawalWindow }) {
  if (!win) return null;
  const days = joinDays(win.days);
  if (win.open_today) {
    return (
      <div className="rounded-lg border border-green-500/30 bg-green-500/[0.08] px-3 py-2 text-sm text-green-200">
        ✓ Kỳ rút đang <strong>mở hôm nay</strong>. Bạn có thể gửi yêu cầu rút ngay. Lịch rút: {days} hàng tháng (mở 00h00).
      </div>
    );
  }
  return (
    <div className="rounded-lg border border-gold-500/25 bg-gold-500/[0.06] px-3 py-2 text-sm text-cream/75">
      ⏳ Chỉ nhận yêu cầu rút vào <strong>{days}</strong> hàng tháng (mở 00h00). Kỳ rút kế tiếp:{" "}
      <strong className="text-gold-300">{formatDay(win.next_date)}</strong>
      {win.days_until > 0 && <> (còn {win.days_until} ngày)</>}.
    </div>
  );
}

// digitsToInt: "1.000.000" / "1000000" -> 1000000 (chỉ giữ chữ số).
function digitsToInt(s: string): number {
  return parseInt(s.replace(/\D/g, ""), 10) || 0;
}

/**
 * WalletPanel — ví rút tiền dùng chung: số dư, lịch rút, nút "Rút tiền" mở modal
 * lập lệnh (rút 1 phần hoặc rút hết), lịch sử rút. Tự fetch dữ liệu.
 *   kind="commission" (mặc định) — ví hoa hồng (đầu tư + bán hàng).
 *   kind="dividend"               — ví cổ tức (số dư rút riêng).
 */
export default function WalletPanel({
  kind = "commission",
  title,
}: {
  kind?: "commission" | "dividend";
  title?: string;
}) {
  const qc = useQueryClient();
  const isDividend = kind === "dividend";
  const heading = title ?? (isDividend ? "Ví cổ tức & Rút tiền" : "Ví hoa hồng & Rút tiền");
  const api = isDividend
    ? {
        wallet: investorApi.dividendWallet,
        withdrawals: investorApi.dividendWithdrawals,
        request: investorApi.requestDividendWithdrawal,
      }
    : {
        wallet: investorApi.wallet,
        withdrawals: investorApi.withdrawals,
        request: investorApi.requestWithdrawal,
      };
  const walletKey = isDividend ? "my-dividend-wallet" : "my-wallet";
  const wdKey = isDividend ? "my-dividend-withdrawals" : "my-withdrawals";

  const [modalOpen, setModalOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [wErr, setWErr] = useState<string | null>(null);

  const { data: wallet } = useQuery({ queryKey: [walletKey], queryFn: api.wallet });
  const { data: withdrawals } = useQuery({ queryKey: [wdKey], queryFn: api.withdrawals });

  const win = wallet?.window;
  const open = !!win?.open_today;
  const pending = wallet?.pending_vnd ?? 0;
  const available = wallet?.available_vnd ?? 0;
  const canWithdraw = open && available > 0;

  const amt = useMemo(() => digitsToInt(amount), [amount]);
  const remaining = available - amt;
  const amountValid = amt > 0 && amt <= available;

  const withdraw = useMutation({
    mutationFn: (a: number) => api.request(a, ""),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [walletKey] });
      qc.invalidateQueries({ queryKey: [wdKey] });
      closeModal();
    },
    onError: (e) => setWErr((e as ApiException).message),
  });

  function openModal() {
    setWErr(null);
    setAmount("");
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setAmount("");
    setWErr(null);
  }

  function fillAll() {
    setWErr(null);
    setAmount(String(available));
  }

  function handleAmountChange(e: React.ChangeEvent<HTMLInputElement>) {
    setWErr(null);
    const n = digitsToInt(e.target.value);
    setAmount(n ? formatNumber(n) : "");
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setWErr(null);
    if (amt <= 0) { setWErr("Nhập số tiền hợp lệ."); return; }
    if (amt > available) { setWErr("Số tiền vượt quá số dư có thể rút."); return; }
    withdraw.mutate(amt);
  }

  return (
    <Card className="space-y-4">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">{heading}</h2>

      <div className="grid gap-4 sm:grid-cols-3">
        <div>
          <p className="text-xs uppercase tracking-wide text-cream/45">{isDividend ? "Tổng cổ tức" : "Tổng hoa hồng"}</p>
          <p className="mt-1 font-serif text-xl text-cream">{formatVnd(wallet?.earned_vnd ?? 0)}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wide text-cream/45">Đã rút / đang chờ</p>
          <p className="mt-1 font-serif text-xl text-cream">{formatVnd(wallet?.withdrawn_vnd ?? 0)}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wide text-cream/45">Số dư có thể rút</p>
          <p className="mt-1 font-serif text-xl text-gold-400">{formatVnd(available)}</p>
        </div>
      </div>

      {/* Nhắc duyệt: hoa hồng chưa duyệt vẫn nằm trong số dư nhưng cần admin duyệt. */}
      {pending > 0 && (
        <div className="rounded-lg border border-amber-400/30 bg-amber-400/[0.08] px-3 py-2 text-sm text-amber-100">
          ⚠️ Có <strong>{formatVnd(pending)}</strong> hoa hồng <strong>chưa được duyệt</strong> (đã tính trong số dư).{" "}
          <span className="text-amber-100/80">Vui lòng nhắc admin duyệt hoa hồng để hoàn tất chi trả khi rút.</span>
        </div>
      )}

      <WithdrawalSchedule win={win} />

      <div className="flex flex-wrap items-center gap-3">
        <Button type="button" onClick={openModal} disabled={!canWithdraw}>
          {!open ? "Chưa tới kỳ rút" : available <= 0 ? "Chưa có hoa hồng để rút" : "Rút tiền"}
        </Button>
        {canWithdraw && (
          <span className="text-xs text-cream/55">
            Có thể rút tới <strong className="text-gold-300">{formatVnd(available)}</strong>.
          </span>
        )}
      </div>

      {/* Khi đang trong kỳ rút mà số dư = 0 → nói rõ lý do thay vì để nút xám câm. */}
      {open && available <= 0 && (
        <p className="text-xs text-cream/55">
          Hiện chưa có hoa hồng khả dụng để rút. Nút sẽ tự mở khi tài khoản phát sinh hoa hồng.
        </p>
      )}

      {withdrawals && withdrawals.length > 0 && (
        <div className="overflow-x-auto">
          <table>
            <thead><tr><th>Số tiền</th><th>Trạng thái</th><th>Ngày YC</th></tr></thead>
            <tbody>
              {withdrawals.map((w) => (
                <tr key={w.id}>
                  <td>{formatVnd(w.amount)}</td>
                  <td><Badge tone={statusTone(w.status)}>{statusLabel(w.status)}</Badge></td>
                  <td>{formatDate(w.requested_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* ---- Modal lập lệnh rút ---- */}
      <Modal
        open={modalOpen}
        onClose={closeModal}
        title="Lập lệnh rút tiền"
        footer={
          <>
            <Button type="button" variant="ghost" onClick={closeModal} disabled={withdraw.isPending}>
              Huỷ
            </Button>
            <Button type="submit" form="withdraw-form" disabled={withdraw.isPending || !amountValid}>
              {withdraw.isPending ? "Đang gửi..." : "Xác nhận rút"}
            </Button>
          </>
        }
      >
        <form id="withdraw-form" onSubmit={handleSubmit} className="space-y-4">
          <div className="rounded-lg border border-white/10 bg-white/[0.03] px-3 py-2.5">
            <p className="text-xs uppercase tracking-wide text-cream/45">Số dư có thể rút</p>
            <p className="mt-1 font-serif text-2xl text-gold-400">{formatVnd(available)}</p>
          </div>

          <div className="flex items-end gap-2">
            <div className="flex-1">
              <Input
                label="Số tiền muốn rút (VNĐ)"
                inputMode="numeric"
                value={amount}
                onChange={handleAmountChange}
                placeholder="Nhập số tiền"
                autoFocus
              />
            </div>
            <Button type="button" variant="secondary" onClick={fillAll}>
              Rút hết
            </Button>
          </div>

          {/* Chặn realtime: không cho rút quá số dư khả dụng. */}
          {amt > available && (
            <p className="text-sm text-red-400">
              Số tiền vượt quá số dư có thể rút ({formatVnd(available)}). Tối đa bạn rút được {formatVnd(available)}.
            </p>
          )}

          <ErrorText>{wErr}</ErrorText>

          {/* Chi tiết khoản rút */}
          <div className="space-y-2 rounded-lg border border-white/10 bg-white/[0.03] px-3 py-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-cream/60">Số tiền rút</span>
              <span className="font-medium text-cream">{formatVnd(amt)}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-cream/60">Số dư còn lại sau khi rút</span>
              <span className={`font-medium ${remaining < 0 ? "text-red-400" : "text-cream"}`}>
                {formatVnd(Math.max(remaining, 0))}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-cream/60">Hình thức</span>
              <span className="text-cream/80">Chuyển khoản — admin duyệt</span>
            </div>
            {win && (
              <div className="flex items-center justify-between">
                <span className="text-cream/60">Lịch rút</span>
                <span className="text-cream/80">{joinDays(win.days)} hàng tháng</span>
              </div>
            )}
          </div>

          {pending > 0 && (
            <p className="text-xs text-amber-200/80">
              ⚠️ Trong số dư có <strong>{formatVnd(pending)}</strong> hoa hồng chưa duyệt — cần admin duyệt để hoàn tất chi trả.
            </p>
          )}
        </form>
      </Modal>
    </Card>
  );
}
