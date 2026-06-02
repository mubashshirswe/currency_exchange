package main

import (
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// GetCompanyBalancesHandler — kompaniya balansi (har bir valyuta uchun).
// company_balances jadvalidan to'g'ridan-to'g'ri o'qiladi — user balanslaridan
// MUSTAQIL. Faqat company_balance_records orqali yangilanadi.
func (app *application) GetCompanyBalancesHandler(w http.ResponseWriter, r *http.Request) {
	companyID := getIDFromContext(r)
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
	companyID := getIDFromContext(r)
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

// currentCompanyID — JWT'dagi foydalanuvchining kompaniya id'sini qaytaradi.
func (app *application) currentCompanyID(r *http.Request) (int64, error) {
	userID, _ := r.Context().Value(UserKey).(int64)
	user, err := app.store.Users.GetById(r.Context(), &userID)
	if err != nil {
		return 0, err
	}
	return user.CompanyId, nil
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
	var payload store.CompanyBalanceRecord
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	payload.ID = getIDFromContext(r)

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
	id := getIDFromContext(r)

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
	companyID := getIDFromContext(r)
	rows, err := app.store.CompanyBalances.UserActivityByCompany(r.Context(), companyID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, rows); err != nil {
		app.internalServerError(w, r, err)
	}
}
