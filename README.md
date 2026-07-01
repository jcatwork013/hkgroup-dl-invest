# HK SHAREHOLDER Platform v2.0

Nền tảng **chào bán cổ phần riêng lẻ (private equity placement)** cho HKGroup.
Nhà đầu tư góp vốn → nhận cổ phần → theo dõi sở hữu & cổ tức **thực chia**.

> ⚠️ Đây **KHÔNG** phải sản phẩm cam kết lợi nhuận, **KHÔNG** phải đa cấp.
> Mọi ràng buộc pháp lý được encode thành invariants ở tầng code & database.

---

## Kiến trúc

- **Backend:** Go 1.22 · Chi · sqlc + pgx · PostgreSQL 16 · Redis · NATS JetStream · goose
- **Frontend:** Next.js 15 (App Router) · TypeScript · TanStack Query · Tailwind · shadcn/ui
- **Auth:** JWT access + refresh, RBAC (investor / admin)

Monorepo theo DDD + Clean Architecture:

```
/backend   Go services  (domain / usecase / repository / handler)
/frontend  Next.js app
/db        migrations (goose) + sqlc queries
/docs      spec, ADR
/deploy    docker-compose, env templates
```

Bounded contexts: `identity` · `offering` · `investment` · `referral` · `dividend` · `admin` · `audit`.

---

## Chạy nhanh (dev)

```bash
cp deploy/.env.example deploy/.env
docker compose -f deploy/docker-compose.yml up --build
# Backend:  http://localhost:8080  (health: /healthz)
# Frontend: http://localhost:3000
```

Migration + seed (offering 9,9 tỷ / 49% / 5 tiers) chạy tự động khi container `migrate` khởi động.

Tài khoản admin seed: `admin@hkgroup.vn` / `Admin@12345` (đổi ngay ở production).

---

## ⛔ HARD CONSTRAINTS (đã encode vào code/DB)

1. **Cấm field lợi nhuận cố định.** Không tồn tại `target_return` / `guaranteed_return` /
   `expected_profit` / `roi_fixed` ở schema/API/UI. Giá trị nhận lại **chỉ** là
   `dividend_paid` (cổ tức thực chia, admin nhập sau quyết định chia).
2. **Referral tối đa 1 tầng.** Bảng `referrals` chỉ lưu `referrer_id` trực tiếp.
   Không closure-table, không đệ quy, không F2/F3.
3. **Tách biệt referral loại.** `referral_type ENUM('customer','investor')`.
   Hoa hồng tiền mặt chỉ bật cho `customer`; `investor` chỉ ghi nhận (flag default off).
4. **Tiền vào TK pháp nhân.** `payments.company_account` là tài khoản công ty.
   Không có cột/luồng nào ghi tiền vào tài khoản cá nhân.
5. **Không tự sinh tiền.** Vốn góp & cổ phần chỉ ghi nhận sau khi admin đối soát
   tiền thực về. Không cron/auto cộng tiền.

---

## 8 INVARIANTS (có test bao phủ — `backend/internal/.../*_invariant_test.go`)

| # | Invariant | Enforce ở |
|---|-----------|-----------|
| 1 | `SUM(shareholdings.shares) <= offering.shares_for_sale` | DB CHECK + usecase |
| 2 | `investments.status`: `pending → reconciled → approved` (actor+ts mỗi bước) | usecase state machine |
| 3 | Issue shares = `share_ledger` insert + `shareholdings` update + `offering.shares_sold` update — **atomic 1 transaction** | repository tx |
| 4 | `shareholdings.shares == SUM(share_ledger.shares_delta)` của user | reconcile query + test |
| 5 | Hoa hồng chỉ sinh trên `investments.status='approved'`, chỉ 1 tầng | usecase |
| 6 | Cổ tức chỉ tồn tại khi có `dividends` do admin tạo (no auto) | usecase + no cron |
| 7 | Không bút toán tiền nào không gắn `payments` đã reconcile | FK + usecase |
| 8 | Mọi duyệt/phát hành/chi ghi `audit_logs` cùng transaction | repository tx |

Append-only & immutable: `share_ledger`, `audit_logs` (UPDATE/DELETE bị trigger chặn ở DB).

---

## Cấu trúc & docs

- `docs/spec.md` — spec rút gọn
- `docs/ADR-0001-equity-model.md` — lý do mô hình equity + ràng buộc pháp lý
- `db/migrations` — schema (nguồn sự thật của invariants)
- `db/queries` — sqlc queries

## Test

```bash
cd backend && go test ./...                 # unit + invariant tests (cần PG cho integration)
make test-invariants                        # chỉ 8 invariant tests
```
