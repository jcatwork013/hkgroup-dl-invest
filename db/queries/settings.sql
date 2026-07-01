-- name: ListSettings :many
SELECT key, value FROM site_settings ORDER BY key;

-- name: UpsertSetting :one
INSERT INTO site_settings (key, value, updated_by, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()
RETURNING *;
