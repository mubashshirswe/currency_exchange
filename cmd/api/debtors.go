package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

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

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if payload.UserID == 0 {
		payload.UserID = t.UserID
	}
	if err := app.authorizeUser(r, t, payload.UserID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

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

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if payload.UserID == 0 {
		payload.UserID = t.UserID
	}
	if err := app.authorizeUser(r, t, payload.UserID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	if payload.DebtorID > 0 {
		if err := app.authorizeResource(r, t, "debtors", payload.DebtorID); err != nil {
			app.handleScopeError(w, r, err)
			return
		}
	}

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

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	payload.UserID = t.UserID
	if payload.DebtorID > 0 {
		if err := app.authorizeResource(r, t, "debtors", payload.DebtorID); err != nil {
			app.handleScopeError(w, r, err)
			return
		}
	}

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

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeResource(r, t, "debts", getIDFromContext(r)); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	userID := t.UserID
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
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "debts", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

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

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeResource(r, t, "debts", getIDFromContext(r)); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	if payload.UserID == 0 {
		payload.UserID = t.UserID
	}
	if err := app.authorizeUser(r, t, payload.UserID); err != nil {
		app.handleScopeError(w, r, err)
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

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companyID := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	debtors, err := app.service.Debtors.GetByCompanyId(r.Context(), t.BusinessID, companyID, textSeach, dateSearch, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, debtors); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// SearchDebtorsHandler — GET /debtors/company/{id}/search?q=<ism>&currency=<val>&limit=<n>
// Transaction'da qarz yozishdan oldin ism bo'yicha mavjud qarzdorlarni taklif qiladi.
// Imlo xatolariga chidamli (trigram similarity), natija o'xshashlik bo'yicha saralanadi.
// Bo'sh ro'yxat qaytsa — bunday qarzdor yo'q, transaction yangi qarzdor ochadi.
func (app *application) SearchDebtorsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companyID := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("full_name")
	}
	currency := r.URL.Query().Get("currency")

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	debtors, err := app.service.Debtors.SearchByName(r.Context(), t.BusinessID, companyID, query, currency, limit)
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
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companyID := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	infos, err := app.store.Debtors.GetByBalanceInfo(r.Context(), companyID)
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
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	debtorID := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "debtors", debtorID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	app.LoadPaginationInfo(r, r.Context())
	debtors, err := app.store.Debts.GetByDebtorID(r.Context(), debtorID, app.Pagination)
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
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "debtors", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	debtors, err := app.store.Debtors.GetById(r.Context(), id)
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
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "debts", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

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
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "debtors", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.store.Debtors.Delete(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
