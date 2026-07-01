#!/usr/bin/env bash
# =============================================================================
# deploy-app.sh — deploy FULL (api + web + migration) cho PROD, an toàn.
#
# An toàn theo bài học sự cố 2026-06-23: dùng ĐÚNG prod compose + .env.prod,
# và chỉ recreate `migrate`, `api`, `web` (--no-deps) → KHÔNG bao giờ đụng tới
# postgres/redis (giữ nguyên data, không lệch mật khẩu).
#
# Quy trình:
#   1. Build api + web   → container cũ vẫn phục vụ (0 downtime khi build).
#   2. Chạy migrate      → áp migration mới (additive).
#   3. Recreate api+web  → swap sang image mới (gián đoạn ~1–2s).
#   4. Health-check.
#
# Dùng:  bash deploy/deploy-app.sh
# =============================================================================
set -euo pipefail
cd "$(dirname "$0")/.."

COMPOSE="docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod"

echo "==> [1/4] Build api + web + migrate (container cũ vẫn chạy)..."
# migrate image chứa file migration ở build-time → BẮT BUỘC build lại để có migration mới.
$COMPOSE build api web migrate

echo "==> [2/4] Chạy migrate (áp migration, KHÔNG đụng postgres)..."
$COMPOSE up -d --no-deps --no-build migrate
docker wait hkgroup-migrate-1 >/dev/null 2>&1 || true
echo "    migrate log:"; docker logs --tail 4 hkgroup-migrate-1 2>&1 | sed 's/^/      /'
code=$(docker inspect hkgroup-migrate-1 --format '{{.State.ExitCode}}' 2>/dev/null || echo "?")
if [ "$code" != "0" ]; then
  echo "!!! migrate exit code = $code — DỪNG, KHÔNG recreate api/web." >&2
  exit 1
fi

echo "==> [3/4] Recreate api + web (--no-deps: không đụng postgres/redis)..."
$COMPOSE up -d --no-deps --no-build api web

echo "==> [4/4] Health-check..."
ok=0
for i in $(seq 1 30); do
  a=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18080/healthz || true)
  wcode=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:13000/admin/policy || true)
  if [ "$a" = "200" ] && [ "$wcode" = "200" ]; then ok=1; echo "    OK (api=$a web=$wcode sau ${i}s)"; break; fi
  sleep 1
done
[ "$ok" = "1" ] || { echo "!!! Health-check fail (api=$a web=$wcode) — xem: docker logs hkgroup-api-1 / hkgroup-web-1" >&2; exit 1; }

echo "==> Deploy xong. (postgres/redis KHÔNG bị đụng.)"
