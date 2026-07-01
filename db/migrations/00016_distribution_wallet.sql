-- +goose Up
-- +goose StatementBegin

-- Revenue distribution: admin nhập doanh thu kỳ -> hệ thống tính Pool Cổ Đông -> chia cổ tức THỰC
-- cho cổ đông theo tỷ lệ vốn. KHÔNG cam kết, KHÔNG mục tiêu/điểm dừng — là cổ tức biến động.
-- investor_pool = total_revenue * pool_rate * investor_share_rate.
CREATE TABLE revenue_distributions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period              TEXT        NOT NULL,
    total_revenue       BIGINT      NOT NULL,
    pool_rate           NUMERIC(5,4) NOT NULL,   -- Pool Cổ Đông (vd 0.15)
    investor_share_rate NUMERIC(5,4) NOT NULL,   -- tỷ lệ mở bán (vd 0.49)
    investor_pool       BIGINT      NOT NULL,     -- số tiền chia cho nhà đầu tư
    dividend_id         UUID        REFERENCES dividends(id),  -- cổ tức sinh ra từ phân bổ này
    created_by          UUID        NOT NULL REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rev_revenue_pos CHECK (total_revenue > 0)
);
CREATE INDEX idx_revdist_created ON revenue_distributions(created_at DESC);

-- Rút tiền từ ví hoa hồng giới thiệu (1 tầng). Admin duyệt & chi.
CREATE TYPE withdrawal_status AS ENUM ('pending', 'approved', 'paid', 'rejected');
CREATE TABLE withdrawals (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id),
    amount       BIGINT      NOT NULL,
    status       withdrawal_status NOT NULL DEFAULT 'pending',
    note         TEXT        NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_by UUID        REFERENCES users(id),
    processed_at TIMESTAMPTZ,
    CONSTRAINT wd_amount_pos CHECK (amount > 0)
);
CREATE INDEX idx_withdrawals_user ON withdrawals(user_id);
CREATE INDEX idx_withdrawals_status ON withdrawals(status);

-- Cho phép hoa hồng giới thiệu nhà đầu tư 1 TẦNG (F1). Vẫn 1 tầng (PK referee_id), không F2/F3.
-- (Nới ràng buộc "customer-only" trước đây để bật F1 cho referral nhà đầu tư.)
DROP TRIGGER IF EXISTS commissions_customer_only ON commissions;

-- Tham số cấu hình (admin sửa được).
INSERT INTO site_settings (key, value) VALUES
    ('pool_rate', '0.15'),               -- Pool Cổ Đông = 15% doanh thu
    ('investor_share_rate', '0.49'),     -- nhà đầu tư hưởng 49% pool
    ('referral_f1_rate', '0.03'),        -- hoa hồng giới thiệu 1 tầng (F1)
    ('referral_investor_cash', 'on'),    -- bật hoa hồng tiền mặt cho referral nhà đầu tư (1 tầng)
    ('show_pool_public', 'off')          -- switch hiển thị Pool ngoài trang invest
ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE withdrawals;
DROP TYPE withdrawal_status;
DROP TABLE revenue_distributions;
DELETE FROM site_settings WHERE key IN ('pool_rate','investor_share_rate','referral_f1_rate','referral_investor_cash');
CREATE TRIGGER commissions_customer_only BEFORE INSERT ON commissions
    FOR EACH ROW EXECUTE FUNCTION trg_commission_customer_only();
-- +goose StatementEnd
