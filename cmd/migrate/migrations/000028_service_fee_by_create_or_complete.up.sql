-- Xizmat haqi kompaniyasi: yaratishda kiritilgan -> received, yakunlashda kiritilgan -> delivered.
-- Yaratishda kiritilgan (fee yozuvi tranzaksiya bilan bir vaqtda): received_company_id.
-- Yakunlashda kiritilgan (fee yozuvi keyinroq yaratilgan): delivered_company_id.

UPDATE transaction_service_fees f
SET company_id = t.delivered_company_id
FROM transactions t
WHERE f.transaction_id = t.id
  AND t.delivered_user_id IS NOT NULL
  AND t.delivered_company_id IS NOT NULL
  AND t.service_fee_amount > 0
  AND f.company_id IS DISTINCT FROM t.delivered_company_id
  AND f.created_at > t.created_at + interval '1 minute';
