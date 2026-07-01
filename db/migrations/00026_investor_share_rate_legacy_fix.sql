-- +goose Up
-- +goose StatementBegin

-- "Tỷ lệ chia cổ đông" thực tế = doanh thu × pool_rate (mặc định 15%); nhà đầu tư hưởng 100% Pool
-- Cổ Đông (xem distribution.go: InvestorShareRate hard-code = 1, KHÔNG nhân investor_share_rate).
-- Giá trị cũ investor_share_rate = 0.49 (seed 00016) là LEGACY, không dùng trong tính toán, nhưng
-- bị hiển thị nhầm thành "chia cổ đông 49%" → gây hiểu lầm. Đặt = 1 cho đúng bản chất (100% pool).
UPDATE site_settings SET value = '1' WHERE key = 'investor_share_rate';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE site_settings SET value = '0.49' WHERE key = 'investor_share_rate';
-- +goose StatementEnd
