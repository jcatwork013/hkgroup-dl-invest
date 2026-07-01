"use client";

import { useState } from "react";
import Guard from "@/components/Guard";
import { Badge, Card, Eyebrow } from "@/components/ui";

// =============================================================================
// Quy trình & Chính sách — trang TÀI LIỆU (chỉ xem) để verify flow + cơ chế %.
//  • Mục 1: ĐẦU TƯ CỔ PHẦN (đang chạy) — hoa hồng F1/F2/F3 + cổ tức 9%+6% theo hạng.
//  • Mục 2: BÁN HÀNG (thiết kế chờ duyệt) — chia 1 đơn hàng thành 6 khoản (100%).
// Không gọi API; mọi con số bám đúng code backend hiện tại / chính sách đã chốt.
// =============================================================================

type Seg = { label: string; pct: number; color: string; to: string; note?: string };

// Thanh chia % (stacked bar) — trực quan hoá việc bổ tiền theo từng khoản.
function SplitBar({ segs }: { segs: Seg[] }) {
  const total = segs.reduce((s, x) => s + x.pct, 0);
  return (
    <div className="space-y-3">
      <div className="flex h-9 w-full overflow-hidden rounded-lg border border-white/10">
        {segs.map((s) => (
          <div
            key={s.label}
            className={`flex items-center justify-center text-[11px] font-semibold text-forest-950 ${s.color}`}
            style={{ width: `${s.pct}%` }}
            title={`${s.label} — ${s.pct}%`}
          >
            {s.pct >= 8 ? `${s.pct}%` : ""}
          </div>
        ))}
      </div>
      <p className="text-right text-xs text-cream/45">Tổng cộng: {total}%</p>
    </div>
  );
}

