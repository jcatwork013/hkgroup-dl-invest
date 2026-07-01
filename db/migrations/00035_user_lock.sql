-- +goose Up
-- +goose StatementBegin
-- Khoá tài khoản: chỉ chặn đăng nhập, KHÔNG xoá dữ liệu (hoa hồng, cổ tức, hồ sơ giữ nguyên).
CREATE TABLE IF NOT EXISTS locked_users (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    locked_by  UUID REFERENCES users(id),
    locked_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS locked_users;
-- +goose StatementEnd
