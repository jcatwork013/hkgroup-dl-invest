-- name: CreateContract :one
INSERT INTO contracts (user_id, tier_id, status)
VALUES ($1, $2, 'draft')
RETURNING *;

-- name: SignContract :one
UPDATE contracts
SET status = 'signed', signed_at = now(), signature_otp_ref = $2, pdf_url = $3
WHERE id = $1 AND status = 'draft'
RETURNING *;

-- name: GetContract :one
SELECT * FROM contracts WHERE id = $1;

-- name: CreateInvestment :one
INSERT INTO investments (code, user_id, contract_id, tier_id, amount_vnd, shares, status, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)
RETURNING *;

-- name: GetInvestmentByIdempotencyKey :one
SELECT * FROM investments WHERE idempotency_key = $1;

-- name: GetInvestment :one
SELECT * FROM investments WHERE id = $1;

-- name: GetInvestmentForUpdate :one
SELECT * FROM investments WHERE id = $1 FOR UPDATE;

-- name: ListInvestmentsByUser :many
SELECT * FROM investments WHERE user_id = $1 ORDER BY created_at DESC;

-- name: ListInvestmentsByStatus :many
SELECT * FROM investments WHERE status = $1 ORDER BY created_at ASC;

-- name: MarkInvestmentReconciled :one
-- INVARIANT 2: pending -> reconciled, stamping actor + timestamp.
UPDATE investments
SET status = 'reconciled', reconciled_by = $2, reconciled_at = now(), updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: MarkInvestmentApproved :one
-- INVARIANT 2: reconciled -> approved, stamping actor + timestamp.
UPDATE investments
SET status = 'approved', approved_by = $2, approved_at = now(), updated_at = now()
WHERE id = $1 AND status = 'reconciled'
RETURNING *;

-- name: MarkInvestmentRejected :one
UPDATE investments
SET status = 'rejected', rejected_by = $2, rejected_at = now(), reject_reason = $3, updated_at = now()
WHERE id = $1 AND status IN ('pending','reconciled')
RETURNING *;

-- name: CreatePayment :one
INSERT INTO payments (investment_id, bank, company_account, company_account_name, amount_vnd, transfer_note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPaymentByInvestment :one
SELECT * FROM payments WHERE investment_id = $1;

-- name: DeclarePaymentTransferred :one
UPDATE payments SET declared_at = now() WHERE investment_id = $1 AND declared_at IS NULL
RETURNING *;

-- name: ReconcilePayment :one
-- INVARIANT 7: funds reconciliation stamps actor + time before any share issuance.
UPDATE payments
SET reconciled_at = now(), reconciled_by = $2
WHERE investment_id = $1 AND reconciled_at IS NULL
RETURNING *;
