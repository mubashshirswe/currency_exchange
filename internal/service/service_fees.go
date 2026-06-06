package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type ServiceFeeService struct {
	store store.Storage
}

func NewServiceFeeService(s store.Storage) *ServiceFeeService {
	return &ServiceFeeService{store: s}
}

func normalizeFeeCurrency(c string) string {
	c = strings.TrimSpace(strings.ToUpper(c))
	if c == "" {
		return "SUM"
	}
	return c
}

type serviceFeeWriter interface {
	Create(context.Context, *store.TransactionServiceFee) error
	GetByTransactionID(context.Context, int64) (*store.TransactionServiceFee, error)
	Update(context.Context, *store.TransactionServiceFee) error
	DeleteByTransactionID(context.Context, int64) error
}

func (s *ServiceFeeService) syncFromTransaction(
	ctx context.Context,
	fees serviceFeeWriter,
	tran *store.Transaction,
	companyID int64,
) error {
	if tran.ServiceFeeAmount <= 0 {
		return fees.DeleteByTransactionID(ctx, tran.ID)
	}

	currency := normalizeFeeCurrency(tran.ServiceFeeCurrency)
	existing, err := fees.GetByTransactionID(ctx, tran.ID)
	if err == sql.ErrNoRows {
		f := &store.TransactionServiceFee{
			TransactionID:   tran.ID,
			CompanyID:       companyID,
			Amount:          tran.ServiceFeeAmount,
			RemainingAmount: tran.ServiceFeeAmount,
			Currency:        currency,
			Details:         tran.ServiceFeeDetails,
			Status:          store.ServiceFeeStatusPending,
		}
		return fees.Create(ctx, f)
	}
	if err != nil {
		return err
	}

	existing.CompanyID = companyID
	existing.Amount = tran.ServiceFeeAmount
	if existing.RemainingAmount > tran.ServiceFeeAmount {
		existing.RemainingAmount = tran.ServiceFeeAmount
	}
	existing.Currency = currency
	existing.Details = tran.ServiceFeeDetails
	if existing.RemainingAmount <= 0 {
		existing.Status = store.ServiceFeeStatusSettled
		existing.RemainingAmount = 0
	} else if existing.RemainingAmount < existing.Amount {
		existing.Status = store.ServiceFeeStatusPending
	} else {
		existing.RemainingAmount = tran.ServiceFeeAmount
		existing.Status = store.ServiceFeeStatusPending
	}
	return fees.Update(ctx, existing)
}

// SyncFromTransactionTx — tranzaksiya ichida xizmat haqini sinxronlaydi.
func (s *ServiceFeeService) SyncFromTransactionTx(
	ctx context.Context,
	tx store.DBTX,
	tran *store.Transaction,
	companyID int64,
) error {
	return s.syncFromTransaction(ctx, store.NewTransactionServiceFeeStorage(tx), tran, companyID)
}

// AttachRemainingToCompanyAmounts — info kartaga taqsimlanmagan summalarni qo'shadi.
func (s *ServiceFeeService) AttachRemainingToCompanyAmounts(
	ctx context.Context,
	amounts []store.CompanyAmount,
	companies []store.Company,
) error {
	if len(companies) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(companies))
	nameByID := map[int64]string{}
	for _, c := range companies {
		ids = append(ids, c.ID)
		nameByID[c.ID] = strings.ToUpper(strings.TrimSpace(c.Name))
	}

	rows, err := s.store.TransactionServiceFees.GetRemainingByCompanies(ctx, ids)
	if err != nil {
		return err
	}

	remainingMap := map[string]float64{}
	for _, r := range rows {
		name := nameByID[r.CompanyID]
		key := name + "|" + strings.ToUpper(r.Currency)
		remainingMap[key] += float64(r.Remaining)
	}

	for i := range amounts {
		key := amounts[i].CompanyName + "|" + strings.ToUpper(amounts[i].Currency)
		if v, ok := remainingMap[key]; ok {
			amounts[i].ServiceFeeRemaining = v
		}
	}
	return nil
}

func (s *ServiceFeeService) ListFees(
	ctx context.Context,
	companyID int64,
	currency string,
	status int64,
	pagination types.Pagination,
) ([]store.TransactionServiceFee, error) {
	return s.store.TransactionServiceFees.ListByCompany(ctx, companyID, currency, status, pagination)
}

func (s *ServiceFeeService) ListSettlements(
	ctx context.Context,
	companyID int64,
	currency string,
	pagination types.Pagination,
) ([]store.ServiceFeeSettlement, error) {
	return s.store.ServiceFeeSettlements.ListByCompany(ctx, companyID, currency, pagination)
}

// Settle — xizmat pulini 0 qilish (yakunlash). Har bir amal alohida yozuv.
func (s *ServiceFeeService) Settle(
	ctx context.Context,
	companyID, userID int64,
	amount int64,
	currency, details string,
) (*store.ServiceFeeSettlement, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("MIQDOR MUSBAT BO'LISHI KERAK")
	}
	currency = normalizeFeeCurrency(currency)

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	feeStorage := store.NewTransactionServiceFeeStorage(tx)
	settlementStorage := store.NewServiceFeeSettlementStorage(tx)
	itemStorage := store.NewServiceFeeSettlementItemStorage(tx)

	pending, err := feeStorage.ListPendingFIFO(ctx, companyID, currency)
	if err != nil {
		return nil, err
	}

	var available int64
	for _, f := range pending {
		available += f.RemainingAmount
	}
	if available < amount {
		return nil, fmt.Errorf("Taqsimlanmagan xizmat puli yetarli emas")
	}

	st := &store.ServiceFeeSettlement{
		CompanyID: companyID,
		UserID:    userID,
		Amount:    amount,
		Currency:  currency,
		Details:   details,
	}
	if err := settlementStorage.Create(ctx, st); err != nil {
		return nil, err
	}

	left := amount
	for i := range pending {
		if left <= 0 {
			break
		}
		f := &pending[i]
		use := f.RemainingAmount
		if use > left {
			use = left
		}
		f.RemainingAmount -= use
		if f.RemainingAmount <= 0 {
			f.RemainingAmount = 0
			f.Status = store.ServiceFeeStatusSettled
		}
		if err := feeStorage.Update(ctx, f); err != nil {
			return nil, err
		}
		if err := itemStorage.Create(ctx, st.ID, f.ID, use); err != nil {
			return nil, err
		}
		left -= use
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return st, nil
}
