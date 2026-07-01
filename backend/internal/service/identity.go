package service

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hkgroup/backend/internal/audit"
	"github.com/hkgroup/backend/internal/db"
	"github.com/hkgroup/backend/internal/platform/idgen"
	"github.com/hkgroup/backend/internal/platform/security"
	"github.com/hkgroup/backend/internal/store"
)

type IdentityService struct {
	store    *store.Store
	jwt      *security.JWTManager
	referral *ReferralService
}

func NewIdentityService(s *store.Store, jwt *security.JWTManager, ref *ReferralService) *IdentityService {
	return &IdentityService{store: s, jwt: jwt, referral: ref}
}

type Tokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
}

type RegisterInput struct {
	FullName     string
	Phone        string
	Email        string
	Password     string
	ReferralCode string // optional: referrer's public code (1 level only)
	ReferralType string // 'customer' | 'investor' (default investor)
}

func (s *IdentityService) Register(ctx context.Context, in RegisterInput) (db.User, Tokens, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.FullName == "" || in.Phone == "" || in.Email == "" {
		return db.User{}, Tokens{}, ErrValidation
	}
	if len(in.Password) < 8 {
		return db.User{}, Tokens{}, errors.Join(ErrValidation, errors.New("password must be at least 8 characters"))
	}

	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return db.User{}, Tokens{}, err
	}

	user, err := s.store.CreateUser(ctx, db.CreateUserParams{
		FullName:     in.FullName,
		Phone:        in.Phone,
		Email:        in.Email,
		PasswordHash: hash,
		Role:         db.UserRoleInvestor,
		ReferralCode: idgen.ReferralCode(),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.User{}, Tokens{}, ErrConflict
		}
		return db.User{}, Tokens{}, err
	}

	// 1-level referral attribution (best-effort; never fails registration).
	if in.ReferralCode != "" {
		_ = s.referral.Attribute(ctx, in.ReferralCode, user.ID, in.ReferralType)
	}

	tokens, err := s.issueTokens(user)
	return user, tokens, err
}

// RegisterCustomer: tự đăng ký ở duoclieuhk.vn → tài khoản KHÁCH HÀNG (role=customer).
// Chưa phải CTV, mã giới thiệu chưa dùng tới cho tới khi được duyệt làm affiliate.
func (s *IdentityService) RegisterCustomer(ctx context.Context, in RegisterInput) (db.User, Tokens, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.FullName == "" || in.Phone == "" || in.Email == "" {
		return db.User{}, Tokens{}, ErrValidation
	}
	if len(in.Password) < 8 {
		return db.User{}, Tokens{}, errors.Join(ErrValidation, errors.New("password must be at least 8 characters"))
	}
	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return db.User{}, Tokens{}, err
	}
	user, err := s.store.CreateUser(ctx, db.CreateUserParams{
		FullName:     in.FullName,
		Phone:        in.Phone,
		Email:        in.Email,
		PasswordHash: hash,
		Role:         db.UserRoleCustomer,
		ReferralCode: idgen.ReferralCode(), // sinh sẵn (unique) nhưng chỉ lộ khi thành CTV
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.User{}, Tokens{}, ErrConflict
		}
		return db.User{}, Tokens{}, err
	}

	// "Ăn ref": nếu đăng ký qua link giới thiệu (?ref=<mã>), KHOÁ FIRST-TOUCH theo SĐT khách →
	// từ nay MỌI đơn của SĐT này gán affiliate = người giới thiệu (checkout đã honor lock). Chỉ nhận
	// mã của CTV (saler) và không tự giới thiệu chính mình. Best-effort: KHÔNG làm hỏng đăng ký.
	if code := strings.TrimSpace(in.ReferralCode); code != "" {
		if ref, e := s.store.GetUserByReferralCode(ctx, code); e == nil &&
			ref.ID != user.ID && ref.Role == db.UserRoleSaler {
			_ = s.store.SetReferralLockIfAbsent(ctx, user.Phone, ref.ID)
		}
	}

	tokens, err := s.issueTokens(user)
	return user, tokens, err
}

func (s *IdentityService) Login(ctx context.Context, email, password string) (db.User, Tokens, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, Tokens{}, ErrInvalidCredential
	}
	if err != nil {
		return db.User{}, Tokens{}, err
	}
	if !security.CheckPassword(user.PasswordHash, password) {
		return db.User{}, Tokens{}, ErrInvalidCredential
	}
	// Tài khoản bị khoá → chặn đăng nhập (dữ liệu vẫn giữ nguyên).
	if locked, _ := s.store.IsUserLocked(ctx, user.ID); locked {
		return db.User{}, Tokens{}, ErrAccountLocked
	}
	tokens, err := s.issueTokens(user)
	return user, tokens, err
}

