// e2e-autopay: KIỂM THỬ — tạo 1 đơn test rồi thanh toán, để chứng minh luồng TỰ ĐỘNG chia cổ tức
// (dividend_auto_distribute=on) chạy đúng: đơn paid → tự sinh dividend + payouts (paid) + swept.
// Chỉ dùng trên môi trường test. Env: DATABASE_URL, ADMIN_ID. Flags: -seller, -product.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/service"
	"github.com/hkgroup/backend/internal/store"
)

func main() {
	seller := flag.String("seller", "", "saler user id")
	product := flag.String("product", "", "product id")
	flag.Parse()

	ctx := context.Background()
	pool, err := store.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)
	set := service.NewSettingsService(st)
	sales := service.NewSalesService(st, set)
	admin := uuid.MustParse(os.Getenv("ADMIN_ID"))

	o, err := sales.CreateOrder(ctx, admin, true, service.OrderInput{
		CustomerName: "E2E AutoPay", CustomerPhone: "0900000001", SellerID: *seller,
		Items: []service.OrderItemInput{{ProductID: *product, Qty: 1}},
	})
	if err != nil {
		log.Fatalf("create order: %v", err)
	}
	log.Printf("[e2e] tạo đơn %s (%s) subtotal=%d — status=%s", o.Code, o.ID, o.SubtotalVnd, o.Status)

	detail, err := sales.PayOrder(ctx, admin, true, o.ID)
	if err != nil {
		log.Fatalf("pay order: %v", err)
	}
	log.Printf("[e2e] thanh toán xong: %s status=%s — kiểm DB xem dividend.auto + payouts paid + swept",
		detail.Order.Code, detail.Order.Status)
}
