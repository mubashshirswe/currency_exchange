package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mubashshir3767/currencyExchange/internal/notify"
	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// CompanyOpsService — v2: exchange/transaction/debt operatsiyalari KOMPANIYA balansiga
// ta'sir qiladi (company_balances + company_balance_records). User balanslarga (balances)
// va eski balance_records'ga TEGMAYDI. Eski v1 servislar o'zgarmaydi.
type CompanyOpsService struct {
	store  store.Storage
	notify notify.DeliveredUser
}

func NewCompanyOpsService(store store.Storage, delivered notify.DeliveredUser) *CompanyOpsService {
	if delivered == nil {
		delivered = notify.NoopDeliveredUser{}
	}
	return &CompanyOpsService{store: store, notify: delivered}
}

type opLink struct {
	ExchangeId    *int64
	TransactionId *int64
	DebtId        *int64
}

// applyCompanyOp — bitta kirim/chiqimni company_balances'ga qo'llaydi va
// company_balance_records'ga yozadi (hodim user_id + bog'langan operatsiya id bilan).
func applyCompanyOp(
	ctx context.Context,
	cbStorage *store.CompanyBalanceStorage,
	cbrStorage *store.CompanyBalanceRecordStorage,
	companyID, userID int64,
	currency string,
	amount, recordType int64,
	details string,
	link opLink,
) error {
	if amount <= 0 {
		return nil
	}

	cb, err := cbStorage.GetByCompanyIdAndCurrency(ctx, companyID, currency)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		cb = &store.CompanyBalance{CompanyID: companyID, Currency: currency}
		if cerr := cbStorage.Create(ctx, cb); cerr != nil {
			return cerr
		}
	}

	if err := applyCompanyBalance(cb, recordType, amount); err != nil {
		return err
	}
	if err := cbStorage.Update(ctx, cb); err != nil {
		return err
	}

	rec := &store.CompanyBalanceRecord{
		CompanyID:        companyID,
		UserID:           userID,
		CompanyBalanceID: cb.ID,
		Amount:           amount,
		Currency:         currency,
		Type:             recordType,
		Details:          details,
		Status:           store.STATUS_CREATED,
		ExchangeId:       link.ExchangeId,
		TransactionId:    link.TransactionId,
		DebtId:           link.DebtId,
	}
	return cbrStorage.Create(ctx, rec)
}

func (s *CompanyOpsService) companyOf(ctx context.Context, userID int64) (int64, error) {
	u, err := s.store.Users.GetById(ctx, &userID)
	if err != nil {
		return 0, err
	}
	return u.CompanyId, nil
}

// reverseAndDeleteByLink — bog'langan operatsiya yozuvlarini company_balances'ga teskari
// qo'llaydi va o'chiradi. Update/Delete v2'da eski ta'sirni bekor qilish uchun ishlatiladi.
func reverseAndDeleteByLink(
	ctx context.Context,
	cbStorage *store.CompanyBalanceStorage,
	cbrStorage *store.CompanyBalanceRecordStorage,
	field string,
	id int64,
) error {
	recs, err := cbrStorage.ListByLink(ctx, field, id)
	if err != nil {
		return err
	}
	for _, rec := range recs {
		cb, err := cbStorage.GetByCompanyIdAndCurrency(ctx, rec.CompanyID, rec.Currency)
		if err != nil {
			return err
		}
		if err := reverseCompanyBalance(cb, rec.Type, rec.Amount); err != nil {
			return err
		}
		if err := cbStorage.Update(ctx, cb); err != nil {
			return err
		}
		if err := cbrStorage.Delete(ctx, rec.ID); err != nil {
			return err
		}
	}
	return nil
}

// CreateExchangeV2 — exchange yaratadi va kompaniya balansiga ta'sir qiladi.
// received => kirim (BUY), selled => chiqim (SELL). exchange.UserId = amalni bajargan hodim.
func (s *CompanyOpsService) CreateExchangeV2(ctx context.Context, exchange *store.Exchange) error {
	companyID, err := s.companyOf(ctx, exchange.UserId)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	exchangeStore := store.NewExchangeStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	exchange.CompanyID = companyID
	if err := exchangeStore.Create(ctx, exchange); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE CREATING EXCHANGE %v", err)
	}

	link := opLink{ExchangeId: &exchange.ID}

	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.ReceivedCurrency, exchange.ReceivedMoney, TYPE_BUY, exchange.Details, link); err != nil {
		return err
	}

	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.SelledCurrency, exchange.SelledMoney, TYPE_SELL, exchange.Details, link); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateExchangeV2 — exchange'ni yangilaydi: eski company balans ta'sirini bekor qiladi,
