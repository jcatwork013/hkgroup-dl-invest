// sweep-dividend: lệnh MỘT LẦN — quét các đơn thành công (paid) CHƯA gộp và chia Pool Cổ Đông (15%
// đã trích sẵn mỗi đơn) thành 1 đợt cổ tức thực. Lần đầu tự cuốn cả đơn CŨ (backfill).
//
// AN TOÀN: gọi ĐÚNG service.SweepDividend (idempotent qua cột swept + FOR UPDATE). Mặc định PREVIEW
// (không ghi). Thêm -apply để thực sự quét.
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
	apply := flag.Bool("apply", false, "thực sự quét & chia (mặc định false = preview, không ghi)")
	adminStr := flag.String("admin", os.Getenv("SWEEP_ADMIN_ID"), "admin uuid (declared_by/created_by)")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("cần DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := store.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("kết nối DB: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	dividendSvc := service.NewDividendService(st)
	settingsSvc := service.NewSettingsService(st)
	distSvc := service.NewDistributionService(st, dividendSvc, settingsSvc)

	pv, err := distSvc.SweepPreview(ctx)
	if err != nil {
		log.Fatalf("preview: %v", err)
	}
	log.Printf("[sweep] đơn chưa gộp=%d · doanh thu=%d₫ · pool cổ đông 15%%=%d₫",
		pv.SweptOrders, pv.RevenueVnd, pv.PoolVnd)
	if pv.PoolVnd > 0 {
		log.Printf("[sweep] kế hoạch chia: equalPool=%d bonusPool=%d distributed=%d residual=%d payouts=%d",
			pv.Plan.EqualPool, pv.Plan.BonusPool, pv.Plan.Distributed, pv.Plan.Residual, len(pv.Plan.Payouts))
	}

	if !*apply {
		log.Println("[sweep] PREVIEW — chưa ghi gì. Thêm -apply để quét thật.")
		return
	}
	if pv.PoolVnd <= 0 {
		log.Println("[sweep] không có gì để quét — dừng.")
		return
	}
	admin, err := uuid.Parse(*adminStr)
	if err != nil {
		log.Fatalf("admin uuid không hợp lệ (-admin hoặc SWEEP_ADMIN_ID): %v", err)
	}
	res, err := distSvc.SweepDividend(ctx, admin)
	if err != nil {
		log.Fatalf("sweep: %v", err)
	}
	log.Printf("[sweep] XONG: quét %d đơn · pool=%d₫ · dividend=%s · payouts=%d · distributed=%d₫",
		res.SweptOrders, res.PoolVnd, res.Dividend.ID, len(res.Payouts), res.Plan.Distributed)

	// Kiểm chứng bất biến ngay tại chỗ: Σ payouts == pool.
	var sum int64
	for _, p := range res.Payouts {
		sum += p.Amount
	}
	if sum != res.PoolVnd {
		log.Fatalf("BẤT BIẾN VỠ: Σpayouts=%d != pool=%d", sum, res.PoolVnd)
	}
	log.Printf("[sweep] ✓ bất biến Σpayouts == pool (%d₫)", sum)
}
