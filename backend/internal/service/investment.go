package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/events"
	"github.com/hkgroup/backend/internal/platform/idgen"
	"github.com/hkgroup/backend/internal/platform/otp"
	"github.com/hkgroup/backend/internal/store"
)

// CompanyBank holds the legal-entity account that receives ALL funds (HARD CONSTRAINT 4).
type CompanyBank struct {
	Bank        string
	Account     string
	AccountName string
}

type InvestmentService struct {
	store    *store.Store
	otp      *otp.Service
	referral *ReferralService
	bank     CompanyBank
	settings *SettingsService
	events   events.Publisher
}

func NewInvestmentService(s *store.Store, otpSvc *otp.Service, ref *ReferralService, bank CompanyBank) *InvestmentService {
	return &InvestmentService{store: s, otp: otpSvc, referral: ref, bank: bank, events: events.Noop{}}
}

// SetSettings lets payment use the admin-configured company account (fallback: env CompanyBank).
func (s *InvestmentService) SetSettings(svc *SettingsService) { s.settings = svc }

// companyBank resolves the company receiving account (settings override env). Always a company
// (legal-entity) account — HARD CONSTRAINT 4.
func (s *InvestmentService) companyBank(ctx context.Context) CompanyBank {
	cb := s.bank
	if s.settings != nil {
		if bn, acc, an, ok := s.settings.CompanyBank(ctx); ok {
			cb = CompanyBank{Bank: bn, Account: acc, AccountName: an}
		}
	}
	return cb
}

// SetEvents installs a domain-event publisher (nil-safe: defaults to Noop).
func (s *InvestmentService) SetEvents(p events.Publisher) {
	if p != nil {
		s.events = p
	}
}

// ----- Read endpoints -----

func (s *InvestmentService) Offering(ctx context.Context) (db.Offering, []db.InvestmentTier, error) {
	off, err := s.store.GetActiveOffering(ctx)
	if err != nil {
		return db.Offering{}, nil, err
	}
	tiers, err := s.store.ListTiers(ctx, off.ID)
	return off, tiers, err
}

// ----- Step: start a contract (choose tier) + issue signing OTP -----

type ContractStart struct {
	Contract db.Contract `json:"contract"`
	OTPRef   string      `json:"otp_ref"`
	OTPCode  string      `json:"otp_code,omitempty"` // dev convenience; delivered by SMS in prod
}

func (s *InvestmentService) StartContract(ctx context.Context, userID, tierID uuid.UUID) (ContractStart, error) {
	// KYC đã gỡ bỏ (luật mới không cho thu thập eKYC) — không còn chặn theo kyc_status.
	// Khóa khi vòng gọi vốn đã bán hết — chờ admin mở vòng mới.
	if off, e := s.store.GetActiveOffering(ctx); e == nil && off.SharesSold >= off.SharesForSale {
		return ContractStart{}, ErrPoolExhausted
	}
	tier, err := s.store.GetTier(ctx, tierID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContractStart{}, ErrNotFound
	}
	if err != nil {
		return ContractStart{}, err
	}

	contract, err := s.store.CreateContract(ctx, db.CreateContractParams{UserID: userID, TierID: tier.ID})
	if err != nil {
		return ContractStart{}, err
	}
	ch, err := s.otp.Issue(ctx, "contract", contract.ID.String())
	if err != nil {
		return ContractStart{}, err
	}
	return ContractStart{Contract: contract, OTPRef: ch.Ref, OTPCode: ch.Code}, nil
}

// ----- Step: sign contract with OTP + create the (pending) investment & payment instruction -----

type InvestmentCreated struct {
	Investment db.Investment `json:"investment"`
	Payment    db.Payment    `json:"payment"`
}

type SignInput struct {
	UserID         uuid.UUID
	ContractID     uuid.UUID
	OTPRef         string
	OTPCode        string
	IdempotencyKey string
}

