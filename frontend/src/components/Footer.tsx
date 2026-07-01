"use client";

import { useQuery } from "@tanstack/react-query";
import { publicApi } from "@/lib/endpoints";
import Logo from "./Logo";

export default function Footer() {
  const { data } = useQuery({
    queryKey: ["settings"],
    queryFn: publicApi.settings,
    staleTime: 5 * 60 * 1000,
  });

  const hotline = data?.contact_hotline || "0948 579 759";
  const address =
    data?.contact_address ||
    "Số 18B1 Đường B1, khu dân cư Hưng Phú, phường Hưng Phú, TP Cần Thơ";
  const email = data?.contact_email || "info@duoclieuhk.vn";
  const since = data?.brand_since || "2026";

  return (
    <footer className="border-t border-white/10 bg-forest-950/40">
      <div className="w-full px-4 py-10 sm:px-6 lg:px-10">
        <div className="grid gap-8 md:grid-cols-3">
          <div className="space-y-3">
            <Logo imgClass="h-9" />
            <p className="max-w-xs text-sm leading-relaxed text-cream/55">
              Dược liệu lên men — tinh hoa từ thiên nhiên Việt. Nền tảng chào bán
              cổ phần đơn hàng ký gửi minh bạch của HKGroup.
            </p>
          </div>

          <div className="text-sm text-cream/65">
            <p className="mb-3 font-semibold uppercase tracking-wide text-cream/80">
              Thông tin liên hệ
            </p>
            <ul className="space-y-2">
              <li>
                <span className="text-cream/45">Hotline: </span>
                <a href={`tel:${hotline.replace(/\s/g, "")}`} className="text-gold-300 hover:text-gold-200">
                  {hotline}
                </a>
              </li>
              <li>
                <span className="text-cream/45">Email: </span>
                <a href={`mailto:${email}`} className="hover:text-cream">
                  {email}
                </a>
              </li>
              <li className="leading-relaxed">
                <span className="text-cream/45">Địa chỉ: </span>
                {address}
              </li>
            </ul>
          </div>

          <div className="text-sm text-cream/55 md:text-right">
            <p className="font-serif text-lg text-gold-400">HKGroup</p>
            <p className="mt-1">{since}</p>
          </div>
        </div>

        <div className="mt-8 border-t border-white/10 pt-5 text-xs leading-relaxed text-cream/40">
          HKGroup cam kết bảo toàn quyền lợi nhà đầu tư. Phần giá trị chưa nhận
          đủ sẽ được quy đổi thành sản phẩm tương đương theo giá bán lẻ niêm yết,
          đảm bảo hoàn thành 100% quyền lợi của gói đầu tư.
        </div>
      </div>
    </footer>
  );
}
