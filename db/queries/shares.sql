-- name: InsertShareLedger :one
INSERT INTO share_ledger (user_id, investment_id, shares_delta, reason)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpsertShareholding :one
-- Keeps the read model in lock-step with the ledger inside the issuing tx.
INSERT INTO shareholdings (user_id, shares, ownership_pct, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (user_id) DO UPDATE
SET shares = shareholdings.shares + EXCLUDED.shares,
    ownership_pct = $3,
    updated_at = now()
RETURNING *;

-- name: GetShareholding :one
SELECT * FROM shareholdings WHERE user_id = $1;

-- name: SumLedgerByUser :one
-- INVARIANT 4: reconcile shareholdings against the append-only ledger at any time.
SELECT COALESCE(SUM(shares_delta), 0)::bigint AS total FROM share_ledger WHERE user_id = $1;

-- name: CapTable :many
SELECT s.user_id, u.full_name, u.email, s.shares, s.ownership_pct
FROM shareholdings s
JOIN users u ON u.id = s.user_id
WHERE s.shares > 0
ORDER BY s.shares DESC;

-- name: SetShareholdingPct :exec
UPDATE shareholdings SET ownership_pct = $2, updated_at = now() WHERE user_id = $1;

-- name: ListAllShareholdings :many
SELECT * FROM shareholdings WHERE shares > 0;

-- name: ListActiveAccounts :many
-- Active shareholders with their TRUE invested capital (sum of approved investments) — the basis
-- for the tiered bonus banding. invested_vnd uses amount_vnd, not shares×price, so a 50tr package
-- bands exactly on the 50tr boundary rather than 49.5tr.
SELECT sh.user_id,
       sh.shares,
       COALESCE(SUM(i.amount_vnd) FILTER (WHERE i.status = 'approved'), 0)::bigint AS invested_vnd
FROM shareholdings sh
LEFT JOIN investments i ON i.user_id = sh.user_id
WHERE sh.shares > 0
GROUP BY sh.user_id, sh.shares;
