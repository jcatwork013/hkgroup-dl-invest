package service

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

// ReferralService attributes referrals (1-level storage) and pays MULTI-LEVEL commission (F1/F2/F3)
// by walking the direct-referrer chain upward at commission time. Storage stays 1-level.
type ReferralService struct {
	store *store.Store

	commissionRate float64 // fallback customer commission rate, e.g. 0.05
	pitRate        float64 // thuế TNCN withholding, e.g. 0.10
	// recordInvestorReferrals: whether to even persist an investor referral row.
	recordInvestorReferrals bool
	settings                *SettingsService // F1 rate + investor-cash toggle (1 level only)
}

func NewReferralService(s *store.Store, commissionRate, pitRate float64, recordInvestor bool) *ReferralService {
	return &ReferralService{store: s, commissionRate: commissionRate, pitRate: pitRate, recordInvestorReferrals: recordInvestor}
}

// SetSettings enables admin-configured F1 (1-level) investor referral commission.
func (s *ReferralService) SetSettings(svc *SettingsService) { s.settings = svc }

// f1Config reads the admin-configured F1 rate + on/off toggle for investor referral commission.
func (s *ReferralService) f1Config(ctx context.Context) (float64, bool) {
	if s.settings == nil {
		return 0, false
	}
	m, err := s.settings.List(ctx)
	if err != nil || m["referral_investor_cash"] != "on" {
		return 0, false
	}
	f, _ := strconv.ParseFloat(m["referral_f1_rate"], 64)
	return f, f > 0
}

// Attribute records a 1-level referral: referee -> direct referrer. No upline beyond this.
func (s *ReferralService) Attribute(ctx context.Context, referrerCode string, refereeID uuid.UUID, referralType string) error {
	referrer, err := s.store.GetUserByReferralCode(ctx, referrerCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // unknown code: silently ignore
	}
	if err != nil {
		return err
	}
	if referrer.ID == refereeID {
		return nil // cannot refer yourself
	}

	rtype := db.ReferralTypeInvestor
	if referralType == string(db.ReferralTypeCustomer) {
		rtype = db.ReferralTypeCustomer
	}
	if rtype == db.ReferralTypeInvestor {
		// Record investor referral if F1 is enabled (admin settings) or the env flag is on.
		if _, on := s.f1Config(ctx); !on && !s.recordInvestorReferrals {
			return nil
		}
	}

	_, err = s.store.CreateReferral(ctx, db.CreateReferralParams{
		RefereeID:    refereeID,
		ReferrerID:   referrer.ID,
		ReferralType: rtype,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // ON CONFLICT DO NOTHING — referee already attributed
	}
	return err
}

// generateCommission is called INSIDE the approve+issue transaction (q is tx-bound).
// INVARIANT 5: commission only on approved investments. MULTI-LEVEL (F1/F2/F3): we walk UP the
// 1-level `referrals` chain (direct referrer -> their referrer -> ...) up to 3 levels and credit a
// commission to each upline earner. The referrals table stays 1-level; the upline is derived here.
func (s *ReferralService) generateCommission(ctx context.Context, q *db.Queries, inv db.Investment, admin uuid.UUID) error {
	// The chain ORIGIN referral (the investor's own direct referral) decides whether the WHOLE chain
	// is paid. Gating the whole chain — not just F1 — keeps F1/F2/F3 consistent: an investor-referral
	// chain pays NOTHING when admin has cash commission OFF, instead of skipping F1 but still paying
	// F2/F3 (the old inconsistency).
	origin, err := q.GetReferralByReferee(ctx, inv.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // investor was not referred — no upline to pay
	}
	if err != nil {
		return err
	}
	if origin.ReferralType == db.ReferralTypeInvestor {
		if _, on := s.f1Config(ctx); !on {
			return nil // investor-referral cash disabled => no F1/F2/F3 at all
		}
	}

	// Cycle / self-credit guard: never credit the investor themselves or anyone already in the chain
	// (the 1-level table forbids direct self-referral but NOT a 2-cycle A->B->A, which could otherwise
	// pay the investor their own F2). Seeding with the investor blocks any such self-payout.
	seen := map[uuid.UUID]bool{inv.UserID: true}
	current := inv.UserID // referee whose referrer earns at this level
	for level := 1; level <= 3; level++ {
		ref, err := q.GetReferralByReferee(ctx, current)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // chain ended — no more upline to pay
		}
		if err != nil {
			return err
		}
		beneficiary := ref.ReferrerID
		if seen[beneficiary] {
			return nil // cycle detected — climbing further only revisits seen earners
		}
		seen[beneficiary] = true

		if rate, ok := s.levelRate(ctx, q, level, origin.ReferralType, inv.TierID); ok && rate > 0 {
			if err := s.createLevelCommission(ctx, q, inv, beneficiary, level, rate, admin); err != nil {
				return err
			}
		}
		current = beneficiary // climb one level up
	}
	return nil
}

