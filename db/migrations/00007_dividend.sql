-- +goose Up
-- +goose StatementBegin

-- INVARIANT 6: a dividend only exists when an admin creates this row. There is NO cron/auto
-- process anywhere that generates dividends. dividend_payouts.amount is the REAL paid amount.
-- This is the ONLY representation of "money received back" — there is no projected/target return.
CREATE TABLE dividends (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    declared_by   UUID        NOT NULL REFERENCES users(id),
    declared_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    period        TEXT        NOT NULL,         -- e.g. '2026-Q1'
    total_amount  BIGINT      NOT NULL,         -- total VND distributed
    note          TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dividend_total_pos CHECK (total_amount > 0)
);

CREATE TABLE dividend_payouts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dividend_id UUID        NOT NULL REFERENCES dividends(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id),
    shares      BIGINT      NOT NULL,           -- shareholding snapshot used to compute the split
    amount      BIGINT      NOT NULL,           -- REAL amount paid to this shareholder (dividend_paid)
    paid_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT payout_amount_nonneg CHECK (amount >= 0),
    CONSTRAINT uq_payout_per_user_per_dividend UNIQUE (dividend_id, user_id)
);
CREATE INDEX idx_payouts_user ON dividend_payouts(user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE dividend_payouts;
DROP TABLE dividends;
-- +goose StatementEnd
