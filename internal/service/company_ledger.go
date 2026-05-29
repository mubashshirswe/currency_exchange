package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/mubashshir3767/currencyExchange/internal/env"
	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// CompanyLedgerEnabled — true bo'lganda kompaniya balansi user balansiga qo'shimcha yangilanadi.
// Default false: mavjud tranzaksiya/exchange/debt/balance-record oqimlari o'zgarmaydi.
func CompanyLedgerEnabled() bool {
	return env.GetBool("COMPANY_LEDGER_ENABLED", false)
}

// CompanyBalanceChange — kompaniya balansiga ta'sir (faqat company_balances jadvali).
type CompanyBalanceChange struct {
	CompanyID  int64
	Currency   string
	Amount     int64
	RecordType int64
}

// MaybeApplyCompanyBalanceChange — COMPANY_LEDGER_ENABLED=true bo'lganda company_balances ni yangilaydi.
// balance_records va user balances ga tegmaydi — eski kod bilan bir xil qoladi.
func MaybeApplyCompanyBalanceChange(ctx context.Context, tx store.DBTX, p CompanyBalanceChange) error {
	if !CompanyLedgerEnabled() {
		return nil
	}
	return ApplyCompanyBalanceChange(ctx, tx, p)
}

func ApplyCompanyBalanceChange(ctx context.Context, tx store.DBTX, p CompanyBalanceChange) error {
	if p.Amount < 0 {
		p.Amount = int64(math.Abs(float64(p.Amount)))
	}

	cbStore := store.NewCompanyBalanceStorage(tx)
	cb, err := cbStore.GetByCompanyIdAndCurrency(ctx, p.CompanyID, p.Currency)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("company %d has no %s balance", p.CompanyID, p.Currency)
		}
		return err
	}

	switch p.RecordType {
	case TYPE_SELL:
		if cb.Balance < p.Amount {
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
		cb.Balance -= p.Amount
		cb.InOutLay += p.Amount
	case TYPE_BUY:
		cb.Balance += p.Amount
		cb.OutInLay += p.Amount
	default:
		return fmt.Errorf("unknown record type %d", p.RecordType)
	}

	return cbStore.Update(ctx, cb)
}

// MaybeReverseCompanyBalanceChange — ledger yoqilganda operatsiyani bekor qilish.
func MaybeReverseCompanyBalanceChange(ctx context.Context, tx store.DBTX, companyID int64, currency string, recordType int64, amount int64) error {
	if !CompanyLedgerEnabled() {
		return nil
	}
	if amount < 0 {
		amount = -amount
	}

	cbStore := store.NewCompanyBalanceStorage(tx)
	cb, err := cbStore.GetByCompanyIdAndCurrency(ctx, companyID, currency)
	if err != nil {
		return err
	}

	switch recordType {
	case TYPE_SELL:
		cb.Balance += amount
		cb.InOutLay -= amount
	case TYPE_BUY:
		if cb.Balance < amount {
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
		cb.Balance -= amount
		cb.OutInLay -= amount
	default:
		return fmt.Errorf("unknown record type %d", recordType)
	}

	return cbStore.Update(ctx, cb)
}
