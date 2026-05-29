package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
