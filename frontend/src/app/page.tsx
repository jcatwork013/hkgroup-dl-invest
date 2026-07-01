"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { publicApi } from "@/lib/endpoints";
import { useAuth } from "@/components/AuthContext";
import { formatNumber, formatPct, formatVnd } from "@/lib/format";
import { Button, Card, Eyebrow, Spinner, statusLabel } from "@/components/ui";

const FEATURES = [
  {
    title: "Minh bạch tuyệt đối",
    desc: "Mọi cổ phần phát hành đều ghi vào sổ bút toán bất biến, đối soát được bất cứ lúc nào.",
  },
  {
    title: "Hợp đồng số ký OTP",
    desc: "Ký hợp đồng điện tử xác thực OTP, sinh PDF và mã hợp đồng riêng cho mỗi nhà đầu tư.",
  },
  {
    title: "Đối soát ngân hàng",
    desc: "Tiền góp vào tài khoản pháp nhân công ty; cổ phần chỉ phát hành sau khi đối soát tiền thực về.",
  },
  {
    title: "Cổ tức thực chia",
    desc: "Quyền lợi nhận lại (nếu có) đến từ cổ tức công ty thực sự công bố và chi trả — không cam kết lợi nhuận.",
  },
];

export default function LandingPage() {
  const { user, ready } = useAuth();
  const home = user?.role === "admin" ? "/admin" : "/dashboard";
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["offering"],
    queryFn: publicApi.offering,
  });

  const offering = data?.offering;
  const soldPct =
    offering && offering.shares_for_sale > 0
      ? Math.min(100, (offering.shares_sold / offering.shares_for_sale) * 100)
      : 0;
  const offerPct =
    offering && offering.total_shares > 0
      ? (offering.shares_for_sale / offering.total_shares) * 100
      : 49;

  return (
    <div className="space-y-16">
      {/* HERO */}
      <section className="grid items-center gap-10 pt-4 md:grid-cols-2">
        <div className="space-y-6">
          <Eyebrow>HKGroup · Chào bán cổ phần · Cổ phần đơn hàng ký gửi</Eyebrow>
          <h1 className="font-serif text-4xl leading-[1.08] text-cream md:text-[3.25rem]">
            Dược liệu lên men,{" "}
            <span className="italic text-gold-400">tinh hoa</span> từ thiên nhiên
            Việt
          </h1>
          <p className="max-w-lg text-base leading-relaxed text-cream/70">
            Trở thành cổ đông HKGroup — góp vốn sở hữu cổ phần doanh nghiệp một
            cách minh bạch. Toàn bộ ký hợp đồng, đối soát chuyển khoản và cơ cấu
            sở hữu đều được ghi nhận rõ ràng.
          </p>
          <div className="flex flex-wrap gap-3">
            {ready && user ? (
              <>
                <Link href={home}>
                  <Button>Vào bảng điều khiển</Button>
                </Link>
                <Link href="/invest">
                  <Button variant="secondary">Đầu tư ngay</Button>
                </Link>
              </>
            ) : (
              <>
                <Link href="/register">
                  <Button>Đăng ký làm cổ đông</Button>
                </Link>
                <Link href="/login">
                  <Button variant="secondary">Đăng nhập</Button>
                </Link>
              </>
            )}
          </div>
        </div>

        <Card className="space-y-5 ring-1 ring-gold-500/10">
          {isLoading && <Spinner />}
          {isError && (
            <p className="text-sm text-red-300">
              Không tải được thông tin đợt chào bán:{" "}
              {(error as Error)?.message}
            </p>
          )}
          {offering && (
            <>
              <p className="text-xs uppercase tracking-[0.2em] text-gold-400">
                Đợt chào bán hiện tại
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-xs uppercase tracking-wide text-cream/45">
                    Định giá đơn hàng doanh nghiệp
                  </p>
                  <p className="mt-1 font-serif text-2xl font-semibold text-cream">
                    {formatVnd(offering.valuation_vnd)}
                  </p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-cream/45">
                    Tỷ lệ chào bán
                  </p>
                  <p className="mt-1 font-serif text-2xl font-semibold text-gold-400">
                    {formatPct(offerPct)}
                  </p>
                </div>
              </div>

              <div>
                <div className="mb-1.5 flex items-center justify-between text-xs text-cream/55">
                  <span>
                    Đã bán {formatNumber(offering.shares_sold)} /{" "}
                    {formatNumber(offering.shares_for_sale)} cổ phần đơn hàng
                  </span>
                  <span className="text-gold-300">{soldPct.toFixed(1)}%</span>
                </div>
                <div className="h-2.5 w-full overflow-hidden rounded-full bg-white/10">
                  <div
                    className="h-full rounded-full bg-gradient-to-r from-gold-500 to-gold-300 transition-all"
                    style={{ width: `${soldPct}%` }}
                  />
                </div>
              </div>

              <p className="text-xs text-cream/40">
                Tổng số cổ phần đơn hàng: {formatNumber(offering.total_shares)} • Trạng
                thái: {statusLabel(offering.status)}
              </p>
            </>
          )}
        </Card>
      </section>

      {/* FEATURES */}
      <section className="space-y-6">
        <div className="text-center">
          <Eyebrow>Vì sao chọn HKGroup</Eyebrow>
          <h2 className="mt-3 font-serif text-3xl text-cream">
            Đầu tư cổ phần minh bạch, đúng pháp lý
          </h2>
        </div>
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {FEATURES.map((f, i) => (
            <Card
              key={f.title}
              className="transition hover:border-gold-500/30 hover:bg-white/[0.06]"
            >
              <span className="font-serif text-2xl text-gold-400">
                0{i + 1}
              </span>
              <h3 className="mt-3 text-base font-semibold text-cream">
                {f.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-cream/60">
                {f.desc}
              </p>
            </Card>
          ))}
        </div>
      </section>

      {/* CTA */}
      <section>
        <Card className="relative overflow-hidden text-center ring-1 ring-gold-500/15">
          <h2 className="font-serif text-2xl text-cream">
            {ready && user ? "Tiếp tục hành trình cổ đông của bạn" : "Quan tâm trở thành cổ đông?"}
          </h2>
          <p className="mx-auto mt-2 max-w-xl text-sm text-cream/65">
            {ready && user
              ? "Theo dõi cổ phần, cổ tức và các gói đầu tư ngay trên bảng điều khiển của bạn."
              : "Vui lòng đăng ký / đăng nhập để được tư vấn và xem thông tin tham gia cổ phần."}
          </p>
          <div className="mt-5 flex justify-center gap-3">
            {ready && user ? (
              <Link href={home}>
                <Button>Vào bảng điều khiển</Button>
              </Link>
            ) : (
              <>
                <Link href="/register">
                  <Button>Đăng ký</Button>
                </Link>
                <Link href="/login">
                  <Button variant="secondary">Đăng nhập</Button>
                </Link>
              </>
            )}
          </div>
        </Card>
      </section>
    </div>
  );
}
