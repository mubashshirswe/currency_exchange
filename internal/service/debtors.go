package service

import (
	"context"
	"strings"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type DebtorsService struct {
	store store.Storage
}

const (
	// debtorSearchMinScore — trigram o'xshashlik chegarasi: bundan past natijalar
	// tasodifiy mos kelish deb qaraladi. 0.3 — kichik imlo xatolarini o'tkazadi.
	debtorSearchMinScore = 0.3
	debtorSearchLimit    = 10
	debtorSearchMaxLimit = 50
)

// SearchByName — transaction'da qarz yozishdan oldin ism bo'yicha mavjud qarzdorlarni
// taklif qiladi. Harflar xato yozilgan bo'lsa ham mos qarzdorlar chiqadi.
// Bo'sh ro'yxat qaytsa — frontend yangi qarzdor yaratish yo'lidan boradi.
func (s *DebtorsService) SearchByName(
	ctx context.Context,
	businessID, companyID int64,
	query, currency string,
	limit int,
) ([]map[string]interface{}, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []map[string]interface{}{}, nil
	}

	if limit <= 0 {
		limit = debtorSearchLimit
	}
	if limit > debtorSearchMaxLimit {
		limit = debtorSearchMaxLimit
	}

	matches, err := s.store.Debtors.SearchByName(
		ctx, companyID, query, strings.TrimSpace(currency), debtorSearchMinScore, limit,
	)
	if err != nil {
		return nil, err
	}

	users, err := s.store.Users.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}

	res := make([]map[string]interface{}, 0, len(matches))
	for _, m := range matches {
		res = append(res, map[string]interface{}{
			"id":         m.ID,
			"balance":    m.Balance,
			"currency":   m.Currency,
			"username":   GetUser(users, m.UserID).Username,
			"user_id":    m.UserID,
			"company_id": m.CompanyID,
			"phone":      m.Phone,
			"full_name":  m.FullName,
			"score":      m.Score,
			"created_at": m.CreatedAt,
		})
	}

	return res, nil
}

func (s *DebtorsService) GetByCompanyId(ctx context.Context, businessID int64, companyId int64, search *string, dateSearch *string, pagination types.Pagination) ([]map[string]interface{}, error) {

	debtors, err := s.store.Debtors.GetByCompanyId(ctx, companyId, search, dateSearch, pagination)
	if err != nil {
		return nil, err
	}

	users, err := s.store.Users.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}

	res := make([]map[string]interface{}, 0, len(debtors))

	for _, debtor := range debtors {

		res = append(res, map[string]interface{}{
			"id":         debtor.ID,
			"balance":    debtor.Balance,
			"currency":   debtor.Currency,
			"username":   GetUser(users, debtor.UserID).Username,
			"user_id":    debtor.UserID,
			"company_id": debtor.CompanyID,
			"phone":      debtor.Phone,
			"full_name":  debtor.FullName,
			"created_at": debtor.CreatedAt,
		})
	}

	return res, nil
}
