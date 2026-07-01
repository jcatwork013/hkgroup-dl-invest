-- name: CreateDividend :one
-- INVARIANT 6: dividends exist ONLY because an admin created this row. No auto/cron path.
INSERT INTO dividends (declared_by, period, total_amount, note)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateDividendPayout :one
INSERT INTO dividend_payouts (dividend_id, user_id, shares, amount, equal_share, bonus, band, band_rate, invested_vnd)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (dividend_id, user_id) DO NOTHING
RETURNING *;

-- name: MarkPayoutPaid :one
UPDATE dividend_payouts SET paid_at = now() WHERE id = $1 AND paid_at IS NULL
RETURNING *;

-- name: ListDividends :many
SELECT * FROM dividends ORDER BY declared_at DESC;

-- DeleteDividend: xoá 1 đợt cổ tức; dividend_payouts cascade theo FK (00007 ON DELETE CASCADE).
-- Lưu ý: phải xoá revenue_distributions trỏ tới nó TRƯỚC (FK NO ACTION).
-- name: DeleteDividend :exec
DELETE FROM dividends WHERE id = $1;

-- name: ListPayoutsByDividend :many
SELECT p.*, u.full_name, u.email FROM dividend_payouts p
JOIN users u ON u.id = p.user_id
WHERE p.dividend_id = $1
ORDER BY p.amount DESC;

-- name: ListPayoutsByUser :many
SELECT p.*, d.period, d.note FROM dividend_payouts p
JOIN dividends d ON d.id = p.dividend_id
WHERE p.user_id = $1
ORDER BY p.created_at DESC;

-- name: SumDividendPaidToUser :one
-- The ONLY "money received back" figure shown to investors: real dividends actually paid.
SELECT COALESCE(SUM(amount), 0)::bigint AS total
FROM dividend_payouts WHERE user_id = $1 AND paid_at IS NOT NULL;
