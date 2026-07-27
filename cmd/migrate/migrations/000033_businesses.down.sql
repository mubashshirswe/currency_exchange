DROP TRIGGER IF EXISTS trg_users_business_guard ON users;
DROP FUNCTION IF EXISTS assert_user_company_same_business();

DROP INDEX IF EXISTS uq_users_business_phone;
DROP INDEX IF EXISTS idx_users_business_company;
DROP INDEX IF EXISTS idx_users_business_id;
DROP INDEX IF EXISTS idx_companies_business_id;

ALTER TABLE users DROP COLUMN IF EXISTS business_id;
ALTER TABLE companies DROP COLUMN IF EXISTS business_id;

DROP TABLE IF EXISTS businesses;