func (s *InvestmentService) SignAndCreateInvestment(ctx context.Context, in SignInput) (InvestmentCreated, error) {
	// Idempotency: a repeated submit with the same key returns the original investment.
	if in.IdempotencyKey != "" {
		if existing, err := s.store.GetInvestmentByIdempotencyKey(ctx, pgText(in.IdempotencyKey)); err == nil {
			pay, _ := s.store.GetPaymentByInvestment(ctx, existing.ID)
			return InvestmentCreated{Investment: existing, Payment: pay}, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return InvestmentCreated{}, err
		}
	}

	ok, err := s.otp.Verify(ctx, "contract", in.ContractID.String(), in.OTPRef, in.OTPCode)
	if err != nil {
		return InvestmentCreated{}, err
	}
	if !ok {
		return InvestmentCreated{}, ErrOTPInvalid
	}

	bank := s.companyBank(ctx)
	var result InvestmentCreated
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		contract, e := q.GetContract(ctx, in.ContractID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if contract.UserID != in.UserID {
			return ErrForbidden
		}
		tier, e := q.GetTier(ctx, contract.TierID)
		if e != nil {
			return e
		}

		pdfURL := fmt.Sprintf("/contracts/%s.pdf", contract.ID)
		signed, e := q.SignContract(ctx, db.SignContractParams{
			ID: contract.ID, SignatureOtpRef: pgText(in.OTPRef), PdfUrl: pgText(pdfURL),
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidState // already signed / not draft
		}
		if e != nil {
			return e
		}

		code := idgen.InvestmentCode()
		inv, e := q.CreateInvestment(ctx, db.CreateInvestmentParams{
			Code:           code,
			UserID:         in.UserID,
			ContractID:     signed.ID,
			TierID:         tier.ID,
			AmountVnd:      tier.AmountVnd,
			Shares:         tier.Shares,
			IdempotencyKey: pgText(in.IdempotencyKey),
		})
		if e != nil {
			if isUniqueViolation(e) {
				return ErrConflict
			}
			return e
		}

		// HARD CONSTRAINT 4: funds always go to the company (legal-entity) account.
		// transfer_note = the investment code so reconciliation can match the memo.
		pay, e := q.CreatePayment(ctx, db.CreatePaymentParams{
			InvestmentID:       inv.ID,
			Bank:               bank.Bank,
			CompanyAccount:     bank.Account,
			CompanyAccountName: bank.AccountName,
			AmountVnd:          inv.AmountVnd,
			TransferNote:       inv.Code,
		})
		if e != nil {
			return e
		}

		result = InvestmentCreated{Investment: inv, Payment: pay}
		return audit.Write(ctx, q, audit.Actor(in.UserID), "investment.create", "investments", inv.ID.String(), nil, inv)
	})
	return result, err
}

// ----- Step: investor declares "I have transferred" (idempotent) -----

func (s *InvestmentService) DeclareTransfer(ctx context.Context, userID, investmentID uuid.UUID) (db.Payment, error) {
	inv, err := s.store.GetInvestment(ctx, investmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Payment{}, ErrNotFound
	}
	if err != nil {
		return db.Payment{}, err
	}
	if inv.UserID != userID {
		return db.Payment{}, ErrForbidden
	}
	pay, err := s.store.DeclarePaymentTransferred(ctx, investmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		// already declared — idempotent: return current payment
		return s.store.GetPaymentByInvestment(ctx, investmentID)
	}
	return pay, err
}

// ----- Admin: reconcile (3-layer check) -----

// Reconcile verifies the layers (funds arrived matching amount+note / contract signed), then marks
// the payment + investment reconciled. INVARIANT 7: a money entry requires a
// reconciled payment; this is where that reconciliation happens. Idempotent.
func (s *InvestmentService) Reconcile(ctx context.Context, admin, investmentID uuid.UUID) (db.Investment, error) {
	var result db.Investment
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		inv, e := q.GetInvestmentForUpdate(ctx, investmentID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if inv.Status == db.InvestmentStatusReconciled {
			result = inv
			return nil // idempotent
		}
		if inv.Status != db.InvestmentStatusPending {
			return ErrInvalidState
		}

		// Layer 1 (KYC) đã gỡ bỏ — luật mới không cho thu thập eKYC nên không còn đối soát kyc_status.

		// Layer 3: contract signed.
		contract, e := q.GetContract(ctx, inv.ContractID)
		if e != nil {
			return e
		}
		if contract.Status != db.ContractStatusSigned {
			return ErrInvalidState
		}

		// Layer 2: funds arrived (declared) with the exact amount.
		pay, e := q.GetPaymentByInvestment(ctx, investmentID)
		if e != nil {
			return e
		}
		if !pay.DeclaredAt.Valid {
			return errors.Join(ErrInvalidState, errors.New("investor has not declared transfer"))
		}
		if pay.AmountVnd != inv.AmountVnd {
			return errors.Join(ErrValidation, errors.New("payment amount mismatch"))
		}

		if _, e = q.ReconcilePayment(ctx, db.ReconcilePaymentParams{
			InvestmentID: investmentID, ReconciledBy: uuid.NullUUID{UUID: admin, Valid: true},
		}); e != nil {
			return e
		}
		result, e = q.MarkInvestmentReconciled(ctx, db.MarkInvestmentReconciledParams{
			ID: investmentID, ReconciledBy: uuid.NullUUID{UUID: admin, Valid: true},
		})
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "investment.reconcile", "investments", investmentID.String(), inv, result)
	})
	return result, err
}

// ----- Admin: approve + issue shares (the atomic core) -----

