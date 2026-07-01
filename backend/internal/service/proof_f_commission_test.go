package service_test

import (
	"testing"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/service"
)

// PROOF: hoa hồng giới thiệu 3 cấp phải là F1 = 3%, F2 = 2%, F3 = 1% (đồng nhất mọi gói).
// Dựng chuỗi F3 <- F2 <- F1 <- buyer, buyer đầu tư gói 100.000.000đ rồi được duyệt.
// Thuế TNCN 10% trên hoa hồng gross. Khẳng định tỷ lệ + số tiền chính xác từng cấp.
func TestProof_F1_3pct_F2_2pct_F3_1pct(t *testing.T) {
	e := newTestEnv(t)

	// Bật F2/F3 như admin cấu hình (site_settings bị truncate trong test env).
	ss := service.NewSettingsService(e.store)
	if _, err := ss.Update(e.ctx(), e.admin.ID, map[string]string{
		"referral_f2_rate": "0.02",
		"referral_f3_rate": "0.01",
	}); err != nil {
		t.Fatalf("set F2/F3: %v", err)
	}
	e.referral.SetSettings(ss)

	f3 := e.newApprovedInvestor("proof-f3@test.vn")
	f2 := e.newApprovedInvestor("proof-f2@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = f3.ReferralCode
		in.ReferralType = "customer"
	})
	f1 := e.newApprovedInvestor("proof-f1@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = f2.ReferralCode
		in.ReferralType = "customer"
	})
	buyer := e.newApprovedInvestor("proof-buyer@test.vn", func(in *service.RegisterInput) {
		in.ReferralCode = f1.ReferralCode
		in.ReferralType = "customer"
	})

	tier := e.tierByAmount(100_000_000) // Gói 4 — chính là gói trong ảnh chụp (trước đây sai 20%).
	e.fullInvest(buyer, tier)

	// (earner, cấp, %, gross, thuế 10%, thực nhận)
	type want struct {
		who   db.User
		level int16
		rate  float64
		gross int64
		tax   int64
		net   int64
	}
	cases := []want{
		{f1, 1, 0.03, 3_000_000, 300_000, 2_700_000},
		{f2, 2, 0.02, 2_000_000, 200_000, 1_800_000},
		{f3, 3, 0.01, 1_000_000, 100_000, 900_000},
	}

	for _, w := range cases {
		comms, err := e.referral.ListByReferrer(e.ctx(), w.who.ID)
		if err != nil {
			t.Fatalf("F%d list: %v", w.level, err)
		}
		if len(comms) != 1 {
			t.Fatalf("F%d: muốn 1 hoa hồng, có %d", w.level, len(comms))
		}
		c := comms[0]
		if c.Level != w.level {
			t.Fatalf("F%d: sai cấp, got level %d", w.level, c.Level)
		}
		if c.Rate != w.rate {
			t.Fatalf("F%d: tỷ lệ muốn %.2f got %.2f", w.level, w.rate, c.Rate)
		}
		if c.BaseAmount != tier.AmountVnd {
			t.Fatalf("F%d: cơ sở tính muốn %d got %d", w.level, tier.AmountVnd, c.BaseAmount)
		}
		if c.Amount != w.gross || c.TaxPit != w.tax || c.NetAmount != w.net {
			t.Fatalf("F%d: muốn gross=%d thuế=%d net=%d; got gross=%d thuế=%d net=%d",
				w.level, w.gross, w.tax, w.net, c.Amount, c.TaxPit, c.NetAmount)
		}
		t.Logf("✓ F%d: %.0f%% trên %dđ = %dđ (thuế %dđ, thực nhận %dđ)",
			w.level, w.rate*100, c.BaseAmount, c.Amount, c.TaxPit, c.NetAmount)
	}

	// buyer không tự ăn hoa hồng trên khoản đầu tư của chính mình.
	if bc, _ := e.referral.ListByReferrer(e.ctx(), buyer.ID); len(bc) != 0 {
		t.Fatalf("buyer không được có hoa hồng, got %d", len(bc))
	}
}
