-- Transaction'dan yaratilgan qarz yozuvlarini transaction bilan bog'laydi.
-- Bu bog'lanish transaction yangilanganda/o'chirilganda qarzni ham to'g'ri
-- qaytarish (reverse) uchun kerak. transaction_stage: 1 = yaratishda,
-- 2 = yakunlashda yozilgan qarz.
ALTER TABLE debts
    ADD COLUMN IF NOT EXISTS transaction_id bigint REFERENCES transactions(id) ON DELETE SET NULL;

ALTER TABLE debts
    ADD COLUMN IF NOT EXISTS transaction_stage smallint NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_debts_transaction_id ON debts (transaction_id);
