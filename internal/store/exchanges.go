package store

import (
	"context"
	"fmt"
	"time"

	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type Exchange struct {
	ID                 int64     `json:"id"`
	ReceivedMoney      int64     `json:"received_money"`
	ReceivedCurrency   string    `json:"received_currency"`
	SelledMoney        int64     `json:"selled_money"`
	SelledCurrency     string    `json:"selled_currency"`
	ProfitAmount       int64     `json:"profit_amount"`
	ProfitCurrency     string    `json:"profit_currency"`
	UserId             int64     `json:"user_id"`
	Details            string    `json:"details"`
	CompanyID          int64     `json:"company_id"`
	Status             int64     `json:"status"`
	CreatedAt          time.Time `json:"-"`
	CreatedAtFormatted string    `json:"created_at"`
}

type ExchangeStorage struct {
	db DBTX
}

func NewExchangeStorage(db DBTX) *ExchangeStorage {
	return &ExchangeStorage{db: db}
}

func (s *ExchangeStorage) Archive(ctx context.Context, companyId int64) error {
	query := `UPDATE exchanges SET status = $1 WHERE company_id = $2`
	rows, err := s.db.ExecContext(ctx, query, STATUS_ARCHIVED, companyId)
	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return fmt.Errorf("ERROR NOT FOUND")
	}

	return nil
}

func (s *ExchangeStorage) Create(ctx context.Context, exchange *Exchange) error {
	query := `INSERT INTO exchanges(received_money, received_currency, selled_money, selled_currency,
				profit_amount, profit_currency, user_id, company_id, details, status)
				VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at`

	err := s.db.QueryRowContext(
		ctx,
		query,
		exchange.ReceivedMoney,
		exchange.ReceivedCurrency,
		exchange.SelledMoney,
		exchange.SelledCurrency,
		exchange.ProfitAmount,
		exchange.ProfitCurrency,
		exchange.UserId,
		exchange.CompanyID,
		exchange.Details,
		STATUS_CREATED,
	).Scan(
		&exchange.ID,
		&exchange.CreatedAt,
	)

	return err
}

func (s *ExchangeStorage) Update(ctx context.Context, exchange *Exchange) error {
	query := `
				UPDATE exchanges SET received_money = $1, received_currency = $2, 
				selled_money = $3, selled_currency = $4, profit_amount = $5, profit_currency = $6,
				user_id = $7, company_id = $8, details = $9 WHERE id = $10`

	rows, err := s.db.ExecContext(
		ctx,
		query,
		exchange.ReceivedMoney,
		exchange.ReceivedCurrency,
		exchange.SelledMoney,
		exchange.SelledCurrency,
		exchange.ProfitAmount,
		exchange.ProfitCurrency,
		exchange.UserId,
		exchange.CompanyID,
		exchange.Details,
		exchange.ID,
	)
	if err != nil {
		return err
	}
	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return fmt.Errorf("NOT FOUND")
	}

	return err
}

func (s *ExchangeStorage) Archived(ctx context.Context, pagination types.Pagination) ([]Exchange, error) {
	query := `
				SELECT id, received_money, received_currency, selled_money,
				selled_currency, profit_amount, profit_currency, user_id, company_id, details, created_at 
				FROM exchanges WHERE status = $1  	ORDER BY created_at DESC
	` + fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		STATUS_ARCHIVED,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exchanges []Exchange
	for rows.Next() {
		exchage := &Exchange{}
		err := rows.Scan(
			&exchage.ID,
			&exchage.ReceivedMoney,
			&exchage.ReceivedCurrency,
			&exchage.SelledMoney,
			&exchage.SelledCurrency,
			&exchage.ProfitAmount,
			&exchage.ProfitCurrency,
			&exchage.UserId,
			&exchage.CompanyID,
			&exchage.Details,
			&exchage.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		loc, _ := time.LoadLocation("Asia/Tashkent")
		createdAtInTashkent := exchage.CreatedAt.In(loc)
		exchage.CreatedAtFormatted = createdAtInTashkent.Format("2006-01-02 15:04:05")

		exchanges = append(exchanges, *exchage)
	}

	return exchanges, nil
}

func (s *ExchangeStorage) GetByField(ctx context.Context, fieldName string, fieldValue any, pagination types.Pagination) ([]Exchange, error) {
	query := `
				SELECT id, received_money, received_currency, selled_money,
				selled_currency, profit_amount, profit_currency, user_id, company_id, details, created_at 
				FROM exchanges WHERE status != $1 AND ` + fmt.Sprintf(" %v = %v ORDER BY created_at DESC OFFSET %v LIMIT %v", fieldName, fieldValue, pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		STATUS_ARCHIVED,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exchanges []Exchange
	for rows.Next() {
		exchage := &Exchange{}
		err := rows.Scan(
			&exchage.ID,
			&exchage.ReceivedMoney,
			&exchage.ReceivedCurrency,
			&exchage.SelledMoney,
			&exchage.SelledCurrency,
			&exchage.ProfitAmount,
			&exchage.ProfitCurrency,
			&exchage.UserId,
			&exchage.CompanyID,
			&exchage.Details,
			&exchage.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		loc, _ := time.LoadLocation("Asia/Tashkent")
		createdAtInTashkent := exchage.CreatedAt.In(loc)
		exchage.CreatedAtFormatted = createdAtInTashkent.Format("2006-01-02 15:04:05")

		exchanges = append(exchanges, *exchage)
	}

	return exchanges, nil
}

func (s *ExchangeStorage) GetById(ctx context.Context, id int64) (*Exchange, error) {
	query := `
				SELECT id, received_money, received_currency, selled_money, 
				selled_currency, profit_amount, profit_currency, user_id, company_id, details, created_at 
				FROM exchanges WHERE id = $1`

	exchage := &Exchange{}

	err := s.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&exchage.ID,
		&exchage.ReceivedMoney,
		&exchage.ReceivedCurrency,
		&exchage.SelledMoney,
		&exchage.SelledCurrency,
		&exchage.ProfitAmount,
		&exchage.ProfitCurrency,
		&exchage.UserId,
		&exchage.CompanyID,
		&exchage.Details,
		&exchage.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation("Asia/Tashkent")
	createdAtInTashkent := exchage.CreatedAt.In(loc)
	exchage.CreatedAtFormatted = createdAtInTashkent.Format("2006-01-02 15:04:05")

	return exchage, nil
}

func (s *ExchangeStorage) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM exchanges WHERE id  = $1`

	rows, err := s.db.ExecContext(
		ctx,
		query,
		id,
	)
	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return fmt.Errorf("NOT FOUND")
	}

	return nil
}
