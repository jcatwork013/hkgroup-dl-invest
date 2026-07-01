"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatVnd } from "@/lib/format";
import { Badge, Button, Card, ErrorText, Input, Spinner } from "@/components/ui";

type Field = { key: string; label: string; placeholder: string };

// Cấu hình hệ thống — CHỈ XEM (không sửa ở đây). format: pct = tỷ lệ %, vnd = số tiền, onoff = Bật/Tắt.
type ConfigRow = { key: string; label: string; fmt: "pct" | "vnd" | "onoff" | "mode" };
const COMMISSION_CONFIG: ConfigRow[] = [
  { key: "referral_f1_rate", label: "Hoa hồng Đại lý 1 (F1)", fmt: "pct" },
  { key: "referral_f2_rate", label: "Hoa hồng Đại lý 2 (F2)", fmt: "pct" },
  { key: "referral_f3_rate", label: "Hoa hồng Đại lý 3 (F3)", fmt: "pct" },
  { key: "referral_investor_cash", label: "Trả hoa hồng cho nhà đầu tư", fmt: "onoff" },
];
const DIST_CONFIG: ConfigRow[] = [
  { key: "pool_rate", label: "Tỷ lệ chia cổ đông (trên doanh thu)", fmt: "pct" },
  { key: "dist_equal_rate", label: "Phân bổ — đồng chia", fmt: "pct" },
  { key: "dist_bonus_rate", label: "Phân bổ — bonus theo bậc", fmt: "pct" },
  { key: "dist_band1_max", label: "Bậc 1 — ngưỡng tối đa", fmt: "vnd" },
  { key: "dist_band1_rate", label: "Bậc 1 — tỷ lệ", fmt: "pct" },
  { key: "dist_band2_max", label: "Bậc 2 — ngưỡng tối đa", fmt: "vnd" },
  { key: "dist_band2_rate", label: "Bậc 2 — tỷ lệ", fmt: "pct" },
  { key: "dist_band3_rate", label: "Bậc 3 — tỷ lệ", fmt: "pct" },
  { key: "dist_residual_mode", label: "Xử lý phần dư", fmt: "mode" },
  { key: "show_pool_public", label: "Hiển thị pool công khai", fmt: "onoff" },
];

function fmtConfig(v: string | undefined, fmt: ConfigRow["fmt"]): string {
  if (v === undefined || v === "") return "—";
  if (fmt === "pct") {
    const n = parseFloat(v);
    return Number.isFinite(n) ? `${(n * 100).toFixed(2)}%` : v;
  }
  if (fmt === "vnd") {
    const n = parseInt(v, 10);
    return Number.isFinite(n) ? formatVnd(n) : v;
  }
  if (fmt === "onoff") return v === "on" ? "Bật" : "Tắt";
  if (fmt === "mode") return v === "rollover" ? "Chuyển sang đợt sau (rollover)" : v === "retain" ? "Giữ lại (retain)" : v;
  return v;
}

const SALES_CONFIG: ConfigRow[] = [
  { key: "sales_seller_rate", label: "Người bán (mỗi đơn)", fmt: "pct" },
  { key: "sales_affiliate_rate", label: "Affiliate giới thiệu", fmt: "pct" },
  { key: "sales_equalshare_rate", label: "Đồng chia", fmt: "pct" },
  { key: "sales_pool_rate", label: "Pool cổ đông", fmt: "pct" },
  { key: "sales_cost_rate", label: "Giá vốn sản phẩm", fmt: "pct" },
  { key: "sales_operations_rate", label: "Vận hành & phát triển", fmt: "pct" },
  { key: "sales_equalshare_min", label: "Ngưỡng đồng chia (đơn ≥)", fmt: "vnd" },
];

function ConfigGrid({ rows, data }: { rows: ConfigRow[]; data: Record<string, string> }) {
  return (
    <div className="grid gap-x-6 gap-y-1 sm:grid-cols-2 lg:grid-cols-3">
      {rows.map((r) => (
        <div key={r.key} className="flex items-center justify-between gap-3 border-b border-white/5 py-1.5">
          <span className="text-xs text-cream/60">{r.label}</span>
          <span className="font-mono text-xs font-medium text-cream">{fmtConfig(data[r.key], r.fmt)}</span>
        </div>
      ))}
    </div>
  );
}

const CONTACT: Field[] = [
  { key: "contact_hotline", label: "Hotline", placeholder: "0948 579 759" },
  { key: "contact_email", label: "Email", placeholder: "info@duoclieuhk.vn" },
  { key: "contact_address", label: "Địa chỉ", placeholder: "Số 18B1 Đường B1, ..." },
  { key: "brand_since", label: "Năm thành lập (Since)", placeholder: "2026" },
];