// AdminSetRole: admin đổi vai trò (thăng/giáng chức). role ∈ {customer, saler, investor, admin}.
func (s *IdentityService) AdminSetRole(ctx context.Context, admin, target uuid.UUID, role string) error {
	var r db.UserRole
	switch role {
	case "customer":
		r = db.UserRoleCustomer
	case "saler":
		r = db.UserRoleSaler
	case "investor":
		r = db.UserRoleInvestor
	case "admin":
		r = db.UserRoleAdmin
	default:
		return ErrValidation
	}
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.UpdateUserRole(ctx, target, r); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "user.set_role", "users", target.String(), nil, map[string]string{"role": role})
	})
}

// AdminLock / AdminUnlock: khoá/mở tài khoản (chỉ chặn login, không mất dữ liệu).
func (s *IdentityService) AdminLock(ctx context.Context, admin, target uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.LockUser(ctx, target, admin); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "user.lock", "users", target.String(), nil, nil)
	})
}
func (s *IdentityService) AdminUnlock(ctx context.Context, admin, target uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.UnlockUser(ctx, target); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "user.unlock", "users", target.String(), nil, nil)
	})
}
func (s *IdentityService) ListLockedUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	return s.store.ListLockedUserIDs(ctx)
}

func (s *IdentityService) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	claims, err := s.jwt.Verify(refreshToken, security.RefreshToken)
	if err != nil {
		return Tokens{}, ErrInvalidCredential
	}
	user, err := s.store.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return Tokens{}, ErrInvalidCredential
	}
	return s.issueTokens(user)
}

// ChangePassword lets an authenticated user (investor OR admin) change their own password after
// re-verifying the current one. The route only needs a valid token, so the same flow serves both
// roles. We re-check the current password to stop a stolen/borrowed session from silently taking
// over the account, and reject a no-op change.
func (s *IdentityService) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) error {
	if len(next) < 8 {
		return errors.Join(ErrValidation, errors.New("mật khẩu mới phải có ít nhất 8 ký tự"))
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !security.CheckPassword(user.PasswordHash, current) {
		return errors.Join(ErrValidation, errors.New("mật khẩu hiện tại không đúng"))
	}
	if security.CheckPassword(user.PasswordHash, next) {
		return errors.Join(ErrValidation, errors.New("mật khẩu mới phải khác mật khẩu hiện tại"))
	}
	hash, err := security.HashPassword(next)
	if err != nil {
		return err
	}
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		if e := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: userID, PasswordHash: hash}); e != nil {
			return e
		}
		// Audit the event WITHOUT logging any password material.
		return audit.Write(ctx, q, audit.Actor(userID), "user.change_password", "users", userID.String(), nil, nil)
	})
}

func (s *IdentityService) issueTokens(u db.User) (Tokens, error) {
	access, err := s.jwt.IssueAccess(u.ID, string(u.Role))
	if err != nil {
		return Tokens{}, err
	}
	refresh, err := s.jwt.IssueRefresh(u.ID, string(u.Role))
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{Access: access, Refresh: refresh}, nil
}

// GetUser returns the current user row (for GET /me — keeps the client's KYC status/message fresh
// after an admin review or re-KYC request, driving the notification bell).
func (s *IdentityService) GetUser(ctx context.Context, id uuid.UUID) (db.User, error) {
	u, err := s.store.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, ErrNotFound
	}
	return u, err
}

// SubmitKYC stores an eKYC record (CCCD mặt trước + mặt sau + selfie) and moves the user to
// kyc_status='pending'. Cả hai mặt CCCD đều bắt buộc.
func (s *IdentityService) SubmitKYC(ctx context.Context, userID uuid.UUID, cccd, cccdURL, cccdBackURL, selfieURL string) (db.KycRecord, error) {
	if cccd == "" || cccdURL == "" || cccdBackURL == "" || selfieURL == "" {
		return db.KycRecord{}, ErrValidation
	}
	var rec db.KycRecord
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		rec, e = q.CreateKYCRecord(ctx, db.CreateKYCRecordParams{
			UserID: userID, CccdNumber: cccd, CccdImageUrl: cccdURL, CccdBackUrl: cccdBackURL, SelfieUrl: selfieURL,
		})
		if e != nil {
			return e
		}
		// New submission clears any prior rejection message and moves to 'pending'.
		if e = q.SetUserKYCResult(ctx, db.SetUserKYCResultParams{ID: userID, KycStatus: db.KycStatusPending, KycMessage: ""}); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(userID), "kyc.submit", "kyc_records", rec.ID.String(), nil, rec)
	})
	return rec, err
}

