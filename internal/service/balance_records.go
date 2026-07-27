package service

import (
	"context"
	"fmt"
	"log"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type BalanceRecordService struct {
	store store.Storage
}

func (s *BalanceRecordService) PerformBalanceRecord(ctx context.Context, balanceRecord types.BalanceRecordPayload) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)
	usersStorage := store.NewUserStorage(tx)

	user, err := usersStorage.GetById(ctx, &balanceRecord.UserId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// SELLED MONEY PERFORM
	if balanceRecord.SelledMoney > 0 {
		fmt.Println("balanceRecord.UserId: ", &balanceRecord.UserId, "balanceRecord.SelledCurrency", balanceRecord.SelledCurrency)

		selledCurrencyBalance, err := balancesStorage.GetByUserIdAndCurrency(ctx, &balanceRecord.UserId, balanceRecord.SelledCurrency)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf(types.BALANCE_CURRENCY_NOT_FOUND)
		}

		fmt.Println("selledCurrencyBalance: ", selledCurrencyBalance)

		if selledCurrencyBalance.Balance < balanceRecord.SelledMoney {
			tx.Rollback()
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
		selledCurrencyBalance.Balance -= balanceRecord.SelledMoney
		selledCurrencyBalance.InOutLay += balanceRecord.SelledMoney

		fmt.Println("selledCurrencyBalance: ", selledCurrencyBalance)

		if err := balancesStorage.Update(ctx, selledCurrencyBalance); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE UPDATING selledCurrencyBalance %v", err)
		}

		selledMoneyRecord := &store.BalanceRecord{
			Amount:    balanceRecord.SelledMoney,
			Currency:  balanceRecord.SelledCurrency,
			CompanyID: user.CompanyId,
			BalanceID: selledCurrencyBalance.ID,
			Details:   balanceRecord.Details,
			UserID:    balanceRecord.UserId,
			Type:      TYPE_SELL,
		}

		if err := balanceRecordsStorage.Create(ctx, selledMoneyRecord); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE CREATING BALANCE RECORD %v", err)
		}
	}

	// RECEIVED MONEY PERFORM
	if balanceRecord.ReceivedMoney > 0 {
		receivedCurrencyBalance, err := balancesStorage.GetByUserIdAndCurrency(ctx, &balanceRecord.UserId, balanceRecord.ReceivedCurrency)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf(types.BALANCE_CURRENCY_NOT_FOUND)
		}

		if receivedCurrencyBalance != nil {
			receivedCurrencyBalance.Balance += balanceRecord.ReceivedMoney
			receivedCurrencyBalance.OutInLay += balanceRecord.ReceivedMoney
		}

		if err := balancesStorage.Update(ctx, receivedCurrencyBalance); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE UPDATING receivedCurrencyBalance %v", err)
		}

		receivedMoneyRecord := &store.BalanceRecord{
			Amount:    balanceRecord.ReceivedMoney,
			Currency:  balanceRecord.ReceivedCurrency,
			CompanyID: user.CompanyId,
			BalanceID: receivedCurrencyBalance.ID,
			Details:   balanceRecord.Details,
			UserID:    balanceRecord.UserId,
			Type:      TYPE_BUY,
		}

		if err := balanceRecordsStorage.Create(ctx, receivedMoneyRecord); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE CREATING BALANCE RECORD %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	log.Println("Transaction committed successfully")
	return nil
}

func (s *BalanceRecordService) RollbackBalanceRecord(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)

	records, err := balanceRecordsStorage.ListByFieldInternal(ctx, "id", id, types.Pagination{Limit: 100, Offset: 0})
	if err != nil {
		tx.Rollback()
		return err
	}
	record := records[0]

	balance, err := balancesStorage.GetById(ctx, &record.BalanceID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE GETTING BALANCE WITH ID %v", err)
	}

	switch record.Type {
	case TYPE_SELL:
		balance.Balance += record.Amount
		balance.InOutLay -= record.Amount
	case TYPE_BUY:
		if balance.Balance >= record.Amount {
			balance.Balance -= record.Amount
			balance.OutInLay -= record.Amount
		} else {
			tx.Rollback()
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
	default:
		tx.Rollback()
		return fmt.Errorf("FOUND UNKOWN RECORD TYPE")
	}

	if err := balancesStorage.Update(ctx, balance); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING BALANCE %v", err)
	}

	if err := balanceRecordsStorage.Delete(ctx, record.ID); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE DELETING BALANCE RECORD %v", err)
	}

	tx.Commit()
	return nil
}

func (s *BalanceRecordService) UpdateRecord(ctx context.Context, balanceRecord store.BalanceRecord) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)

	record, err := balanceRecordsStorage.ListByFieldInternal(ctx, "id", &balanceRecord.ID, types.Pagination{Limit: 100, Offset: 0})
	if err != nil {
		tx.Rollback()
		return err
	}
	oldRecord := record[0]

	balance, err := balancesStorage.GetById(ctx, &oldRecord.BalanceID)
	if err != nil {
		tx.Rollback()
		return err
	}

	switch oldRecord.Type {
	case TYPE_SELL:
		balance.Balance += oldRecord.Amount
		balance.InOutLay -= oldRecord.Amount
	case TYPE_BUY:
		if balance.Balance >= oldRecord.Amount {
			balance.Balance -= oldRecord.Amount
			balance.OutInLay -= oldRecord.Amount

		} else {
			tx.Rollback()
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
	}

	switch balanceRecord.Type {
	case TYPE_SELL:
		if balance.Balance >= balanceRecord.Amount {
			balance.Balance -= balanceRecord.Amount
			balance.InOutLay += balanceRecord.Amount
		} else {
			tx.Rollback()
			return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
		}
	case TYPE_BUY:
		balance.Balance += balanceRecord.Amount
		balance.OutInLay += balanceRecord.Amount
	}

	if err := balancesStorage.Update(ctx, balance); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING BALANCE %v", err)
	}

	if err := balanceRecordsStorage.Update(ctx, &balanceRecord); err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}
