-- +goose Up
-- +goose StatementBegin

-- Chuẩn hoá lại các gói đầu tư cho ĐÚNG pool: 1 cổ phần = giá pool = valuation/total_shares
-- (vd 9.9 tỷ / 990.000 = 10.000đ/cp). Trước đây cổ phần & % được nhập tay nên lệch chuẩn
-- (vd Gói 1 = 5tr bị ghi 4.950 cp / 0.5% thay vì 500 cp / 0.0505%). Từ nay backend tự tính
-- cổ phần & % từ amount_vnd nên không thể sai nữa; migration này sửa dữ liệu lịch sử.

-- 1) Gỡ các gói "đã ẩn" và CHƯA phát sinh hợp đồng/đầu tư (vd Gói 7 trùng lặp tạo tay).
DELETE FROM investment_tiers t
WHERE t.active = false
  AND NOT EXISTS (SELECT 1 FROM contracts   c WHERE c.tier_id = t.id)
  AND NOT EXISTS (SELECT 1 FROM investments i WHERE i.tier_id = t.id);

-- 2) Tính lại cổ phần & % sở hữu cho mọi gói còn lại theo giá pool.
UPDATE investment_tiers t
SET shares        = round(t.amount_vnd::numeric / (o.valuation_vnd::numeric / o.total_shares::numeric)),
    ownership_pct = round(t.amount_vnd::numeric / o.valuation_vnd::numeric * 100, 4)
FROM offering o
WHERE t.offering_id = o.id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Không khôi phục (dữ liệu cũ sai). No-op.
SELECT 1;
-- +goose StatementEnd
