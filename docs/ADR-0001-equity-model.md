# ADR-0001 — Mô hình Equity Placement & ràng buộc pháp lý

- **Status:** Accepted
- **Date:** 2026-06-20
- **Context:** HK SHAREHOLDER Platform v2.0

## Bối cảnh

HKGroup cần một nền tảng để **chào bán cổ phần riêng lẻ (private equity placement)**: nhà đầu tư
góp vốn, nhận **cổ phần** thực tế trong pháp nhân, theo dõi tỷ lệ sở hữu và **cổ tức thực chia**.

Rủi ro pháp lý cần tránh tuyệt đối:
- Bị quy là **huy động vốn cam kết lợi nhuận** (giống hợp đồng hợp tác/“lãi suất”), vi phạm quy
  định về chứng khoán/huy động vốn.
- Bị quy là **kinh doanh đa cấp** trái phép (hoa hồng theo nhiều tầng F2/F3…).
- Vi phạm bảo vệ dữ liệu cá nhân (Nghị định 13/2023).
- Dòng tiền chảy vào tài khoản cá nhân thay vì pháp nhân.

## Quyết định

Chọn mô hình **equity (cổ phần)** thuần, và **encode các ràng buộc pháp lý thành invariant ở
tầng database + usecase** thay vì chỉ dựa vào quy ước/UI. Lý do: ràng buộc pháp lý là bất biến
kinh doanh sống còn — phải được bảo vệ ở lớp khó lách nhất (DB constraint, trigger, transaction),
không thể bị bỏ qua bởi một bug ở tầng trên.

### Các ràng buộc đã hiện thực

| Ràng buộc pháp lý | Cách encode |
|---|---|
| **Không cam kết lợi nhuận** | Schema/API/UI không tồn tại field `target_return`/`guaranteed_return`/`expected_profit`/`roi_*`. Giá trị nhận lại duy nhất là `dividend_payouts.amount` (cổ tức **đã trả thực tế**), do admin khai báo sau quyết định chia. |
| **Referral 1 tầng** | Bảng `referrals` có `referee_id` là **PRIMARY KEY** (mỗi người có tối đa 1 người giới thiệu trực tiếp) và `referrer_id`. Không closure-table, không cột “upline”, không đệ quy. Sinh hoa hồng chỉ tra cứu **đúng 1** referral của người mua. |
| **Hoa hồng chỉ cho `customer`** | `referral_type ENUM('customer','investor')`. Trigger DB `commissions_customer_only` từ chối mọi commission gắn referral `investor`. Referral `investor` mặc định **record-only** (flag `INVESTOR_REFERRAL_CASH_ENABLED` chỉ quyết định có lưu hay không, **không bao giờ** bật tiền mặt cho investor). |
| **Tiền vào TK pháp nhân** | `payments.company_account` + `company_account_name` lấy từ cấu hình công ty. Không có cột/luồng nào cho tài khoản cá nhân. |
| **Không tự sinh tiền** | Vốn góp & cổ phần chỉ được ghi nhận qua luồng `reconcile → approve` do admin thực hiện. Không cron/job nào cộng tiền hay phát hành cổ phần tự động. |

### 8 invariant toàn vẹn

1. `SUM(shareholdings.shares) ≤ offering.shares_for_sale` — CHECK `offering_sold_within_pool` +
   kiểm tra trong usecase (khóa `FOR UPDATE`).
2. `investments.status`: `pending → reconciled → approved` (hoặc `rejected`), mỗi bước có actor +
   timestamp (CHECK + state machine trong usecase, từng câu UPDATE ràng buộc `WHERE status=...`).
3. Phát hành cổ phần = `share_ledger` insert + `shareholdings` upsert + `offering.shares_sold`
   update + audit, **atomic trong 1 transaction** (`Store.ExecTx`).
4. `shareholdings.shares == SUM(share_ledger.shares_delta)` của user — `share_ledger` append-only,
   có endpoint/integrity-check đối soát.
5. Hoa hồng chỉ sinh khi `investments.status='approved'`, đúng 1 tầng, đúng `customer`.
6. Cổ tức chỉ tồn tại khi có bản ghi `dividends` do admin tạo — không auto.
7. Không bút toán phát hành nào khi `payments` chưa `reconciled` (kiểm tra trong usecase approve).
8. Mọi approve/issue/payout ghi `audit_logs` **trong cùng transaction**; bảng append-only & bất
   biến (trigger chặn UPDATE/DELETE).

## Hệ quả

- **Tích cực:** Ràng buộc pháp lý được bảo vệ ở lớp khó lách nhất; có test tự động cho từng
  invariant; audit trail bất biến phục vụ thanh tra; dễ chứng minh tuân thủ.
- **Đánh đổi:** Một số thao tác “sửa nhanh” bị cấm ở DB (vd: sửa `share_ledger`), buộc phải dùng
  bút toán bù trừ (compensating entry) — đúng tinh thần sổ kế toán, nhưng kém tiện cho thao tác ad-hoc.
- **Mở rộng:** Muốn thêm “quyền lợi” cho nhà đầu tư phải đi qua cơ chế cổ tức thực chia, không
  được thêm field hứa hẹn lợi nhuận.

## Phương án đã loại bỏ

- **Mô hình “hợp đồng hợp tác kinh doanh có lãi suất”** — loại vì bản chất là cam kết lợi nhuận.
- **Referral đa tầng + closure table** — loại vì nguy cơ bị quy đa cấp.
- **Chỉ ràng buộc ở tầng UI/usecase** — loại vì dễ bị bug/đường tắt phá vỡ; đưa xuống DB.
