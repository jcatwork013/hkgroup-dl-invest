# HK SHAREHOLDER Platform v2.0 — Spec (rút gọn)

## Mục tiêu
Nền tảng chào bán cổ phần riêng lẻ: NĐT góp vốn → nhận cổ phần → theo dõi sở hữu & cổ tức thực chia.
KHÔNG cam kết lợi nhuận, KHÔNG đa cấp. Xem [ADR-0001](./ADR-0001-equity-model.md).

## Bounded contexts
- **identity** — auth (JWT access/refresh), RBAC (investor/admin), eKYC (CCCD + selfie), consent (NĐ 13/2023).
- **offering** — định giá, tổng cổ phần, pool chào bán, tiers, cap table (read model).
- **investment** — hợp đồng (ký OTP), payment (TK pháp nhân), đối soát, duyệt, phát hành cổ phần.
- **referral** — 1 tầng, hoa hồng (chỉ customer), thuế TNCN.
- **dividend** — admin khai báo & chia cổ tức thực.
- **audit** — log bất biến, append-only.

## Luồng nhà đầu tư
1. Đăng ký → eKYC (upload CCCD + selfie) → consent dữ liệu.
2. Landing: định giá 9,9 tỷ · 49% cổ phần · tiến độ bán · disclaimer rủi ro (no return promise).
3. Trang đầu tư: chọn tier → xem số cổ phần & % sở hữu (KHÔNG hiển thị lợi nhuận).
4. Ký hợp đồng số (OTP) → sinh PDF → nhận mã `HKG-INV-xxxx` → thông tin chuyển khoản (TK công ty).
5. Bấm “Tôi đã chuyển khoản”.
6. Dashboard: vốn đã góp · số cổ phần · % sở hữu · cổ tức đã nhận (thực) · tải hợp đồng.
7. Link giới thiệu 1 tầng.

## Luồng admin
- Đối soát 3 lớp: **KYC ✓** / **tiền về đúng số + nội dung ✓** / **hợp đồng ký ✓** → Duyệt → phát hành cổ phần.
- Cap table / sổ cổ đông + integrity check.
- Referral 1 tầng + tính/chi hoa hồng (chỉ customer) + khấu trừ thuế TNCN.
- Khai báo & chia cổ tức.
- Audit log viewer.
- Dashboard: tổng vốn đã đối soát · số cổ đông · % cổ phần đã bán · tổng hoa hồng (tách loại).

## State machine `investments.status`
```
pending ──reconcile(admin)──► reconciled ──approve(admin)──► approved
   │                              │
   └──────────── reject(admin) ◄──┘
```
Mỗi chuyển trạng thái ghi actor + timestamp và 1 dòng audit.

## API
Xem bảng contract trong README và `internal/server/server.go`. Prefix: `/api/v1`.

## Tham số seed
- Offering: valuation 9.900.000.000₫, total_shares 1.000.000 (100%), shares_for_sale 490.000 (49%).
- Giá/cổ phần = 9.900₫. Tiers: 1% / 5% / 10% / 20% / 49%.
