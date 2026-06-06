-- Xizmat haqi faqat SUM valyutasida.
UPDATE transactions
SET service_fee_currency = 'SUM'
WHERE service_fee_amount > 0
  AND upper(coalesce(nullif(trim(service_fee_currency), ''), 'SUM')) != 'SUM';

UPDATE transaction_service_fees
SET currency = 'SUM'
WHERE upper(currency) != 'SUM';

UPDATE service_fee_settlements
SET currency = 'SUM'
WHERE upper(currency) != 'SUM';
