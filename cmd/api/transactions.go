package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type TransactionPayload struct {
	ServiceFeeAmount   int64                     `json:"service_fee_amount"`
	ServiceFeeCurrency string                    `json:"service_fee_currency"`
	ServiceFeeDetails  string                    `json:"service_fee_details"`
	ReceivedIncomes    []types.ReceivedIncomes   `json:"received_incomes"`
	DeliveredOutcomes  []types.DeliveredOutcomes `json:"delivered_outcomes"`
	ReceivedCompanyId  int64                     `json:"received_company_id"`
	DeliveredCompanyId int64                     `json:"delivered_company_id"`
	ReceivedUserId     int64                     `json:"received_user_id"`
	DeliveredUserId    *int64                    `json:"delivered_user_id"`
	Phone              string                    `json:"phone"`
	Details            string                    `json:"details"`
	Type               int64                     `json:"type"`

	// Qarzdorlik (debt) maydonlari — yuborilsa, transaction bilan birga debt bo'limiga
	// ham yozuv tushadi. debt_phone bo'sh bo'lsa, transaction phone ishlatiladi.
	DebtorID     int64  `json:"debtor_id"`
	DebtFullName string `json:"full_name"`
	DebtPhone    string `json:"debt_phone"`
	DebtDetails  string `json:"debt_details"`
	DebtType     int    `json:"debt_type"`
}

// debtInput — transaction payload'idagi qarzdorlik maydonlarini ajratib beradi.
func (p TransactionPayload) debtInput() types.TransactionDebtInput {
	return types.TransactionDebtInput{
		DebtorID: p.DebtorID,
		FullName: p.DebtFullName,
		Phone:    p.DebtPhone,
		Details:  p.DebtDetails,
		Type:     p.DebtType,
	}
}

func (app *application) CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var payload TransactionPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeTransactionCompanies(r, t, payload.ReceivedCompanyId, payload.DeliveredCompanyId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	if payload.ReceivedUserId == 0 {
		payload.ReceivedUserId = t.UserID
	}
	if err := app.authorizeUser(r, t, payload.ReceivedUserId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	if payload.DeliveredUserId != nil {
		if err := app.authorizeUser(r, t, *payload.DeliveredUserId); err != nil {
			app.handleScopeError(w, r, err)
			return
		}
	}

	transaction := &store.Transaction{
		ServiceFeeAmount:   payload.ServiceFeeAmount,
		ServiceFeeCurrency: payload.ServiceFeeCurrency,
		ServiceFeeDetails:  payload.ServiceFeeDetails,
		ReceivedIncomes:    payload.ReceivedIncomes,
		DeliveredOutcomes:  payload.DeliveredOutcomes,
		ReceivedCompanyId:  payload.ReceivedCompanyId,
		DeliveredCompanyId: payload.DeliveredCompanyId,
		ReceivedUserId:     payload.ReceivedUserId,
		DeliveredUserId:    payload.DeliveredUserId,
		Phone:              payload.Phone,
		Details:            payload.Details,
		Type:               payload.Type,
		Status:             1,
	}

	if err := app.service.Transactions.PerformTransaction(r.Context(), transaction); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, ""); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// CreateTransactionV2Handler — transaction yaratadi va KOMPANIYA balansiga ta'sir qiladi
// (received_incomes). Amalni bajargan hodim user_id JWT'dan olinadi.
func (app *application) CreateTransactionV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload TransactionPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeTransactionCompanies(r, t, payload.ReceivedCompanyId, payload.DeliveredCompanyId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	userID := t.UserID
	transaction := &store.Transaction{
		ServiceFeeAmount:   payload.ServiceFeeAmount,
		ServiceFeeCurrency: payload.ServiceFeeCurrency,
		ServiceFeeDetails:  payload.ServiceFeeDetails,
		ReceivedIncomes:    payload.ReceivedIncomes,
		DeliveredOutcomes:  payload.DeliveredOutcomes,
		ReceivedCompanyId:  payload.ReceivedCompanyId,
		DeliveredCompanyId: payload.DeliveredCompanyId,
		ReceivedUserId:     payload.ReceivedUserId,
		DeliveredUserId:    payload.DeliveredUserId,
		Phone:              payload.Phone,
		Details:            payload.Details,
		Type:               payload.Type,
		Status:             1,
	}

	if err := app.service.CompanyOps.PerformTransactionV2(r.Context(), transaction, userID, payload.debtInput()); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, ""); err != nil {
		app.internalServerError(w, r, err)
	}
}

