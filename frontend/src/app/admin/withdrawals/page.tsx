"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatDate, formatDay, formatVnd } from "@/lib/format";
import { Badge, Button, Card, ErrorText, Input, Spinner, statusTone, statusLabel } from "@/components/ui";
import type { WithdrawalWindow } from "@/lib/types";

// "ngày 15 và 30"
function joinDays(ds: number[]): string {
  if (!ds || ds.length === 0) return "ngày rút cố định";
  if (ds.length === 1) return `ngày ${ds[0]}`;
  return `ngày ${ds.slice(0, -1).join(", ")} và ${ds[ds.length - 1]}`;
}

function ScheduleBar({ win }: { win?: WithdrawalWindow }) {
  if (!win) return null;
  return (
    <Card className="flex flex-wrap items-center justify-between gap-3 border-gold-500/25 bg-gold-500/[0.05] py-3">
      <div className="text-sm text-cream/75">
        Lịch rút: <strong className="text-cream">{joinDays(win.days)}</strong> hàng tháng (mở 00h00).{" "}
        <span className="text-cream/55">
          Nhà đầu tư chỉ gửi được yêu cầu vào các ngày này; kỳ kế tiếp{" "}
          <strong className="text-gold-300">{formatDay(win.next_date)}</strong>
          {win.days_until > 0 ? ` (còn ${win.days_until} ngày)` : ""}.
        </span>
      </div>
      <Badge tone={win.open_today ? "green" : "slate"}>{win.open_today ? "Đang mở hôm nay" : "Đang đóng"}</Badge>
    </Card>
  );
}

