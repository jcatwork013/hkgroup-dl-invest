-- +goose Up
-- +goose StatementBegin

-- Append-only ledger of share issuance. The SINGLE source of truth for who owns what.
-- INVARIANT 3: every insert here happens in the same tx as the shareholdings + offering update.
-- INVARIANT 4: shareholdings.shares of a user == SUM(share_ledger.shares_delta) of that user.
CREATE TABLE share_ledger (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id),
    investment_id UUID        REFERENCES investments(id),  -- the approved investment that issued shares
    shares_delta  BIGINT      NOT NULL,                    -- +issue, -buyback (always tied to a decision)
    reason        TEXT        NOT NULL,                    -- e.g. 'issue:approved-investment'
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ledger_delta_nonzero CHECK (shares_delta <> 0)
);
CREATE INDEX idx_ledger_user ON share_ledger(user_id);
-- one issuance entry per investment (idempotent issue)
CREATE UNIQUE INDEX uq_ledger_issue_per_investment
    ON share_ledger(investment_id) WHERE investment_id IS NOT NULL;

-- Read model / cap-table projection. Kept in lock-step with share_ledger inside the issuing tx.
CREATE TABLE shareholdings (
    user_id       UUID PRIMARY KEY REFERENCES users(id),
    shares        BIGINT       NOT NULL DEFAULT 0,
    ownership_pct NUMERIC(7,4) NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT holding_shares_nonneg CHECK (shares >= 0),
    CONSTRAINT holding_pct_range     CHECK (ownership_pct >= 0 AND ownership_pct <= 100)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE shareholdings;
DROP TABLE share_ledger;
-- +goose StatementEnd
