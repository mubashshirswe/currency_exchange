-- Xizmat puli jadvali — tranzaksiyalardan kelgan xizmat haqlari (taqsimlash uchun).
CREATE TABLE IF NOT EXISTS transaction_service_fees (
    id bigserial PRIMARY KEY,
    transaction_id bigint NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    company_id bigint NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    amount bigint NOT NULL,
    remaining_amount bigint NOT NULL,
    currency varchar(16) NOT NULL DEFAULT 'SUM',
    details varchar(255),
    status bigint NOT NULL DEFAULT 1,
    created_at timestamp(0) with time zone NOT NULL DEFAULT now(),
    CONSTRAINT uq_transaction_service_fees_tx UNIQUE (transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_transaction_service_fees_company
    ON transaction_service_fees (company_id, currency, status);

-- Xizmat pulini 0 qilish (yakunlash) yozuvlari.
CREATE TABLE IF NOT EXISTS service_fee_settlements (
    id bigserial PRIMARY KEY,
    company_id bigint NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id bigint REFERENCES users(id),
    amount bigint NOT NULL,
    currency varchar(16) NOT NULL DEFAULT 'SUM',
    details varchar(255),
    created_at timestamp(0) with time zone NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_service_fee_settlements_company
    ON service_fee_settlements (company_id, currency);

-- Qaysi xizmat haqi qatorlari qaysi yakunlashda ishlatilgani.
CREATE TABLE IF NOT EXISTS service_fee_settlement_items (
    id bigserial PRIMARY KEY,
    settlement_id bigint NOT NULL REFERENCES service_fee_settlements(id) ON DELETE CASCADE,
    service_fee_id bigint NOT NULL REFERENCES transaction_service_fees(id) ON DELETE CASCADE,
    amount bigint NOT NULL
);

-- Mavjud tranzaksiyalardan xizmat haqlarini ko'chirish.
INSERT INTO transaction_service_fees (
    transaction_id, company_id, amount, remaining_amount, currency, details, status
)
SELECT
    t.id,
    CASE
        WHEN t.delivered_user_id IS NOT NULL THEN t.delivered_company_id
        ELSE t.received_company_id
    END,
    t.service_fee_amount,
    t.service_fee_amount,
    upper(coalesce(nullif(trim(t.service_fee_currency), ''), 'SUM')),
    coalesce(t.service_fee_details, ''),
    1
FROM transactions t
WHERE t.service_fee_amount > 0
  AND t.status != 3
ON CONFLICT (transaction_id) DO NOTHING;
