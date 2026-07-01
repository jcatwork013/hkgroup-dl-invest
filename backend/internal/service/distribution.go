package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

// DistributionService implements the COMPLIANT pool distribution: admin enters period revenue,
// the system computes the shareholder pool (revenue × pool_rate × investor_share_rate) and pays
// it to active shareholders pro-rata via a REAL dividend. Variable, not guaranteed; no target/cap.
type DistributionService struct {
	store    *store.Store
	dividend *DividendService
	settings *SettingsService
}

func NewDistributionService(s *store.Store, div *DividendService, set *SettingsService) *DistributionService {
	return &DistributionService{store: s, dividend: div, settings: set}
}

type DistributionResult struct {
	Distribution db.RevenueDistribution `json:"distribution"`
	Dividend     db.Dividend            `json:"dividend"`
	Payouts      []db.DividendPayout    `json:"payouts"`
}

func (s *DistributionService) Distribute(ctx context.Context, admin uuid.UUID, period string, revenue int64) (DistributionResult, error) {
	if period == "" || revenue <= 0 {
		return DistributionResult{}, ErrValidation
	}
	poolRate := s.settings.Float(ctx, "pool_rate", 0.15)
	// Toàn bộ Pool Cổ Đông (pool_rate của doanh thu) chia thẳng cho nhà đầu tư — KHÔNG nhân
	// investor_share_rate (49%). Nhà đầu tư hưởng 100% pool.
	investorPool := int64(math.Floor(float64(revenue) * poolRate))
	if investorPool <= 0 {
		return DistributionResult{}, ErrValidation
	}

	// Real dividend, distributed pro-rata to active shareholders by shares (= capital).
	div, payouts, err := s.dividend.Declare(ctx, admin, period, investorPool, "Phân bổ doanh thu kỳ "+period)
	if err != nil {
		return DistributionResult{}, err
	}

	rec, err := s.store.CreateRevenueDistribution(ctx, db.CreateRevenueDistributionParams{
		Period:            period,
		TotalRevenue:      revenue,
		PoolRate:          poolRate,
		InvestorShareRate: 1,
		InvestorPool:      investorPool,
		DividendID:        uuid.NullUUID{UUID: div.ID, Valid: true},
		CreatedBy:         admin,
	})
	if err != nil {
		return DistributionResult{}, err
	}
	return DistributionResult{Distribution: rec, Dividend: div, Payouts: payouts}, nil
}

func (s *DistributionService) List(ctx context.Context) ([]db.RevenueDistribution, error) {
	return s.store.ListRevenueDistributions(ctx)
}

// DeleteDistribution xoá tay 1 lần phân bổ doanh thu: gỡ bản ghi revenue_distributions VÀ đợt cổ tức
// gắn với nó (dividend_payouts cascade theo FK 00007). Dùng để dọn dữ liệu test/sai.
func (s *DistributionService) DeleteDistribution(ctx context.Context, admin, distID uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		rec, e := q.GetRevenueDistribution(ctx, distID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		// Xoá bản ghi phân bổ trước (gỡ FK trỏ tới dividend), rồi xoá đợt cổ tức + payouts (cascade).
		if e = q.DeleteRevenueDistribution(ctx, distID); e != nil {
			return e
		}
		if rec.DividendID.Valid {
			if e = q.DeleteDividend(ctx, rec.DividendID.UUID); e != nil {
				return e
			}
		}
		return audit.Write(ctx, q, audit.Actor(admin), "distribution.delete", "revenue_distributions", distID.String(), rec, nil)
	})
}

// Pool returns the capital-raise pool status (target/raised/remaining) for the admin pool view.
type PoolStatus struct {
	ValuationVnd  int64   `json:"valuation_vnd"`
	PoolTargetVnd int64   `json:"pool_target_vnd"` // shares_for_sale × price
	RaisedVnd     int64   `json:"raised_vnd"`      // shares_sold × price
	RemainingVnd  int64   `json:"remaining_vnd"`
	ProgressPct   float64 `json:"progress_pct"`
	SharesSold    int64   `json:"shares_sold"`
	SharesForSale int64   `json:"shares_for_sale"`
	// distribution totals to date
	TotalRevenueVnd   int64   `json:"total_revenue_vnd"`
	TotalInvestorPool int64   `json:"total_investor_pool_vnd"`
	PoolRate          float64 `json:"pool_rate"`
	InvestorShareRate float64 `json:"investor_share_rate"`
}