// Bảng chi tiết từng khoản: %, chi cho ai, điều kiện.
function SplitTable({ segs }: { segs: Seg[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr>
            <th className="text-left">Khoản</th>
            <th className="text-right">Tỷ lệ</th>
            <th className="text-left">Đích đến</th>
          </tr>
        </thead>
        <tbody>
          {segs.map((s) => (
            <tr key={s.label}>
              <td>
                <span className={`mr-2 inline-block h-2.5 w-2.5 rounded-full align-middle ${s.color}`} />
                <span className="font-medium text-cream">{s.label}</span>
              </td>
              <td className="text-right font-mono font-semibold text-gold-300">{s.pct}%</td>
              <td className="text-cream/70">
                {s.to}
                {s.note && <span className="block text-xs text-cream/40">{s.note}</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// Một bước trong quy trình (timeline dọc, có số thứ tự + mũi tên nối).
function Step({ n, title, children, last }: { n: number; title: string; children: React.ReactNode; last?: boolean }) {
  return (
    <div className="relative pl-12">
      <div className="absolute left-0 top-0 flex h-9 w-9 items-center justify-center rounded-full border border-gold-500/40 bg-gold-500/10 font-serif text-sm font-semibold text-gold-300">
        {n}
      </div>
      {!last && <div className="absolute left-[17px] top-9 h-[calc(100%-1rem)] w-px bg-white/10" />}
      <p className="font-semibold text-cream">{title}</p>
      <div className="mt-1 pb-6 text-sm leading-relaxed text-cream/65">{children}</div>
    </div>
  );
}

// "Thẻ" cơ chế nhỏ (vd 1 cấp hoa hồng) dùng trong sơ đồ cây F1/F2/F3.
function Chip({ label, sub, tone = "gold" }: { label: string; sub?: string; tone?: "gold" | "ghost" }) {
  const cls = tone === "gold" ? "border-gold-500/40 bg-gold-500/10 text-gold-200" : "border-white/15 bg-white/5 text-cream/80";
  return (
    <div className={`rounded-xl border px-4 py-2 text-center ${cls}`}>
      <p className="text-sm font-semibold">{label}</p>
      {sub && <p className="text-xs opacity-70">{sub}</p>}
    </div>
  );
}

const TAX_NOTE = "Hoa hồng bị khấu trừ 10% thuế TNCN → người nhận thực lãnh = gross − 10%.";

// --------------------------- MỤC 1: ĐẦU TƯ ----------------------------------
function InvestmentPolicy() {
  return (
    <div className="space-y-6">
      <Card>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-serif text-lg text-cream">Quy trình đầu tư cổ phần</h2>
          <Badge tone="green">Đang vận hành</Badge>
        </div>
        <div>
          <Step n={1} title="Đăng ký + gắn người giới thiệu">
            Nhà đầu tư đăng ký, nhập mã giới thiệu → lưu <b>1 cấp trực tiếp</b> vào bảng <code>referrals</code>
            (referee → referrer). Cấp 2/3 suy ra bằng cách leo cây lúc tính hoa hồng (không lưu sẵn).
          </Step>
          <Step n={2} title="Chọn gói + ký hợp đồng (OTP)">
            Chọn gói đầu tư → ký bằng OTP → tạo <code>investment</code> trạng thái <Badge tone="yellow">Chờ duyệt</Badge> kèm
            thông tin chuyển khoản.
          </Step>
          <Step n={3} title="Chuyển khoản + khai báo">Nhà đầu tư chuyển tiền và khai báo đã chuyển.</Step>
          <Step n={4} title="Admin đối soát">Admin xác nhận đã nhận tiền → <Badge tone="green">Đã đối soát</Badge>.</Step>
          <Step n={5} title="Admin duyệt → phát cổ phần + sinh hoa hồng">
            Khi duyệt (<Badge tone="green">Đã duyệt</Badge>): hệ thống phát cổ phần (ghi <code>share_ledger</code> + cập nhật
            <code> shareholdings</code>) <b>và</b> sinh hoa hồng Đại Lý cấp 1/2/3 trong cùng 1 giao dịch.
          </Step>
          <Step n={6} title="Chi hoa hồng">
            Hoa hồng: <Badge tone="yellow">Chờ duyệt</Badge> → admin duyệt → admin chi (<Badge tone="green">Đã chi</Badge>).
            Người nhận rút qua ví hoa hồng.
          </Step>
          <Step n={7} title="Chia cổ tức định kỳ" last>
            Admin nhập doanh thu kỳ → chia <b>pool cổ đông</b> cho mọi nhà đầu tư theo cơ chế 9% đồng chia + 6% bonus hạng (xem dưới).
          </Step>
        </div>
      </Card>

      <Card>
        <h2 className="mb-1 font-serif text-lg text-cream">Cơ chế hoa hồng giới thiệu — Đại Lý cấp 1 / 2 / 3</h2>
        <p className="mb-5 text-sm text-cream/60">
          Tính trên <b>số tiền đầu tư</b> của người mua, leo cây lên tối đa 3 cấp (có chặn tự giới thiệu / vòng lặp).
        </p>

        {/* Sơ đồ cây: Người mua → Đại Lý cấp 1 → cấp 2 → cấp 3 */}
        <div className="mb-6 flex flex-wrap items-center gap-3">
          <Chip label="Người mua" sub="đầu tư X đồng" tone="ghost" />
          <span className="text-gold-400">→</span>
          <Chip label="Đại Lý cấp 1 · 3%" sub="người giới thiệu trực tiếp" />
          <span className="text-gold-400">→</span>
          <Chip label="Đại Lý cấp 2 · 2%" sub="người giới thiệu của cấp 1" />
          <span className="text-gold-400">→</span>
          <Chip label="Đại Lý cấp 3 · 1%" sub="người giới thiệu của cấp 2" />
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr>
                <th className="text-left">Cấp</th>
                <th className="text-right">Tỷ lệ</th>
                <th className="text-right">VD đơn 100.000.000đ</th>
                <th className="text-right">Thuế 10%</th>
                <th className="text-right">Thực nhận</th>
              </tr>
            </thead>
            <tbody>
              {[
                { lv: "Đại Lý cấp 1", rate: 3, gross: 3_000_000 },
                { lv: "Đại Lý cấp 2", rate: 2, gross: 2_000_000 },
                { lv: "Đại Lý cấp 3", rate: 1, gross: 1_000_000 },
              ].map((r) => (
                <tr key={r.lv}>
                  <td><Badge tone="blue">{r.lv}</Badge></td>
                  <td className="text-right font-mono text-gold-300">{r.rate}%</td>
                  <td className="text-right font-mono">{r.gross.toLocaleString("vi-VN")}đ</td>
                  <td className="text-right font-mono text-red-300">−{(r.gross * 0.1).toLocaleString("vi-VN")}đ</td>
                  <td className="text-right font-mono font-semibold text-cream">
                    {(r.gross * 0.9).toLocaleString("vi-VN")}đ
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="mt-3 text-xs text-cream/45">{TAX_NOTE} Nguồn tỷ lệ: gói (customer) hoặc cấu hình <code>referral_f*_rate</code> (investor).</p>
      </Card>

      <Card>
        <h2 className="mb-1 font-serif text-lg text-cream">Cơ chế chia cổ tức — đồng chia 9% + bonus 6% theo hạng</h2>
        <p className="mb-5 text-sm text-cream/60">
          Trên pool cổ đông mỗi kỳ (mặc định 15% doanh thu). Khác hoàn toàn với hoa hồng giới thiệu.
        </p>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="rounded-xl border border-white/10 bg-white/[0.03] p-4">
            <p className="font-semibold text-cream">Đồng chia — 9%</p>
            <p className="mt-1 text-sm text-cream/60">Chia <b>đều</b> cho mọi cổ đông đang hoạt động (mỗi người 1 phần bằng nhau).</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/[0.03] p-4">
            <p className="font-semibold text-cream">Bonus theo hạng — 6%</p>
            <ul className="mt-1 space-y-1 text-sm text-cream/60">
              <li>• Hạng 1: 5–49 triệu → <span className="font-mono text-gold-300">1,5%</span></li>
              <li>• Hạng 2: 50–299 triệu → <span className="font-mono text-gold-300">2,0%</span></li>
              <li>• Hạng 3: 300 triệu+ → <span className="font-mono text-gold-300">2,5%</span></li>
            </ul>
          </div>
        </div>
      </Card>
    </div>
  );
}

// --------------------------- MỤC 2: BÁN HÀNG --------------------------------
// 25% = thưởng người bán, "tự chia BÊN NGOÀI". 5 khoản còn lại (75%) = "Dòng Tiền Hệ Thống".
const SALES_SEGS: Seg[] = [
  { label: "Người bán (mỗi đơn phát sinh)", pct: 25, color: "bg-gold-400", to: "Thưởng trực tiếp người bán — chia NGOÀI (−10% TNCN)" },
  { label: "Đồng chia", pct: 5, color: "bg-brand-300", to: "Pool RIÊNG — chia đều cho MỌI người mua đơn ≥ 1tr (không phải cổ đông)", note: "chỉ khi đơn ≥ 1.000.000đ" },
  { label: "Affiliate & giới thiệu", pct: 10, color: "bg-gold-500", to: "Ví hoa hồng affiliate 1 cấp (−10% TNCN)" },
  { label: "Giá vốn sản phẩm", pct: 30, color: "bg-forest-600", to: "Chi phí — giá vốn từ danh mục sản phẩm" },
  { label: "Vận hành & phát triển", pct: 15, color: "bg-forest-700", to: "Chi phí hệ thống" },
  { label: "Pool Cổ Đông HK", pct: 15, color: "bg-brand-500", to: "Chia cho CỔ ĐÔNG: 9% đồng chia + 6% bonus theo hạng" },
];

function SalesPolicy() {
  return (
    <div className="space-y-6">
      <Card>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-serif text-lg text-cream">Quy trình bán hàng</h2>
          <Badge tone="green">Đang vận hành</Badge>
        </div>
        <div>
          <Step n={1} title="Admin tạo danh mục + sản phẩm">
            Mỗi sản phẩm có <b>giá bán</b> và <b>giá vốn</b> (giá vốn ≈ phần 30%). Ảnh dán link hoặc upload trực tiếp.
          </Step>
          <Step n={2} title="Tạo đơn hàng">
            <b>Người bán (saler)</b> tạo đơn cho khách, <b>hoặc</b> admin nhập đơn thủ công (chọn người bán + affiliate).
            Đơn ở trạng thái <Badge tone="yellow">Chờ thanh toán</Badge>.
          </Step>
          <Step n={3} title="Xác nhận thanh toán">
            Đơn <Badge tone="green">Đã thanh toán</Badge> → hệ thống chia dòng tiền (1 giao dịch, tổng đúng 100%).
          </Step>
          <Step n={4} title="Thưởng người bán 25% (chia ngoài)">
            25% thưởng trực tiếp người bán → ví hoa hồng (−10% TNCN). Đây là phần <b>tự chia bên ngoài</b> Dòng Tiền Hệ Thống.
          </Step>
          <Step n={5} title="Affiliate 10%">10% cho người giới thiệu khách (1 cấp) → ví hoa hồng (−10% TNCN).</Step>
          <Step n={6} title="Pool Cổ Đông 15%">
            15% → <b>Pool Cổ Đông HK</b>, chia cho cổ đông theo <b>9% đồng chia + 6% bonus theo hạng</b>.
          </Step>
          <Step n={7} title="Pool đồng chia 5% (người mua ≥1tr)">
            5% (chỉ đơn ≥ 1tr) → pool <b>RIÊNG</b> chia đều cho <b>mọi người mua</b> — KHÔNG phải cổ đông.
          </Step>
          <Step n={8} title="Chi phí 45%" last>30% giá vốn + 15% vận hành & phát triển → ghi nhận chi phí.</Step>
        </div>
      </Card>

      <Card>
        <h2 className="mb-1 font-serif text-lg text-cream">Cơ chế chia 1 đơn hàng (100%)</h2>
        <p className="mb-5 text-sm text-cream/60">
          <b>25%</b> thưởng người bán tự chia <b>bên ngoài</b>; <b>75% còn lại</b> = “Dòng Tiền Hệ Thống” (5 khoản dưới).
        </p>
        <div className="mb-6"><SplitBar segs={SALES_SEGS} /></div>
        <SplitTable segs={SALES_SEGS} />
        <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div className="rounded-xl border border-gold-500/30 bg-gold-500/[0.06] p-3 text-center">
            <p className="text-xs text-cream/55">Người bán (chia ngoài)</p>
            <p className="font-serif text-2xl font-semibold text-gold-300">25%</p>
          </div>
          <div className="rounded-xl border border-brand-500/30 bg-brand-500/[0.08] p-3 text-center">
            <p className="text-xs text-cream/55">Pool Cổ Đông (9%+6%)</p>
            <p className="font-serif text-2xl font-semibold text-brand-200">15%</p>
          </div>
          <div className="rounded-xl border border-brand-300/30 bg-brand-300/[0.10] p-3 text-center">
            <p className="text-xs text-cream/55">Pool người mua (≥1tr)</p>
            <p className="font-serif text-2xl font-semibold text-brand-200">5%</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/[0.04] p-3 text-center">
            <p className="text-xs text-cream/55">Affiliate + Chi phí</p>
            <p className="font-serif text-2xl font-semibold text-cream/80">55%</p>
          </div>
        </div>
        <p className="mt-4 text-xs text-cream/45">
          VD đơn 2.000.000đ: người bán 500k (−50k thuế = 450k) · affiliate 200k (−20k = 180k) · pool cổ đông 300k ·
          pool người mua 100k · giá vốn 600k · vận hành 300k. (Nếu đơn không có affiliate, 10% dồn vào vận hành.)
        </p>
      </Card>
    </div>
  );
}

// --------------------------------- TRANG ------------------------------------
function PolicyInner() {
  const [tab, setTab] = useState<"investment" | "sales">("investment");
  return (
    <div className="space-y-6">
      <div>
        <Eyebrow>Quy trình & Chính sách</Eyebrow>
        <h1 className="mt-2 font-serif text-2xl text-cream">Sơ đồ vận hành & cơ chế chia %</h1>
        <p className="mt-1 text-sm text-cream/55">Tài liệu nội bộ để đối chiếu/verify. Chỉ xem, không chỉnh sửa tại đây.</p>
      </div>

      <div className="inline-flex rounded-xl border border-white/10 bg-white/[0.03] p-1">
        {[
          { k: "investment", label: "Đầu tư cổ phần" },
          { k: "sales", label: "Bán hàng" },
        ].map((t) => (
          <button
            key={t.k}
            onClick={() => setTab(t.k as "investment" | "sales")}
            className={`rounded-lg px-5 py-2 text-sm font-medium transition ${
              tab === t.k ? "bg-gold-500/20 text-gold-200" : "text-cream/60 hover:text-cream"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "investment" ? <InvestmentPolicy /> : <SalesPolicy />}
    </div>
  );
}

export default function PolicyPage() {
  return (
    <Guard requireRole="admin">
      <PolicyInner />
    </Guard>
  );
}
