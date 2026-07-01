# Triển khai production — duoclieuhk.vn

## Sơ đồ domain (đang chạy)

```
Người dùng ─► Cloudflare (proxy, SSL Full strict) ─► host nginx (62.146.239.31) ─► container (docker compose, bind 127.0.0.1)

duoclieuhk.vn  / www   ─► nginx ─► 127.0.0.1:13000  (web — Next.js app đầu tư; trang chính)
                          + /api/*  ─► 127.0.0.1:18080  (same-origin API)
invest.duoclieuhk.vn   ─► nginx ─► 127.0.0.1:13000  (web — cùng app; giữ tương thích link cũ)
                          + /api/*  ─► 127.0.0.1:18080  (same-origin API)
admin.duoclieuhk.vn    ─► nginx ─► 127.0.0.1:13000  (cùng app; "/" 302 → /admin)
                          + /api/*  ─► 127.0.0.1:18080  (same-origin API)
api-web.duoclieuhk.vn  ─► nginx ─► 127.0.0.1:18080  (backend Go API — host API chính của web app)
api.duoclieuhk.vn      ─► nginx ─► 127.0.0.1:18080  (DÀNH RIÊNG cho hệ product/order sync — hiện tạm trỏ backend invest)
```

> **Same-origin API:** Frontend gọi `/api/*` ngay trên chính host của nó (`duoclieuhk.vn`/`invest`/`admin`),
> nginx proxy về backend. Nhờ vậy app **không phụ thuộc** hostname API riêng hay CORS →
> tránh hẳn lỗi `ERR_NAME_NOT_RESOLVED`. Build frontend với `NEXT_PUBLIC_API_URL=` (rỗng) = đường dẫn tương đối.
>
> **`api-web.duoclieuhk.vn`** = host API chính cho client/tích hợp ngoài của web app đầu tư.
> **`api.duoclieuhk.vn`** được **để dành cho hệ đồng bộ product & đơn hàng** (tích hợp với folder `hkgroup`);
> hiện tại tạm trỏ về cùng backend invest cho tới khi hệ product/order riêng lên.

- TLS: Let's Encrypt (webroot `/var/www/html`) trên origin, tương thích Cloudflare **Full (strict)**.
  Certs: `duoclieuhk.vn` (+www), `invest.duoclieuhk.vn`, `admin.duoclieuhk.vn`, `api.duoclieuhk.vn`, `api-web.duoclieuhk.vn`.
  Tự gia hạn qua certbot timer + deploy-hook reload nginx (`/etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh`).
- Ports 3000/8080 đã bị app khác chiếm → stack này bind **127.0.0.1:13000** (web) và **127.0.0.1:18080** (api).

## App = 1 docker compose project (`hkgroup`)

```bash
cd /root/hkgroup
# build + chạy
docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod up -d --build
# xem log
docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod logs -f
```

Env production ở `deploy/.env.prod` (đã sinh sẵn JWT/Postgres/admin password ngẫu nhiên).
`NEXT_PUBLIC_API_URL=https://api.duoclieuhk.vn` được **bake lúc build** frontend; `CORS_ORIGIN`
cho phép `https://invest.duoclieuhk.vn` và `https://admin.duoclieuhk.vn`.

Admin seed: `admin@duoclieuhk.vn` (mật khẩu trong `deploy/.env.prod` → đổi sau lần đăng nhập đầu).

> ⚠️ Trước khi go-live: sửa `COMPANY_ACCOUNT` / `COMPANY_ACCOUNT_NAME` trong `deploy/.env.prod`
> thành **tài khoản công ty thật** (pháp nhân) rồi `up -d api` lại.

## Gỡ sạch (teardown)

**App (1 lệnh xoá hết container + volume Postgres):**
```bash
docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod down -v
```

**nginx + static + cert (nếu muốn gỡ hẳn domain khỏi host):**
```bash
rm -f /etc/nginx/sites-enabled/{duoclieuhk.vn,invest.duoclieuhk.vn,admin.duoclieuhk.vn,api.duoclieuhk.vn}.conf
rm -f /etc/nginx/sites-available/{duoclieuhk.vn,invest.duoclieuhk.vn,admin.duoclieuhk.vn,api.duoclieuhk.vn}.conf
rm -rf /var/www/duoclieuhk
nginx -t && systemctl reload nginx
# certs (tuỳ chọn)
for c in duoclieuhk.vn invest.duoclieuhk.vn admin.duoclieuhk.vn api.duoclieuhk.vn; do certbot delete --cert-name $c -n; done
```

Bản sao các file nginx vhost được lưu trong `deploy/nginx/` để version-control / dựng lại.

## Cấu hình Cloudflare cần đảm bảo
- DNS records (proxied / orange cloud) trỏ về origin `62.146.239.31`:
  `@`, `www`, `invest`, `admin`, `api` (hoặc wildcard `*`).
- SSL/TLS mode: **Full (strict)**.
- "Always Use HTTPS": bật được (origin đã có cert hợp lệ); ACME challenge đi qua `/.well-known/` HTTP vẫn hoạt động.
