-- Soft balance — biznes egasining daromadi (kompaniya operatsion balansidan MUSTAQIL).
CREATE TABLE IF NOT EXISTS soft_balances (
    id bigserial PRIMARY KEY,
    company_id bigint NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    currency varchar(255) NOT NULL,
    balance bigint NOT NULL DEFAULT 0,
    created_at timestamp(0) with time zone NOT NULL DEFAULT now(),
    CONSTRAINT uq_soft_balances_company_currency UNIQUE (company_id, currency)
);

CREATE TABLE IF NOT EXISTS soft_balance_records (
    id bigserial PRIMARY KEY,
    company_id bigint NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id bigint REFERENCES users(id),
    soft_balance_id bigint REFERENCES soft_balances(id),
    amount bigint NOT NULL,
    currency varchar(255) NOT NULL,
    type bigint NOT NULL,
    details varchar(255),
    status bigint NOT NULL DEFAULT 1,
    created_at timestamp(0) with time zone NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_soft_balance_records_company_id
    ON soft_balance_records (company_id);
CREATE INDEX IF NOT EXISTS idx_soft_balance_records_company_currency
    ON soft_balance_records (company_id, currency);

-- Mavjud kompaniyalar uchun default USD/SUM qatorlari.
INSERT INTO soft_balances (company_id, currency, balance)
SELECT c.id, v.currency, 0
FROM companies c
CROSS JOIN (VALUES ('USD'), ('SUM')) AS v(currency)
ON CONFLICT (company_id, currency) DO NOTHING;
