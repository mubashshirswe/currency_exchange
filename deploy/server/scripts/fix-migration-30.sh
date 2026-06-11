#!/usr/bin/env bash
# Migration 30 "dirty" holatini tiklash (production).
# Sabab: number ustuni varchar bo'lib qolgan, COALESCE(number, id::bigint) yiqilgan.
#
# Ishlatish: serverda deploy papkasida
#   ./scripts/fix-migration-30.sh
#
# Oldin tuzatilgan 000030 bilan yangi API image deploy qilingan bo'lishi kerak.

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

psql() {
  dc exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"
}

echo "=== schema_migrations holati ==="
psql -c "SELECT version, dirty FROM schema_migrations;"

echo ""
echo "=== number ustunini bigint ga o'tkazish (agar varchar bo'lsa) ==="
psql <<'SQL'
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'transactions'
          AND column_name = 'number'
          AND data_type IN ('text', 'character varying')
    ) THEN
        ALTER TABLE transactions
            ALTER COLUMN number TYPE bigint
            USING NULLIF(trim(number::text), '')::bigint;
    END IF;
END $$;

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS delivered_number bigint;
SQL

echo ""
echo "=== 30-dirty holatni 29 ga qaytarish (000030 qayta migrate up uchun) ==="
psql -c "UPDATE schema_migrations SET version = 29, dirty = false WHERE dirty = true OR version >= 30;"

echo ""
echo "=== API ni qayta ishga tushirish (migrate up) ==="
dc up -d --no-deps --force-recreate api

echo ""
echo "Kutilmoqda (10s)..."
sleep 10

echo ""
echo "=== API log (oxirgi 40 qator) ==="
dc logs api --tail 40

echo ""
echo "=== schema_migrations (yakuniy) ==="
psql -c "SELECT version, dirty FROM schema_migrations;"
