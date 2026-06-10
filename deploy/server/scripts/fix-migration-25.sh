#!/usr/bin/env bash
# Migration 25 "dirty" holatini tiklash (production).
# Ishlatish: serverda /app/backend/currency-exchange ichida
#   ./scripts/fix-migration-25.sh
#
# Oldin yangi API image (tuzatilgan 000025) deploy qilingan bo'lishi kerak.

set -euo pipefail

cd "$(dirname "$0")/.."

if [ ! -f .env ]; then
  echo "Xato: .env topilmadi"
  exit 1
fi

set -a
# shellcheck disable=SC1091
source .env
set +a

DB_URL="${DB_URL:-postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@db:5432/${POSTGRES_DB}?sslmode=disable}"

if docker info >/dev/null 2>&1; then
  dc() { docker compose "$@"; }
else
  dc() { sudo -E docker compose "$@"; }
fi

DB_CID="$(dc ps -q db)"
if [ -z "$DB_CID" ]; then
  echo "Xato: db konteyneri ishlamayapti"
  exit 1
fi

echo "=== schema_migrations holati ==="
dc exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
  "SELECT version, dirty FROM schema_migrations;"

echo ""
echo "=== 25-dirty holatni 24 ga qaytarish (qayta migrate up uchun) ==="
dc exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
  "UPDATE schema_migrations SET version = 24, dirty = false WHERE dirty = true;"

echo ""
echo "=== API ni qayta ishga tushirish (migrate up) ==="
dc up -d --no-deps --force-recreate api

echo ""
echo "Kutilmoqda (10s)..."
sleep 10

echo ""
echo "=== API log (oxirgi 30 qator) ==="
dc logs api --tail 30

echo ""
echo "=== schema_migrations (yakuniy) ==="
dc exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
  "SELECT version, dirty FROM schema_migrations;"
