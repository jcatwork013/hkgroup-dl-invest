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

// Thương hiệu & nội dung hiển thị trên web bán hàng duoclieuhk.vn (shop đọc public).
const BRAND: Field[] = [
  { key: "brand_name", label: "Tên thương hiệu", placeholder: "HKGROUP" },
  { key: "brand_tagline", label: "Slogan (tagline)", placeholder: "Dược liệu lên men" },
  { key: "hero_subtitle", label: "Mô tả trang chủ (hero)", placeholder: "Công thức cổ truyền kết hợp công nghệ lên men hiện đại..." },
  { key: "footer_about", label: "Giới thiệu (footer)", placeholder: "Dược liệu lên men theo công thức cổ truyền..." },
  { key: "seo_title", label: "SEO — Tiêu đề", placeholder: "HKGROUP — Dược liệu lên men..." },
  { key: "seo_description", label: "SEO — Mô tả", placeholder: "Mô tả ngắn cho công cụ tìm kiếm" },
  { key: "seo_keywords", label: "SEO — Từ khoá", placeholder: "dược liệu, lên men, thảo dược" },
  { key: "social_facebook", label: "Facebook URL", placeholder: "https://facebook.com/..." },
  { key: "social_youtube", label: "YouTube URL", placeholder: "https://youtube.com/@..." },
  { key: "social_zalo", label: "Zalo URL", placeholder: "https://zalo.me/..." },
];

const ALL = [...CONTACT, ...COMPANY, ...BRAND];