// exchange qatorini yangilaydi va yangi ta'sirni qo'llaydi. User balanslarga tegmaydi.
func (s *CompanyOpsService) UpdateExchangeV2(ctx context.Context, exchange *store.Exchange) error {
	companyID, err := s.companyOf(ctx, exchange.UserId)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	exchangeStore := store.NewExchangeStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "exchange_id", exchange.ID); err != nil {
		return err
	}

	exchange.CompanyID = companyID
	if err := exchangeStore.Update(ctx, exchange); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING EXCHANGE %v", err)
	}

	link := opLink{ExchangeId: &exchange.ID}
	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.ReceivedCurrency, exchange.ReceivedMoney, TYPE_BUY, exchange.Details, link); err != nil {
		return err
	}
	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.SelledCurrency, exchange.SelledMoney, TYPE_SELL, exchange.Details, link); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteExchangeV2 — exchange'ni o'chiradi va company balans ta'sirini bekor qiladi.
func (s *CompanyOpsService) DeleteExchangeV2(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	exchangeStore := store.NewExchangeStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "exchange_id", id); err != nil {
		return err
	}
	if err := exchangeStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE DELETING EXCHANGE %v", err)
	}

	return tx.Commit()
}

// PerformTransactionV2 — transaction yaratadi; received_incomes kompaniya balansiga ta'sir qiladi.
// SELL => chiqim, BUY => kirim. actingUserID = amalni bajargan hodim (JWT).
func (s *CompanyOpsService) PerformTransactionV2(ctx context.Context, transaction *store.Transaction, actingUserID int64) error {
	companyID, err := s.companyOf(ctx, actingUserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if err := transactionsStorage.Create(ctx, transaction); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE Transactions.Create %v", err)
	}

	link := opLink{TransactionId: &transaction.ID}
	for _, tr := range transaction.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, transaction.Type, transaction.Details, link); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if transaction.DeliveredUserId != nil {
		uid := *transaction.DeliveredUserId
		tid := transaction.ID
		phone := transaction.Phone
		details := transaction.Details
		go func() {
			ctxN, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			s.notify.NotifyPendingDelivery(ctxN, &uid, tid, phone, details)
		}()
	}
	return nil
}

// CompleteTransactionV2 — transaction yakunlaydi; delivered_outcomes kompaniya balansiga ta'sir qiladi.
// SELL transaction => yetkazib berish kirim (BUY); BUY transaction => chiqim (SELL).
func (s *CompanyOpsService) CompleteTransactionV2(ctx context.Context, complete types.TransactionComplete, actingUserID int64) error {
	companyID, err := s.companyOf(ctx, actingUserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	tran, err := transactionsStorage.GetById(ctx, complete.TransactionID)
	if err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE transactionsStorage.GetById %v", err)
	}

	link := opLink{TransactionId: &tran.ID}
	for _, tr := range tran.DeliveredOutcomes {
		var recordType int64
		if tran.Type == TYPE_SELL {
			recordType = TYPE_BUY // kirim
		} else {
			recordType = TYPE_SELL // chiqim
		}
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
			tr.DeliveredCurrency, tr.DeliveredAmount, recordType, tran.Details, link); err != nil {
			return err
		}
	}

	tran.Status = TRANSACTION_STATUS_COMPLETED
	tran.DeliveredUserId = &complete.DeliveredUserId
	if err := transactionsStorage.Update(ctx, tran); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE transactionsStorage.Update %v", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	deliveredID := complete.DeliveredUserId
	tid := tran.ID
	details := tran.Details
	go func() {
		ctxN, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		s.notify.NotifyDeliveryCompleted(ctxN, deliveredID, tid, details)
	}()
	return nil
}

