package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// defaultCompanyCurrencies — kompaniya yaratilganda 0 balans bilan ochiladigan
// standart valyutalar (user yaratilishidagi balanslar bilan bir xil).
var defaultCompanyCurrencies = []string{"USD", "SUM"}

// CompanyPayload — business egasi yuboradigan kompaniya ma'lumoti. business_id
// mijozdan olinmaydi: u doim tokendagi businessdan keladi.
type CompanyPayload struct {
	Name     string `json:"name"`
	Details  string `json:"details"`
	Password string `json:"password"`
}

// Kompaniya CRUD business egasi uchun ochiq (business ichida), platforma admin
// API'si (cmd/api/admin.go, X-Admin-Key) esa istalgan business bilan ishlaydi.

// GetAllCompanyHandler — faqat joriy business kompaniyalari.
func (app *application) GetAllCompanyHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companies, err := app.store.Companies.ListByBusiness(r.Context(), t.BusinessID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, companies); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GetCompanyByIdHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	company, err := app.store.Companies.GetByIdInBusiness(r.Context(), id, t.BusinessID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, company); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// CreateCompanyHandler — business egasi o'z businessida yangi kompaniya
// (jamoa) ochadi. Kompaniya va uning standart balans qatorlari bitta
// tranzaksiyada yaratiladi, shuning uchun yarim ochilgan kompaniya qolmaydi.
func (app *application) CreateCompanyHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireOwner(w, r)
	if !ok {
		return
	}

	var payload CompanyPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		app.badRequestResponse(w, r, fmt.Errorf("KOMPANIYA NOMI TALAB QILINADI"))
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
		BusinessID: t.BusinessID,
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

// UpdateCompanyHandler — kompaniya ma'lumotini yangilash (faqat ega, faqat o'z
// businessi ichida). Bo'sh yuborilgan maydon eskisicha qoladi.
func (app *application) UpdateCompanyHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireOwner(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	var payload CompanyPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	current, err := app.store.Companies.GetByIdInBusiness(r.Context(), id, t.BusinessID)
	if err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	company := mergeCompanyPayload(current, payload)
	if err := app.store.Companies.Update(r.Context(), company); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, company); err != nil {
		app.internalServerError(w, r, err)
	}
}

// mergeCompanyPayload — mijoz yubormagan maydonlar mavjud qiymatida qoladi:
// parol yuborilmasa eskisi o'chib ketmasin.
func mergeCompanyPayload(current *store.Company, payload CompanyPayload) *store.Company {
	company := &store.Company{
		ID:         current.ID,
		Name:       strings.TrimSpace(payload.Name),
		Details:    payload.Details,
		Password:   payload.Password,
		BusinessID: current.BusinessID,
		CreatedAt:  current.CreatedAt,
	}

	if company.Name == "" {
		company.Name = current.Name
	}
	if company.Password == "" {
		company.Password = current.Password
	}

	return company
}

// DeleteCompanyHandler — bo'sh kompaniyani o'chirish (faqat ega). Kompaniya
// o'chirilganda uning balanslari va hisob yozuvlari kaskad bilan yo'qoladi,
// shuning uchun ishlatilayotgan kompaniya o'chirilmaydi.
func (app *application) DeleteCompanyHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireOwner(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeCompany(r, t, id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if id == t.CompanyID {
		app.badRequestResponse(w, r, fmt.Errorf("O'ZINGIZ TURGAN KOMPANIYANI O'CHIRIB BO'LMAYDI"))
		return
	}

	inUse, err := app.store.Companies.InUse(r.Context(), id)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if inUse {
		app.badRequestResponse(w, r, fmt.Errorf("KOMPANIYADA HODIM YOKI OPERATSIYA BOR: AVVAL ULARNI KO'CHIRING"))
		return
	}

	if err := app.store.Companies.Delete(r.Context(), &id, t.BusinessID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
	}
}