// DelegateWithdraw — admin "lập lệnh rút dùm": danh sách số dư từng tài khoản + ô nhập
// + nút rút (luôn HIỆN; disable khi số dư = 0 thay vì biến mất). Bỏ qua ràng buộc ngày.
function DelegateWithdraw() {
  const qc = useQueryClient();
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [err, setErr] = useState<string | null>(null);
  const [okMsg, setOkMsg] = useState<string | null>(null);
  const { data: wallets, isLoading } = useQuery({ queryKey: ["admin-wallets"], queryFn: adminApi.wallets });

  const create = useMutation({
    mutationFn: (v: { userId: string; amount: number }) => adminApi.createWithdrawalFor(v.userId, v.amount, "Admin lập lệnh rút dùm"),
    onSuccess: (_d, v) => {
      setOkMsg("Đã lập lệnh rút dùm thành công.");
      setAmounts((m) => ({ ...m, [v.userId]: "" }));
      qc.invalidateQueries({ queryKey: ["admin-wallets"] });
      qc.invalidateQueries({ queryKey: ["admin-withdrawals"] });
    },
    onError: (e) => { setOkMsg(null); setErr((e as ApiException).message); },
  });

  function submit(userId: string, available: number) {
    setErr(null); setOkMsg(null);
    const raw = amounts[userId];
    // Bỏ trống = rút toàn bộ số dư khả dụng.
    const amt = raw ? parseInt(raw.replace(/\D/g, ""), 10) || 0 : available;
    if (!amt) { setErr("Nhập số tiền hợp lệ."); return; }
    if (amt > available) { setErr("Số tiền vượt quá số dư khả dụng của tài khoản."); return; }
    create.mutate({ userId, amount: amt });
  }

  return (
    <Card className="space-y-3">
      <div>
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">Lập lệnh rút dùm</h2>
        <p className="mt-1 text-xs text-cream/55">
          Admin chủ động tạo lệnh rút thay cho tài khoản (không bị giới hạn ngày 15/30). Bỏ trống ô số tiền = rút toàn bộ số dư khả dụng. Lệnh tạo ra ở trạng thái chờ duyệt như bình thường.
        </p>
      </div>
      <ErrorText>{err}</ErrorText>
      {okMsg && <p className="text-sm text-green-300">{okMsg}</p>}
      {isLoading ? <Spinner /> : (
        <div className="overflow-x-auto">
          <table>
            <thead><tr><th>Tài khoản</th><th>Khả dụng</th><th>Chưa duyệt</th><th>Số tiền rút</th><th>Hành động</th></tr></thead>
            <tbody>
              {wallets?.map((u) => {
                const empty = u.available_vnd <= 0;
                return (
                  <tr key={u.id}>
                    <td className="font-medium text-cream">{u.full_name}<span className="block text-xs text-cream/45">{u.email}</span></td>
                    <td className="text-gold-300">{formatVnd(u.available_vnd)}</td>
                    <td className="text-amber-200/80">{u.pending_vnd > 0 ? formatVnd(u.pending_vnd) : "—"}</td>
                    <td>
                      <Input inputMode="numeric" value={amounts[u.id] ?? ""} placeholder={empty ? "—" : "Cả số dư"}
                        onChange={(e) => setAmounts((m) => ({ ...m, [u.id]: e.target.value }))} disabled={empty} />
                    </td>
                    <td>
                      <Button variant="secondary" onClick={() => submit(u.id, u.available_vnd)}
                        disabled={empty || create.isPending}>
                        {empty ? "Đã rút hết" : "Lập lệnh rút"}
                      </Button>
                    </td>
                  </tr>
                );
              })}
              {wallets?.length === 0 && <tr><td colSpan={5} className="text-center text-cream/45">Chưa có tài khoản nào phát sinh hoa hồng.</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

function WInner() {
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const { data, isLoading } = useQuery({ queryKey: ["admin-withdrawals"], queryFn: adminApi.withdrawals });

  const process = useMutation({
    mutationFn: (v: { id: string; status: string }) => adminApi.processWithdrawal(v.id, v.status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-withdrawals"] }),
    onError: (e) => setError((e as ApiException).message),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-serif text-2xl text-cream">Yêu cầu rút tiền (ví hoa hồng)</h1>
        <p className="mt-1 text-sm text-cream/55">Duyệt / chi / từ chối các yêu cầu rút từ ví hoa hồng giới thiệu.</p>
      </div>
      <ErrorText>{error}</ErrorText>
      <ScheduleBar win={data?.window} />
      <DelegateWithdraw />
      {isLoading ? <Spinner /> : (
        <Card className="overflow-x-auto p-0">
          <table>
            <thead><tr><th>Người nhận</th><th>Số tiền</th><th>Nguồn</th><th>Trạng thái</th><th>Ngày YC</th><th>Hành động</th></tr></thead>
            <tbody>
              {data?.items.map((w) => (
                <tr key={w.id}>
                  <td className="font-medium text-cream">{w.full_name}<span className="block text-xs text-cream/45">{w.email}</span></td>
                  <td className="text-gold-300">{formatVnd(w.amount)}</td>
                  <td><Badge tone={w.source === "dividend" ? "blue" : "slate"}>{w.source === "dividend" ? "Cổ tức" : "Hoa hồng"}</Badge></td>
                  <td><Badge tone={statusTone(w.status)}>{statusLabel(w.status)}</Badge></td>
                  <td>{formatDate(w.requested_at)}</td>
                  <td>
                    {w.status === "pending" && (
                      <div className="flex flex-wrap gap-2">
                        <Button variant="secondary" onClick={() => process.mutate({ id: w.id, status: "approved" })} disabled={process.isPending}>Duyệt</Button>
                        <Button variant="ghost" onClick={() => process.mutate({ id: w.id, status: "rejected" })} disabled={process.isPending}>Từ chối</Button>
                      </div>
                    )}
                    {w.status === "approved" && (
                      <Button variant="secondary" onClick={() => process.mutate({ id: w.id, status: "paid" })} disabled={process.isPending}>Đánh dấu đã chi</Button>
                    )}
                    {(w.status === "paid" || w.status === "rejected") && <span className="text-xs text-cream/40">—</span>}
                  </td>
                </tr>
              ))}
              {data?.items.length === 0 && <tr><td colSpan={6} className="text-center text-cream/45">Chưa có yêu cầu rút tiền.</td></tr>}
            </tbody>
          </table>
        </Card>
      )}
    </div>
  );
}

export default function AdminWithdrawalsPage() {
  return <Guard requireRole="admin"><WInner /></Guard>;
}
