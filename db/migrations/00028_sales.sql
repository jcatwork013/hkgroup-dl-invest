-- +goose Up
-- +goose StatementBegin

-- ===========================================================================
-- MODULE BÁN HÀNG — tách biệt HOÀN TOÀN với đầu tư cổ phần.
--  • Role mới `saler`: tài khoản bán hàng (khác `investor` của đầu tư).
--  • Danh mục + sản phẩm (catalog).
--  • Đơn hàng → khi `paid` chia 6 khoản: người bán 25% · affiliate 10% ·
--    đồng chia 5% (đơn ≥ 1tr) · pool cổ đông 15% · giá vốn 30% · vận hành 15%.
--  • Hoa hồng bán (seller/affiliate) vào VÍ CHUNG (cộng với hoa hồng đầu tư).
--  • 20% (5%+15%) tích luỹ vào pool bán hàng để admin gộp vào cổ tức.
-- ===========================================================================

-- Role bán hàng. PG16 cho phép ADD VALUE trong transaction miễn không dùng ngay trong cùng tx.
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'saler';

-- ---- Danh mục sản phẩm ----
CREATE TABLE product_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INT  NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Sản phẩm ----
CREATE TABLE products (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id   UUID REFERENCES product_categories(id) ON DELETE SET NULL,
    sku           TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    badge         TEXT NOT NULL DEFAULT '',                 -- nhãn nhỏ (vd "299K")
    price_vnd     BIGINT NOT NULL CHECK (price_vnd >= 0),   -- giá bán
    cost_vnd      BIGINT NOT NULL DEFAULT 0 CHECK (cost_vnd >= 0), -- giá vốn (≈30%)
    image_url     TEXT NOT NULL DEFAULT '',
    summary       TEXT NOT NULL DEFAULT '',                 -- mô tả ngắn
    description   TEXT NOT NULL DEFAULT '',                 -- mô tả chi tiết
    spec_warranty TEXT NOT NULL DEFAULT 'Chính hãng 100%',  -- Cam kết
    spec_trace    TEXT NOT NULL DEFAULT 'Theo từng lô',     -- Truy xuất
    spec_delivery TEXT NOT NULL DEFAULT 'Hub theo khu vực', -- Giao hàng
    spec_return   TEXT NOT NULL DEFAULT 'Trong 7 ngày',     -- Đổi trả
    active        BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_products_category ON products(category_id);

-- ---- Đơn hàng bán ----
CREATE TYPE sales_order_status AS ENUM ('pending', 'paid', 'cancelled');

CREATE TABLE sales_orders (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code           TEXT NOT NULL UNIQUE,                    -- HKG-SO-XXXXXX
    customer_name  TEXT NOT NULL DEFAULT '',
    customer_phone TEXT NOT NULL DEFAULT '',
    seller_id      UUID NOT NULL REFERENCES users(id),      -- nhận 25%
    affiliate_id   UUID REFERENCES users(id),               -- nhận 10% (nullable)
    subtotal_vnd   BIGINT NOT NULL CHECK (subtotal_vnd >= 0), -- tổng giá bán
    cost_vnd       BIGINT NOT NULL DEFAULT 0,               -- tổng giá vốn
    status         sales_order_status NOT NULL DEFAULT 'pending',
    note           TEXT NOT NULL DEFAULT '',
    created_by     UUID NOT NULL REFERENCES users(id),      -- saler hoặc admin
    paid_by        UUID REFERENCES users(id),
    paid_at        TIMESTAMPTZ,
    cancelled_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sales_orders_seller ON sales_orders(seller_id);
CREATE INDEX idx_sales_orders_status ON sales_orders(status);

CREATE TABLE sales_order_items (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       UUID NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    product_id     UUID NOT NULL REFERENCES products(id),
    name           TEXT NOT NULL,                           -- snapshot tên sản phẩm
    qty            BIGINT NOT NULL CHECK (qty > 0),
    unit_price_vnd BIGINT NOT NULL,
    unit_cost_vnd  BIGINT NOT NULL DEFAULT 0,
    line_total_vnd BIGINT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sales_order_items_order ON sales_order_items(order_id);

-- ---- Breakdown 6 khoản khi đơn paid (minh bạch + audit) ----
CREATE TABLE sales_distributions (
    order_id          UUID PRIMARY KEY REFERENCES sales_orders(id) ON DELETE CASCADE,
    total_vnd         BIGINT NOT NULL,
    seller_vnd        BIGINT NOT NULL,   -- 25%
    affiliate_vnd     BIGINT NOT NULL,   -- 10%
    equal_share_vnd   BIGINT NOT NULL,   -- 5% (0 nếu đơn < 1.000.000đ)
    pool_vnd          BIGINT NOT NULL,   -- 15%
    cost_vnd          BIGINT NOT NULL,   -- 30%
    operations_vnd    BIGINT NOT NULL,   -- 15%
    dividend_pool_vnd BIGINT NOT NULL,   -- equal_share + pool (phần vào pool cổ tức)
    swept             BOOLEAN NOT NULL DEFAULT false, -- đã gộp vào 1 đợt cổ tức chưa
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---- Hoa hồng bán hàng (seller + affiliate) — tách bảng khỏi commissions đầu tư ----
CREATE TYPE sales_commission_kind AS ENUM ('seller', 'affiliate');

CREATE TABLE sales_commissions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       UUID NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    beneficiary_id UUID NOT NULL REFERENCES users(id),
    kind           sales_commission_kind NOT NULL,
    base_amount    BIGINT NOT NULL,                 -- subtotal đơn
    rate           NUMERIC(5,4) NOT NULL,           -- 0.2500 / 0.1000
    amount         BIGINT NOT NULL,                 -- gross
    tax_pit        BIGINT NOT NULL DEFAULT 0,       -- 10% TNCN
    net_amount     BIGINT NOT NULL,                 -- gross − thuế
    status         commission_status NOT NULL DEFAULT 'pending',  -- reuse enum đầu tư
    approved_by    UUID REFERENCES users(id),
    paid_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sales_commission UNIQUE (order_id, kind)
);
CREATE INDEX idx_sales_commissions_beneficiary ON sales_commissions(beneficiary_id);

-- Tỷ lệ chia mặc định (admin có thể đổi qua site_settings, key dưới đây).
INSERT INTO site_settings (key, value) VALUES
    ('sales_seller_rate',     '0.25'),
    ('sales_affiliate_rate',  '0.10'),
    ('sales_equalshare_rate', '0.05'),
    ('sales_pool_rate',       '0.15'),
    ('sales_cost_rate',       '0.30'),
    ('sales_operations_rate', '0.15'),
    ('sales_equalshare_min',  '1000000')
ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sales_commissions;
DROP TYPE IF EXISTS sales_commission_kind;
DROP TABLE IF EXISTS sales_distributions;
DROP TABLE IF EXISTS sales_order_items;
DROP TABLE IF EXISTS sales_orders;
DROP TYPE IF EXISTS sales_order_status;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS product_categories;
DELETE FROM site_settings WHERE key IN (
    'sales_seller_rate','sales_affiliate_rate','sales_equalshare_rate',
    'sales_pool_rate','sales_cost_rate','sales_operations_rate','sales_equalshare_min'
);
-- Lưu ý: KHÔNG gỡ giá trị enum 'saler' khỏi user_role (Postgres không hỗ trợ DROP VALUE).
-- +goose StatementEnd
