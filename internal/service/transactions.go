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

const (
	TRANSACTION_STATUS_PENDING   = 1
	TRANSACTION_STATUS_COMPLETED = 2
	TYPE_SELL                    = 1
	TYPE_BUY                     = 2
)

type TransactionService struct {
	store  store.Storage
	notify notify.DeliveredUser
}

func NewTransactionService(store store.Storage, delivered notify.DeliveredUser) *TransactionService {
	if delivered == nil {
		delivered = notify.NoopDeliveredUser{}
	}
	return &TransactionService{store: store, notify: delivered}
}

func (s *TransactionService) PerformTransaction(ctx context.Context, transaction *store.Transaction) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)
	transactionsStorage := store.NewTransactionStorage(tx)
	if err := transactionsStorage.Create(ctx, transaction); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE Transactions.Create %v", err)
	}

	for _, tr := range transaction.ReceivedIncomes {
		balance, err := balancesStorage.GetByUserIdAndCurrency(ctx, &transaction.ReceivedUserId, tr.ReceivedCurrency)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return fmt.Errorf("user %d does not have a balance for currency %s", transaction.ReceivedUserId, tr.ReceivedCurrency)
			} else {
				return fmt.Errorf("ERROR OCCURRED WHILE balancesStorage.GetByUserIdAndCurrency %v", err)
			}
		}

		switch transaction.Type {
		case TYPE_SELL:
			if balance.Balance >= tr.ReceivedAmount {
				balance.Balance -= tr.ReceivedAmount
				balance.InOutLay += tr.ReceivedAmount
			} else {
				tx.Rollback()
				return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
			}
		case TYPE_BUY:
			balance.Balance += tr.ReceivedAmount
			balance.OutInLay += tr.ReceivedAmount
		default:
			tx.Rollback()
			return fmt.Errorf("FOUND UNKNOWN TYPE")
		}

		balanceRecord := &store.BalanceRecord{
			Amount:        tr.ReceivedAmount,
			Currency:      tr.ReceivedCurrency,
			BalanceID:     balance.ID,
			CompanyID:     balance.CompanyId,
			UserID:        transaction.ReceivedUserId,
			Type:          transaction.Type,
			TransactionId: &transaction.ID,
		}

		if err := balanceRecordsStorage.Create(ctx, balanceRecord); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE BalanceRecords.Create %v", err)
		}

		if err := balancesStorage.Update(ctx, balance); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE balancesStorage.Update %v", err)
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

func (s *TransactionService) CompleteTransaction(ctx context.Context, transaction types.TransactionComplete) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)
	transactionsStorage := store.NewTransactionStorage(tx)

	tran, err := transactionsStorage.GetById(ctx, transaction.TransactionID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE transactionsStorage.GetById %v", err)
	}

	for _, tr := range tran.DeliveredOutcomes {
		balance, err := balancesStorage.GetByUserIdAndCurrency(ctx, &transaction.DeliveredUserId, tr.DeliveredCurrency)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return fmt.Errorf("user %d does not have a balance for currency %s", transaction.DeliveredUserId, tr.DeliveredCurrency)
			}
			return fmt.Errorf("ERROR OCCURRED WHILE balancesStorage.GetByUserIdAndCurrency %v", err)
		}

		var recordType int64
		if tran.Type == TYPE_SELL {
			recordType = TYPE_BUY
			balance.Balance += tr.DeliveredAmount
			balance.OutInLay += tr.DeliveredAmount
		} else {
			recordType = TYPE_SELL
			if balance.Balance >= tr.DeliveredAmount {
				balance.Balance -= tr.DeliveredAmount
				balance.InOutLay += tr.DeliveredAmount
			} else {
				tx.Rollback()
				return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
			}
		}

		balanceRecord := &store.BalanceRecord{
			Amount:        tr.DeliveredAmount,
			Currency:      tr.DeliveredCurrency,
			BalanceID:     balance.ID,
			UserID:        transaction.DeliveredUserId,
			TransactionId: &tran.ID,
			CompanyID:     balance.CompanyId,
			Type:          recordType,
		}

		if err := balanceRecordsStorage.Create(ctx, balanceRecord); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE balanceRecordsStorage.Create %v", err)
		}

		if err := balancesStorage.Update(ctx, balance); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE balancesStorage.Update %v", err)
		}
	}

	tran.Status = TRANSACTION_STATUS_COMPLETED
	tran.DeliveredUserId = &transaction.DeliveredUserId

	// Agar tranzaksiya yaratilganda xizmat puli kiritilmagan bo'lsa,
	// yakunlash bosqichida kiritilgan summa + valyuta + izoh saqlanadi.
	if transaction.ServiceFeeAmount > 0 {
		tran.ServiceFeeAmount = transaction.ServiceFeeAmount
		tran.ServiceFeeCurrency = transaction.ServiceFeeCurrency
		tran.ServiceFeeDetails = transaction.ServiceFeeDetails
	}

	if err := transactionsStorage.Update(ctx, tran); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE transactionsStorage.Update %v", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	deliveredID := transaction.DeliveredUserId
	tid := tran.ID
	details := tran.Details
	go func() {
		ctxN, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		s.notify.NotifyDeliveryCompleted(ctxN, deliveredID, tid, details)
	}()
	return nil
}

