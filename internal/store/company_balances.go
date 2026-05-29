package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type CompanyBalance struct {
	ID        int64  `json:"id"`
	CompanyID int64  `json:"company_id"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	InOutLay  int64  `json:"in_out_lay"`
	OutInLay  int64  `json:"out_in_lay"`
	CreatedAt string `json:"created_at"`
}

type CompanyBalanceStorage struct {
	db DBTX
}

func NewCompanyBalanceStorage(db DBTX) *CompanyBalanceStorage {
	return &CompanyBalanceStorage{db: db}
}

func (s *CompanyBalanceStorage) Create(ctx context.Context, cb *CompanyBalance) error {
	query := `INSERT INTO company_balances (company_id, currency, balance, in_out_lay, out_in_lay)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (company_id, currency) DO NOTHING
		RETURNING id, created_at`
	err := s.db.QueryRowContext(ctx, query,
		cb.CompanyID, cb.Currency, cb.Balance, cb.InOutLay, cb.OutInLay,
	).Scan(&cb.ID, &cb.CreatedAt)
	if err == sql.ErrNoRows {
		existing, lookupErr := s.lookup(ctx, cb.CompanyID, cb.Currency)
		if lookupErr != nil {
			return lookupErr
		}
		*cb = *existing
		return nil
	}
	return err
}

func (s *CompanyBalanceStorage) lookup(ctx context.Context, companyID int64, currency string) (*CompanyBalance, error) {
	query := `SELECT id, company_id, currency, balance, in_out_lay, out_in_lay, created_at
		FROM company_balances WHERE company_id = $1 AND currency = $2`
	cb := &CompanyBalance{}
	err := s.db.QueryRowContext(ctx, query, companyID, currency).Scan(
		&cb.ID, &cb.CompanyID, &cb.Currency, &cb.Balance, &cb.InOutLay, &cb.OutInLay, &cb.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return cb, nil
}

func (s *CompanyBalanceStorage) GetByCompanyIdAndCurrency(ctx context.Context, companyID int64, currency string) (*CompanyBalance, error) {
	return s.lookup(ctx, companyID, currency)
}

// AggregateByCompanyId — kompaniya balansini user balanslaridan jamlab hisoblaydi
// (har bir valyuta uchun SUM). company_balances jadvaliga bog'liq emas — drift bo'lmaydi,
// chunki har bir operatsiya allaqachon balances jadvalini yangilaydi.
func (s *CompanyBalanceStorage) AggregateByCompanyId(ctx context.Context, companyID int64) ([]CompanyBalance, error) {
	query := `
		SELECT currency,
		       COALESCE(SUM(balance), 0)    AS balance,
		       COALESCE(SUM(in_out_lay), 0) AS in_out_lay,
		       COALESCE(SUM(out_in_lay), 0) AS out_in_lay
		FROM balances
		WHERE company_id = $1 AND currency IS NOT NULL AND currency != ''
		GROUP BY currency
		ORDER BY currency`
	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CompanyBalance{}
	for rows.Next() {
		cb := CompanyBalance{CompanyID: companyID}
		if err := rows.Scan(&cb.Currency, &cb.Balance, &cb.InOutLay, &cb.OutInLay); err != nil {
			return nil, err
		}
		out = append(out, cb)
	}
	return out, rows.Err()
}

// CompanyBalanceRecordRow — kompaniya balansiga kirim/chiqim qatori,
// operatsiyani bajargan hodim (user_id + username) bilan birga.
type CompanyBalanceRecordRow struct {
	ID                 int64  `json:"id"`
	Amount             int64  `json:"amount"`
	UserID             int64  `json:"user_id"`
	Username           string `json:"username"`
	Currency           string `json:"currency"`
	Type               int64  `json:"type"`
	Details            string `json:"details"`
	TransactionId      *int64 `json:"transaction_id"`
	DebtId             *int64 `json:"debt_id"`
	ExchangeId         *int64 `json:"exchange_id"`
	CreatedAtFormatted string `json:"created_at"`
}

// ListRecordsByCompanyAndCurrency — kompaniya bo'yicha balance_records ro'yxati.
// currency bo'sh bo'lmasa, faqat o'sha valyuta qatorlari qaytadi.
func (s *CompanyBalanceStorage) ListRecordsByCompanyAndCurrency(ctx context.Context, companyID int64, currency string, pagination types.Pagination) ([]CompanyBalanceRecordRow, error) {
	query := `
		SELECT br.id, br.amount, br.user_id, COALESCE(u.username, ''),
		       br.currency, br.type, COALESCE(br.details, ''),
		       br.transaction_id, br.debt_id, br.exchange_id, br.created_at
		FROM balance_records br
		LEFT JOIN users u ON u.id = br.user_id
		WHERE br.company_id = $1 AND br.status != $2 AND br.amount != 0`
	args := []any{companyID, STATUS_ARCHIVED}
	if currency != "" {
		query += " AND br.currency = $3"
		args = append(args, currency)
	}
	query += " ORDER BY br.created_at DESC"
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

func (s *CompanyBalanceStorage) GetByCompanyId(ctx context.Context, companyID int64) ([]CompanyBalance, error) {
	query := `SELECT id, company_id, currency, balance, in_out_lay, out_in_lay, created_at
		FROM company_balances WHERE company_id = $1 ORDER BY currency`
	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompanyBalance
	for rows.Next() {
		var cb CompanyBalance
		if err := rows.Scan(&cb.ID, &cb.CompanyID, &cb.Currency, &cb.Balance, &cb.InOutLay, &cb.OutInLay, &cb.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, cb)
	}
	return out, rows.Err()
}

func (s *CompanyBalanceStorage) Update(ctx context.Context, cb *CompanyBalance) error {
	query := `UPDATE company_balances SET balance = $1, in_out_lay = $2, out_in_lay = $3 WHERE id = $4`
	res, err := s.db.ExecContext(ctx, query, cb.Balance, cb.InOutLay, cb.OutInLay, cb.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("company balance not found")
	}
	return nil
}

func (s *CompanyBalanceStorage) EnsureDefaults(ctx context.Context, companyID int64, currencies []string) error {
	for _, c := range currencies {
		cb := &CompanyBalance{CompanyID: companyID, Currency: c}
		if err := s.Create(ctx, cb); err != nil {
			return fmt.Errorf("company balance %s: %w", c, err)
		}
	}
	return nil
}

type UserActivityRow struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Currency string `json:"currency"`
	Total    int64  `json:"total"`
}

func (s *CompanyBalanceStorage) UserActivityByCompany(ctx context.Context, companyID int64) ([]UserActivityRow, error) {
	query := `
		SELECT br.user_id, COALESCE(u.username, ''), br.currency, COALESCE(SUM(ABS(br.amount)), 0)
		FROM balance_records br
		LEFT JOIN users u ON u.id = br.user_id
		WHERE br.company_id = $1 AND br.status != $2
		GROUP BY br.user_id, u.username, br.currency
		ORDER BY br.user_id, br.currency`
	rows, err := s.db.QueryContext(ctx, query, companyID, STATUS_ARCHIVED)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserActivityRow
	for rows.Next() {
		var r UserActivityRow
		if err := rows.Scan(&r.UserID, &r.Username, &r.Currency, &r.Total); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
