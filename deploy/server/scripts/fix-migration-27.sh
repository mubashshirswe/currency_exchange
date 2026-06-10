#!/usr/bin/env bash
# Migration 27 "dirty" holatini tiklash (production).
# Sabab: transaction_company_counters jadvali yo'q (25-qism to'liq qo'llanmagan yoki force o'tkazilgan).
#
# Ishlatish: serverda deploy papkasida
#   ./scripts/fix-migration-27.sh
#
# Oldin yangi API image (tuzatilgan 000027) deploy qilingan bo'lishi kerak.

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
echo "=== transaction_company_counters jadvalini tiklash (agar yo'q bo'lsa) ==="
psql <<'SQL'
CREATE TABLE IF NOT EXISTS transaction_company_counters (
    company_id bigint PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_number bigint NOT NULL DEFAULT 0
);

-- received_company_id bo'yicha counter (000025 mantiq)
INSERT INTO transaction_company_counters (company_id, last_number)
SELECT
    received_company_id,
    COALESCE(MAX(number::bigint), 0)::bigint
FROM transactions
WHERE received_company_id IS NOT NULL
  AND EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = 'public'
        AND table_name = 'transactions'
        AND column_name = 'number'
  )
GROUP BY received_company_id
ON CONFLICT (company_id) DO UPDATE
    SET last_number = GREATEST(
        transaction_company_counters.last_number,
        EXCLUDED.last_number
    );
SQL

echo ""
echo "=== 27-dirty holatni 26 ga qaytarish (000027 qayta migrate up uchun) ==="
psql -c "UPDATE schema_migrations SET version = 26, dirty = false WHERE dirty = true OR version >= 27;"

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
