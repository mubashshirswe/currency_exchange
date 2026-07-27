package main

import (
	"fmt"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// GetCompanyBalancesHandler — kompaniya balansi (har bir valyuta uchun).
// company_balances jadvalidan to'g'ridan-to'g'ri o'qiladi — user balanslaridan
// MUSTAQIL. Faqat company_balance_records orqali yangilanadi.
func (app *application) GetCompanyBalancesHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companyID := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	balances, err := app.store.CompanyBalances.GetByCompanyId(r.Context(), companyID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, balances); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetCompanyBalanceRecordsHandler — kompaniya balansiga kirim/chiqim tarixi.
// ?currency=USD bilan valyuta bo'yicha filtrlanadi; ?page=&limit= bilan pagination.
// Har bir qatorda operatsiyani bajargan hodim (user_id + username) ko'rsatiladi.
func (app *application) GetCompanyBalanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companyID := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	currency := r.URL.Query().Get("currency")
	app.LoadPaginationInfo(r, r.Context())

	rows, err := app.store.CompanyBalanceRecords.ListByCompany(r.Context(), companyID, currency, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, rows); err != nil {
		app.internalServerError(w, r, err)
	}
}

// currentCompanyID — joriy foydalanuvchining kompaniya id'si (tenant kontekstidan).
func (app *application) currentCompanyID(r *http.Request) (int64, error) {
	t := tenantFrom(r)
	if t.CompanyID == 0 {
		return 0, fmt.Errorf("foydalanuvchi hech qanday kompaniyaga biriktirilmagan")
	}
	return t.CompanyID, nil
}

func (app *application) currentUser(r *http.Request) (*store.User, error) {
	userID, _ := r.Context().Value(UserKey).(int64)
	return app.store.Users.GetById(r.Context(), &userID)
}

// isAdminUser — business egasi (barcha kompaniyalarni ko'radi).
func (app *application) isAdminUser(user *store.User) bool {
	return user != nil && user.Role == store.ROLE_OWNER
}

// GetMyCompanyBalancesHandler — joriy foydalanuvchi kompaniyasining balansi (valyutalar bo'yicha).
func (app *application) GetMyCompanyBalancesHandler(w http.ResponseWriter, r *http.Request) {
	companyID, err := app.currentCompanyID(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	balances, err := app.store.CompanyBalances.GetByCompanyId(r.Context(), companyID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, balances); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetMyCompanyBalanceRecordsHandler — joriy foydalanuvchi kompaniyasining kirim/chiqim tarixi.
// ?currency=USD bilan valyuta bo'yicha; ?page=&limit= bilan pagination.
func (app *application) GetMyCompanyBalanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	companyID, err := app.currentCompanyID(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	currency := r.URL.Query().Get("currency")
	app.LoadPaginationInfo(r, r.Context())

	rows, err := app.store.CompanyBalanceRecords.ListByCompany(r.Context(), companyID, currency, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, rows); err != nil {
		app.internalServerError(w, r, err)
	}
}

// CreateMyCompanyBalanceRecordHandler — kompaniya balansiga kirim/chiqim (deposit/withdraw).
// MUSTAQIL: faqat company_balances + company_balance_records'ga ta'sir qiladi, user
// balanslarga (balances) tegmaydi. Operatsiyani bajargan hodim user_id va kompaniya
// id JWT'dan olinadi (mobil yubormaydi).
func (app *application) CreateMyCompanyBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	var payload types.CompanyBalanceRecordPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	userID, _ := r.Context().Value(UserKey).(int64)
	companyID, err := app.currentCompanyID(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	payload.UserId = userID
	payload.CompanyID = companyID

	if err := app.service.CompanyBalanceRecords.PerformCompanyBalanceRecord(r.Context(), payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
	}
}

// UpdateMyCompanyBalanceRecordHandler — mavjud kompaniya balansi yozuvini yangilaydi.
func (app *application) UpdateMyCompanyBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	var payload store.CompanyBalanceRecord
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	payload.ID = getIDFromContext(r)
	if err := app.authorizeResource(r, t, "company_balance_records", payload.ID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.CompanyBalanceRecords.UpdateCompanyBalanceRecord(r.Context(), payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteMyCompanyBalanceRecordHandler — kompaniya balansi yozuvini bekor qiladi (rollback).
func (app *application) DeleteMyCompanyBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "company_balance_records", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.CompanyBalanceRecords.RollbackCompanyBalanceRecord(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetCompanyUserActivityHandler — yangi endpoint: balance_records bo'yicha user faolligi.
func (app *application) GetCompanyUserActivityHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companyID := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	rows, err := app.store.CompanyBalances.UserActivityByCompany(r.Context(), companyID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, rows); err != nil {
		app.internalServerError(w, r, err)
	}
}
