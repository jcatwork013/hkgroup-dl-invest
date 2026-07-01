package service_test

import (
	"errors"
	"math"
	"testing"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

// Happy path: register -> KYC -> contract -> sign -> declare -> reconcile -> approve -> shares.
func TestHappyPathInvestment(t *testing.T) {
	e := newTestEnv(t)
	user := e.newApprovedInvestor("happy@test.vn")
	tier := e.tierByAmount(50_000_000) // Gói 3 = 5,000 shares

	approved := e.fullInvest(user, tier)
	if approved.Status != db.InvestmentStatusApproved {
		t.Fatalf("want approved, got %s", approved.Status)
	}

	holding, err := e.store.GetShareholding(e.ctx(), user.ID)
	if err != nil {
		t.Fatalf("holding: %v", err)
	}
	if holding.Shares != tier.Shares {
		t.Fatalf("shares: want %d got %d", tier.Shares, holding.Shares)
	}
	// A single full-tier investment yields exactly the tier's ownership_pct (both are
	// shares/total_shares*100 stored as numeric(,4), so they round identically).
	if holding.OwnershipPct != tier.OwnershipPct {
		t.Fatalf("pct: want %v got %v", tier.OwnershipPct, holding.OwnershipPct)
	}

	dash, err := e.dashboard.Investor(e.ctx(), user.ID)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dash.CapitalContributed != tier.AmountVnd {
		t.Fatalf("capital: want %d got %d", tier.AmountVnd, dash.CapitalContributed)
	}
	if dash.DividendReceived != 0 {
		t.Fatalf("dividend should be 0 before any payout")
	}
}

// INVARIANT 1: SUM(shareholdings.shares) <= offering.shares_for_sale (cannot oversell the pool).
func TestInvariant1_CannotOversellPool(t *testing.T) {
	e := newTestEnv(t)
	whole := e.tierByAmount(500_000_000) // Gói 6 = 50,000 shares (largest single tier)

	// No single tier equals the 490,000 pool anymore, so shrink the pool to exactly one max tier
	// for this test (restored on cleanup) — keeps the invariant assertion crisp.
	off := e.activeOffering()
	if _, err := e.store.Pool().Exec(e.ctx(), `UPDATE offering SET shares_for_sale=$2 WHERE id=$1`, off.ID, whole.Shares); err != nil {
		t.Fatalf("shrink pool: %v", err)
	}
	t.Cleanup(func() {
		_, _ = e.store.Pool().Exec(e.ctx(), `UPDATE offering SET shares_for_sale=$2 WHERE id=$1`, off.ID, off.SharesForSale)
	})

	a := e.newApprovedInvestor("whale@test.vn")
	e.fullInvest(a, whole) // fills the pool exactly

	off2, _ := e.store.GetActiveOffering(e.ctx())
	if off2.SharesSold != off2.SharesForSale {
		t.Fatalf("pool not full: %d/%d", off2.SharesSold, off2.SharesForSale)
	}

	// Second investor cannot get any more shares.
	b := e.newApprovedInvestor("late@test.vn")
	small := e.tierByAmount(5_000_000)
	inv := e.createPendingInvestment(b, small)
	if _, err := e.investment.Reconcile(e.ctx(), e.admin.ID, inv.ID); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	_, err := e.investment.ApproveAndIssueShares(e.ctx(), e.admin.ID, inv.ID)
	if !errors.Is(err, service.ErrPoolExhausted) {
		t.Fatalf("want ErrPoolExhausted, got %v", err)
	}
}

// INVARIANT 2: status only flows pending -> reconciled -> approved. Skipping a step is rejected.
func TestInvariant2_StateMachine(t *testing.T) {
	e := newTestEnv(t)
	user := e.newApprovedInvestor("sm@test.vn")
	inv := e.createPendingInvestment(user, e.tierByAmount(5_000_000))

	// Cannot approve a pending (not yet reconciled) investment.
	if _, err := e.investment.ApproveAndIssueShares(e.ctx(), e.admin.ID, inv.ID); !errors.Is(err, service.ErrInvalidState) {
		t.Fatalf("approve(pending) want ErrInvalidState, got %v", err)
	}

	reconciled, err := e.investment.Reconcile(e.ctx(), e.admin.ID, inv.ID)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reconciled.Status != db.InvestmentStatusReconciled || !reconciled.ReconciledBy.Valid || !reconciled.ReconciledAt.Valid {
		t.Fatalf("reconcile must stamp actor+ts: %+v", reconciled)
	}

	approved, err := e.investment.ApproveAndIssueShares(e.ctx(), e.admin.ID, inv.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if !approved.ApprovedBy.Valid || !approved.ApprovedAt.Valid {
		t.Fatalf("approve must stamp actor+ts: %+v", approved)
	}
}

// INVARIANT 3 (atomicity) + 8 (audit in same tx): one approval issues ledger + holding + offering
// bump + audit logs together. We assert all four are consistent after a single approve.
func TestInvariant3and8_AtomicIssueWithAudit(t *testing.T) {
	e := newTestEnv(t)
	user := e.newApprovedInvestor("atomic@test.vn")
	tier := e.tierByAmount(100_000_000) // Gói 4 = 10,000 shares
	approved := e.fullInvest(user, tier)

	// ledger entry exists
	ledger, err := e.store.SumLedgerByUser(e.ctx(), user.ID)
	if err != nil || ledger != tier.Shares {
		t.Fatalf("ledger: want %d got %d (err %v)", tier.Shares, ledger, err)
	}
	// offering bumped
	off, _ := e.store.GetActiveOffering(e.ctx())
	if off.SharesSold != tier.Shares {
		t.Fatalf("offering.shares_sold: want %d got %d", tier.Shares, off.SharesSold)
	}
	// audit logs for approve + issue exist in the same flow
	logs, err := e.store.ListAuditLogsByEntity(e.ctx(), db.ListAuditLogsByEntityParams{
		Entity: "investments", EntityID: approved.ID.String(),
	})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	var sawApprove bool
	for _, l := range logs {
		if l.Action == "investment.approve" {
			sawApprove = true
		}
	}
	if !sawApprove {
		t.Fatalf("missing investment.approve audit log")
	}
}

// INVARIANT 4: shareholdings.shares == SUM(share_ledger.shares_delta) for every user.
func TestInvariant4_HoldingsReconcileWithLedger(t *testing.T) {
	e := newTestEnv(t)
	// two separate investments for the same user accumulate correctly
	user := e.newApprovedInvestor("acc@test.vn")
	t1 := e.tierByAmount(100_000_000) // Gói 4 = 10,000 shares
	t2 := e.tierByAmount(50_000_000)  // Gói 3 = 5,000 shares
	e.fullInvest(user, t1)
	e.fullInvest(user, t2)

	holding, _ := e.store.GetShareholding(e.ctx(), user.ID)
	ledger, _ := e.store.SumLedgerByUser(e.ctx(), user.ID)
	if holding.Shares != ledger {
		t.Fatalf("holding %d != ledger %d", holding.Shares, ledger)
	}
	if holding.Shares != t1.Shares+t2.Shares {
		t.Fatalf("want %d got %d", t1.Shares+t2.Shares, holding.Shares)
	}

	mismatches, err := e.dashboard.IntegrityCheck(e.ctx())
	if err != nil {
		t.Fatalf("integrity: %v", err)
	}
	if len(mismatches) != 0 {
		t.Fatalf("integrity mismatches: %+v", mismatches)
	}
}

// INVARIANT 5: commission only on approved investments. F1 (direct) customer commission uses the
// tier's per-package rate. Multi-level payout (F2/F3) is covered by TestInvariant5_MultiLevelCommission.
func TestInvariant5_CommissionF1Customer(t *testing.T) {
	e := newTestEnv(t)

	// referrer -> referee (customer): commission expected.
	referrer := e.newApprovedInvestor("ref-cust@test.vn")
	referee := e.newApprovedInvestor("buyer-cust@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = referrer.ReferralCode
		in.ReferralType = "customer"
	})
	tier := e.tierByAmount(50_000_000)
	e.fullInvest(referee, tier)

	comms, err := e.referral.ListByReferrer(e.ctx(), referrer.ID)
	if err != nil {
		t.Fatalf("list comm: %v", err)
	}
	if len(comms) != 1 {
		t.Fatalf("want 1 commission, got %d", len(comms))
	}
	c := comms[0]
	// Commission base = invested amount; rate = the tier's commission_rate; amount = round(base×rate).
	if c.BaseAmount != tier.AmountVnd {
		t.Fatalf("commission base: want %d got %d", tier.AmountVnd, c.BaseAmount)
	}
	if c.Rate != tier.CommissionRate {
		t.Fatalf("commission rate: want %v got %v", tier.CommissionRate, c.Rate)
	}
	wantAmount := int64(math.Round(float64(c.BaseAmount) * c.Rate))
	if c.Amount != wantAmount {
		t.Fatalf("commission amount: want %d got %d", wantAmount, c.Amount)
	}
	if c.NetAmount != c.Amount-c.TaxPit {
		t.Fatalf("net must equal amount - tax")
	}
	if c.Level != 1 {
		t.Fatalf("direct referrer commission must be level 1, got %d", c.Level)
	}
}

// INVARIANT 5 (multi-level): one investment pays F1/F2/F3 up the direct-referrer chain. F1 = the
// referee's direct referrer (per-tier rate), F2/F3 = the next two uplines (configured flat rates).
// Applies to BOTH customer and investor chains; here we use a 4-level customer chain.
func TestInvariant5_MultiLevelCommission(t *testing.T) {
	e := newTestEnv(t)
	// Wire settings + configure F2/F3 rates (as an admin would), then enable them on the referral svc.
	ss := service.NewSettingsService(e.store)
	if _, err := ss.Update(e.ctx(), e.admin.ID, map[string]string{
		"referral_f2_rate": "0.02",
		"referral_f3_rate": "0.01",
	}); err != nil {
		t.Fatalf("set F2/F3 rates: %v", err)
	}
	e.referral.SetSettings(ss)

	f3 := e.newApprovedInvestor("f3-up@test.vn") // top of chain, no referrer above
	f2 := e.newApprovedInvestor("f2-up@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = f3.ReferralCode
		in.ReferralType = "customer"
	})
	f1 := e.newApprovedInvestor("f1-up@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = f2.ReferralCode
		in.ReferralType = "customer"
	})
	buyer := e.newApprovedInvestor("buyer-ml@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = f1.ReferralCode
		in.ReferralType = "customer"
	})

	tier := e.tierByAmount(50_000_000)
	e.fullInvest(buyer, tier)

	// Each upline earns exactly one commission at its level; the buyer earns nothing.
	check := func(user db.User, wantLevel int16, wantRate float64) {
		t.Helper()
		comms, err := e.referral.ListByReferrer(e.ctx(), user.ID)
		if err != nil {
			t.Fatalf("list comm: %v", err)
		}
		if len(comms) != 1 {
			t.Fatalf("level %d earner want 1 commission, got %d", wantLevel, len(comms))
		}
		c := comms[0]
		if c.Level != wantLevel {
			t.Fatalf("want level %d, got %d", wantLevel, c.Level)
		}
		if c.Rate != wantRate {
			t.Fatalf("level %d rate: want %v got %v", wantLevel, wantRate, c.Rate)
		}
		wantAmount := int64(math.Round(float64(tier.AmountVnd) * wantRate))
		if c.Amount != wantAmount {
			t.Fatalf("level %d amount: want %d got %d", wantLevel, wantAmount, c.Amount)
		}
		if c.NetAmount != c.Amount-c.TaxPit {
			t.Fatalf("level %d net must equal amount - tax", wantLevel)
		}
	}
	check(f1, 1, tier.CommissionRate) // F1 = per-tier rate
	check(f2, 2, 0.02)                // F2 = 2%
	check(f3, 3, 0.01)                // F3 = 1%

	if buyerComms, _ := e.referral.ListByReferrer(e.ctx(), buyer.ID); len(buyerComms) != 0 {
		t.Fatalf("buyer must not earn commission on own investment, got %d", len(buyerComms))
	}
}

