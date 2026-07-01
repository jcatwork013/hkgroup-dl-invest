"use client";

import Guard from "@/components/Guard";
import OrdersPanel from "@/components/OrdersPanel";

export default function AdminOrdersPage() {
  return (
    <Guard requireRole="admin">
      <div className="space-y-6">
        <div>
          <h1 className="font-serif text-2xl text-cream">Đơn hàng bán</h1>
          <p className="mt-1 text-sm text-cream/55">Tạo đơn cho người bán, xác nhận thanh toán để chia hoa hồng + dòng tiền.</p>
        </div>
        <OrdersPanel admin />
      </div>
    </Guard>
  );
}