func (s *TransactionService) Update(ctx context.Context, transaction *store.Transaction) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)
	transactionsStorage := store.NewTransactionStorage(tx)

	records, err := balanceRecordsStorage.GetByField(ctx, "transaction_id", transaction.ID, types.Pagination{Limit: 1000, Offset: 0})
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, record := range records {
		balance, err := balancesStorage.GetById(ctx, &record.BalanceID)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return fmt.Errorf("balance %d not found", record.BalanceID)
			}
			return err
		}

		// Revert without sufficiency checks, as we're restoring prior state
		if record.Type == TYPE_SELL {
			balance.Balance += record.Amount
			balance.InOutLay -= record.Amount
		} else { // TYPE_BUY
			balance.Balance -= record.Amount
			balance.OutInLay -= record.Amount
		}

		if err := balanceRecordsStorage.Delete(ctx, record.ID); err != nil {
			tx.Rollback()
			return err
		}

		if err := balancesStorage.Update(ctx, balance); err != nil {
			tx.Rollback()
			return err
		}
	}

	if transaction.ReceivedUserId != 0 {
		for _, tr := range transaction.ReceivedIncomes {
			balance, err := balancesStorage.GetByUserIdAndCurrency(ctx, &transaction.ReceivedUserId, tr.ReceivedCurrency)
			if err != nil {
				tx.Rollback()
				if err == sql.ErrNoRows {
					return fmt.Errorf("user %d does not have a balance for currency %s", transaction.ReceivedUserId, tr.ReceivedCurrency)
				}
				return err
			}

			record := &store.BalanceRecord{
				Amount:        tr.ReceivedAmount,
				Currency:      tr.ReceivedCurrency,
				UserID:        transaction.ReceivedUserId,
				CompanyID:     balance.CompanyId,
				TransactionId: &transaction.ID,
				BalanceID:     balance.ID,
				Details:       transaction.Details,
				Type:          transaction.Type,
			}

			switch transaction.Type {
			case TYPE_SELL:
				if balance.Balance < tr.ReceivedAmount {
					tx.Rollback()
					return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
				}
				balance.Balance -= tr.ReceivedAmount
				balance.InOutLay += tr.ReceivedAmount
			case TYPE_BUY:
				balance.Balance += tr.ReceivedAmount
				balance.OutInLay += tr.ReceivedAmount
			default:
				tx.Rollback()
				return fmt.Errorf("FOUND UNKNOWN TYPE")
			}

			if err := balancesStorage.Update(ctx, balance); err != nil {
				tx.Rollback()
				return err
			}

			if err := balanceRecordsStorage.Create(ctx, record); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if transaction.DeliveredUserId != nil {
		for _, tr := range transaction.DeliveredOutcomes {
			balance, err := balancesStorage.GetByUserIdAndCurrency(ctx, transaction.DeliveredUserId, tr.DeliveredCurrency)
			if err != nil {
				tx.Rollback()
				if err == sql.ErrNoRows {
					return fmt.Errorf("user %d does not have a balance for currency %s", *transaction.DeliveredUserId, tr.DeliveredCurrency)
				}
				return err
			}

			var recordType int64
			if transaction.Type == TYPE_SELL {
				recordType = TYPE_BUY
				balance.Balance += tr.DeliveredAmount
				balance.OutInLay += tr.DeliveredAmount
			} else {
				recordType = TYPE_SELL
				if balance.Balance < tr.DeliveredAmount {
					tx.Rollback()
					return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
				}
				balance.Balance -= tr.DeliveredAmount
				balance.InOutLay += tr.DeliveredAmount
			}

			record := &store.BalanceRecord{
				Amount:        tr.DeliveredAmount,
				Currency:      tr.DeliveredCurrency,
				UserID:        *transaction.DeliveredUserId,
				CompanyID:     balance.CompanyId,
				TransactionId: &transaction.ID,
				BalanceID:     balance.ID,
				Details:       transaction.Details,
				Type:          recordType,
			}

			if err := balancesStorage.Update(ctx, balance); err != nil {
				tx.Rollback()
				return err
			}

			if err := balanceRecordsStorage.Create(ctx, record); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	if err := transactionsStorage.Update(ctx, transaction); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING TRANSACTION %v", err)
	}

	tx.Commit()
	return nil
}

func (s *TransactionService) Delete(ctx context.Context, id *int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)
	transactionsStorage := store.NewTransactionStorage(tx)

	tran, err := transactionsStorage.GetById(ctx, *id)
	if err != nil {
		tx.Rollback()
		return err
	}

	records, err := balanceRecordsStorage.GetByField(ctx, "transaction_id", tran.ID, types.Pagination{Limit: 1000, Offset: 0})
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, record := range records {
		balance, err := balancesStorage.GetById(ctx, &record.BalanceID)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return fmt.Errorf("balance %d not found", record.BalanceID)
			}
			return err
		}

		// Revert without sufficiency checks, as we're restoring prior state
		if record.Type == TYPE_SELL {
			balance.Balance += record.Amount
			balance.InOutLay -= record.Amount
		} else { // TYPE_BUY
			balance.Balance -= record.Amount
			balance.OutInLay -= record.Amount
		}
		if err := balancesStorage.Update(ctx, balance); err != nil {
			tx.Rollback()
			return err
		}
		if err := balanceRecordsStorage.Delete(ctx, record.ID); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := transactionsStorage.Delete(ctx, &tran.ID); err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (s *TransactionService) GetByCompanyId(ctx context.Context, companyId int64, pagination types.Pagination) ([]map[string]interface{}, error) {
	trans, err := s.store.Transactions.GetByField(ctx, nil, "delivered_company_id", companyId, pagination)
	if err != nil {
		return nil, err
	}

	companies, err := s.store.Companies.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.store.Users.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}
	getCurrencies := make(map[string]int64)
	giveCurrencies := make(map[string]int64)

	for _, tran := range trans {
		if tran.Status == TRANSACTION_STATUS_PENDING {
			if tran.Type == TYPE_SELL {
				for _, tr := range tran.DeliveredOutcomes {
					getCurrencies[tr.DeliveredCurrency] += tr.DeliveredAmount
				}
			} else {
				for _, tr := range tran.DeliveredOutcomes {
					giveCurrencies[tr.DeliveredCurrency] += tr.DeliveredAmount
				}
			}

			receivedCompany := GetCompany(companies, tran.ReceivedCompanyId)
			receivedUser := GetUser(users, tran.ReceivedUserId)
			deliveryUser := ""
			if tran.DeliveredUserId != nil {
				deliveryUserUser := GetUser(users, *tran.DeliveredUserId)
				if deliveryUserUser != nil {
					deliveryUser = deliveryUserUser.Username
				}
			}

			res := map[string]interface{}{
				"service_fee":          formatServiceFee(tran.ServiceFeeAmount, tran.ServiceFeeCurrency, tran.ServiceFeeDetails),
				"service_fee_amount":   tran.ServiceFeeAmount,
				"service_fee_currency": tran.ServiceFeeCurrency,
				"service_fee_details":  tran.ServiceFeeDetails,
				"received_incomes":     tran.ReceivedIncomes,
				"delivered_outcomes":   tran.DeliveredOutcomes,
				"received_company":     "",
				"received_user":        "",
				"delivered_user":       deliveryUser,
				"phone":                tran.Phone,
				"details":              tran.Details,
				"created_at":           tran.CreatedAt,
				"type":                 tran.Type,
				"status":               tran.Status,
			}
			if receivedCompany != nil {
				res["received_company"] = receivedCompany.Name
			}
			if receivedUser != nil {
				res["received_user"] = receivedUser.Username
			}

			response = append(response, res)
		}
	}
	response = append(response, map[string]interface{}{
		"get_currencies": getCurrencies,
		"giveCurrencies": giveCurrencies,
	})

	return response, nil
}