const COMPANY: Field[] = [
  { key: "company_bank_name", label: "Ngân hàng (hiển thị)", placeholder: "Vietcombank" },
  { key: "company_bank_code", label: "Mã ngân hàng VietQR", placeholder: "VCB / VPB / TCB / 970436" },
  { key: "company_account", label: "Số tài khoản CÔNG TY", placeholder: "Số tài khoản pháp nhân" },
  { key: "company_account_name", label: "Chủ tài khoản (tên công ty)", placeholder: "CONG TY CO PHAN DUOC LIEU HK" },
];

const ALL = [...CONTACT, ...COMPANY];

// Chuẩn hoá CSV ngày rút -> mảng ngày hợp lệ (1..31, unique, sorted). Rỗng -> mặc định 15,30.
function parseDays(s: string | undefined): number[] {
  const seen = new Set<number>();
  for (const p of (s ?? "").split(",")) {
    const n = parseInt(p.trim(), 10);
    if (Number.isInteger(n) && n >= 1 && n <= 31) seen.add(n);
  }
  const out = [...seen].sort((a, b) => a - b);
  return out.length ? out : [15, 30];
}

function daysLabel(ds: number[]): string {
  if (ds.length === 1) return `ngày ${ds[0]}`;
  return `ngày ${ds.slice(0, -1).join(", ")} và ${ds[ds.length - 1]}`;
}

