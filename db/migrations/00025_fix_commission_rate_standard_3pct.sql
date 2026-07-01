-- +goose Up
-- +goose StatementBegin

-- BUG F1 = 5%: cột investment_tiers.commission_rate có DEFAULT 0.05 (migration 00014). Admin tạo gói
-- mới qua UI (CreateTier) KHÔNG set commission_rate nên gói mới mặc định 5% → trang "Hoa hồng của tôi"
-- hiện F1 khách = 5.00% thay vì 3%. Sửa default về 0.03 (chuẩn F1=3%) cho mọi gói tạo về sau.
ALTER TABLE investment_tiers ALTER COLUMN commission_rate SET DEFAULT 0.0300;

-- Sửa dữ liệu lịch sử: mọi gói còn lệch chuẩn → 0.03.
UPDATE investment_tiers SET commission_rate = 0.0300 WHERE commission_rate <> 0.0300;

-- Khoá CHUẨN tỷ lệ hoa hồng 3 cấp trong site_settings (đề phòng admin lỡ chỉnh lệch hoặc thiếu key):
-- F1 = 3% (nhà đầu tư), F2 = 2%, F3 = 1%. F1 khách lấy từ investment_tiers.commission_rate ở trên.
-- UPSERT để chắc chắn có key + đúng giá trị dù DB hiện thiếu (vd referral_f1_rate chưa seed).
INSERT INTO site_settings (key, value) VALUES
    ('referral_f1_rate', '0.03'),
    ('referral_f2_rate', '0.02'),
    ('referral_f3_rate', '0.01')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- SYNC TẤT CẢ hoa hồng (kể cả ĐÃ CHI 'paid') về chuẩn F1=3%/F2=2%/F3=1% — yêu cầu của chủ hệ thống:
-- tính lại cho chuẩn số, áp 3% thay cho 5% (sai). CHỈ chừa 'rejected' (đã huỷ, không tính tiền).
-- Lưu ý: số dư ví tính live từ bảng commissions nên hạ rate sẽ giảm "tổng hoa hồng"/số dư khả dụng
-- của người nhận về đúng chuẩn 3%. Thuế TNCN 10% trên gross. Idempotent.
UPDATE commissions
SET rate       = std.rate,
    amount     = round(base_amount * std.rate),
    tax_pit    = round(round(base_amount * std.rate) * 0.10),
    net_amount = round(base_amount * std.rate) - round(round(base_amount * std.rate) * 0.10)
FROM (VALUES (1, 0.0300::numeric), (2, 0.0200::numeric), (3, 0.0100::numeric)) AS std(level, rate)
WHERE commissions.level = std.level
  AND commissions.status <> 'rejected'
  AND commissions.rate <> std.rate;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Khôi phục default cũ (0.05). Không khôi phục dữ liệu (đã chuẩn hoá). No-op cho data.
ALTER TABLE investment_tiers ALTER COLUMN commission_rate SET DEFAULT 0.0500;
-- +goose StatementEnd
