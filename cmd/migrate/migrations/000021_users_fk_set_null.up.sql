-- User o'chirilganda bog'langan yozuvlarda user_id NULL bo'ladi (tarix saqlanadi).
ALTER TABLE exchanges
    DROP CONSTRAINT IF EXISTS exchanges_user_id_fkey;
ALTER TABLE exchanges
    ADD CONSTRAINT exchanges_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE balance_records
    DROP CONSTRAINT IF EXISTS balance_records_user_id_fkey;
ALTER TABLE balance_records
    ADD CONSTRAINT balance_records_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE company_balance_records
    DROP CONSTRAINT IF EXISTS company_balance_records_user_id_fkey;
ALTER TABLE company_balance_records
    ADD CONSTRAINT company_balance_records_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE soft_balance_records
    DROP CONSTRAINT IF EXISTS soft_balance_records_user_id_fkey;
ALTER TABLE soft_balance_records
    ADD CONSTRAINT soft_balance_records_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