// INVARIANT 5 (negative): investor-type referrals never generate cash commission.
func TestInvariant5_InvestorReferralNoCommission(t *testing.T) {
	e := newTestEnv(t)
	referrer := e.newApprovedInvestor("ref-inv@test.vn")
	referee := e.newApprovedInvestor("buyer-inv@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = referrer.ReferralCode
		in.ReferralType = "investor"
	})
	e.fullInvest(referee, e.tierByAmount(50_000_000))

	comms, _ := e.referral.ListByReferrer(e.ctx(), referrer.ID)
	if len(comms) != 0 {
		t.Fatalf("investor referral must NOT pay cash commission, got %d", len(comms))
	}
}

// INVARIANT 6: dividends exist only when an admin declares them; payouts are pro-rata and the
// investor "money received back" equals the REAL paid dividend.
func TestInvariant6_DividendOnlyByAdmin(t *testing.T) {
	e := newTestEnv(t)
	a := e.newApprovedInvestor("d-a@test.vn")
	b := e.newApprovedInvestor("d-b@test.vn")
	e.fullInvest(a, e.tierByAmount(100_000_000)) // Gói 4 = 10,000 shares
	e.fullInvest(b, e.tierByAmount(50_000_000))  // Gói 3 =  5,000 shares  => a:b = 2:1

	// No dividend exists until admin declares one.
	if before, _ := e.store.SumDividendPaidToUser(e.ctx(), a.ID); before != 0 {
		t.Fatalf("no dividend should exist before declaration")
	}

	div, payouts, err := e.dividend.Declare(e.ctx(), e.admin.ID, "2026-Q1", 30000000, "test")
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if div.TotalAmount != 30000000 || len(payouts) != 2 {
		t.Fatalf("unexpected dividend: total=%d payouts=%d", div.TotalAmount, len(payouts))
	}

	// pro-rata: a gets 2/3, b gets 1/3
	share := map[string]int64{}
	for _, p := range payouts {
		share[p.UserID.String()] = p.Amount
	}
	if share[a.ID.String()] != 20000000 || share[b.ID.String()] != 10000000 {
		t.Fatalf("pro-rata wrong: a=%d b=%d", share[a.ID.String()], share[b.ID.String()])
	}

	// Mark a's payout paid -> dashboard reflects REAL received dividend.
	for _, p := range payouts {
		if p.UserID == a.ID {
			if _, err := e.dividend.MarkPaid(e.ctx(), e.admin.ID, p.ID); err != nil {
				t.Fatalf("mark paid: %v", err)
			}
		}
	}
	got, _ := e.store.SumDividendPaidToUser(e.ctx(), a.ID)
	if got != 20000000 {
		t.Fatalf("dividend received: want 20000000 got %d", got)
	}
	// b not marked paid yet -> still 0
	if gotB, _ := e.store.SumDividendPaidToUser(e.ctx(), b.ID); gotB != 0 {
		t.Fatalf("unpaid payout must not count: got %d", gotB)
	}
}

