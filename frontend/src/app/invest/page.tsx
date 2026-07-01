"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { investorApi, publicApi } from "@/lib/endpoints";
import { ApiException, newIdempotencyKey } from "@/lib/api";
import { formatNumber, formatPct, formatVnd } from "@/lib/format";
import RiskDisclaimer from "@/components/RiskDisclaimer";
import { Donut } from "@/components/Donut";
import {
  Badge,
  Button,
  Card,
  ErrorText,
  Input,
  Spinner,
  statusLabel,
} from "@/components/ui";
import type {
  ContractResponse,
  OfferingTier,
  SignResponse,
} from "@/lib/types";

type Step = "choose" | "otp" | "pay" | "done";

function InvestInner() {
  const { data, isLoading } = useQuery({
    queryKey: ["my-offering"],
    queryFn: investorApi.myOffering,
  });
  const { data: settings } = useQuery({
    queryKey: ["settings"],
    queryFn: publicApi.settings,
    staleTime: 5 * 60 * 1000,
  });
  const showPool = settings?.show_pool_public === "on";
  const { data: pool } = useQuery({
    queryKey: ["pool"],
    queryFn: publicApi.pool,
    enabled: showPool,
  });
  const tiers = data?.tiers ?? [];
  const off = data?.offering;
  const soldOut = !!off && off.shares_sold >= off.shares_for_sale;

  const [step, setStep] = useState<Step>("choose");
  const [confirmTier, setConfirmTier] = useState<OfferingTier | null>(null);
  const [agreed, setAgreed] = useState(false);
  const [selected, setSelected] = useState<OfferingTier | null>(null);
  const [contract, setContract] = useState<ContractResponse | null>(null);
  const [otpInput, setOtpInput] = useState("");
  const [otpSending, setOtpSending] = useState(false);
  const [sign, setSign] = useState<SignResponse | null>(null);
  const [transferDeclared, setTransferDeclared] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function startContract(tier: OfferingTier) {
    setError(null);
    setBusy(true);
    setSelected(tier);
    try {
      const res = await investorApi.startContract(tier.id);
      setContract(res);
      setOtpInput("");
      setConfirmTier(null);
      setStep("otp");
      // Simulate OTP delivery: "đang gửi..." for 5s, then reveal the code.
      setOtpSending(true);
      setTimeout(() => {
        setOtpSending(false);
        setOtpInput(res.otp_code || "");
      }, 5000);
    } catch (err) {
      setError(err instanceof ApiException ? err.message : "Không tạo được hợp đồng");
      setSelected(null);
    } finally {
      setBusy(false);
    }
  }

  async function signContract() {
    if (!contract) return;
    setError(null);
    setBusy(true);
    try {
      const res = await investorApi.signContract(
        {
          contract_id: contract.contract.id,
          otp_ref: contract.otp_ref,
          otp_code: otpInput,
        },
        newIdempotencyKey()
      );
      setSign(res);
      setStep("pay");
    } catch (err) {
      setError(err instanceof ApiException ? err.message : "Ký hợp đồng thất bại");
    } finally {
      setBusy(false);
    }
  }

  async function declareTransfer() {
    if (!sign) return;
    setError(null);
    setBusy(true);
    try {
      await investorApi.declareTransfer(sign.investment.id);
      setTransferDeclared(true);
      setStep("done");
    } catch (err) {
      setError(
        err instanceof ApiException ? err.message : "Báo chuyển khoản thất bại"
      );
    } finally {
      setBusy(false);
    }
  }

  function reset() {
    setStep("choose");
    setConfirmTier(null);
    setAgreed(false);
    setSelected(null);
    setContract(null);
    setOtpInput("");
    setSign(null);
    setTransferDeclared(false);
    setError(null);
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-cream">Đầu tư góp vốn</h1>
        <p className="mt-1 text-sm text-cream/55">
          Chọn gói, ký hợp đồng bằng OTP, sau đó chuyển khoản theo hướng dẫn.
        </p>
      </div>

      <Stepper step={step} />

      {showPool && pool && step === "choose" && !confirmTier && (
        <Card className="ring-1 ring-gold-500/15">
          <div className="flex flex-col items-center gap-6 sm:flex-row sm:gap-8">
            <Donut
              size={148}
              thickness={18}
              segments={[
                { label: "Đã bán", value: pool.raised_vnd, color: "#c9a24a" },
                {
                  label: "Còn lại",
                  value: Math.max(0, pool.remaining_vnd),
                  color: "rgba(255,255,255,0.12)",
                },
              ]}
              centerLabel={`${pool.progress_pct.toFixed(1)}%`}
              centerSub="Đã bán"
            />
            <div className="w-full flex-1 space-y-3">
              <div>
                <p className="text-xs uppercase tracking-[0.2em] text-gold-400">
                  Cổ phần đơn hàng ký gửi
                </p>
                <p className="mt-1 font-serif text-xl text-cream">
                  Đã bán {formatVnd(pool.raised_vnd)} / {formatVnd(pool.pool_target_vnd)}
                </p>
              </div>
              <div className="h-2.5 w-full overflow-hidden rounded-full bg-white/10">
                <div className="h-full rounded-full bg-gradient-to-r from-gold-500 to-gold-300"
                  style={{ width: `${Math.min(100, pool.progress_pct)}%` }} />
              </div>
              <p className="text-xs text-cream/55">
                Còn lại {formatVnd(pool.remaining_vnd)} cổ phần đang mở bán.
              </p>
            </div>
          </div>
        </Card>
      )}

      <ErrorText>{error}</ErrorText>

      {step === "choose" && !confirmTier && soldOut && (
        <Card className="max-w-xl space-y-2">
          <p className="font-semibold text-gold-300">Vòng gọi vốn đã bán hết</p>
          <p className="text-sm text-cream/70">
            Toàn bộ cổ phần chào bán của vòng hiện tại đã được đăng ký. Vui lòng chờ
            quản trị viên mở vòng gọi vốn tiếp theo.
          </p>
        </Card>
      )}

      {step === "choose" && !confirmTier && !soldOut && (
        <>
          {isLoading ? (
            <Spinner />
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {tiers.map((t) => (
                <Card key={t.id} className="flex flex-col gap-3">
                  <div>
                    <p className="text-sm font-semibold text-cream">{t.name}</p>
                    <p className="mt-1 text-2xl font-bold text-gold-300">
                      {formatVnd(t.amount_vnd)}
                    </p>
                  </div>
                  <dl className="space-y-1 text-sm text-cream/70">
                    <div className="flex justify-between">
                      <dt>Số cổ phần</dt>
                      <dd>{formatNumber(t.shares)}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt>Tỷ lệ sở hữu</dt>
                      <dd>{formatPct(t.ownership_pct)}</dd>
                    </div>
                  </dl>
                  <Button
                    onClick={() => {
                      setError(null);
                      setAgreed(false);
                      setConfirmTier(t);
                    }}
                    disabled={busy}
                    className="mt-auto"
                  >
                    Chọn gói này
                  </Button>
                </Card>
              ))}
            </div>
          )}
          <RiskDisclaimer />
        </>
      )}

      {step === "choose" && confirmTier && (
        <Card className="max-w-xl space-y-4">
          <div>
            <p className="text-sm text-cream/55">Gói đã chọn</p>
            <p className="font-serif text-2xl text-cream">
              {confirmTier.name} —{" "}
              <span className="text-gold-300">
                {formatVnd(confirmTier.amount_vnd)}
              </span>
            </p>
            <p className="mt-1 text-sm text-cream/65">
              {formatNumber(confirmTier.shares)} cổ phần ·{" "}
              {formatPct(confirmTier.ownership_pct)} tỷ lệ sở hữu
            </p>
          </div>

          <div className="rounded-xl border border-gold-500/25 bg-gold-500/[0.06] p-4 text-sm leading-relaxed text-cream/80">
            <p className="font-semibold text-gold-300">
              Miễn trừ trách nhiệm &amp; xác nhận quyết định
            </p>
            <p className="mt-1">
              Việc tham gia gói này là{" "}
              <strong className="text-cream">
                quyết định đầu tư của riêng bạn
              </strong>
              . Nền tảng không cam kết lợi nhuận cố định; quyền lợi nhận lại
              (nếu có) đến từ cổ tức thực tế do công ty công bố. Vui lòng đảm
              bảo bạn đã cân nhắc kỹ và tự chịu trách nhiệm với quyết định của
              mình.
            </p>
          </div>

          <label className="flex cursor-pointer items-start gap-3 text-sm text-cream/85">
            <input
              type="checkbox"
              checked={agreed}
              onChange={(e) => setAgreed(e.target.checked)}
              className="mt-0.5 h-5 w-5 shrink-0 cursor-pointer accent-gold-500"
            />
            <span>
              Tôi xác nhận đã đọc, hiểu rõ và <strong className="text-cream">tự
              chịu trách nhiệm</strong> với quyết định đầu tư của mình.
            </span>
          </label>

          <div className="flex gap-2 pt-1">
            <Button
              onClick={() => startContract(confirmTier)}
              disabled={busy || !agreed}
            >
              {busy ? "Đang xử lý..." : "Xác nhận & tiếp tục"}
            </Button>
            <Button
              variant="ghost"
              onClick={() => setConfirmTier(null)}
              disabled={busy}
            >
              Quay lại
            </Button>
          </div>
        </Card>
      )}

      {step === "otp" && contract && selected && (
        <Card className="max-w-md space-y-4">
          <p className="text-sm text-cream/70">
            Gói đã chọn: <strong>{selected.name}</strong> —{" "}
            {formatVnd(selected.amount_vnd)}
          </p>
          {otpSending ? (
            <div className="flex items-center gap-3 rounded-md border border-white/10 bg-white/[0.04] p-3 text-sm text-cream/70">
              <span className="h-4 w-4 animate-spin rounded-full border-2 border-gold-400 border-t-transparent" />
              Đang gửi mã OTP về điện thoại của bạn... vui lòng chờ giây lát.
            </div>
          ) : (
            <div className="rounded-md bg-white/10 p-3 text-sm text-gold-300">
              Mã OTP (môi trường demo):{" "}
              <span className="font-mono font-bold tracking-widest">{contract.otp_code}</span>
            </div>
          )}
          <Input
            label="Nhập mã OTP để ký hợp đồng"
            value={otpInput}
            placeholder={otpSending ? "Đang chờ mã..." : "Nhập 6 số"}
            onChange={(e) => setOtpInput(e.target.value)}
            disabled={otpSending}
          />
          <div className="flex gap-2">
            <Button onClick={signContract} disabled={busy || otpSending || !otpInput}>
              {busy ? "Đang ký..." : "Ký hợp đồng"}
            </Button>
            <Button variant="ghost" onClick={reset} disabled={busy}>
              Hủy
            </Button>
          </div>
        </Card>
      )}

      {step === "pay" && sign && (
        <Card className="max-w-lg space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-cream/55">Mã khoản đầu tư</p>
              <p className="text-lg font-bold text-cream">
                {sign.investment.code}
              </p>
            </div>
            <Badge tone="yellow">{statusLabel(sign.investment.status)}</Badge>
          </div>

          <div className="rounded-lg border border-white/10 bg-white/[0.04] p-4">
            <p className="mb-3 text-sm font-semibold text-cream/85">
              Chuyển khoản đến <span className="text-gold-300">tài khoản công ty</span>
            </p>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
              {settings?.company_bank_code && sign.payment.company_account && (
                /* eslint-disable-next-line @next/next/no-img-element */
                <img
                  src={`https://img.vietqr.io/image/${settings.company_bank_code}-${sign.payment.company_account}-compact2.png?amount=${sign.payment.amount_vnd}&addInfo=${encodeURIComponent(sign.payment.transfer_note)}&accountName=${encodeURIComponent(sign.payment.company_account_name)}`}
                  alt="VietQR chuyển khoản công ty"
                  className="h-44 w-44 shrink-0 rounded-lg bg-white p-2"
                />
              )}
              <dl className="flex-1 space-y-2 text-sm">
                <Row label="Ngân hàng" value={sign.payment.bank} />
                <Row label="Số tài khoản" value={sign.payment.company_account} mono />
                <Row label="Chủ tài khoản" value={sign.payment.company_account_name} />
                <Row label="Số tiền" value={formatVnd(sign.payment.amount_vnd)} />
                <Row label="Nội dung CK" value={sign.payment.transfer_note} mono />
              </dl>
            </div>
          </div>

          <p className="text-xs leading-relaxed text-cream/55">
            Quét mã VietQR hoặc chuyển khoản thủ công. Vui lòng giữ{" "}
            <strong className="text-cream/80">đúng nội dung chuyển khoản</strong>{" "}
            (mã {sign.investment.code}) để được ghi nhận đúng khoản góp vốn cổ
            phần. Tiền vào tài khoản pháp nhân công ty, đảm bảo chứng từ &amp; an
            toàn về thuế. Sau khi chuyển khoản, bấm nút bên dưới.
          </p>

          <div className="flex gap-2">
            <Button onClick={declareTransfer} disabled={busy}>
              {busy ? "Đang gửi..." : "Tôi đã chuyển khoản"}
            </Button>
          </div>
        </Card>
      )}

      {step === "done" && sign && (
        <Card className="max-w-lg space-y-3">
          <p className="text-sm font-semibold text-green-700">
            Đã ghi nhận thông báo chuyển khoản.
          </p>
          <p className="text-sm text-cream/70">
            Khoản đầu tư <strong>{sign.investment.code}</strong> đang chờ quản
            trị viên đối soát và phê duyệt. Bạn có thể theo dõi trạng thái tại
            bảng điều khiển.
          </p>
          {transferDeclared && (
            <Badge tone="blue">Đã báo chuyển khoản</Badge>
          )}
          <div className="flex gap-2 pt-2">
            <Link href="/dashboard">
              <Button>Về bảng điều khiển</Button>
            </Link>
            <Button variant="secondary" onClick={reset}>
              Đầu tư thêm
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}

function Row({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="text-cream/55">{label}</dt>
      <dd
        className={`text-right font-medium text-cream ${
          mono ? "font-mono" : ""
        }`}
      >
        {value}
      </dd>
    </div>
  );
}

function Stepper({ step }: { step: Step }) {
  const steps: { key: Step; label: string }[] = [
    { key: "choose", label: "Chọn gói" },
    { key: "otp", label: "Ký OTP" },
    { key: "pay", label: "Chuyển khoản" },
    { key: "done", label: "Hoàn tất" },
  ];
  const order: Step[] = ["choose", "otp", "pay", "done"];
  const currentIdx = order.indexOf(step);
  return (
    <ol className="flex flex-wrap items-center gap-2 text-xs">
      {steps.map((s, i) => {
        const active = i === currentIdx;
        const passed = i < currentIdx;
        return (
          <li key={s.key} className="flex items-center gap-2">
            <span
              className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold ${
                active
                  ? "bg-gold-500 text-forest-950"
                  : passed
                  ? "bg-green-500 text-white"
                  : "bg-white/10 text-cream/55"
              }`}
            >
              {i + 1}
            </span>
            <span
              className={active ? "font-semibold text-cream" : "text-cream/55"}
            >
              {s.label}
            </span>
            {i < steps.length - 1 && (
              <span className="mx-1 text-cream/40">›</span>
            )}
          </li>
        );
      })}
    </ol>
  );
}

export default function InvestPage() {
  return (
    <Guard requireRole="investor">
      <InvestInner />
    </Guard>
  );
}
