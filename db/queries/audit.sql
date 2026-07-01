-- name: InsertAuditLog :one
-- INVARIANT 8: called inside the same tx as the mutation it records. Table is append-only.
INSERT INTO audit_logs (actor_id, action, entity, entity_id, before, after, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAuditLogsByEntity :many
SELECT * FROM audit_logs
WHERE entity = $1 AND entity_id = $2
ORDER BY created_at DESC;
