package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// CompanyBalanceRecordService — kompaniya balansi uchun MUSTAQIL servis.
// Faqat company_balances va company_balance_records jadvallariga yozadi.
// User balanslarga (balances) va eski balance_records'ga TEGMAYDI.
type CompanyBalanceRecordService struct {
	store store.Storage
}

// PerformCompanyBalanceRecord — kirim/chiqimni company_balances'ga qo'llaydi va
// company_balance_records'ga yozadi (operatsiyani bajargan hodim user_id bilan).
func (s *CompanyBalanceRecordService) PerformCompanyBalanceRecord(ctx context.Context, payload types.CompanyBalanceRecordPayload) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	apply := func(currency string, amount int64, recordType int64) error {
		cb, err := cbStorage.GetByCompanyIdAndCurrency(ctx, payload.CompanyID, currency)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			// Bu valyuta uchun kompaniya balansi hali yo'q — ochamiz.
			cb = &store.CompanyBalance{CompanyID: payload.CompanyID, Currency: currency}
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
			CompanyID:        payload.CompanyID,
			UserID:           payload.UserId,
			CompanyBalanceID: cb.ID,
			Amount:           amount,
			Currency:         currency,
			Type:             recordType,
			Details:          payload.Details,
			Status:           store.STATUS_CREATED,
		}
		return cbrStorage.Create(ctx, rec)
	}

	if payload.ReceivedMoney > 0 {
		if err := apply(payload.ReceivedCurrency, payload.ReceivedMoney, TYPE_BUY); err != nil {
			tx.Rollback()
			return err
		}
	}

	if payload.SelledMoney > 0 {
		if err := apply(payload.SelledCurrency, payload.SelledMoney, TYPE_SELL); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	return nil
}

// RollbackCompanyBalanceRecord — yozuvni bekor qiladi: company_balances'ga teskari
// ta'sir qo'llanadi va yozuv o'chiriladi. User balanslarga tegmaydi.
func (s *CompanyBalanceRecordService) RollbackCompanyBalanceRecord(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	rec, err := cbrStorage.GetById(ctx, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	cb, err := cbStorage.GetByCompanyIdAndCurrency(ctx, rec.CompanyID, rec.Currency)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := reverseCompanyBalance(cb, rec.Type, rec.Amount); err != nil {
		tx.Rollback()
		return err
	}

	if err := cbStorage.Update(ctx, cb); err != nil {
		tx.Rollback()
		return err
	}

	if err := cbrStorage.Delete(ctx, rec.ID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// UpdateCompanyBalanceRecord — mavjud yozuvni yangilaydi: eski ta'sir bekor qilinadi,
// yangi ta'sir qo'llanadi va yozuv yangilanadi. Valyuta o'zgarmaydi (eski qoladi).
func (s *CompanyBalanceRecordService) UpdateCompanyBalanceRecord(ctx context.Context, payload store.CompanyBalanceRecord) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	old, err := cbrStorage.GetById(ctx, payload.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	cb, err := cbStorage.GetByCompanyIdAndCurrency(ctx, old.CompanyID, old.Currency)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Eski ta'sirni bekor qilamiz
	if err := reverseCompanyBalance(cb, old.Type, old.Amount); err != nil {
		tx.Rollback()
		return err
	}

	// Yangi ta'sirni qo'llaymiz
	if err := applyCompanyBalance(cb, payload.Type, payload.Amount); err != nil {
		tx.Rollback()
		return err
	}

	if err := cbStorage.Update(ctx, cb); err != nil {
		tx.Rollback()
		return err
	}

	// Yozuvni yangilaymiz (valyuta/kompaniya/user o'zgarmaydi)
	old.Amount = payload.Amount
	old.Type = payload.Type
	old.Details = payload.Details
	if err := cbrStorage.Update(ctx, old); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// applyCompanyBalance — kirim/chiqimni company balansga qo'llaydi.
func applyCompanyBalance(cb *store.CompanyBalance, recordType int64, amount int64) error {
	switch recordType {
	case TYPE_BUY: // kirim
		cb.Balance += amount
		cb.OutInLay += amount
	case TYPE_SELL: // chiqim
		if cb.Balance < amount {
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
		cb.Balance -= amount
		cb.InOutLay += amount
	default:
		return fmt.Errorf("unknown record type %d", recordType)
	}
	return nil
}

// reverseCompanyBalance — kirim/chiqimning teskarisini company balansga qo'llaydi.
func reverseCompanyBalance(cb *store.CompanyBalance, recordType int64, amount int64) error {
	switch recordType {
	case TYPE_BUY: // kirimni bekor qilish
		if cb.Balance < amount {
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
		cb.Balance -= amount
		cb.OutInLay -= amount
	case TYPE_SELL: // chiqimni bekor qilish
		cb.Balance += amount
		cb.InOutLay -= amount
	default:
		return fmt.Errorf("unknown record type %d", recordType)
	}
	return nil
}