func (s *DistributionService) Pool(ctx context.Context) (PoolStatus, error) {
	off, err := s.store.GetActiveOffering(ctx)
	if err != nil {
		return PoolStatus{}, err
	}
	price := float64(off.ValuationVnd) / float64(off.TotalShares) // VND/share
	target := int64(float64(off.SharesForSale) * price)
	raised := int64(float64(off.SharesSold) * price)
	sum, err := s.store.SumRevenue(ctx)
	if err != nil {
		return PoolStatus{}, err
	}
	progress := 0.0
	if target > 0 {
		progress = float64(raised) / float64(target) * 100
	}
	return PoolStatus{
		ValuationVnd:      off.ValuationVnd,
		PoolTargetVnd:     target,
		RaisedVnd:         raised,
		RemainingVnd:      target - raised,
		ProgressPct:       progress,
		SharesSold:        off.SharesSold,
		SharesForSale:     off.SharesForSale,
		TotalRevenueVnd:   sum.TotalRevenue,
		TotalInvestorPool: sum.TotalInvestorPool,
		PoolRate:          s.settings.Float(ctx, "pool_rate", 0.15),
		InvestorShareRate: s.settings.Float(ctx, "investor_share_rate", 0.49),
	}, nil
}

// ---- Tiered "đồng chia + bonus" distribution -------------------------------------------------
//
// An ADDITIVE distribution mode on top of the compliant pro-rata Distribute(). The shareholder
// pool (a share of ACTUAL period revenue — variable, not promised) is split in two:
//
//	equal pool  (default 9%): chia đều cho mọi tài khoản active (mỗi shareholder 1 phần bằng nhau)
//	bonus pool  (default 6%): chia HẾT theo TỈ LỆ hạng — mỗi tài khoản nhận bonusPool × (rate hạng /
//	              Σ rate hạng). Hạng theo vốn đã góp: 5–49tr → 1,5% · 50–299tr → 2% · 300–500tr →
//	              2,5% (cấu hình qua settings). VD 2,5%/2%/1,5% trên pool 60k → 25k/20k/15k.
//
// Ràng buộc ADR-0001 vẫn giữ: mọi khoản trả là một dòng dividend_payouts THỰC (đại lượng "tiền
// nhận về" duy nhất), có audit log, công khai (không quỹ ẩn, không cam kết lợi nhuận/target).
//
// Vì bonus chuẩn hoá theo Σ rate nên Σ bonus ≈ pool bonus (chỉ lệch vài đồng do làm tròn floor);
// phần lẻ đó được dồn theo settings.dist_residual_mode ("rollover" mặc định = chia đều, hoặc
// "retain" = giữ lại). Không còn tình huống bonus vượt pool.

type bandDef struct {
	Key   string
	Label string
	Max   int64   // upper bound (exclusive) on invested VND; the last band uses math.MaxInt64
	Rate  float64 // share of the bonus pool granted to each account in this band
}

type tieredConfig struct {
	EqualRate    float64
	BonusRate    float64
	Bands        []bandDef
	ResidualMode string // "rollover" (default) | "retain"
}

func (c tieredConfig) bandFor(investedVnd int64) bandDef {
	for _, b := range c.Bands {
		if investedVnd < b.Max {
			return b
		}
	}
	return c.Bands[len(c.Bands)-1]
}

// PlannedPayout is the computed amount for one active account (before it becomes a real payout).
type PlannedPayout struct {
	UserID      uuid.UUID `json:"user_id"`
	Shares      int64     `json:"shares"`
	InvestedVnd int64     `json:"invested_vnd"`
	Band        string    `json:"band"`       // band key (vd "band3")
	BandLabel   string    `json:"band_label"` // nhãn hạng (vd "300–500tr")
	BandRate    float64   `json:"band_rate"`  // tỉ lệ hạng (vd 0.025)
	EqualShare  int64     `json:"equal_share"`
	Bonus       int64     `json:"bonus"`
	Amount      int64     `json:"amount"`
}

