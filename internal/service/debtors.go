package service

import (
	"context"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type DebtorsService struct {
	store store.Storage
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
