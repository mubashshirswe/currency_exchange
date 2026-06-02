ALTER TABLE exchanges
    ADD COLUMN IF NOT EXISTS profit_amount bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS profit_currency varchar(255) NOT NULL DEFAULT '';

ALTER TABLE soft_balance_records
    ADD COLUMN IF NOT EXISTS exchange_id bigint REFERENCES exchanges(id);

CREATE INDEX IF NOT EXISTS idx_soft_balance_records_exchange_id
    ON soft_balance_records (exchange_id);
