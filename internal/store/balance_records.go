package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type BalanceRecord struct {
	ID                 int64     `json:"id"`
	Amount             int64     `json:"amount"`
	UserID             int64     `json:"user_id"`
	BalanceID          int64     `json:"balance_id"`
	CompanyID          int64     `json:"company_id"`
	TransactionId      *int64    `json:"transaction_id"`
	DebtId             *int64    `json:"debt_id"`
	ExchangeId         *int64    `json:"exchange_id"`
	Details            string    `json:"details"`
	Currency           string    `json:"currency"`
	Type               int64     `json:"type"`
	Status             int64     `json:"status"`
	CreatedAt          time.Time `json:"-"`
	CreatedAtFormatted string    `json:"created_at"`
}

type BalanceRecordStorage struct {
	db DBTX
}

func NewBalanceRecordStorage(db DBTX) *BalanceRecordStorage {
	return &BalanceRecordStorage{db: db}
}

func (s *BalanceRecordStorage) Archive(ctx context.Context, companyId int64) error {
	query := `UPDATE balance_records SET status = $1 WHERE company_id = $2`
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

func (s *BalanceRecordStorage) Create(ctx context.Context, balanceRecord *BalanceRecord) error {
	query := `
		INSERT INTO balance_records (
			amount, user_id, balance_id, company_id,
			transaction_id, debt_id, exchange_id,
			details, currency, type, status
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10, $11
		)
		RETURNING id, created_at
	`

	jsonD, _ := json.Marshal(balanceRecord)
	fmt.Printf("balanceRecord %v", string(jsonD))

	row := s.db.QueryRowContext(ctx, query,
		balanceRecord.Amount,
		balanceRecord.UserID,
		balanceRecord.BalanceID,
		balanceRecord.CompanyID,
		balanceRecord.TransactionId,
		balanceRecord.DebtId,
		balanceRecord.ExchangeId,
		balanceRecord.Details,
		balanceRecord.Currency,
		balanceRecord.Type,
		STATUS_CREATED,
	)

	if err := row.Scan(&balanceRecord.ID, &balanceRecord.CreatedAt); err != nil {
		return fmt.Errorf("failed to insert balance_record: %w", err)
	}

	return nil
}

func (s *BalanceRecordStorage) Update(ctx context.Context, balanceRecord *BalanceRecord) error {
	query := `
				UPDATE balance_records SET amount = $1, user_id = $2, balance_id = $3, company_id = $4, transaction_id = $5, debt_id = $6,  
				details = $7, currency = $8, type = $9, exchange_id = $10 WHERE id = $11
			`

	_, err := s.db.ExecContext(
		ctx,
		query,
		balanceRecord.Amount,
		balanceRecord.UserID,
		balanceRecord.BalanceID,
		balanceRecord.CompanyID,
		balanceRecord.TransactionId,
		balanceRecord.DebtId,
		balanceRecord.Details,
		balanceRecord.Currency,
		balanceRecord.Type,
		balanceRecord.ExchangeId,
		balanceRecord.ID,
	)

	return err
}

func (s *BalanceRecordStorage) GetByFieldAndDate(ctx context.Context, businessID int64, fieldName string, from, to *string, fieldValue any, pagination types.Pagination) ([]BalanceRecord, error) {
	if err := checkFilterField(balanceRecordFilterFields, fieldName); err != nil {
		return nil, err
	}

	query := `
				SELECT id, amount, user_id, balance_id, company_id, transaction_id, debt_id, exchange_id, details, currency, type, created_at
				FROM balance_records WHERE ` + fieldName + ` = $1 AND status != $2  AND amount != 0 AND created_at BETWEEN $3 AND $4 AND business_id = $5 	ORDER BY created_at DESC
	` + fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		fieldValue,
		STATUS_ARCHIVED,
		from,
		to,
		businessID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.FetchDataFromQuery(rows)
}

// ListByFieldInternal — service ichidagi bog'liq yozuvlarni topish uchun (rollback,
// update). Business filtri yo'q: chaqiruvchi allaqachon avtorizatsiya qilingan
// yozuvning id'si bilan ishlaydi, tashqi kirish nuqtasi emas.
func (s *BalanceRecordStorage) ListByFieldInternal(ctx context.Context, fieldName string, fieldValue any, pagination types.Pagination) ([]BalanceRecord, error) {
	if err := checkFilterField(balanceRecordFilterFields, fieldName); err != nil {
		return nil, err
	}

	query := `
				SELECT id, amount, user_id, balance_id, company_id, transaction_id, debt_id, exchange_id, details, currency, type, created_at
				FROM balance_records WHERE ` + fieldName + ` = $1 AND status != $2 AND amount != 0 	ORDER BY created_at DESC
	` + fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(ctx, query, fieldValue, STATUS_ARCHIVED)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.FetchDataFromQuery(rows)
}

func (s *BalanceRecordStorage) GetByField(ctx context.Context, businessID int64, fieldName string, fieldValue any, pagination types.Pagination) ([]BalanceRecord, error) {
	if err := checkFilterField(balanceRecordFilterFields, fieldName); err != nil {
		return nil, err
	}

	query := `
				SELECT id, amount, user_id, balance_id, company_id, transaction_id, debt_id, exchange_id, details, currency, type, created_at
				FROM balance_records WHERE ` + fieldName + ` = $1 AND status != $2 AND amount != 0 AND business_id = $3 	ORDER BY created_at DESC
	` + fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		fieldValue,
		STATUS_ARCHIVED,
		businessID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.FetchDataFromQuery(rows)
}

func (s *BalanceRecordStorage) Archived(ctx context.Context, businessID int64, pagination types.Pagination) ([]BalanceRecord, error) {
	query := `
				SELECT id, amount, user_id, balance_id, company_id, transaction_id, debt_id, exchange_id, details, currency, type, created_at
				FROM balance_records WHERE status = $1 AND amount != 0 AND business_id = $2 	ORDER BY created_at DESC
	` + fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)
	rows, err := s.db.QueryContext(
		ctx,
		query,
		STATUS_ARCHIVED,
		businessID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.FetchDataFromQuery(rows)
}

func (s *BalanceRecordStorage) FetchDataFromQuery(rows *sql.Rows) ([]BalanceRecord, error) {
	var balanceRecords []BalanceRecord
	for rows.Next() {
		balance := BalanceRecord{}

		err := rows.Scan(
			&balance.ID,
			&balance.Amount,
			&balance.UserID,
			&balance.BalanceID,
			&balance.CompanyID,
			&balance.TransactionId,
			&balance.DebtId,
			&balance.ExchangeId,
			&balance.Details,
			&balance.Currency,
			&balance.Type,
			&balance.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		loc, _ := time.LoadLocation("Asia/Tashkent")
		createdAtInTashkent := balance.CreatedAt.In(loc)
		balance.CreatedAtFormatted = createdAtInTashkent.Format("2006-01-02 15:04:05")

		balanceRecords = append(balanceRecords, balance)
	}

	return balanceRecords, nil
}

func (s *BalanceRecordStorage) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM balance_records WHERE id = $1`

	_, err := s.db.ExecContext(
		ctx,
		query,
		id,
	)

	return err
}

func (s *BalanceRecordStorage) DeleteByExchangeId(ctx context.Context, id int64) error {
	query := `DELETE FROM balance_records WHERE exchange_id = $1`

	_, err := s.db.ExecContext(
		ctx,
		query,
		id,
	)

	return err
}

func (s *BalanceRecordStorage) DeleteByDebtId(ctx context.Context, id int64) error {
	query := `DELETE FROM balance_records WHERE debt_id = $1`

	_, err := s.db.ExecContext(
		ctx,
		query,
		id,
	)

	return err
}
