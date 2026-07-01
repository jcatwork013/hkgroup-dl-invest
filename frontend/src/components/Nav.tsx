"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "./AuthContext";
import Logo from "./Logo";
import { Button } from "./ui";

type Item = { href: string; label: string };
type Section = { title?: string; items: Item[] };

const INVESTOR: Section[] = [
  {
    items: [
      { href: "/", label: "Tổng quan" },
      { href: "/dashboard", label: "Bảng điều khiển" },
      { href: "/profile", label: "Hồ sơ" },
      { href: "/invest", label: "Đầu tư" },
      { href: "/referral", label: "Giới thiệu" },
      { href: "/security", label: "Bảo mật" },
    ],
  },
];

const SALER: Section[] = [
  {
    items: [
      { href: "/", label: "Tổng quan" },
      { href: "/sales/orders", label: "Bán hàng" },
      { href: "/sales/commissions", label: "Hoa hồng & Ví" },
      { href: "/security", label: "Bảo mật" },
    ],
  },
];

const ADMIN: Section[] = [
  {
    items: [
      { href: "/admin", label: "Bảng điều khiển" },
      { href: "/admin/policy", label: "Quy trình & Chính sách" },
    ],
  },
  {
    title: "Bán hàng",
    items: [
      { href: "/admin/orders", label: "Đơn hàng" },
      { href: "/admin/products", label: "Sản phẩm" },
      { href: "/admin/categories", label: "Danh mục" },
      { href: "/admin/salers", label: "Giám sát Saler" },
    ],
  },
  {
    title: "Đầu tư",
    items: [
      { href: "/admin/tiers", label: "Gói đầu tư" },
      { href: "/admin/reconcile", label: "Đối soát" },
      { href: "/admin/cap-table", label: "Cơ cấu sở hữu" },
    ],
  },
  {
    title: "Tài chính",
    items: [
      { href: "/admin/distribution", label: "Pool & Phân bổ" },
      { href: "/admin/dividends", label: "Cổ tức" },
      { href: "/admin/withdrawals", label: "Rút tiền" },
    ],
  },
  {
    title: "Hệ thống",
    items: [
      { href: "/admin/users", label: "Người dùng" },
      { href: "/admin/audit", label: "Nhật ký" },
      { href: "/admin/settings", label: "Thiết lập" },
      { href: "/security", label: "Bảo mật" },
    ],
  },
];

function menuFor(role: string | undefined, signedIn: boolean): Section[] {
  if (!signedIn) return [{ items: [{ href: "/", label: "Tổng quan" }] }];
  if (role === "admin") return ADMIN;
  if (role === "saler") return SALER;
  return INVESTOR;
}

