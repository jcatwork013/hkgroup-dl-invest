package service

import (
	"math"

	"github.com/hkgroup/backend/internal/db"
)

// TieredCfgForTest is a minimal knob set for white-box tests of the tiered planner.
type TieredCfgForTest struct {
	ResidualMode string
}

// PlanTieredForTest runs PlanTiered with the default rates/bands and the given residual mode,
// so distribution_tiered_test.go (package service_test) can exercise the pure planner without a DB.
func PlanTieredForTest(period string, revenue int64, accounts []db.ListActiveAccountsRow, c TieredCfgForTest) TieredPlan {
	cfg := tieredConfig{
		EqualRate: 0.09,
		BonusRate: 0.06,
		Bands: []bandDef{
			{Key: "band1", Label: "5–49tr", Max: 50_000_000, Rate: 0.015},
			{Key: "band2", Label: "50–299tr", Max: 300_000_000, Rate: 0.02},
			{Key: "band3", Label: "300–500tr", Max: math.MaxInt64, Rate: 0.025},
		},
		ResidualMode: c.ResidualMode,
	}
	return PlanTiered(period, revenue, accounts, cfg)
}

// PlanTieredFromPoolForTest chia một pool cho sẵn (không cắt 15% lần nữa) — để test bất biến
// Σ payouts == pool của luồng Quét cổ tức.
func PlanTieredFromPoolForTest(period string, revenue, pool int64, accounts []db.ListActiveAccountsRow, c TieredCfgForTest) TieredPlan {
	cfg := tieredConfig{
		EqualRate: 0.09,
		BonusRate: 0.06,
		Bands: []bandDef{
			{Key: "band1", Label: "5–49tr", Max: 50_000_000, Rate: 0.015},
			{Key: "band2", Label: "50–299tr", Max: 300_000_000, Rate: 0.02},
			{Key: "band3", Label: "300–500tr", Max: math.MaxInt64, Rate: 0.025},
		},
		ResidualMode: c.ResidualMode,
	}
	return PlanTieredFromPool(period, revenue, pool, accounts, cfg)
}