// UpdateTransactionV2 — transaction'ni yangilaydi: eski company balans ta'sirini bekor qiladi,
// transaction qatorini yangilaydi va yangi ta'sirni qayta qo'llaydi (received + delivered).
func (s *CompanyOpsService) UpdateTransactionV2(ctx context.Context, transaction *store.Transaction, actingUserID int64) error {
	companyID, err := s.companyOf(ctx, actingUserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "transaction_id", transaction.ID); err != nil {
		return err
	}

	if err := transactionsStorage.Update(ctx, transaction); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING TRANSACTION %v", err)
	}

	link := opLink{TransactionId: &transaction.ID}

	if transaction.ReceivedUserId != 0 {
		for _, tr := range transaction.ReceivedIncomes {
			if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
				tr.ReceivedCurrency, tr.ReceivedAmount, transaction.Type, transaction.Details, link); err != nil {
				return err
			}
		}
	}

	if transaction.DeliveredUserId != nil {
		for _, tr := range transaction.DeliveredOutcomes {
			var recordType int64
			if transaction.Type == TYPE_SELL {
				recordType = TYPE_BUY
			} else {
				recordType = TYPE_SELL
			}
			if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
				tr.DeliveredCurrency, tr.DeliveredAmount, recordType, transaction.Details, link); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// DeleteTransactionV2 — transaction'ni o'chiradi va company balans ta'sirini bekor qiladi.
func (s *CompanyOpsService) DeleteTransactionV2(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "transaction_id", id); err != nil {
		return err
	}
	if err := transactionsStorage.Delete(ctx, &id); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE DELETING TRANSACTION %v", err)
	}

	return tx.Commit()
}

// CreateDebtV2 — debtor + debt yaratadi; received_incomes kompaniya balansiga ta'sir qiladi.
// SELL => chiqim, BUY => kirim. Debtor balansi (debtors jadvali) v1'dagidek yuritiladi.
func (s *CompanyOpsService) CreateDebtV2(ctx context.Context, debt *store.Debts) error {
	if len(debt.ReceivedIncomes) == 0 {
		return fmt.Errorf("received incomes cannot be empty")
	}

	companyID, err := s.companyOf(ctx, debt.UserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtorsStorage := store.NewDebtorsStorage(tx)
	debtsStorage := store.NewDebtsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	debtor := &store.Debtors{
		FullName:  debt.FullName,
		Balance:   0,
		Currency:  debt.DebtedCurrency,
		UserID:    debt.UserID,
		CompanyID: companyID,
		Phone:     debt.Phone,
	}
	if err := debtorsStorage.Create(ctx, debtor); err != nil {
		return fmt.Errorf("failed to create debtor: %w", err)
	}

	originalPositiveDebted := debt.DebtedAmount
	switch debt.Type {
	case types.TYPE_SELL:
		debt.DebtedAmount = -debt.DebtedAmount
		debt.State = 1
	case types.TYPE_BUY:
		// stays positive
	default:
		return fmt.Errorf("invalid debt type: %d", debt.Type)
	}
	debt.DebtorID = debtor.ID
	debt.CompanyID = companyID

	if err := debtsStorage.Create(ctx, debt); err != nil {
		return fmt.Errorf("failed to create debt: %w", err)
	}

	link := opLink{DebtId: &debt.ID}
	for _, tr := range debt.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, debt.UserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, int64(debt.Type), debt.Details, link); err != nil {
			return err
		}
	}

	if debt.Type == types.TYPE_SELL {
		debtor.Balance -= originalPositiveDebted
	} else {
		debtor.Balance += originalPositiveDebted
	}
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	return tx.Commit()
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// UpdateDebtV2 — debt'ni yangilaydi: eski company balans + debtor ta'sirini bekor qiladi,
// debt qatorini yangilaydi va yangi ta'sirni qayta qo'llaydi. debt.UserID = amalni bajargan hodim.
func (s *CompanyOpsService) UpdateDebtV2(ctx context.Context, debt *store.Debts) error {
	if len(debt.ReceivedIncomes) == 0 {
		return fmt.Errorf("received incomes cannot be empty")
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtsStorage := store.NewDebtsStorage(tx)
	debtorsStorage := store.NewDebtorsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	oldDebt, err := debtsStorage.GetByID(ctx, debt.ID)
	if err != nil {
		return fmt.Errorf("failed to get old debt: %w", err)
	}

	debtor, err := debtorsStorage.GetById(ctx, oldDebt.DebtorID)
	if err != nil {
		return fmt.Errorf("failed to get debtor: %w", err)
	}

	companyID := debtor.CompanyID

	// Eski company balans ta'sirini bekor qilamiz
	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "debt_id", oldDebt.ID); err != nil {
		return err
	}

	// Eski debtor ta'sirini bekor qilamiz
	oldPositive := absInt64(oldDebt.DebtedAmount)
	if oldDebt.Type == types.TYPE_SELL {
		debtor.Balance += oldPositive
	} else {
		debtor.Balance -= oldPositive
	}

	// Yangi qiymatlarni qo'llaymiz
	originalPositiveDebted := debt.DebtedAmount
	switch debt.Type {
	case types.TYPE_SELL:
		debt.DebtedAmount = -debt.DebtedAmount
	case types.TYPE_BUY:
		// stays positive
	default:
		return fmt.Errorf("invalid debt type: %d", debt.Type)
	}
	debt.DebtorID = debtor.ID
	debt.CompanyID = companyID

	if err := debtsStorage.Update(ctx, debt); err != nil {
		return fmt.Errorf("failed to update debt: %w", err)
	}

	link := opLink{DebtId: &debt.ID}
	for _, tr := range debt.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, debt.UserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, int64(debt.Type), debt.Details, link); err != nil {
			return err
		}
	}

	if debt.Type == types.TYPE_SELL {
		debtor.Balance -= originalPositiveDebted
	} else {
		debtor.Balance += originalPositiveDebted
	}
	debtor.Currency = debt.DebtedCurrency
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	return tx.Commit()
}

