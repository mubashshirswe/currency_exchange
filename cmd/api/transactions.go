package main

import (
	"database/sql"
	"fmt"
	"net/http"

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
}

func (app *application) CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var payload TransactionPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
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

	userID, _ := r.Context().Value(UserKey).(int64)
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

	if err := app.service.CompanyOps.PerformTransactionV2(r.Context(), transaction, userID); err != nil {
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

	userID, _ := r.Context().Value(UserKey).(int64)
	if err := app.service.CompanyOps.CompleteTransactionV2(r.Context(), payload, userID); err != nil {
		if err == sql.ErrNoRows {
			app.badRequestResponse(w, r, fmt.Errorf("BUYURTMA ALLAQACHON YAKUNLANGAN"))
		} else {
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "SUCCESS"); err != nil {
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

	userID, _ := r.Context().Value(UserKey).(int64)
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

	if err := app.service.CompanyOps.UpdateTransactionV2(r.Context(), transaction, userID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, transaction); err != nil {
		app.internalServerError(w, r, err)
	}
}

// DeleteTransactionV2Handler — transaction'ni o'chiradi (company balans).
func (app *application) DeleteTransactionV2Handler(w http.ResponseWriter, r *http.Request) {
	id := getIDFromContext(r)

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

	transactions, err := app.service.Transactions.GetByField(r.Context(), search, payload.FieldName, payload.FieldValue, app.Pagination)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	user, err := app.currentUser(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if !app.isAdminUser(user) {
		transactions = maskTransactionListServiceFees(transactions, user.CompanyId)
	}

	if err := app.writeResponse(w, http.StatusOK, transactions); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetTransactionsCompanyIdHandler(w http.ResponseWriter, r *http.Request) {
	app.LoadPaginationInfo(r, r.Context())
	transactions, err := app.service.Transactions.GetByCompanyId(r.Context(), getIDFromContext(r), app.Pagination)
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
	user, err := app.currentUser(r)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	transactions, err := app.service.Transactions.GetInfos(
		r.Context(), chi.URLParam(r, "date"),
	)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if !app.isAdminUser(user) {
		for i := range transactions {
			if transactions[i].CompanyID != user.CompanyId {
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

	transactions, err := app.store.Transactions.GetByFieldAndDate(r.Context(), payload.FieldName, *payload.From, *payload.To, payload.FieldValue, app.Pagination)
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
	userId := r.Context().Value(UserKey).(int64)
	println("userId", userId)

	user, err := app.store.Users.GetById(r.Context(), &userId)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.store.Transactions.Archive(r.Context(), user.CompanyId); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "ARCHIVED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) ArchivedTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	app.LoadPaginationInfo(r, r.Context())
	transactions, err := app.service.Transactions.Archived(r.Context(), app.Pagination)
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
	id := getIDFromContext(r)
	if err := app.service.Transactions.Delete(r.Context(), &id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func maskTransactionListServiceFees(
	items []map[string]interface{},
	viewerCompanyID int64,
) []map[string]interface{} {
	for i, item := range items {
		if transactionServiceFeeVisibleToCompany(item, viewerCompanyID) {
			continue
		}
		item["service_fee"] = ""
		item["service_fee_amount"] = int64(0)
		item["service_fee_currency"] = ""
		item["service_fee_details"] = ""
		items[i] = item
	}
	return items
}

func transactionServiceFeeVisibleToCompany(
	item map[string]interface{},
	companyID int64,
) bool {
	deliveredUserID := mapInt64Ptr(item["delivered_user_id"])
	receivedCompanyID := mapInt64(item["received_company_id"])
	deliveredCompanyID := mapInt64(item["delivered_company_id"])

	if deliveredUserID != nil && *deliveredUserID > 0 {
		return deliveredCompanyID == companyID
	}
	return receivedCompanyID == companyID
}

func mapInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func mapInt64Ptr(v any) *int64 {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case *int64:
		return n
	case int64:
		return &n
	case int:
		i := int64(n)
		return &i
	case float64:
		i := int64(n)
		return &i
	default:
		return nil
	}
}
