package store

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// Business — tenant. Ierarxiya: business => company => users.
// Har bir business boshqasidan to'liq izolyatsiya qilingan.
type Business struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Details   string `json:"details"`
	Phone     string `json:"phone"`
	Status    int64  `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

const (
	BUSINESS_ACTIVE   = 1
	BUSINESS_DISABLED = 2
)

const businessColumns = `id, name, coalesce(details, ''), coalesce(phone, ''), status, created_at, updated_at`

type BusinessStorage struct {
	db DBTX
}

func NewBusinessStorage(db DBTX) *BusinessStorage {
	return &BusinessStorage{db: db}
}

func (s *BusinessStorage) Create(ctx context.Context, business *Business) error {
	query := `INSERT INTO businesses(name, details, phone, status)
				VALUES($1, $2, $3, $4) RETURNING id, created_at, updated_at`

	if business.Status == 0 {
		business.Status = BUSINESS_ACTIVE
	}

	return s.db.QueryRowContext(
		ctx,
		query,
		business.Name,
		business.Details,
		business.Phone,
		business.Status,
	).Scan(
		&business.ID,
		&business.CreatedAt,
		&business.UpdatedAt,
	)
}

func (s *BusinessStorage) GetById(ctx context.Context, id int64) (*Business, error) {
	query := `SELECT ` + businessColumns + ` FROM businesses WHERE id = $1`

	business := &Business{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&business.ID,
		&business.Name,
		&business.Details,
		&business.Phone,
		&business.Status,
		&business.CreatedAt,
		&business.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return business, nil
}

// ListByIDs — berilgan id'lar bo'yicha businesslar, id => Business ko'rinishida.
// Topilmagan id javobda umuman bo'lmaydi (xato emas).
func (s *BusinessStorage) ListByIDs(ctx context.Context, ids []int64) (map[int64]Business, error) {
	result := make(map[int64]Business, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, "$"+strconv.Itoa(i+1))
		args = append(args, id)
	}

	query := `SELECT ` + businessColumns + ` FROM businesses WHERE id IN (` +
		strings.Join(placeholders, ", ") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		business := Business{}
		if err := rows.Scan(
			&business.ID,
			&business.Name,
			&business.Details,
			&business.Phone,
			&business.Status,
			&business.CreatedAt,
			&business.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result[business.ID] = business
	}

	return result, rows.Err()
}

func (s *BusinessStorage) Update(ctx context.Context, business *Business) error {
	query := `UPDATE businesses SET name = $1, details = $2, phone = $3, updated_at = now() WHERE id = $4`

	res, err := s.db.ExecContext(ctx, query, business.Name, business.Details, business.Phone, business.ID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("BUSINESS NOT FOUND")
	}

	return nil
}

func (s *BusinessStorage) SetStatus(ctx context.Context, id int64, status int64) error {
	query := `UPDATE businesses SET status = $1, updated_at = now() WHERE id = $2`

	res, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("BUSINESS NOT FOUND")
	}

	return nil
}

func (s *BusinessStorage) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM businesses WHERE id = $1`

	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("BUSINESS NOT FOUND")
	}

	return nil
}
