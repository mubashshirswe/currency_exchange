DROP INDEX IF EXISTS idx_transactions_number;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_received_company_number_unique;

ALTER TABLE transactions
    DROP COLUMN IF EXISTS number;

DROP TABLE IF EXISTS transaction_company_counters;