// levelRate resolves the commission rate for an upline level given the chain ORIGIN type. Level 1
// (F1): customer = per-tier rate, investor = admin-gated F1. Levels 2-3 (F2/F3) use the configured
// flat rates. The investor-cash toggle is enforced once at the chain level in generateCommission, so
// by the time we resolve F2/F3 the chain has already passed that gate.
func (s *ReferralService) levelRate(ctx context.Context, q *db.Queries, level int, rtype db.ReferralType, tierID uuid.UUID) (float64, bool) {
	switch level {
	case 1:
		switch rtype {
		case db.ReferralTypeCustomer:
			rate := s.commissionRate // per-tier rate (set per package), fallback to the global rate.
			if tier, e := q.GetTier(ctx, tierID); e == nil && tier.CommissionRate > 0 {
				rate = tier.CommissionRate
			}
			return rate, true
		case db.ReferralTypeInvestor:
			f1, on := s.f1Config(ctx) // investor F1 only if admin enabled it.
			if !on || f1 <= 0 {
				return 0, false
			}
			return f1, true
		}
		return 0, false
	case 2:
		if s.settings == nil {
			return 0, false
		}
		return s.settings.Float(ctx, "referral_f2_rate", 0), true
	case 3:
		if s.settings == nil {
			return 0, false
		}
		return s.settings.Float(ctx, "referral_f3_rate", 0), true
	}
	return 0, false
}

// createLevelCommission writes one (investment, level) commission row crediting beneficiary.
func (s *ReferralService) createLevelCommission(ctx context.Context, q *db.Queries, inv db.Investment, beneficiary uuid.UUID, level int, rate float64, admin uuid.UUID) error {
	base := inv.AmountVnd
	amount := int64(math.Round(float64(base) * rate))
	if amount <= 0 {
		return nil
	}
	tax := int64(math.Round(float64(amount) * s.pitRate))
	net := amount - tax

	comm, err := q.CreateCommission(ctx, db.CreateCommissionParams{
		ReferralID:    inv.UserID, // chain origin (investor's referral row) — drives type lookups
		InvestmentID:  inv.ID,
		BaseAmount:    base,
		Rate:          rate,
		Amount:        amount,
		TaxPit:        tax,
		NetAmount:     net,
		Level:         int16(level),
		BeneficiaryID: beneficiary,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // already generated for this (investment, level)
	}
	if err != nil {
		return err
	}
	return audit.Write(ctx, q, audit.Actor(admin), "commission.generate", "commissions", comm.ID.String(), nil, comm)
}

// ----- Admin commission management -----

func (s *ReferralService) ApproveCommission(ctx context.Context, admin, id uuid.UUID) (db.Commission, error) {
	var c db.Commission
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		c, e = q.ApproveCommission(ctx, db.ApproveCommissionParams{ID: id, ApprovedBy: uuid.NullUUID{UUID: admin, Valid: true}})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidState
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "commission.approve", "commissions", id.String(), nil, c)
	})
	return c, err
}

func (s *ReferralService) PayCommission(ctx context.Context, admin, id uuid.UUID) (db.Commission, error) {
	var c db.Commission
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		c, e = q.PayCommission(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidState
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "commission.pay", "commissions", id.String(), nil, c)
	})
	return c, err
}

func (s *ReferralService) ListByReferrer(ctx context.Context, referrer uuid.UUID) ([]db.Commission, error) {
	return s.store.ListCommissionsByReferrer(ctx, referrer)
}

func (s *ReferralService) ListReferralsByReferrer(ctx context.Context, referrer uuid.UUID) ([]db.Referral, error) {
	return s.store.ListReferralsByReferrer(ctx, referrer)
}
