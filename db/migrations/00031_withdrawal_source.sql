-- +goose Up
-- +goose StatementBegin

-- VÍ CỔ TỨC: rút cổ tức là số dư RIÊNG, song song ví hoa hồng. Phân biệt nguồn tiền của
-- mỗi yêu cầu rút bằng cột `source`: 'commission' (ví hoa hồng) | 'dividend' (cổ tức).
-- Mọi bản ghi cũ mặc định 'commission' (giữ nguyên ý nghĩa trước đây).
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'commission';

ALTER TABLE withdrawals DROP CONSTRAINT IF EXISTS withdrawals_source_check;
ALTER TABLE withdrawals ADD CONSTRAINT withdrawals_source_check
    CHECK (source IN ('commission', 'dividend'));

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_source ON withdrawals (user_id, source);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_withdrawals_user_source;
ALTER TABLE withdrawals DROP CONSTRAINT IF EXISTS withdrawals_source_check;
ALTER TABLE withdrawals DROP COLUMN IF EXISTS source;
-- +goose StatementEnd
