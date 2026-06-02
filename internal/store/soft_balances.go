package store

import (
	"context"
	"database/sql"
)

type SoftBalance struct {
	ID        int64  `json:"id"`
	CompanyID int64  `json:"company_id"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	CreatedAt string `json:"created_at"`
}

type SoftBalanceStorage struct {
	db DBTX
}

func NewSoftBalanceStorage(db DBTX) *SoftBalanceStorage {
	return &SoftBalanceStorage{db: db}
}

func (s *SoftBalanceStorage) lookup(ctx context.Context, companyID int64, currency string) (*SoftBalance, error) {
	query := `SELECT id, company_id, currency, balance, created_at
		FROM soft_balances WHERE company_id = $1 AND currency = $2`
	sb := &SoftBalance{}
	err := s.db.QueryRowContext(ctx, query, companyID, currency).Scan(
		&sb.ID, &sb.CompanyID, &sb.Currency, &sb.Balance, &sb.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sb, nil
}

func (s *SoftBalanceStorage) Create(ctx context.Context, sb *SoftBalance) error {
	query := `INSERT INTO soft_balances (company_id, currency, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (company_id, currency) DO NOTHING
		RETURNING id, created_at`
	err := s.db.QueryRowContext(ctx, query, sb.CompanyID, sb.Currency, sb.Balance).
		Scan(&sb.ID, &sb.CreatedAt)
	if err == sql.ErrNoRows {
		existing, lookupErr := s.lookup(ctx, sb.CompanyID, sb.Currency)
		if lookupErr != nil {
			return lookupErr
		}
		*sb = *existing
		return nil
	}
	return err
}

func (s *SoftBalanceStorage) GetByCompanyIdAndCurrency(ctx context.Context, companyID int64, currency string) (*SoftBalance, error) {
	return s.lookup(ctx, companyID, currency)
}

func (s *SoftBalanceStorage) GetByCompanyId(ctx context.Context, companyID int64) ([]SoftBalance, error) {
	query := `SELECT id, company_id, currency, balance, created_at
		FROM soft_balances WHERE company_id = $1 ORDER BY currency`
	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SoftBalance
	for rows.Next() {
		var sb SoftBalance
		if err := rows.Scan(&sb.ID, &sb.CompanyID, &sb.Currency, &sb.Balance, &sb.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sb)
	}
	return out, rows.Err()
}

func (s *SoftBalanceStorage) Update(ctx context.Context, sb *SoftBalance) error {
	query := `UPDATE soft_balances SET balance = $1 WHERE id = $2`
	res, err := s.db.ExecContext(ctx, query, sb.Balance, sb.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SoftBalanceStorage) EnsureDefaults(ctx context.Context, companyID int64, currencies []string) error {
	for _, cur := range currencies {
		if err := s.Create(ctx, &SoftBalance{CompanyID: companyID, Currency: cur, Balance: 0}); err != nil {
			return err
		}
	}
	return nil
}
