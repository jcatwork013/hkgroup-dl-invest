-- +goose Up
-- +goose StatementBegin

-- 1) KYC: thêm ảnh CCCD MẶT SAU. Trước đây hồ sơ chỉ có 1 ảnh CCCD (mặt trước) + selfie.
--    Cột để DEFAULT '' để các hồ sơ cũ vẫn hợp lệ; luồng nộp mới bắt buộc đủ 2 mặt (validate ở app).
ALTER TABLE kyc_records ADD COLUMN IF NOT EXISTS cccd_back_url TEXT NOT NULL DEFAULT '';

-- 1b) Thông điệp KYC hiển thị cho người dùng (chuông thông báo): lý do từ chối / yêu cầu nộp lại
--     (vd "ảnh mờ", "sai định dạng"). Đặt khi admin từ chối / yêu cầu KYC lại; xoá khi duyệt.
ALTER TABLE users ADD COLUMN IF NOT EXISTS kyc_message TEXT NOT NULL DEFAULT '';

-- 2) Xoá tài khoản (admin) — "xoá an toàn": chỉ xoá tài khoản KHÔNG có dấu vết tài chính.
--    Vấn đề: audit_logs.actor_id tham chiếu users(id) và audit_logs là BẤT BIẾN (chặn UPDATE/DELETE
--    bởi trigger ở 00009). Mỗi nhà đầu tư nộp KYC đều tạo 1 dòng audit với actor = chính họ, nên FK
--    RESTRICT sẽ chặn việc xoá user. Giải pháp tôn trọng tính bất biến: KHÔNG xoá/sửa NỘI DUNG audit,
--    chỉ CẮT con trỏ actor (đặt actor_id = NULL) khi user bị xoá — đúng tinh thần "quyền được xoá"
--    (NĐ13): lịch sử hành động được giữ nguyên, chỉ gỡ liên kết tới cá nhân đã xoá.

-- 2a) Đổi FK actor_id sang ON DELETE SET NULL (PostgreSQL thực thi bằng một UPDATE nội bộ).
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_actor_id_fkey;
ALTER TABLE audit_logs
    ADD CONSTRAINT audit_logs_actor_id_fkey
    FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL;

-- 2b) Nới trigger chặn-UPDATE của audit_logs để CHO PHÉP DUY NHẤT thao tác cắt actor về NULL
--     (mọi cột nội dung phải giữ nguyên). Mọi UPDATE khác vẫn bị chặn. DELETE vẫn bị chặn hoàn toàn.
CREATE OR REPLACE FUNCTION trg_audit_no_update() RETURNS trigger AS $$
BEGIN
    IF NEW.actor_id IS NULL
       AND NEW.action     = OLD.action
       AND NEW.entity     = OLD.entity
       AND NEW.entity_id  = OLD.entity_id
       AND NEW.before     IS NOT DISTINCT FROM OLD.before
       AND NEW.after      IS NOT DISTINCT FROM OLD.after
       AND NEW.created_at = OLD.created_at THEN
        RETURN NEW; -- chỉ cắt liên kết actor của tài khoản đã xoá
    END IF;
    RAISE EXCEPTION 'audit_logs is append-only: UPDATE not allowed'
        USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
CREATE TRIGGER audit_logs_no_update BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION trg_audit_no_update();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
CREATE TRIGGER audit_logs_no_update BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION trg_block_mutation();
DROP FUNCTION IF EXISTS trg_audit_no_update;

ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_actor_id_fkey;
ALTER TABLE audit_logs
    ADD CONSTRAINT audit_logs_actor_id_fkey
    FOREIGN KEY (actor_id) REFERENCES users(id);

ALTER TABLE users DROP COLUMN IF EXISTS kyc_message;
ALTER TABLE kyc_records DROP COLUMN IF EXISTS cccd_back_url;

-- +goose StatementEnd
