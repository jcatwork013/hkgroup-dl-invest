-- +goose Up
-- +goose StatementBegin

-- The placement being offered. Single active offering for v2, but modelled as a table.
-- INVARIANT (DB level): shares_for_sale <= total_shares ; 0 <= shares_sold <= shares_for_sale.
-- NOTE: deliberately NO target_return / guaranteed_return / expected_profit / roi columns.
CREATE TABLE offering (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    valuation_vnd   BIGINT      NOT NULL,        -- company valuation in VND (integer, no cents)
    total_shares    BIGINT      NOT NULL,        -- 100% of equity
    shares_for_sale BIGINT      NOT NULL,        -- pool offered in this placement
    shares_sold     BIGINT      NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'open',  -- open | closed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT offering_valuation_pos    CHECK (valuation_vnd > 0),
    CONSTRAINT offering_total_pos        CHECK (total_shares > 0),
    CONSTRAINT offering_pool_within_cap  CHECK (shares_for_sale > 0 AND shares_for_sale <= total_shares),
    -- INVARIANT 1 (DB guard): cannot sell more than the pool, cannot go negative.
    CONSTRAINT offering_sold_within_pool CHECK (shares_sold >= 0 AND shares_sold <= shares_for_sale)
);

-- Fixed investment tiers. ownership_pct is derived (shares / total_shares) and stored for display.
-- amount_vnd == shares * price_per_share. No return/profit fields by design.
CREATE TABLE investment_tiers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offering_id   UUID          NOT NULL REFERENCES offering(id) ON DELETE CASCADE,
    name          TEXT          NOT NULL,
    amount_vnd    BIGINT        NOT NULL,
    shares        BIGINT        NOT NULL,
    ownership_pct NUMERIC(7,4)  NOT NULL,        -- e.g. 1.0000 = 1%
    sort_order    INT           NOT NULL DEFAULT 0,
    active        BOOLEAN       NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT tier_amount_pos CHECK (amount_vnd > 0),
    CONSTRAINT tier_shares_pos CHECK (shares > 0),
    CONSTRAINT tier_pct_range  CHECK (ownership_pct > 0 AND ownership_pct <= 100)
);
CREATE INDEX idx_tiers_offering ON investment_tiers(offering_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE investment_tiers;
DROP TABLE offering;
-- +goose StatementEnd
