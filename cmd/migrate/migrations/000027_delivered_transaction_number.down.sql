DROP INDEX IF EXISTS idx_transactions_delivered_number;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_delivered_company_number_unique;

ALTER TABLE transactions
    DROP COLUMN IF EXISTS delivered_number;
