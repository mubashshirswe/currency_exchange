ALTER TABLE soft_balance_records
    DROP CONSTRAINT IF EXISTS soft_balance_records_user_id_fkey;
ALTER TABLE soft_balance_records
    ADD CONSTRAINT soft_balance_records_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE company_balance_records
    DROP CONSTRAINT IF EXISTS company_balance_records_user_id_fkey;
ALTER TABLE company_balance_records
    ADD CONSTRAINT company_balance_records_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE balance_records
    DROP CONSTRAINT IF EXISTS balance_records_user_id_fkey;
ALTER TABLE balance_records
    ADD CONSTRAINT balance_records_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE exchanges
    DROP CONSTRAINT IF EXISTS exchanges_user_id_fkey;
ALTER TABLE exchanges
    ADD CONSTRAINT exchanges_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);
