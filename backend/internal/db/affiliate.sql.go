// Hand-written (sqlc generate bị chặn bởi legacy reservation.sql). Vòng đời affiliate:
// khách hàng → yêu cầu làm CTV → admin duyệt → nâng role saler + cấp mã.
package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Vai trò khách hàng (mặc định khi tự đăng ký ở shop). saler/investor/admin ở nơi khác.
const UserRoleCustomer UserRole = "customer"

type AffiliateRequestRow struct {
	UserID    uuid.UUID          `json:"user_id"`
	FullName  string             `json:"full_name"`
	Email     string             `json:"email"`
	Phone     string             `json:"phone"`
	Status    string             `json:"status"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

const upsertAffiliateRequest = `
INSERT INTO affiliate_requests (user_id, status) VALUES ($1, 'pending')
ON CONFLICT (user_id) DO UPDATE SET status = 'pending', updated_at = now()
`

func (q *Queries) UpsertAffiliateRequest(ctx context.Context, userID uuid.UUID) error {
	_, err := q.db.Exec(ctx, upsertAffiliateRequest, userID)
	return err
}

const getAffiliateRequestStatus = `SELECT status FROM affiliate_requests WHERE user_id = $1`

func (q *Queries) GetAffiliateRequestStatus(ctx context.Context, userID uuid.UUID) (string, error) {
	var status string
	err := q.db.QueryRow(ctx, getAffiliateRequestStatus, userID).Scan(&status)
	return status, err
}

const listPendingAffiliateRequests = `
SELECT ar.user_id, u.full_name, u.email, u.phone, ar.status, ar.created_at
FROM affiliate_requests ar JOIN users u ON u.id = ar.user_id
WHERE ar.status = 'pending' ORDER BY ar.created_at
`

func (q *Queries) ListPendingAffiliateRequests(ctx context.Context) ([]AffiliateRequestRow, error) {
	rows, err := q.db.Query(ctx, listPendingAffiliateRequests)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AffiliateRequestRow{}
	for rows.Next() {
		var i AffiliateRequestRow
		if err := rows.Scan(&i.UserID, &i.FullName, &i.Email, &i.Phone, &i.Status, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const setAffiliateRequestStatus = `UPDATE affiliate_requests SET status = $2, reviewed_by = $3, updated_at = now() WHERE user_id = $1`

func (q *Queries) SetAffiliateRequestStatus(ctx context.Context, userID uuid.UUID, status string, reviewedBy uuid.UUID) error {
	_, err := q.db.Exec(ctx, setAffiliateRequestStatus, userID, status, reviewedBy)
	return err
}

const updateUserRole = `UPDATE users SET role = $2, updated_at = now() WHERE id = $1`

func (q *Queries) UpdateUserRole(ctx context.Context, id uuid.UUID, role UserRole) error {
	_, err := q.db.Exec(ctx, updateUserRole, id, role)
	return err
}

// ---- Khoá / mở tài khoản (locked_users) ----

const lockUser = `INSERT INTO locked_users (user_id, locked_by) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`

func (q *Queries) LockUser(ctx context.Context, userID, by uuid.UUID) error {
	_, err := q.db.Exec(ctx, lockUser, userID, by)
	return err
}

const unlockUser = `DELETE FROM locked_users WHERE user_id = $1`

func (q *Queries) UnlockUser(ctx context.Context, userID uuid.UUID) error {
	_, err := q.db.Exec(ctx, unlockUser, userID)
	return err
}

const isUserLocked = `SELECT EXISTS(SELECT 1 FROM locked_users WHERE user_id = $1)`

func (q *Queries) IsUserLocked(ctx context.Context, userID uuid.UUID) (bool, error) {
	var locked bool
	err := q.db.QueryRow(ctx, isUserLocked, userID).Scan(&locked)
	return locked, err
}

const listLockedUserIDs = `SELECT user_id FROM locked_users`

func (q *Queries) ListLockedUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := q.db.Query(ctx, listLockedUserIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ---- Khoá giới thiệu first-touch theo SĐT khách (customer_referral_locks) ----

const getReferralLock = `SELECT referrer_id FROM customer_referral_locks WHERE customer_phone = $1`

func (q *Queries) GetReferralLock(ctx context.Context, phone string) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.db.QueryRow(ctx, getReferralLock, phone).Scan(&id)
	return id, err
}

// SetReferralLockIfAbsent chỉ ghi khi CHƯA có (ON CONFLICT DO NOTHING) → first-touch không bị ghi đè.
const setReferralLockIfAbsent = `
INSERT INTO customer_referral_locks (customer_phone, referrer_id) VALUES ($1, $2)
ON CONFLICT (customer_phone) DO NOTHING`

func (q *Queries) SetReferralLockIfAbsent(ctx context.Context, phone string, referrer uuid.UUID) error {
	_, err := q.db.Exec(ctx, setReferralLockIfAbsent, phone, referrer)
	return err
}
