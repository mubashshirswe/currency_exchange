package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// Tenant konteksti: har bir so'rov aynan bitta business ichida bajariladi.
// Ierarxiya: business => company => users.
const (
	BusinessKey contextkey = "BusinessID"
	CompanyKey  contextkey = "CompanyID"
	RoleKey     contextkey = "Role"
)

var (
	errNoBusiness      = errors.New("foydalanuvchi hech qanday businessga tegishli emas")
	errForeignBusiness = errors.New("resurs boshqa businessga tegishli")
	errForeignCompany  = errors.New("resurs boshqa kompaniyaga tegishli")
	errOwnerOnly       = errors.New("faqat business egasi uchun")
)

// tenant — joriy so'rovning tenant konteksti.
type tenant struct {
	UserID     int64
	BusinessID int64
	CompanyID  int64
	Role       int64
}

// IsOwner — business egasi o'z businessidagi barcha kompaniyalarni ko'radi.
func (t tenant) IsOwner() bool {
	return t.Role == store.ROLE_OWNER
}

// tenantFrom — middleware joylagan tenant konteksti.
func tenantFrom(r *http.Request) tenant {
	ctx := r.Context()
	userID, _ := ctx.Value(UserKey).(int64)
	businessID, _ := ctx.Value(BusinessKey).(int64)
	companyID, _ := ctx.Value(CompanyKey).(int64)
	role, _ := ctx.Value(RoleKey).(int64)

	return tenant{UserID: userID, BusinessID: businessID, CompanyID: companyID, Role: role}
}

// requireTenant — tenant konteksti bo'lmasa 401 yozadi va false qaytaradi.
func (app *application) requireTenant(w http.ResponseWriter, r *http.Request) (tenant, bool) {
	t := tenantFrom(r)
	if t.BusinessID == 0 {
		app.unauthorizedErrorResponse(w, r, errNoBusiness)
		return t, false
	}

	return t, true
}

// requireOwner — faqat business egasiga ruxsat.
func (app *application) requireOwner(w http.ResponseWriter, r *http.Request) (tenant, bool) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return t, false
	}
	if !t.IsOwner() {
		app.forbiddenResponse(w, r, errOwnerOnly)
		return t, false
	}

	return t, true
}

// authorizeCompany — so'ralgan kompaniya joriy business ichidami; oddiy hodim
// esa faqat o'z kompaniyasi bilan ishlaydi.
func (app *application) authorizeCompany(r *http.Request, t tenant, companyID int64) error {
	if companyID == 0 {
		return fmt.Errorf("company_id talab qilinadi")
	}

	if !t.IsOwner() && companyID != t.CompanyID {
		return errForeignCompany
	}

	ok, err := app.store.Companies.BelongsToBusiness(r.Context(), companyID, t.BusinessID)
	if err != nil {
		return err
	}
	if !ok {
		return errForeignBusiness
	}

	return nil
}

// authorizeUser — so'ralgan user joriy business ichidami (hodim uchun — o'z kompaniyasi).
func (app *application) authorizeUser(r *http.Request, t tenant, userID int64) error {
	if userID == 0 {
		return fmt.Errorf("user_id talab qilinadi")
	}
	if userID == t.UserID {
		return nil
	}

	user, err := app.store.Users.GetByIdInBusiness(r.Context(), userID, t.BusinessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errForeignBusiness
		}
		return err
	}

	if !t.IsOwner() && user.CompanyId != t.CompanyID {
		return errForeignCompany
	}

	return nil
}

// authorizeResource — {id} orqali kelgan yozuv joriy businessga tegishlimi.
func (app *application) authorizeResource(r *http.Request, t tenant, table string, id int64) error {
	if id == 0 {
		return fmt.Errorf("id talab qilinadi")
	}

	businessID, err := app.store.BusinessIDOf(r.Context(), table, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errForeignBusiness
		}
		return err
	}

	if businessID != t.BusinessID {
		return errForeignBusiness
	}

	return nil
}

// authorizeFilter — mijoz yuborgan field_name/field_value juftligini tekshiradi:
// kompaniya/user id'lari faqat o'z business (hodim uchun — o'z kompaniyasi) doirasida.
func (app *application) authorizeFilter(r *http.Request, t tenant, fieldName string, fieldValue any) error {
	id := anyToInt64(fieldValue)

	switch fieldName {
	case "company_id", "received_company_id", "delivered_company_id":
		return app.authorizeCompany(r, t, id)
	case "user_id", "received_user_id", "delivered_user_id":
		return app.authorizeUser(r, t, id)
	case "id":
		// Yozuvning o'zi keyingi qatlamda business filtri bilan tekshiriladi.
		return nil
	default:
		return fmt.Errorf("filter maydoni ruxsat etilmagan: %v", fieldName)
	}
}

// handleScopeError — guard xatolarini to'g'ri HTTP status bilan qaytaradi.
func (app *application) handleScopeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errForeignBusiness), errors.Is(err, errForeignCompany), errors.Is(err, errOwnerOnly):
		app.forbiddenResponse(w, r, err)
	case errors.Is(err, sql.ErrNoRows):
		app.notFoundResponse(w, r, err)
	default:
		app.badRequestResponse(w, r, err)
	}
}

func anyToInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case string:
		var parsed int64
		if _, err := fmt.Sscanf(n, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

// authorizeTransactionCompanies — tranzaksiyaning ikkala tomoni ham joriy
// business ichida bo'lishi shart. Hodim uchun kamida bitta tomon o'z kompaniyasi
// bo'lishi kerak (boshqa kompaniyalar orasidagi tranzaksiya yaratolmaydi).
func (app *application) authorizeTransactionCompanies(r *http.Request, t tenant, receivedCompanyID, deliveredCompanyID int64) error {
	for _, companyID := range []int64{receivedCompanyID, deliveredCompanyID} {
		if companyID == 0 {
			continue
		}

		ok, err := app.store.Companies.BelongsToBusiness(r.Context(), companyID, t.BusinessID)
		if err != nil {
			return err
		}
		if !ok {
			return errForeignBusiness
		}
	}

	if t.IsOwner() {
		return nil
	}
	if receivedCompanyID != t.CompanyID && deliveredCompanyID != t.CompanyID {
		return errForeignCompany
	}

	return nil
}
