ALTER TABLE transactions
    ADD COLUMN completed_at timestamp(0) with time zone NULL;
