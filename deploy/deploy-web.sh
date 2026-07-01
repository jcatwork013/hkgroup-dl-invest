#!/usr/bin/env bash
# =============================================================================
# deploy-web.sh — triển khai LẠI riêng frontend (web) cho PROD, an toàn + nhanh.
#
# BÀI HỌC từ sự cố 2026-06-23: KHÔNG được chạy `docker compose up` chung chung —
# nó recreate cả postgres/migrate và làm lệch mật khẩu DB → sập stack. Script này
# CHỈ đụng tới service `web` (`--no-deps`), tuyệt đối không chạm DB/redis/api.
#
# Quy trình (gần như zero-downtime):
#   1. Build image web MỚI  → container CŨ vẫn đang phục vụ (0 downtime khi build).
#   2. Recreate CHỈ `web`   → đổi sang image mới (gián đoạn ~1–2s lúc swap).
#   3. Health-check         → nếu fail thì báo lỗi để rollback thủ công.
#
# Dùng:  bash deploy/deploy-web.sh
# =============================================================================
set -euo pipefail

cd "$(dirname "$0")/.."   # về repo root (/root/hkgroup)

COMPOSE="docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod"
PORT=13000
URL="http://127.0.0.1:${PORT}/admin/policy"

echo "==> [1/3] Build image web mới (container cũ vẫn chạy, KHÔNG downtime)..."
$COMPOSE build web

echo "==> [2/3] Recreate CHỈ service web (--no-deps: không đụng postgres/api/redis)..."
$COMPOSE up -d --no-deps --no-build web

echo "==> [3/3] Health-check ${URL} ..."
ok=0
for i in $(seq 1 30); do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$URL" || true)
  if [ "$code" = "200" ]; then ok=1; echo "    OK (HTTP 200 sau ${i}s)"; break; fi
  sleep 1
done

if [ "$ok" != "1" ]; then
  echo "!!! Web KHÔNG trả 200 sau 30s — kiểm tra: docker logs hkgroup-web-1" >&2
  exit 1
fi

echo "==> Xong. Web đã chạy image mới. (DB/api KHÔNG bị đụng tới.)"