// ApproveAndIssueShares performs INVARIANTS 2,3,5,7,8 atomically:
//   - status reconciled -> approved (actor + ts)
//   - append share_ledger (+shares)  [append-only]
//   - upsert shareholdings read model
//   - bump offering.shares_sold (DB CHECK guards INVARIANT 1)
//   - generate customer commission (INVARIANT 5)
//   - write audit logs
//
// All in ONE transaction. Idempotent: a second call on an approved investment is a no-op.
func (s *InvestmentService) ApproveAndIssueShares(ctx context.Context, admin, investmentID uuid.UUID) (db.Investment, error) {
	var result db.Investment
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		inv, e := q.GetInvestmentForUpdate(ctx, investmentID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if inv.Status == db.InvestmentStatusApproved {
			result = inv
			return nil // idempotent — shares already issued
		}
		if inv.Status != db.InvestmentStatusReconciled {
			return ErrInvalidState
		}

		// INVARIANT 7: never issue without a reconciled payment.
		pay, e := q.GetPaymentByInvestment(ctx, investmentID)
		if e != nil {
			return e
		}
		if !pay.ReconciledAt.Valid {
			return errors.Join(ErrInvalidState, errors.New("payment not reconciled"))
		}

		// Lock the offering row and check capacity (DB CHECK is the hard backstop = INVARIANT 1).
		active, e := q.GetActiveOffering(ctx)
		if e != nil {
			return e
		}
		off, e := q.GetOfferingForUpdate(ctx, active.ID)
		if e != nil {
			return e
		}
		if off.SharesSold+inv.Shares > off.SharesForSale {
			return ErrPoolExhausted
		}

		approved, e := q.MarkInvestmentApproved(ctx, db.MarkInvestmentApprovedParams{
			ID: investmentID, ApprovedBy: uuid.NullUUID{UUID: admin, Valid: true},
		})
		if e != nil {
			return e
		}

		// Bump shares_sold (CHECK offering_sold_within_pool fails the tx if it would overflow).
		if _, e = q.AddSharesSold(ctx, db.AddSharesSoldParams{ID: off.ID, SharesSold: inv.Shares}); e != nil {
			if isCheckViolation(e) {
				return ErrPoolExhausted
			}
			return e
		}

		// Append-only issuance entry (unique per investment => idempotent at the DB layer too).
		ledger, e := q.InsertShareLedger(ctx, db.InsertShareLedgerParams{
			UserID:       inv.UserID,
			InvestmentID: uuid.NullUUID{UUID: inv.ID, Valid: true},
			SharesDelta:  inv.Shares,
			Reason:       "issue:approved-investment",
		})
		if e != nil {
			if isUniqueViolation(e) {
				return ErrConflict
			}
			return e
		}

		// Update read model. ownership_pct = total user shares / offering.total_shares * 100.
		var prior int64
		if holding, e2 := q.GetShareholding(ctx, inv.UserID); e2 == nil {
			prior = holding.Shares
		} else if !errors.Is(e2, pgx.ErrNoRows) {
			return e2
		}
		newPct := float64(prior+inv.Shares) / float64(off.TotalShares) * 100
		if _, e = q.UpsertShareholding(ctx, db.UpsertShareholdingParams{
			UserID: inv.UserID, Shares: inv.Shares, OwnershipPct: newPct,
		}); e != nil {
			return e
		}

		// INVARIANT 5: commission only here, on the approved investment, 1 level, customer only.
		if e = s.referral.generateCommission(ctx, q, approved, admin); e != nil {
			return e
		}

		// INVARIANT 8: audit in the same tx.
		if e = audit.Write(ctx, q, audit.Actor(admin), "investment.approve", "investments", investmentID.String(), inv, approved); e != nil {
			return e
		}
		if e = audit.Write(ctx, q, audit.Actor(admin), "shares.issue", "share_ledger", ledger.ID.String(), nil, ledger); e != nil {
			return e
		}
		result = approved
		return nil
	})
	if err == nil && result.Status == db.InvestmentStatusApproved {
		_ = s.events.Publish(ctx, "hk.investment.approved", result)
	}
	return result, err
}

func (s *InvestmentService) Reject(ctx context.Context, admin, investmentID uuid.UUID, reason string) (db.Investment, error) {
	var result db.Investment
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		inv, e := q.GetInvestmentForUpdate(ctx, investmentID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		result, e = q.MarkInvestmentRejected(ctx, db.MarkInvestmentRejectedParams{
			ID: investmentID, RejectedBy: uuid.NullUUID{UUID: admin, Valid: true}, RejectReason: pgText(reason),
		})
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrInvalidState
		}
		if e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "investment.reject", "investments", investmentID.String(), inv, result)
	})
	return result, err
}

// ----- listings -----

func (s *InvestmentService) ListByUser(ctx context.Context, userID uuid.UUID) ([]db.Investment, error) {
	return s.store.ListInvestmentsByUser(ctx, userID)
}

func (s *InvestmentService) ListByStatus(ctx context.Context, status db.InvestmentStatus) ([]db.Investment, error) {
	return s.store.ListInvestmentsByStatus(ctx, status)
}
