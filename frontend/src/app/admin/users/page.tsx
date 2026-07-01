"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatDate, formatVnd } from "@/lib/format";
import {
  Badge,
  Button,
  Card,
  ErrorText,
  Input,
  Spinner,
  statusTone,
  statusLabel,
} from "@/components/ui";
import type { AdminUser } from "@/lib/types";

function UsersInner() {
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [detail, setDetail] = useState<AdminUser | null>(null);
  const [form, setForm] = useState({
    full_name: "", phone: "", email: "", password: "", role: "investor",
  });

  const { data: users, isLoading } = useQuery({
    queryKey: ["admin-users"],
    queryFn: adminApi.users,
  });

  const { data: profile, isLoading: profileLoading } = useQuery({
    queryKey: ["admin-user-profile", detail?.id],
    queryFn: () => adminApi.userProfile(detail!.id),
    enabled: !!detail,
  });

  // Hoa hồng (ví + danh sách) của user đang xem — CHỈ XEM, admin không sửa được ở đây.
  const { data: comm, isLoading: commLoading } = useQuery({
    queryKey: ["admin-user-commissions", detail?.id],
    queryFn: () => adminApi.userCommissions(detail!.id),
    enabled: !!detail,
  });

  const refresh = () => qc.invalidateQueries({ queryKey: ["admin-users"] });

  const create = useMutation({
    mutationFn: () => adminApi.createUser(form),
    onSuccess: () => {
      refresh();
      setForm({ full_name: "", phone: "", email: "", password: "", role: "investor" });
    },
    onError: (e) => setError((e as ApiException).message),
  });

  const del = useMutation({
    mutationFn: (id: string) => adminApi.deleteUser(id),
    onSuccess: () => {
      refresh();
      setDetail(null);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  function handleDelete(u: AdminUser) {
    const ok = window.confirm(
      `Xoá TOÀN BỘ tài khoản "${u.full_name}" (${u.email})?\n\n` +
        "Hành động KHÔNG THỂ hoàn tác. Xoá cả dữ liệu tài chính: cổ phần, đầu tư, cổ tức đã nhận, " +
        "hoa hồng (gồm hoa hồng đã trả cho người giới thiệu) và yêu cầu rút tiền. " +
        "Cổ phần sẽ được trả lại pool."
    );
    if (!ok) return;
    setError(null);
    del.mutate(u.id);
  }

  const reset = useMutation({
    mutationFn: (id: string) => adminApi.resetUserPassword(id),
    onSuccess: () => window.alert("Đã gửi email chứa link đặt lại mật khẩu cho người dùng."),
    onError: (e) => setError((e as ApiException).message),
  });

  function handleReset(u: AdminUser) {
    if (!window.confirm(`Gửi email đặt lại mật khẩu tới "${u.full_name}" (${u.email})?`)) return;
    setError(null);
    reset.mutate(u.id);
  }

  function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!form.full_name || !form.phone || !form.email || form.password.length < 8) {
      setError("Điền đủ thông tin; mật khẩu tối thiểu 8 ký tự.");
      return;
    }
    create.mutate();
  }

  const set = (k: string, v: string) => setForm((s) => ({ ...s, [k]: v }));

  return (
    <div className="space-y-8">
      <div>
        <h1 className="font-serif text-2xl text-cream">Quản lý người dùng</h1>
        <p className="mt-1 text-sm text-cream/55">
          Mặc định tài khoản là <strong className="text-cream/80">investor</strong>.
          Chỉ admin mới tạo được tài khoản &amp; cấp quyền{" "}
          <strong className="text-cream/80">admin</strong>.
        </p>
      </div>

      <ErrorText>{error}</ErrorText>

      <Card className="max-w-2xl space-y-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
          Tạo tài khoản mới
        </h2>
        <form onSubmit={handleCreate} className="grid gap-4 sm:grid-cols-2">
          <Input label="Họ tên" value={form.full_name} onChange={(e) => set("full_name", e.target.value)} />
          <Input label="Số điện thoại" value={form.phone} onChange={(e) => set("phone", e.target.value)} />
          <Input label="Email" type="email" value={form.email} onChange={(e) => set("email", e.target.value)} />
          <Input label="Mật khẩu (≥ 8 ký tự)" type="password" value={form.password} onChange={(e) => set("password", e.target.value)} />
          <label className="block">
            <span className="mb-1.5 block text-sm font-medium text-cream/80">Vai trò</span>
            <select
              value={form.role}
              onChange={(e) => set("role", e.target.value)}
              className="w-full rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm text-cream focus:border-gold-500 focus:outline-none"
            >
              <option value="investor">Investor (nhà đầu tư)</option>
              <option value="saler">Saler (nhân viên bán hàng)</option>
              <option value="admin">Admin (quản trị)</option>
            </select>
          </label>
          <div className="flex items-end">
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? "Đang tạo..." : "Tạo tài khoản"}
            </Button>
          </div>
        </form>
      </Card>

      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">Danh sách người dùng</h2>
        {isLoading ? (
          <Spinner />
        ) : (
          <Card className="overflow-x-auto p-0">
            <table>
              <thead>
                <tr>
                  <th>Họ tên</th>
                  <th>Email / SĐT</th>
                  <th>Vai trò</th>
                  <th>Ngày tạo</th>
                  <th>Hành động</th>
                </tr>
              </thead>
              <tbody>
                {users?.map((u) => (
                  <tr key={u.id}>
                    <td className="font-medium text-cream">{u.full_name}</td>
                    <td>
                      {u.email}
                      <span className="block text-xs text-cream/45">{u.phone}</span>
                    </td>
                    <td><Badge tone={u.role === "admin" ? "yellow" : u.role === "saler" ? "blue" : "slate"}>{u.role === "admin" ? "Quản trị viên" : u.role === "saler" ? "Nhân viên bán hàng" : "Nhà đầu tư"}</Badge></td>
                    <td>{formatDate(u.created_at)}</td>
                    <td>
                      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 whitespace-nowrap text-xs font-medium">
                        <ActionLink onClick={() => setDetail(u)}>Chi tiết</ActionLink>
                        <ActionLink onClick={() => handleReset(u)} disabled={reset.isPending}>
                          Đặt lại MK
                        </ActionLink>
                        {u.role === "investor" && (
                          <ActionLink tone="danger" onClick={() => handleDelete(u)} disabled={del.isPending}>
                            Xoá
                          </ActionLink>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        )}
      </section>

      {detail && (
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-cream">
              Hồ sơ: {detail.full_name}
            </h2>
            <Button variant="ghost" onClick={() => setDetail(null)}>Đóng</Button>
          </div>

          {profileLoading ? (
            <Spinner />
          ) : (
            <Card className="grid gap-x-8 gap-y-3 sm:grid-cols-2">
              <Field label="Email" value={detail.email} />
              <Field label="Số điện thoại" value={detail.phone} />
              <Field label="Ngày sinh" value={profile?.date_of_birth} />
              <Field label="Giới tính" value={profile?.gender} />
              <Field label="Quốc tịch" value={profile?.nationality} />
              <Field label="Nghề nghiệp" value={profile?.occupation} />
              <Field label="Số CCCD/CMND" value={profile?.cccd_number} />
              <Field label="Ngày cấp" value={profile?.cccd_issue_date} />
              <Field label="Nơi cấp" value={profile?.cccd_issue_place} />
              <Field label="Mã số thuế" value={profile?.tax_code} />
              <Field label="Địa chỉ thường trú" value={profile?.permanent_address} />
              <Field label="Địa chỉ liên hệ" value={profile?.contact_address} />
              <Field label="Ngân hàng nhận cổ tức" value={profile?.bank_name} />
              <Field label="Số tài khoản" value={profile?.bank_account_number} />
              <Field label="Chủ tài khoản" value={profile?.bank_account_name} />
            </Card>
          )}

          {/* Hoa hồng của tài khoản — CHỈ XEM (admin không sửa ở đây). */}
          <Card className="mt-4 space-y-4">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
              Hoa hồng giới thiệu (chỉ xem)
            </h3>
            {commLoading ? (
              <Spinner />
            ) : (
              <>
                <div className="grid gap-4 sm:grid-cols-3">
                  <Field label="Tổng hoa hồng (thực nhận)" value={formatVnd(comm?.wallet.earned_vnd ?? 0)} />
                  <Field label="Đã rút / đang chờ" value={formatVnd(comm?.wallet.withdrawn_vnd ?? 0)} />
                  <Field label="Số dư khả dụng" value={formatVnd(comm?.wallet.available_vnd ?? 0)} />
                </div>
                <div className="overflow-x-auto">
                  <table>
                    <thead>
                      <tr>
                        <th>Cấp</th>
                        <th>Cơ sở tính</th>
                        <th>Tỷ lệ</th>
                        <th>Hoa hồng</th>
                        <th>Thuế TNCN</th>
                        <th>Thực nhận</th>
                        <th>Trạng thái</th>
                      </tr>
                    </thead>
                    <tbody>
                      {comm?.commissions.map((c) => (
                        <tr key={c.id}>
                          <td><Badge tone="blue">F{c.level}</Badge></td>
                          <td>{formatVnd(c.base_amount)}</td>
                          <td>{(c.rate * 100).toFixed(2)}%</td>
                          <td>{formatVnd(c.amount)}</td>
                          <td>{formatVnd(c.tax_pit)}</td>
                          <td className="font-medium text-cream">{formatVnd(c.net_amount)}</td>
                          <td><Badge tone={statusTone(c.status)}>{statusLabel(c.status)}</Badge></td>
                        </tr>
                      ))}
                      {(!comm || comm.commissions.length === 0) && (
                        <tr>
                          <td colSpan={7} className="text-center text-cream/45">Chưa có hoa hồng.</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </Card>
        </section>
      )}
    </div>
  );
}

// Compact text-link action for table rows — gọn & tinh tế hơn các nút pill.
function ActionLink({
  tone = "default",
  children,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { tone?: "default" | "gold" | "danger" }) {
  const tones = {
    default: "text-cream/65 hover:text-cream",
    gold: "text-gold-300 hover:text-gold-200",
    danger: "text-red-400/80 hover:text-red-300",
  } as const;
  return (
    <button
      className={`${tones[tone]} transition disabled:cursor-not-allowed disabled:opacity-40 focus:outline-none focus:underline`}
      {...props}
    >
      {children}
    </button>
  );
}

function Field({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <p className="text-xs uppercase tracking-wide text-cream/45">{label}</p>
      <p className="mt-0.5 text-sm text-cream/90">{value || "—"}</p>
    </div>
  );
}

export default function AdminUsersPage() {
  return (
    <Guard requireRole="admin">
      <UsersInner />
    </Guard>
  );
}
