DROP INDEX IF EXISTS idx_balance_records_company_balance_id;
ALTER TABLE balance_records DROP COLUMN IF EXISTS company_balance_id;
DROP INDEX IF EXISTS idx_company_balances_company_id;
DROP TABLE IF EXISTS company_balances;
