"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatVnd } from "@/lib/format";
import { Badge, Button, Card, ErrorText, Input, Select, Spinner, Textarea } from "@/components/ui";
import type { Product } from "@/lib/types";

type Form = {
  category_id: string;
  sku: string;
  name: string;
  badge: string;
  price_vnd: number;
  cost_vnd: number;
  image_url: string;
  summary: string;
  description: string;
  spec_warranty: string;
  spec_trace: string;
  spec_delivery: string;
  spec_return: string;
  active: boolean;
};

const EMPTY: Form = {
  category_id: "",
  sku: "",
  name: "",
  badge: "",
  price_vnd: 0,
  cost_vnd: 0,
  image_url: "",
  summary: "",
  description: "",
  spec_warranty: "Chính hãng 100%",
  spec_trace: "Theo từng lô",
  spec_delivery: "Hub theo khu vực",
  spec_return: "Trong 7 ngày",
  active: true,
};

function ProductsInner() {
  const qc = useQueryClient();
  const [form, setForm] = useState<Form>(EMPTY);
  const [editId, setEditId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [uploading, setUploading] = useState(false);

  async function onImageFile(file: File | undefined) {
    if (!file) return;
    setUploading(true);
    setError(null);
    try {
      const res = await adminApi.uploadProductImage(file);
      setForm((f) => ({ ...f, image_url: res.url }));
    } catch (e) {
      setError((e as ApiException).message);
    } finally {
      setUploading(false);
    }
  }

  const { data: products, isLoading } = useQuery({ queryKey: ["admin-products"], queryFn: adminApi.products });
  const { data: categories } = useQuery({ queryKey: ["admin-categories"], queryFn: adminApi.categories });

  const catName = (id: string | null) => categories?.find((c) => c.id === id)?.name ?? "—";

  const reset = () => {
    setForm(EMPTY);
    setEditId(null);
    setError(null);
    setShowForm(false);
  };
  const done = () => {
    qc.invalidateQueries({ queryKey: ["admin-products"] });
    reset();
  };
  const onErr = (e: unknown) => setError((e as ApiException).message);

  const create = useMutation({ mutationFn: () => adminApi.createProduct({ ...form, category_id: form.category_id || null }), onSuccess: done, onError: onErr });
  const update = useMutation({ mutationFn: () => adminApi.updateProduct(editId!, { ...form, category_id: form.category_id || null }), onSuccess: done, onError: onErr });
  const del = useMutation({
    mutationFn: (id: string) => adminApi.deleteProduct(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-products"] }),
    onError: onErr,
  });

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) {
      setError("Nhập tên sản phẩm.");
      return;
    }
    if (form.price_vnd < 0 || form.cost_vnd < 0) {
      setError("Giá không hợp lệ.");
      return;
    }
    (editId ? update : create).mutate();
  }

  function edit(p: Product) {
    setEditId(p.id);
    setForm({
      category_id: p.category_id ?? "",
      sku: p.sku,
      name: p.name,
      badge: p.badge,
      price_vnd: p.price_vnd,
      cost_vnd: p.cost_vnd,
      image_url: p.image_url,
      summary: p.summary,
      description: p.description,
      spec_warranty: p.spec_warranty,
      spec_trace: p.spec_trace,
      spec_delivery: p.spec_delivery,
      spec_return: p.spec_return,
      active: p.active,
    });
    setError(null);
    setShowForm(true);
  }

  // % giá vốn trên giá bán — giúp đối chiếu với mốc 30% của chính sách bán hàng.
  const costPct = form.price_vnd > 0 ? Math.round((form.cost_vnd / form.price_vnd) * 100) : 0;

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="font-serif text-2xl text-cream">Sản phẩm</h1>
          <p className="mt-1 text-sm text-cream/55">Quản lý sản phẩm bán hàng (giá bán, giá vốn, mô tả, thông số).</p>
        </div>
        {!showForm && <Button onClick={() => setShowForm(true)}>+ Thêm sản phẩm</Button>}
      </div>

      {error && <ErrorText>{error}</ErrorText>}

      {showForm && (
        <Card className="space-y-4">
          <h2 className="text-sm font-semibold text-cream/80">{editId ? "Sửa sản phẩm" : "Thêm sản phẩm"}</h2>
          <form onSubmit={submit} className="grid gap-4 sm:grid-cols-2">
            <Input label="Tên sản phẩm" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            <Select label="Danh mục" value={form.category_id} onChange={(e) => setForm({ ...form, category_id: e.target.value })}>
              <option value="">— Không thuộc danh mục —</option>
              {categories?.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </Select>
            <Input label="Mã SKU (tự tạo nếu trống)" value={form.sku} onChange={(e) => setForm({ ...form, sku: e.target.value })} />
            <Input label='Nhãn nhỏ (vd "299K")' value={form.badge} onChange={(e) => setForm({ ...form, badge: e.target.value })} />
            <Input
              label="Giá bán (VNĐ)"
              type="number"
              value={form.price_vnd}
              onChange={(e) => setForm({ ...form, price_vnd: parseInt(e.target.value || "0", 10) })}
            />
            <div>
              <Input
                label="Giá vốn (VNĐ)"
                type="number"
                value={form.cost_vnd}
                onChange={(e) => setForm({ ...form, cost_vnd: parseInt(e.target.value || "0", 10) })}
              />
              <p className="mt-1 text-xs text-cream/45">
                ≈ {costPct}% giá bán {costPct > 30 && <span className="text-gold-300">(chính sách gợi ý giá vốn ~30%)</span>}
              </p>
            </div>
            <div>
              <span className="mb-1.5 block text-sm font-medium text-cream/80">Ảnh sản phẩm</span>
              <div className="flex items-center gap-3">
                {form.image_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={form.image_url} alt="preview" className="h-16 w-16 rounded-lg object-cover" />
                ) : (
                  <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-white/5 text-xs text-cream/40">Ảnh</div>
                )}
                <div className="flex-1 space-y-1.5">
                  <input
                    type="file"
                    accept="image/*"
                    onChange={(e) => onImageFile(e.target.files?.[0])}
                    className="block w-full text-xs text-cream/70 file:mr-3 file:rounded-full file:border-0 file:bg-gold-500/20 file:px-3 file:py-1.5 file:text-sm file:text-gold-200 hover:file:bg-gold-500/30"
                  />
                  {uploading && <p className="text-xs text-gold-300">Đang tải ảnh...</p>}
                  {form.image_url && <button type="button" onClick={() => setForm({ ...form, image_url: "" })} className="text-xs text-red-300 hover:underline">Gỡ ảnh</button>}
                </div>
              </div>
              <div className="mt-2 flex items-center gap-2">
                <span className="text-xs text-cream/40">hoặc</span>
                <input
                  type="url"
                  placeholder="Dán link ảnh (https://...)"
                  value={form.image_url}
                  onChange={(e) => setForm({ ...form, image_url: e.target.value })}
                  className="w-full rounded-lg border border-white/15 bg-white/5 px-3 py-1.5 text-xs text-cream placeholder:text-cream/35 focus:border-gold-500 focus:outline-none"
                />
              </div>
            </div>
            <label className="flex items-end gap-2 pb-2">
              <input type="checkbox" checked={form.active} onChange={(e) => setForm({ ...form, active: e.target.checked })} />
              <span className="text-sm text-cream/80">Đang bán</span>
            </label>
            <div className="sm:col-span-2">
              <Textarea label="Mô tả ngắn" rows={2} value={form.summary} onChange={(e) => setForm({ ...form, summary: e.target.value })} />
            </div>
            <div className="sm:col-span-2">
              <Textarea label="Mô tả chi tiết" rows={4} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
            <Input label="Cam kết" value={form.spec_warranty} onChange={(e) => setForm({ ...form, spec_warranty: e.target.value })} />
            <Input label="Truy xuất" value={form.spec_trace} onChange={(e) => setForm({ ...form, spec_trace: e.target.value })} />
            <Input label="Giao hàng" value={form.spec_delivery} onChange={(e) => setForm({ ...form, spec_delivery: e.target.value })} />
            <Input label="Đổi trả" value={form.spec_return} onChange={(e) => setForm({ ...form, spec_return: e.target.value })} />
            <div className="flex gap-2 sm:col-span-2">
              <Button type="submit" disabled={create.isPending || update.isPending}>
                {editId ? "Lưu thay đổi" : "Thêm sản phẩm"}
              </Button>
              <Button type="button" variant="ghost" onClick={reset}>
                Hủy
              </Button>
            </div>
          </form>
        </Card>
      )}

      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">Danh sách ({products?.length ?? 0})</h2>
        {isLoading ? (
          <Spinner />
        ) : (
          <Card className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="text-left">Sản phẩm</th>
                  <th className="text-left">Danh mục</th>
                  <th className="text-right">Giá bán</th>
                  <th className="text-right">Giá vốn</th>
                  <th className="text-left">Trạng thái</th>
                  <th className="text-right">Thao tác</th>
                </tr>
              </thead>
              <tbody>
                {products?.map((p) => (
                  <tr key={p.id}>
                    <td>
                      <div className="flex items-center gap-3">
                        {p.image_url ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img src={p.image_url} alt={p.name} className="h-10 w-10 rounded-lg object-cover" />
                        ) : (
                          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-white/5 text-xs text-cream/40">—</div>
                        )}
                        <div>
                          <p className="font-medium text-cream">{p.name}</p>
                          <p className="text-xs text-cream/45">{p.sku}</p>
                        </div>
                      </div>
                    </td>
                    <td className="text-cream/70">{catName(p.category_id)}</td>
                    <td className="text-right font-mono text-cream">{formatVnd(p.price_vnd)}</td>
                    <td className="text-right font-mono text-cream/60">{formatVnd(p.cost_vnd)}</td>
                    <td>
                      <Badge tone={p.active ? "green" : "slate"}>{p.active ? "Đang bán" : "Ẩn"}</Badge>
                    </td>
                    <td className="text-right">
                      <button onClick={() => edit(p)} className="mr-3 text-gold-300 hover:underline">
                        Sửa
                      </button>
                      <button
                        onClick={() => window.confirm(`Xoá sản phẩm "${p.name}"?`) && del.mutate(p.id)}
                        className="text-red-300 hover:underline"
                      >
                        Xoá
                      </button>
                    </td>
                  </tr>
                ))}
                {products?.length === 0 && (
                  <tr>
                    <td colSpan={6} className="py-6 text-center text-cream/45">
                      Chưa có sản phẩm nào.
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

export default function ProductsPage() {
  return (
    <Guard requireRole="admin">
      <ProductsInner />
    </Guard>
  );
}
