package service

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/mubashshir3767/currencyExchange/internal/notify"
	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type Service struct {
	Exchanges interface {
		Create(context.Context, *store.Exchange) error
		Update(context.Context, *store.Exchange) error
		Delete(context.Context, int64) error
	}

	Balances interface {
		GetByCompanyId(context.Context, int64) ([]map[string]interface{}, error)
		GetAll(context.Context) ([]map[string]interface{}, error)
	}

	Debts interface {
		Create(context.Context, *store.Debts) error
		Transaction(context.Context, *store.Debts) error
		Update(context.Context, *store.Debts) error
		Delete(context.Context, int64) error
	}

	Debtors interface {
		GetByCompanyId(context.Context, int64, *string, *string, types.Pagination) ([]map[string]interface{}, error)
	}

	BalanceRecords interface {
		PerformBalanceRecord(context.Context, types.BalanceRecordPayload) error
		RollbackBalanceRecord(context.Context, int64) error
		UpdateRecord(context.Context, store.BalanceRecord) error
	}

	CompanyBalanceRecords interface {
		PerformCompanyBalanceRecord(context.Context, types.CompanyBalanceRecordPayload) error
		RollbackCompanyBalanceRecord(context.Context, int64) error
		UpdateCompanyBalanceRecord(context.Context, store.CompanyBalanceRecord) error
	}

	SoftBalanceRecords interface {
		PerformSoftBalanceRecord(context.Context, types.CompanyBalanceRecordPayload) error
		RollbackSoftBalanceRecord(context.Context, int64) error
	}

	// CompanyOps — v2: exchange/transaction/debt kompaniya balansiga ta'sir qiladi.
	CompanyOps interface {
		CreateExchangeV2(context.Context, *store.Exchange) error
		UpdateExchangeV2(context.Context, *store.Exchange) error
		DeleteExchangeV2(context.Context, int64) error
		PerformTransactionV2(context.Context, *store.Transaction, int64) error
		CompleteTransactionV2(context.Context, types.TransactionComplete, int64) error
		UpdateTransactionV2(context.Context, *store.Transaction, int64) error
		DeleteTransactionV2(context.Context, int64) error
		CreateDebtV2(context.Context, *store.Debts) error
		DebtTransactionV2(context.Context, *store.Debts) error
		UpdateDebtV2(context.Context, *store.Debts) error
		DeleteDebtV2(context.Context, int64) error
	}

	Transactions interface {
		GetByField(context.Context, *string, string, any, types.Pagination) ([]map[string]interface{}, error)
		PerformTransaction(context.Context, *store.Transaction) error
		CompleteTransaction(context.Context, types.TransactionComplete) error
		GetByCompanyId(context.Context, int64, types.Pagination) ([]map[string]interface{}, error)
		GetInfos(ctx context.Context, date string) ([]store.CompanyAmount, error)
		Archived(context.Context, types.Pagination) ([]map[string]interface{}, error)
		Update(context.Context, *store.Transaction) error
		Delete(context.Context, *int64) error
	}
}

func NewService(store store.Storage, delivered notify.DeliveredUser) Service {
	return Service{
		Debtors:               &DebtorsService{store: store},
		Balances:              &BalanceService{store: store},
		Exchanges:             &ExchangeService{store: store},
		BalanceRecords:        &BalanceRecordService{store: store},
		CompanyBalanceRecords: &CompanyBalanceRecordService{store: store},
		SoftBalanceRecords:    &SoftBalanceRecordService{store: store},
		CompanyOps:            NewCompanyOpsService(store, delivered),
		Transactions:          NewTransactionService(store, delivered),
		Debts:                 &DebtsService{store: store},
	}
}

func GenerateSerialNo(id int64) string {
	return fmt.Sprintf("%v%v", id, rand.Intn(10000000))
}
