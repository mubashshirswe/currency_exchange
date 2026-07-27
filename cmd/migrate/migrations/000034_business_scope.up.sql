-- Barcha operatsion jadvallarga business_id qo'shiladi: izolyatsiya bitta
-- WHERE business_id = $1 bilan ta'minlanadi, kompaniya bo'ylab JOIN kerak emas.

ALTER TABLE balances                  ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE balance_records           ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE exchanges                 ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE transactions              ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE debtors                   ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE debts                     ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE company_balances          ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE company_balance_records   ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE soft_balances             ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE soft_balance_records      ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE transaction_service_fees  ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;
ALTER TABLE service_fee_settlements   ADD COLUMN IF NOT EXISTS business_id bigint REFERENCES businesses(id) ON DELETE CASCADE;

-- Backfill: har bir yozuv o'z kompaniyasining businessiga tegishli.
UPDATE balances t                 SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE balance_records t          SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE exchanges t                SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE debtors t                  SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE debts t                    SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE company_balances t         SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE company_balance_records t  SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE soft_balances t            SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE soft_balance_records t     SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE transaction_service_fees t SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;
UPDATE service_fee_settlements t  SET business_id = c.business_id FROM companies c WHERE t.company_id = c.id AND t.business_id IS NULL;

UPDATE transactions t SET business_id = c.business_id FROM companies c WHERE t.received_company_id  = c.id AND t.business_id IS NULL;
UPDATE transactions t SET business_id = c.business_id FROM companies c WHERE t.delivered_company_id = c.id AND t.business_id IS NULL;

-- Kompaniyasi yo'q eski yozuvlar (company_id NULL) — userning businessiga bog'lanadi.
UPDATE balances t        SET business_id = u.business_id FROM users u WHERE t.user_id = u.id AND t.business_id IS NULL;
UPDATE balance_records t SET business_id = u.business_id FROM users u WHERE t.user_id = u.id AND t.business_id IS NULL;
UPDATE exchanges t       SET business_id = u.business_id FROM users u WHERE t.user_id = u.id AND t.business_id IS NULL;
UPDATE debtors t         SET business_id = u.business_id FROM users u WHERE t.user_id = u.id AND t.business_id IS NULL;
UPDATE debts t           SET business_id = u.business_id FROM users u WHERE t.user_id = u.id AND t.business_id IS NULL;

-- Insert paytida business_id avtomatik to'ldiriladi: TG_ARGV — nomzod kompaniya
-- ustunlari ro'yxati, birinchi NOT NULL qiymati ishlatiladi.
CREATE OR REPLACE FUNCTION set_business_id_from_company() RETURNS trigger AS $$
DECLARE
    payload jsonb;
    candidate text;
    company bigint;
    resolved bigint;
BEGIN
    IF NEW.business_id IS NOT NULL THEN
        RETURN NEW;
    END IF;

    payload := to_jsonb(NEW);

    FOREACH candidate IN ARRAY TG_ARGV LOOP
        company := NULLIF(payload ->> candidate, '')::bigint;
        IF company IS NOT NULL THEN
            SELECT business_id INTO resolved FROM companies WHERE id = company;
            IF resolved IS NOT NULL THEN
                NEW.business_id := resolved;
                RETURN NEW;
            END IF;
        END IF;
    END LOOP;

    -- Kompaniya topilmadi: user orqali aniqlashga urinamiz.
    company := NULLIF(payload ->> 'user_id', '')::bigint;
    IF company IS NOT NULL THEN
        SELECT business_id INTO resolved FROM users WHERE id = company;
        NEW.business_id := resolved;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- debts/soft yozuvlari uchun debtor orqali aniqlash.
CREATE OR REPLACE FUNCTION set_business_id_from_debtor() RETURNS trigger AS $$
DECLARE
    resolved bigint;
BEGIN
    IF NEW.business_id IS NOT NULL OR NEW.debtor_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT business_id INTO resolved FROM debtors WHERE id = NEW.debtor_id;
    NEW.business_id := resolved;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_balances_business ON balances;
