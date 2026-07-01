-- +goose Up
-- +goose StatementBegin

-- MULTI-LEVEL referral commission (F1/F2/F3). The 1-LEVEL `referrals` table is UNCHANGED: each
-- user still stores only their DIRECT referrer (PK referee_id). F2/F3 are DERIVED by walking up
-- that chain at commission time (referrer-of-referrer), NOT by a closure table. An investment can
-- now generate up to 3 commission rows (one per upline level), each crediting a different earner.
--   level 1 (F1) = direct referrer            — rate as before (customer per-tier / investor F1)
--   level 2 (F2) = referrer's referrer        — referral_f2_rate (both customer & investor)
--   level 3 (F3) = referrer's referrer's ...  — referral_f3_rate (both customer & investor)
ALTER TABLE commissions
    ADD COLUMN level          SMALLINT,
    ADD COLUMN beneficiary_id UUID REFERENCES users(id);

-- Backfill existing rows: they are all F1, earner = the referee's direct referrer.
UPDATE commissions c
SET level = 1,
    beneficiary_id = r.referrer_id
FROM referrals r
WHERE r.referee_id = c.referral_id;

-- Safety net for any orphan row (should be none): treat as F1 self-credit-less.
UPDATE commissions SET level = 1 WHERE level IS NULL;

ALTER TABLE commissions
    ALTER COLUMN level SET DEFAULT 1,
    ALTER COLUMN level SET NOT NULL,
    ALTER COLUMN beneficiary_id SET NOT NULL;

ALTER TABLE commissions ADD CONSTRAINT comm_level_range CHECK (level BETWEEN 1 AND 3);

-- One commission per (investment, level) instead of per investment (idempotent generation).
ALTER TABLE commissions DROP CONSTRAINT uq_commission_per_investment;
ALTER TABLE commissions ADD CONSTRAINT uq_commission_per_investment_level UNIQUE (investment_id, level);

CREATE INDEX idx_commissions_beneficiary ON commissions(beneficiary_id);

-- F2/F3 rates (F1 unchanged). Apply to BOTH customer and investor referrals.
INSERT INTO site_settings (key, value) VALUES
    ('referral_f2_rate', '0.02'),   -- F2 = 2%
    ('referral_f3_rate', '0.01')    -- F3 = 1%
ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_commissions_beneficiary;
ALTER TABLE commissions DROP CONSTRAINT IF EXISTS uq_commission_per_investment_level;
ALTER TABLE commissions ADD CONSTRAINT uq_commission_per_investment UNIQUE (investment_id);
ALTER TABLE commissions DROP CONSTRAINT IF EXISTS comm_level_range;
ALTER TABLE commissions DROP COLUMN beneficiary_id;
ALTER TABLE commissions DROP COLUMN level;
DELETE FROM site_settings WHERE key IN ('referral_f2_rate', 'referral_f3_rate');
-- +goose StatementEnd
