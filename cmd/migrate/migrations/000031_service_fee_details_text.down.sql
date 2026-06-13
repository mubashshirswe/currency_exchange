ALTER TABLE transactions
    ALTER COLUMN service_fee_details TYPE varchar
    USING service_fee_details::varchar;

ALTER TABLE transaction_service_fees
    ALTER COLUMN details TYPE varchar(255)
    USING details::varchar(255);

ALTER TABLE service_fee_settlements
    ALTER COLUMN details TYPE varchar(255)
    USING details::varchar(255);