export default function Nav() {
  const { user, ready, logout } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const [open, setOpen] = useState(false); // chỉ dùng cho mobile

  const sections = menuFor(user?.role, ready && !!user);
  const allItems = sections.flatMap((s) => s.items);
  // Chỉ mục khớp dài nhất mới active (để /admin không sáng cùng /admin/products).
  const activeHref = allItems
    .filter((it) => pathname === it.href || (it.href !== "/" && pathname.startsWith(it.href + "/")))
    .reduce<string | null>((best, it) => (it.href.length > (best?.length ?? -1) ? it.href : best), null);
  const isActive = (p: string) => p === activeHref;

  const handleLogout = () => {
    logout();
    setOpen(false);
    router.push("/");
  };

  // Đóng menu mobile khi đổi route.
  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  const roleLabel = user?.role === "admin" ? "Quản trị viên" : user?.role === "saler" ? "Nhân viên bán hàng" : "Nhà đầu tư";

  // Nội dung sidebar (dùng chung cho desktop cố định + mobile drawer).
  const SidebarBody = (
    <>
      <div className="flex items-center justify-between border-b border-white/10 px-5 py-4">
        <Link href="/" className="flex items-center gap-2">
          <Logo imgClass="h-7" />
        </Link>
        <button
          onClick={() => setOpen(false)}
          aria-label="Đóng menu"
          className="flex h-9 w-9 items-center justify-center rounded-lg text-cream/60 transition hover:bg-white/5 hover:text-cream lg:hidden"
        >
          <span className="text-lg leading-none">✕</span>
        </button>
      </div>

      {ready && user && (
        <div className="border-b border-white/10 px-5 py-4">
          <p className="truncate text-sm font-semibold text-cream">{user.full_name}</p>
          <p className="mt-0.5 text-xs text-cream/45">{roleLabel}</p>
        </div>
      )}

      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {sections.map((sec, i) => (
          <div key={i} className="mb-2">
            {sec.title && (
              <p className="px-4 pb-1 pt-3 text-[11px] font-semibold uppercase tracking-wider text-cream/35">{sec.title}</p>
            )}
            <ul className="flex flex-col gap-0.5">
              {sec.items.map((it) => (
                <li key={it.href}>
                  <Link
                    href={it.href}
                    className={`flex items-center justify-between rounded-xl px-4 py-2 text-sm font-medium transition ${
                      isActive(it.href)
                        ? "bg-gold-500/15 text-gold-300"
                        : "text-cream/75 hover:bg-white/5 hover:text-cream"
                    }`}
                  >
                    {it.label}
                    {isActive(it.href) && <span className="h-1.5 w-1.5 rounded-full bg-gold-400" />}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </nav>

      <div className="border-t border-white/10 p-4">
        {ready && !user && (
          <div className="flex items-center gap-2">
            <Link href="/login" className="flex-1">
              <Button className="w-full px-3 py-2 text-[13px] whitespace-nowrap">
                Đăng nhập
              </Button>
            </Link>
            <Link href="/register" className="flex-1">
              <Button
                variant="secondary"
                className="w-full px-3 py-2 text-[13px] whitespace-nowrap"
              >
                Đăng ký
              </Button>
            </Link>
          </div>
        )}
        {ready && user && (
          <Button
            variant="ghost"
            onClick={handleLogout}
            className="w-full border border-white/10 text-cream/75"
          >
            Đăng xuất
          </Button>
        )}
      </div>
    </>
  );

  return (
    <>
      {/* Thanh trên cùng — CHỈ mobile (nút mở sidebar) */}
      <header className="sticky top-0 z-30 flex items-center justify-between border-b border-white/10 bg-forest-950/80 px-4 py-3 backdrop-blur-md lg:hidden">
        <Link href="/" className="flex items-center gap-2">
          <Logo imgClass="h-7" />
        </Link>
        <button
          onClick={() => setOpen(true)}
          aria-label="Mở menu"
          className="flex h-10 items-center gap-2 rounded-lg border border-white/10 px-3 text-cream transition hover:border-gold-500/40 hover:bg-white/5"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
            <path d="M3 6h18M3 12h18M3 18h18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
          </svg>
          <span className="text-sm font-medium">Menu</span>
        </button>
      </header>

      {/* Sidebar cố định — desktop (lg+) */}
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-64 flex-col border-r border-white/10 bg-forest-950 lg:flex">
        {SidebarBody}
      </aside>

      {/* Drawer mobile */}
      <div
        onClick={() => setOpen(false)}
        aria-hidden
        className={`fixed inset-0 z-40 bg-black/60 backdrop-blur-sm transition-opacity duration-300 lg:hidden ${
          open ? "opacity-100" : "pointer-events-none opacity-0"
        }`}
      />
      <aside
        role="dialog"
        aria-modal="true"
        className={`fixed inset-y-0 left-0 z-50 flex h-screen w-[80%] max-w-xs flex-col border-r border-white/10 bg-forest-950 shadow-[0_0_60px_-10px_rgba(0,0,0,.8)] transition-transform duration-300 ease-out lg:hidden ${
          open ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        {SidebarBody}
      </aside>
    </>
  );
}
