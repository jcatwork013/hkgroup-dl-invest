"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { useAuth } from "@/components/AuthContext";
import { investorApi } from "@/lib/endpoints";
import { formatDate, formatVnd } from "@/lib/format";
import { Badge, Button, Card, Spinner, statusTone, statusLabel } from "@/components/ui";
import WalletPanel from "@/components/WalletPanel";

function ReferralInner() {
  const { user } = useAuth();
  const { data, isLoading } = useQuery({
    queryKey: ["my-referrals"],
    queryFn: investorApi.referrals,
  });

  const [origin, setOrigin] = useState("");
  const [copied, setCopied] = useState(false);

  // Tổng doanh số giới thiệu = tổng số tiền đầu tư của tuyến dưới (F1+F2+F3) đã sinh hoa hồng cho mình.
  const referralSales = (data?.commissions ?? []).reduce((sum, c) => sum + c.base_amount, 0);

  useEffect(() => {
    if (typeof window !== "undefined") setOrigin(window.location.origin);
  }, []);

  const link =
    user?.referral_code && origin
      ? `${origin}/register?ref=${user.referral_code}`
      : "";

  async function copy() {
    if (!link) return;
    try {
      await navigator.clipboard.writeText(link);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* ignore */
    }
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-xl font-bold text-cream">
          Chương trình giới thiệu
        </h1>
        <p className="mt-1 text-sm text-cream/55">
          Chia sẻ liên kết của bạn. Bạn nhận hoa hồng <strong>3 cấp</strong>: F1
          (người bạn trực tiếp giới thiệu), F2 và F3 (tuyến dưới của họ).
        </p>
      </div>

      <Card className="space-y-3">
        <p className="text-sm font-semibold text-cream/85">
          Liên kết giới thiệu của bạn
        </p>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <input
            readOnly
            value={link}
            className="w-full min-w-0 flex-1 truncate rounded-md border border-white/15 bg-white/[0.04] px-3 py-2 font-mono text-sm text-cream/85"
          />
          <Button
            onClick={copy}
            disabled={!link}
            className="w-full shrink-0 whitespace-nowrap sm:w-auto sm:min-w-[8rem]"
          >
            {copied ? "✓ Đã sao chép" : "Sao chép"}
          </Button>
        </div>
        {user?.referral_code && (
          <p className="text-xs text-cream/55">
            Mã giới thiệu: <span className="font-mono">{user.referral_code}</span>
          </p>
        )}
      </Card>

      {/* Ví hoa hồng + lập lệnh rút (gồm cả hoa hồng chưa duyệt + nhắc admin duyệt). */}
      <WalletPanel />

      {/* Tổng doanh số giới thiệu — tổng tiền đầu tư tuyến dưới đã sinh hoa hồng */}
      <Card className="flex flex-col gap-1 border-gold-500/25 bg-gold-500/[0.06]">
        <p className="text-xs uppercase tracking-wide text-cream/45">Tổng doanh số giới thiệu</p>
        <p className="font-serif text-2xl text-gold-300">{formatVnd(referralSales)}</p>
        <p className="text-xs text-cream/55">
          Tổng số tiền đầu tư của tuyến dưới (F1 + F2 + F3) đã phát sinh hoa hồng cho bạn.
        </p>
      </Card>

      {isLoading ? (
        <Spinner />
      ) : (
        <>
          <section>
            <h2 className="mb-3 text-lg font-semibold text-cream">
              Người được giới thiệu trực tiếp
            </h2>
            <Card className="overflow-x-auto p-0">
              <table>
                <thead>
                  <tr>
                    <th>ID người được giới thiệu</th>
                    <th>Loại</th>
                    <th>Ngày</th>
                  </tr>
                </thead>
                <tbody>
                  {data?.referrals.map((r) => (
                    <tr key={r.referee_id}>
                      <td className="font-mono text-xs">{r.referee_id}</td>
                      <td>
                        <Badge tone="blue">{r.referral_type === "investor" ? "Nhà đầu tư" : "Khách hàng"}</Badge>
                      </td>
                      <td>{formatDate(r.created_at)}</td>
                    </tr>
                  ))}
                  {data?.referrals.length === 0 && (
                    <tr>
                      <td colSpan={3} className="text-center text-cream/45">
                        Chưa có người được giới thiệu.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </Card>
          </section>

          <section>
            <h2 className="mb-3 text-lg font-semibold text-cream">
              Hoa hồng của tôi
            </h2>
            <Card className="overflow-x-auto p-0">
              <table>
                <thead>
                  <tr>
                    <th>Cấp</th>
                    <th>Cơ sở tính</th>
                    <th>Tỷ lệ</th>
                    <th>Hoa hồng</th>
                    <th>Thuế TNCN</th>
                    <th>Thực nhận</th>
                    <th>Trạng thái</th>
                  </tr>
                </thead>
                <tbody>
                  {data?.commissions.map((c) => (
                    <tr key={c.id}>
                      <td>
                        <Badge tone="blue">F{c.level}</Badge>
                      </td>
                      <td>{formatVnd(c.base_amount)}</td>
                      <td>{(c.rate * 100).toFixed(2)}%</td>
                      <td>{formatVnd(c.amount)}</td>
                      <td>{formatVnd(c.tax_pit)}</td>
                      <td className="font-medium text-cream">
                        {formatVnd(c.net_amount)}
                      </td>
                      <td>
                        <Badge tone={statusTone(c.status)}>{statusLabel(c.status)}</Badge>
                      </td>
                    </tr>
                  ))}
                  {data?.commissions.length === 0 && (
                    <tr>
                      <td colSpan={7} className="text-center text-cream/45">
                        Chưa có hoa hồng.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </Card>
          </section>
        </>
      )}
    </div>
  );
}

export default function ReferralPage() {
  return (
    <Guard requireRole="investor">
      <ReferralInner />
    </Guard>
  );
}
