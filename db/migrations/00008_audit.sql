-- +goose Up
-- +goose StatementBegin

-- INVARIANT 8: every approve / issue / payout writes here in the SAME transaction as the change.
-- Append-only & immutable: enforced by triggers in 00009. before/after capture the entity diff.
CREATE TABLE audit_logs (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id   UUID        REFERENCES users(id),   -- null = system/migration
    action     TEXT        NOT NULL,               -- e.g. 'investment.approve'
    entity     TEXT        NOT NULL,               -- e.g. 'investments'
    entity_id  TEXT        NOT NULL,
    before     JSONB,
    after      JSONB,
    ip         INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_entity ON audit_logs(entity, entity_id);
CREATE INDEX idx_audit_actor  ON audit_logs(actor_id);
CREATE INDEX idx_audit_created ON audit_logs(created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE audit_logs;
-- +goose StatementEnd
