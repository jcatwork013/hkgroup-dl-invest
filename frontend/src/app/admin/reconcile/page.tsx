"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatDate, formatNumber, formatVnd } from "@/lib/format";
import {
  Badge,
  Button,
  Card,
  ErrorText,
  Spinner,
  statusTone,
  statusLabel,
} from "@/components/ui";

const FILTERS = ["pending", "reconciled", "approved"] as const;
type Filter = (typeof FILTERS)[number];

function ReconcileInner() {
  const qc = useQueryClient();
  const [filter, setFilter] = useState<Filter>("pending");
  const [error, setError] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-investments", filter],
    queryFn: () => adminApi.investments(filter),
  });

  function invalidate() {
    qc.invalidateQueries({ queryKey: ["admin-investments"] });
    qc.invalidateQueries({ queryKey: ["admin-dashboard"] });
    qc.invalidateQueries({ queryKey: ["cap-table"] });
  }

  const reconcile = useMutation({
    mutationFn: (id: string) => adminApi.reconcile(id),
    onSuccess: invalidate,
    onError: (e) => setError((e as ApiException).message),
  });
  const approve = useMutation({
    mutationFn: (id: string) => adminApi.approveInvestment(id),
    onSuccess: invalidate,
    onError: (e) => setError((e as ApiException).message),
  });
  const reject = useMutation({
    mutationFn: (v: { id: string; reason: string }) =>
      adminApi.rejectInvestment(v.id, v.reason),
    onSuccess: invalidate,
    onError: (e) => setError((e as ApiException).message),
  });

  function handleReject(id: string) {
    const reason = window.prompt("Lý do từ chối?") ?? "";
    if (!reason) return;
    setError(null);
    reject.mutate({ id, reason });
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-cream">
          Đối soát khoản đầu tư
        </h1>
        <p className="mt-1 text-sm text-cream/55">
          Quy trình 3 lớp: <strong>Đối soát chuyển khoản</strong> → kiểm tra →{" "}
          <strong>Phê duyệt</strong> (ghi nhận cổ phần). Có thể từ chối kèm lý
          do.
        </p>
      </div>

      <div className="flex gap-2">
        {FILTERS.map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`rounded-md px-3 py-1.5 text-sm font-medium ${
              filter === f
                ? "bg-gold-500 text-forest-950"
                : "bg-white/[0.04] text-cream/70 border border-white/10 hover:bg-white/5"
            }`}
          >
            {statusLabel(f)}
          </button>
        ))}
      </div>

      <ErrorText>{error}</ErrorText>

      {isLoading ? (
        <Spinner />
      ) : (
        <Card className="overflow-x-auto p-0">
          <table>
            <thead>
              <tr>
                <th>Mã</th>
                <th>Số tiền</th>
                <th>Cổ phần</th>
                <th>Trạng thái</th>
                <th>Ngày tạo</th>
                <th>Hành động</th>
              </tr>
            </thead>
            <tbody>
              {data?.map((inv) => {
                const busy =
                  reconcile.isPending || approve.isPending || reject.isPending;
                return (
                  <tr key={inv.id}>
                    <td className="font-medium text-cream">{inv.code}</td>
                    <td>{formatVnd(inv.amount_vnd)}</td>
                    <td>{formatNumber(inv.shares)}</td>
                    <td>
                      <Badge tone={statusTone(inv.status)}>{statusLabel(inv.status)}</Badge>
                    </td>
                    <td>{formatDate(inv.created_at)}</td>
                    <td>
                      <div className="flex flex-wrap gap-2">
                        {inv.status === "pending" && (
                          <Button
                            variant="secondary"
                            onClick={() => {
                              setError(null);
                              reconcile.mutate(inv.id);
                            }}
                            disabled={busy}
                          >
                            Đối soát
                          </Button>
                        )}
                        {inv.status === "reconciled" && (
                          <Button
                            onClick={() => {
                              setError(null);
                              approve.mutate(inv.id);
                            }}
                            disabled={busy}
                          >
                            Phê duyệt
                          </Button>
                        )}
                        {(inv.status === "pending" ||
                          inv.status === "reconciled") && (
                          <Button
                            variant="danger"
                            onClick={() => handleReject(inv.id)}
                            disabled={busy}
                          >
                            Từ chối
                          </Button>
                        )}
                        {inv.status === "approved" && (
                          <span className="text-xs text-cream/45">
                            Đã phê duyệt
                          </span>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
              {data?.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center text-cream/45">
                    Không có khoản nào ở trạng thái này.
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

export default function ReconcilePage() {
  return (
    <Guard requireRole="admin">
      <ReconcileInner />
    </Guard>
  );
}
