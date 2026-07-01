"use client";

import Guard from "@/components/Guard";
import OrdersPanel from "@/components/OrdersPanel";

export default function SalerOrdersPage() {
  return (
    <Guard requireRole="saler">
      <div className="space-y-6">
        <div>
          <h1 className="font-serif text-2xl text-cream">Bán hàng</h1>
          <p className="mt-1 text-sm text-cream/55">Tạo đơn cho khách. Khi đơn được xác nhận thanh toán, bạn nhận hoa hồng 25%.</p>
        </div>
        <OrdersPanel admin={false} />
      </div>
    </Guard>
  );
}
