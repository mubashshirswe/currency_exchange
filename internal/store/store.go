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
	// STATUS_ACCEPTED — 3 bosqichli oqimdagi oraliq holat: buyurtma qabul qilingan,
	// lekin hali topshirilmagan. Balansga ta'sir qilmaydi.
	STATUS_ACCEPTED = 4
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
		GetByField(context.Context, int64, string, any, types.Pagination) ([]Exchange, error)
		Delete(context.Context, int64) error
		Archive(context.Context, int64) error
		Archived(context.Context, int64, types.Pagination) ([]Exchange, error)
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

	Businesses interface {
		Create(context.Context, *Business) error
		GetById(context.Context, int64) (*Business, error)
		Update(context.Context, *Business) error
		SetStatus(context.Context, int64, int64) error
		Delete(context.Context, int64) error
	}

	BusinessSettings interface {
		GetByBusinessID(context.Context, int64) (*BusinessSettings, error)
		Upsert(context.Context, *BusinessSettings) error
	}

	Users interface {
		LoginCandidates(context.Context, string, string) ([]User, error)
		Create(context.Context, *User) error
		Update(context.Context, *User) error
		ListByBusiness(context.Context, int64) ([]User, error)
		ListByCompany(context.Context, int64, int64) ([]User, error)
		GetById(context.Context, *int64) (*User, error)
		GetByIdInBusiness(context.Context, int64, int64) (*User, error)
		Delete(context.Context, *int64, int64) error
	}

	Balances interface {
		Create(context.Context, *Balance) error
		GetById(context.Context, *int64) (*Balance, error)
		GetByUserIdAndCurrency(context.Context, *int64, string) (*Balance, error)
		GetByUserId(context.Context, *int64) ([]Balance, error)
		GetByCompanyId(context.Context, *int64) ([]Balance, error)
		GetAll(context.Context, int64) ([]Balance, error)
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

	SoftBalances interface {
		Create(context.Context, *SoftBalance) error
		GetByCompanyIdAndCurrency(context.Context, int64, string) (*SoftBalance, error)
		GetByCompanyId(context.Context, int64) ([]SoftBalance, error)
		Update(context.Context, *SoftBalance) error
		EnsureDefaults(context.Context, int64, []string) error
	}

	SoftBalanceRecords interface {
		Create(context.Context, *SoftBalanceRecord) error
		GetById(context.Context, int64) (*SoftBalanceRecord, error)
		Delete(context.Context, int64) error
		ListByLink(context.Context, string, int64) ([]SoftBalanceRecord, error)
		ListByCompany(context.Context, int64, string, types.Pagination) ([]SoftBalanceRecordRow, error)
	}

	BalanceRecords interface {
		Create(context.Context, *BalanceRecord) error
		GetByField(context.Context, int64, string, any, types.Pagination) ([]BalanceRecord, error)
		GetByFieldAndDate(context.Context, int64, string, *string, *string, any, types.Pagination) ([]BalanceRecord, error)
		Update(context.Context, *BalanceRecord) error
		Delete(context.Context, int64) error
		Archive(context.Context, int64) error
		Archived(context.Context, int64, types.Pagination) ([]BalanceRecord, error)
	}

	Transactions interface {
		Create(context.Context, *Transaction) error
		Update(context.Context, *Transaction) error
		SetAccepted(ctx context.Context, id, userID, companyID int64) error
		GetById(context.Context, int64) (*Transaction, error)
		Delete(context.Context, *int64) error
		GetByField(context.Context, int64, *string, string, any, types.Pagination) ([]Transaction, error)
		GetInfos(ctx context.Context, companyId int64) ([]Transaction, error)
		GetCompanyFinalAmounts(ctx context.Context, companyIDs []int64, date string) ([]CompanyAmount, error)
		GetByFieldAndDate(context.Context, int64, string, string, string, any, types.Pagination) ([]Transaction, error)
		Archive(context.Context, int64) error
		Archived(context.Context, int64, types.Pagination) ([]Transaction, error)
	}

	TransactionServiceFees interface {
		Create(context.Context, *TransactionServiceFee) error
		GetByTransactionID(context.Context, int64) (*TransactionServiceFee, error)
		Update(context.Context, *TransactionServiceFee) error
		DeleteByTransactionID(context.Context, int64) error
		ListPendingFIFO(context.Context, int64, string) ([]TransactionServiceFee, error)
		ListAllPending(context.Context, int64, string) ([]TransactionServiceFee, error)
		ListByCompany(context.Context, int64, string, int64, types.Pagination) ([]TransactionServiceFee, error)
		ListAll(context.Context, int64, string, int64, types.Pagination) ([]TransactionServiceFee, error)
		GetRemainingByCompanies(context.Context, []int64) ([]ServiceFeeRemainingRow, error)
	}

	ServiceFeeSettlements interface {
		Create(context.Context, *ServiceFeeSettlement) error
		ListByCompany(context.Context, int64, string, types.Pagination) ([]ServiceFeeSettlement, error)
		ListAll(context.Context, int64, string, types.Pagination) ([]ServiceFeeSettlement, error)
	}

	ServiceFeeSettlementItems interface {
		Create(context.Context, int64, int64, int64) error
	}

	Companies interface {
		Create(context.Context, *Company) error
		ListByBusiness(context.Context, int64) ([]Company, error)
		GetById(context.Context, *int64) (*Company, error)
		GetByIdInBusiness(context.Context, int64, int64) (*Company, error)
		BelongsToBusiness(context.Context, int64, int64) (bool, error)
		Update(context.Context, *Company) error
		Delete(context.Context, *int64, int64) error
	}

	UserSessions interface {
		Upsert(context.Context, *UserSession) error
		ListByUserID(context.Context, int64) ([]UserSession, error)
		GetByIDForUser(context.Context, int64, int64) (*UserSession, error)
		UpdateFCM(context.Context, int64, int64, string, *string) error
		Delete(context.Context, int64, int64) error
		FCMTokensByUserID(context.Context, int64) ([]string, error)
		FCMTokensByCompanyID(context.Context, int64) ([]string, error)
		DeleteByFCMToken(context.Context, string) error
	}
}

