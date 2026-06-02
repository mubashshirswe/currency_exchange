-- company_balances endi balances'dan MUSTAQIL. Sinxron triggerni olib tashlaymiz:
-- foydalanuvchi balansi (balances) va kompaniya balansi (company_balances) bir-biriga
-- bog'liq emas. Kompaniya balansi faqat company_balance_records orqali boshqariladi.
DROP TRIGGER IF EXISTS trg_sync_company_balance ON balances;
DROP FUNCTION IF EXISTS sync_company_balance();

-- Kompaniya balansi yozuvlari (kirim/chiqim). Eski balance_records'dan butunlay alohida.
-- type: 2 = kirim (BUY), 1 = chiqim (SELL) — mobil model bilan bir xil.
CREATE TABLE IF NOT EXISTS company_balance_records (
    id bigserial PRIMARY KEY,
    company_id bigint NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id bigint REFERENCES users(id),
    company_balance_id bigint REFERENCES company_balances(id),
    amount bigint NOT NULL,
    currency varchar(255) NOT NULL,
    type bigint NOT NULL,
    details varchar(255),
    status bigint NOT NULL DEFAULT 1,
    created_at timestamp(0) with time zone NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_company_balance_records_company_id ON company_balance_records (company_id);
CREATE INDEX IF NOT EXISTS idx_company_balance_records_company_currency ON company_balance_records (company_id, currency);

-- Mustaqil ledger toza boshlanishi uchun company_balances qiymatlarini 0 ga tushiramiz
-- (avvalgi qiymatlar user balanslaridan trigger orqali kelgan edi). Valyuta qatorlari qoladi.
UPDATE company_balances SET balance = 0, in_out_lay = 0, out_in_lay = 0;
