"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { Badge, Button, Card, ErrorText, Input, Spinner, Textarea } from "@/components/ui";

type Policy = { slug: string; title: string; summary: string; body: string; sort_order: number; active: boolean };
const EMPTY: Policy = { slug: "", title: "", summary: "", body: "", sort_order: 0, active: true };

function slugify(s: string): string {
  return s
    .normalize("NFD").replace(/[̀-ͯ]/g, "")
    .replace(/đ/g, "d").replace(/Đ/g, "D")
    .toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");
}

function PolicyInner() {
  const qc = useQueryClient();
  const [form, setForm] = useState<Policy>(EMPTY);
  const [editingSlug, setEditingSlug] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const { data, isLoading } = useQuery({ queryKey: ["admin-policies"], queryFn: adminApi.policies });

  const save = useMutation({
    mutationFn: () => adminApi.upsertPolicy({ ...form, slug: form.slug || slugify(form.title) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-policies"] });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      reset();
    },
    onError: (e) => setError((e as ApiException).message),
  });
  const del = useMutation({
    mutationFn: (slug: string) => adminApi.deletePolicy(slug),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-policies"] }),
    onError: (e) => setError((e as ApiException).message),
  });

  const set = (k: keyof Policy, v: string | number | boolean) => setForm((f) => ({ ...f, [k]: v }));
  function reset() { setForm(EMPTY); setEditingSlug(null); setError(null); }
  function edit(p: Policy) { setForm(p); setEditingSlug(p.slug); setError(null); }
  function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!form.title.trim()) { setError("Nhập tiêu đề."); return; }
    save.mutate();
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-serif text-2xl text-cream">Chính sách website</h1>
        <p className="mt-1 text-sm text-cream/55">Nội dung tại duoclieuhk.vn/chinh-sach. Có thể thêm chính sách riêng (vd hoa hồng CTV). Đồng bộ ~1 phút sau khi lưu.</p>
      </div>
      <ErrorText>{error}</ErrorText>

      <div className="grid gap-6 lg:grid-cols-[1fr_1.2fr] lg:items-start">
        <Card className="space-y-0 p-0">
          {isLoading ? <div className="p-6"><Spinner /></div> : (
            <ul className="divide-y divide-white/5">
              {data?.map((p) => (
                <li key={p.slug} className="flex items-center justify-between gap-3 p-4">
                  <div>
                    <p className="font-medium text-cream">{p.title} {!p.active && <Badge tone="slate">Ẩn</Badge>}</p>
                    <p className="text-xs text-cream/45">/{p.slug}</p>
                  </div>
                  <div className="flex shrink-0 gap-3 text-sm">
                    <button onClick={() => edit(p)} className="text-gold-300 hover:underline">Sửa</button>
                    <button onClick={() => window.confirm(`Xoá "${p.title}"?`) && del.mutate(p.slug)} className="text-red-300 hover:underline">Xoá</button>
                  </div>
                </li>
              ))}
              {data?.length === 0 && <li className="p-6 text-center text-cream/45">Chưa có chính sách.</li>}
            </ul>
          )}
          <div className="p-4"><Button variant="ghost" onClick={reset}>+ Thêm chính sách mới</Button></div>
        </Card>

        <Card className="space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-gold-300">
            {editingSlug ? `Sửa: ${editingSlug}` : "Thêm chính sách"}
          </h2>
          <form onSubmit={submit} className="space-y-4">
            <Input label="Tiêu đề" value={form.title} onChange={(e) => set("title", e.target.value)} />
            <Input
              label="Đường dẫn (slug) — để trống sẽ tự tạo"
              value={form.slug}
              placeholder={form.title ? slugify(form.title) : "vd: hoa-hong-ctv"}
              onChange={(e) => set("slug", e.target.value)}
              disabled={!!editingSlug}
            />
            <Input label="Mô tả ngắn" value={form.summary} onChange={(e) => set("summary", e.target.value)} />
            <Textarea label="Nội dung (mỗi đoạn cách nhau 1 dòng trống)" rows={12} value={form.body} onChange={(e) => set("body", e.target.value)} />
            <div className="flex flex-wrap items-end gap-4">
              <Input label="Thứ tự" type="number" value={form.sort_order} onChange={(e) => set("sort_order", parseInt(e.target.value || "0", 10))} />
              <label className="flex items-center gap-2 pb-2 text-sm text-cream/80">
                <input type="checkbox" checked={form.active} onChange={(e) => set("active", e.target.checked)} /> Hiển thị
              </label>
            </div>
            <div className="flex items-center gap-3">
              <Button type="submit" disabled={save.isPending}>{save.isPending ? "Đang lưu..." : "Lưu"}</Button>
              {editingSlug && <Button type="button" variant="ghost" onClick={reset}>Huỷ</Button>}
              {saved && <span className="text-sm text-gold-300">Đã lưu ✓</span>}
            </div>
          </form>
        </Card>
      </div>
    </div>
  );
}

export default function AdminPolicyPage() {
  return <Guard requireRole="admin"><PolicyInner /></Guard>;
}
