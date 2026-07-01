-- name: CreateUpload :one
INSERT INTO uploads (user_id, kind, content_type, path)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUpload :one
SELECT * FROM uploads WHERE id = $1;