func (s *TransactionService) GetByField(ctx context.Context, search *string, fieldName string, value any, pagination types.Pagination) ([]map[string]interface{}, error) {
	trans, err := s.store.Transactions.GetByField(ctx, search, fieldName, value, pagination)
	if err != nil {
		return nil, err
	}

	companies, err := s.store.Companies.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.store.Users.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}
	for _, tran := range trans {
		receiverUser := GetUser(users, tran.ReceivedUserId)
		receiverUserName := ""
		if receiverUser != nil {
			receiverUserName = receiverUser.Username
		}
		deliveryUser := ""
		if tran.DeliveredUserId != nil {
			deliveryUserUser := GetUser(users, *tran.DeliveredUserId)
			if deliveryUserUser != nil {
				deliveryUser = deliveryUserUser.Username
			}
		}

		receivedCompany := GetCompany(companies, tran.ReceivedCompanyId)
		receivedCompanyName := ""
		if receivedCompany != nil {
			receivedCompanyName = receivedCompany.Name
		}
		deliveredCompany := GetCompany(companies, tran.DeliveredCompanyId)
		deliveredCompanyName := ""
		if deliveredCompany != nil {
			deliveredCompanyName = deliveredCompany.Name
		}

		res := map[string]interface{}{
			"id":                   tran.ID,
			"number":               tran.Number,
			"received_company_id":  tran.ReceivedCompanyId,
			"received_company":     receivedCompanyName,
			"received_user_id":     tran.ReceivedUserId,
			"received_user":        receiverUserName,
			"received_incomes":     tran.ReceivedIncomes,
			"delivered_outcomes":   tran.DeliveredOutcomes,
			"delivered_company_id": tran.DeliveredCompanyId,
			"delivered_company":    deliveredCompanyName,
			"delivered_user":       deliveryUser,
			"delivered_user_id":    tran.DeliveredUserId,
			"service_fee":          formatServiceFee(tran.ServiceFeeAmount, tran.ServiceFeeCurrency, tran.ServiceFeeDetails),
			"service_fee_amount":   tran.ServiceFeeAmount,
			"service_fee_currency": tran.ServiceFeeCurrency,
			"service_fee_details":  tran.ServiceFeeDetails,
			"phone":                tran.Phone,
			"details":              tran.Details,
			"created_at":           tran.CreatedAt,
			"type":                 tran.Type,
			"status":               tran.Status,
		}

		response = append(response, res)
	}

	return response, nil
}

