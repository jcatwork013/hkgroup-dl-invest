-- +goose Up
-- +goose StatementBegin
-- Vai trò KHÁCH HÀNG: tài khoản mặc định khi tự đăng ký ở duoclieuhk.vn (chưa phải CTV).
-- PG cho ADD VALUE trong transaction miễn không dùng ngay trong cùng tx.
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'customer';
-- +goose StatementEnd

-- +goose StatementBegin
-- Yêu cầu trở thành Cộng tác viên bán hàng (affiliate). Admin duyệt → nâng role lên saler + cấp mã.
CREATE TABLE IF NOT EXISTS affiliate_requests (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending', -- pending | approved | rejected
    note        TEXT NOT NULL DEFAULT '',
    reviewed_by UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_affiliate_requests_status ON affiliate_requests(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS affiliate_requests;
-- Lưu ý: KHÔNG gỡ enum value 'customer' (Postgres không hỗ trợ DROP VALUE).
-- +goose StatementEnd
