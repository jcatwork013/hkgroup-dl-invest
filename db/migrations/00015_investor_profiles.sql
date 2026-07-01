-- +goose Up
-- +goose StatementBegin

-- Full investor profile (shareholder register grade): identity, address, tax, payout bank account.
-- 1:1 with users. Text fields kept simple/nullable-as-empty for an easy API.
CREATE TABLE investor_profiles (
    user_id             UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    date_of_birth       TEXT NOT NULL DEFAULT '',  -- YYYY-MM-DD
    gender              TEXT NOT NULL DEFAULT '',  -- male | female | other
    nationality         TEXT NOT NULL DEFAULT 'Việt Nam',
    cccd_number         TEXT NOT NULL DEFAULT '',  -- số CCCD/CMND
    cccd_issue_date     TEXT NOT NULL DEFAULT '',
    cccd_issue_place    TEXT NOT NULL DEFAULT '',
    permanent_address   TEXT NOT NULL DEFAULT '',  -- địa chỉ thường trú
    contact_address     TEXT NOT NULL DEFAULT '',  -- địa chỉ liên hệ
    occupation          TEXT NOT NULL DEFAULT '',  -- nghề nghiệp
    tax_code            TEXT NOT NULL DEFAULT '',  -- mã số thuế (khấu trừ TNCN)
    bank_name           TEXT NOT NULL DEFAULT '',  -- TK nhận cổ tức
    bank_account_number TEXT NOT NULL DEFAULT '',
    bank_account_name   TEXT NOT NULL DEFAULT '',
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE investor_profiles;
-- +goose StatementEnd
