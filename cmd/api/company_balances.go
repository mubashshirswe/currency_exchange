package main

import (
	"net/http"
)

// GetCompanyBalancesHandler — yangi endpoint: faqat company_balances jadvalidan o'qiydi.
// Eski GET /user/balances/company/{id} o'zgarishsiz qoladi.
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
