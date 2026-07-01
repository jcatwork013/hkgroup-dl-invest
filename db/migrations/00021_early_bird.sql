-- +goose Up
-- +goose StatementBegin

-- "Early-bird" / whitelist packages. A special tier is a RECORD-ONLY reservation: it lets a
-- whitelisted investor (one referred by an admin) reserve a package early WITHOUT paying and
-- WITHOUT touching the real cap-table. It NEVER issues shares, NEVER bumps offering.shares_sold,
-- NEVER creates a dividend or a commission. The 9.9 tỷ total supply is unaffected by design.
ALTER TABLE investment_tiers ADD COLUMN is_special BOOLEAN NOT NULL DEFAULT false;

-- Early-bird reservations live in their OWN table, fully decoupled from investments/shareholdings/
-- offering. No immutability trigger, no money entry, no share_ledger row. Pure display record.
CREATE TYPE reservation_status AS ENUM ('reserved', 'cancelled');

CREATE TABLE early_reservations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID               NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier_id       UUID               NOT NULL REFERENCES investment_tiers(id),
    -- snapshot of the tier at reservation time (for display; tier may change later)
    amount_vnd    BIGINT             NOT NULL,
    shares        BIGINT             NOT NULL,
    ownership_pct NUMERIC(7,4)       NOT NULL,
    status        reservation_status NOT NULL DEFAULT 'reserved',
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),
    CONSTRAINT reservation_amount_pos CHECK (amount_vnd > 0),
    CONSTRAINT reservation_shares_pos CHECK (shares > 0),
    -- one active reservation per (user, tier): re-reserving is idempotent at the app layer.
    CONSTRAINT uq_reservation_user_tier UNIQUE (user_id, tier_id)
);
CREATE INDEX idx_early_reservations_user ON early_reservations(user_id);
CREATE INDEX idx_early_reservations_tier ON early_reservations(tier_id);

-- Seed one early-bird package on the active offering. is_special=true => only admin-referred
-- users see it; it never consumes the 490k-share sale pool.
INSERT INTO investment_tiers (offering_id, name, amount_vnd, shares, ownership_pct, sort_order, is_special)
SELECT '00000000-0000-0000-0000-0000000000ff', 'Gói Early-bird (Đăng ký sớm)', 99000000, 10000, 1.0000, 99, true
WHERE EXISTS (SELECT 1 FROM offering WHERE id = '00000000-0000-0000-0000-0000000000ff');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE early_reservations;
DROP TYPE reservation_status;
DELETE FROM investment_tiers WHERE is_special = true;
ALTER TABLE investment_tiers DROP COLUMN is_special;
-- +goose StatementEnd