function SettingsInner() {
  const qc = useQueryClient();
  const [form, setForm] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-settings"],
    queryFn: adminApi.settings,
  });

  useEffect(() => {
    if (data) setForm((f) => ({ ...data, ...f }));
  }, [data]);

  const save = useMutation({
    mutationFn: (body: Record<string, string>) => adminApi.updateSettings(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-settings"] });
      qc.invalidateQueries({ queryKey: ["settings"] });
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const body: Record<string, string> = {};
    ALL.forEach((f) => (body[f.key] = form[f.key] ?? ""));
    // Lịch rút tiền: lưu dạng CSV đã chuẩn hoá (vd "15,30").
    body["withdrawal_days"] = parseDays(form["withdrawal_days"]).join(",");
    // Cấu hình Resend (đặt lại mật khẩu). API key là write-only: chỉ gửi khi admin
    // nhập giá trị mới; để trống = giữ nguyên key cũ (backend bỏ qua secret rỗng).
    body["resend_from_email"] = form["resend_from_email"] ?? "";
    body["resend_from_name"] = form["resend_from_name"] ?? "";
    body["app_base_url"] = form["app_base_url"] ?? "";
    if ((form["resend_api_key"] ?? "").trim() !== "") {
      body["resend_api_key"] = form["resend_api_key"].trim();
    }
    save.mutate(body);
  }

  const set = (k: string, v: string) => setForm((s) => ({ ...s, [k]: v }));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-serif text-2xl text-cream">Thiết lập website</h1>
        <p className="mt-1 text-sm text-cream/55">
          Thông tin liên hệ &amp; tài khoản công ty nhận góp vốn. Lưu lại sẽ cập
          nhật ngay trên toàn site.
        </p>
      </div>

      <ErrorText>{error}</ErrorText>

      {/* Cấu hình hệ thống — CHỈ XEM, không sửa ở đây. */}
      <Card className="space-y-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
            Cấu hình hệ thống
          </h2>
          <Badge tone="slate">Chỉ xem</Badge>
        </div>
        {isLoading || !data ? (
          <Spinner />
        ) : (
          <div className="space-y-6">
            <div className="space-y-3">
              <p className="text-xs uppercase tracking-wide text-cream/45">Hoa hồng giới thiệu (đại lý)</p>
              <ConfigGrid rows={COMMISSION_CONFIG} data={data} />
            </div>
            <div className="space-y-3">
              <p className="text-xs uppercase tracking-wide text-cream/45">Pool &amp; phân bổ doanh thu</p>
              <ConfigGrid rows={DIST_CONFIG} data={data} />
            </div>
            <div className="space-y-3">
              <p className="text-xs uppercase tracking-wide text-cream/45">Chia dòng tiền bán hàng (mỗi đơn)</p>
              <ConfigGrid rows={SALES_CONFIG} data={data} />
            </div>
            <p className="text-xs text-cream/40">
              Đây là các thông số đang áp dụng. Để thay đổi tỷ lệ hoa hồng/phân bổ cần chỉnh ở cơ sở dữ liệu — màn này chỉ hiển thị để kiểm soát.
            </p>
          </div>
        )}
      </Card>

      {isLoading ? (
        <Spinner />
      ) : (
        <form onSubmit={handleSave} className="space-y-6">
          <Card className="max-w-xl space-y-4">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
              Thông tin liên hệ
            </h2>
            {CONTACT.map((f) => (
              <Input
                key={f.key}
                label={f.label}
                placeholder={f.placeholder}
                value={form[f.key] ?? ""}
                onChange={(e) => set(f.key, e.target.value)}
              />
            ))}
          </Card>

          <Card className="max-w-xl space-y-4">
            <div>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
                Tài khoản công ty nhận góp vốn
              </h2>
              <p className="mt-1 text-xs leading-relaxed text-cream/50">
                ⚠️ Bắt buộc là <strong className="text-cream/80">tài khoản pháp
                nhân của công ty</strong> (không dùng tài khoản cá nhân) và{" "}
                <strong className="text-cream/80">tài khoản mà công ty kiểm soát
                được</strong> — để đối soát dòng tiền chính xác. Mã VietQR dùng để
                sinh QR chuyển khoản tự động.
              </p>
            </div>
            {COMPANY.map((f) => (
              <Input
                key={f.key}
                label={f.label}
                placeholder={f.placeholder}
                value={form[f.key] ?? ""}
                onChange={(e) => set(f.key, e.target.value)}
              />
            ))}
          </Card>

          <Card className="max-w-xl space-y-4">
            <div>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
                Lịch rút tiền (ví hoa hồng)
              </h2>
              <p className="mt-1 text-xs leading-relaxed text-cream/50">
                Các <strong className="text-cream/80">ngày trong tháng</strong> mà nhà đầu tư/đại lý được gửi yêu cầu rút
                (mở cửa lúc <strong className="text-cream/80">00h00</strong>). Nhập danh sách ngày, cách nhau bằng dấu phẩy.
                Mặc định <strong className="text-cream/80">15, 30</strong>. Nếu tháng không có ngày đó (vd 30/2) thì tính vào
                ngày cuối tháng.
              </p>
            </div>
            <Input
              label="Ngày rút (cách nhau bằng dấu phẩy)"
              placeholder="15,30"
              inputMode="numeric"
              value={form["withdrawal_days"] ?? ""}
              onChange={(e) => set("withdrawal_days", e.target.value)}
            />
            <p className="text-xs text-cream/55">
              Áp dụng: <span className="font-medium text-gold-300">{daysLabel(parseDays(form["withdrawal_days"]))}</span> hàng tháng.
            </p>
          </Card>

          <Card className="max-w-xl space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
                Gửi email — Resend (đặt lại mật khẩu)
              </h2>
              {data?.["resend_api_key_configured"] === "true" ? (
                <Badge tone="green">Đã cấu hình</Badge>
              ) : (
                <Badge tone="slate">Chưa cấu hình</Badge>
              )}
            </div>
            <p className="text-xs leading-relaxed text-cream/50">
              Cần thiết để tính năng <strong className="text-cream/80">đặt lại mật khẩu</strong> hoạt động.
              Lấy API key tại <span className="text-cream/70">resend.com</span>, và dùng địa chỉ gửi thuộc{" "}
              <strong className="text-cream/80">tên miền đã xác thực</strong> trên Resend. Nếu bỏ trống, người dùng sẽ
              không gửi được yêu cầu đặt lại mật khẩu.
            </p>
            <Input
              label="Resend API Key"
              type="password"
              placeholder={
                data?.["resend_api_key_configured"] === "true"
                  ? "•••••••• (đã lưu — nhập để thay key mới)"
                  : "re_..."
              }
              autoComplete="off"
              value={form["resend_api_key"] ?? ""}
              onChange={(e) => set("resend_api_key", e.target.value)}
            />
            <Input
              label="Email người gửi (From)"
              placeholder="no-reply@duoclieuhk.vn"
              value={form["resend_from_email"] ?? ""}
              onChange={(e) => set("resend_from_email", e.target.value)}
            />
            <Input
              label="Tên người gửi (From name)"
              placeholder="Dược Liệu HK"
              value={form["resend_from_name"] ?? ""}
              onChange={(e) => set("resend_from_name", e.target.value)}
            />
            <Input
              label="URL gốc của web (cho link đặt lại)"
              placeholder="https://duoclieuhk.vn"
              value={form["app_base_url"] ?? ""}
              onChange={(e) => set("app_base_url", e.target.value)}
            />
            <p className="text-xs text-cream/45">
              API key chỉ lưu phía máy chủ và không bao giờ trả về trang quản trị (write-only). Để trống ô API key
              khi lưu sẽ giữ nguyên key đã cấu hình.
            </p>
          </Card>

          <div className="flex items-center gap-3">
            <Button type="submit" disabled={save.isPending}>
              {save.isPending ? "Đang lưu..." : "Lưu thay đổi"}
            </Button>
            {saved && <span className="text-sm text-gold-300">Đã lưu ✓</span>}
          </div>
        </form>
      )}
    </div>
  );
}

export default function AdminSettingsPage() {
  return (
    <Guard requireRole="admin">
      <SettingsInner />
    </Guard>
  );
}
