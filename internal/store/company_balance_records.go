package store

import (
	"context"
	"fmt"
	"time"

	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// CompanyBalanceRecord — kompaniya balansiga kirim/chiqim yozuvi.
// Eski balance_records'dan butunlay alohida jadval.
type CompanyBalanceRecord struct {
	ID               int64  `json:"id"`
	CompanyID        int64  `json:"company_id"`
	UserID           int64  `json:"user_id"`
	CompanyBalanceID int64  `json:"company_balance_id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Type             int64  `json:"type"`
	Details          string `json:"details"`
	Status           int64  `json:"status"`
	ExchangeId       *int64 `json:"exchange_id"`
	TransactionId    *int64 `json:"transaction_id"`
	DebtId           *int64 `json:"debt_id"`
	CreatedAt        string `json:"created_at"`
}

type CompanyBalanceRecordStorage struct {
	db DBTX
}

func NewCompanyBalanceRecordStorage(db DBTX) *CompanyBalanceRecordStorage {
	return &CompanyBalanceRecordStorage{db: db}
}

func (s *CompanyBalanceRecordStorage) Create(ctx context.Context, r *CompanyBalanceRecord) error {
	query := `INSERT INTO company_balance_records
		(company_id, user_id, company_balance_id, amount, currency, type, details, status, exchange_id, transaction_id, debt_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`
	return s.db.QueryRowContext(ctx, query,
		r.CompanyID, r.UserID, r.CompanyBalanceID, r.Amount, r.Currency, r.Type, r.Details, r.Status,
		r.ExchangeId, r.TransactionId, r.DebtId,
	).Scan(&r.ID, &r.CreatedAt)
}

func (s *CompanyBalanceRecordStorage) GetById(ctx context.Context, id int64) (*CompanyBalanceRecord, error) {
	query := `SELECT id, company_id, user_id, COALESCE(company_balance_id, 0),
		amount, currency, type, COALESCE(details, ''), status,
		exchange_id, transaction_id, debt_id, created_at
		FROM company_balance_records WHERE id = $1`
	r := &CompanyBalanceRecord{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.CompanyID, &r.UserID, &r.CompanyBalanceID,
		&r.Amount, &r.Currency, &r.Type, &r.Details, &r.Status,
		&r.ExchangeId, &r.TransactionId, &r.DebtId, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *CompanyBalanceRecordStorage) Update(ctx context.Context, r *CompanyBalanceRecord) error {
	query := `UPDATE company_balance_records
		SET amount = $1, currency = $2, type = $3, details = $4, status = $5
		WHERE id = $6`
	res, err := s.db.ExecContext(ctx, query, r.Amount, r.Currency, r.Type, r.Details, r.Status, r.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("company balance record %d not found", r.ID)
	}
	return nil
}

func (s *CompanyBalanceRecordStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM company_balance_records WHERE id = $1`, id)
	return err
}

// ListByLink — bog'langan operatsiya (exchange/transaction/debt) bo'yicha yozuvlarni qaytaradi.
// field faqat ichki konstanta: "exchange_id" | "transaction_id" | "debt_id".
func (s *CompanyBalanceRecordStorage) ListByLink(ctx context.Context, field string, id int64) ([]CompanyBalanceRecord, error) {
	var col string
	switch field {
	case "exchange_id":
		col = "exchange_id"
	case "transaction_id":
		col = "transaction_id"
	case "debt_id":
		col = "debt_id"
	default:
		return nil, fmt.Errorf("invalid link field %s", field)
	}

	query := fmt.Sprintf(`SELECT id, company_id, user_id, COALESCE(company_balance_id, 0),
		amount, currency, type, COALESCE(details, ''), status,
		exchange_id, transaction_id, debt_id, created_at
		FROM company_balance_records WHERE %s = $1 AND status != $2`, col)

	rows, err := s.db.QueryContext(ctx, query, id, STATUS_ARCHIVED)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CompanyBalanceRecord{}
	for rows.Next() {
		var r CompanyBalanceRecord
		if err := rows.Scan(
			&r.ID, &r.CompanyID, &r.UserID, &r.CompanyBalanceID,
			&r.Amount, &r.Currency, &r.Type, &r.Details, &r.Status,
			&r.ExchangeId, &r.TransactionId, &r.DebtId, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListByCompany — kompaniya bo'yicha kirim/chiqim tarixi (operatsiyani bajargan
// hodim user_id + username bilan). currency bo'sh bo'lmasa, faqat o'sha valyuta.
// CompanyBalanceRecordRow turi (company_balances.go'da) qayta ishlatiladi; bu yangi
// jadvalda transaction/debt/exchange id bo'lmagani uchun ular nil qoladi.
func (s *CompanyBalanceRecordStorage) ListByCompany(ctx context.Context, companyID int64, currency string, pagination types.Pagination) ([]CompanyBalanceRecordRow, error) {
	query := `
		SELECT r.id, r.amount, r.user_id, COALESCE(u.username, ''),
		       r.currency, r.type, COALESCE(r.details, ''),
		       r.transaction_id, r.debt_id, r.exchange_id, r.created_at
		FROM company_balance_records r
		LEFT JOIN users u ON u.id = r.user_id
		WHERE r.company_id = $1 AND r.status != $2`
	args := []any{companyID, STATUS_ARCHIVED}
	if currency != "" {
		query += " AND r.currency = $3"
		args = append(args, currency)
	}
	query += " ORDER BY r.created_at DESC"
	query += fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Asia/Tashkent")
	out := []CompanyBalanceRecordRow{}
	for rows.Next() {
		var r CompanyBalanceRecordRow
		var createdAt time.Time
		if err := rows.Scan(
			&r.ID, &r.Amount, &r.UserID, &r.Username,
			&r.Currency, &r.Type, &r.Details,
			&r.TransactionId, &r.DebtId, &r.ExchangeId, &createdAt,
		); err != nil {
			return nil, err
		}
		if loc != nil {
			r.CreatedAtFormatted = createdAt.In(loc).Format("2006-01-02 15:04:05")
		} else {
			r.CreatedAtFormatted = createdAt.Format("2006-01-02 15:04:05")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
