package service

import (
	"context"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

type BalanceService struct {
	store store.Storage
}

func (s *BalanceService) GetByCompanyId(ctx context.Context, businessID int64, companyId int64) ([]map[string]interface{}, error) {
	balances, err := s.store.Balances.GetByCompanyId(ctx, &companyId)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}

	users, err := s.store.Users.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}

	currencies := make(map[string]int64)

	for _, balance := range balances {

		currencies[balance.Currency] += balance.Balance

		user := GetUser(users, balance.UserId)

		res := map[string]interface{}{
			"username": user.Username,
			"phone":    user.Phone,
			"balance":  balance.Balance,
			"currency": balance.Currency,
		}

		response = append(response, res)
	}
	response = append(response, map[string]interface{}{
		"currencies": currencies,
	})

	return response, nil
}

func (s *BalanceService) GetAll(ctx context.Context, businessID int64) ([]map[string]interface{}, error) {
	balances, err := s.store.Balances.GetAll(ctx, businessID)
	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}

	users, err := s.store.Users.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}

	companies, err := s.store.Companies.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}

	currencies := make(map[string]int64)

	for _, balance := range balances {

		currencies[balance.Currency] += balance.Balance

		user := GetUser(users, balance.UserId)

		res := map[string]interface{}{
			"username": user.Username,
			"phone":    user.Phone,
			"balance":  balance.Balance,
			"currency": balance.Currency,
			"company":  GetCompany(companies, balance.CompanyId).Name,
		}

		response = append(response, res)
	}
	response = append(response, map[string]interface{}{
		"currencies": currencies,
	})

	return response, nil
}

func GetUser(users []store.User, id int64) *store.User {
	for _, user := range users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}
