-- Business sozlamalari: har bir tenant o'z ish oqimini tanlaydi.
-- transaction_flow:
--   1 = SIMPLE      — yaratish => topshirish (hozirgi holat)
--   2 = THREE_STAGE — yaratish => qabul qilish => topshirish
-- Qabul qilish bosqichi balansga ta'sir qilmaydi: pul harakati avvalgidek
-- yaratish (received_incomes) va topshirish (delivered_outcomes) bosqichlarida.

CREATE TABLE IF NOT EXISTS business_settings (
    business_id bigint PRIMARY KEY REFERENCES businesses(id) ON DELETE CASCADE,
    transaction_flow bigint NOT NULL DEFAULT 1,
    created_at timestamp(0) with time zone NOT NULL DEFAULT now(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT now()
);

ALTER TABLE business_settings
    DROP CONSTRAINT IF EXISTS chk_business_settings_transaction_flow;
ALTER TABLE business_settings
    ADD CONSTRAINT chk_business_settings_transaction_flow CHECK (transaction_flow IN (1, 2));

-- Mavjud businesslar oddiy (SIMPLE) oqimda qoladi.
INSERT INTO business_settings (business_id)
SELECT id FROM businesses
ON CONFLICT (business_id) DO NOTHING;

-- Qabul qilish bosqichi izlari: kim, qaysi kompaniya, qachon qabul qilgan.
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS accepted_user_id bigint REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS accepted_company_id bigint REFERENCES companies(id) ON DELETE SET NULL;
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS accepted_at timestamp(0) with time zone;

CREATE INDEX IF NOT EXISTS idx_transactions_accepted_company
    ON transactions (accepted_company_id)
    WHERE accepted_company_id IS NOT NULL;
