-- +goose Up
-- +goose StatementBegin

-- Seed the single active offering: valuation 9.9 tỷ VND, 49% of equity for sale.
-- total_shares = 1,000,000 (= 100%). price/share = 9,900,000,000 / 1,000,000 = 9,900 VND.
-- shares_for_sale = 490,000 (= 49%).
WITH o AS (
    INSERT INTO offering (id, name, valuation_vnd, total_shares, shares_for_sale, shares_sold, status)
    VALUES (
        '00000000-0000-0000-0000-0000000000ff',
        'HKGroup — Chào bán cổ phần riêng lẻ 2026',
        9900000000, 1000000, 490000, 0, 'open'
    )
    RETURNING id
)
INSERT INTO investment_tiers (offering_id, name, amount_vnd, shares, ownership_pct, sort_order)
SELECT o.id, t.name, t.amount_vnd, t.shares, t.ownership_pct, t.sort_order
FROM o, (VALUES
    ('Gói 1%',          99000000::bigint,    10000::bigint,  1.0000::numeric, 1),
    ('Gói 5%',         495000000::bigint,    50000::bigint,  5.0000::numeric, 2),
    ('Gói 10%',        990000000::bigint,   100000::bigint, 10.0000::numeric, 3),
    ('Gói 20%',       1980000000::bigint,   200000::bigint, 20.0000::numeric, 4),
    ('Gói 49% (toàn bộ)', 4851000000::bigint, 490000::bigint, 49.0000::numeric, 5)
) AS t(name, amount_vnd, shares, ownership_pct, sort_order);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM investment_tiers WHERE offering_id = '00000000-0000-0000-0000-0000000000ff';
DELETE FROM offering WHERE id = '00000000-0000-0000-0000-0000000000ff';
-- +goose StatementEnd