// BandBreakdown is one slice of the bonus pie chart shown to investors.
type BandBreakdown struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Rate       float64 `json:"rate"`
	Accounts   int     `json:"accounts"`
	BonusTotal int64   `json:"bonus_total"`
}

// TieredPlan is the fully-resolved distribution (pure, deterministic — safe to preview).
type TieredPlan struct {
	Period       string          `json:"period"`
	Revenue      int64           `json:"revenue"`
	EqualRate    float64         `json:"equal_rate"`
	BonusRate    float64         `json:"bonus_rate"`
	EqualPool    int64           `json:"equal_pool"`
	BonusPool    int64           `json:"bonus_pool"`
	Accounts     int             `json:"accounts"`
	EqualEach    int64           `json:"equal_each"`
	Bands        []BandBreakdown `json:"bands"`
	ResidualMode string          `json:"residual_mode"`
	Residual     int64           `json:"residual"`    // undistributed (retain mode); 0 on rollover
	Distributed  int64           `json:"distributed"` // == EqualPool+BonusPool on rollover
	Scaled       bool            `json:"scaled"`      // true if the overflow van fired
	Payouts      []PlannedPayout `json:"payouts"`
}

// PlanTiered computes the distribution purely from inputs (no DB) so it can be unit-tested and
// previewed for the pie chart without committing anything. Banding uses each account's TRUE
// invested capital (amount_vnd), so a 50tr package lands on the 50tr boundary exactly.
func PlanTiered(period string, revenue int64, accounts []db.ListActiveAccountsRow, cfg tieredConfig) TieredPlan {
	equalPool := int64(math.Floor(float64(revenue) * cfg.EqualRate))
	bonusPool := int64(math.Floor(float64(revenue) * cfg.BonusRate))
	return planTieredPools(period, revenue, equalPool, bonusPool, accounts, cfg)
}

// PlanTieredFromPool chia một Pool Cổ Đông ĐÃ TÍNH SẴN (vd tổng dividend_pool_vnd 15% của các đơn
// được quét) theo đúng tỉ lệ đồng-chia:bonus (9:6) — KHÔNG cắt 15% thêm lần nữa. equalPool =
// floor(pool × equalRate/(equalRate+bonusRate)); bonusPool = pool − equalPool, nên equalPool+
// bonusPool == pool CHÍNH XÁC (bất biến: chia HẾT pool, không đẻ/mất đồng nào). revenue chỉ để hiển
// thị (doanh thu gốc sinh ra pool).
func PlanTieredFromPool(period string, revenue, pool int64, accounts []db.ListActiveAccountsRow, cfg tieredConfig) TieredPlan {
	denom := cfg.EqualRate + cfg.BonusRate
	var equalPool int64
	if denom > 0 {
		equalPool = int64(math.Floor(float64(pool) * cfg.EqualRate / denom))
	}
	bonusPool := pool - equalPool
	return planTieredPools(period, revenue, equalPool, bonusPool, accounts, cfg)
}

