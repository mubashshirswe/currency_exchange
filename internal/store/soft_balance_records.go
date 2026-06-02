package store

import (
	"context"
	"fmt"
	"time"

	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type SoftBalanceRecord struct {
	ID            int64  `json:"id"`
	CompanyID     int64  `json:"company_id"`
	UserID        int64  `json:"user_id"`
	SoftBalanceID int64  `json:"soft_balance_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Type          int64  `json:"type"`
	Details       string `json:"details"`
	Status        int64  `json:"status"`
	ExchangeId    *int64 `json:"exchange_id"`
	CreatedAt     string `json:"created_at"`
}

type SoftBalanceRecordRow struct {
	ID                 int64  `json:"id"`
	Amount             int64  `json:"amount"`
	UserID             int64  `json:"user_id"`
	Username           string `json:"username"`
	Currency           string `json:"currency"`
	Type               int64  `json:"type"`
	Details            string `json:"details"`
	CreatedAtFormatted string `json:"created_at"`
}

type SoftBalanceRecordStorage struct {
	db DBTX
}

func NewSoftBalanceRecordStorage(db DBTX) *SoftBalanceRecordStorage {
	return &SoftBalanceRecordStorage{db: db}
}

func (s *SoftBalanceRecordStorage) Create(ctx context.Context, r *SoftBalanceRecord) error {
	query := `INSERT INTO soft_balance_records
		(company_id, user_id, soft_balance_id, amount, currency, type, details, status, exchange_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return s.db.QueryRowContext(ctx, query,
		r.CompanyID, r.UserID, r.SoftBalanceID, r.Amount, r.Currency, r.Type, r.Details, r.Status, r.ExchangeId,
	).Scan(&r.ID, &r.CreatedAt)
}

func (s *SoftBalanceRecordStorage) GetById(ctx context.Context, id int64) (*SoftBalanceRecord, error) {
	query := `SELECT id, company_id, user_id, COALESCE(soft_balance_id, 0),
		amount, currency, type, COALESCE(details, ''), status, created_at
		FROM soft_balance_records WHERE id = $1`
	r := &SoftBalanceRecord{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.CompanyID, &r.UserID, &r.SoftBalanceID,
		&r.Amount, &r.Currency, &r.Type, &r.Details, &r.Status, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *SoftBalanceRecordStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM soft_balance_records WHERE id = $1`, id)
	return err
}

func (s *SoftBalanceRecordStorage) ListByLink(ctx context.Context, field string, id int64) ([]SoftBalanceRecord, error) {
	query := fmt.Sprintf(`SELECT id, company_id, user_id, COALESCE(soft_balance_id, 0),
		amount, currency, type, COALESCE(details, ''), status, exchange_id, created_at
		FROM soft_balance_records WHERE %s = $1`, field)

	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SoftBalanceRecord{}
	for rows.Next() {
		var r SoftBalanceRecord
		if err := rows.Scan(
			&r.ID, &r.CompanyID, &r.UserID, &r.SoftBalanceID,
			&r.Amount, &r.Currency, &r.Type, &r.Details, &r.Status, &r.ExchangeId, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SoftBalanceRecordStorage) ListByCompany(
	ctx context.Context,
	companyID int64,
	currency string,
	pagination types.Pagination,
) ([]SoftBalanceRecordRow, error) {
	query := `
		SELECT r.id, r.amount, r.user_id, COALESCE(u.username, ''),
		       r.currency, r.type, COALESCE(r.details, ''), r.created_at
		FROM soft_balance_records r
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
	out := []SoftBalanceRecordRow{}
	for rows.Next() {
		var r SoftBalanceRecordRow
		var createdAt time.Time
		if err := rows.Scan(
			&r.ID, &r.Amount, &r.UserID, &r.Username,
			&r.Currency, &r.Type, &r.Details, &createdAt,
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
