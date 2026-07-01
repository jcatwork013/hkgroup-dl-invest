-- +goose Up
-- +goose StatementBegin
-- Chính sách web bán hàng — admin sửa được (thay cho trang tĩnh trên shop).
CREATE TABLE IF NOT EXISTS policies (
    slug       TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    summary    TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL DEFAULT '',   -- nội dung nhiều đoạn (ngăn cách bằng dòng trống)
    sort_order INT  NOT NULL DEFAULT 0,
    active     BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO policies (slug, title, summary, body, sort_order) VALUES
('bao-mat', 'Chính sách bảo mật', 'Cam kết thu thập, sử dụng và bảo vệ thông tin cá nhân của khách hàng.',
 E'HKGROUP tôn trọng và bảo vệ thông tin cá nhân của khách hàng khi truy cập, mua hàng tại duoclieuhk.vn.\n\nThông tin thu thập: họ tên, số điện thoại, email, địa chỉ nhận hàng khi bạn đặt hàng hoặc đăng ký nhận tư vấn.\n\nMục đích: xử lý và chăm sóc đơn hàng, thông báo ưu đãi khi được bạn đồng ý, nâng cao chất lượng dịch vụ.\n\nBảo mật: thông tin được lưu trữ an toàn và KHÔNG bán/chia sẻ cho bên thứ ba vì mục đích thương mại (trừ đối tác vận chuyển/thanh toán để hoàn tất đơn).\n\nQuyền của bạn: có thể yêu cầu tra cứu, chỉnh sửa hoặc xoá thông tin cá nhân bằng cách liên hệ hotline/email của chúng tôi.', 1),
('doi-tra', 'Chính sách đổi trả & hoàn tiền', 'Điều kiện, thời hạn và quy trình đổi trả, hoàn tiền.',
 E'HKGROUP hỗ trợ đổi trả trong vòng 7 ngày kể từ ngày nhận hàng với sản phẩm đủ điều kiện.\n\nĐiều kiện: còn nguyên tem/nhãn/bao bì, chưa qua sử dụng, còn hạn dùng, có thông tin đơn hàng hợp lệ; hoặc sản phẩm lỗi nhà sản xuất, giao sai, hư hỏng khi vận chuyển.\n\nKhông áp dụng: sản phẩm đã mở niêm phong/đã sử dụng (trừ lỗi nhà sản xuất), quá 7 ngày.\n\nQuy trình: liên hệ hotline/email trong thời hạn, cung cấp mã đơn và hình ảnh tình trạng; chúng tôi xác nhận và hướng dẫn gửi trả.\n\nHoàn tiền: với trường hợp hợp lệ, hoàn qua phương thức thanh toán ban đầu trong 3–7 ngày làm việc.', 2),
('van-chuyen', 'Chính sách vận chuyển & giao hàng', 'Phạm vi, thời gian và phí giao hàng.',
 E'HKGROUP giao hàng toàn quốc qua mạng hub khu vực và các đối tác vận chuyển uy tín.\n\nThời gian: nội thành khu vực có hub 1–2 ngày làm việc; khu vực khác 3–5 ngày; có thể thay đổi dịp lễ Tết.\n\nPhí vận chuyển: tính theo địa chỉ nhận và khối lượng, hiển thị trước khi xác nhận đơn. Nhiều chương trình miễn phí theo giá trị đơn.\n\nKiểm tra khi nhận: vui lòng kiểm tra kiện hàng; nếu hư hỏng hãy chụp ảnh và liên hệ ngay để được hỗ trợ.', 3),
('thanh-toan', 'Chính sách thanh toán', 'Các phương thức thanh toán và nguyên tắc an toàn.',
 E'HKGROUP hỗ trợ nhiều hình thức thanh toán an toàn.\n\nPhương thức: thanh toán khi nhận hàng (COD); chuyển khoản ngân hàng; các cổng/ví điện tử hỗ trợ tại thời điểm đặt hàng.\n\nAn toàn: mọi giao dịch được ghi nhận minh bạch; chúng tôi không lưu thông tin nhạy cảm thẻ/tài khoản trên website.\n\nHoá đơn: khách có nhu cầu xuất hoá đơn vui lòng cung cấp thông tin khi đặt hàng hoặc liên hệ CSKH.', 4),
('dieu-khoan', 'Điều khoản sử dụng', 'Quy định chung khi sử dụng website và mua hàng.',
 E'Khi truy cập và mua hàng tại duoclieuhk.vn, bạn đồng ý với các điều khoản dưới đây.\n\nSử dụng website: không dùng vào mục đích trái pháp luật, phá hoại hệ thống hoặc xâm phạm quyền lợi của HKGROUP và bên thứ ba.\n\nThông tin & giá: cập nhật từ hệ thống quản trị; chúng tôi có quyền điều chỉnh giá và khuyến mãi; giá áp dụng là giá tại thời điểm xác nhận đơn.\n\nSản phẩm dược liệu: là thực phẩm/thực phẩm bảo vệ sức khỏe, không phải thuốc và không thay thế thuốc chữa bệnh.\n\nSở hữu trí tuệ: toàn bộ nội dung, thương hiệu, hình ảnh thuộc HKGROUP; nghiêm cấm sao chép khi chưa được đồng ý bằng văn bản.', 5)
ON CONFLICT (slug) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS policies;
-- +goose StatementEnd