CREATE TRIGGER trg_balances_business BEFORE INSERT ON balances
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_balance_records_business ON balance_records;
CREATE TRIGGER trg_balance_records_business BEFORE INSERT ON balance_records
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_exchanges_business ON exchanges;
CREATE TRIGGER trg_exchanges_business BEFORE INSERT ON exchanges
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_transactions_business ON transactions;
CREATE TRIGGER trg_transactions_business BEFORE INSERT ON transactions
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('received_company_id', 'delivered_company_id');

DROP TRIGGER IF EXISTS trg_debtors_business ON debtors;
CREATE TRIGGER trg_debtors_business BEFORE INSERT ON debtors
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_debts_business ON debts;
CREATE TRIGGER trg_debts_business BEFORE INSERT ON debts
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_debts_business_debtor ON debts;
CREATE TRIGGER trg_debts_business_debtor BEFORE INSERT ON debts
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_debtor();

DROP TRIGGER IF EXISTS trg_company_balances_business ON company_balances;
CREATE TRIGGER trg_company_balances_business BEFORE INSERT ON company_balances
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_company_balance_records_business ON company_balance_records;
CREATE TRIGGER trg_company_balance_records_business BEFORE INSERT ON company_balance_records
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_soft_balances_business ON soft_balances;
CREATE TRIGGER trg_soft_balances_business BEFORE INSERT ON soft_balances
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_soft_balance_records_business ON soft_balance_records;
CREATE TRIGGER trg_soft_balance_records_business BEFORE INSERT ON soft_balance_records
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_transaction_service_fees_business ON transaction_service_fees;
CREATE TRIGGER trg_transaction_service_fees_business BEFORE INSERT ON transaction_service_fees
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

DROP TRIGGER IF EXISTS trg_service_fee_settlements_business ON service_fee_settlements;
CREATE TRIGGER trg_service_fee_settlements_business BEFORE INSERT ON service_fee_settlements
    FOR EACH ROW EXECUTE FUNCTION set_business_id_from_company('company_id');

CREATE INDEX IF NOT EXISTS idx_balances_business                 ON balances (business_id);
CREATE INDEX IF NOT EXISTS idx_balance_records_business          ON balance_records (business_id);
CREATE INDEX IF NOT EXISTS idx_exchanges_business                ON exchanges (business_id, status);
CREATE INDEX IF NOT EXISTS idx_transactions_business             ON transactions (business_id, status);
CREATE INDEX IF NOT EXISTS idx_debtors_business                  ON debtors (business_id);
CREATE INDEX IF NOT EXISTS idx_debts_business                    ON debts (business_id);
CREATE INDEX IF NOT EXISTS idx_company_balances_business         ON company_balances (business_id);
CREATE INDEX IF NOT EXISTS idx_company_balance_records_business  ON company_balance_records (business_id);
CREATE INDEX IF NOT EXISTS idx_soft_balances_business            ON soft_balances (business_id);
CREATE INDEX IF NOT EXISTS idx_soft_balance_records_business     ON soft_balance_records (business_id);
CREATE INDEX IF NOT EXISTS idx_transaction_service_fees_business ON transaction_service_fees (business_id, currency, status);
CREATE INDEX IF NOT EXISTS idx_service_fee_settlements_business  ON service_fee_settlements (business_id, currency);

-- Tranzaksiya faqat bitta business ichidagi kompaniyalar orasida bo'ladi.
CREATE OR REPLACE FUNCTION assert_transaction_same_business() RETURNS trigger AS $$
DECLARE
    received_business bigint;
    delivered_business bigint;
BEGIN
    SELECT business_id INTO received_business  FROM companies WHERE id = NEW.received_company_id;
    SELECT business_id INTO delivered_business FROM companies WHERE id = NEW.delivered_company_id;

    IF received_business IS NOT NULL
       AND delivered_business IS NOT NULL
       AND received_business <> delivered_business THEN
        RAISE EXCEPTION 'tranzaksiya turli businesslar orasida bo''lishi mumkin emas (% <> %)',
            received_business, delivered_business;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_transactions_same_business ON transactions;
CREATE TRIGGER trg_transactions_same_business
    BEFORE INSERT OR UPDATE OF received_company_id, delivered_company_id ON transactions
    FOR EACH ROW EXECUTE FUNCTION assert_transaction_same_business();
