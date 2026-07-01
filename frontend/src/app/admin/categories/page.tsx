"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { Badge, Button, Card, ErrorText, Input, Spinner, Textarea } from "@/components/ui";
import type { ProductCategory } from "@/lib/types";

type Form = { name: string; slug: string; description: string; sort_order: number; active: boolean };
const EMPTY: Form = { name: "", slug: "", description: "", sort_order: 0, active: true };

function CategoriesInner() {
  const qc = useQueryClient();
  const [form, setForm] = useState<Form>(EMPTY);
  const [editId, setEditId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const { data, isLoading } = useQuery({ queryKey: ["admin-categories"], queryFn: adminApi.categories });

  const reset = () => {
    setForm(EMPTY);
    setEditId(null);
    setError(null);
  };
  const done = () => {
    qc.invalidateQueries({ queryKey: ["admin-categories"] });
    reset();
  };
  const onErr = (e: unknown) => setError((e as ApiException).message);

  const create = useMutation({ mutationFn: () => adminApi.createCategory(form), onSuccess: done, onError: onErr });
  const update = useMutation({ mutationFn: () => adminApi.updateCategory(editId!, form), onSuccess: done, onError: onErr });
  const del = useMutation({
    mutationFn: (id: string) => adminApi.deleteCategory(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-categories"] }),
    onError: onErr,
  });

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) {
      setError("Nhập tên danh mục.");
      return;
    }
    (editId ? update : create).mutate();
  }

  function edit(c: ProductCategory) {
    setEditId(c.id);
    setForm({ name: c.name, slug: c.slug, description: c.description, sort_order: c.sort_order, active: c.active });
    setError(null);
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="font-serif text-2xl text-cream">Danh mục sản phẩm</h1>
        <p className="mt-1 text-sm text-cream/55">Nhóm sản phẩm theo danh mục để quản lý bán hàng.</p>
      </div>

      {error && <ErrorText>{error}</ErrorText>}

      <Card className="max-w-2xl space-y-4">
        <h2 className="text-sm font-semibold text-cream/80">{editId ? "Sửa danh mục" : "Thêm danh mục"}</h2>
        <form onSubmit={submit} className="grid gap-4 sm:grid-cols-2">
          <Input label="Tên danh mục" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <Input label="Slug (tự tạo nếu trống)" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} />
          <Input
            label="Thứ tự hiển thị"
            type="number"
            value={form.sort_order}
            onChange={(e) => setForm({ ...form, sort_order: parseInt(e.target.value || "0", 10) })}
          />
          <label className="flex items-end gap-2 pb-2">
            <input type="checkbox" checked={form.active} onChange={(e) => setForm({ ...form, active: e.target.checked })} />
            <span className="text-sm text-cream/80">Đang hoạt động</span>
          </label>
          <div className="sm:col-span-2">
            <Textarea label="Mô tả" rows={2} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </div>
          <div className="flex gap-2 sm:col-span-2">
            <Button type="submit" disabled={create.isPending || update.isPending}>
              {editId ? "Lưu thay đổi" : "Thêm danh mục"}
            </Button>
            {editId && (
              <Button type="button" variant="ghost" onClick={reset}>
                Hủy
              </Button>
            )}
          </div>
        </form>
      </Card>

      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">Danh sách ({data?.length ?? 0})</h2>
        {isLoading ? (
          <Spinner />
        ) : (
          <Card className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="text-left">Tên</th>
                  <th className="text-left">Slug</th>
                  <th className="text-right">Thứ tự</th>
                  <th className="text-left">Trạng thái</th>
                  <th className="text-right">Thao tác</th>
                </tr>
              </thead>
              <tbody>
                {data?.map((c) => (
                  <tr key={c.id}>
                    <td className="font-medium text-cream">{c.name}</td>
                    <td className="text-cream/60">{c.slug}</td>
                    <td className="text-right">{c.sort_order}</td>
                    <td>
                      <Badge tone={c.active ? "green" : "slate"}>{c.active ? "Hoạt động" : "Ẩn"}</Badge>
                    </td>
                    <td className="text-right">
                      <button onClick={() => edit(c)} className="mr-3 text-gold-300 hover:underline">
                        Sửa
                      </button>
                      <button
                        onClick={() => window.confirm(`Xoá danh mục "${c.name}"?`) && del.mutate(c.id)}
                        className="text-red-300 hover:underline"
                      >
                        Xoá
                      </button>
                    </td>
                  </tr>
                ))}
                {data?.length === 0 && (
                  <tr>
                    <td colSpan={5} className="py-6 text-center text-cream/45">
                      Chưa có danh mục nào.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </Card>
        )}
      </section>
    </div>
  );
}

export default function CategoriesPage() {
  return (
    <Guard requireRole="admin">
      <CategoriesInner />
    </Guard>
  );
}
