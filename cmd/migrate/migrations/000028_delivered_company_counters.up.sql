-- delivered_number uchun alohida counter (number bilan aralashmasin).

CREATE TABLE IF NOT EXISTS transaction_delivered_company_counters (
    company_id bigint PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_number bigint NOT NULL DEFAULT 0
);

INSERT INTO transaction_delivered_company_counters (company_id, last_number)
SELECT
    delivered_company_id,
    COALESCE(MAX(delivered_number::bigint), 0)::bigint
FROM transactions
WHERE delivered_company_id IS NOT NULL
GROUP BY delivered_company_id
ON CONFLICT (company_id) DO UPDATE
    SET last_number = GREATEST(
        transaction_delivered_company_counters.last_number,
        EXCLUDED.last_number
    );
