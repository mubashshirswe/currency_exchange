package main

import (
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

type BalancePayload struct {
	Balance   int64  `json:"balance"`
	UserId    int64  `json:"user_id"`
	InOutLay  int64  `json:"in_out_lay"`
	Currency  string `json:"currency"`
	OutInLay  int64  `json:"out_in_lay"`
	CompanyId int64  `json:"company_id"`
}

func (app *application) CreateBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var payload BalancePayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeUser(r, t, payload.UserId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	user, err := app.store.Users.GetByIdInBusiness(r.Context(), payload.UserId, t.BusinessID)
	if err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	balance := &store.Balance{
		Balance:   payload.Balance,
		UserId:    payload.UserId,
		InOutLay:  payload.InOutLay,
		OutInLay:  payload.OutInLay,
		Currency:  payload.Currency,
		CompanyId: user.CompanyId,
	}

	if err := app.store.Balances.Create(r.Context(), balance); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, balance); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetBalanceByIdHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "balances", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	balance, err := app.store.Balances.GetById(r.Context(), &id)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, balance); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetBalanceByUserIdHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeUser(r, t, id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	balance, err := app.store.Balances.GetByUserId(r.Context(), &id)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, balance); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetBalanceByCompanyIdHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	balances, err := app.service.Balances.GetByCompanyId(r.Context(), t.BusinessID, id)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, balances); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetAllBalanceHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	balance, err := app.service.Balances.GetAll(r.Context(), t.BusinessID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, balance); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) UpdateBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var payload BalancePayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "balances", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	balance := &store.Balance{
		ID:       id,
		Balance:  payload.Balance,
		UserId:   payload.UserId,
		InOutLay: payload.InOutLay,
		OutInLay: payload.OutInLay,
	}

	if err := app.store.Balances.Update(r.Context(), balance); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, balance); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) DeleteBalanceHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "balances", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.store.Balances.Delete(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
