"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { authApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { Button, Card, ErrorText, Input, Spinner } from "@/components/ui";

function ResetPasswordInner() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token") ?? "";

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (password.length < 8) {
      setError("Mật khẩu mới phải có ít nhất 8 ký tự.");
      return;
    }
    if (password !== confirm) {
      setError("Mật khẩu xác nhận không khớp.");
      return;
    }
    setLoading(true);
    try {
      await authApi.resetPassword(token, password);
      setDone(true);
      setTimeout(() => router.push("/login"), 2000);
    } catch (err) {
      setError(err instanceof ApiException ? err.message : "Đặt lại mật khẩu thất bại");
    } finally {
      setLoading(false);
    }
  }

  if (!token) {
    return (
      <Card>
        <h1 className="mb-2 text-xl font-bold text-cream">Liên kết không hợp lệ</h1>
        <p className="text-sm text-cream/55">
          Thiếu mã đặt lại. Vui lòng dùng đúng liên kết trong email, hoặc{" "}
          <Link href="/forgot-password" className="font-medium text-gold-300">
            yêu cầu liên kết mới
          </Link>
          .
        </p>
      </Card>
    );
  }

  return (
    <Card>
      <h1 className="mb-1 text-xl font-bold text-cream">Đặt lại mật khẩu</h1>
      {done ? (
        <div className="mt-4 rounded-lg border border-green-500/30 bg-green-500/[0.08] px-4 py-3 text-sm text-green-200">
          Đổi mật khẩu thành công! Đang chuyển tới trang đăng nhập...
        </div>
      ) : (
        <>
          <p className="mb-5 text-sm text-cream/55">Nhập mật khẩu mới cho tài khoản của bạn.</p>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              label="Mật khẩu mới"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="new-password"
            />
            <Input
              label="Xác nhận mật khẩu mới"
              type="password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              required
              autoComplete="new-password"
            />
            <ErrorText>{error}</ErrorText>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Đang xử lý..." : "Đặt lại mật khẩu"}
            </Button>
          </form>
        </>
      )}
    </Card>
  );
}

export default function ResetPasswordPage() {
  return (
    <div className="mx-auto max-w-md">
      <Suspense fallback={<Spinner />}>
        <ResetPasswordInner />
      </Suspense>
    </div>
  );
}
