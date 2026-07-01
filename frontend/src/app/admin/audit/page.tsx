"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { formatDate } from "@/lib/format";
import { Badge, Button, Card, Spinner } from "@/components/ui";

const LIMIT = 50;

function AuditInner() {
  const [offset, setOffset] = useState(0);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["audit-logs", offset],
    queryFn: () => adminApi.auditLogs(LIMIT, offset),
  });

  const rows = data ?? [];
  const canNext = rows.length === LIMIT;

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-bold text-cream">Nhật ký kiểm toán</h1>

      {isLoading ? (
        <Spinner />
      ) : (
        <Card className="overflow-x-auto p-0">
          <table>
            <thead>
              <tr>
                <th>Thời gian</th>
                <th>Người thực hiện</th>
                <th>Hành động</th>
                <th>Đối tượng</th>
                <th>ID đối tượng</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((log) => (
                <tr key={log.id}>
                  <td>{formatDate(log.created_at)}</td>
                  <td className="font-mono text-xs">{log.actor_id}</td>
                  <td>
                    <Badge tone="blue">{log.action}</Badge>
                  </td>
                  <td>{log.entity}</td>
                  <td className="font-mono text-xs">{log.entity_id}</td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={5} className="text-center text-cream/45">
                    Không có bản ghi.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </Card>
      )}

      <div className="flex items-center justify-between">
        <Button
          variant="secondary"
          onClick={() => setOffset((o) => Math.max(0, o - LIMIT))}
          disabled={offset === 0 || isFetching}
        >
          Trang trước
        </Button>
        <span className="text-sm text-cream/55">
          Bản ghi {offset + 1} – {offset + rows.length}
        </span>
        <Button
          variant="secondary"
          onClick={() => setOffset((o) => o + LIMIT)}
          disabled={!canNext || isFetching}
        >
          Trang sau
        </Button>
      </div>
    </div>
  );
}

export default function AuditPage() {
  return (
    <Guard requireRole="admin">
      <AuditInner />
    </Guard>
  );
}
