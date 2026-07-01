-- +goose Up
-- +goose StatementBegin

-- LỊCH RÚT TIỀN: nhà đầu tư/đại lý chỉ được GỬI yêu cầu rút từ ví hoa hồng vào các
-- "ngày rút" cố định trong tháng — mặc định ngày 15 và ngày 30 (mở cửa 00h00).
-- Admin chỉnh được danh sách ngày tại Thiết lập (key 'withdrawal_days', CSV "15,30").
-- Quy ước biên tháng: nếu tháng không có ngày cấu hình (vd 30/2) thì NGÀY CUỐI THÁNG
-- được tính là ngày rút thay thế (xử lý ở backend). Idempotent.
INSERT INTO site_settings (key, value) VALUES
    ('withdrawal_days', '15,30')
ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM site_settings WHERE key = 'withdrawal_days';
-- +goose StatementEnd
