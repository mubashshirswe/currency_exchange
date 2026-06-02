-- company_balances jadvalini balances bilan avtomatik sinxron ushlaydi.
-- balances har qanday o'zgarsa (INSERT/UPDATE/DELETE) — exchange, transaction,
-- debt, balance-record (kirim/chiqim) — tegishli (company_id, currency) bo'yicha
-- company_balances qayta hisoblanadi. Drift bo'lmaydi, Go kodga tegilmaydi.

CREATE OR REPLACE FUNCTION sync_company_balance() RETURNS trigger AS $$
DECLARE
    c_id bigint;
    cur  varchar;
BEGIN
    -- Yangi/joriy holatdagi bucket
    IF (TG_OP = 'DELETE') THEN
        c_id := OLD.company_id;
        cur  := OLD.currency;
    ELSE
        c_id := NEW.company_id;
        cur  := NEW.currency;
    END IF;

    IF c_id IS NOT NULL AND cur IS NOT NULL AND cur <> '' THEN
        INSERT INTO company_balances (company_id, currency, balance, in_out_lay, out_in_lay)
        SELECT c_id, cur,
               COALESCE(SUM(balance), 0),
               COALESCE(SUM(in_out_lay), 0),
               COALESCE(SUM(out_in_lay), 0)
        FROM balances
        WHERE company_id = c_id AND currency = cur
        ON CONFLICT (company_id, currency) DO UPDATE
            SET balance    = EXCLUDED.balance,
                in_out_lay = EXCLUDED.in_out_lay,
                out_in_lay = EXCLUDED.out_in_lay;
    END IF;

    -- UPDATE company_id yoki currency'ni o'zgartirsa, eski bucket'ni ham qayta hisoblaymiz
    IF (TG_OP = 'UPDATE')
       AND (OLD.company_id IS DISTINCT FROM NEW.company_id
            OR OLD.currency IS DISTINCT FROM NEW.currency)
       AND OLD.company_id IS NOT NULL
       AND OLD.currency IS NOT NULL
       AND OLD.currency <> '' THEN
        INSERT INTO company_balances (company_id, currency, balance, in_out_lay, out_in_lay)
        SELECT OLD.company_id, OLD.currency,
               COALESCE(SUM(balance), 0),
               COALESCE(SUM(in_out_lay), 0),
               COALESCE(SUM(out_in_lay), 0)
        FROM balances
        WHERE company_id = OLD.company_id AND currency = OLD.currency
        ON CONFLICT (company_id, currency) DO UPDATE
            SET balance    = EXCLUDED.balance,
                in_out_lay = EXCLUDED.in_out_lay,
                out_in_lay = EXCLUDED.out_in_lay;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sync_company_balance ON balances;

CREATE TRIGGER trg_sync_company_balance
    AFTER INSERT OR UPDATE OR DELETE ON balances
    FOR EACH ROW EXECUTE FUNCTION sync_company_balance();

-- Mavjud ma'lumotni bir martalik to'liq sinxronlash (trigger faqat kelajakdagi
-- o'zgarishlarda ishlagani uchun).
INSERT INTO company_balances (company_id, currency, balance, in_out_lay, out_in_lay)
SELECT company_id, currency,
       COALESCE(SUM(balance), 0),
       COALESCE(SUM(in_out_lay), 0),
       COALESCE(SUM(out_in_lay), 0)
FROM balances
WHERE company_id IS NOT NULL AND currency IS NOT NULL AND currency <> ''
GROUP BY company_id, currency
ON CONFLICT (company_id, currency) DO UPDATE
    SET balance    = EXCLUDED.balance,
        in_out_lay = EXCLUDED.in_out_lay,
        out_in_lay = EXCLUDED.out_in_lay;
