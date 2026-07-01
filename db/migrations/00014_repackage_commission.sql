-- +goose Up
-- +goose StatementBegin

-- Per-tier referral commission rate (customer referrals only). Default keeps the old global 5%.
ALTER TABLE investment_tiers
    ADD COLUMN IF NOT EXISTS commission_rate NUMERIC(5,4) NOT NULL DEFAULT 0.05;

-- Valuation 9.9 tỷ, clean 10,000 VND/share => total 990,000 cổ phần, chào bán 49% = 485,100.
UPDATE offering
SET valuation_vnd = 9900000000, total_shares = 990000, shares_for_sale = 485100, updated_at = now()
WHERE id = '00000000-0000-0000-0000-0000000000ff';

DELETE FROM investment_tiers WHERE offering_id = '00000000-0000-0000-0000-0000000000ff';

-- amount = shares * 10,000 ; ownership_pct = shares / 990,000 * 100.
-- commission_rate by package band (khách yêu cầu): 5–50tr=15%, >50–100tr=20%, 300–500tr=25%.
INSERT INTO investment_tiers
    (offering_id, name, amount_vnd, shares, ownership_pct, commission_rate, sort_order)
VALUES
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 1',   5000000::bigint,    500::bigint, 0.0505::numeric, 0.1500::numeric, 1),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 2',  20000000::bigint,   2000::bigint, 0.2020::numeric, 0.1500::numeric, 2),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 3',  50000000::bigint,   5000::bigint, 0.5051::numeric, 0.1500::numeric, 3),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 4', 100000000::bigint,  10000::bigint, 1.0101::numeric, 0.2000::numeric, 4),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 5', 300000000::bigint,  30000::bigint, 3.0303::numeric, 0.2500::numeric, 5),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 6', 500000000::bigint,  50000::bigint, 5.0505::numeric, 0.2500::numeric, 6);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE investment_tiers DROP COLUMN IF EXISTS commission_rate;
-- +goose StatementEnd
