-- Qabul qilingan (status = 4) tranzaksiyalar oddiy "yaratilgan" holatga qaytariladi.
UPDATE transactions SET status = 1 WHERE status = 4;

DROP INDEX IF EXISTS idx_transactions_accepted_company;

ALTER TABLE transactions DROP COLUMN IF EXISTS accepted_at;
ALTER TABLE transactions DROP COLUMN IF EXISTS accepted_company_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS accepted_user_id;

DROP TABLE IF EXISTS business_settings;
