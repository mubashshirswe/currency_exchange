package main

import (
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/types"
)

func (app *application) GetMySoftBalancesHandler(w http.ResponseWriter, r *http.Request) {
	companyID, err := app.currentCompanyID(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	balances, err := app.store.SoftBalances.GetByCompanyId(r.Context(), companyID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, balances); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) GetMySoftBalanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	companyID, err := app.currentCompanyID(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	currency := r.URL.Query().Get("currency")
	app.LoadPaginationInfo(r, r.Context())

	rows, err := app.store.SoftBalanceRecords.ListByCompany(r.Context(), companyID, currency, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, rows); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) CreateMySoftBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := app.service.SoftBalanceRecords.PerformSoftBalanceRecord(r.Context(), payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) DeleteMySoftBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := getIDFromContext(r)
	if err := app.service.SoftBalanceRecords.RollbackSoftBalanceRecord(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
	}
}
