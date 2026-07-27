package store

import (
	"context"
	"errors"
	"fmt"
)

type Balance struct {
	ID        int64  `json:"id"`
	Balance   int64  `json:"balance"`
	UserId    int64  `json:"user_id"`
	InOutLay  int64  `json:"in_out_lay"`
	OutInLay  int64  `json:"out_in_lay"`
	CompanyId int64  `json:"company_id"`
	Currency  string `json:"currency"`
	CreatedAt string `json:"created_at"`
}

type BalanceStorage struct {
	db DBTX
}

func NewBalanceStorage(db DBTX) *BalanceStorage {
	return &BalanceStorage{db: db}
}

func (s *BalanceStorage) Create(ctx context.Context, balance *Balance) error {
	query := `SELECT * FROM balances WHERE user_id = $1 and currency = $2`
	rows, err := s.db.ExecContext(ctx, query, balance.UserId, balance.Currency)
	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}
	if res != 0 {
		return fmt.Errorf("ALREADY EXIST WITH CURRENCY %v", balance.Currency)
	}

	query = `INSERT INTO balances(balance, user_id, in_out_lay, out_in_lay, company_id, currency)
				VALUES($1, $2, $3, $4, $5, $6) RETURNING id, created_at`

	err = s.db.QueryRowContext(
		ctx,
		query,
		balance.Balance,
		balance.UserId,
		balance.InOutLay,
		balance.OutInLay,
		balance.CompanyId,
		balance.Currency).Scan(
		&balance.ID,
		&balance.CreatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// GetAll — faqat berilgan business ichidagi barcha balanslar.
func (s *BalanceStorage) GetAll(ctx context.Context, businessID int64) ([]Balance, error) {
	query := `SELECT id, balance, user_id, in_out_lay, out_in_lay, company_id, currency, created_at
				FROM balances WHERE business_id = $1`
	rows, err := s.db.QueryContext(ctx, query, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		balance := &Balance{}
		err := rows.Scan(
			&balance.ID,
			&balance.Balance,
			&balance.UserId,
			&balance.InOutLay,
			&balance.OutInLay,
			&balance.CompanyId,
			&balance.Currency,
			&balance.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		balances = append(balances, *balance)
	}

	return balances, nil
}

func (s *BalanceStorage) GetByUserIdAndCurrency(ctx context.Context, userID *int64, currency string) (*Balance, error) {
	query := `SELECT id, balance, user_id, in_out_lay, out_in_lay, company_id, currency, created_at  FROM balances WHERE user_id = $1 AND currency = $2`
	balance := &Balance{}

	err := s.db.QueryRowContext(
		ctx,
		query,
		userID,
		currency,
	).Scan(
		&balance.ID,
		&balance.Balance,
		&balance.UserId,
		&balance.InOutLay,
		&balance.OutInLay,
		&balance.CompanyId,
		&balance.Currency,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (s *BalanceStorage) GetByCompanyId(ctx context.Context, userId *int64) ([]Balance, error) {
	query := `
				SELECT id, balance, user_id, in_out_lay, out_in_lay, company_id, currency, created_at 
				FROM balances WHERE company_id = $1`
	rows, err := s.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		balance := &Balance{}
		err := rows.Scan(
			&balance.ID,
			&balance.Balance,
			&balance.UserId,
			&balance.InOutLay,
			&balance.OutInLay,
			&balance.CompanyId,
			&balance.Currency,
			&balance.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		balances = append(balances, *balance)
	}

	return balances, nil
}

func (s *BalanceStorage) GetByUserId(ctx context.Context, userId *int64) ([]Balance, error) {
	query := `
				SELECT id, balance, user_id, in_out_lay, out_in_lay, company_id, currency, created_at 
				FROM balances WHERE user_id = $1`
	rows, err := s.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		balance := &Balance{}
		err := rows.Scan(
			&balance.ID,
			&balance.Balance,
			&balance.UserId,
			&balance.InOutLay,
			&balance.OutInLay,
			&balance.CompanyId,
			&balance.Currency,
			&balance.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		balances = append(balances, *balance)
	}

	return balances, nil
}

func (s *BalanceStorage) GetById(ctx context.Context, id *int64) (*Balance, error) {
	query := `SELECT id, balance, user_id, in_out_lay, out_in_lay, company_id, currency, created_at  FROM balances WHERE id = $1`
	balance := &Balance{}

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&balance.ID,
		&balance.Balance,
		&balance.UserId,
		&balance.InOutLay,
		&balance.OutInLay,
		&balance.CompanyId,
		&balance.Currency,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (s *BalanceStorage) Update(ctx context.Context, balance *Balance) error {
	query := `UPDATE balances SET balance = $1, in_out_lay = $2, out_in_lay = $3
		WHERE id = $4`

	rows, err := s.db.ExecContext(ctx, query, balance.Balance, balance.InOutLay, balance.OutInLay, balance.ID)
	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return errors.New("BALANCE THAT WILL BE UPDATE IS NOT FOUND")
	}

	return nil
}

func (s *BalanceStorage) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM balances WHERE id = $1`
	rows, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return errors.New("BALANCE NOT FOUND")
	}

	return nil
}
