ALTER TABLE transaction_service_fees
    ADD COLUMN IF NOT EXISTS remaining_amount bigint NOT NULL DEFAULT 0;

UPDATE transaction_service_fees
SET remaining_amount = amount
WHERE status = 1;

UPDATE transaction_service_fees
SET remaining_amount = 0
WHERE status = 2;
