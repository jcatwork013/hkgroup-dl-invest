package service_test

import (
	"encoding/json"
	"testing"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

// tierByAmount picks a seeded tier by its VND package size (5tr/20tr/50tr/100tr/300tr/500tr).
func (e *testEnv) tierByAmount(amountVnd int64) db.InvestmentTier {
	e.t.Helper()
	tiers, err := e.store.ListTiers(e.ctx(), e.activeOffering().ID)
	if err != nil {
		e.t.Fatalf("tiers: %v", err)
	}
	for _, tr := range tiers {
		if tr.AmountVnd == amountVnd {
			return tr
		}
	}
	e.t.Fatalf("tier with amount %d not found", amountVnd)
	return db.InvestmentTier{}
}

// Exercises the tiered "đồng chia 9% + bonus 6%" distribution end-to-end against a live DB.
// The seeded tiers map onto the bonus bands EXACTLY under default thresholds (banding is by the
// true approved amount_vnd):
//
//	5tr, 20tr  → band1 (<50tr)      bonus 1.5%
//	50tr, 100tr→ band2 (50tr–<300tr) bonus 2%
//	300tr      → band3 (≥300tr)      bonus 2.5%
func TestDistributeTiered_EndToEnd(t *testing.T) {
	e := newTestEnv(t)
	ctx := e.ctx()

	settings := service.NewSettingsService(e.store)
	dist := service.NewDistributionService(e.store, e.dividend, settings)

	// Seed 5 investors across the three bands. Total shares = 500+2000+5000+10000+30000 = 47,500
	// ≤ shares_for_sale (490,000) → invariant 1 holds.
	e.fullInvest(e.newApprovedInvestor("b1a@test.vn"), e.tierByAmount(5_000_000))   // band1
	e.fullInvest(e.newApprovedInvestor("b1b@test.vn"), e.tierByAmount(20_000_000))  // band1
	e.fullInvest(e.newApprovedInvestor("b2a@test.vn"), e.tierByAmount(50_000_000))  // band2 (boundary!)
	e.fullInvest(e.newApprovedInvestor("b2b@test.vn"), e.tierByAmount(100_000_000)) // band2
	e.fullInvest(e.newApprovedInvestor("b3a@test.vn"), e.tierByAmount(300_000_000)) // band3 (boundary!)

	const revenue = 1_000_000_000 // 1 tỷ doanh thu kỳ

	// --- Preview (no write) ---
	plan, err := dist.PreviewTiered(ctx, "2026-06", revenue)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	logJSON(t, "PREVIEW plan", plan)

	if plan.Accounts != 5 {
		t.Fatalf("accounts: want 5 got %d", plan.Accounts)
	}
	if plan.EqualPool != 90_000_000 || plan.BonusPool != 60_000_000 {
		t.Fatalf("pools: equal=%d bonus=%d", plan.EqualPool, plan.BonusPool)
	}
	if plan.EqualEach != 18_000_000 { // 90tr / 5
		t.Fatalf("equalEach: want 18,000,000 got %d", plan.EqualEach)
	}
	// Pie-chart band breakdown — proves the 50tr & 300tr boundaries land in the RIGHT band.
	// Bonus chia HẾT theo tỉ lệ hạng: Σrate = 2×1.5% + 2×2% + 1×2.5% = 9.5%.
	//   band1 mỗi người = 60tr × 0.015/0.095 = 9,473,684 (×2)
	//   band2 mỗi người = 60tr × 0.02/0.095  = 12,631,578 (×2)
	//   band3          = 60tr × 0.025/0.095  = 15,789,473
	wantBand := map[string]struct {
		accounts int
		bonus    int64
	}{
		"band1": {2, 18_947_368},
		"band2": {2, 25_263_156},
		"band3": {1, 15_789_473},
	}
	for _, b := range plan.Bands {
		w := wantBand[b.Key]
		if b.Accounts != w.accounts || b.BonusTotal != w.bonus {
			t.Fatalf("band %s: got accounts=%d bonus=%d want accounts=%d bonus=%d",
				b.Key, b.Accounts, b.BonusTotal, w.accounts, w.bonus)
		}
	}

	// --- Commit (rollover default): whole 15% pool distributed, nothing lost ---
	res, err := dist.DistributeTiered(ctx, e.admin.ID, "2026-06", revenue)
	if err != nil {
		t.Fatalf("distribute: %v", err)
	}
	logJSON(t, "COMMITTED plan", res.Plan)

	if res.Plan.Scaled {
		t.Fatal("overflow van must NOT fire here (pool far exceeds bonus)")
	}
	if res.Plan.Residual != 0 {
		t.Fatalf("rollover residual: want 0 got %d", res.Plan.Residual)
	}
	if res.Plan.Distributed != 150_000_000 { // 15% × 1tỷ, fully distributed
		t.Fatalf("distributed: want 150,000,000 got %d", res.Plan.Distributed)
	}
	if len(res.Payouts) != 5 {
		t.Fatalf("payouts: want 5 got %d", len(res.Payouts))
	}
	var sum int64
	for _, p := range res.Payouts {
		if p.Amount <= 0 {
			t.Fatalf("payout %s non-positive: %d", p.UserID, p.Amount)
		}
		sum += p.Amount
	}
	if sum != 150_000_000 {
		t.Fatalf("sum payouts: want 150,000,000 got %d", sum)
	}
	if res.Dividend.TotalAmount != 150_000_000 {
		t.Fatalf("dividend total: want 150,000,000 got %d", res.Dividend.TotalAmount)
	}

	// --- Persistence: payouts queryable, distribution recorded ---
	stored, err := e.store.ListPayoutsByDividend(ctx, res.Dividend.ID)
	if err != nil {
		t.Fatalf("list payouts: %v", err)
	}
	if len(stored) != 5 {
		t.Fatalf("stored payouts: want 5 got %d", len(stored))
	}
	dists, err := dist.List(ctx)
	if err != nil {
		t.Fatalf("list distributions: %v", err)
	}
	if len(dists) != 1 || dists[0].InvestorPool != 150_000_000 {
		t.Fatalf("revenue_distribution not recorded correctly: %+v", dists)
	}
}

// TestDistributeTiered_RetainShowsResidual: vì bonus nay chia HẾT theo tỉ lệ hạng, "retain" chỉ còn
// giữ lại phần LẺ do làm tròn floor (vài đồng), không còn khoản dư lớn như mô hình % cố định cũ.
func TestDistributeTiered_RetainShowsResidual(t *testing.T) {
	e := newTestEnv(t)
	ctx := e.ctx()

	settings := service.NewSettingsService(e.store)
	dist := service.NewDistributionService(e.store, e.dividend, settings)
	if _, err := settings.Update(ctx, e.admin.ID, map[string]string{
		"dist_residual_mode": "retain",
	}); err != nil {
		t.Fatalf("configure: %v", err)
	}

	// Same 5-account spread as the end-to-end test.
	e.fullInvest(e.newApprovedInvestor("r1a@test.vn"), e.tierByAmount(5_000_000))
	e.fullInvest(e.newApprovedInvestor("r1b@test.vn"), e.tierByAmount(20_000_000))
	e.fullInvest(e.newApprovedInvestor("r2a@test.vn"), e.tierByAmount(50_000_000))
	e.fullInvest(e.newApprovedInvestor("r2b@test.vn"), e.tierByAmount(100_000_000))
	e.fullInvest(e.newApprovedInvestor("r3a@test.vn"), e.tierByAmount(300_000_000))

	res, err := dist.DistributeTiered(ctx, e.admin.ID, "2026-07", 1_000_000_000)
	if err != nil {
		t.Fatalf("distribute: %v", err)
	}
	logJSON(t, "RETAIN plan", res.Plan)

	// equal hết (90tr); bonus chia hết theo tỉ lệ = 59,999,997 (lẻ 3đ do floor) → residual = 3.
	const wantBonusSpent = 59_999_997
	if res.Plan.Residual != 60_000_000-wantBonusSpent {
		t.Fatalf("residual: want %d got %d", 60_000_000-wantBonusSpent, res.Plan.Residual)
	}
	if res.Plan.Distributed != 90_000_000+wantBonusSpent {
		t.Fatalf("distributed: want %d got %d", 90_000_000+wantBonusSpent, res.Plan.Distributed)
	}
}

func logJSON(t *testing.T, label string, v any) {
	t.Helper()
	b, _ := json.MarshalIndent(v, "", "  ")
	t.Logf("%s:\n%s", label, b)
}
