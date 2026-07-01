-- +goose Up
-- +goose StatementBegin

CREATE TYPE contract_status   AS ENUM ('draft', 'signed', 'void');
CREATE TYPE investment_status AS ENUM ('pending', 'reconciled', 'approved', 'rejected');

-- Digital contract, signed by OTP, produces a PDF.
CREATE TABLE contracts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier_id          UUID            NOT NULL REFERENCES investment_tiers(id),
    pdf_url          TEXT,
    signature_otp_ref TEXT,                       -- reference to the OTP challenge used to sign
    signed_at        TIMESTAMPTZ,
    status           contract_status NOT NULL DEFAULT 'draft',
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT contract_signed_complete CHECK (
        (status = 'signed') = (signed_at IS NOT NULL AND signature_otp_ref IS NOT NULL)
    )
);
CREATE INDEX idx_contracts_user ON contracts(user_id);

-- An investment request. Money & shares are NOT recognised here — only after reconcile+approve.
-- INVARIANT 2: status flows pending -> reconciled -> approved (or -> rejected). Each step stamps an actor.
CREATE TABLE investments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code          TEXT              NOT NULL UNIQUE,   -- 'HKG-INV-xxxx'
    user_id       UUID              NOT NULL REFERENCES users(id),
    contract_id   UUID              NOT NULL REFERENCES contracts(id),
    tier_id       UUID              NOT NULL REFERENCES investment_tiers(id),
    amount_vnd    BIGINT            NOT NULL,
    shares        BIGINT            NOT NULL,
    status        investment_status NOT NULL DEFAULT 'pending',
    -- idempotency: client passes a key when creating; protects double-submit.
    idempotency_key TEXT            UNIQUE,
    reconciled_by UUID              REFERENCES users(id),
    reconciled_at TIMESTAMPTZ,
    approved_by   UUID              REFERENCES users(id),
    approved_at   TIMESTAMPTZ,
    rejected_by   UUID              REFERENCES users(id),
    rejected_at   TIMESTAMPTZ,
    reject_reason TEXT,
    created_at    TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ       NOT NULL DEFAULT now(),
    CONSTRAINT inv_amount_pos CHECK (amount_vnd > 0),
    CONSTRAINT inv_shares_pos CHECK (shares > 0),
    -- a reconciled (or beyond) investment must carry reconcile actor+ts
    CONSTRAINT inv_reconcile_stamp CHECK (
        (status IN ('reconciled','approved')) <= (reconciled_by IS NOT NULL AND reconciled_at IS NOT NULL)
    ),
    -- an approved investment must carry approve actor+ts
    CONSTRAINT inv_approve_stamp CHECK (
        (status = 'approved') = (approved_by IS NOT NULL AND approved_at IS NOT NULL)
    ),
    CONSTRAINT inv_reject_stamp CHECK (
        (status = 'rejected') = (rejected_by IS NOT NULL AND rejected_at IS NOT NULL)
    )
);
CREATE INDEX idx_investments_user   ON investments(user_id);
CREATE INDEX idx_investments_status ON investments(status);

-- Payment record. Money ALWAYS lands in a company (legal-entity) account — never personal.
-- INVARIANT 7: no money entry exists without a reconciled payment row.
CREATE TABLE payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id   UUID        NOT NULL UNIQUE REFERENCES investments(id) ON DELETE CASCADE,
    bank            TEXT        NOT NULL,
    company_account TEXT        NOT NULL,         -- company bank account number (legal entity)
    company_account_name TEXT   NOT NULL,         -- legal entity name (no personal accounts)
    amount_vnd      BIGINT      NOT NULL,
    transfer_note   TEXT        NOT NULL,         -- expected memo, e.g. the HKG-INV code
    declared_at     TIMESTAMPTZ,                  -- when investor clicked "I have transferred"
    reconciled_at   TIMESTAMPTZ,                  -- when admin confirmed funds arrived
    reconciled_by   UUID        REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pay_amount_pos CHECK (amount_vnd > 0),
    CONSTRAINT pay_reconcile_stamp CHECK (
        (reconciled_at IS NULL) = (reconciled_by IS NULL)
    )
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE payments;
DROP TABLE investments;
DROP TABLE contracts;
DROP TYPE investment_status;
DROP TYPE contract_status;
-- +goose StatementEnd
