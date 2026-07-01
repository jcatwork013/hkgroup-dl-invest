"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi, salesApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatVnd } from "@/lib/format";
import { Badge, Button, Card, ErrorText, Input, Select, Spinner, Textarea, statusLabel, statusTone } from "@/components/ui";
import type { OrderDetail, Product, SalesOrderRow } from "@/lib/types";

type CartLine = { product: Product; qty: number };

// OrdersPanel — tạo đơn + quản lý đơn. admin=true: chọn người bán + xem mọi đơn; saler: tự là người bán, chỉ đơn của mình.
export default function OrdersPanel({ admin }: { admin: boolean }) {
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [detailId, setDetailId] = useState<string | null>(null);

  // form state
  const [customerName, setCustomerName] = useState("");
  const [customerPhone, setCustomerPhone] = useState("");
  const [sellerId, setSellerId] = useState("");
  const [affiliateId, setAffiliateId] = useState("");
  const [note, setNote] = useState("");
  const [cart, setCart] = useState<CartLine[]>([]);
  const [pickProduct, setPickProduct] = useState("");
  const [pickQty, setPickQty] = useState(1);

  const { data: products } = useQuery({ queryKey: ["sales-products"], queryFn: salesApi.products });
  const { data: salers } = useQuery({ queryKey: ["sales-salers"], queryFn: salesApi.salers });
  const ordersKey = admin ? ["admin-orders"] : ["my-orders"];
  const { data: orders, isLoading } = useQuery({ queryKey: ordersKey, queryFn: admin ? adminApi.allOrders : salesApi.myOrders });

  const subtotal = useMemo(() => cart.reduce((s, l) => s + l.product.price_vnd * l.qty, 0), [cart]);

  const resetForm = () => {
    setCustomerName(""); setCustomerPhone(""); setSellerId(""); setAffiliateId(""); setNote("");
    setCart([]); setPickProduct(""); setPickQty(1); setShowForm(false); setError(null);
  };
  const onErr = (e: unknown) => setError((e as ApiException).message);
  const refresh = () => qc.invalidateQueries({ queryKey: ordersKey });

  const create = useMutation({
    mutationFn: () =>
      salesApi.createOrder({
        customer_name: customerName, customer_phone: customerPhone,
        seller_id: admin ? sellerId : undefined,
        affiliate_id: affiliateId || undefined,
        note, items: cart.map((l) => ({ product_id: l.product.id, qty: l.qty })),
      }),
    onSuccess: () => { refresh(); resetForm(); },
    onError: onErr,
  });
  const pay = useMutation({ mutationFn: (id: string) => salesApi.payOrder(id), onSuccess: () => { refresh(); if (detailId) qc.invalidateQueries({ queryKey: ["order-detail", detailId] }); }, onError: onErr });
  const cancel = useMutation({ mutationFn: (id: string) => salesApi.cancelOrder(id), onSuccess: refresh, onError: onErr });

  function addToCart() {
    const p = products?.find((x) => x.id === pickProduct);
    if (!p || pickQty <= 0) return;
    setCart((c) => {
      const ex = c.find((l) => l.product.id === p.id);
      if (ex) return c.map((l) => (l.product.id === p.id ? { ...l, qty: l.qty + pickQty } : l));
      return [...c, { product: p, qty: pickQty }];
    });
    setPickProduct(""); setPickQty(1);
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (admin && !sellerId) { setError("Chọn người bán (saler)."); return; }
    if (cart.length === 0) { setError("Thêm ít nhất 1 sản phẩm."); return; }
    create.mutate();
  }

  return (
    <div className="space-y-6">
      {error && <ErrorText>{error}</ErrorText>}

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-cream">Đơn hàng ({orders?.length ?? 0})</h2>
        {!showForm && <Button onClick={() => setShowForm(true)}>+ Tạo đơn</Button>}
      </div>

      {showForm && (
        <Card className="space-y-4">
          <form onSubmit={submit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <Input label="Tên khách hàng" value={customerName} onChange={(e) => setCustomerName(e.target.value)} />
              <Input label="SĐT khách" value={customerPhone} onChange={(e) => setCustomerPhone(e.target.value)} />
              {admin && (
                <Select label="Người bán (saler) *" value={sellerId} onChange={(e) => setSellerId(e.target.value)}>
                  <option value="">— Chọn người bán —</option>
                  {salers?.map((s) => <option key={s.id} value={s.id}>{s.full_name} ({s.email})</option>)}
                </Select>
              )}
              <Select label="Affiliate (giới thiệu khách, tuỳ chọn)" value={affiliateId} onChange={(e) => setAffiliateId(e.target.value)}>
                <option value="">— Không có —</option>
                {salers?.filter((s) => s.id !== sellerId).map((s) => <option key={s.id} value={s.id}>{s.full_name}</option>)}
              </Select>
            </div>

            {/* Thêm sản phẩm vào đơn */}
            <div className="rounded-xl border border-white/10 p-3">
              <p className="mb-2 text-sm font-medium text-cream/80">Sản phẩm</p>
              <div className="flex flex-wrap items-end gap-2">
                <Select label="Chọn sản phẩm" className="min-w-[12rem]" value={pickProduct} onChange={(e) => setPickProduct(e.target.value)}>
                  <option value="">— Chọn —</option>
                  {products?.filter((p) => p.active).map((p) => <option key={p.id} value={p.id}>{p.name} — {formatVnd(p.price_vnd)}</option>)}
                </Select>
                <Input label="SL" type="number" className="w-20" value={pickQty} onChange={(e) => setPickQty(parseInt(e.target.value || "1", 10))} />
                <Button type="button" variant="secondary" onClick={addToCart}>Thêm</Button>
              </div>
              {cart.length > 0 && (
                <table className="mt-3 w-full text-sm">
                  <tbody>
                    {cart.map((l) => (
                      <tr key={l.product.id}>
                        <td className="text-cream">{l.product.name}</td>
                        <td className="text-right text-cream/60">{l.qty} × {formatVnd(l.product.price_vnd)}</td>
                        <td className="text-right font-mono text-cream">{formatVnd(l.product.price_vnd * l.qty)}</td>
                        <td className="text-right">
                          <button type="button" onClick={() => setCart((c) => c.filter((x) => x.product.id !== l.product.id))} className="text-red-300 hover:underline">x</button>
                        </td>
                      </tr>
                    ))}
                    <tr className="border-t border-white/10">
                      <td className="pt-2 font-medium text-cream" colSpan={2}>Tổng đơn</td>
                      <td className="pt-2 text-right font-mono font-semibold text-gold-300">{formatVnd(subtotal)}</td>
                      <td />
                    </tr>
                  </tbody>
                </table>
              )}
            </div>

            <Textarea label="Ghi chú" rows={2} value={note} onChange={(e) => setNote(e.target.value)} />
            <div className="flex gap-2">
              <Button type="submit" disabled={create.isPending}>Tạo đơn</Button>
              <Button type="button" variant="ghost" onClick={resetForm}>Hủy</Button>
            </div>
          </form>
        </Card>
      )}

      {isLoading ? (
        <Spinner />
      ) : (
        <Card className="overflow-x-auto p-0">
          <table className="w-full text-sm">
            <thead>
              <tr>
                <th className="text-left">Mã đơn</th>
                {admin && <th className="text-left">Người bán</th>}
                <th className="text-left">Khách</th>
                <th className="text-right">Tổng</th>
                <th className="text-left">Trạng thái</th>
                <th className="text-right">Thao tác</th>
              </tr>
            </thead>
            <tbody>
              {orders?.map((o: SalesOrderRow) => (
                <tr key={o.id}>
                  <td className="font-mono text-cream">{o.code}</td>
                  {admin && <td className="text-cream/70">{o.seller_name}</td>}
                  <td className="text-cream/70">{o.customer_name || "—"}</td>
                  <td className="text-right font-mono text-cream">{formatVnd(o.subtotal_vnd)}</td>
                  <td><Badge tone={statusTone(o.status)}>{statusLabel(o.status)}</Badge></td>
                  <td className="text-right whitespace-nowrap">
                    <button onClick={() => setDetailId(detailId === o.id ? null : o.id)} className="mr-3 text-gold-300 hover:underline">Chi tiết</button>
                    {o.status === "pending" && (
                      <>
                        <button onClick={() => pay.mutate(o.id)} className="mr-3 text-brand-300 hover:underline">Đã thanh toán</button>
                        <button onClick={() => window.confirm(`Huỷ đơn ${o.code}?`) && cancel.mutate(o.id)} className="text-red-300 hover:underline">Huỷ</button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
              {orders?.length === 0 && (
                <tr><td colSpan={admin ? 6 : 5} className="py-6 text-center text-cream/45">Chưa có đơn hàng nào.</td></tr>
              )}
            </tbody>
          </table>
        </Card>
      )}

      {detailId && <OrderDetailCard id={detailId} onClose={() => setDetailId(null)} />}
    </div>
  );
}

function Row({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className="flex items-center justify-between border-b border-white/5 py-1.5">
      <span className="text-sm text-cream/60">{label}</span>
      <span className={`font-mono text-sm ${strong ? "font-semibold text-gold-300" : "text-cream"}`}>{value}</span>
    </div>
  );
}

function OrderDetailCard({ id, onClose }: { id: string; onClose: () => void }) {
  const { data, isLoading } = useQuery<OrderDetail>({ queryKey: ["order-detail", id], queryFn: () => salesApi.orderDetail(id) });
  if (isLoading || !data) return <Card><Spinner /></Card>;
  const d = data.distribution;
  return (
    <Card className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-serif text-lg text-cream">Đơn {data.order.code}</h3>
        <button onClick={onClose} className="text-cream/50 hover:text-cream">✕</button>
      </div>
      <div className="grid gap-6 sm:grid-cols-2">
        <div>
          <p className="mb-2 text-xs uppercase tracking-wide text-cream/45">Sản phẩm</p>
          <table className="w-full text-sm">
            <tbody>
              {data.items.map((it) => (
                <tr key={it.id}>
                  <td className="text-cream">{it.name}</td>
                  <td className="text-right text-cream/60">{it.qty} × {formatVnd(it.unit_price_vnd)}</td>
                  <td className="text-right font-mono text-cream">{formatVnd(it.line_total_vnd)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="mt-3 text-sm text-cream/60">Người bán: <span className="text-cream">{data.seller_name}</span></p>
          {data.affiliate_name && <p className="text-sm text-cream/60">Affiliate: <span className="text-cream">{data.affiliate_name}</span></p>}
        </div>
        <div>
          <p className="mb-2 text-xs uppercase tracking-wide text-cream/45">Chia dòng tiền {d ? "" : "(chưa thanh toán)"}</p>
          {d ? (
            <>
              <Row label="Người bán 25% (chia ngoài)" value={formatVnd(d.seller_vnd)} />
              <Row label="Affiliate 10%" value={formatVnd(d.affiliate_vnd)} />
              <Row label="Pool người mua 5% (≥1tr)" value={formatVnd(d.equal_share_vnd)} />
              <Row label="Pool Cổ Đông 15% (9%+6%)" value={formatVnd(d.pool_vnd)} strong />
              <Row label="Giá vốn 30%" value={formatVnd(d.cost_vnd)} />
              <Row label="Vận hành 15%" value={formatVnd(d.operations_vnd)} />
              <Row label="Tổng đơn" value={formatVnd(d.total_vnd)} strong />
            </>
          ) : (
            <p className="text-sm text-cream/50">Đơn sẽ được chia 6 khoản khi xác nhận thanh toán.</p>
          )}
        </div>
      </div>
    </Card>
  );
}
