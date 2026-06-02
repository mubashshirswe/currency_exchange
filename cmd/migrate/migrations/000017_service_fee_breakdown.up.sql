-- Xizmat puli (service fee) endi summa + valyuta + izoh ko'rinishida saqlanadi.
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS service_fee_amount bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS service_fee_currency varchar(16) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS service_fee_details varchar;

-- Eski matnli service_fee qiymatlarini yo'qotmaslik uchun izohga ko'chiramiz.
UPDATE transactions
SET service_fee_details = service_fee
WHERE service_fee IS NOT NULL
  AND service_fee <> ''
  AND (service_fee_details IS NULL OR service_fee_details = '');
