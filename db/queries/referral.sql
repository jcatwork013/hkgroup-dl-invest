-- name: CreateReferral :one
-- 1-LEVEL ONLY: stores the direct referrer for a referee. PK on referee_id => at most one referrer.
INSERT INTO referrals (referee_id, referrer_id, referral_type)
VALUES ($1, $2, $3)
ON CONFLICT (referee_id) DO NOTHING
RETURNING *;

-- name: GetReferralByReferee :one
SELECT * FROM referrals WHERE referee_id = $1;

-- name: IsReferredByAdmin :one
-- Whitelist check for early-bird tiers: true iff this user's direct referrer is an admin.
SELECT EXISTS (
    SELECT 1 FROM referrals r
    JOIN users u ON u.id = r.referrer_id
    WHERE r.referee_id = $1 AND u.role = 'admin'
)::boolean AS is_admin_referred;

-- name: ListReferralsByReferrer :many
SELECT * FROM referrals WHERE referrer_id = $1 ORDER BY created_at DESC;

-- name: CreateCommission :one
-- INVARIANT 5: only on approved investments. Multi-level (F1/F2/F3): one row per (investment, level),
-- each crediting beneficiary_id (the upline earner at that level). referral_id stays the chain
-- ORIGIN (the investor's referee_id) so type lookups still join on referrals.referral_type.
INSERT INTO commissions (referral_id, investment_id, base_amount, rate, amount, tax_pit, net_amount, status, level, beneficiary_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9)
ON CONFLICT (investment_id, level) DO NOTHING
RETURNING *;

-- name: ListCommissionsByReferrer :many
-- All commissions EARNED by a user across every level (F1/F2/F3) — keyed by beneficiary_id.
SELECT * FROM commissions WHERE beneficiary_id = $1 ORDER BY created_at DESC;

-- name: ApproveCommission :one
UPDATE commissions SET status = 'approved', approved_by = $2 WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: PayCommission :one
UPDATE commissions SET status = 'paid', paid_at = now() WHERE id = $1 AND status = 'approved'
RETURNING *;

-- name: SumCommissionEarnedByReferrer :one
-- Tổng hoa hồng (net) khả dụng của 1 người (mọi cấp F1/F2/F3, mọi trạng thái trừ rejected) — số dư ví.
-- Cộng vào ví ngay; điểm kiểm soát là khâu DUYỆT RÚT TIỀN của admin.
SELECT COALESCE(SUM(net_amount), 0)::bigint AS total
FROM commissions
WHERE beneficiary_id = $1 AND status <> 'rejected';

-- name: SumPendingCommissionByBeneficiary :one
-- Hoa hồng (net) CHƯA DUYỆT của 1 người — phần đang chờ admin duyệt (đã gộp trong số dư khả dụng).
SELECT COALESCE(SUM(net_amount), 0)::bigint AS total
FROM commissions
WHERE beneficiary_id = $1 AND status = 'pending';

-- name: SumCommissionsByType :one
SELECT
    COALESCE(SUM(c.amount), 0)::bigint AS gross,
    COALESCE(SUM(c.tax_pit), 0)::bigint AS tax,
    COALESCE(SUM(c.net_amount), 0)::bigint AS net
FROM commissions c
JOIN referrals r ON r.referee_id = c.referral_id
WHERE r.referral_type = $1;
