-- Xizmat haqi izohi ko'p so'zli matn bo'lishi mumkin (cheklovsiz TEXT).

ALTER TABLE transactions
    ALTER COLUMN service_fee_details TYPE text
    USING service_fee_details::text;

ALTER TABLE transaction_service_fees
    ALTER COLUMN details TYPE text
    USING details::text;

ALTER TABLE service_fee_settlements
    ALTER COLUMN details TYPE text
    USING details::text;
