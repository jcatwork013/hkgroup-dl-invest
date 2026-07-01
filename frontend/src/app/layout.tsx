import type { Metadata } from "next";
import { Playfair_Display, Be_Vietnam_Pro } from "next/font/google";
import "./globals.css";
import Providers from "@/components/Providers";
import { AuthProvider } from "@/components/AuthContext";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

const serif = Playfair_Display({
  subsets: ["latin", "latin-ext", "vietnamese"],
  weight: ["500", "600", "700"],
  style: ["normal", "italic"],
  variable: "--font-serif",
  display: "swap",
});

const sans = Be_Vietnam_Pro({
  subsets: ["latin", "latin-ext", "vietnamese"],
  weight: ["300", "400", "500", "600", "700"],
  variable: "--font-sans",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Dược Liệu HK — Chào bán cổ phần",
  description:
    "Nền tảng chào bán cổ phần riêng lẻ HKGroup. Đầu tư cổ phần tiềm ẩn rủi ro, không cam kết lợi nhuận.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="vi" className={`${serif.variable} ${sans.variable}`}>
      <body>
        <div className="leaf-overlay" aria-hidden />
        <div className="relative z-10 min-h-[100dvh]">
          <Providers>
            <AuthProvider>
              <Nav />
              {/* Sidebar trái rộng 16rem (lg+); nội dung lùi sang phải bằng lg:pl-64. */}
              <div className="flex min-h-[100dvh] flex-col lg:pl-64">
                <main className="w-full flex-1 px-4 py-10 sm:px-6 lg:px-10">
                  {children}
                </main>
                <Footer />
              </div>
            </AuthProvider>
          </Providers>
        </div>
      </body>
    </html>
  );
}
