-- v2 operatsiyalari (exchange/transaction/debt) kompaniya balansiga ta'sir qilganda,
-- har bir yozuv qaysi operatsiyadan kelganini bog'lash uchun id ustunlari.
ALTER TABLE company_balance_records
    ADD COLUMN IF NOT EXISTS exchange_id bigint REFERENCES exchanges(id),
    ADD COLUMN IF NOT EXISTS transaction_id bigint REFERENCES transactions(id),
    ADD COLUMN IF NOT EXISTS debt_id bigint REFERENCES debts(id);
