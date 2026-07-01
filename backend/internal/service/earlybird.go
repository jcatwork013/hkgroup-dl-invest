package service

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
)

// tierShareMath is the SINGLE SOURCE OF TRUTH for package share math. Given the active offering and a
// package amount (VND), it derives số cổ phần và % sở hữu STRICTLY from the pool:
//
//	giá/cổ phần = valuation_vnd / total_shares          (vd 9.9 tỷ / 990.000 = 10.000đ/cp)
//	số cổ phần  = round(amount_vnd / giá)               (vd 5.000.000 / 10.000 = 500 cp)
//	% sở hữu    = amount_vnd / valuation_vnd * 100       (vd 5.000.000 / 9.9 tỷ = 0.0505%)
//
// Admin chỉ chọn SỐ TIỀN; cổ phần & % luôn tự tính theo pool nên không thể nhập sai.
func tierShareMath(off db.Offering, amountVnd int64) (shares int64, ownershipPct float64) {
	if off.TotalShares <= 0 || off.ValuationVnd <= 0 || amountVnd <= 0 {
		return 0, 0
	}
	price := float64(off.ValuationVnd) / float64(off.TotalShares)
	shares = int64(math.Round(float64(amountVnd) / price))
	ownershipPct = float64(amountVnd) / float64(off.ValuationVnd) * 100
	return shares, ownershipPct
}

// ---- Offering + tier management ----------------------------------------------------------------
//
// The early-bird / whitelist reservation system has been removed. Every package now goes through
// the normal, real investment flow (contract + OTP + VietQR transfer + real share issuance).

// OfferingForUser is the authenticated invest-page payload: the active offering and its tiers.
func (s *InvestmentService) OfferingForUser(ctx context.Context, userID uuid.UUID) (db.Offering, []db.InvestmentTier, error) {
	off, err := s.store.GetActiveOffering(ctx)
	if err != nil {
		return db.Offering{}, nil, err
	}
	tiers, err := s.store.ListTiers(ctx, off.ID)
	return off, tiers, err
}

// ---- Admin: funding rounds (vòng gọi vốn) ------------------------------------------------------

func (s *InvestmentService) ListOfferings(ctx context.Context) ([]db.Offering, error) {
	return s.store.ListOfferings(ctx)
}

// OpenNewRoundInput defines a new funding round opened MANUALLY by an admin (vòng 2, 3, ...).
// Admin nhập định giá + tổng cổ phần + số cổ phần chào bán → admin tự quyết cơ cấu, không có
// logic pha loãng ẩn. Vòng đang mở sẽ tự đóng để chỉ còn 1 vòng active.
type OpenNewRoundInput struct {
	Name          string
	ValuationVnd  int64
	TotalShares   int64
	SharesForSale int64
}

func (s *InvestmentService) OpenNewRound(ctx context.Context, admin uuid.UUID, in OpenNewRoundInput) (db.Offering, error) {
	if in.Name == "" || in.ValuationVnd <= 0 || in.TotalShares <= 0 ||
		in.SharesForSale <= 0 || in.SharesForSale > in.TotalShares {
		return db.Offering{}, errors.Join(ErrValidation, errors.New("thông số vòng không hợp lệ (cần định giá, tổng cổ phần, và 0 < số bán ≤ tổng)"))
	}
	var off db.Offering
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.CloseOpenOfferings(ctx); e != nil {
			return e
		}
		var e error
		off, e = q.CreateOffering(ctx, db.CreateOfferingParams{
			Name:          in.Name,
			ValuationVnd:  in.ValuationVnd,
			TotalShares:   in.TotalShares,
			SharesForSale: in.SharesForSale,
		})
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "offering.open_round", "offering", off.ID.String(), nil, off)
	})
	return off, err
}

// ---- Admin tier management ---------------------------------------------------------------------

