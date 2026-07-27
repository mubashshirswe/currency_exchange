package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

type ServiceFeeSettlePayload struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Details  string `json:"details"`
}

func (app *application) GetTransactionServiceFeesHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	currency := strings.TrimSpace(r.URL.Query().Get("currency"))
	status := int64(0)
	if s := r.URL.Query().Get("status"); s != "" {
		if v, err := parseInt64(s); err == nil {
			status = v
		}
	}

	pagination := app.paginationFromRequest(r, r.Context())
	var (
		fees []store.TransactionServiceFee
		err  error
	)
	if t.IsOwner() {
		fees, err = app.service.ServiceFees.ListFeesAll(
			r.Context(), t.BusinessID, currency, status, pagination,
		)
	} else {
		fees, err = app.service.ServiceFees.ListFees(
			r.Context(), t.CompanyID, currency, status, pagination,
		)
	}
	if err != nil {
		if isMissingDBRelation(err) {
			fees = nil
		} else {
			app.internalServerError(w, r, err)
			return
		}
	}
	if err := app.writeResponse(w, http.StatusOK, fees); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) GetServiceFeeSettlementsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	currency := strings.TrimSpace(r.URL.Query().Get("currency"))
	pagination := app.paginationFromRequest(r, r.Context())

	var (
		rows []store.ServiceFeeSettlement
		err  error
	)
	if t.IsOwner() {
		rows, err = app.service.ServiceFees.ListSettlementsAll(
			r.Context(), t.BusinessID, currency, pagination,
		)
	} else {
		rows, err = app.service.ServiceFees.ListSettlements(
			r.Context(), t.CompanyID, currency, pagination,
		)
	}
	if err != nil {
		if isMissingDBRelation(err) {
			rows = nil
		} else {
			app.internalServerError(w, r, err)
			return
		}
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

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	st, err := app.service.ServiceFees.Settle(
		r.Context(), t.CompanyID, t.UserID,
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

func isMissingDBRelation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist")
}
