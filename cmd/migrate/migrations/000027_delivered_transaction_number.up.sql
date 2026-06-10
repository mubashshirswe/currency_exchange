-- delivered_company_id bo'yicha alohida transaction raqami (yetkazib beruvchi kompaniya).

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS delivered_number bigint;

WITH numbered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY delivered_company_id
            ORDER BY created_at ASC, id ASC
        )::bigint AS rn
    FROM transactions
    WHERE delivered_company_id IS NOT NULL
)
UPDATE transactions t
SET delivered_number = n.rn
FROM numbered n
WHERE t.id = n.id
  AND (t.delivered_number IS NULL OR t.delivered_number = 0);

UPDATE transactions
SET delivered_number = id::bigint
WHERE delivered_number IS NULL;

ALTER TABLE transactions
    ALTER COLUMN delivered_number SET NOT NULL;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_delivered_company_number_unique;

ALTER TABLE transactions
    ADD CONSTRAINT transactions_delivered_company_number_unique
        UNIQUE (delivered_company_id, delivered_number);

INSERT INTO transaction_company_counters (company_id, last_number)
SELECT
    delivered_company_id,
    COALESCE(MAX(delivered_number::bigint), 0)::bigint
FROM transactions
WHERE delivered_company_id IS NOT NULL
GROUP BY delivered_company_id
ON CONFLICT (company_id) DO UPDATE
    SET last_number = GREATEST(
        transaction_company_counters.last_number,
        EXCLUDED.last_number
    );

CREATE INDEX IF NOT EXISTS idx_transactions_delivered_number
    ON transactions (delivered_company_id, delivered_number);
