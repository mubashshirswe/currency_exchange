package store

import (
	"context"
	"errors"
)

// Rollar. Business egasi (ROLE_OWNER) o'z businessidagi hamma narsani ko'radi,
// oddiy hodim (ROLE_STAFF) faqat o'z kompaniyasi doirasida ishlaydi.
const (
	ROLE_OWNER = 1
	ROLE_STAFF = 2
)

type User struct {
	ID         int64   `json:"id"`
	Username   string  `json:"username"`
	Phone      string  `json:"phone"`
	Role       int64   `json:"role"`
	Password   string  `json:"password"`
	Avatar     *string `json:"avatar"`
	CompanyId  int64   `json:"company_id"`
	BusinessId int64   `json:"business_id"`
	CreatedAt  string  `json:"created_at"`
}

// userColumns — ustunlar tartibi scanUser bilan bir xil bo'lishi shart.
// `SELECT *` ishlatilmaydi: jadvalga ustun qo'shilganda scan buzilmasin.
const userColumns = `id, phone, role, avatar, username, password, coalesce(company_id, 0), business_id, created_at`

type UserStorage struct {
	db DBTX
}

func NewUserStorage(db DBTX) *UserStorage {
	return &UserStorage{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner, user *User) error {
	return row.Scan(
		&user.ID,
		&user.Phone,
		&user.Role,
		&user.Avatar,
		&user.Username,
		&user.Password,
		&user.CompanyId,
		&user.BusinessId,
		&user.CreatedAt,
	)
}

func (s *UserStorage) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users(username, phone, password, role, company_id, business_id)
				VALUES($1, $2, $3, $4, NULLIF($5, 0), $6) RETURNING id, created_at`

	if user.BusinessId == 0 {
		return errors.New("business_id is required")
	}

	err := s.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Phone,
		user.Password,
		user.Role,
		user.CompanyId,
		user.BusinessId).Scan(
		&user.ID,
		&user.CreatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// LoginCandidates — telefon+parol bo'yicha mos userlar. Telefon business ichida
// unikal, businesslar bo'ylab esa takrorlanishi mumkin, shuning uchun bir nechta
// natija qaytishi mumkin va chaqiruvchi business_id bilan aniqlashtiradi.
func (s *UserStorage) LoginCandidates(ctx context.Context, phone, password string) ([]User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE phone = $1 AND password = $2 ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query, phone, password)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user := User{}
		if err := scanUser(rows, &user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

func (s *UserStorage) GetById(ctx context.Context, id *int64) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	user := &User{}
	if err := scanUser(s.db.QueryRowContext(ctx, query, id), user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByIdInBusiness — userni faqat berilgan business ichidan oladi.
func (s *UserStorage) GetByIdInBusiness(ctx context.Context, id int64, businessID int64) (*User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1 AND business_id = $2`

	user := &User{}
	if err := scanUser(s.db.QueryRowContext(ctx, query, id, businessID), user); err != nil {
		return nil, err
	}

	return user, nil
}

// ListByBusiness — business jamoasi (barcha kompaniyalar hodimlari).
func (s *UserStorage) ListByBusiness(ctx context.Context, businessID int64) ([]User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE business_id = $1 ORDER BY id`

	return s.list(ctx, query, businessID)
}

// ListByCompany — bitta kompaniya jamoasi.
func (s *UserStorage) ListByCompany(ctx context.Context, businessID int64, companyID int64) ([]User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE business_id = $1 AND company_id = $2 ORDER BY id`

	return s.list(ctx, query, businessID, companyID)
}

func (s *UserStorage) list(ctx context.Context, query string, args ...any) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user := User{}
		if err := scanUser(rows, &user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Update — user faqat o'z businessi ichida yangilanadi; business_id o'zgarmaydi.
func (s *UserStorage) Update(ctx context.Context, user *User) error {
	query := `UPDATE users SET username = $1, password = $2, role = $3, avatar = $4, company_id = NULLIF($5, 0)
				WHERE id = $6 AND business_id = $7`

	if user.BusinessId == 0 {
		return errors.New("business_id is required")
	}

	result, err := s.db.ExecContext(
		ctx,
		query,
		user.Username,
		user.Password,
		user.Role,
		user.Avatar,
		user.CompanyId,
		user.ID,
		user.BusinessId)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// Delete — userni "o'chirish" (telefonini bo'shatish) faqat o'z businessi ichida.
func (s *UserStorage) Delete(ctx context.Context, id *int64, businessID int64) error {
	query := `UPDATE users SET phone = $1 WHERE id = $2 AND business_id = $3`

	res, err := s.db.ExecContext(
		ctx,
		query,
		"",
		id,
		businessID,
	)

	if err != nil {
		return err
	}

	result, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if result == 0 {
		return errors.New("NOT FOUND")
	}

	return nil
}
