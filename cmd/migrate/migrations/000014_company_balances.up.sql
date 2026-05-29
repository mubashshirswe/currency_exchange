CREATE TABLE IF NOT EXISTS company_balances (
    id bigserial PRIMARY KEY,
    company_id bigint NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    currency varchar(255) NOT NULL,
    balance bigint NOT NULL DEFAULT 0,
    in_out_lay bigint NOT NULL DEFAULT 0,
    out_in_lay bigint NOT NULL DEFAULT 0,
    created_at timestamp(0) with time zone NOT NULL DEFAULT now(),
    CONSTRAINT uq_company_balances_company_currency UNIQUE (company_id, currency)
);

CREATE INDEX IF NOT EXISTS idx_company_balances_company_id ON company_balances (company_id);

-- Mavjud user balanslardan kompaniya balansini to'ldirish
INSERT INTO company_balances (company_id, currency, balance, in_out_lay, out_in_lay)
SELECT
    company_id,
    currency,
    COALESCE(SUM(balance), 0),
    COALESCE(SUM(in_out_lay), 0),
    COALESCE(SUM(out_in_lay), 0)
FROM balances
WHERE company_id IS NOT NULL
GROUP BY company_id, currency
ON CONFLICT (company_id, currency) DO UPDATE SET
    balance = EXCLUDED.balance,
    in_out_lay = EXCLUDED.in_out_lay,
    out_in_lay = EXCLUDED.out_in_lay;

ALTER TABLE balance_records
    ALTER COLUMN balance_id DROP NOT NULL;

ALTER TABLE balance_records
    ADD COLUMN IF NOT EXISTS company_balance_id bigint REFERENCES company_balances(id);

CREATE INDEX IF NOT EXISTS idx_balance_records_company_balance_id ON balance_records (company_balance_id);