// DeleteDebtV2 — debt'ni o'chiradi va company balans + debtor ta'sirini bekor qiladi.
func (s *CompanyOpsService) DeleteDebtV2(ctx context.Context, debtId int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtsStorage := store.NewDebtsStorage(tx)
	debtorsStorage := store.NewDebtorsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	debt, err := debtsStorage.GetByID(ctx, debtId)
	if err != nil {
		return fmt.Errorf("failed to get debt: %w", err)
	}

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "debt_id", debtId); err != nil {
		return err
	}

	debtor, err := debtorsStorage.GetById(ctx, debt.DebtorID)
	if err != nil {
		return fmt.Errorf("failed to get debtor: %w", err)
	}

	originalPositive := absInt64(debt.DebtedAmount)
	if debt.Type == types.TYPE_SELL {
		debtor.Balance += originalPositive
	} else {
		debtor.Balance -= originalPositive
	}
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	if err := debtsStorage.Delete(ctx, debtId); err != nil {
		return fmt.Errorf("failed to delete debt: %w", err)
	}

	return tx.Commit()
}

// DebtTransactionV2 — mavjud debtorga qarz tranzaksiyasi; kompaniya balansiga ta'sir qiladi.
func (s *CompanyOpsService) DebtTransactionV2(ctx context.Context, debt *store.Debts) error {
	if len(debt.ReceivedIncomes) == 0 {
		return fmt.Errorf("received incomes cannot be empty")
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtsStorage := store.NewDebtsStorage(tx)
	debtorsStorage := store.NewDebtorsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	debtor, err := debtorsStorage.GetById(ctx, debt.DebtorID)
	if err != nil {
		return fmt.Errorf("failed to get debtor: %w", err)
	}
	if debtor.Currency != debt.DebtedCurrency {
		return fmt.Errorf("debted currencies do not match: %s != %s", debtor.Currency, debt.DebtedCurrency)
	}

	companyID := debtor.CompanyID

	originalPositiveDebted := debt.DebtedAmount
	switch debt.Type {
	case types.TYPE_SELL:
		debt.DebtedAmount = -debt.DebtedAmount
	case types.TYPE_BUY:
		// stays positive
	default:
		return fmt.Errorf("invalid debt type: %d", debt.Type)
	}
	debt.CompanyID = companyID

	if err := debtsStorage.Create(ctx, debt); err != nil {
		return fmt.Errorf("failed to create debt: %w", err)
	}

	link := opLink{DebtId: &debt.ID}
	for _, tr := range debt.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, debt.UserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, int64(debt.Type), debt.Details, link); err != nil {
			return err
		}
	}

	if debt.Type == types.TYPE_SELL {
		debtor.Balance -= originalPositiveDebted
	} else {
		debtor.Balance += originalPositiveDebted
	}
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	return tx.Commit()
}
