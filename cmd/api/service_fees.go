package main

import (
	"fmt"
	"net/http"
	"strings"
)

type ServiceFeeSettlePayload struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Details  string `json:"details"`
}

func (app *application) GetTransactionServiceFeesHandler(w http.ResponseWriter, r *http.Request) {
	companyID, err := app.currentCompanyID(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	currency := strings.TrimSpace(r.URL.Query().Get("currency"))
	status := int64(0)
	if s := r.URL.Query().Get("status"); s != "" {
		if v, err := parseInt64(s); err == nil {
			status = v
		}
	}

	app.LoadPaginationInfo(r, r.Context())
	fees, err := app.service.ServiceFees.ListFees(
		r.Context(), companyID, currency, status, app.Pagination,
	)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, fees); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) GetServiceFeeSettlementsHandler(w http.ResponseWriter, r *http.Request) {
	companyID, err := app.currentCompanyID(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	currency := strings.TrimSpace(r.URL.Query().Get("currency"))
	app.LoadPaginationInfo(r, r.Context())

	rows, err := app.service.ServiceFees.ListSettlements(
		r.Context(), companyID, currency, app.Pagination,
	)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.writeResponse(w, http.StatusOK, rows); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) CreateServiceFeeSettlementHandler(w http.ResponseWriter, r *http.Request) {
	var payload ServiceFeeSettlePayload
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

	st, err := app.service.ServiceFees.Settle(
		r.Context(), companyID, userID,
		payload.Amount, payload.Currency, payload.Details,
	)
	if err != nil {
		if strings.Contains(err.Error(), "yetarli emas") || strings.Contains(err.Error(), "MUSBAT") {
			app.badRequestResponse(w, r, err)
		} else {
			app.internalServerError(w, r, err)
		}
		return
	}
	if err := app.writeResponse(w, http.StatusOK, st); err != nil {
		app.internalServerError(w, r, err)
	}
}

func parseInt64(s string) (int64, error) {
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