// Cơ chế hoa hồng bán hàng — lưu dạng thập phân (0.25), nhập/hiển thị dạng % (25).
const SALES_RATE_FIELDS: { key: string; label: string }[] = [
  { key: "sales_seller_rate", label: "Người bán / đơn" },
  { key: "sales_affiliate_rate", label: "Affiliate giới thiệu" },
  { key: "sales_equalshare_rate", label: "Đồng chia" },
  { key: "sales_pool_rate", label: "Pool cổ đông" },
  { key: "sales_cost_rate", label: "Giá vốn sản phẩm" },
  { key: "sales_operations_rate", label: "Vận hành & phát triển" },
];
const SALES_RATE_KEYS = SALES_RATE_FIELDS.map((f) => f.key);
const pctOf = (dec: string | undefined) => {
  const n = parseFloat(dec ?? "");
  return Number.isFinite(n) ? String(Math.round(n * 1000) / 10) : "";
};
const decOf = (pct: string) => {
  const n = parseFloat(pct);
  return Number.isFinite(n) ? String(Math.round(n * 100) / 10000) : "0";
};

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
  const [logoUploading, setLogoUploading] = useState(false);

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
    // Logo (URL ảnh công khai) — upload hoặc dán URL.
    body["brand_logo_url"] = form["brand_logo_url"] ?? "";
    // Cơ chế hoa hồng bán hàng (thập phân) + ngưỡng + KPI.
    SALES_RATE_KEYS.forEach((k) => (body[k] = form[k] ?? ""));
    body["sales_equalshare_min"] = form["sales_equalshare_min"] ?? "";
    body["sales_kpi_monthly_target"] = form["sales_kpi_monthly_target"] ?? "";
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

  async function onLogoFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setError(null);
    setLogoUploading(true);
    try {
      const res = await adminApi.uploadProductImage(file); // trả { id, url } (ảnh công khai)
      set("brand_logo_url", res.url);
    } catch (err) {
      setError((err as ApiException).message);
    } finally {
      setLogoUploading(false);
      e.target.value = "";
    }
  }

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
            <p className="text-xs text-cream/40">
              Đây là các thông số đầu tư đang áp dụng (chỉ xem). Cơ chế hoa hồng BÁN HÀNG chỉnh ở khối bên dưới.
            </p>
          </div>
        )}
      </Card>

      {isLoading ? (
        <Spinner />
      ) : (
        <form onSubmit={handleSave} className="space-y-6">
          <div className="grid items-start gap-6 lg:grid-cols-2">
          <Card className="space-y-4">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
              Thông tin liên hệ
            </h2>
            <div className="grid gap-4 sm:grid-cols-2">
              {CONTACT.map((f) => (
                <Input
                  key={f.key}
                  label={f.label}
                  placeholder={f.placeholder}
                  value={form[f.key] ?? ""}
                  onChange={(e) => set(f.key, e.target.value)}
                />
              ))}
            </div>
          </Card>

          {/* Thương hiệu & nội dung web bán hàng duoclieuhk.vn */}
          <Card className="space-y-4">
            <div>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
                Thương hiệu &amp; nội dung web bán hàng
              </h2>
              <p className="mt-1 text-xs leading-relaxed text-cream/50">
                Logo, tên thương hiệu, SEO &amp; nội dung hiển thị trên{" "}
                <strong className="text-cream/80">duoclieuhk.vn</strong>. Lưu xong đồng bộ sang web bán hàng (tối đa ~1 phút).
              </p>
            </div>

            {/* Logo */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-cream/70">Logo thương hiệu</label>
              <div className="flex items-center gap-3">
                <div className="flex h-16 w-16 items-center justify-center overflow-hidden rounded-lg bg-white/90 ring-1 ring-white/10">
                  {form["brand_logo_url"] ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={form["brand_logo_url"]} alt="logo" className="h-full w-full object-contain" />
                  ) : (
                    <span className="text-xs text-ink/40">Chưa có</span>
                  )}
                </div>
                <div className="space-y-1">
                  <label className="inline-flex cursor-pointer items-center rounded-lg border border-white/15 px-3 py-1.5 text-xs text-cream/80 hover:bg-white/5">
                    {logoUploading ? "Đang tải..." : "Tải logo lên"}
                    <input type="file" accept="image/*" className="hidden" onChange={onLogoFile} disabled={logoUploading} />
                  </label>
                  {form["brand_logo_url"] && (
                    <button type="button" onClick={() => set("brand_logo_url", "")} className="block text-xs text-red-300 hover:underline">
                      Gỡ logo
                    </button>
                  )}
                </div>
              </div>
              <Input
                label="hoặc dán URL logo"
                placeholder="/api/v1/public/images/... hoặc https://..."
                value={form["brand_logo_url"] ?? ""}
                onChange={(e) => set("brand_logo_url", e.target.value)}
              />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              {BRAND.map((f) => (
                <Input
                  key={f.key}
                  label={f.label}
                  placeholder={f.placeholder}
                  value={form[f.key] ?? ""}
                  onChange={(e) => set(f.key, e.target.value)}
                />
              ))}
            </div>
          </Card>

          {/* Cơ chế hoa hồng bán hàng (mỗi đơn) — admin cấu hình % + KPI */}
          <Card className="space-y-4">
            <div>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
                Cơ chế hoa hồng bán hàng (mỗi đơn)
              </h2>
              <p className="mt-1 text-xs leading-relaxed text-cream/50">
                Chia doanh số mỗi đơn hoàn tất. Tổng các mục{" "}
                <strong className="text-cream/80">nên = 100%</strong>. Áp dụng cho hoa hồng người bán &amp; affiliate.
              </p>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              {SALES_RATE_FIELDS.map((f) => (
                <label key={f.key} className="block">
                  <span className="mb-1.5 block text-xs font-medium text-cream/70">{f.label} (%)</span>
                  <input
                    type="number"
                    step="0.5"
                    min="0"
                    max="100"
                    value={pctOf(form[f.key])}
                    onChange={(e) => set(f.key, decOf(e.target.value))}
                    className="w-full rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm text-cream focus:border-gold-500 focus:outline-none"
                  />
                </label>
              ))}
            </div>
            {(() => {
              const total = SALES_RATE_KEYS.reduce((a, k) => a + (parseFloat(pctOf(form[k])) || 0), 0);
              const ok = Math.round(total * 10) / 10 === 100;
              return (
                <p className={`text-xs font-medium ${ok ? "text-green-300" : "text-gold-300"}`}>
                  Tổng chia: {Math.round(total * 10) / 10}% {ok ? "✓" : "(nên = 100%)"}
                </p>
              );
            })()}
            <Input
              label="Ngưỡng đồng chia — chỉ chia 'đồng chia' khi đơn ≥ (VNĐ)"
              type="number"
              value={form["sales_equalshare_min"] ?? ""}
              onChange={(e) => set("sales_equalshare_min", e.target.value)}
            />
            <Input
              label="KPI doanh số / tháng cho CTV (VNĐ)"
              type="number"
              placeholder="vd 50000000"
              value={form["sales_kpi_monthly_target"] ?? ""}
              onChange={(e) => set("sales_kpi_monthly_target", e.target.value)}
            />
          </Card>

          <Card className="space-y-4">
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
            <div className="grid gap-4 sm:grid-cols-2">
              {COMPANY.map((f) => (
                <Input
                  key={f.key}
                  label={f.label}
                  placeholder={f.placeholder}
                  value={form[f.key] ?? ""}
                  onChange={(e) => set(f.key, e.target.value)}
                />
              ))}
            </div>
          </Card>

          <Card className="space-y-4">
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

          <Card className="space-y-4">
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

          </div>

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