func (s *InvestmentService) ListAllTiers(ctx context.Context) ([]db.InvestmentTier, error) {
	off, err := s.store.GetActiveOffering(ctx)
	if err != nil {
		return nil, err
	}
	return s.store.ListAllTiers(ctx, off.ID)
}

type TierInput struct {
	Name         string
	AmountVnd    int64
	Shares       int64
	OwnershipPct float64
	SortOrder    int32
	Active       bool
}

// valid only requires a name + a positive amount; cổ phần và % được server tự tính theo pool.
func (in TierInput) valid() bool {
	return in.Name != "" && in.AmountVnd > 0
}

func (s *InvestmentService) CreateTier(ctx context.Context, admin uuid.UUID, in TierInput) (db.InvestmentTier, error) {
	if !in.valid() {
		return db.InvestmentTier{}, ErrValidation
	}
	off, err := s.store.GetActiveOffering(ctx)
	if err != nil {
		return db.InvestmentTier{}, err
	}
	// Auto theo pool: bỏ qua mọi giá trị shares/% client gửi, tự tính lại từ amount + offering.
	shares, pct := tierShareMath(off, in.AmountVnd)
	if shares <= 0 {
		return db.InvestmentTier{}, errors.Join(ErrValidation, errors.New("số tiền quá nhỏ so với giá một cổ phần"))
	}
	var t db.InvestmentTier
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		t, e = q.CreateTier(ctx, db.CreateTierParams{
			OfferingID:   off.ID,
			Name:         in.Name,
			AmountVnd:    in.AmountVnd,
			Shares:       shares,
			OwnershipPct: pct,
			SortOrder:    in.SortOrder,
			Active:       in.Active,
			IsSpecial:    false,
		})
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "tier.create", "investment_tiers", t.ID.String(), nil, t)
	})
	return t, err
}

func (s *InvestmentService) UpdateTier(ctx context.Context, admin, tierID uuid.UUID, in TierInput) (db.InvestmentTier, error) {
	if !in.valid() {
		return db.InvestmentTier{}, ErrValidation
	}
	off, err := s.store.GetActiveOffering(ctx)
	if err != nil {
		return db.InvestmentTier{}, err
	}
	shares, pct := tierShareMath(off, in.AmountVnd)
	if shares <= 0 {
		return db.InvestmentTier{}, errors.Join(ErrValidation, errors.New("số tiền quá nhỏ so với giá một cổ phần"))
	}
	var t db.InvestmentTier
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		before, e := q.GetTier(ctx, tierID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		t, e = q.UpdateTier(ctx, db.UpdateTierParams{
			ID:           tierID,
			Name:         in.Name,
			AmountVnd:    in.AmountVnd,
			Shares:       shares,
			OwnershipPct: pct,
			SortOrder:    in.SortOrder,
			Active:       in.Active,
			IsSpecial:    false,
		})
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "tier.update", "investment_tiers", t.ID.String(), before, t)
	})
	return t, err
}

// DeleteTier removes a package permanently. Refused if any contract/investment references it —
// dùng SetTierActive (ẩn) cho gói đã phát sinh giao dịch để bảo toàn lịch sử.
func (s *InvestmentService) DeleteTier(ctx context.Context, admin, tierID uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		before, e := q.GetTier(ctx, tierID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		inUse, e := q.TierInUse(ctx, tierID)
		if e != nil {
			return e
		}
		if inUse {
			return errors.Join(ErrConflict, errors.New("gói đã có hợp đồng/đầu tư — hãy ẩn thay vì xoá"))
		}
		if e = q.DeleteTier(ctx, tierID); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "tier.delete", "investment_tiers", tierID.String(), before, nil)
	})
}

func (s *InvestmentService) SetTierActive(ctx context.Context, admin, tierID uuid.UUID, active bool) (db.InvestmentTier, error) {
	var t db.InvestmentTier
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		t, e = q.SetTierActive(ctx, db.SetTierActiveParams{ID: tierID, Active: active})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "tier.set_active", "investment_tiers", t.ID.String(), nil, t)
	})
	return t, err
}
