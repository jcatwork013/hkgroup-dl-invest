package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

type DashboardService struct {
	store *store.Store
}

func NewDashboardService(s *store.Store) *DashboardService { return &DashboardService{store: s} }

// InvestorDashboard: capital contributed · shares · ownership % · REAL dividends received.
// There is deliberately NO projected/target return field anywhere in this payload.
type InvestorDashboard struct {
	CapitalContributed int64   `json:"capital_contributed_vnd"` // sum of APPROVED investments
	Shares             int64   `json:"shares"`
	OwnershipPct       float64 `json:"ownership_pct"`
	DividendReceived   int64   `json:"dividend_received_vnd"` // real paid dividends only
}

func (s *DashboardService) Investor(ctx context.Context, userID uuid.UUID) (InvestorDashboard, error) {
	var d InvestorDashboard

	holding, err := s.store.GetShareholding(ctx, userID)
	if err == nil {
		d.Shares = holding.Shares
		d.OwnershipPct = holding.OwnershipPct
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return d, err
	}

	invs, err := s.store.ListInvestmentsByUser(ctx, userID)
	if err != nil {
		return d, err
	}
	for _, inv := range invs {
		if inv.Status == db.InvestmentStatusApproved {
			d.CapitalContributed += inv.AmountVnd
		}
	}

	div, err := s.store.SumDividendPaidToUser(ctx, userID)
	if err != nil {
		return d, err
	}
	d.DividendReceived = div
	return d, nil
}

// ----- Admin -----

type AdminDashboard struct {
	Stats             db.AdminStatsRow `json:"stats"`
	CustomerCommGross int64            `json:"customer_commission_gross_vnd"`
	CustomerCommTax   int64            `json:"customer_commission_tax_vnd"`
	CustomerCommNet   int64            `json:"customer_commission_net_vnd"`
	InvestorCommGross int64            `json:"investor_commission_gross_vnd"` // expected 0 (record-only)
}

func (s *DashboardService) Admin(ctx context.Context) (AdminDashboard, error) {
	var d AdminDashboard
	stats, err := s.store.AdminStats(ctx)
	if err != nil {
		return d, err
	}
	d.Stats = stats

	cust, err := s.store.SumCommissionsByType(ctx, db.ReferralTypeCustomer)
	if err != nil {
		return d, err
	}
	d.CustomerCommGross, d.CustomerCommTax, d.CustomerCommNet = cust.Gross, cust.Tax, cust.Net

	inv, err := s.store.SumCommissionsByType(ctx, db.ReferralTypeInvestor)
	if err != nil {
		return d, err
	}
	d.InvestorCommGross = inv.Gross
	return d, nil
}

func (s *DashboardService) CapTable(ctx context.Context) ([]db.CapTableRow, error) {
	return s.store.CapTable(ctx)
}

func (s *DashboardService) AuditLogs(ctx context.Context, limit, offset int32) ([]db.AuditLog, error) {
	return s.store.ListAuditLogs(ctx, db.ListAuditLogsParams{Limit: limit, Offset: offset})
}

// IntegrityCheck reconciles INVARIANT 4 for every shareholder: shareholdings.shares must equal
// SUM(share_ledger.shares_delta). Returns the list of users that fail (empty => healthy).
type IntegrityMismatch struct {
	UserID        uuid.UUID `json:"user_id"`
	HoldingShares int64     `json:"holding_shares"`
	LedgerShares  int64     `json:"ledger_shares"`
}

func (s *DashboardService) IntegrityCheck(ctx context.Context) ([]IntegrityMismatch, error) {
	holdings, err := s.store.ListAllShareholdings(ctx)
	if err != nil {
		return nil, err
	}
	var mismatches []IntegrityMismatch
	for _, h := range holdings {
		ledger, err := s.store.SumLedgerByUser(ctx, h.UserID)
		if err != nil {
			return nil, err
		}
		if ledger != h.Shares {
			mismatches = append(mismatches, IntegrityMismatch{UserID: h.UserID, HoldingShares: h.Shares, LedgerShares: ledger})
		}
	}
	return mismatches, nil
}
