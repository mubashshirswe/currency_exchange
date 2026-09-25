-- "Dostavka" sanog'i endi hisoblagichdan (last_number) emas, transactions
-- jadvalidan hisoblanadi: reset_after_id dan katta id'li "Dostavka"
-- operatsiyalari soni. Hisoblagich update/delete'da siljib ketardi.

ALTER TABLE transaction_delivery_company_counters
    ADD COLUMN IF NOT EXISTS reset_after_id bigint NOT NULL DEFAULT 0;

-- Mavjud reset holatini saqlash: last_number = L bo'lsa, oxirgi L ta
-- "Dostavka" sanoqda qoladi — reset_after_id = (L+1)-chi oxirgisining id'si.
WITH ranked AS (
    SELECT id, received_company_id,
           ROW_NUMBER() OVER (PARTITION BY received_company_id ORDER BY id DESC) AS rn
    FROM transactions
    WHERE trim(service_fee_details) = 'Dostavka'
)
UPDATE transaction_delivery_company_counters c
SET reset_after_id = r.id
FROM ranked r
WHERE r.received_company_id = c.company_id
  AND r.rn = c.last_number + 1;
