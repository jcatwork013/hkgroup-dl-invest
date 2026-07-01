-- +goose Up
-- +goose StatementBegin

-- Repackage the offering: 6 fixed tiers at a clean 10,000 VND/share.
-- valuation 10 tỷ, total 1,000,000 cổ phần, chào bán 49% (490,000). % sở hữu = shares / total * 100.
UPDATE offering
SET valuation_vnd = 10000000000, total_shares = 1000000, shares_for_sale = 490000, updated_at = now()
WHERE id = '00000000-0000-0000-0000-0000000000ff';

DELETE FROM investment_tiers WHERE offering_id = '00000000-0000-0000-0000-0000000000ff';

INSERT INTO investment_tiers (offering_id, name, amount_vnd, shares, ownership_pct, sort_order)
VALUES
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 1',  10000000::bigint,   1000::bigint, 0.1000::numeric, 1),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 2',  30000000::bigint,   3000::bigint, 0.3000::numeric, 2),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 3',  50000000::bigint,   5000::bigint, 0.5000::numeric, 3),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 4', 100000000::bigint,  10000::bigint, 1.0000::numeric, 4),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 5', 300000000::bigint,  30000::bigint, 3.0000::numeric, 5),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 6', 500000000::bigint,  50000::bigint, 5.0000::numeric, 6);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE offering
SET valuation_vnd = 9900000000, total_shares = 1000000, shares_for_sale = 490000, updated_at = now()
WHERE id = '00000000-0000-0000-0000-0000000000ff';

DELETE FROM investment_tiers WHERE offering_id = '00000000-0000-0000-0000-0000000000ff';

INSERT INTO investment_tiers (offering_id, name, amount_vnd, shares, ownership_pct, sort_order)
VALUES
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 1%',           99000000::bigint,  10000::bigint,  1.0000::numeric, 1),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 5%',          495000000::bigint,  50000::bigint,  5.0000::numeric, 2),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 10%',         990000000::bigint, 100000::bigint, 10.0000::numeric, 3),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 20%',        1980000000::bigint, 200000::bigint, 20.0000::numeric, 4),
    ('00000000-0000-0000-0000-0000000000ff', 'Gói 49% (toàn bộ)', 4851000000::bigint, 490000::bigint, 49.0000::numeric, 5);

-- +goose StatementEnd
