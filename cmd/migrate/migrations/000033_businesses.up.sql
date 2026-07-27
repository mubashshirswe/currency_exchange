-- Business (tenant) qatlami: business => company => users.
-- Har bir business boshqasidan to'liq izolyatsiya qilingan: barcha ma'lumot
-- business_id bo'yicha ajratiladi.

CREATE TABLE IF NOT EXISTS businesses (
    id bigserial PRIMARY KEY,
    name varchar(255) NOT NULL,
    details varchar(255),
    phone varchar(255),
    status bigint NOT NULL DEFAULT 1,
    created_at timestamp(0) with time zone NOT NULL DEFAULT now(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT now()
);

ALTER TABLE companies
    ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;

-- Mavjud ma'lumotlar bitta default business ostiga o'tkaziladi.
INSERT INTO businesses (id, name, details, status)
SELECT 1, 'Default Business', 'migratsiyada yaratilgan', 1
WHERE EXISTS (SELECT 1 FROM companies)
   OR EXISTS (SELECT 1 FROM users)
ON CONFLICT (id) DO NOTHING;

UPDATE companies SET business_id = 1 WHERE business_id IS NULL;

-- User o'z kompaniyasining businessiga tegishli; kompaniyasi yo'q userlar default'ga.
UPDATE users u
SET business_id = c.business_id
FROM companies c
WHERE u.company_id = c.id
  AND u.business_id IS NULL;

UPDATE users SET business_id = 1 WHERE business_id IS NULL AND EXISTS (SELECT 1 FROM businesses WHERE id = 1);

-- Sequence'ni qo'lda kiritilgan id=1 dan keyinga suramiz.
SELECT setval('businesses_id_seq', GREATEST((SELECT COALESCE(MAX(id), 1) FROM businesses), 1));

ALTER TABLE companies ALTER COLUMN business_id SET NOT NULL;
ALTER TABLE users ALTER COLUMN business_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_companies_business_id ON companies (business_id);
CREATE INDEX IF NOT EXISTS idx_users_business_id ON users (business_id);
CREATE INDEX IF NOT EXISTS idx_users_business_company ON users (business_id, company_id);

-- Telefon business ichida unikal bo'lishi shart. Mavjud bazada takror telefonlar
-- bo'lsa, indeks yaratishdan oldin tushunarli xato beramiz.
DO $$
DECLARE
    duplicate record;
BEGIN
    SELECT business_id, phone, count(*) AS total
    INTO duplicate
    FROM users
    WHERE phone <> ''
    GROUP BY business_id, phone
    HAVING count(*) > 1
    LIMIT 1;

    IF FOUND THEN
        RAISE EXCEPTION 'takrorlanuvchi telefon: business_id=% phone=% (% ta user). Migratsiyadan oldin tozalang.',
            duplicate.business_id, duplicate.phone, duplicate.total;
    END IF;
END $$;

-- O'chirilgan user phone = '' bo'lgani uchun bo'sh qiymatlar indeksdan chiqariladi.
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_business_phone
    ON users (business_id, phone)
    WHERE phone <> '';

-- User doim o'zi tegishli business ichidagi kompaniyada bo'lishi shart.
CREATE OR REPLACE FUNCTION assert_user_company_same_business() RETURNS trigger AS $$
DECLARE
    company_business bigint;
BEGIN
    IF NEW.company_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT business_id INTO company_business FROM companies WHERE id = NEW.company_id;

    IF company_business IS NULL THEN
        RETURN NEW;
    END IF;

    IF NEW.business_id IS NULL THEN
        NEW.business_id := company_business;
    ELSIF NEW.business_id <> company_business THEN
        RAISE EXCEPTION 'user business_id (%) kompaniya business_id (%) bilan mos emas',
            NEW.business_id, company_business;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_business_guard ON users;
CREATE TRIGGER trg_users_business_guard
    BEFORE INSERT OR UPDATE OF company_id, business_id ON users
    FOR EACH ROW EXECUTE FUNCTION assert_user_company_same_business();
