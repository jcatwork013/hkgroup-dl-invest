"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/components/AuthContext";
import { authApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { Button, Card, ErrorText, Input } from "@/components/ui";
import RiskDisclaimer from "@/components/RiskDisclaimer";

function RegisterForm() {
  const router = useRouter();
  const params = useSearchParams();
  const { setSession } = useAuth();

  const refFromQuery = params.get("ref") ?? "";

  const [fullName, setFullName] = useState("");
  const [phone, setPhone] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [referralCode, setReferralCode] = useState(refFromQuery);
  // referral_type defaults to "customer" when a referral code is present.
  const [referralType, setReferralType] = useState("customer");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await authApi.register({
        full_name: fullName,
        phone,
        email,
        password,
        referral_code: referralCode || undefined,
        referral_type: referralCode ? referralType : undefined,
      });
      setSession(res);
      router.push(res.user.role === "admin" ? "/admin" : "/dashboard");
    } catch (err) {
      setError(err instanceof ApiException ? err.message : "Đăng ký thất bại");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-md space-y-5">
      <Card>
        <h1 className="mb-1 text-xl font-bold text-cream">
          Đăng ký làm cổ đông
        </h1>
        <p className="mb-5 text-sm text-cream/55">
          Tạo tài khoản để bắt đầu quá trình góp vốn.
        </p>
        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Họ và tên"
            value={fullName}
            onChange={(e) => setFullName(e.target.value)}
            required
          />
          <Input
            label="Số điện thoại"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            required
          />
          <Input
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
          <Input
            label="Mật khẩu"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="new-password"
          />
          <Input
            label="Mã giới thiệu (nếu có)"
            value={referralCode}
            onChange={(e) => setReferralCode(e.target.value)}
            placeholder="Tùy chọn"
          />
          {referralCode && (
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-cream/85">
                Loại giới thiệu
              </span>
              <select
                value={referralType}
                onChange={(e) => setReferralType(e.target.value)}
                className="w-full rounded-md border border-white/15 px-3 py-2 text-sm text-cream focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              >
                <option value="customer">Khách hàng (customer)</option>
                <option value="investor">Nhà đầu tư (investor)</option>
              </select>
            </label>
          )}
          <ErrorText>{error}</ErrorText>
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "Đang xử lý..." : "Đăng ký"}
          </Button>
        </form>
        <p className="mt-4 text-center text-sm text-cream/55">
          Đã có tài khoản?{" "}
          <Link href="/login" className="font-medium text-gold-300">
            Đăng nhập
          </Link>
        </p>
      </Card>
      <RiskDisclaimer />
    </div>
  );
}

export default function RegisterPage() {
  return (
    <Suspense fallback={<div className="py-10 text-center text-sm text-cream/55">Đang tải...</div>}>
      <RegisterForm />
    </Suspense>
  );
}