// RecordConsent appends a consent log entry (Nghị định 13/2023).
func (s *IdentityService) RecordConsent(ctx context.Context, userID uuid.UUID, ctype, ip, ua string) error {
	_, err := s.store.RecordConsent(ctx, db.RecordConsentParams{
		UserID:    userID,
		Type:      ctype,
		Ip:        parseIP(ip),
		UserAgent: pgText(ua),
	})
	return err
}

// ----- Admin KYC review -----

func (s *IdentityService) ListPendingKYC(ctx context.Context) ([]db.KycRecord, error) {
	return s.store.ListPendingKYC(ctx)
}

// GetUserKYC returns a user's latest KYC record (ảnh CCCD mặt trước/sau + selfie) so an admin can
// review the submitted images before approving from the user-management screen. ErrNotFound when the
// user has never submitted KYC (vd tài khoản admin tạo tay).
func (s *IdentityService) GetUserKYC(ctx context.Context, userID uuid.UUID) (db.KycRecord, error) {
	rec, err := s.store.GetLatestKYCByUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.KycRecord{}, ErrNotFound
	}
	return rec, err
}

// ----- Admin user management -----

func (s *IdentityService) ListUsers(ctx context.Context, limit, offset int32) ([]db.ListUsersRow, error) {
	return s.store.ListUsers(ctx, db.ListUsersParams{Limit: limit, Offset: offset})
}

// AdminCreateUser lets an admin create an account with a chosen role.
// Only admins reach this (enforced by RBAC). Creating an 'admin' is therefore admin-only.
func (s *IdentityService) AdminCreateUser(ctx context.Context, admin uuid.UUID, in RegisterInput, role string) (db.User, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.FullName == "" || in.Phone == "" || in.Email == "" || len(in.Password) < 8 {
		return db.User{}, ErrValidation
	}
	r := db.UserRoleInvestor
	switch role {
	case string(db.UserRoleAdmin):
		r = db.UserRoleAdmin
	case string(db.UserRoleSaler):
		r = db.UserRoleSaler
	}
	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return db.User{}, err
	}
	var user db.User
	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		var e error
		user, e = q.CreateUser(ctx, db.CreateUserParams{
			FullName: in.FullName, Phone: in.Phone, Email: in.Email,
			PasswordHash: hash, Role: r, ReferralCode: idgen.ReferralCode(),
		})
		if e != nil {
			if isUniqueViolation(e) {
				return ErrConflict
			}
			return e
		}
		// Admins & salers don't go through investor eKYC.
		if r == db.UserRoleAdmin || r == db.UserRoleSaler {
			if e = q.SetUserKYCStatus(ctx, db.SetUserKYCStatusParams{ID: user.ID, KycStatus: db.KycStatusApproved}); e != nil {
				return e
			}
			user.KycStatus = db.KycStatusApproved
		}
		return audit.Write(ctx, q, audit.Actor(admin), "user.create", "users", user.ID.String(), nil, map[string]any{"email": user.Email, "role": r})
	})
	return user, err
}

// AdminSetKYC manually sets a user's KYC status (thủ công), without requiring an upload. When
// approve=false this is also the "yêu cầu KYC lại" (request re-KYC) flow: status -> 'rejected' and
// `reason` is stored in users.kyc_message so the user's notification bell shows why (vd ảnh mờ, sai
// định dạng). On approve the message is cleared.
func (s *IdentityService) AdminSetKYC(ctx context.Context, admin, userID uuid.UUID, approve bool, reason string) (db.User, error) {
	status := db.KycStatusRejected
	msg := reason
	if approve {
		status = db.KycStatusApproved
		msg = ""
	}
	var user db.User
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		before, e := q.GetUserByID(ctx, userID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if e = q.SetUserKYCResult(ctx, db.SetUserKYCResultParams{ID: userID, KycStatus: status, KycMessage: msg}); e != nil {
			return e
		}
		user = before
		user.KycStatus = status
		user.KycMessage = msg
		return audit.Write(ctx, q, audit.Actor(admin), "kyc.manual", "users", userID.String(),
			map[string]any{"kyc_status": before.KycStatus}, map[string]any{"kyc_status": status, "reason": reason})
	})
	return user, err
}

