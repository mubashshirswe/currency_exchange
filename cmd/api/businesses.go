package main

import (
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// BusinessPayload — business ma'lumotini yangilash (platforma admini uchun).
type BusinessPayload struct {
	Name    string `json:"name"`
	Details string `json:"details"`
	Phone   string `json:"phone"`
}

// GetMyBusinessHandler — joriy foydalanuvchining businessi.
func (app *application) GetMyBusinessHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	business, err := app.store.Businesses.GetById(r.Context(), t.BusinessID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, business); err != nil {
		app.internalServerError(w, r, err)
	}
}

// GetBusinessTeamHandler — business jamoasi: kompaniyalar va ulardagi hodimlar.
// Oddiy hodim faqat o'z kompaniyasi jamoasini ko'radi.
func (app *application) GetBusinessTeamHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	companies, err := app.store.Companies.ListByBusiness(r.Context(), t.BusinessID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var users []store.User
	if t.IsOwner() {
		users, err = app.store.Users.ListByBusiness(r.Context(), t.BusinessID)
	} else {
		users, err = app.store.Users.ListByCompany(r.Context(), t.BusinessID, t.CompanyID)
	}
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	usersByCompany := map[int64][]store.User{}
	for _, user := range users {
		usersByCompany[user.CompanyId] = append(usersByCompany[user.CompanyId], user)
	}

	teams := make([]map[string]any, 0, len(companies))
	for _, company := range companies {
		if !t.IsOwner() && company.ID != t.CompanyID {
			continue
		}

		members := usersByCompany[company.ID]
		if members == nil {
			members = []store.User{}
		}

		teams = append(teams, map[string]any{
			"company_id":   company.ID,
			"company_name": company.Name,
			"members":      members,
		})
	}

	if err := app.writeResponse(w, http.StatusOK, teams); err != nil {
		app.internalServerError(w, r, err)
	}
}