func (s *TransactionService) Archived(ctx context.Context, pagination types.Pagination) ([]map[string]interface{}, error) {
	trans, err := s.store.Transactions.Archived(ctx, pagination)
	if err != nil {
		return nil, err
	}

	companies, err := s.store.Companies.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.store.Users.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}
	for _, tran := range trans {
		receivedUser := GetUser(users, tran.ReceivedUserId)
		receivedUserName := ""
		if receivedUser != nil {
			receivedUserName = receivedUser.Username
		}
		DeliveredUser := ""
		if tran.DeliveredUserId != nil {
			deliveryUserUser := GetUser(users, *tran.DeliveredUserId)
			if deliveryUserUser != nil {
				DeliveredUser = deliveryUserUser.Username
			}
		}

		receivedCompany := GetCompany(companies, tran.ReceivedCompanyId)
		receivedCompanyName := ""
		if receivedCompany != nil {
			receivedCompanyName = receivedCompany.Name
		}
		deliveredCompany := GetCompany(companies, tran.DeliveredCompanyId)
		deliveredCompanyName := ""
		if deliveredCompany != nil {
			deliveredCompanyName = deliveredCompany.Name
		}

		res := map[string]interface{}{
			"id":                   tran.ID,
			"received_company_id":  tran.ReceivedCompanyId,
			"received_company":     receivedCompanyName,
			"received_user_id":     tran.ReceivedUserId,
			"received_user":        receivedUserName,
			"received_incomes":     tran.ReceivedIncomes,
			"delivered_outcomes":   tran.DeliveredOutcomes,
			"delivered_company_id": tran.DeliveredCompanyId,
			"delivered_company":    deliveredCompanyName,
			"delivered_user":       DeliveredUser,
			"delivered_user_id":    tran.DeliveredUserId,
			"service_fee":          formatServiceFee(tran.ServiceFeeAmount, tran.ServiceFeeCurrency, tran.ServiceFeeDetails),
			"service_fee_amount":   tran.ServiceFeeAmount,
			"service_fee_currency": tran.ServiceFeeCurrency,
			"service_fee_details":  tran.ServiceFeeDetails,
			"phone":                tran.Phone,
			"details":              tran.Details,
			"created_at":           tran.CreatedAt,
			"type":                 tran.Type,
			"status":               tran.Status,
		}

		response = append(response, res)
	}
	return response, nil
}

func (s *TransactionService) GetInfos(ctx context.Context, date string) ([]store.CompanyAmount, error) {
	trans, err := s.store.Transactions.GetCompanyFinalAmounts(ctx, []int64{1, 2}, date)
	if err != nil {
		return nil, fmt.Errorf("ERROR OCCURRED WHILE Transactions.GetByField %v", err)
	}

	return trans, nil
}

// formatServiceFee — eski mobil versiyalar bilan moslik uchun xizmat puli matn ko'rinishini qaytaradi.
func formatServiceFee(amount int64, currency, details string) string {
	if amount > 0 {
		if currency != "" {
			return fmt.Sprintf("%d %s", amount, currency)
		}
		return fmt.Sprintf("%d", amount)
	}
	return details
}

func GetOne(ids map[string]int64, id string) int64 {
	return ids[id]
}

func GetCompany(companies []store.Company, companyId int64) *store.Company {
	for _, company := range companies {
		if company.ID == companyId {
			return &company
		}
	}
	return nil
}
