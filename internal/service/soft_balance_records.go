package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type SoftBalanceRecordService struct {
	store store.Storage
}

func (s *SoftBalanceRecordService) PerformSoftBalanceRecord(ctx context.Context, payload types.CompanyBalanceRecordPayload) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	sbStorage := store.NewSoftBalanceStorage(tx)
	sbrStorage := store.NewSoftBalanceRecordStorage(tx)

	apply := func(currency string, amount int64, recordType int64) error {
		sb, err := sbStorage.GetByCompanyIdAndCurrency(ctx, payload.CompanyID, currency)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			sb = &store.SoftBalance{CompanyID: payload.CompanyID, Currency: currency}
			if cerr := sbStorage.Create(ctx, sb); cerr != nil {
				return cerr
			}
		}

		if err := applySoftBalance(sb, recordType, amount); err != nil {
			return err
		}

		if err := sbStorage.Update(ctx, sb); err != nil {
			return err
		}

		rec := &store.SoftBalanceRecord{
			CompanyID:     payload.CompanyID,
			UserID:        payload.UserId,
			SoftBalanceID: sb.ID,
			Amount:        amount,
			Currency:      currency,
			Type:          recordType,
			Details:       payload.Details,
			Status:        store.STATUS_CREATED,
		}
		return sbrStorage.Create(ctx, rec)
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

	return tx.Commit()
}

func (s *SoftBalanceRecordService) RollbackSoftBalanceRecord(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	sbStorage := store.NewSoftBalanceStorage(tx)
	sbrStorage := store.NewSoftBalanceRecordStorage(tx)

	rec, err := sbrStorage.GetById(ctx, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	sb, err := sbStorage.GetByCompanyIdAndCurrency(ctx, rec.CompanyID, rec.Currency)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := reverseSoftBalance(sb, rec.Type, rec.Amount); err != nil {
		tx.Rollback()
		return err
	}

	if err := sbStorage.Update(ctx, sb); err != nil {
		tx.Rollback()
		return err
	}

	if err := sbrStorage.Delete(ctx, rec.ID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func applySoftBalance(sb *store.SoftBalance, recordType int64, amount int64) error {
	switch recordType {
	case TYPE_BUY:
		sb.Balance += amount
	case TYPE_SELL:
		if sb.Balance < amount {
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
		sb.Balance -= amount
	default:
		return fmt.Errorf("unknown record type %d", recordType)
	}
	return nil
}

func reverseSoftBalance(sb *store.SoftBalance, recordType int64, amount int64) error {
	switch recordType {
	case TYPE_BUY:
		if sb.Balance < amount {
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
		sb.Balance -= amount
	case TYPE_SELL:
		sb.Balance += amount
	default:
		return fmt.Errorf("unknown record type %d", recordType)
	}
	return nil
}
