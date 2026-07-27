package main

import (
	"fmt"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

type UserPayload struct {
	Username  string  `json:"username"`
	Phone     string  `json:"phone"`
	Role      int64   `json:"role"`
	Password  string  `json:"password"`
	CompanyId int64   `json:"company_id"`
	Avatar    *string `json:"avatar"`
}

type LoginUserPayload struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
	// BusinessID — bir xil telefon raqami bir nechta businessda uchrasa,
	// qaysi biriga kirish kerakligini aniqlashtiradi.
	BusinessID int64 `json:"business_id"`
}

type RefreshTokenPayload struct {
	RefreshToken string `json:"refresh_token"`
}

// CreateUserHandler — joriy business ichida yangi hodim ochadi (faqat business egasi).
// company_id yuborilmasa, egasining kompaniyasi olinadi; yuborilsa u shu business
// ichida bo'lishi shart.
func (app *application) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireOwner(w, r)
	if !ok {
		return
	}

	var payload UserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	companyID := payload.CompanyId
	if companyID == 0 {
		companyID = t.CompanyID
	}
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	role := payload.Role
	if role == 0 {
		role = store.ROLE_STAFF
	}

	user := &store.User{
		Username:   payload.Username,
		Phone:      payload.Phone,
		Role:       role,
		Password:   payload.Password,
		CompanyId:  companyID,
		BusinessId: t.BusinessID,
	}

	if err := app.store.Users.Create(r.Context(), user); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	for _, currency := range defaultCompanyCurrencies {
		if err := app.store.Balances.Create(r.Context(), &store.Balance{
			Balance:   0,
			UserId:    user.ID,
			CompanyId: user.CompanyId,
			InOutLay:  0,
			OutInLay:  0,
			Currency:  currency,
		}); err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
	}

	if err := app.writeResponse(w, http.StatusOK, user); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) LoginUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload LoginUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	candidates, err := app.store.Users.LoginCandidates(r.Context(), payload.Phone, payload.Password)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	user, err := pickLoginUser(candidates, payload.BusinessID)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	app.respondWithTokens(w, r, user)
}

// pickLoginUser — telefon business ichida unikal, businesslar bo'ylab takrorlanishi
// mumkin. Bir nechta mos user topilsa, mijoz business_id yuborishi kerak.
func pickLoginUser(candidates []store.User, businessID int64) (*store.User, error) {
	if businessID > 0 {
		for i := range candidates {
			if candidates[i].BusinessId == businessID {
				return &candidates[i], nil
			}
		}
		return nil, fmt.Errorf("LOGIN YOKI PAROL XATO")
	}

	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("LOGIN YOKI PAROL XATO")
	case 1:
		return &candidates[0], nil
	default:
		return nil, fmt.Errorf("BU TELEFON BIR NECHTA BUSINESSDA MAVJUD: business_id yuboring")
	}
}

// RefreshTokenHandler exchanges a valid refresh token for a fresh access token
// and a rotated refresh token, so an active client never has to log in again
// while its refresh token is alive.
func (app *application) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// The token may arrive in the body or in the Authorization header, so an
	// absent or unreadable body is not an error by itself.
	var payload RefreshTokenPayload
	_ = readJSON(w, r, &payload)

	refreshToken := payload.RefreshToken
	if refreshToken == "" {
		refreshToken = GetTokenFromRequest(r)
	}

	userID, err := userIDFromToken(refreshToken, refreshTokenType)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	// Confirm the user still exists before minting new tokens.
	user, err := app.GetUser(r.Context(), userID)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	app.respondWithTokens(w, r, user)
}

// respondWithTokens issues a token pair for the user and writes the standard
// authentication payload shared by login and refresh.
func (app *application) respondWithTokens(w http.ResponseWriter, r *http.Request, user *store.User) {
	accessToken, refreshToken, err := issueUserTokenPair(user)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	cfg := tokens()

	if err := app.writeResponse(w, http.StatusOK, map[string]any{
		"token":              accessToken,
		"refresh_token":      refreshToken,
		"token_type":         "Bearer",
		"expires_in":         int(cfg.accessTTL.Seconds()),
		"refresh_expires_in": int(cfg.refreshTTL.Seconds()),
		"user":               user,
	}); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// GetAllUserHandler — joriy business hodimlari. Oddiy hodim faqat o'z
// kompaniyasidagilarni ko'radi.
func (app *application) GetAllUserHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	var (
		users []store.User
		err   error
	)
	if t.IsOwner() {
		users, err = app.store.Users.ListByBusiness(r.Context(), t.BusinessID)
	} else {
		users, err = app.store.Users.ListByCompany(r.Context(), t.BusinessID, t.CompanyID)
	}
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, users); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	// Hodim faqat o'zini yangilay oladi; boshqalarni — business egasi.
	if id != t.UserID && !t.IsOwner() {
		app.forbiddenResponse(w, r, errOwnerOnly)
		return
	}
	if err := app.authorizeUser(r, t, id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	var payload UserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	current, err := app.store.Users.GetByIdInBusiness(r.Context(), id, t.BusinessID)
	if err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	companyID := payload.CompanyId
	if companyID == 0 {
		companyID = current.CompanyId
	}
	if err := app.authorizeCompany(r, t, companyID); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	role := payload.Role
	if role == 0 {
		role = current.Role
	}
	// Rolni faqat business egasi o'zgartira oladi.
	if role != current.Role && !t.IsOwner() {
		app.forbiddenResponse(w, r, errOwnerOnly)
		return
	}

	user := &store.User{
		ID:         id,
		Username:   payload.Username,
		Role:       role,
		Password:   payload.Password,
		Avatar:     payload.Avatar,
		CompanyId:  companyID,
		BusinessId: t.BusinessID,
	}

	if err := app.store.Users.Update(r.Context(), user); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, user); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireOwner(w, r)
	if !ok {
		return
	}

	id := getIDFromContext(r)
	if err := app.authorizeUser(r, t, id); err != nil {
		app.handleScopeError(w, r, err)
		return
	}

	if err := app.store.Users.Delete(r.Context(), &id, t.BusinessID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
