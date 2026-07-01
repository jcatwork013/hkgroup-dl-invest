"use client";

import { useRouter } from "next/navigation";
import Link from "next/link";
import { useState } from "react";
import { useAuth } from "@/components/AuthContext";
import { authApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { Button, Card, ErrorText, Input } from "@/components/ui";

export default function LoginPage() {
  const router = useRouter();
  const { setSession } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await authApi.login(email, password);
      setSession(res);
      router.push(res.user.role === "admin" ? "/admin" : "/dashboard");
    } catch (err) {
      setError(
        err instanceof ApiException ? err.message : "Đăng nhập thất bại"
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-md">
      <Card>
        <h1 className="mb-1 text-xl font-bold text-cream">Đăng nhập</h1>
        <p className="mb-5 text-sm text-cream/55">
          Truy cập tài khoản cổ đông của bạn.
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
          <Input
            label="Mật khẩu"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
          />
          <ErrorText>{error}</ErrorText>
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "Đang xử lý..." : "Đăng nhập"}
          </Button>
        </form>
        <p className="mt-3 text-center text-sm">
          <Link href="/forgot-password" className="text-cream/55 hover:text-gold-300">
            Quên mật khẩu?
          </Link>
        </p>
        <p className="mt-4 text-center text-sm text-cream/55">
          Chưa có tài khoản?{" "}
          <Link href="/register" className="font-medium text-gold-300">
            Đăng ký
          </Link>
        </p>
      </Card>
    </div>
  );
}
