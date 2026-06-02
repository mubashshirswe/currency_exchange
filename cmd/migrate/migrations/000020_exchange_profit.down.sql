DROP INDEX IF EXISTS idx_soft_balance_records_exchange_id;

ALTER TABLE soft_balance_records
    DROP COLUMN IF EXISTS exchange_id;

ALTER TABLE exchanges
    DROP COLUMN IF EXISTS profit_currency,
    DROP COLUMN IF EXISTS profit_amount;
