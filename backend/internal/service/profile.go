package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/store"
)

type ProfileService struct {
	store *store.Store
}

func NewProfileService(s *store.Store) *ProfileService { return &ProfileService{store: s} }

// ProfileInput carries the editable investor profile fields.
type ProfileInput struct {
	DateOfBirth       string `json:"date_of_birth"`
	Gender            string `json:"gender"`
	Nationality       string `json:"nationality"`
	CccdNumber        string `json:"cccd_number"`
	CccdIssueDate     string `json:"cccd_issue_date"`
	CccdIssuePlace    string `json:"cccd_issue_place"`
	PermanentAddress  string `json:"permanent_address"`
	ContactAddress    string `json:"contact_address"`
	Occupation        string `json:"occupation"`
	TaxCode           string `json:"tax_code"`
	BankName          string `json:"bank_name"`
	BankAccountNumber string `json:"bank_account_number"`
	BankAccountName   string `json:"bank_account_name"`
}

// Get returns the profile, or an empty (but user-bound) profile if none exists yet.
func (s *ProfileService) Get(ctx context.Context, userID uuid.UUID) (db.InvestorProfile, error) {
	p, err := s.store.GetProfile(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.InvestorProfile{UserID: userID, Nationality: "Việt Nam"}, nil
	}
	return p, err
}

func (s *ProfileService) Upsert(ctx context.Context, userID uuid.UUID, in ProfileInput) (db.InvestorProfile, error) {
	nat := in.Nationality
	if nat == "" {
		nat = "Việt Nam"
	}
	return s.store.UpsertProfile(ctx, db.UpsertProfileParams{
		UserID:            userID,
		DateOfBirth:       in.DateOfBirth,
		Gender:            in.Gender,
		Nationality:       nat,
		CccdNumber:        in.CccdNumber,
		CccdIssueDate:     in.CccdIssueDate,
		CccdIssuePlace:    in.CccdIssuePlace,
		PermanentAddress:  in.PermanentAddress,
		ContactAddress:    in.ContactAddress,
		Occupation:        in.Occupation,
		TaxCode:           in.TaxCode,
		BankName:          in.BankName,
		BankAccountNumber: in.BankAccountNumber,
		BankAccountName:   in.BankAccountName,
	})
}