// DeleteUser performs an admin "hard delete": it removes an account and ALL data tied to it —
// personal data (profile, KYC records, uploaded images, consents) AND the full financial footprint
// (shares/share_ledger, investments + payments + contracts, dividend payouts, withdrawals, and every
// commission linked to the account: both the ones it EARNED and the ones its investments paid to its
// upline). The freed shares are returned to the pool (offering.shares_sold recomputed). Other holders'
// ownership_pct is unaffected (denominator is the fixed total_shares). Only admin accounts and
// self-deletion are refused. Audit CONTENT is preserved — only the deleted user's actor link is cut
// (FK ON DELETE SET NULL, migration 00023), per the NĐ13 erasure policy.
// NOTE: clawing back upline commission lowers OTHER users' wallet balance — intended ("xoá tất cả liên quan").
func (s *IdentityService) DeleteUser(ctx context.Context, admin, userID uuid.UUID) error {
	if admin == userID {
		return errors.Join(ErrForbidden, errors.New("không thể tự xoá tài khoản của chính bạn"))
	}
	target, err := s.store.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if target.Role == db.UserRoleAdmin {
		return errors.Join(ErrForbidden, errors.New("không thể xoá tài khoản quản trị viên"))
	}

	// Snapshot encrypted upload paths first so we can delete the blobs after the rows commit.
	paths, err := s.store.ListUploadPathsByUser(ctx, userID)
	if err != nil {
		return err
	}

	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		// 1) Xoá TOÀN BỘ dấu vết tài chính của tài khoản. Thứ tự tôn trọng các FK RESTRICT.
		//    Hoa hồng: cả phần ĐẦU TƯ của tài khoản này đã trả cho upline (commissions.investment_id),
		//    lẫn phần tài khoản này KIẾM được từ người nó giới thiệu (commissions.referral_id).
		//    Lưu ý: việc này làm GIẢM số dư ví của upline — đúng yêu cầu "xoá tất cả liên quan".
		if e := q.DeleteCommissionsForInvestmentsByUser(ctx, userID); e != nil {
			return e
		}
		if e := q.DeleteCommissionsEarnedByReferrer(ctx, userID); e != nil {
			return e
		}
		if e := q.DeleteDividendPayoutsByUser(ctx, userID); e != nil {
			return e
		}
		if e := q.DeleteWithdrawalsByUser(ctx, userID); e != nil {
			return e
		}

		// share_ledger là append-only (trigger 00009 chặn DELETE). Tạm tắt USER-trigger TRONG tx này
		// (FK/cascade vẫn chạy); DDL có tính giao dịch nên nếu tx rollback trigger tự bật lại.
		if e := q.DisableShareLedgerImmutability(ctx); e != nil {
			return e
		}
		if e := q.DeleteShareLedgerByUser(ctx, userID); e != nil {
			return e
		}
		if e := q.EnableShareLedgerImmutability(ctx); e != nil {
			return e
		}
		if e := q.DeleteShareholdingByUser(ctx, userID); e != nil {
			return e
		}

		// 2) Đầu tư (FK RESTRICT) trước hợp đồng; payments cascade theo đầu tư.
		if e := q.DeleteInvestmentsByUser(ctx, userID); e != nil {
			return e
		}
		if e := q.DeleteOutgoingReferrals(ctx, userID); e != nil {
			return e
		}
		if e := q.DeleteContractsByUser(ctx, userID); e != nil {
			return e
		}

		// 3) Trả cổ phần đã gỡ về pool: dựng lại shares_sold từ các đầu tư 'approved' còn lại.
		if e := q.RecomputeSharesSold(ctx); e != nil {
			return e
		}

		// 4) Ghi audit TRƯỚC khi xoá user (actor = admin, không bị xoá).
		if e := audit.Write(ctx, q, audit.Actor(admin), "user.delete", "users", userID.String(),
			map[string]any{"email": target.Email, "full_name": target.FullName, "role": target.Role}, nil); e != nil {
			return e
		}
		// Cascades: investor_profiles, uploads, kyc_records, consents, referrals(referee_id).
		return q.DeleteUser(ctx, userID)
	})
	if err != nil {
		return err
	}

	// Best-effort cleanup of the encrypted KYC blobs on disk (DB rows already gone).
	for _, p := range paths {
		_ = os.Remove(p)
	}
	return nil
}

func (s *IdentityService) ReviewKYC(ctx context.Context, admin, kycID uuid.UUID, approve bool, reason string) (db.KycRecord, error) {
	var rec db.KycRecord
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		existing, e := q.GetKYCByID(ctx, kycID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		status := db.KycStatusRejected
		if approve {
			status = db.KycStatusApproved
		}
		rec, e = q.ReviewKYC(ctx, db.ReviewKYCParams{
			ID:           kycID,
			Status:       status,
			RejectReason: pgText(reason),
			ReviewedBy:   uuid.NullUUID{UUID: admin, Valid: true},
		})
		if e != nil {
			return e
		}
		msg := ""
		if !approve {
			msg = reason
		}
		if e = q.SetUserKYCResult(ctx, db.SetUserKYCResultParams{ID: existing.UserID, KycStatus: status, KycMessage: msg}); e != nil {
			return e
		}
		return audit.Write(ctx, q, audit.Actor(admin), "kyc.review", "kyc_records", kycID.String(), existing, rec)
	})
	return rec, err
}
