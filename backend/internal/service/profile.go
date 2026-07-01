package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/security"
	"github.com/hkgroup/backend/internal/store"
)

type ProfileService struct {
	store   *store.Store
	cryptor *security.Cryptor
}

func NewProfileService(s *store.Store, cryptor *security.Cryptor) *ProfileService {
	return &ProfileService{store: s, cryptor: cryptor}
}

// Thông tin ngân hàng (số TK, chủ TK) là PII → mã hoá at-rest với tiền tố "enc:".
// Giá trị cũ (plaintext, chưa tiền tố) vẫn đọc được → không vỡ dữ liệu hiện có.
const encPrefix = "enc:"

func (s *ProfileService) enc(v string) string {
	if v == "" || s.cryptor == nil {
		return v
	}
	ct, err := s.cryptor.Encrypt([]byte(v))
	if err != nil {
		return v
	}
	return encPrefix + base64.StdEncoding.EncodeToString(ct)
}

func (s *ProfileService) dec(v string) string {
	if s.cryptor == nil || !strings.HasPrefix(v, encPrefix) {
		return v // plaintext cũ
	}
	ct, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(v, encPrefix))
	if err != nil {
		return v
	}
	pt, err := s.cryptor.Decrypt(ct)
	if err != nil {
		return v
	}
	return string(pt)
}

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
	if err != nil {
		return p, err
	}
	// Giải mã PII ngân hàng trước khi trả cho chủ tài khoản / admin.
	p.BankAccountNumber = s.dec(p.BankAccountNumber)
	p.BankAccountName = s.dec(p.BankAccountName)
	return p, nil
}

func (s *ProfileService) Upsert(ctx context.Context, userID uuid.UUID, in ProfileInput) (db.InvestorProfile, error) {
	nat := in.Nationality
	if nat == "" {
		nat = "Việt Nam"
	}
	p, err := s.store.UpsertProfile(ctx, db.UpsertProfileParams{
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
		BankAccountNumber: s.enc(in.BankAccountNumber),
		BankAccountName:   s.enc(in.BankAccountName),
	})
	if err != nil {
		return p, err
	}
	// Trả lại giá trị đã giải mã (không lộ ciphertext ra response).
	p.BankAccountNumber = in.BankAccountNumber
	p.BankAccountName = in.BankAccountName
	return p, nil
}
