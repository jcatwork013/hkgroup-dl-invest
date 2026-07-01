-- +goose Up
-- +goose StatementBegin

CREATE TYPE referral_type     AS ENUM ('customer', 'investor');
CREATE TYPE commission_status AS ENUM ('pending', 'approved', 'paid', 'rejected');

-- 1-LEVEL ONLY referral. We store the DIRECT referrer and nothing else.
-- HARD CONSTRAINT 2: no closure table, no parent-of-parent, no F2/F3. A referee has at most
-- one referrer (PRIMARY KEY on referee_id), and cannot refer themselves.
CREATE TABLE referrals (
    referee_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,  -- exactly one referrer max
    referrer_id   UUID          NOT NULL REFERENCES users(id),
    referral_type referral_type NOT NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT referral_no_self CHECK (referee_id <> referrer_id)
);
CREATE INDEX idx_referrals_referrer ON referrals(referrer_id);

-- Commission is cash only for referral_type='customer'. For 'investor' it is record-only by default.
-- HARD CONSTRAINT 3 is enforced in the usecase (config flag, default off) AND guarded here:
-- a commission row must reference a referral whose type is 'customer'. INVARIANT 5: only on
-- approved investments, 1 level. PIT (thuế TNCN) withheld at payout.
CREATE TABLE commissions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id   UUID              NOT NULL REFERENCES referrals(referee_id) ON DELETE CASCADE,
    investment_id UUID              NOT NULL REFERENCES investments(id),
    base_amount   BIGINT            NOT NULL,        -- the approved investment amount
    rate          NUMERIC(5,4)      NOT NULL,        -- e.g. 0.0500 = 5%
    amount        BIGINT            NOT NULL,        -- gross commission
    tax_pit       BIGINT            NOT NULL DEFAULT 0,  -- PIT withheld (thuế TNCN)
    net_amount    BIGINT            NOT NULL,        -- amount - tax_pit
    status        commission_status NOT NULL DEFAULT 'pending',
    approved_by   UUID              REFERENCES users(id),
    paid_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ       NOT NULL DEFAULT now(),
    -- one commission per investment (idempotent generation)
    CONSTRAINT uq_commission_per_investment UNIQUE (investment_id),
    CONSTRAINT comm_amount_nonneg CHECK (amount >= 0 AND base_amount >= 0 AND net_amount >= 0),
    CONSTRAINT comm_net_consistent CHECK (net_amount = amount - tax_pit)
);
CREATE INDEX idx_commissions_referral ON commissions(referral_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE commissions;
DROP TABLE referrals;
DROP TYPE commission_status;
DROP TYPE referral_type;
-- +goose StatementEnd
