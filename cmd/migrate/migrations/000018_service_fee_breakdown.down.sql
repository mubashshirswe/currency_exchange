ALTER TABLE transactions
    DROP COLUMN IF EXISTS service_fee_amount,
    DROP COLUMN IF EXISTS service_fee_currency,
    DROP COLUMN IF EXISTS service_fee_details;
