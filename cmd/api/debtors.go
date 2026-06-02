package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type DebtorPayload struct {
	FullName        string                  `json:"full_name"`
	ReceivedIncomes []types.ReceivedIncomes `json:"received_incomes"`
	DebtedAmount    int64                   `json:"debted_amount"`
	DebtedCurrency  string                  `json:"debted_currency"`
	UserID          int64                   `json:"user_id"`
	Details         string                  `json:"details"`
	Phone           string                  `json:"phone"`
	IsBalanceEffect int                     `json:"is_balance_effect"`
	Type            int                     `json:"type"`
}

func (app *application) CreateDebtorsHandler(w http.ResponseWriter, r *http.Request) {
	var payload DebtorPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	jsonPayload, _ := json.Marshal(payload)
	log.Println("PAYLOAD")
	log.Println(string(jsonPayload))

	debtor := &store.Debts{
		FullName:        payload.FullName,
		ReceivedIncomes: payload.ReceivedIncomes,
		DebtedAmount:    payload.DebtedAmount,
		DebtedCurrency:  payload.DebtedCurrency,
		UserID:          payload.UserID,
		Details:         payload.Details,
		Phone:           payload.Phone,
		IsBalanceEffect: payload.IsBalanceEffect,
		Type:            payload.Type,
	}

	if err := app.service.Debts.Create(r.Context(), debtor); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, debtor); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) CreateDebtorTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var payload *store.Debts
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	jsonPayload, _ := json.Marshal(payload)
	log.Println("PAYLOAD: ")
	log.Println(string(jsonPayload))

	if err := app.service.Debts.Transaction(r.Context(), payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// CreateDebtorsV2Handler — debtor + debt yaratadi va KOMPANIYA balansiga ta'sir qiladi
// (received_incomes). Amalni bajargan hodim user_id JWT'dan olinadi.
func (app *application) CreateDebtorsV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload DebtorPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	userID, _ := r.Context().Value(UserKey).(int64)
	debt := &store.Debts{
		FullName:        payload.FullName,
		ReceivedIncomes: payload.ReceivedIncomes,
		DebtedAmount:    payload.DebtedAmount,
		DebtedCurrency:  payload.DebtedCurrency,
		UserID:          userID,
		Details:         payload.Details,
		Phone:           payload.Phone,
		IsBalanceEffect: payload.IsBalanceEffect,
		Type:            payload.Type,
	}

	if err := app.service.CompanyOps.CreateDebtV2(r.Context(), debt); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, debt); err != nil {
		app.internalServerError(w, r, err)
	}
}

// CreateDebtorTransactionV2Handler — mavjud debtorga qarz tranzaksiyasi; KOMPANIYA balansiga ta'sir.
// Amalni bajargan hodim user_id JWT'dan olinadi.
func (app *application) CreateDebtorTransactionV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload *store.Debts
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	userID, _ := r.Context().Value(UserKey).(int64)
	payload.UserID = userID

	if err := app.service.CompanyOps.DebtTransactionV2(r.Context(), payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
	}
}

// UpdateDebtsV2Handler — debt'ni yangilaydi (company balans). user_id JWT'dan.
func (app *application) UpdateDebtsV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload *store.Debts
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	userID, _ := r.Context().Value(UserKey).(int64)
	debt := &store.Debts{
		ID:              getIDFromContext(r),
		FullName:        payload.FullName,
		ReceivedIncomes: payload.ReceivedIncomes,
		DebtedAmount:    payload.DebtedAmount,
		DebtedCurrency:  payload.DebtedCurrency,
		UserID:          userID,
		Details:         payload.Details,
		Phone:           payload.Phone,
		IsBalanceEffect: payload.IsBalanceEffect,
		Type:            payload.Type,
		DebtorID:        payload.DebtorID,
	}

	if err := app.service.CompanyOps.UpdateDebtV2(r.Context(), debt); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, debt); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteDebtsV2Handler — debt'ni o'chiradi (company balans).
func (app *application) DeleteDebtsV2Handler(w http.ResponseWriter, r *http.Request) {
	id := getIDFromContext(r)

	if err := app.service.CompanyOps.DeleteDebtV2(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) UpdateDebtsHandler(w http.ResponseWriter, r *http.Request) {
	var payload *store.Debts
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	debt := &store.Debts{
		ID:              getIDFromContext(r),
		FullName:        payload.FullName,
		ReceivedIncomes: payload.ReceivedIncomes,
		DebtedAmount:    payload.DebtedAmount,
		DebtedCurrency:  payload.DebtedCurrency,
		UserID:          payload.UserID,
		Details:         payload.Details,
		Phone:           payload.Phone,
		IsBalanceEffect: payload.IsBalanceEffect,
		Type:            payload.Type,
		DebtorID:        payload.DebtorID,
	}

	if err := app.service.Debts.Update(r.Context(), debt); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetDebtorsByCompanyIdHandler(w http.ResponseWriter, r *http.Request) {
	app.LoadPaginationInfo(r, r.Context())
	search := r.URL.Query().Get("search")
	date := r.URL.Query().Get("date")

	var textSeach *string
	if search == "" {
		textSeach = nil
	} else {
		textSeach = &search
	}

	var dateSearch *string
	if date == "" {
		dateSearch = nil
	} else {
		dateSearch = &date
	}

	debtors, err := app.service.Debtors.GetByCompanyId(r.Context(), getIDFromContext(r), textSeach, dateSearch, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, debtors); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetDebtorsTotalBalanceInfo(w http.ResponseWriter, r *http.Request) {
	infos, err := app.store.Debtors.GetByBalanceInfo(r.Context(), getIDFromContext(r))
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, infos); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetDebtsByDebtorIdHandler(w http.ResponseWriter, r *http.Request) {
	app.LoadPaginationInfo(r, r.Context())
	debtors, err := app.store.Debts.GetByDebtorID(r.Context(), getIDFromContext(r), app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, debtors); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetDebtorsByIdHandler(w http.ResponseWriter, r *http.Request) {
	debtors, err := app.store.Debtors.GetById(r.Context(), getIDFromContext(r))
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, debtors); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) DeleteDebtsHandler(w http.ResponseWriter, r *http.Request) {
	id := getIDFromContext(r)

	if err := app.service.Debts.Delete(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) DeleteDebtorsHandler(w http.ResponseWriter, r *http.Request) {
	id := getIDFromContext(r)

	if err := app.store.Debtors.Delete(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
