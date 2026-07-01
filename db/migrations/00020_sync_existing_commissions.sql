-- +goose Up
-- +goose StatementBegin

-- SYNC DATA CŨ: các hoa hồng đã phát sinh TRƯỚC khi thống nhất chính sách F1=3%/F2=2%/F3=1%
-- (migration 00019) vẫn lưu tỷ lệ cũ (vd F1 khách = 15–25% theo gói). Tính lại các dòng CHƯA chi
-- (status 'pending'/'approved') theo chuẩn mới. KHÔNG đụng vào dòng đã 'paid' (tiền thật đã chi) hay
-- 'rejected'. Thuế TNCN 10% trên hoa hồng gross (PIT_RATE mặc định). Idempotent: chạy lại không đổi.
UPDATE commissions
SET rate       = std.rate,
    amount     = round(base_amount * std.rate),
    tax_pit    = round(round(base_amount * std.rate) * 0.10),
    net_amount = round(base_amount * std.rate) - round(round(base_amount * std.rate) * 0.10)
FROM (VALUES (1, 0.0300::numeric), (2, 0.0200::numeric), (3, 0.0100::numeric)) AS std(level, rate)
WHERE commissions.level = std.level
  AND commissions.status IN ('pending', 'approved')
  AND commissions.rate <> std.rate;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Không thể khôi phục tỷ lệ cũ theo gói một cách an toàn (đã mất thông tin gói gốc của F1). No-op.
SELECT 1;
-- +goose StatementEnd
