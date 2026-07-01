package service

import (
	"testing"

	"github.com/hkgroup/backend/internal/db"
)

// policyRates = đúng chính sách 25/10/5/15/30/15 (ngưỡng đồng chia 1tr).
var policyRates = salesRates{
	seller: 0.25, affiliate: 0.10, equalShare: 0.05, pool: 0.15, cost: 0.30, operations: 0.15,
	equalShareMin: 1_000_000,
}

func assertSum(t *testing.T, d db.CreateSalesDistributionParams) {
	t.Helper()
	sum := d.SellerVnd + d.AffiliateVnd + d.EqualShareVnd + d.PoolVnd + d.CostVnd + d.OperationsVnd
	if sum != d.TotalVnd {
		t.Fatalf("tổng 6 khoản = %d, phải = subtotal %d", sum, d.TotalVnd)
	}
	// Pool CỔ ĐÔNG = 15% (pool_vnd) MÀ THÔI. 5% đồng chia là pool RIÊNG của người mua, không gộp vào.
	if d.DividendPoolVnd != d.PoolVnd {
		t.Fatalf("pool co dong = %d, phai = pool_vnd = %d", d.DividendPoolVnd, d.PoolVnd)
	}
}

func TestSplitOrder_FullWithAffiliate(t *testing.T) {
	d := splitOrder(2_000_000, true, policyRates)
	if d.SellerVnd != 500_000 || d.AffiliateVnd != 200_000 || d.EqualShareVnd != 100_000 ||
		d.PoolVnd != 300_000 || d.CostVnd != 600_000 || d.OperationsVnd != 300_000 || d.DividendPoolVnd != 300_000 {
		t.Fatalf("chia sai: %+v", d)
	}
	assertSum(t, d)
}

func TestSplitOrder_NoAffiliate_FoldsIntoOps(t *testing.T) {
	// Không affiliate → 10% (200k) dồn vào vận hành (300k+200k=500k). Tổng vẫn = subtotal.
	d := splitOrder(2_000_000, false, policyRates)
	if d.AffiliateVnd != 0 {
		t.Fatalf("affiliate phải = 0, được %d", d.AffiliateVnd)
	}
	if d.OperationsVnd != 500_000 {
		t.Fatalf("operations phai hap thu 10pct affiliate = 500k, duoc %d", d.OperationsVnd)
	}
	assertSum(t, d)
}

func TestSplitOrder_BelowEqualShareThreshold(t *testing.T) {
	// Đơn < 1tr → KHÔNG đồng chia; 5% dồn vào vận hành. Tổng vẫn = subtotal.
	d := splitOrder(500_000, true, policyRates)
	if d.EqualShareVnd != 0 {
		t.Fatalf("đơn < 1tr không được có đồng chia, được %d", d.EqualShareVnd)
	}
	assertSum(t, d)
}

func TestSplitOrder_AlwaysSumsToSubtotal(t *testing.T) {
	for _, sub := range []int64{1, 999_999, 1_000_000, 1_234_567, 99_999_999} {
		for _, aff := range []bool{true, false} {
			assertSum(t, splitOrder(sub, aff, policyRates))
		}
	}
}
