DROP INDEX IF EXISTS idx_debts_transaction_id;

ALTER TABLE debts DROP COLUMN IF EXISTS transaction_stage;
ALTER TABLE debts DROP COLUMN IF EXISTS transaction_id;
