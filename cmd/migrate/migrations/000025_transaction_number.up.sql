-- Har bir kompaniya (received_company_id) uchun alohida transaction raqami.
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS number bigint;

CREATE TABLE IF NOT EXISTS transaction_company_counters (
    company_id bigint PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_number bigint NOT NULL DEFAULT 0
);

-- Mavjud yozuvlarni received_company_id bo'yicha ketma-ket raqamlash.
WITH numbered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY received_company_id
            ORDER BY created_at ASC, id ASC
        ) AS rn
    FROM transactions
    WHERE received_company_id IS NOT NULL
)
UPDATE transactions t
SET number = n.rn
FROM numbered n
WHERE t.id = n.id;

UPDATE transactions
SET number = id
WHERE number IS NULL;

ALTER TABLE transactions
    ALTER COLUMN number SET NOT NULL;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_received_company_number_unique;

ALTER TABLE transactions
    ADD CONSTRAINT transactions_received_company_number_unique
        UNIQUE (received_company_id, number);

INSERT INTO transaction_company_counters (company_id, last_number)
SELECT received_company_id, MAX(number)
FROM transactions
WHERE received_company_id IS NOT NULL
GROUP BY received_company_id
ON CONFLICT (company_id) DO UPDATE
    SET last_number = GREATEST(
        transaction_company_counters.last_number,
        EXCLUDED.last_number
    );

CREATE INDEX IF NOT EXISTS idx_transactions_number
    ON transactions (received_company_id, number);
