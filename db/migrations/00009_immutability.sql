-- +goose Up
-- +goose StatementBegin

-- Generic guard: reject UPDATE/DELETE on append-only tables.
CREATE OR REPLACE FUNCTION trg_block_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'table % is append-only: % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;

-- audit_logs: fully immutable (no update, no delete).
CREATE TRIGGER audit_logs_no_update BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION trg_block_mutation();
CREATE TRIGGER audit_logs_no_delete BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION trg_block_mutation();

-- share_ledger: append-only (corrections are made by inserting a compensating delta).
CREATE TRIGGER share_ledger_no_update BEFORE UPDATE ON share_ledger
    FOR EACH ROW EXECUTE FUNCTION trg_block_mutation();
CREATE TRIGGER share_ledger_no_delete BEFORE DELETE ON share_ledger
    FOR EACH ROW EXECUTE FUNCTION trg_block_mutation();

-- HARD CONSTRAINT 3 (DB guard): a commission may only reference a 'customer' referral.
-- Cash commission is never created for 'investor' referrals at the data layer.
CREATE OR REPLACE FUNCTION trg_commission_customer_only() RETURNS trigger AS $$
DECLARE
    rtype referral_type;
BEGIN
    SELECT referral_type INTO rtype FROM referrals WHERE referee_id = NEW.referral_id;
    IF rtype IS DISTINCT FROM 'customer' THEN
        RAISE EXCEPTION 'commission allowed only for referral_type=customer (got %)', rtype
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER commissions_customer_only BEFORE INSERT ON commissions
    FOR EACH ROW EXECUTE FUNCTION trg_commission_customer_only();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER commissions_customer_only ON commissions;
DROP FUNCTION trg_commission_customer_only;
DROP TRIGGER share_ledger_no_delete ON share_ledger;
DROP TRIGGER share_ledger_no_update ON share_ledger;
DROP TRIGGER audit_logs_no_delete ON audit_logs;
DROP TRIGGER audit_logs_no_update ON audit_logs;
DROP FUNCTION trg_block_mutation;
-- +goose StatementEnd
