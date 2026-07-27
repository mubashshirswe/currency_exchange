package main

import (
	"net/http"
)

// defaultCompanyCurrencies — kompaniya yaratilganda 0 balans bilan ochiladigan
// standart valyutalar (user yaratilishidagi balanslar bilan bir xil).
var defaultCompanyCurrencies = []string{"USD", "SUM"}

// Kompaniya ochish/o'zgartirish/o'chirish bu yerda yo'q: u faqat platforma
// admin API'sida (cmd/api/admin.go, X-Admin-Key) bajariladi. Business egasi
// kompaniya qo'sha olmaydi.

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
