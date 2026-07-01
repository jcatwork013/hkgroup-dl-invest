-- +goose Up
-- +goose StatementBegin

-- Encrypted file uploads (KYC images: CCCD + selfie). Files are AES-256-GCM encrypted at rest;
-- only the path/metadata live here. Access: owner or admin only.
CREATE TABLE uploads (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind         TEXT        NOT NULL,   -- 'cccd' | 'selfie'
    content_type TEXT        NOT NULL,
    path         TEXT        NOT NULL,   -- encrypted blob path on disk
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_uploads_user ON uploads(user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE uploads;
-- +goose StatementEnd
