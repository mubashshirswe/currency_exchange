-- Qarzdor ismini xato yozilgan holda ham topish uchun trigram qidiruv.
-- pg_trgm similarity() ishlatiladi: "Aliev" -> "Aliyev" kabi kichik xatolar mos keladi.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_debtors_full_name_trgm
    ON debtors USING gin (LOWER(full_name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_debtors_company_id
    ON debtors (company_id);
