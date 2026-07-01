-- +goose Up
-- +goose StatementBegin

-- Remove the early-bird / whitelist feature entirely. Packages are now ONLY the normal,
-- real investment flow (contract + OTP + VietQR transfer + real share issuance into the pool).
-- The record-only reservation system and the seeded early-bird tier are dropped.
--
-- NOTE: the investment_tiers.is_special column is intentionally LEFT in place (default false)
-- so the existing sqlc-generated row scans keep working. No tier is special anymore; the admin
-- UI no longer exposes the flag, so it stays false forever.
DELETE FROM early_reservations;
DELETE FROM investment_tiers WHERE is_special = true;

DROP TABLE IF EXISTS early_reservations;
DROP TYPE IF EXISTS reservation_status;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Best-effort restore of the early-bird subsystem (mirrors 00021_early_bird.sql).
CREATE TYPE reservation_status AS ENUM ('reserved', 'cancelled');

CREATE TABLE early_reservations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID               NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier_id       UUID               NOT NULL REFERENCES investment_tiers(id),
    amount_vnd    BIGINT             NOT NULL,
    shares        BIGINT             NOT NULL,
    ownership_pct NUMERIC(7,4)       NOT NULL,
    status        reservation_status NOT NULL DEFAULT 'reserved',
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),
    CONSTRAINT reservation_amount_pos CHECK (amount_vnd > 0),
    CONSTRAINT reservation_shares_pos CHECK (shares > 0),
    CONSTRAINT uq_reservation_user_tier UNIQUE (user_id, tier_id)
);
CREATE INDEX idx_early_reservations_user ON early_reservations(user_id);
CREATE INDEX idx_early_reservations_tier ON early_reservations(tier_id);

INSERT INTO investment_tiers (offering_id, name, amount_vnd, shares, ownership_pct, sort_order, is_special)
SELECT '00000000-0000-0000-0000-0000000000ff', 'Gói Early-bird (Đăng ký sớm)', 99000000, 10000, 1.0000, 99, true
WHERE EXISTS (SELECT 1 FROM offering WHERE id = '00000000-0000-0000-0000-0000000000ff');

-- +goose StatementEnd
