ALTER TABLE company_balance_records
    DROP COLUMN IF EXISTS exchange_id,
    DROP COLUMN IF EXISTS transaction_id,
    DROP COLUMN IF EXISTS debt_id;
