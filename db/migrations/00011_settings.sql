-- +goose Up
-- +goose StatementBegin

-- Editable site-wide settings (contact info, brand year...). Managed by admin, shown publicly.
CREATE TABLE site_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT        NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID        REFERENCES users(id)
);

INSERT INTO site_settings (key, value) VALUES
    ('contact_hotline', '0948 579 759'),
    ('contact_address', 'Số 18B1 Đường B1, khu dân cư Hưng Phú, phường Hưng Phú, TP Cần Thơ'),
    ('contact_email',   'info@duoclieuhk.vn'),
    ('brand_since',     '2026')
ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE site_settings;
-- +goose StatementEnd
