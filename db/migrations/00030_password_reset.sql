-- +goose Up
-- +goose StatementBegin

-- ĐẶT LẠI MẬT KHẨU QUA EMAIL: lưu token reset (chỉ lưu HASH sha256, không lưu token thô).
-- Token sống ngắn (mặc định 1 giờ, do backend đặt) và DÙNG 1 LẦN (used_at). Khi user
-- yêu cầu reset mới, các token cũ chưa dùng của họ bị vô hiệu (xoá) để tránh tồn đọng.
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user ON password_reset_tokens (user_id);

-- Cấu hình Resend (gửi email) — khai báo key trong site_settings để admin chỉnh ở Thiết lập.
-- resend_api_key là BÍ MẬT: không bao giờ trả về qua endpoint công khai (xử lý ở backend).
-- Để trống = tính năng đặt lại mật khẩu KHÔNG hoạt động.
INSERT INTO site_settings (key, value) VALUES
    ('resend_api_key', ''),
    ('resend_from_email', ''),
    ('resend_from_name', ''),
    ('app_base_url', '')
ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS password_reset_tokens;
DELETE FROM site_settings WHERE key IN ('resend_api_key', 'resend_from_email', 'resend_from_name', 'app_base_url');
-- +goose StatementEnd
