"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { useAuth } from "@/components/AuthContext";
import { authApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { Button, Card, ErrorText, Input } from "@/components/ui";

function SecurityInner() {
  const { user } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      setDone(true);
      setCurrent("");
      setNext("");
      setConfirm("");
      setError(null);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setDone(false);
    if (next.length < 8) {
      setError("Mật khẩu mới phải có ít nhất 8 ký tự.");
      return;
    }
    if (next !== confirm) {
      setError("Xác nhận mật khẩu không khớp.");
      return;
    }
    if (next === current) {
      setError("Mật khẩu mới phải khác mật khẩu hiện tại.");
      return;
    }
    change.mutate();
  }

  return (
    <div className="mx-auto max-w-md space-y-6">
      <div>
        <h1 className="font-serif text-2xl text-cream">Bảo mật tài khoản</h1>
        <p className="mt-1 text-sm text-cream/55">
          Đổi mật khẩu đăng nhập cho tài khoản{" "}
          <span className="text-cream/80">{user?.email}</span>. Mật khẩu được mã
          hoá bcrypt và không bao giờ lưu dạng văn bản thường.
        </p>
      </div>

      <ErrorText>{error}</ErrorText>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Mật khẩu hiện tại"
            type="password"
            autoComplete="current-password"
            required
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
          />
          <Input
            label="Mật khẩu mới (tối thiểu 8 ký tự)"
            type="password"
            autoComplete="new-password"
            required
            minLength={8}
            value={next}
            onChange={(e) => setNext(e.target.value)}
          />
          <Input
            label="Xác nhận mật khẩu mới"
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <div className="flex items-center gap-3 pt-1">
            <Button type="submit" disabled={change.isPending}>
              {change.isPending ? "Đang cập nhật..." : "Đổi mật khẩu"}
            </Button>
            {done && <span className="text-sm text-gold-300">Đã đổi mật khẩu ✓</span>}
          </div>
        </form>
      </Card>
    </div>
  );
}

export default function SecurityPage() {
  // No requireRole — investor AND admin can manage their own password.
  return (
    <Guard>
      <SecurityInner />
    </Guard>
  );
}
