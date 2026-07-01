-- name: CreateReservation :one
-- Early-bird reservation: record-only. Re-reserving the same (user, tier) is idempotent.
INSERT INTO early_reservations (user_id, tier_id, amount_vnd, shares, ownership_pct)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, tier_id) DO NOTHING
RETURNING *;

-- name: GetReservationByUserTier :one
SELECT * FROM early_reservations WHERE user_id = $1 AND tier_id = $2;

-- name: ListReservationsByUser :many
SELECT r.*, t.name AS tier_name
FROM early_reservations r
JOIN investment_tiers t ON t.id = r.tier_id
WHERE r.user_id = $1
ORDER BY r.created_at DESC;

-- name: ListAllReservations :many
SELECT r.*, t.name AS tier_name, u.full_name, u.email
FROM early_reservations r
JOIN investment_tiers t ON t.id = r.tier_id
JOIN users u ON u.id = r.user_id
ORDER BY r.created_at DESC;

-- name: SetReservationStatus :one
UPDATE early_reservations SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;
