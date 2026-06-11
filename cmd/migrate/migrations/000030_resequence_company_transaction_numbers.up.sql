-- Har bir kompaniya uchun transaction raqamlarini 1, 2, 3... dan qayta tiklash.
-- Eski global id yoki noto'g'ri raqamlarga bog'liq emas.

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_received_company_number_unique;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_delivered_company_number_unique;

-- received_company_id bo'yicha ketma-ket raqamlash (yaratuvchi kompaniya).
WITH numbered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY received_company_id
            ORDER BY created_at ASC, id ASC
        )::bigint AS rn
    FROM transactions
    WHERE received_company_id IS NOT NULL
)
UPDATE transactions t
SET number = n.rn
FROM numbered n
WHERE t.id = n.id;

UPDATE transactions
SET number = id::bigint
WHERE received_company_id IS NULL AND number IS NULL;

-- delivered_company_id bo'yicha ketma-ket raqamlash (yetkazuvchi kompaniya).
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
WHERE t.id = n.id;

UPDATE transactions
SET delivered_number = COALESCE(number, id::bigint)
WHERE delivered_company_id IS NULL AND delivered_number IS NULL;

ALTER TABLE transactions
    ALTER COLUMN number SET NOT NULL;

ALTER TABLE transactions
    ALTER COLUMN delivered_number SET NOT NULL;

ALTER TABLE transactions
    ADD CONSTRAINT transactions_received_company_number_unique
        UNIQUE (received_company_id, number);

ALTER TABLE transactions
    ADD CONSTRAINT transactions_delivered_company_number_unique
        UNIQUE (delivered_company_id, delivered_number);

CREATE TABLE IF NOT EXISTS transaction_company_counters (
    company_id bigint PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_number bigint NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS transaction_delivered_company_counters (
    company_id bigint PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_number bigint NOT NULL DEFAULT 0
);

INSERT INTO transaction_company_counters (company_id, last_number)
SELECT
    received_company_id,
    COALESCE(MAX(number::bigint), 0)::bigint
FROM transactions
WHERE received_company_id IS NOT NULL
GROUP BY received_company_id
ON CONFLICT (company_id) DO UPDATE
    SET last_number = EXCLUDED.last_number;

INSERT INTO transaction_delivered_company_counters (company_id, last_number)
SELECT
    delivered_company_id,
    COALESCE(MAX(delivered_number::bigint), 0)::bigint
FROM transactions
WHERE delivered_company_id IS NOT NULL
GROUP BY delivered_company_id
ON CONFLICT (company_id) DO UPDATE
    SET last_number = EXCLUDED.last_number;
