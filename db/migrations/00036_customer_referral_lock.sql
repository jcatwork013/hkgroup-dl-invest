-- +goose Up
-- +goose StatementBegin
-- Khoá giới thiệu FIRST-TOUCH: đơn ĐẦU TIÊN của một khách (theo SĐT) qua link affiliate nào thì
-- khách đó vĩnh viễn thuộc affiliate đó. Các đơn sau luôn ghi hoa hồng về đúng người đã giới thiệu,
-- kể cả khi khách bấm link người khác. Khoá theo customer_phone (checkout là khách vãng lai).
CREATE TABLE IF NOT EXISTS customer_referral_locks (
    customer_phone TEXT PRIMARY KEY,
    referrer_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    locked_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_customer_referral_locks_referrer ON customer_referral_locks(referrer_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS customer_referral_locks;
-- +goose StatementEnd
