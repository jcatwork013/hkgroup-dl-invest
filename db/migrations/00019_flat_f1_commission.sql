-- +goose Up
-- +goose StatementBegin

-- Chính sách hoa hồng giới thiệu thống nhất 3 cấp: F1 = 3%, F2 = 2%, F3 = 1% (khách yêu cầu).
-- Trước đây F1 (khách) dùng commission_rate theo gói (15–25% từ migration 00014). Nay F1 phẳng 3%
-- cho mọi gói, khớp với referral_f1_rate (0.03) đang dùng cho F1 nhà đầu tư. F2/F3 đã là 2%/1%.
UPDATE investment_tiers
SET commission_rate = 0.0300
WHERE offering_id = '00000000-0000-0000-0000-0000000000ff';

-- referral_f1_rate đã = 0.03 từ migration 00016; đặt lại cho chắc (đề phòng đã bị sửa).
UPDATE site_settings SET value = '0.03' WHERE key = 'referral_f1_rate';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Khôi phục commission_rate theo gói như migration 00014 (5–50tr=15%, >50–100tr=20%, 300–500tr=25%).
UPDATE investment_tiers SET commission_rate = 0.1500 WHERE offering_id = '00000000-0000-0000-0000-0000000000ff' AND name IN ('Gói 1', 'Gói 2', 'Gói 3');
UPDATE investment_tiers SET commission_rate = 0.2000 WHERE offering_id = '00000000-0000-0000-0000-0000000000ff' AND name = 'Gói 4';
UPDATE investment_tiers SET commission_rate = 0.2500 WHERE offering_id = '00000000-0000-0000-0000-0000000000ff' AND name IN ('Gói 5', 'Gói 6');
-- +goose StatementEnd
