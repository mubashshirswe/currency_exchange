DROP INDEX IF EXISTS idx_transactions_delivery_number;
DROP TABLE IF EXISTS transaction_delivery_company_counters;
ALTER TABLE transactions DROP COLUMN IF EXISTS delivery_number;
