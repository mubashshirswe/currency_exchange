package main

import (
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type FieldRequestPayload struct {
	From       *string `json:"from"`
	To         *string `json:"to"`
	Search     *string `json:"search"`
	FieldName  string  `json:"field_name"`
	FieldValue any     `json:"field_value"`
}

func (app *application) CreateBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	var payload types.BalanceRecordPayload
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
	payload.CompanyID = &t.CompanyID

	if err := app.service.BalanceRecords.PerformBalanceRecord(r.Context(), payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetBalanceRecordsByUserIdHandler(w http.ResponseWriter, r *http.Request) {
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

	records, err := app.store.BalanceRecords.GetByField(r.Context(), t.BusinessID, payload.FieldName, payload.FieldValue, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, records); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetBalanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
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

	records, err := app.store.BalanceRecords.GetByField(r.Context(), t.BusinessID, payload.FieldName, payload.FieldValue, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, records); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) UpdateBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	var payload store.BalanceRecord
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	payload.ID = getIDFromContext(r)
	if err := app.authorizeResource(r, t, "balance_records", payload.ID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.BalanceRecords.UpdateRecord(r.Context(), payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) DeleteBalanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "balance_records", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.BalanceRecords.RollbackBalanceRecord(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) ArchiveBalanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	if err := app.store.BalanceRecords.Archive(r.Context(), t.CompanyID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "ARCHIVED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) ArchivedBalanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	app.LoadPaginationInfo(r, r.Context())
	balanceRecords, err := app.store.BalanceRecords.Archived(r.Context(), t.BusinessID, app.Pagination)

	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, balanceRecords); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
