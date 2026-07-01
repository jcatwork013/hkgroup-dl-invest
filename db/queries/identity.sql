-- name: CreateKYCRecord :one
INSERT INTO kyc_records (user_id, cccd_number, cccd_image_url, cccd_back_url, selfie_url, status)
VALUES ($1, $2, $3, $4, $5, 'pending')
RETURNING *;

-- name: GetLatestKYCByUser :one
SELECT * FROM kyc_records WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: GetKYCByID :one
SELECT * FROM kyc_records WHERE id = $1;

-- name: ListPendingKYC :many
SELECT * FROM kyc_records WHERE status = 'pending' ORDER BY created_at ASC;

-- name: ReviewKYC :one
UPDATE kyc_records
SET status = $2, reject_reason = $3, reviewed_by = $4, reviewed_at = now()
WHERE id = $1
RETURNING *;

-- name: RecordConsent :one
INSERT INTO consents (user_id, type, ip, user_agent)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListConsentsByUser :many
SELECT * FROM consents WHERE user_id = $1 ORDER BY granted_at DESC;
