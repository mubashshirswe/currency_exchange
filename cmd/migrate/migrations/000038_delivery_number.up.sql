-- "Dostavka" (yetkazib berish) deb belgilangan operatsiyalar uchun alohida
-- ketma-ket raqam (1,2,3,...) — received_company_id bo'yicha alohida hisoblanadi.
-- number/delivered_number bilan aralashmasin, shuning uchun mustaqil ustun va counter.

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS delivery_number bigint NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS transaction_delivery_company_counters (
    company_id bigint PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_number bigint NOT NULL DEFAULT 0
);

-- Mavjud "Dostavka" yozuvlarini received_company_id bo'yicha ketma-ket raqamlash.
WITH numbered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY received_company_id
            ORDER BY created_at ASC, id ASC
        )::bigint AS rn
    FROM transactions
    WHERE received_company_id IS NOT NULL
      AND trim(service_fee_details) = 'Dostavka'
)
UPDATE transactions t
SET delivery_number = n.rn
FROM numbered n
WHERE t.id = n.id;

INSERT INTO transaction_delivery_company_counters (company_id, last_number)
SELECT
    received_company_id,
    COALESCE(MAX(delivery_number), 0)::bigint
FROM transactions
WHERE received_company_id IS NOT NULL
  AND trim(service_fee_details) = 'Dostavka'
GROUP BY received_company_id
ON CONFLICT (company_id) DO UPDATE
    SET last_number = GREATEST(
        transaction_delivery_company_counters.last_number,
        EXCLUDED.last_number
    );

CREATE INDEX IF NOT EXISTS idx_transactions_delivery_number
    ON transactions (received_company_id, delivery_number);