// planTieredPools là lõi chung: nhận thẳng equalPool/bonusPool đã chốt rồi chia theo đồng-chia +
// bonus hạng (Hamilton) + rollover phần dư. Dùng bởi cả PlanTiered (pool = % doanh thu) lẫn
// PlanTieredFromPool (pool cho sẵn).
func planTieredPools(period string, revenue, equalPool, bonusPool int64, accounts []db.ListActiveAccountsRow, cfg tieredConfig) TieredPlan {
	n := len(accounts)

	plan := TieredPlan{
		Period: period, Revenue: revenue,
		EqualRate: cfg.EqualRate, BonusRate: cfg.BonusRate,
		EqualPool: equalPool, BonusPool: bonusPool,
		Accounts: n, ResidualMode: cfg.ResidualMode,
		Bands: bandSkeleton(cfg),
	}
	if n == 0 {
		plan.Residual = equalPool + bonusPool
		return plan
	}

	equalEach := equalPool / int64(n)
	plan.EqualEach = equalEach

	bonus := make([]int64, n)
	bands := make([]bandDef, n)
	// Chia HẾT pool bonus theo TỈ LỆ hạng: mỗi tài khoản nhận bonusPool × (rate hạng / Σ rate hạng
	// của mọi tài khoản). VD 3 tài khoản 2,5%/2%/1,5% (Σ=6%) trên pool 60k → 25k/20k/15k. Toàn bộ
	// pool bonus được phát hết theo tỉ lệ hạng, không còn phần dư bonus phải rollover/giữ lại.
	var sumRates float64
	for i, a := range accounts {
		b := cfg.bandFor(a.InvestedVnd)
		bands[i] = b
		sumRates += b.Rate
	}
	if sumRates > 0 {
		// Chia theo "phần dư lớn nhất" (Hamilton) để Σ bonus == bonusPool CHÍNH XÁC (không thất
		// thoát đồng nào do làm tròn float): floor trước, rồi phát từng đồng lẻ còn lại cho các tài
		// khoản có phần thập phân lớn nhất.
		rem := make([]float64, n)
		var assigned int64
		for i := range accounts {
			exact := float64(bonusPool) * bands[i].Rate / sumRates
			bonus[i] = int64(math.Floor(exact))
			rem[i] = exact - math.Floor(exact)
			assigned += bonus[i]
		}
		order := make([]int, n)
		for i := range order {
			order[i] = i
		}
		sort.SliceStable(order, func(a, b int) bool { return rem[order[a]] > rem[order[b]] })
		for k := int64(0); k < bonusPool-assigned && k < int64(n); k++ {
			bonus[order[k]]++
		}
	}

	payouts := make([]PlannedPayout, n)
	var allocated int64
	for i, a := range accounts {
		amt := equalEach + bonus[i]
		payouts[i] = PlannedPayout{
			UserID: a.UserID, Shares: a.Shares, InvestedVnd: a.InvestedVnd,
			Band: bands[i].Key, BandLabel: bands[i].Label, BandRate: bands[i].Rate,
			EqualShare: equalEach, Bonus: bonus[i], Amount: amt,
		}
		allocated += amt
	}

	// Phần dư = pool chưa phát ra (luôn ≥ 0: equalEach*n ≤ equalPool và Σbonus ≤ bonusPool).
	leftover := equalPool + bonusPool - allocated
	if cfg.ResidualMode != "retain" && leftover > 0 {
		base := leftover / int64(n)
		extra := leftover % int64(n)
		for i := range payouts {
			add := base
			if int64(i) < extra {
				add++
			}
			payouts[i].Amount += add
			payouts[i].EqualShare += add
			allocated += add
		}
		leftover = 0
	}

	// Lát biểu đồ tròn cho 6% bonus, gộp theo hạng.
	for i := range bands {
		for j := range plan.Bands {
			if plan.Bands[j].Key == bands[i].Key {
				plan.Bands[j].Accounts++
				plan.Bands[j].BonusTotal += bonus[i]
			}
		}
	}

	plan.Payouts = payouts
	plan.Distributed = allocated
	plan.Residual = leftover
	return plan
}

func bandSkeleton(cfg tieredConfig) []BandBreakdown {
	out := make([]BandBreakdown, len(cfg.Bands))
	for i, b := range cfg.Bands {
		out[i] = BandBreakdown{Key: b.Key, Label: b.Label, Rate: b.Rate}
	}
	return out
}

func (s *DistributionService) tieredConfig(ctx context.Context) tieredConfig {
	return loadTieredConfig(ctx, s.settings)
}

