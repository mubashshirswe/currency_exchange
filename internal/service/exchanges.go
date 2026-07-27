package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type ExchangeService struct {
	store store.Storage
}

func (s *ExchangeService) Create(ctx context.Context, exchange *store.Exchange) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	exchangeStore := store.NewExchangeStorage(tx)
	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)
	usersStorage := store.NewUserStorage(tx)

	user, err := usersStorage.GetById(ctx, &exchange.UserId)
	if err != nil {
		tx.Rollback()
		return err
	}

	exchange.CompanyID = user.CompanyId

	if err := exchangeStore.Create(ctx, exchange); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE CREATING EXCHANGE  %v", err)
	}

	receivedCurrencyBalance, err := balancesStorage.GetByUserIdAndCurrency(ctx, &exchange.UserId, exchange.ReceivedCurrency)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf(types.BALANCE_CURRENCY_NOT_FOUND)
	}

	// RECEIVED MONEY PERFORM
	if receivedCurrencyBalance != nil {
		receivedCurrencyBalance.Balance += exchange.ReceivedMoney
		receivedCurrencyBalance.OutInLay += exchange.ReceivedMoney
	}

	if err := balancesStorage.Update(ctx, receivedCurrencyBalance); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING receivedCurrencyBalance %v", err)
	}

	receivedMoneyRecord := &store.BalanceRecord{
		Amount:     exchange.ReceivedMoney,
		Currency:   exchange.ReceivedCurrency,
		CompanyID:  user.CompanyId,
		BalanceID:  receivedCurrencyBalance.ID,
		Details:    exchange.Details,
		UserID:     exchange.UserId,
		Type:       TYPE_BUY,
		ExchangeId: &exchange.ID,
	}

	if err := balanceRecordsStorage.Create(ctx, receivedMoneyRecord); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE CREATING BALANCE RECORD %v", err)
	}

	selledCurrencyBalance, err := balancesStorage.GetByUserIdAndCurrency(ctx, &exchange.UserId, exchange.SelledCurrency)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf(types.BALANCE_CURRENCY_NOT_FOUND)
	}

	/// SELLED MONEY PERFORM
	if selledCurrencyBalance.Balance >= exchange.SelledMoney {
		selledCurrencyBalance.Balance -= exchange.SelledMoney
		selledCurrencyBalance.InOutLay += exchange.SelledMoney
	} else {
		tx.Rollback()
		return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
	}

	if err := balancesStorage.Update(ctx, selledCurrencyBalance); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING selledCurrencyBalance %v", err)
	}

	selledMoneyRecord := &store.BalanceRecord{
		Amount:     exchange.SelledMoney,
		Currency:   exchange.SelledCurrency,
		CompanyID:  user.CompanyId,
		BalanceID:  selledCurrencyBalance.ID,
		Details:    exchange.Details,
		UserID:     exchange.UserId,
		Type:       TYPE_SELL,
		ExchangeId: &exchange.ID,
	}

	if err := balanceRecordsStorage.Create(ctx, selledMoneyRecord); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE CREATING BALANCE RECORD %v", err)
	}

	tx.Commit()
	return nil
}

func (s *ExchangeService) Update(ctx context.Context, exchange *store.Exchange) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	exchangeStorage := store.NewExchangeStorage(tx)
	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)

	old, err := exchangeStorage.GetById(ctx, exchange.ID)
	if err != nil {
		return err
	}
	exchange.CompanyID = old.CompanyID

	records, err := balanceRecordsStorage.ListByFieldInternal(ctx, "exchange_id", exchange.ID, types.Pagination{Limit: 100, Offset: 0})
	if err != nil {
		return err
	}

	for _, record := range records {
		balance, err := balancesStorage.GetById(ctx, &record.BalanceID)
		if err != nil {
			return err
		}

		switch record.Type {
		case TYPE_BUY:
			if balance.Balance >= record.Amount {
				balance.Balance -= record.Amount
				balance.OutInLay -= record.Amount
			} else {
				return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
			}
		case TYPE_SELL:
			balance.Balance += record.Amount
			balance.InOutLay -= record.Amount
		}

		if record.Type == TYPE_BUY {
			record.Amount = exchange.ReceivedMoney
			record.Currency = exchange.ReceivedCurrency

			balance.Balance += record.Amount
			balance.OutInLay += record.Amount
		}

		if record.Type == TYPE_SELL {
			record.Amount = exchange.SelledMoney
			record.Currency = exchange.SelledCurrency

			balance.Balance -= record.Amount
			balance.InOutLay += record.Amount
		}

		if err := balancesStorage.Update(ctx, balance); err != nil {
			return fmt.Errorf("ERROR OCCURRED WHILE UPDATING BALANCE %v", err)
		}

		if err := balanceRecordsStorage.Update(ctx, &record); err != nil {
			return err
		}
	}

	if err := exchangeStorage.Update(ctx, exchange); err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func (s *ExchangeService) Delete(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	exchangeStorage := store.NewExchangeStorage(tx)
	balancesStorage := store.NewBalanceStorage(tx)
	balanceRecordsStorage := store.NewBalanceRecordStorage(tx)

	exchanges, err := exchangeStorage.ListByFieldInternal(ctx, "id", id, types.Pagination{Limit: 100, Offset: 0})
	if err != nil {
		tx.Rollback()
		return err
	}
	if len(exchanges) == 0 {
		return sql.ErrNoRows
	}

	exchange := exchanges[0]

	balances, err := balancesStorage.GetByUserId(ctx, &exchange.UserId)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE GETTING BALANCE WITH ID %v", err)
	}

	for _, balance := range balances {

		if balance.Currency == exchange.SelledCurrency {
			balance.Balance += exchange.SelledMoney
			balance.OutInLay -= exchange.ReceivedMoney
		}

		if balance.Currency == exchange.ReceivedCurrency {
			if balance.Balance >= exchange.ReceivedMoney {
				balance.Balance -= exchange.ReceivedMoney
				balance.OutInLay -= exchange.ReceivedMoney
			} else {
				return fmt.Errorf(types.BALANCE_NO_ENOUGH_MONEY)
			}
		}

		if err := balancesStorage.Update(ctx, &balance); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR OCCURRED WHILE UPDATING BALANCE %v", err)
		}
	}

	if err := balanceRecordsStorage.DeleteByExchangeId(ctx, exchange.ID); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE DELETING BALANCE RECORD %v", err)
	}

	if err := exchangeStorage.Delete(ctx, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("ERROR OCCURRED WHILE DELETING BALANCE RECORD %v", err)
	}

	tx.Commit()
	return nil
}
