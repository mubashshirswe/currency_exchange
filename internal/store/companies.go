package store

import (
	"context"
	"errors"
)

type Company struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Details    string `json:"details"`
	Password   string `json:"password"`
	BusinessID int64  `json:"business_id"`
	CreatedAt  string `json:"created_at"`
}

// companyColumns — ustunlar tartibi scanCompany bilan bir xil.
const companyColumns = `id, coalesce(name, ''), coalesce(details, ''), coalesce(password, ''), business_id, created_at`

type CompanyStorage struct {
	db DBTX
}

func NewCompanyStorage(db DBTX) *CompanyStorage {
	return &CompanyStorage{db: db}
}

func scanCompany(row rowScanner, company *Company) error {
	return row.Scan(
		&company.ID,
		&company.Name,
		&company.Details,
		&company.Password,
		&company.BusinessID,
		&company.CreatedAt,
	)
}

func (s *CompanyStorage) Create(ctx context.Context, company *Company) error {
	if company.BusinessID == 0 {
		return errors.New("business_id is required")
	}

	query := `INSERT INTO companies(name, details, password, business_id) VALUES($1, $2, $3, $4) RETURNING id, created_at`
	err := s.db.QueryRowContext(ctx, query, company.Name, company.Details, company.Password, company.BusinessID).Scan(
		&company.ID,
		&company.CreatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// Update — kompaniya faqat o'z businessi ichida yangilanadi.
func (s *CompanyStorage) Update(ctx context.Context, company *Company) error {
	if company.BusinessID == 0 {
		return errors.New("business_id is required")
	}

	query := `UPDATE companies SET name = $1, details = $2, password = $3 WHERE id = $4 AND business_id = $5`

	rows, err := s.db.ExecContext(ctx, query, company.Name, company.Details, company.Password, company.ID, company.BusinessID)

	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return errors.New("NOT FOUND")
	}

	return nil
}

// ListByBusiness — businessga tegishli kompaniyalar (jamoalar).
func (s *CompanyStorage) ListByBusiness(ctx context.Context, businessID int64) ([]Company, error) {
	query := `SELECT ` + companyColumns + ` FROM companies WHERE business_id = $1 ORDER BY id`

	var companies []Company
	rows, err := s.db.QueryContext(ctx, query, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		company := Company{}
		if err := scanCompany(rows, &company); err != nil {
			return nil, err
		}
		companies = append(companies, company)
	}

	return companies, rows.Err()
}

func (s *CompanyStorage) GetById(ctx context.Context, id *int64) (*Company, error) {
	query := `SELECT ` + companyColumns + ` FROM companies WHERE id = $1`

	company := &Company{}
	if err := scanCompany(s.db.QueryRowContext(ctx, query, id), company); err != nil {
		return nil, err
	}

	return company, nil
}

// GetByIdInBusiness — kompaniyani faqat berilgan business ichidan oladi.
func (s *CompanyStorage) GetByIdInBusiness(ctx context.Context, id int64, businessID int64) (*Company, error) {
	query := `SELECT ` + companyColumns + ` FROM companies WHERE id = $1 AND business_id = $2`

	company := &Company{}
	if err := scanCompany(s.db.QueryRowContext(ctx, query, id, businessID), company); err != nil {
		return nil, err
	}

	return company, nil
}

// BelongsToBusiness — kompaniya shu businessga tegishlimi (guard uchun arzon tekshiruv).
func (s *CompanyStorage) BelongsToBusiness(ctx context.Context, id int64, businessID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM companies WHERE id = $1 AND business_id = $2)`

	var ok bool
	if err := s.db.QueryRowContext(ctx, query, id, businessID).Scan(&ok); err != nil {
		return false, err
	}

	return ok, nil
}

func (s *CompanyStorage) Delete(ctx context.Context, id *int64, businessID int64) error {
	query := `DELETE FROM companies WHERE id = $1 AND business_id = $2`

	rows, err := s.db.ExecContext(ctx, query, id, businessID)

	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return errors.New("NOT FOUND")
	}

	return nil
}
