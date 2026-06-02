package store

import (
	"context"
	"database/sql"

	"github.com/mubashshir3767/currencyExchange/internal/types"
)

const (
	STATUS_CREATED   = 1
	STATUS_COMPLETED = 2
	STATUS_ARCHIVED  = 3
)

type DBTX interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Commit() error
	Rollback() error
}

type Storage struct {
	DB *sql.DB

	Exchanges interface {
		Create(context.Context, *Exchange) error
		Update(context.Context, *Exchange) error
		GetById(context.Context, int64) (*Exchange, error)
		GetByField(context.Context, string, any, types.Pagination) ([]Exchange, error)
		Delete(context.Context, int64) error
		Archive(context.Context, int64) error
		Archived(context.Context, types.Pagination) ([]Exchange, error)
	}

	Debtors interface {
		Create(context.Context, *Debtors) error
		Update(context.Context, *Debtors) error
		GetById(context.Context, int64) (*Debtors, error)
		GetByUserId(context.Context, int64, types.Pagination) ([]Debtors, error)
		GetByCompanyId(context.Context, int64, *string, *string, types.Pagination) ([]Debtors, error)
		GetByBalanceInfo(context.Context, int64) ([]map[string]interface{}, error)
		Delete(context.Context, int64) error
	}

	Debts interface {
		Create(context.Context, *Debts) error
		Update(context.Context, *Debts) error
		GetByID(context.Context, int64) (*Debts, error)
		GetByUserID(context.Context, int64, types.Pagination) ([]Debts, error)
		GetByDebtorID(context.Context, int64, types.Pagination) ([]Debts, error)
		Delete(context.Context, int64) error
	}

	Users interface {
		Login(context.Context, *User) error
		Create(context.Context, *User) error
		Update(context.Context, *User) error
		GetAll(context.Context) ([]User, error)
		GetById(context.Context, *int64) (*User, error)
		Delete(context.Context, *int64) error
	}

	Balances interface {
		Create(context.Context, *Balance) error
		GetById(context.Context, *int64) (*Balance, error)
		GetByUserIdAndCurrency(context.Context, *int64, string) (*Balance, error)
		GetByUserId(context.Context, *int64) ([]Balance, error)
		GetByCompanyId(context.Context, *int64) ([]Balance, error)
		GetAll(context.Context) ([]Balance, error)
		Update(context.Context, *Balance) error
		Delete(context.Context, int64) error
	}

	CompanyBalances interface {
		Create(context.Context, *CompanyBalance) error
		GetByCompanyIdAndCurrency(context.Context, int64, string) (*CompanyBalance, error)
		GetByCompanyId(context.Context, int64) ([]CompanyBalance, error)
		AggregateByCompanyId(context.Context, int64) ([]CompanyBalance, error)
		ListRecordsByCompanyAndCurrency(context.Context, int64, string, types.Pagination) ([]CompanyBalanceRecordRow, error)
		Update(context.Context, *CompanyBalance) error
		EnsureDefaults(context.Context, int64, []string) error
		UserActivityByCompany(context.Context, int64) ([]UserActivityRow, error)
	}

	CompanyBalanceRecords interface {
		Create(context.Context, *CompanyBalanceRecord) error
		GetById(context.Context, int64) (*CompanyBalanceRecord, error)
		Update(context.Context, *CompanyBalanceRecord) error
		Delete(context.Context, int64) error
		ListByLink(context.Context, string, int64) ([]CompanyBalanceRecord, error)
		ListByCompany(context.Context, int64, string, types.Pagination) ([]CompanyBalanceRecordRow, error)
	}

	BalanceRecords interface {
		Create(context.Context, *BalanceRecord) error
		GetByField(context.Context, string, any, types.Pagination) ([]BalanceRecord, error)
		GetByFieldAndDate(context.Context, string, *string, *string, any, types.Pagination) ([]BalanceRecord, error)
		Update(context.Context, *BalanceRecord) error
		Delete(context.Context, int64) error
		Archive(context.Context, int64) error
		Archived(context.Context, types.Pagination) ([]BalanceRecord, error)
	}

	Transactions interface {
		Create(context.Context, *Transaction) error
		Update(context.Context, *Transaction) error
		Delete(context.Context, *int64) error
		GetByField(context.Context, *string, string, any, types.Pagination) ([]Transaction, error)
		GetInfos(ctx context.Context, companyId int64) ([]Transaction, error)
		GetCompanyFinalAmounts(ctx context.Context, companyIDs []int64, date string) ([]CompanyAmount, error)
		GetByFieldAndDate(context.Context, string, string, string, any, types.Pagination) ([]Transaction, error)
		Archive(context.Context, int64) error
		Archived(context.Context, types.Pagination) ([]Transaction, error)
	}

	Companies interface {
		Create(context.Context, *Company) error
		GetAll(context.Context) ([]Company, error)
		GetById(context.Context, *int64) (*Company, error)
		Update(context.Context, *Company) error
		Delete(context.Context, *int64) error
	}

	UserSessions interface {
		Upsert(context.Context, *UserSession) error
		ListByUserID(context.Context, int64) ([]UserSession, error)
		GetByIDForUser(context.Context, int64, int64) (*UserSession, error)
		UpdateFCM(context.Context, int64, int64, string, *string) error
		Delete(context.Context, int64, int64) error
		FCMTokensByUserID(context.Context, int64) ([]string, error)
		DeleteByFCMToken(context.Context, string) error
	}
}

func NewStorage(db *sql.DB) Storage {
	dbwrapper := &DBWrapper{db: db}

	return Storage{
		DB:                    db,
		Debts:                 &DebtsStorage{db: dbwrapper},
		Exchanges:             &ExchangeStorage{db: dbwrapper},
		Debtors:               &DebtorsStorage{db: dbwrapper},
		Users:                 &UserStorage{db: dbwrapper},
		Transactions:          &TransactionStorage{db: dbwrapper},
		Balances:              &BalanceStorage{db: dbwrapper},
		CompanyBalances:       &CompanyBalanceStorage{db: dbwrapper},
		CompanyBalanceRecords: &CompanyBalanceRecordStorage{db: dbwrapper},
		Companies:             &CompanyStorage{db: dbwrapper},
		BalanceRecords:        &BalanceRecordStorage{db: dbwrapper},
		UserSessions:          &UserSessionStorage{db: dbwrapper},
	}
}

func (s *Storage) BeginTx(ctx context.Context) (DBTX, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &TxWrapper{tx: tx}, nil
}
