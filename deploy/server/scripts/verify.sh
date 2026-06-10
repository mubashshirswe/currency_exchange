#!/usr/bin/env bash
# Server holatini tekshirish: ./scripts/verify.sh
set -euo pipefail

cd "$(dirname "$0")/.."
ok=0
warn=0
fail=0

pass() { echo "  OK   $1"; ok=$((ok + 1)); }
warn() { echo "  WARN $1"; warn=$((warn + 1)); }
fail() { echo "  FAIL $1"; fail=$((fail + 1)); }

echo "=== Deploy papka: $(pwd) ==="

[ -f docker-compose.yml ] && pass "docker-compose.yml" || fail "docker-compose.yml yo'q"
[ -f .env ] && pass ".env" || fail ".env yo'q (cp env.example .env)"
[ -x scripts/deploy.sh ] && pass "scripts/deploy.sh" || warn "scripts/deploy.sh executable emas"

if [ -f .env ]; then
  grep -q '^API_IMAGE=' .env && pass "API_IMAGE" || fail "API_IMAGE .env da yo'q"
  grep -q '^JWTSECRET=' .env && pass "JWTSECRET" || fail "JWTSECRET .env da yo'q"
  grep -q '^POSTGRES_PASSWORD=' .env && pass "POSTGRES_PASSWORD" || fail "POSTGRES_PASSWORD .env da yo'q"
fi

if grep -q 'firebase.json:/secrets/firebase.json' docker-compose.yml 2>/dev/null; then
  pass "docker-compose.yml (Firebase volume)"
fi

if [ -f secrets/firebase.json ]; then
  pass "secrets/firebase.json"
elif [ -f secrets/firebase-adminsdk.json ]; then
  warn "secrets/firebase-adminsdk.json bor — mv secrets/firebase-adminsdk.json secrets/firebase.json"
elif grep -q 'FIREBASE_CREDENTIALS_PATH=/secrets' .env 2>/dev/null; then
  fail "Firebase JSON yo'q: secrets/firebase.json"
fi

echo ""
echo "=== Docker ==="
command -v docker >/dev/null && pass "docker" || fail "docker o'rnatilmagan"

if docker compose ps >/dev/null 2>&1; then
  docker compose ps
  db_running() {
    local id
    id="$(docker compose ps -q db 2>/dev/null || true)"
    [ -n "$id" ] && [ "$(docker inspect -f '{{.State.Running}}' "$id" 2>/dev/null)" = "true" ]
  }
  db_running && pass "db ishlayapti" || warn "db ishlamayapti"
  docker compose ps -q api 2>/dev/null | grep -q . && pass "api konteyner bor" || warn "api konteyner yo'q"
else
  warn "docker compose ps ishlamadi"
fi

echo ""
echo "=== Volume (data shu yerda) ==="
docker volume ls 2>/dev/null | grep -E 'currency-exchange|pgdata|redisdata' || warn "volume topilmadi (hali birinchi up qilinmagan?)"

echo ""
echo "Natija: OK=$ok WARN=$warn FAIL=$fail"
[ "$fail" -eq 0 ]
