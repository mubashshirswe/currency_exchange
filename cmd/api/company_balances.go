package main

import (
	"net/http"
)

// GetCompanyBalancesHandler — kompaniya balansi (har bir valyuta uchun) user
// balanslaridan jamlanadi. Shu sabab har bir exchange/transaction/debt/balance-record
// avtomatik kompaniya balansiga ta'sir qiladi va drift bo'lmaydi.
// Eski GET /user/balances/company/{id} o'zgarishsiz qoladi.
func (app *application) GetCompanyBalancesHandler(w http.ResponseWriter, r *http.Request) {
	companyID := getIDFromContext(r)
	balances, err := app.store.CompanyBalances.AggregateByCompanyId(r.Context(), companyID)
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

	rows, err := app.store.CompanyBalances.ListRecordsByCompanyAndCurrency(r.Context(), companyID, currency, app.Pagination)
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
	balances, err := app.store.CompanyBalances.AggregateByCompanyId(r.Context(), companyID)
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

	rows, err := app.store.CompanyBalances.ListRecordsByCompanyAndCurrency(r.Context(), companyID, currency, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, rows); err != nil {
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
