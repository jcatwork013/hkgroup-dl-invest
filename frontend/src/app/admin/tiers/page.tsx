"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Guard from "@/components/Guard";
import { adminApi, publicApi } from "@/lib/endpoints";
import { ApiException } from "@/lib/api";
import { formatPct, formatVnd } from "@/lib/format";
import { Badge, Button, Card, ErrorText, Input, Spinner } from "@/components/ui";
import { Modal } from "@/components/Modal";
import type { OfferingTier } from "@/lib/types";

type Form = {
  name: string;
  amount_vnd: string;
  sort_order: string;
  active: boolean;
};

const EMPTY: Form = { name: "", amount_vnd: "", sort_order: "0", active: true };

function num(s: string): number {
  return parseFloat(s.replace(/[^\d.]/g, "")) || 0;
}

function TiersInner() {
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [form, setForm] = useState<Form>(EMPTY);

  // Auto-allocate modal state.
  const [allocOpen, setAllocOpen] = useState(false);
  const [allocCount, setAllocCount] = useState("6");
  const [allocStart, setAllocStart] = useState("5000000");
  const [allocReplace, setAllocReplace] = useState(true);
  const [allocBusy, setAllocBusy] = useState(false);

  // New funding round modal state.
  const [roundOpen, setRoundOpen] = useState(false);
  const [roundName, setRoundName] = useState("");
  const [roundValuation, setRoundValuation] = useState("");
  const [roundPrice, setRoundPrice] = useState("10000");
  const [roundPct, setRoundPct] = useState("49");

  const tiers = useQuery({ queryKey: ["admin-tiers"], queryFn: adminApi.tiers });
  // Offering drives ALL share math. price/share = valuation / total_shares (vd 10.000đ/cp);
  // % sở hữu = số tiền / valuation × 100. Pool bán = shares_for_sale × price (49% = 4.851 tỷ).
  const offering = useQuery({
    queryKey: ["public-offering"],
    queryFn: publicApi.offering,
  });
  const valuation = offering.data?.offering.valuation_vnd ?? 0;
  const totalShares = offering.data?.offering.total_shares ?? 0;
  const sharesForSale = offering.data?.offering.shares_for_sale ?? 0;
  const pricePerShare = totalShares > 0 ? valuation / totalShares : 0;
  const poolMaxAmount = Math.round(sharesForSale * pricePerShare); // = 49% pool (vd 4.851 tỷ)

  // Derived shares + % for the amount currently typed in the create/edit form.
  const formAmount = num(form.amount_vnd);
  const formShares = pricePerShare > 0 ? Math.round(formAmount / pricePerShare) : 0;
  const formPct = valuation > 0 ? (formAmount / valuation) * 100 : 0;

  function openCreate() {
    setEditId(null);
    setForm(EMPTY);
    setError(null);
    setModalOpen(true);
  }

  function openEdit(t: OfferingTier) {
    setEditId(t.id);
    setForm({
      name: t.name,
      amount_vnd: String(t.amount_vnd),
      sort_order: String(t.sort_order ?? 0),
      active: t.active ?? true,
    });
    setError(null);
    setModalOpen(true);
  }

  const save = useMutation({
    mutationFn: () => {
      // Server tự tính shares & % theo pool — chỉ gửi tên + số tiền + thứ tự + trạng thái.
      const body: Partial<OfferingTier> = {
        name: form.name,
        amount_vnd: num(form.amount_vnd),
        sort_order: num(form.sort_order),
        active: form.active,
      };
      return editId ? adminApi.updateTier(editId, body) : adminApi.createTier(body);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-tiers"] });
      setModalOpen(false);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  const toggle = useMutation({
    mutationFn: (v: { id: string; active: boolean }) =>
      adminApi.setTierActive(v.id, v.active),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-tiers"] }),
    onError: (e) => setError((e as ApiException).message),
  });

  const remove = useMutation({
    mutationFn: (id: string) => adminApi.deleteTier(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-tiers"] }),
    onError: (e) => setError((e as ApiException).message),
  });

  // New round preview math.
  const rValuation = num(roundValuation);
  const rPrice = num(roundPrice);
  const rPct = num(roundPct);
  const rTotalShares = rPrice > 0 ? Math.round(rValuation / rPrice) : 0;
  const rForSale = Math.round((rTotalShares * rPct) / 100);
  const rPoolVnd = rForSale * rPrice;

  const openRound = useMutation({
    mutationFn: () =>
      adminApi.openRound({
        name: roundName.trim(),
        valuation_vnd: rValuation,
        total_shares: rTotalShares,
        shares_for_sale: rForSale,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["public-offering"] });
      qc.invalidateQueries({ queryKey: ["admin-tiers"] });
      setRoundOpen(false);
    },
    onError: (e) => setError((e as ApiException).message),
  });

  function submitRound() {
    setError(null);
    if (!roundName.trim() || rValuation <= 0 || rTotalShares <= 0 || rForSale <= 0 || rForSale > rTotalShares) {
      setError("Nhập đủ tên vòng, định giá, giá/cổ phần và % bán (0 < % ≤ 100).");
      return;
    }
    openRound.mutate();
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!form.name || num(form.amount_vnd) <= 0) {
      setError("Nhập đủ tên và số tiền hợp lệ.");
      return;
    }
    save.mutate();
  }

  function askDelete(t: OfferingTier) {
    if (
      window.confirm(
        `Xoá vĩnh viễn "${t.name}" (${formatVnd(t.amount_vnd)})?\nGói đã có hợp đồng/đầu tư sẽ không xoá được — hãy ẩn.`
      )
    ) {
      setError(null);
      remove.mutate(t.id);
    }
  }

  // ----- Auto-allocate: sinh N gói từ số tiền bắt đầu, gói lớn nhất phủ ĐỦ 49% pool -----
  const preview = useMemo(() => {
    const n = Math.max(1, Math.min(20, Math.floor(num(allocCount))));
    const start = num(allocStart);
    if (pricePerShare <= 0 || start <= 0 || poolMaxAmount <= 0 || start > poolMaxAmount)
      return [] as { name: string; amount: number; shares: number; pct: number }[];
    const amounts: number[] = [];
    if (n === 1) {
      amounts.push(poolMaxAmount);
    } else {
      const ratio = Math.pow(poolMaxAmount / start, 1 / (n - 1)); // cấp số nhân từ start → 49%
      for (let i = 0; i < n; i++) {
        const raw = start * Math.pow(ratio, i);
        // làm tròn về triệu cho số đẹp; gói cuối = đúng pool 49%.
        amounts.push(i === n - 1 ? poolMaxAmount : Math.round(raw / 1_000_000) * 1_000_000);
      }
    }
    return amounts.map((amount, i) => ({
      name: `Gói ${i + 1}`,
      amount,
      shares: Math.round(amount / pricePerShare),
      pct: valuation > 0 ? (amount / valuation) * 100 : 0,
    }));
  }, [allocCount, allocStart, pricePerShare, poolMaxAmount, valuation]);

  async function applyAllocate() {
    if (preview.length === 0) {
      setError("Tham số không hợp lệ (số tiền bắt đầu phải > 0 và ≤ pool 49%).");
      return;
    }
    setError(null);
    setAllocBusy(true);
    try {
      // 1) (tuỳ chọn) Xoá các gói hiện có — gói đã có đầu tư sẽ bị bỏ qua (không xoá được).
      if (allocReplace) {
        for (const t of tiers.data ?? []) {
          try {
            await adminApi.deleteTier(t.id);
          } catch {
            /* gói đang được sử dụng — giữ lại, bỏ qua */
          }
        }
      }
      // 2) Tạo bộ gói mới (server tự tính cổ phần & %).
      for (let i = 0; i < preview.length; i++) {
        const p = preview[i];
        await adminApi.createTier({
          name: p.name,
          amount_vnd: p.amount,
          sort_order: i + 1,
          active: true,
        });
      }
      qc.invalidateQueries({ queryKey: ["admin-tiers"] });
      setAllocOpen(false);
    } catch (e) {
      setError((e as ApiException).message ?? "Tự phân bổ thất bại");
    } finally {
      setAllocBusy(false);
    }
  }

  const allTiers = tiers.data ?? [];
  const soldPct = totalShares > 0 ? (sharesForSale / totalShares) * 100 : 0;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-serif text-2xl text-cream">Gói đầu tư</h1>
          <p className="mt-1 max-w-2xl text-sm text-cream/55">
            Cổ phần &amp; tỷ lệ sở hữu của mỗi gói được{" "}
            <strong className="text-cream/80">tự động tính theo pool</strong> — admin chỉ
            chọn số tiền. Giá {pricePerShare.toLocaleString("vi-VN")} đ/cổ phần · định giá{" "}
            {formatVnd(valuation)} · bán ra {formatPct(soldPct)} ({formatVnd(poolMaxAmount)}).
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" onClick={() => setRoundOpen(true)}>
            🔁 Mở vòng mới
          </Button>
          <Button variant="secondary" onClick={() => setAllocOpen(true)}>
            ⚡ Tự phân bổ
          </Button>
          <Button onClick={openCreate}>＋ Tạo gói mới</Button>
        </div>
      </div>

      <ErrorText>{error}</ErrorText>

      {/* Summary chips */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Stat label="Tổng số gói" value={String(allTiers.length)} />
        <Stat label="Đang mở bán" value={String(allTiers.filter((t) => t.active).length)} accent />
        <Stat label="Pool 49%" value={formatVnd(poolMaxAmount)} />
      </div>

      {/* Tier list */}
      <section>
        <h2 className="mb-3 text-lg font-semibold text-cream">Danh sách gói</h2>
        {tiers.isLoading ? (
          <Spinner />
        ) : allTiers.length === 0 ? (
          <Card className="text-center text-sm text-cream/45">
            Chưa có gói nào. Bấm “Tạo gói mới” hoặc “Tự phân bổ”.
          </Card>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {allTiers.map((t) => (
              <Card key={t.id} className={`flex flex-col gap-4 ${t.active ? "" : "opacity-60"}`}>
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <p className="font-semibold text-cream">{t.name}</p>
                    <p className="mt-1 text-2xl font-bold text-gold-300">{formatVnd(t.amount_vnd)}</p>
                  </div>
                  {t.active ? <Badge tone="green">Đang mở</Badge> : <Badge tone="red">Đã ẩn</Badge>}
                </div>
                <dl className="space-y-1.5 text-sm text-cream/70">
                  <Row label="Số cổ phần" value={t.shares.toLocaleString("vi-VN")} />
                  <Row label="Tỷ lệ sở hữu" value={formatPct(t.ownership_pct)} />
                  <Row label="Thứ tự" value={String(t.sort_order ?? 0)} />
                </dl>
                <div className="mt-auto flex flex-wrap gap-2 border-t border-white/10 pt-3">
                  <Button variant="secondary" className="flex-1" onClick={() => openEdit(t)}>
                    Sửa
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => toggle.mutate({ id: t.id, active: !(t.active ?? true) })}
                    disabled={toggle.isPending}
                  >
                    {t.active ? "Ẩn" : "Mở lại"}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => askDelete(t)}
                    disabled={remove.isPending}
                    className="text-red-300 hover:text-red-200"
                  >
                    Xoá
                  </Button>
                </div>
              </Card>
            ))}
          </div>
        )}
      </section>

      {/* Create / Edit modal */}
      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editId ? "Sửa gói" : "Tạo gói mới"}
        footer={
          <>
            <Button variant="ghost" onClick={() => setModalOpen(false)}>
              Hủy
            </Button>
            <Button onClick={submit} disabled={save.isPending}>
              {save.isPending ? "Đang lưu..." : editId ? "Lưu thay đổi" : "Tạo gói"}
            </Button>
          </>
        }
      >
        <form onSubmit={submit} className="grid gap-3 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <Input
              label="Tên gói"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="VD: Gói 1"
            />
          </div>
          <Input
            label="Giá trị (VNĐ)"
            inputMode="numeric"
            value={form.amount_vnd}
            onChange={(e) => setForm({ ...form, amount_vnd: e.target.value })}
            placeholder="5000000"
          />
          <Input
            label="Thứ tự hiển thị"
            inputMode="numeric"
            value={form.sort_order}
            onChange={(e) => setForm({ ...form, sort_order: e.target.value })}
          />
          <div className="sm:col-span-2 rounded-lg border border-gold-500/20 bg-gold-500/[0.05] p-3 text-sm text-cream/80">
            <p className="font-medium text-gold-300">Tự tính theo pool</p>
            <div className="mt-1.5 flex justify-between">
              <span>Số cổ phần</span>
              <span className="font-medium text-cream">{formShares.toLocaleString("vi-VN")}</span>
            </div>
            <div className="flex justify-between">
              <span>Tỷ lệ sở hữu</span>
              <span className="font-medium text-cream">{formatPct(formPct)}</span>
            </div>
            <p className="mt-1.5 text-xs text-cream/45">
              Giá {pricePerShare.toLocaleString("vi-VN")} đ/cổ phần. Cổ phần &amp; % không nhập tay
              — luôn đúng theo pool.
            </p>
          </div>
          <div className="flex items-center gap-6 pt-1 sm:col-span-2">
            <label className="flex cursor-pointer items-center gap-2 text-sm text-cream/85">
              <input
                type="checkbox"
                checked={form.active}
                onChange={(e) => setForm({ ...form, active: e.target.checked })}
                className="h-5 w-5 cursor-pointer accent-gold-500"
              />
              Đang mở bán
            </label>
          </div>
          <button type="submit" className="hidden" />
        </form>
        <ErrorText>{error}</ErrorText>
      </Modal>

      {/* Auto-allocate modal */}
      <Modal
        open={allocOpen}
        onClose={() => !allocBusy && setAllocOpen(false)}
        title="Tự phân bổ gói theo pool 49%"
        footer={
          <>
            <Button variant="ghost" onClick={() => setAllocOpen(false)} disabled={allocBusy}>
              Hủy
            </Button>
            <Button onClick={applyAllocate} disabled={allocBusy || preview.length === 0}>
              {allocBusy ? "Đang áp dụng..." : "Áp dụng"}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <p className="text-sm text-cream/65">
            Chọn số gói &amp; số tiền bắt đầu. Hệ thống tự sinh các gói tăng dần, gói lớn nhất phủ
            đúng <strong className="text-gold-300">49% pool = {formatVnd(poolMaxAmount)}</strong>. Cổ
            phần &amp; % tự tính — bạn không cần tính toán.
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            <Input
              label="Số gói"
              inputMode="numeric"
              value={allocCount}
              onChange={(e) => setAllocCount(e.target.value)}
              placeholder="6"
            />
            <Input
              label="Số tiền bắt đầu (VNĐ)"
              inputMode="numeric"
              value={allocStart}
              onChange={(e) => setAllocStart(e.target.value)}
              placeholder="5000000"
            />
          </div>
          <label className="flex cursor-pointer items-center gap-2 text-sm text-cream/85">
            <input
              type="checkbox"
              checked={allocReplace}
              onChange={(e) => setAllocReplace(e.target.checked)}
              className="h-5 w-5 cursor-pointer accent-gold-500"
            />
            Xoá các gói hiện có trước (gói đã có đầu tư sẽ được giữ lại)
          </label>

          {preview.length > 0 ? (
            <div className="overflow-hidden rounded-lg border border-white/10">
              <table className="w-full text-sm">
                <thead className="bg-white/5 text-cream/55">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium">Gói</th>
                    <th className="px-3 py-2 text-right font-medium">Số tiền</th>
                    <th className="px-3 py-2 text-right font-medium">Cổ phần</th>
                    <th className="px-3 py-2 text-right font-medium">% sở hữu</th>
                  </tr>
                </thead>
                <tbody>
                  {preview.map((p) => (
                    <tr key={p.name} className="border-t border-white/5 text-cream/80">
                      <td className="px-3 py-2">{p.name}</td>
                      <td className="px-3 py-2 text-right">{formatVnd(p.amount)}</td>
                      <td className="px-3 py-2 text-right">{p.shares.toLocaleString("vi-VN")}</td>
                      <td className="px-3 py-2 text-right">{formatPct(p.pct)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-red-300">
              Số tiền bắt đầu phải &gt; 0 và ≤ {formatVnd(poolMaxAmount)}.
            </p>
          )}
        </div>
        <ErrorText>{error}</ErrorText>
      </Modal>

      {/* New funding round modal */}
      <Modal
        open={roundOpen}
        onClose={() => !openRound.isPending && setRoundOpen(false)}
        title="Mở vòng gọi vốn mới"
        footer={
          <>
            <Button variant="ghost" onClick={() => setRoundOpen(false)} disabled={openRound.isPending}>
              Hủy
            </Button>
            <Button onClick={submitRound} disabled={openRound.isPending}>
              {openRound.isPending ? "Đang mở..." : "Mở vòng"}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <p className="text-sm text-cream/65">
            Vòng đang mở sẽ tự đóng; gói đầu tư thuộc về từng vòng. Sau khi mở, dùng{" "}
            <strong className="text-gold-300">⚡ Tự phân bổ</strong> để tạo gói cho vòng mới. Các vòng
            cũ &amp; cổ phần đã phát hành vẫn được giữ nguyên.
          </p>
          <Input
            label="Tên vòng"
            value={roundName}
            onChange={(e) => setRoundName(e.target.value)}
            placeholder="VD: Vòng 2"
          />
          <div className="grid gap-3 sm:grid-cols-3">
            <Input
              label="Định giá (VNĐ)"
              inputMode="numeric"
              value={roundValuation}
              onChange={(e) => setRoundValuation(e.target.value)}
              placeholder="9900000000"
            />
            <Input
              label="Giá/cổ phần (VNĐ)"
              inputMode="numeric"
              value={roundPrice}
              onChange={(e) => setRoundPrice(e.target.value)}
              placeholder="10000"
            />
            <Input
              label="% chào bán"
              inputMode="decimal"
              value={roundPct}
              onChange={(e) => setRoundPct(e.target.value)}
              placeholder="49"
            />
          </div>
          {rTotalShares > 0 && rForSale > 0 && (
            <div className="rounded-lg border border-gold-500/20 bg-gold-500/[0.05] p-3 text-sm text-cream/80">
              <div className="flex justify-between">
                <span>Tổng cổ phần (100%)</span>
                <span className="font-medium text-cream">{rTotalShares.toLocaleString("vi-VN")}</span>
              </div>
              <div className="flex justify-between">
                <span>Chào bán {formatPct(rPct)}</span>
                <span className="font-medium text-cream">
                  {rForSale.toLocaleString("vi-VN")} cp · {formatVnd(rPoolVnd)}
                </span>
              </div>
            </div>
          )}
        </div>
        <ErrorText>{error}</ErrorText>
      </Modal>
    </div>
  );
}

function Stat({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <Card>
      <p className="text-xs uppercase tracking-wide text-cream/45">{label}</p>
      <p className={`mt-2 font-serif text-2xl font-semibold ${accent ? "text-gold-400" : "text-cream"}`}>
        {value}
      </p>
    </Card>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <dt className="text-cream/50">{label}</dt>
      <dd className="font-medium text-cream/85">{value}</dd>
    </div>
  );
}

export default function AdminTiersPage() {
  return (
    <Guard requireRole="admin">
      <TiersInner />
    </Guard>
  );
}
