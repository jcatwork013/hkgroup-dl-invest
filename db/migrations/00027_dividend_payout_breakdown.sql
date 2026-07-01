-- +goose Up
-- +goose StatementBegin

-- Lưu CHI TIẾT cấu thành mỗi khoản chi trả cổ tức (tiered "đồng chia + bonus") ngay tại
-- dividend_payouts, để bảng chi trả hiển thị rõ: 55k = 30k đồng chia (9%) + 25k bonus (hạng
-- 2,5% / vốn 300tr). Trước đây chỉ lưu `amount` tổng nên không truy được nguồn gốc con số.
-- LƯU Ý: đây CHỈ là phần chia theo ĐẦU TƯ (cổ tức) — KHÔNG bao gồm hoa hồng % bán hàng theo đơn.
ALTER TABLE dividend_payouts
  ADD COLUMN IF NOT EXISTS equal_share  bigint  NOT NULL DEFAULT 0, -- phần đồng chia (9%, cào bằng)
  ADD COLUMN IF NOT EXISTS bonus        bigint  NOT NULL DEFAULT 0, -- phần bonus theo hạng (6%)
  ADD COLUMN IF NOT EXISTS band         text    NOT NULL DEFAULT '', -- nhãn hạng vốn (vd "300–500tr")
  ADD COLUMN IF NOT EXISTS band_rate    numeric NOT NULL DEFAULT 0, -- tỉ lệ hạng (vd 0.025)
  ADD COLUMN IF NOT EXISTS invested_vnd bigint  NOT NULL DEFAULT 0; -- vốn đã góp (gói đầu tư) tại kỳ chia

-- +goose StatementEnd

-- +goose StatementBegin
-- Backfill các kỳ tiered đã chia trước migration: equal_share = đồng chia (tổng pool 9% / N),
-- bonus = amount - equal_share; band/band_rate theo vốn góp hiện tại (ngưỡng 50tr / 300tr).
-- Chỉ áp cho đợt cổ tức có ghi chú "Đồng chia" (đường tiered), bỏ qua cổ tức pro-rata thủ công.
UPDATE dividend_payouts p
SET invested_vnd = sub.inv,
    band = CASE WHEN sub.inv >= 300000000 THEN '300–500tr'
                WHEN sub.inv >= 50000000  THEN '50–299tr'
                ELSE '5–49tr' END,
    band_rate = CASE WHEN sub.inv >= 300000000 THEN 0.025
                     WHEN sub.inv >= 50000000  THEN 0.02
                     ELSE 0.015 END,
    equal_share = sub.equal_each,
    bonus = p.amount - sub.equal_each
FROM (
  SELECT sh.user_id,
         COALESCE(SUM(i.amount_vnd) FILTER (WHERE i.status = 'approved'), 0)::bigint AS inv,
         (SELECT floor((d.total_amount * 9.0 / 15.0) / NULLIF(cnt.n, 0))::bigint)     AS equal_each,
         d.id AS div_id
  FROM shareholdings sh
  LEFT JOIN investments i ON i.user_id = sh.user_id
  CROSS JOIN dividends d
  CROSS JOIN (SELECT count(*) AS n FROM shareholdings WHERE shares > 0) cnt
  WHERE sh.shares > 0 AND d.note LIKE 'Đồng chia%'
  GROUP BY sh.user_id, d.id, d.total_amount, cnt.n
) sub
WHERE p.user_id = sub.user_id AND p.dividend_id = sub.div_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE dividend_payouts
  DROP COLUMN IF EXISTS equal_share,
  DROP COLUMN IF EXISTS bonus,
  DROP COLUMN IF EXISTS band,
  DROP COLUMN IF EXISTS band_rate,
  DROP COLUMN IF EXISTS invested_vnd;
-- +goose StatementEnd
