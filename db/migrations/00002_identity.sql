-- +goose Up
-- +goose StatementBegin

CREATE TYPE user_role  AS ENUM ('investor', 'admin');
CREATE TYPE kyc_status AS ENUM ('unverified', 'pending', 'approved', 'rejected');

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name     TEXT        NOT NULL,
    phone         TEXT        NOT NULL UNIQUE,
    email         CITEXT      NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          user_role   NOT NULL DEFAULT 'investor',
    kyc_status    kyc_status  NOT NULL DEFAULT 'unverified',
    -- public referral handle so a user can share a 1-level link
    referral_code TEXT        NOT NULL UNIQUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_full_name_not_blank CHECK (length(btrim(full_name)) > 0)
);

-- eKYC: CCCD + selfie. Image URLs point at encrypted-at-rest object storage.
CREATE TABLE kyc_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cccd_number     TEXT        NOT NULL,
    cccd_image_url  TEXT        NOT NULL,
    selfie_url      TEXT        NOT NULL,
    status          kyc_status  NOT NULL DEFAULT 'pending',
    reject_reason   TEXT,
    reviewed_by     UUID        REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- a reviewed record must carry both reviewer and timestamp
    CONSTRAINT kyc_review_complete CHECK (
        (status IN ('approved','rejected')) = (reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL)
    )
);
CREATE INDEX idx_kyc_user   ON kyc_records(user_id);
CREATE INDEX idx_kyc_status ON kyc_records(status);

-- Consent log for Nghị định 13/2023 (personal data protection).
CREATE TABLE consents (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT        NOT NULL,            -- e.g. 'data_processing', 'risk_disclaimer'
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip         INET,
    user_agent TEXT
);
CREATE INDEX idx_consents_user ON consents(user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE consents;
DROP TABLE kyc_records;
DROP TABLE users;
DROP TYPE kyc_status;
DROP TYPE user_role;
-- +goose StatementEnd
