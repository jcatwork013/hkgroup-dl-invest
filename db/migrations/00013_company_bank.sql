-- +goose Up
-- +goose StatementBegin

-- Company (legal-entity) receiving account — admin-editable, shown in the transfer step + VietQR.
-- HARD CONSTRAINT 4: this is a COMPANY account only. Seeded with placeholders; admin must set the
-- real legal-entity account. (Never a personal account.)
INSERT INTO site_settings (key, value) VALUES
    ('company_bank_code',    'VCB'),                       -- VietQR bank code/BIN (e.g. VCB, VPB, TCB, 970436)
    ('company_bank_name',    'Vietcombank'),               -- display name
    ('company_account',      '0123456789'),                -- COMPANY account number (placeholder)
    ('company_account_name', 'CONG TY CO PHAN DUOC LIEU HK')
ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM site_settings WHERE key IN
    ('company_bank_code', 'company_bank_name', 'company_account', 'company_account_name');
-- +goose StatementEnd
