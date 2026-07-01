"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { useAuth } from "@/components/AuthContext";
import { investorApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { Badge, Button, Card, ErrorText, Input, Spinner } from "@/components/ui";
import type { InvestorProfile } from "@/lib/types";

const EMPTY: InvestorProfile = {
  date_of_birth: "", gender: "", nationality: "Việt Nam", cccd_number: "",
  cccd_issue_date: "", cccd_issue_place: "", permanent_address: "", contact_address: "",
  occupation: "", tax_code: "", bank_name: "", bank_account_number: "", bank_account_name: "",
};

function ProfileInner() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [form, setForm] = useState<InvestorProfile>(EMPTY);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["my-profile"],
    queryFn: investorApi.getProfile,
  });

  useEffect(() => {
    if (data) setForm((f) => ({ ...EMPTY, ...data, ...(f.cccd_number ? f : {}) }));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  const save = useMutation({
    mutationFn: (body: InvestorProfile) => investorApi.updateProfile(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["my-profile"] });
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  const set = (k: keyof InvestorProfile, v: string) =>
    setForm((s) => ({ ...s, [k]: v }));

  function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    // Số định danh (CCCD/CMND/Hộ chiếu) nay KHÔNG bắt buộc — đã gỡ KYC theo luật mới; chỉ lưu nếu
    // người dùng tự điền (phục vụ sổ cổ đông/thuế). Vẫn kiểm định dạng nếu có nhập, để tránh số rác.
    const idNo = form.cccd_number.trim().toUpperCase();
    if (idNo) {
      const isCmndOrCccd = /^\d{9}$/.test(idNo) || /^\d{12}$/.test(idNo);
      const isPassport = /^[A-Z][0-9]{7,8}$/.test(idNo);
      if (!isCmndOrCccd && !isPassport) {
        setError("Số giấy tờ không hợp lệ: CMND 9 số, CCCD 12 số, hoặc hộ chiếu (1 chữ cái + 7–8 số).");
        return;
      }
    }
    // Strip read-only fields (user_id, updated_at) — backend rejects unknown fields.
    const { ...rest } = form;
    delete (rest as Record<string, unknown>).user_id;
    delete (rest as Record<string, unknown>).updated_at;
    save.mutate(rest);
  }

  const initial = (user?.full_name || "?").trim().charAt(0).toUpperCase();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-serif text-2xl text-cream">Hồ sơ cá nhân</h1>
        <p className="mt-1 text-sm text-cream/55">
          Thông tin định danh nhà đầu tư phục vụ sổ cổ đông, khấu trừ thuế và chi
          trả cổ tức. Vui lòng điền chính xác.
        </p>
      </div>

      <ErrorText>{error}</ErrorText>

      {/* Account summary */}
      <Card className="flex flex-wrap items-center gap-5">
        <span className="flex h-16 w-16 items-center justify-center rounded-full bg-gold-500 font-serif text-2xl font-bold text-forest-950">
          {initial}
        </span>
        <div className="min-w-[200px] flex-1">
          <p className="font-serif text-xl text-cream">{user?.full_name}</p>
          <p className="text-sm text-cream/55">{user?.email} · {user?.phone}</p>
          <div className="mt-2 flex flex-wrap gap-2">
            <Badge tone={user?.role === "admin" ? "yellow" : "slate"}>
              {user?.role === "admin" ? "Quản trị viên" : "Nhà đầu tư"}
            </Badge>
          </div>
        </div>
        <div className="text-right text-sm">
          <p className="text-cream/45">Mã giới thiệu</p>
          <p className="font-mono text-gold-300">{user?.referral_code}</p>
          {data?.updated_at && (
            <p className="mt-2 text-xs text-cream/40">
              Cập nhật: {formatDate(data.updated_at)}
            </p>
          )}
        </div>
      </Card>

      {isLoading ? (
        <Spinner />
      ) : (
        <form onSubmit={handleSave} className="space-y-6">
          <Section title="Thông tin cá nhân">
            <Input label="Ngày sinh" placeholder="dd/mm/yyyy" inputMode="numeric" value={form.date_of_birth} onChange={(e) => set("date_of_birth", e.target.value)} />
            <label className="block">
              <span className="mb-1.5 block text-sm font-medium text-cream/80">Giới tính</span>
              <select
                value={form.gender}
                onChange={(e) => set("gender", e.target.value)}
                className="w-full rounded-lg border border-white/15 bg-white/5 px-3 py-2 text-sm text-cream focus:border-gold-500 focus:outline-none"
              >
                <option value="">— Chọn —</option>
                <option value="male">Nam</option>
                <option value="female">Nữ</option>
                <option value="other">Khác</option>
              </select>
            </label>
            <Input label="Quốc tịch" value={form.nationality} onChange={(e) => set("nationality", e.target.value)} />
            <Input label="Nghề nghiệp" value={form.occupation} onChange={(e) => set("occupation", e.target.value)} />
          </Section>

          <Section title="Định danh (CCCD/CMND/Hộ chiếu) — không bắt buộc">
            <Input label="Số CCCD/CMND/Hộ chiếu" maxLength={12} placeholder="CMND 9 số · CCCD 12 số · Hộ chiếu" value={form.cccd_number} onChange={(e) => set("cccd_number", e.target.value.replace(/[^a-zA-Z0-9]/g, "").toUpperCase().slice(0, 12))} />
            <Input label="Ngày cấp" placeholder="dd/mm/yyyy" inputMode="numeric" value={form.cccd_issue_date} onChange={(e) => set("cccd_issue_date", e.target.value)} />
            <Input label="Nơi cấp" value={form.cccd_issue_place} onChange={(e) => set("cccd_issue_place", e.target.value)} />
          </Section>

          <Section title="Địa chỉ">
            <Input label="Địa chỉ thường trú" value={form.permanent_address} onChange={(e) => set("permanent_address", e.target.value)} />
            <Input label="Địa chỉ liên hệ" value={form.contact_address} onChange={(e) => set("contact_address", e.target.value)} />
          </Section>

          <Section title="Thuế & tài khoản nhận cổ tức">
            <Input label="Mã số thuế cá nhân (không bắt buộc)" value={form.tax_code} onChange={(e) => set("tax_code", e.target.value)} />
            <Input label="Ngân hàng" value={form.bank_name} onChange={(e) => set("bank_name", e.target.value)} />
            <Input label="Số tài khoản" value={form.bank_account_number} onChange={(e) => set("bank_account_number", e.target.value)} />
            <Input label="Chủ tài khoản" value={form.bank_account_name} onChange={(e) => set("bank_account_name", e.target.value)} />
          </Section>

          <div className="flex items-center gap-3">
            <Button type="submit" disabled={save.isPending}>
              {save.isPending ? "Đang lưu..." : "Lưu hồ sơ"}
            </Button>
            {saved && <span className="text-sm text-gold-300">Đã lưu ✓</span>}
          </div>
        </form>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card className="space-y-4">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
        {title}
      </h2>
      <div className="grid gap-4 sm:grid-cols-2">{children}</div>
    </Card>
  );
}

export default function ProfilePage() {
  return (
    <Guard requireRole="investor">
      <ProfileInner />
    </Guard>
  );
}
