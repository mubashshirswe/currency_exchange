package main

import (
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

type ExchangePayload struct {
	ID               *int64 `json:"id"`
	ReceivedMoney    int64  `json:"received_money"`
	ReceivedCurrency string `json:"received_currency"`
	SelledMoney      int64  `json:"selled_money"`
	SelledCurrency   string `json:"selled_currency"`
	ProfitAmount     int64  `json:"profit_amount"`
	ProfitCurrency   string `json:"profit_currency"`
	UserId           int64  `json:"user_id"`
	CompanyID        *int64 `json:"company_id"`
	Details          string `json:"details"`
}

func (app *application) CreateExchangeHandler(w http.ResponseWriter, r *http.Request) {
	var payload ExchangePayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if payload.UserId == 0 {
		payload.UserId = t.UserID
	}
	if err := app.authorizeUser(r, t, payload.UserId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	exchange := &store.Exchange{
		ReceivedMoney:    payload.ReceivedMoney,
		ReceivedCurrency: payload.ReceivedCurrency,
		SelledMoney:      payload.SelledMoney,
		SelledCurrency:   payload.SelledCurrency,
		ProfitAmount:     payload.ProfitAmount,
		ProfitCurrency:   payload.ProfitCurrency,
		UserId:           payload.UserId,
		Details:          payload.Details,
	}

	if err := app.service.Exchanges.Create(r.Context(), exchange); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// CreateExchangeV2Handler — exchange yaratadi va KOMPANIYA balansiga ta'sir qiladi.
// Amalni bajargan hodim user_id JWT'dan olinadi. User balanslarga tegmaydi.
func (app *application) CreateExchangeV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload ExchangePayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	userID, _ := r.Context().Value(UserKey).(int64)
	exchange := &store.Exchange{
		ReceivedMoney:    payload.ReceivedMoney,
		ReceivedCurrency: payload.ReceivedCurrency,
		SelledMoney:      payload.SelledMoney,
		SelledCurrency:   payload.SelledCurrency,
		ProfitAmount:     payload.ProfitAmount,
		ProfitCurrency:   payload.ProfitCurrency,
		UserId:           userID,
		Details:          payload.Details,
	}

	if err := app.service.CompanyOps.CreateExchangeV2(r.Context(), exchange); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, exchange); err != nil {
		app.internalServerError(w, r, err)
	}
}

// UpdateExchangeV2Handler — exchange'ni yangilaydi (company balans). user_id JWT'dan.
func (app *application) UpdateExchangeV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload ExchangePayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	userID := t.UserID
	if err := app.authorizeResource(r, t, "exchanges", getIDFromContext(r)); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	exchange := &store.Exchange{
		ID:               getIDFromContext(r),
		ReceivedMoney:    payload.ReceivedMoney,
		ReceivedCurrency: payload.ReceivedCurrency,
		SelledMoney:      payload.SelledMoney,
		SelledCurrency:   payload.SelledCurrency,
		ProfitAmount:     payload.ProfitAmount,
		ProfitCurrency:   payload.ProfitCurrency,
		UserId:           userID,
		Details:          payload.Details,
	}

	if err := app.service.CompanyOps.UpdateExchangeV2(r.Context(), exchange); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, exchange); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteExchangeV2Handler — exchange'ni o'chiradi (company balans).
func (app *application) DeleteExchangeV2Handler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "exchanges", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.CompanyOps.DeleteExchangeV2(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) GetExchangesHandler(w http.ResponseWriter, r *http.Request) {
	var payload FieldRequestPayload
	app.LoadPaginationInfo(r, r.Context())

	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeFilter(r, t, payload.FieldName, payload.FieldValue); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	records, err := app.store.Exchanges.GetByField(r.Context(), t.BusinessID, payload.FieldName, payload.FieldValue, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, records); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) UpdateExchangeHandler(w http.ResponseWriter, r *http.Request) {
	var payload ExchangePayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "exchanges", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	companyID := t.CompanyID
	if payload.CompanyID != nil {
		companyID = *payload.CompanyID
	}
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	if payload.UserId == 0 {
		payload.UserId = t.UserID
	}
	if err := app.authorizeUser(r, t, payload.UserId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	exchange := &store.Exchange{
		ID:               id,
		ReceivedMoney:    payload.ReceivedMoney,
		ReceivedCurrency: payload.ReceivedCurrency,
		SelledMoney:      payload.SelledMoney,
		SelledCurrency:   payload.SelledCurrency,
		ProfitAmount:     payload.ProfitAmount,
		ProfitCurrency:   payload.ProfitCurrency,
		UserId:           payload.UserId,
		Details:          payload.Details,
		CompanyID:        companyID,
	}

	if err := app.service.Exchanges.Update(r.Context(), exchange); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) DeleteExchangeHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "exchanges", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.Exchanges.Delete(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) ArchiveExchangesHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	if err := app.store.Exchanges.Archive(r.Context(), t.CompanyID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "ARCHIVED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) ArchivedExchangesHandler(w http.ResponseWriter, r *http.Request) {
	app.LoadPaginationInfo(r, r.Context())

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	Exchanges, err := app.store.Exchanges.Archived(r.Context(), t.BusinessID, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, Exchanges); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
