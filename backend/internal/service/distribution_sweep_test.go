package service_test

import (
	"testing"

	"github.com/hkgroup/backend/internal/service"
)

// Quét cổ tức chia một pool ĐÃ TÍNH SẴN (tổng 15% dividend_pool_vnd của các đơn) — KHÔNG cắt 15%
// lần nữa. Bất biến sống còn: Σ payouts == pool CHÍNH XÁC (rollover), không thất thoát/đẻ đồng nào,
// với mọi pool lẻ và mọi số tài khoản.
func TestPlanTieredFromPool_DistributesExactPool(t *testing.T) {
	pools := []int64{1, 7, 100, 45_000, 209_550, 999_999, 1_000_000_003}
	sizes := []int{1, 3, 8, 30}
	for _, pool := range pools {
		for _, n := range sizes {
			hs := make([]int64, n)
			for i := range hs {
				hs[i] = int64(i+1) * 10_000_000 // rải qua cả 3 hạng
			}
			plan := service.PlanTieredFromPoolForTest("QCT", pool*3, pool, accounts(hs...), cfg("rollover"))

			if plan.EqualPool+plan.BonusPool != pool {
				t.Fatalf("pool=%d n=%d: split equal=%d + bonus=%d != pool", pool, n, plan.EqualPool, plan.BonusPool)
			}
			var sum int64
			for _, p := range plan.Payouts {
				sum += p.Amount
			}
			if sum != pool {
				t.Fatalf("pool=%d n=%d: Σpayouts=%d != pool", pool, n, sum)
			}
			if plan.Distributed != pool || plan.Residual != 0 {
				t.Fatalf("pool=%d n=%d: distributed=%d residual=%d (muốn distributed=pool, residual=0)", pool, n, plan.Distributed, plan.Residual)
			}
		}
	}
}

// Không có tài khoản active ⇒ không chia được (toàn bộ pool là residual, không đẻ payout).
func TestPlanTieredFromPool_NoAccounts(t *testing.T) {
	plan := service.PlanTieredFromPoolForTest("QCT", 1_000_000, 150_000, accounts(), cfg("rollover"))
	if plan.Distributed != 0 || len(plan.Payouts) != 0 {
		t.Fatalf("no accounts: distributed=%d payouts=%d (muốn 0/0)", plan.Distributed, len(plan.Payouts))
	}
	if plan.Residual != 150_000 {
		t.Fatalf("no accounts: residual=%d (muốn = pool 150000)", plan.Residual)
	}
}
