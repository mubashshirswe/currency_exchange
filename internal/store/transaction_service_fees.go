package store

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

const (
	ServiceFeeStatusPending = 1 // taqsimlanmagan
	ServiceFeeStatusSettled = 2 // to'liq yakunlangan
)

type TransactionServiceFee struct {
	ID               int64  `json:"id"`
	TransactionID    int64  `json:"transaction_id"`
	CompanyID        int64  `json:"company_id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Details          string `json:"details"`
	Status           int64  `json:"status"`
	TransactionPhone string `json:"transaction_phone,omitempty"`
	TransactionNo    int64  `json:"transaction_number,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type ServiceFeeSettlement struct {
	ID        int64  `json:"id"`
	CompanyID int64  `json:"company_id"`
	UserID    int64  `json:"user_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Details   string `json:"details"`
	Username  string `json:"username,omitempty"`
	CreatedAt string `json:"created_at"`
}

type ServiceFeeRemainingRow struct {
	CompanyID int64
	Currency  string
	Remaining int64
}

type TransactionServiceFeeStorage struct {
	db DBTX
}

func NewTransactionServiceFeeStorage(db DBTX) *TransactionServiceFeeStorage {
	return &TransactionServiceFeeStorage{db: db}
}

func (s *TransactionServiceFeeStorage) Create(ctx context.Context, f *TransactionServiceFee) error {
	query := `INSERT INTO transaction_service_fees
		(transaction_id, company_id, amount, currency, details, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	var createdAt time.Time
	if err := s.db.QueryRowContext(ctx, query,
		f.TransactionID, f.CompanyID, f.Amount,
		f.Currency, f.Details, f.Status,
	).Scan(&f.ID, &createdAt); err != nil {
		return err
	}
	f.CreatedAt = formatTashkent(createdAt)
	return nil
}

func (s *TransactionServiceFeeStorage) GetByTransactionID(ctx context.Context, txID int64) (*TransactionServiceFee, error) {
	query := `SELECT id, transaction_id, company_id, amount,
		currency, COALESCE(details, ''), status, created_at
		FROM transaction_service_fees WHERE transaction_id = $1`
	f := &TransactionServiceFee{}
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx, query, txID).Scan(
		&f.ID, &f.TransactionID, &f.CompanyID, &f.Amount,
		&f.Currency, &f.Details, &f.Status, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	f.CreatedAt = formatTashkent(createdAt)
	return f, nil
}

func (s *TransactionServiceFeeStorage) Update(ctx context.Context, f *TransactionServiceFee) error {
	query := `UPDATE transaction_service_fees SET
		company_id = $1, amount = $2,
		currency = $3, details = $4, status = $5
		WHERE id = $6`
	_, err := s.db.ExecContext(ctx, query,
		f.CompanyID, f.Amount,
		f.Currency, f.Details, f.Status, f.ID,
	)
	return err
}

func (s *TransactionServiceFeeStorage) DeleteByTransactionID(ctx context.Context, txID int64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM transaction_service_fees WHERE transaction_id = $1`, txID)
	return err
}

func (s *TransactionServiceFeeStorage) ListPendingFIFO(
	ctx context.Context, companyID int64, currency string,
) ([]TransactionServiceFee, error) {
	query := `SELECT id, transaction_id, company_id, amount,
		currency, COALESCE(details, ''), status, created_at
		FROM transaction_service_fees
		WHERE company_id = $1 AND currency = $2
		  AND status = $3 AND amount > 0
		ORDER BY created_at ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, query, companyID, currency, ServiceFeeStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanServiceFees(rows)
}

// ListAllPending — barcha kompaniyalardagi status=1 (taqsimlanmagan) yozuvlar.
func (s *TransactionServiceFeeStorage) ListAllPending(
	ctx context.Context, currency string,
) ([]TransactionServiceFee, error) {
	query := `SELECT id, transaction_id, company_id, amount,
		currency, COALESCE(details, ''), status, created_at
		FROM transaction_service_fees
		WHERE status = $1 AND amount > 0`
	args := []any{ServiceFeeStatusPending}
	if currency != "" {
		query += ` AND currency = $2`
		args = append(args, currency)
	}
	query += ` ORDER BY created_at ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanServiceFees(rows)
}

func (s *TransactionServiceFeeStorage) ListByCompany(
	ctx context.Context,
	companyID int64,
	currency string,
	status int64,
	pagination types.Pagination,
) ([]TransactionServiceFee, error) {
	query := `
		SELECT f.id, f.transaction_id, f.company_id, f.amount,
		       f.currency, COALESCE(f.details, ''), f.status, f.created_at,
		       COALESCE(t.phone, ''),
		       COALESCE(NULLIF(trim(t.number::text), '')::bigint, 0)
		FROM transaction_service_fees f
		LEFT JOIN transactions t ON t.id = f.transaction_id
		WHERE f.company_id = $1`
	args := []any{companyID}
	argN := 2
	if currency != "" {
		query += fmt.Sprintf(" AND f.currency = $%d", argN)
		args = append(args, currency)
		argN++
	}
	if status > 0 {
		query += fmt.Sprintf(" AND f.status = $%d", argN)
		args = append(args, status)
		argN++
	}
	query += " ORDER BY f.created_at DESC"
	query += fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Asia/Tashkent")
	out := []TransactionServiceFee{}
	for rows.Next() {
		var f TransactionServiceFee
		var createdAt time.Time
		if err := rows.Scan(
			&f.ID, &f.TransactionID, &f.CompanyID, &f.Amount,
			&f.Currency, &f.Details, &f.Status, &createdAt,
			&f.TransactionPhone, &f.TransactionNo,
		); err != nil {
			return nil, err
		}
		if loc != nil {
			f.CreatedAt = createdAt.In(loc).Format("2006-01-02 15:04:05")
		} else {
			f.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *TransactionServiceFeeStorage) GetRemainingByCompanies(
	ctx context.Context, companyIDs []int64,
) ([]ServiceFeeRemainingRow, error) {
	query := `SELECT company_id, currency, COALESCE(SUM(amount), 0)::bigint
		FROM transaction_service_fees
		WHERE company_id = ANY($1) AND status = $2 AND amount > 0
		GROUP BY company_id, currency`
	rows, err := s.db.QueryContext(ctx, query, pq.Array(companyIDs), ServiceFeeStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ServiceFeeRemainingRow{}
	for rows.Next() {
		var r ServiceFeeRemainingRow
		if err := rows.Scan(&r.CompanyID, &r.Currency, &r.Remaining); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type ServiceFeeSettlementStorage struct {
	db DBTX
}

func NewServiceFeeSettlementStorage(db DBTX) *ServiceFeeSettlementStorage {
	return &ServiceFeeSettlementStorage{db: db}
}

func (s *ServiceFeeSettlementStorage) Create(ctx context.Context, st *ServiceFeeSettlement) error {
	query := `INSERT INTO service_fee_settlements
		(company_id, user_id, amount, currency, details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	var createdAt time.Time
	if err := s.db.QueryRowContext(ctx, query,
		st.CompanyID, st.UserID, st.Amount, st.Currency, st.Details,
	).Scan(&st.ID, &createdAt); err != nil {
		return err
	}
	st.CreatedAt = formatTashkent(createdAt)
	return nil
}

func (s *ServiceFeeSettlementStorage) ListByCompany(
	ctx context.Context,
	companyID int64,
	currency string,
	pagination types.Pagination,
) ([]ServiceFeeSettlement, error) {
	query := `
		SELECT s.id, s.company_id, s.user_id, s.amount, s.currency,
		       COALESCE(s.details, ''), COALESCE(u.username, ''), s.created_at
		FROM service_fee_settlements s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.company_id = $1`
	args := []any{companyID}
	if currency != "" {
		query += " AND s.currency = $2"
		args = append(args, currency)
	}
	query += " ORDER BY s.created_at DESC"
	query += fmt.Sprintf(" OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Asia/Tashkent")
	out := []ServiceFeeSettlement{}
	for rows.Next() {
		var st ServiceFeeSettlement
		var createdAt time.Time
		if err := rows.Scan(
			&st.ID, &st.CompanyID, &st.UserID, &st.Amount, &st.Currency,
			&st.Details, &st.Username, &createdAt,
		); err != nil {
			return nil, err
		}
		if loc != nil {
			st.CreatedAt = createdAt.In(loc).Format("2006-01-02 15:04:05")
		} else {
			st.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

type ServiceFeeSettlementItemStorage struct {
	db DBTX
}

func NewServiceFeeSettlementItemStorage(db DBTX) *ServiceFeeSettlementItemStorage {
	return &ServiceFeeSettlementItemStorage{db: db}
}

func (s *ServiceFeeSettlementItemStorage) Create(ctx context.Context, settlementID, feeID, amount int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO service_fee_settlement_items (settlement_id, service_fee_id, amount)
		 VALUES ($1, $2, $3)`,
		settlementID, feeID, amount,
	)
	return err
}

func scanServiceFees(rows feeRowScanner) ([]TransactionServiceFee, error) {
	loc, _ := time.LoadLocation("Asia/Tashkent")
	out := []TransactionServiceFee{}
	for rows.Next() {
		var f TransactionServiceFee
		var createdAt time.Time
		if err := rows.Scan(
			&f.ID, &f.TransactionID, &f.CompanyID, &f.Amount,
			&f.Currency, &f.Details, &f.Status, &createdAt,
		); err != nil {
			return nil, err
		}
		if loc != nil {
			f.CreatedAt = createdAt.In(loc).Format("2006-01-02 15:04:05")
		} else {
			f.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func formatTashkent(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return t.In(loc).Format("2006-01-02 15:04:05")
}

type feeRowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}
