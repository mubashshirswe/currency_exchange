DROP TRIGGER IF EXISTS trg_transactions_same_business ON transactions;
DROP FUNCTION IF EXISTS assert_transaction_same_business();

DROP TRIGGER IF EXISTS trg_service_fee_settlements_business ON service_fee_settlements;
DROP TRIGGER IF EXISTS trg_transaction_service_fees_business ON transaction_service_fees;
DROP TRIGGER IF EXISTS trg_soft_balance_records_business ON soft_balance_records;
DROP TRIGGER IF EXISTS trg_soft_balances_business ON soft_balances;
DROP TRIGGER IF EXISTS trg_company_balance_records_business ON company_balance_records;
DROP TRIGGER IF EXISTS trg_company_balances_business ON company_balances;
DROP TRIGGER IF EXISTS trg_debts_business_debtor ON debts;
DROP TRIGGER IF EXISTS trg_debts_business ON debts;
DROP TRIGGER IF EXISTS trg_debtors_business ON debtors;
DROP TRIGGER IF EXISTS trg_transactions_business ON transactions;
DROP TRIGGER IF EXISTS trg_exchanges_business ON exchanges;
DROP TRIGGER IF EXISTS trg_balance_records_business ON balance_records;
DROP TRIGGER IF EXISTS trg_balances_business ON balances;

DROP FUNCTION IF EXISTS set_business_id_from_debtor();
DROP FUNCTION IF EXISTS set_business_id_from_company();

ALTER TABLE service_fee_settlements   DROP COLUMN IF EXISTS business_id;
ALTER TABLE transaction_service_fees  DROP COLUMN IF EXISTS business_id;
ALTER TABLE soft_balance_records      DROP COLUMN IF EXISTS business_id;
ALTER TABLE soft_balances             DROP COLUMN IF EXISTS business_id;
ALTER TABLE company_balance_records   DROP COLUMN IF EXISTS business_id;
ALTER TABLE company_balances          DROP COLUMN IF EXISTS business_id;
ALTER TABLE debts                     DROP COLUMN IF EXISTS business_id;
ALTER TABLE debtors                   DROP COLUMN IF EXISTS business_id;
ALTER TABLE transactions              DROP COLUMN IF EXISTS business_id;
ALTER TABLE exchanges                 DROP COLUMN IF EXISTS business_id;
ALTER TABLE balance_records           DROP COLUMN IF EXISTS business_id;
ALTER TABLE balances                  DROP COLUMN IF EXISTS business_id;
