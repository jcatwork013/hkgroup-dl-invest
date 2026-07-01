-- +goose Up
-- +goose StatementBegin
-- Thêm slug cho sản phẩm để website bán hàng công khai (duoclieuhk.vn) route theo URL đẹp
-- /san-pham/<slug> thay vì UUID. Slug là khoá công khai, UNIQUE, sinh từ tên (bỏ dấu tiếng Việt).
ALTER TABLE products ADD COLUMN slug TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
-- Hàm tạm: slugify có bỏ dấu tiếng Việt. translate() nếu from/to lệch độ dài thì chỉ XOÁ ký tự
-- thừa (không lỗi), nên backfill luôn an toàn. repeat() bảo đảm chuỗi 'to' đúng số lượng.
CREATE OR REPLACE FUNCTION hk_slugify(txt TEXT) RETURNS TEXT AS $$
  SELECT trim(both '-' FROM
    regexp_replace(
      translate(
        lower(txt),
        'àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩịòóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđ',
        repeat('a',17)||repeat('e',11)||repeat('i',5)||repeat('o',17)||repeat('u',11)||repeat('y',5)||'d'
      ),
      '[^a-z0-9]+', '-', 'g'
    )
  );
$$ LANGUAGE sql IMMUTABLE;
-- +goose StatementEnd

-- +goose StatementBegin
-- Backfill: slug đẹp từ tên; trùng thì thêm hậu tố -N; rỗng thì fallback 'sp'.
WITH base AS (
  SELECT id,
         COALESCE(NULLIF(hk_slugify(name), ''), 'sp') AS s,
         row_number() OVER (
           PARTITION BY COALESCE(NULLIF(hk_slugify(name), ''), 'sp')
           ORDER BY created_at, id
         ) AS rn
  FROM products
)
UPDATE products p
SET slug = base.s || CASE WHEN base.rn > 1 THEN '-' || base.rn::text ELSE '' END
FROM base
WHERE p.id = base.id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION hk_slugify(TEXT);
-- Partial unique: nhiều bản ghi slug='' (nếu có) không đụng nhau; app luôn set slug khác rỗng.
CREATE UNIQUE INDEX uq_products_slug ON products(slug) WHERE slug <> '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_products_slug;
ALTER TABLE products DROP COLUMN IF EXISTS slug;
-- +goose StatementEnd
