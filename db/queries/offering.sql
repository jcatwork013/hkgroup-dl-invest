-- name: GetActiveOffering :one
SELECT * FROM offering WHERE status = 'open' ORDER BY created_at ASC LIMIT 1;

-- name: ListOfferings :many
-- Tất cả vòng gọi vốn (mở + đã đóng), mới nhất trước.
SELECT * FROM offering ORDER BY created_at DESC;

-- name: CloseOpenOfferings :exec
-- Đóng mọi vòng đang mở (gọi trước khi mở vòng mới để chỉ có 1 vòng active).
UPDATE offering SET status = 'closed', updated_at = now() WHERE status = 'open';

-- name: CreateOffering :one
-- Mở một vòng gọi vốn mới (vòng 2, 3, ...). shares_sold=0, status='open'.
INSERT INTO offering (name, valuation_vnd, total_shares, shares_for_sale, shares_sold, status)
VALUES ($1, $2, $3, $4, 0, 'open')
RETURNING *;

-- name: GetOfferingForUpdate :one
SELECT * FROM offering WHERE id = $1 FOR UPDATE;

-- name: AddSharesSold :one
-- INVARIANT 1 guard lives in the offering_sold_within_pool CHECK constraint; this fails the tx
-- if it would push shares_sold past shares_for_sale.
UPDATE offering
SET shares_sold = shares_sold + $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListTiers :many
-- Public landing: normal tiers only. Special (early-bird) tiers are NEVER shown here.
SELECT * FROM investment_tiers
WHERE offering_id = $1 AND active = true AND is_special = false
ORDER BY sort_order ASC;

-- name: ListTiersForUser :many
-- Invest page (authenticated): normal tiers + special tiers only when the user is whitelisted
-- (referred by an admin). $2 = include_special.
SELECT * FROM investment_tiers
WHERE offering_id = $1 AND active = true
  AND (is_special = false OR $2::boolean = true)
ORDER BY sort_order ASC;

-- name: ListAllTiers :many
-- Admin tier management: every tier (active + inactive, normal + special).
SELECT * FROM investment_tiers
WHERE offering_id = $1
ORDER BY sort_order ASC;

-- name: GetTier :one
SELECT * FROM investment_tiers WHERE id = $1;

-- name: CreateTier :one
INSERT INTO investment_tiers (offering_id, name, amount_vnd, shares, ownership_pct, sort_order, active, is_special)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTier :one
UPDATE investment_tiers
SET name = $2, amount_vnd = $3, shares = $4, ownership_pct = $5, sort_order = $6, active = $7, is_special = $8
WHERE id = $1
RETURNING *;

-- name: SetTierActive :one
UPDATE investment_tiers SET active = $2 WHERE id = $1 RETURNING *;

-- name: TierInUse :one
-- True nếu gói đã được tham chiếu bởi hợp đồng hoặc khoản đầu tư (không cho xoá, chỉ ẩn).
SELECT EXISTS (
    SELECT 1 FROM contracts   WHERE tier_id = $1
    UNION ALL
    SELECT 1 FROM investments WHERE tier_id = $1
) AS in_use;

-- name: DeleteTier :exec
DELETE FROM investment_tiers WHERE id = $1;

-- name: SumShareholdings :one
SELECT COALESCE(SUM(shares), 0)::bigint AS total FROM shareholdings;