// loadTieredConfig đọc cấu hình tầng từ settings — dùng chung cho DistributionService lẫn luồng TỰ
// ĐỘNG chia cổ tức trong SalesService.PayOrder (cùng package service).
func loadTieredConfig(ctx context.Context, set *SettingsService) tieredConfig {
	f := func(k string, d float64) float64 { return set.Float(ctx, k, d) }
	return tieredConfig{
		EqualRate: f("dist_equal_rate", 0.09),
		BonusRate: f("dist_bonus_rate", 0.06),
		Bands: []bandDef{
			{Key: "band1", Label: "5–49tr", Max: int64(f("dist_band1_max", 50_000_000)), Rate: f("dist_band1_rate", 0.015)},
			{Key: "band2", Label: "50–299tr", Max: int64(f("dist_band2_max", 300_000_000)), Rate: f("dist_band2_rate", 0.02)},
			{Key: "band3", Label: "300–500tr", Max: math.MaxInt64, Rate: f("dist_band3_rate", 0.025)},
		},
		ResidualMode: set.Str(ctx, "dist_residual_mode", "rollover"),
	}
}

// PreviewTiered computes the plan WITHOUT writing anything — feeds the admin preview & pie chart.
func (s *DistributionService) PreviewTiered(ctx context.Context, period string, revenue int64) (TieredPlan, error) {
	if revenue <= 0 {
		return TieredPlan{}, ErrValidation
	}
	accounts, err := s.store.ListActiveAccounts(ctx)
	if err != nil {
		return TieredPlan{}, err
	}
	return PlanTiered(period, revenue, accounts, s.tieredConfig(ctx)), nil
}

// TieredDistributionResult is the committed outcome of DistributeTiered.
type TieredDistributionResult struct {
	Distribution db.RevenueDistribution `json:"distribution"`
	Dividend     db.Dividend            `json:"dividend"`
	Plan         TieredPlan             `json:"plan"`
	Payouts      []db.DividendPayout    `json:"payouts"`
}

// DistributeTiered commits the 9%+6% plan as a REAL dividend: one dividend_payouts row per active
// account (UNPAID until admin marks paid), a revenue_distributions record, and an audit log — all
// atomic. Mirrors Distribute()'s compliance posture; differs only in the split rule.
func (s *DistributionService) DistributeTiered(ctx context.Context, admin uuid.UUID, period string, revenue int64) (TieredDistributionResult, error) {
	if period == "" || revenue <= 0 {
		return TieredDistributionResult{}, ErrValidation
	}
	cfg := s.tieredConfig(ctx)

	var out TieredDistributionResult
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		accounts, e := q.ListActiveAccounts(ctx)
		if e != nil {
			return e
		}
		plan := PlanTiered(period, revenue, accounts, cfg)
		if plan.Distributed <= 0 {
			return errors.Join(ErrValidation, errors.New("no active accounts / nothing to distribute"))
		}

		div, e := q.CreateDividend(ctx, db.CreateDividendParams{
			DeclaredBy:  admin,
			Period:      period,
			TotalAmount: plan.Distributed,
			Note:        pgText(fmt.Sprintf("Đồng chia %s + bonus %s kỳ %s", pctStr(cfg.EqualRate), pctStr(cfg.BonusRate), period)),
		})
		if e != nil {
			return e
		}

		var payouts []db.DividendPayout
		for _, p := range plan.Payouts {
			if p.Amount <= 0 {
				continue
			}
			po, e := q.CreateDividendPayout(ctx, db.CreateDividendPayoutParams{
				DividendID: div.ID, UserID: p.UserID, Shares: p.Shares, Amount: p.Amount,
				EqualShare: p.EqualShare, Bonus: p.Bonus, Band: p.BandLabel,
				BandRate: p.BandRate, InvestedVnd: p.InvestedVnd,
			})
			if e != nil {
				return e
			}
			payouts = append(payouts, po)
		}

		rec, e := q.CreateRevenueDistribution(ctx, db.CreateRevenueDistributionParams{
			Period:            period,
			TotalRevenue:      revenue,
			PoolRate:          cfg.EqualRate + cfg.BonusRate,
			InvestorShareRate: 1,
			InvestorPool:      plan.Distributed,
			DividendID:        uuid.NullUUID{UUID: div.ID, Valid: true},
			CreatedBy:         admin,
		})
		if e != nil {
			return e
		}

		out = TieredDistributionResult{Distribution: rec, Dividend: div, Plan: plan, Payouts: payouts}
		return audit.Write(ctx, q, audit.Actor(admin), "distribution.tiered", "dividends", div.ID.String(), nil, plan)
	})
	if err != nil {
		return TieredDistributionResult{}, err
	}
	return out, nil
}

