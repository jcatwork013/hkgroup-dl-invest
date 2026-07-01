"use client";

import Link from "next/link";
import { useState } from "react";
import { authApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { Button, Card, ErrorText, Input } from "@/components/ui";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await authApi.forgotPassword(email);
      setSent(true);
    } catch (err) {
      setError(err instanceof ApiException ? err.message : "Không gửi được yêu cầu");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-md">
      <Card>
        <h1 className="mb-1 text-xl font-bold text-cream">Quên mật khẩu</h1>
        {sent ? (
          <div className="mt-4 space-y-4">
            <div className="rounded-lg border border-green-500/30 bg-green-500/[0.08] px-4 py-3 text-sm text-green-200">
              Nếu email này tồn tại trong hệ thống, chúng tôi đã gửi một liên kết đặt lại mật khẩu.
              Vui lòng kiểm tra hộp thư (kể cả mục Spam). Liên kết có hiệu lực trong <strong>1 giờ</strong>.
            </div>
            <p className="text-center text-sm">
              <Link href="/login" className="font-medium text-gold-300">
                ← Quay lại đăng nhập
              </Link>
            </p>
          </div>
        ) : (
          <>
            <p className="mb-5 text-sm text-cream/55">
              Nhập email tài khoản. Chúng tôi sẽ gửi liên kết để bạn đặt lại mật khẩu.
            </p>
            <form onSubmit={handleSubmit} className="space-y-4">
              <Input
                label="Email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoComplete="email"
              />
              <ErrorText>{error}</ErrorText>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "Đang gửi..." : "Gửi liên kết đặt lại"}
              </Button>
            </form>
            <p className="mt-4 text-center text-sm">
              <Link href="/login" className="text-cream/55 hover:text-gold-300">
                ← Quay lại đăng nhập
              </Link>
            </p>
          </>
        )}
      </Card>
    </div>
  );
}
