-- Xizmat haqi tranzaksiyani yaratgan kompaniyaga (received_company_id) bog'lanadi.
UPDATE transaction_service_fees f
SET company_id = t.received_company_id
FROM transactions t
WHERE f.transaction_id = t.id
  AND t.received_company_id IS NOT NULL
  AND f.company_id IS DISTINCT FROM t.received_company_id;