func pctStr(rate float64) string { return fmt.Sprintf("%g%%", rate*100) }

// ---- Quét cổ tức từ đơn hàng ---------------------------------------------------------------
//
// Mỗi đơn khi paid đã trích sẵn 15% (dividend_pool_vnd) làm Pool Cổ Đông trong sales_distributions
// (xem splitOrder). SweepDividend GOM khoản đó của mọi đơn paid CHƯA swept, chia HẾT cho cổ đông
// theo tầng 9%+6% (PlanTieredFromPool), rồi đánh dấu swept=true. Bất biến & an toàn:
//   - Idempotent: cột swept + khoá FOR UPDATE ⇒ một đơn KHÔNG bao giờ chia cổ tức hai lần.
//   - Σ payouts == pool (rollover) ⇒ không thất thoát/đẻ đồng nào.
//   - Là dividend THỰC (dividend_payouts UNPAID) — giữ posture "biến thiên theo doanh thu thật,
//     admin chủ động bấm quét, không cron, không cam kết".

// ErrNothingToSweep: không có đơn paid nào chưa gộp — không tạo đợt cổ tức rỗng.
var ErrNothingToSweep = errors.New("không có đơn thành công nào chưa gộp cổ tức")

// SweepResult là kết quả một lần quét.
type SweepResult struct {
	SweptOrders  int                    `json:"swept_orders"`
	RevenueVnd   int64                  `json:"revenue_vnd"` // Σ subtotal các đơn được quét
	PoolVnd      int64                  `json:"pool_vnd"`    // Σ 15% pool cổ đông đã trích
	Distribution db.RevenueDistribution `json:"distribution"`
	Dividend     db.Dividend            `json:"dividend"`
	Plan         TieredPlan             `json:"plan"`
	Payouts      []db.DividendPayout    `json:"payouts"`
}

// SweepPreview đọc (KHÔNG khoá, KHÔNG ghi) số sẽ quét + kế hoạch chia, cho admin xem trước.
func (s *DistributionService) SweepPreview(ctx context.Context) (SweepResult, error) {
	sum, err := s.store.SumUnsweptPaidPool(ctx)
	if err != nil {
		return SweepResult{}, err
	}
	out := SweepResult{SweptOrders: int(sum.Orders), RevenueVnd: sum.Revenue, PoolVnd: sum.Pool}
	if sum.Pool <= 0 {
		return out, nil
	}
	accounts, err := s.store.ListActiveAccounts(ctx)
	if err != nil {
		return SweepResult{}, err
	}
	out.Plan = PlanTieredFromPool("(preview)", sum.Revenue, sum.Pool, accounts, s.tieredConfig(ctx))
	return out, nil
}

// ErrNoActiveShareholders: có pool nhưng chưa có cổ đông active để chia. Luồng tự động (PayOrder)
// coi đây là "để đơn unswept, chia sau" — KHÔNG huỷ thanh toán.
var ErrNoActiveShareholders = errors.New("chưa có cổ đông active để chia cổ tức")

// SweepDividend (admin bấm) gom pool 15% của mọi đơn paid chưa swept thành MỘT đợt cổ tức thực,
// trong 1 giao dịch. Lần chạy đầu tự cuốn cả đơn CŨ (backfill). Rỗng ⇒ ErrNothingToSweep.
func (s *DistributionService) SweepDividend(ctx context.Context, admin uuid.UUID) (SweepResult, error) {
	cfg := s.tieredConfig(ctx)
	var out SweepResult
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		res, e := sweepUnsweptTx(ctx, q, admin, cfg, false)
		if e != nil {
			return e
		}
		out = res
		return nil
	})
	if err != nil {
		return SweepResult{}, err
	}
	return out, nil
}

