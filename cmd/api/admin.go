package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// Platforma operatori uchun endpointlar (X-Admin-Key). Business va kompaniya
// faqat shu yerdan ochiladi — mijoz ilovasida bunday oqim yo'q, business egasi
// ham kompaniya qo'sha olmaydi.

// AdminBusinessPayload — yangi tenant: business + birinchi kompaniya + egasi.
type AdminBusinessPayload struct {
	Name        string `json:"name"`
	Details     string `json:"details"`
	Phone       string `json:"phone"`
	CompanyName string `json:"company_name"`
	Owner       struct {
		Username string `json:"username"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	} `json:"owner"`
}

// AdminCompanyPayload — mavjud business ichida yangi kompaniya (jamoa).
type AdminCompanyPayload struct {
	BusinessID int64  `json:"business_id"`
	Name       string `json:"name"`
	Details    string `json:"details"`
	Password   string `json:"password"`
}

// AdminCreateBusinessHandler — business, uning birinchi kompaniyasi va egasi
// bitta DB tranzaksiyasida yaratiladi, shuning uchun yarim ochilgan tenant
// qolmaydi.
func (app *application) AdminCreateBusinessHandler(w http.ResponseWriter, r *http.Request) {
	var payload AdminBusinessPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.CompanyName = strings.TrimSpace(payload.CompanyName)
	payload.Owner.Username = strings.TrimSpace(payload.Owner.Username)
	payload.Owner.Phone = strings.TrimSpace(payload.Owner.Phone)

	if payload.Name == "" {
		app.badRequestResponse(w, r, fmt.Errorf("business nomi (name) talab qilinadi"))
		return
	}
	if payload.Owner.Phone == "" || payload.Owner.Password == "" {
		app.badRequestResponse(w, r, fmt.Errorf("owner.phone va owner.password talab qilinadi"))
		return
	}
	if payload.CompanyName == "" {
		payload.CompanyName = payload.Name
	}
	if payload.Owner.Username == "" {
		payload.Owner.Username = payload.Name
	}

	tx, err := app.store.BeginTx(r.Context())
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	defer tx.Rollback()

	business := &store.Business{
		Name:    payload.Name,
		Details: payload.Details,
		Phone:   payload.Phone,
		Status:  store.BUSINESS_ACTIVE,
	}
	if err := store.NewBusinessStorage(tx).Create(r.Context(), business); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	company := &store.Company{
		Name:       payload.CompanyName,
		Details:    payload.Details,
		BusinessID: business.ID,
	}
	if err := app.createCompanyTx(r, tx, company); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	owner := &store.User{
		Username:   payload.Owner.Username,
		Phone:      payload.Owner.Phone,
		Password:   payload.Owner.Password,
		Role:       store.ROLE_OWNER,
		CompanyId:  company.ID,
		BusinessId: business.ID,
	}
	if err := store.NewUserStorage(tx).Create(r.Context(), owner); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	balances := store.NewBalanceStorage(tx)
	for _, currency := range defaultCompanyCurrencies {
		if err := balances.Create(r.Context(), &store.Balance{
			UserId:    owner.ID,
			CompanyId: company.ID,
			Currency:  currency,
		}); err != nil {
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, map[string]any{
		"business": business,
		"company":  company,
		"user":     owner,
	}); err != nil {
		app.internalServerError(w, r, err)
	}
}

// AdminCreateCompanyHandler — mavjud business ichida yangi kompaniya.
func (app *application) AdminCreateCompanyHandler(w http.ResponseWriter, r *http.Request) {
	var payload AdminCompanyPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if payload.BusinessID <= 0 {
		app.badRequestResponse(w, r, fmt.Errorf("business_id talab qilinadi"))
		return
	}
	if payload.Name == "" {
		app.badRequestResponse(w, r, fmt.Errorf("kompaniya nomi (name) talab qilinadi"))
		return
	}

	// Business mavjudligini tekshiramiz: noto'g'ri id bilan yetim kompaniya
	// yaratilmasin.
	if _, err := app.store.Businesses.GetById(r.Context(), payload.BusinessID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	tx, err := app.store.BeginTx(r.Context())
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	defer tx.Rollback()

	company := &store.Company{
		Name:       payload.Name,
		Details:    payload.Details,
		Password:   payload.Password,
		BusinessID: payload.BusinessID,
	}
	if err := app.createCompanyTx(r, tx, company); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, company); err != nil {
		app.internalServerError(w, r, err)
	}
}

// AdminUpdateCompanyHandler — kompaniya ma'lumotini yangilash.
func (app *application) AdminUpdateCompanyHandler(w http.ResponseWriter, r *http.Request) {
	var payload AdminCompanyPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	id := getIDFromContext(r)
	current, err := app.store.Companies.GetById(r.Context(), &id)
	if err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	company := &store.Company{
		ID:         id,
		Name:       strings.TrimSpace(payload.Name),
		Details:    payload.Details,
		Password:   payload.Password,
		BusinessID: current.BusinessID,
	}
	if company.Name == "" {
		company.Name = current.Name
	}

	if err := app.store.Companies.Update(r.Context(), company); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, company); err != nil {
		app.internalServerError(w, r, err)
	}
}

// AdminDeleteCompanyHandler — kompaniyani o'chirish.
func (app *application) AdminDeleteCompanyHandler(w http.ResponseWriter, r *http.Request) {
	id := getIDFromContext(r)

	current, err := app.store.Companies.GetById(r.Context(), &id)
	if err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.store.Companies.Delete(r.Context(), &id, current.BusinessID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
	}
}

// AdminUpdateBusinessHandler — business ma'lumotini yangilash.
func (app *application) AdminUpdateBusinessHandler(w http.ResponseWriter, r *http.Request) {
	var payload BusinessPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	business := &store.Business{
		ID:      getIDFromContext(r),
		Name:    strings.TrimSpace(payload.Name),
		Details: payload.Details,
		Phone:   payload.Phone,
	}

	if err := app.store.Businesses.Update(r.Context(), business); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, business); err != nil {
		app.internalServerError(w, r, err)
	}
}

// AdminGetBusinessHandler — business ma'lumoti (tekshirish uchun).
func (app *application) AdminGetBusinessHandler(w http.ResponseWriter, r *http.Request) {
	business, err := app.store.Businesses.GetById(r.Context(), getIDFromContext(r))
	if err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, business); err != nil {
		app.internalServerError(w, r, err)
	}
}

// createCompanyTx — kompaniya va uning standart balans qatorlarini bitta
// tranzaksiyada yaratadi.
func (app *application) createCompanyTx(r *http.Request, tx store.DBTX, company *store.Company) error {
	if err := store.NewCompanyStorage(tx).Create(r.Context(), company); err != nil {
		return err
	}
	if err := store.NewCompanyBalanceStorage(tx).EnsureDefaults(r.Context(), company.ID, defaultCompanyCurrencies); err != nil {
		return err
	}

	return store.NewSoftBalanceStorage(tx).EnsureDefaults(r.Context(), company.ID, defaultCompanyCurrencies)
}