func NewStorage(db *sql.DB) Storage {
	dbwrapper := &DBWrapper{db: db}

	return Storage{
		DB:                        db,
		Businesses:                &BusinessStorage{db: dbwrapper},
		BusinessSettings:          &BusinessSettingsStorage{db: dbwrapper},
		Debts:                     &DebtsStorage{db: dbwrapper},
		Exchanges:                 &ExchangeStorage{db: dbwrapper},
		Debtors:                   &DebtorsStorage{db: dbwrapper},
		Users:                     &UserStorage{db: dbwrapper},
		Transactions:              &TransactionStorage{db: dbwrapper},
		TransactionServiceFees:    &TransactionServiceFeeStorage{db: dbwrapper},
		ServiceFeeSettlements:     &ServiceFeeSettlementStorage{db: dbwrapper},
		ServiceFeeSettlementItems: &ServiceFeeSettlementItemStorage{db: dbwrapper},
		Balances:                  &BalanceStorage{db: dbwrapper},
		CompanyBalances:           &CompanyBalanceStorage{db: dbwrapper},
		CompanyBalanceRecords:     &CompanyBalanceRecordStorage{db: dbwrapper},
		SoftBalances:              &SoftBalanceStorage{db: dbwrapper},
		SoftBalanceRecords:        &SoftBalanceRecordStorage{db: dbwrapper},
		Companies:                 &CompanyStorage{db: dbwrapper},
		BalanceRecords:            &BalanceRecordStorage{db: dbwrapper},
		UserSessions:              &UserSessionStorage{db: dbwrapper},
	}
}

func (s *Storage) BeginTx(ctx context.Context) (DBTX, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &TxWrapper{tx: tx}, nil
}