// INVARIANT 7: shares are never issued without a reconciled payment.
func TestInvariant7_NoIssueWithoutReconciledPayment(t *testing.T) {
	e := newTestEnv(t)
	user := e.newApprovedInvestor("norecon@test.vn")
	inv := e.createPendingInvestment(user, e.tierByAmount(5_000_000))

	// Force the investment to 'reconciled' WITHOUT reconciling the payment, to prove the service
	// still refuses to issue. (Direct status flip mimics a buggy/manual path.)
	_, err := e.store.Pool().Exec(e.ctx(),
		`UPDATE investments SET status='reconciled', reconciled_by=$2, reconciled_at=now() WHERE id=$1`,
		inv.ID, e.admin.ID)
	if err != nil {
		t.Fatalf("force reconcile: %v", err)
	}

	_, err = e.investment.ApproveAndIssueShares(e.ctx(), e.admin.ID, inv.ID)
	if !errors.Is(err, service.ErrInvalidState) {
		t.Fatalf("want ErrInvalidState (payment not reconciled), got %v", err)
	}
	// no shares issued
	if _, err := e.store.GetShareholding(e.ctx(), user.ID); err == nil {
		t.Fatalf("no shareholding should exist")
	}
}

// Idempotency: approving twice issues shares only once (protects the "duyệt" double-click).
func TestIdempotentApprove(t *testing.T) {
	e := newTestEnv(t)
	user := e.newApprovedInvestor("idem@test.vn")
	tier := e.tierByAmount(50_000_000)
	inv := e.createPendingInvestment(user, tier)
	if _, err := e.investment.Reconcile(e.ctx(), e.admin.ID, inv.ID); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if _, err := e.investment.ApproveAndIssueShares(e.ctx(), e.admin.ID, inv.ID); err != nil {
		t.Fatalf("approve 1: %v", err)
	}
	if _, err := e.investment.ApproveAndIssueShares(e.ctx(), e.admin.ID, inv.ID); err != nil {
		t.Fatalf("approve 2 (idempotent) should not error: %v", err)
	}
	holding, _ := e.store.GetShareholding(e.ctx(), user.ID)
	if holding.Shares != tier.Shares {
		t.Fatalf("double approve issued twice: want %d got %d", tier.Shares, holding.Shares)
	}
}