// sweepUnsweptTx là LÕI quét cổ tức, chạy TRONG một tx cho sẵn (q). Gom pool 15% của mọi đơn paid
// chưa swept → 1 đợt cổ tức tiered, đánh dấu swept. autoPay=true ⇒ set paid_at ngay (tiền vào thẳng
// ví cổ đông, không cần admin duyệt) — dùng cho luồng tự động trong PayOrder.
//
// Bất biến giữ nguyên: Σ payouts == pool (rollover); idempotent qua swept + FOR UPDATE. Trả
// ErrNothingToSweep nếu không có đơn/pool; ErrNoActiveShareholders nếu có pool nhưng chưa cổ đông.
func sweepUnsweptTx(ctx context.Context, q *db.Queries, admin uuid.UUID, cfg tieredConfig, autoPay bool) (SweepResult, error) {
	rows, err := q.ListUnsweptPaidDistributions(ctx) // khoá FOR UPDATE
	if err != nil {
		return SweepResult{}, err
	}
	if len(rows) == 0 {
		return SweepResult{}, ErrNothingToSweep
	}
	var revenue, pool int64
	for _, d := range rows {
		revenue += d.TotalVnd
		pool += d.DividendPoolVnd
	}
	if pool <= 0 {
		return SweepResult{}, ErrNothingToSweep
	}

	accounts, err := q.ListActiveAccounts(ctx)
	if err != nil {
		return SweepResult{}, err
	}
	period := "QCT-" + time.Now().Format("20060102-150405")
	plan := PlanTieredFromPool(period, revenue, pool, accounts, cfg)
	if plan.Distributed <= 0 {
		return SweepResult{}, ErrNoActiveShareholders
	}

	mode := "Quét cổ tức"
	action := "dividend.sweep"
	if autoPay {
		mode = "Tự động chia cổ tức"
		action = "dividend.auto"
	}
	div, err := q.CreateDividend(ctx, db.CreateDividendParams{
		DeclaredBy:  admin,
		Period:      period,
		TotalAmount: plan.Distributed,
		Note:        pgText(fmt.Sprintf("%s %d đơn — pool cổ đông %d₫ (đồng chia %s + bonus %s)", mode, len(rows), pool, pctStr(cfg.EqualRate), pctStr(cfg.BonusRate))),
	})
	if err != nil {
		return SweepResult{}, err
	}

	var payouts []db.DividendPayout
	for _, p := range plan.Payouts {
		if p.Amount <= 0 {
			continue
		}
		po, err := q.CreateDividendPayout(ctx, db.CreateDividendPayoutParams{
			DividendID: div.ID, UserID: p.UserID, Shares: p.Shares, Amount: p.Amount,
			EqualShare: p.EqualShare, Bonus: p.Bonus, Band: p.BandLabel,
			BandRate: p.BandRate, InvestedVnd: p.InvestedVnd,
		})
		if err != nil {
			return SweepResult{}, err
		}
		if autoPay {
			if _, err := q.MarkPayoutPaid(ctx, po.ID); err != nil {
				return SweepResult{}, err
			}
		}
		payouts = append(payouts, po)
	}

	rec, err := q.CreateRevenueDistribution(ctx, db.CreateRevenueDistributionParams{
		Period:            period,
		TotalRevenue:      revenue,
		PoolRate:          cfg.EqualRate + cfg.BonusRate,
		InvestorShareRate: 1,
		InvestorPool:      plan.Distributed,
		DividendID:        uuid.NullUUID{UUID: div.ID, Valid: true},
		CreatedBy:         admin,
	})
	if err != nil {
		return SweepResult{}, err
	}

	// Đánh dấu swept — chốt idempotent. Mỗi đơn phải đổi đúng 1 dòng (đã khoá ở trên).
	for _, d := range rows {
		n, err := q.MarkSalesDistributionSwept(ctx, d.OrderID)
		if err != nil {
			return SweepResult{}, err
		}
		if n != 1 {
			return SweepResult{}, fmt.Errorf("đơn %s không đánh dấu swept được (đã swept?) — huỷ giao dịch", d.OrderID)
		}
	}

	out := SweepResult{
		SweptOrders: len(rows), RevenueVnd: revenue, PoolVnd: pool,
		Distribution: rec, Dividend: div, Plan: plan, Payouts: payouts,
	}
	if err := audit.Write(ctx, q, audit.Actor(admin), action, "dividends", div.ID.String(), nil, out); err != nil {
		return SweepResult{}, err
	}
	return out, nil
}
