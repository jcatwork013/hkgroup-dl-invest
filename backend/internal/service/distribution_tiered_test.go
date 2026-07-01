package service_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

// accounts builds active-account rows from invested-VND amounts (banding is by amount_vnd).
func accounts(investedVnd ...int64) []db.ListActiveAccountsRow {
	out := make([]db.ListActiveAccountsRow, len(investedVnd))
	for i, inv := range investedVnd {
		out[i] = db.ListActiveAccountsRow{UserID: uuid.New(), Shares: 100, InvestedVnd: inv}
	}
	return out
}

func cfg(residual string) service.TieredCfgForTest {
	return service.TieredCfgForTest{ResidualMode: residual}
}

// Reproduces the hand calculation: revenue 1,000,000 · 30 accounts all in band1.
//
//	equalPool = 90,000 → 3,000 each ; bonusPool = 60,000 chia HẾT theo tỉ lệ hạng.
//	Σrate = 30 × 1.5% = 0.45 → mỗi người = 60,000 × 0.015/0.45 = 2,000. Pool bonus phát hết.
func TestPlanTiered_ReproducesHandCalc(t *testing.T) {
	// 10tr invested → band1 (<50tr).
	hs := make([]int64, 30)
	for i := range hs {
		hs[i] = 10_000_000
	}
	plan := service.PlanTieredForTest("2026-06", 1_000_000, accounts(hs...), cfg("retain"))

	if plan.EqualPool != 90_000 || plan.BonusPool != 60_000 {
		t.Fatalf("pools: equal=%d bonus=%d", plan.EqualPool, plan.BonusPool)
	}
	if plan.EqualEach != 3_000 {
		t.Fatalf("equalEach: want 3000 got %d", plan.EqualEach)
	}
	if plan.Payouts[0].Bonus != 2_000 {
		t.Fatalf("band1 bonus: want 2000 got %d", plan.Payouts[0].Bonus)
	}
	if plan.Payouts[0].Amount != 5_000 {
		t.Fatalf("amount: want 5000 got %d", plan.Payouts[0].Amount)
	}
	// Bonus chia hết theo tỉ lệ (30×2,000 = 60,000) + equal hết (90,000) → không còn dư.
	if plan.Residual != 0 {
		t.Fatalf("residual: want 0 got %d", plan.Residual)
	}
	if plan.Distributed != 150_000 {
		t.Fatalf("distributed: want 150000 got %d", plan.Distributed)
	}
}

// rollover must distribute the WHOLE pool (pie chart always full): 90,000+60,000 = 150,000.
func TestPlanTiered_RolloverFillsPool(t *testing.T) {
	hs := make([]int64, 30)
	for i := range hs {
		hs[i] = 10_000_000 // band1
	}
	plan := service.PlanTieredForTest("2026-06", 1_000_000, accounts(hs...), cfg("rollover"))

	if plan.Residual != 0 {
		t.Fatalf("residual: want 0 got %d", plan.Residual)
	}
	if plan.Distributed != 150_000 {
		t.Fatalf("distributed: want 150000 got %d", plan.Distributed)
	}
	var sum int64
	for _, p := range plan.Payouts {
		sum += p.Amount
	}
	if sum != 150_000 {
		t.Fatalf("sum payouts: want 150000 got %d", sum)
	}
	// 30 tài khoản cùng band1 → bonus chia đều 2,000 + equal 3,000 = 5,000; pool phát hết.
	if plan.Payouts[0].Amount != 5_000 {
		t.Fatalf("amount: want 5000 got %d", plan.Payouts[0].Amount)
	}
}

// Bonus pool chia HẾT theo tỉ lệ hạng: 3 tài khoản 2,5%/2%/1,5% (Σ=6%) trên pool 60k → 25k/20k/15k.
// Cộng đồng chia 30k/người → 55k/50k/45k, phát hết 150k. Đây đúng case client yêu cầu.
func TestPlanTiered_ProportionalByBand(t *testing.T) {
	// 300tr (band3 2,5%), 100tr (band2 2%), 5tr (band1 1,5%).
	plan := service.PlanTieredForTest("2026-06", 1_000_000, accounts(300_000_000, 100_000_000, 5_000_000), cfg("rollover"))

	if plan.EqualEach != 30_000 {
		t.Fatalf("equalEach: want 30000 got %d", plan.EqualEach)
	}
	wantBonus := []int64{25_000, 20_000, 15_000}
	wantAmount := []int64{55_000, 50_000, 45_000}
	for i, p := range plan.Payouts {
		if p.Bonus != wantBonus[i] {
			t.Fatalf("payout %d bonus: want %d got %d", i, wantBonus[i], p.Bonus)
		}
		if p.Amount != wantAmount[i] {
			t.Fatalf("payout %d amount: want %d got %d", i, wantAmount[i], p.Amount)
		}
	}
	if plan.Distributed != 150_000 {
		t.Fatalf("distributed: want 150000 got %d", plan.Distributed)
	}
	if plan.Residual != 0 {
		t.Fatalf("residual: want 0 got %d", plan.Residual)
	}
}
