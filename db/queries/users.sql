-- name: CreateUser :one
INSERT INTO users (full_name, phone, email, password_hash, role, referral_code)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByReferralCode :one
SELECT * FROM users WHERE referral_code = $1;

-- name: SetUserKYCStatus :exec
UPDATE users SET kyc_status = $2, updated_at = now() WHERE id = $1;

-- SetUserKYCResult: đặt trạng thái KYC kèm thông điệp cho người dùng (lý do từ chối / yêu cầu lại).
-- name: SetUserKYCResult :exec
UPDATE users SET kyc_status = $2, kyc_message = $3, updated_at = now() WHERE id = $1;

-- UpdateUserPassword: đổi mật khẩu (đã hash bcrypt) cho chính chủ tài khoản (investor hoặc admin).
-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1;

-- name: EnsureAdmin :one
INSERT INTO users (full_name, phone, email, password_hash, role, kyc_status, referral_code)
VALUES ($1, $2, $3, $4, 'admin', 'approved', $5)
ON CONFLICT (email) DO UPDATE SET role = 'admin'
RETURNING *;

-- name: CountUsersByRole :one
SELECT count(*) FROM users WHERE role = $1;

-- name: ListUsers :many
SELECT id, full_name, phone, email, role, kyc_status, referral_code, created_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- UserDeletionStats: dấu vết tài chính/audit chặn "xoá an toàn". Tất cả = 0 thì mới cho xoá.
-- name: UserDeletionStats :one
SELECT
    COALESCE((SELECT shares FROM shareholdings WHERE user_id = $1), 0)::bigint                       AS shares,
    (SELECT count(*) FROM share_ledger     WHERE user_id = $1)                                       AS ledger_rows,
    (SELECT count(*) FROM investments      WHERE user_id = $1 AND status IN ('reconciled','approved')) AS approved_investments,
    (SELECT count(*) FROM commissions      WHERE beneficiary_id = $1)                                AS commissions_as_beneficiary,
    (SELECT count(*) FROM commissions c JOIN referrals r ON r.referee_id = c.referral_id
                                        WHERE r.referrer_id = $1)                                    AS downline_commissions,
    (SELECT count(*) FROM dividend_payouts WHERE user_id = $1)                                       AS dividend_payouts,
    (SELECT count(*) FROM withdrawals      WHERE user_id = $1)                                       AS withdrawals,
    (SELECT count(*) FROM dividends        WHERE declared_by = $1)
      + (SELECT count(*) FROM revenue_distributions WHERE created_by = $1)                           AS admin_financial;

-- name: ListUploadPathsByUser :many
SELECT path FROM uploads WHERE user_id = $1;

-- DeleteInvestmentsByUser: chỉ chạy sau khi đã xác nhận không có đầu tư 'approved' (payments cascade).
-- name: DeleteInvestmentsByUser :exec
DELETE FROM investments WHERE user_id = $1;

-- name: DeleteOutgoingReferrals :exec
DELETE FROM referrals WHERE referrer_id = $1;

-- name: DeleteContractsByUser :exec
DELETE FROM contracts WHERE user_id = $1;

-- DeleteUser: cascade theo FK (referrals.referee, investor_profiles, uploads, kyc_records, consents);
-- audit_logs.actor_id được ON DELETE SET NULL (giữ nội dung audit, chỉ cắt liên kết cá nhân).
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- ===== Xoá tài khoản kèm TOÀN BỘ dấu vết tài chính (hard delete) =====
-- Dùng cho luồng admin "xoá triệt để": gỡ cổ phần, cổ tức, hoa hồng, rút tiền của tài khoản rồi
-- trả cổ phần về pool. share_ledger là append-only (trigger chặn DELETE 00009) nên phải tạm tắt
-- user-trigger TRONG transaction (chỉ tắt user-trigger, FK/cascade vẫn hoạt động).

-- name: DisableShareLedgerImmutability :exec
ALTER TABLE share_ledger DISABLE TRIGGER USER;

-- name: EnableShareLedgerImmutability :exec
ALTER TABLE share_ledger ENABLE TRIGGER USER;

-- name: DeleteShareLedgerByUser :exec
DELETE FROM share_ledger WHERE user_id = $1;

-- name: DeleteShareholdingByUser :exec
DELETE FROM shareholdings WHERE user_id = $1;

-- name: DeleteDividendPayoutsByUser :exec
DELETE FROM dividend_payouts WHERE user_id = $1;

-- name: DeleteWithdrawalsByUser :exec
DELETE FROM withdrawals WHERE user_id = $1;

-- DeleteCommissionsForInvestmentsByUser: hoa hồng mà ĐẦU TƯ của user này đã trả cho upline.
-- name: DeleteCommissionsForInvestmentsByUser :exec
DELETE FROM commissions
WHERE investment_id IN (SELECT id FROM investments WHERE user_id = $1);

-- DeleteCommissionsEarnedByReferrer: hoa hồng mà user này KIẾM được (từ người nó giới thiệu).
-- name: DeleteCommissionsEarnedByReferrer :exec
DELETE FROM commissions
WHERE referral_id IN (SELECT referee_id FROM referrals WHERE referrer_id = $1);

-- RecomputeSharesSold: dựng lại pool đã bán cho MỌI offering từ các đầu tư 'approved' còn lại
-- (sau khi đã xoá đầu tư của tài khoản). shares_sold == tổng cổ phần đã cấp; ownership_pct của
-- người khác KHÔNG đổi vì mẫu số là total_shares cố định.
-- name: RecomputeSharesSold :exec
UPDATE offering o SET shares_sold = COALESCE((
    SELECT SUM(i.shares) FROM investments i
    JOIN investment_tiers t ON t.id = i.tier_id
    WHERE t.offering_id = o.id AND i.status = 'approved'
), 0), updated_at = now();