// CompleteTransactionV2Handler — transaction yakunlaydi va KOMPANIYA balansiga ta'sir qiladi
// (delivered_outcomes). Amalni bajargan hodim user_id JWT'dan olinadi.
func (app *application) CompleteTransactionV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload types.TransactionComplete
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeResource(r, t, "transactions", payload.TransactionID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	userID := t.UserID
	if err := app.service.CompanyOps.CompleteTransactionV2(r.Context(), payload, userID, t.BusinessID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			app.badRequestResponse(w, r, fmt.Errorf("BUYURTMA ALLAQACHON YAKUNLANGAN"))
		case strings.Contains(err.Error(), types.TRANSACTION_NOT_ACCEPTED):
			app.badRequestResponse(w, r, fmt.Errorf(types.TRANSACTION_NOT_ACCEPTED))
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "SUCCESS"); err != nil {
		app.internalServerError(w, r, err)
	}
}

// AcceptTransactionV2Handler — 3 bosqichli oqimning o'rta bosqichi: buyurtmani qabul qilish.
// BALANSGA TA'SIR QILMAYDI. Sozlamada oqim oddiy bo'lsa 400 qaytadi.
func (app *application) AcceptTransactionV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload types.TransactionAccept
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeResource(r, t, "transactions", payload.TransactionID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.CompanyOps.AcceptTransactionV2(
		r.Context(), payload.TransactionID, t.UserID, t.BusinessID,
	); err != nil {
		switch {
		case err == sql.ErrNoRows:
			app.badRequestResponse(w, r, fmt.Errorf("BUYURTMA ALLAQACHON YAKUNLANGAN"))
		case strings.Contains(err.Error(), types.TRANSACTION_ACCEPT_DISABLED),
			strings.Contains(err.Error(), types.TRANSACTION_ALREADY_ACCEPTED):
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "ACCEPTED"); err != nil {
		app.internalServerError(w, r, err)
	}
}

// UpdateTransactionV2Handler — transaction'ni yangilaydi (company balans). user_id JWT'dan.
func (app *application) UpdateTransactionV2Handler(w http.ResponseWriter, r *http.Request) {
	var payload TransactionPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeResource(r, t, "transactions", getIDFromContext(r)); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	if err := app.authorizeTransactionCompanies(r, t, payload.ReceivedCompanyId, payload.DeliveredCompanyId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	userID := t.UserID
	transaction := &store.Transaction{
		ID:                 getIDFromContext(r),
		ServiceFeeAmount:   payload.ServiceFeeAmount,
		ServiceFeeCurrency: payload.ServiceFeeCurrency,
		ServiceFeeDetails:  payload.ServiceFeeDetails,
		ReceivedIncomes:    payload.ReceivedIncomes,
		DeliveredOutcomes:  payload.DeliveredOutcomes,
		ReceivedCompanyId:  payload.ReceivedCompanyId,
		DeliveredCompanyId: payload.DeliveredCompanyId,
		ReceivedUserId:     payload.ReceivedUserId,
		DeliveredUserId:    payload.DeliveredUserId,
		Phone:              payload.Phone,
		Details:            payload.Details,
		Type:               payload.Type,
		Status:             1,
	}

	if err := app.service.CompanyOps.UpdateTransactionV2(r.Context(), transaction, userID, payload.debtInput()); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, transaction); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteTransactionV2Handler — transaction'ni o'chiradi (company balans).
func (app *application) DeleteTransactionV2Handler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "transactions", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.CompanyOps.DeleteTransactionV2(r.Context(), id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) UpdateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var payload TransactionPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeResource(r, t, "transactions", getIDFromContext(r)); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	if err := app.authorizeTransactionCompanies(r, t, payload.ReceivedCompanyId, payload.DeliveredCompanyId); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	transaction := &store.Transaction{
		ID:                 getIDFromContext(r),
		ServiceFeeAmount:   payload.ServiceFeeAmount,
		ServiceFeeCurrency: payload.ServiceFeeCurrency,
		ServiceFeeDetails:  payload.ServiceFeeDetails,
		ReceivedIncomes:    payload.ReceivedIncomes,
		DeliveredOutcomes:  payload.DeliveredOutcomes,
		ReceivedCompanyId:  payload.ReceivedCompanyId,
		DeliveredCompanyId: payload.DeliveredCompanyId,
		ReceivedUserId:     payload.ReceivedUserId,
		DeliveredUserId:    payload.DeliveredUserId,
		Phone:              payload.Phone,
		Details:            payload.Details,
		Type:               payload.Type,
		Status:             1,
	}

	if err := app.service.Transactions.Update(r.Context(), transaction); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, transaction); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) CompleteTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var payload types.TransactionComplete
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	if err := app.authorizeResource(r, t, "transactions", payload.TransactionID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}
	// v1 oqimi eski user balanslari bilan ishlaydi, lekin sozlamadagi bosqichlar
	// unga ham tegishli: 3 bosqichli oqimda avval qabul qilish shart.
	if err := app.requireAcceptedWhenThreeStage(r, t.BusinessID, payload.TransactionID); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.service.Transactions.CompleteTransaction(r.Context(), payload); err != nil {
		if err == sql.ErrNoRows {
			app.badRequestResponse(w, r, fmt.Errorf("BUYURTMA ALLAQACHON YAKUNLANGAN"))
		} else {
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "SUCCESS"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetTransactionsByFieldHandler(w http.ResponseWriter, r *http.Request) {
	app.LoadPaginationInfo(r, r.Context())
	src := r.URL.Query().Get("search")
	var search *string
	if src != "" && src != "null" {
		search = &src
	}

	var payload FieldRequestPayload
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

	transactions, err := app.service.Transactions.GetByField(r.Context(), t.BusinessID, search, payload.FieldName, payload.FieldValue, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Xizmat haqi tranzaksiyaning ikkala tomoniga ham ko'rinadi (kim olgani
	// `service_fee_company` da), shuning uchun maskalash yo'q.
	if err := app.writeResponse(w, http.StatusOK, transactions); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetTransactionsCompanyIdHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companyID := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	app.LoadPaginationInfo(r, r.Context())
	transactions, err := app.service.Transactions.GetByCompanyId(r.Context(), t.BusinessID, companyID, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, transactions); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetInfosByCompanyIdHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	transactions, err := app.service.Transactions.GetInfos(
		r.Context(), t.BusinessID, chi.URLParam(r, "date"),
	)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if !t.IsOwner() {
		for i := range transactions {
			if transactions[i].CompanyID != t.CompanyID {
				transactions[i].ServiceFeeAmount = 0
				transactions[i].ServiceFeeRemaining = 0
			}
		}
	}

	if err := app.writeResponse(w, http.StatusOK, transactions); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetTransactionsByFieldAndDateHandler(w http.ResponseWriter, r *http.Request) {
	app.LoadPaginationInfo(r, r.Context())
	var payload FieldRequestPayload
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

	transactions, err := app.store.Transactions.GetByFieldAndDate(r.Context(), t.BusinessID, payload.FieldName, *payload.From, *payload.To, payload.FieldValue, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, transactions); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) ArchiveTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	if err := app.store.Transactions.Archive(r.Context(), t.CompanyID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "ARCHIVED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) ArchivedTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	app.LoadPaginationInfo(r, r.Context())
	transactions, err := app.service.Transactions.Archived(r.Context(), t.BusinessID, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, transactions); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) DeleteTransactionHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeResource(r, t, "transactions", id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.service.Transactions.Delete(r.Context(), &id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
