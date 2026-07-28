package main

import (
	"context"
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
	// qaysi biriga kirish kerakligini aniqlashtiradi. Yuborilmasa birinchi
	// business tanlanadi va qolganlari javobdagi ro'yxatda qaytadi.
	BusinessID int64 `json:"business_id"`
}

type RefreshTokenPayload struct {
	RefreshToken string `json:"refresh_token"`
}

type SwitchBusinessPayload struct {
	BusinessID int64 `json:"business_id"`
}

// businessMembership — bitta telefon egasining bitta businessdagi a'zoligi.
// Mijoz shu ro'yxatdan business tanlab, parolsiz almashadi.
type businessMembership struct {
	BusinessID     int64  `json:"business_id"`
	BusinessName   string `json:"business_name"`
	BusinessStatus int64  `json:"business_status"`
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	CompanyID      int64  `json:"company_id"`
	Role           int64  `json:"role"`
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

	app.respondWithTokens(w, r, user, candidates)
}

// SwitchBusinessHandler — joriy sessiyani boshqa businessga o'tkazadi: parol
// qayta so'ralmaydi, chunki almashish faqat bir xil telefon + bir xil parolga
// ega qatorlar orasida mumkin (login paytidagi tekshiruv bilan bir xil shart,
// lekin DB'dan jonli o'qiladi — parol o'zgarsa eski token bilan o'tib bo'lmaydi).
func (app *application) SwitchBusinessHandler(w http.ResponseWriter, r *http.Request) {
	current, err := app.currentUser(r)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	var payload SwitchBusinessPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if payload.BusinessID <= 0 {
		app.badRequestResponse(w, r, fmt.Errorf("business_id majburiy"))
		return
	}

	candidates, err := app.switchCandidates(r.Context(), current)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if payload.BusinessID == current.BusinessId {
		// Xuddi shu business — yangi token juftligi berilaveradi, xato emas.
		app.respondWithTokens(w, r, current, candidates)
		return
	}

	target, err := pickLoginUser(candidates, payload.BusinessID)
	if err != nil {
		app.forbiddenResponse(w, r, fmt.Errorf("BU BUSINESSGA RUXSAT YO'Q"))
		return
	}

	app.respondWithTokens(w, r, target, candidates)
}

// ListBusinessesHandler — joriy foydalanuvchi parolsiz o'ta oladigan businesslar.
func (app *application) ListMyBusinessesHandler(w http.ResponseWriter, r *http.Request) {
	current, err := app.currentUser(r)
	if err != nil {
		app.unauthorizedErrorResponse(w, r, err)
		return
	}

	candidates, err := app.switchCandidates(r.Context(), current)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	memberships, err := app.memberships(r.Context(), candidates)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, memberships); err != nil {
		app.internalServerError(w, r, err)
	}
}

// switchCandidates — shu telefon+parol bilan ochiladigan barcha business qatorlari.
// O'chirilgan userlarda phone bo'sh bo'lgani uchun ular hech qachon mos kelmaydi.
func (app *application) switchCandidates(ctx context.Context, current *store.User) ([]store.User, error) {
	if current.Phone == "" || current.Password == "" {
		return []store.User{*current}, nil
	}

	candidates, err := app.store.Users.LoginCandidates(ctx, current.Phone, current.Password)
	if err != nil {
		return nil, err
	}

	// Joriy user ro'yxatda bo'lishi kafolatlanadi (parol o'zgarish poygasi).
	for i := range candidates {
		if candidates[i].ID == current.ID {
			return candidates, nil
		}
	}

	return append([]store.User{*current}, candidates...), nil
}

// memberships — nomlari bilan boyitilgan a'zoliklar ro'yxati.
func (app *application) memberships(ctx context.Context, users []store.User) ([]businessMembership, error) {
	ids := make([]int64, 0, len(users))
	for i := range users {
		ids = append(ids, users[i].BusinessId)
	}

	businesses, err := app.store.Businesses.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]businessMembership, 0, len(users))
	for i := range users {
		business := businesses[users[i].BusinessId]
		result = append(result, businessMembership{
			BusinessID:     users[i].BusinessId,
			BusinessName:   business.Name,
			BusinessStatus: business.Status,
			UserID:         users[i].ID,
			Username:       users[i].Username,
			CompanyID:      users[i].CompanyId,
			Role:           users[i].Role,
		})
	}

	return result, nil
}

// pickLoginUser — telefon business ichida unikal, businesslar bo'ylab takrorlanishi
// mumkin. business_id berilsa o'sha business qatori, berilmasa birinchisi olinadi;
// qolgan businesslarga mijoz keyin parolsiz almashadi.
func pickLoginUser(candidates []store.User, businessID int64) (*store.User, error) {
	if businessID > 0 {
		for i := range candidates {
			if candidates[i].BusinessId == businessID {
				return &candidates[i], nil
			}
		}
		return nil, fmt.Errorf("LOGIN YOKI PAROL XATO")
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("LOGIN YOKI PAROL XATO")
	}

	return &candidates[0], nil
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

	candidates, err := app.switchCandidates(r.Context(), user)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.respondWithTokens(w, r, user, candidates)
}

// respondWithTokens issues a token pair for the user and writes the standard
// authentication payload shared by login, refresh and business switching.
// `candidates` — shu telefon+parol ochadigan barcha business qatorlari; javobdagi
// `businesses` ro'yxati mijozga qaysi businessga parolsiz o'tish mumkinligini
// ko'rsatadi.
func (app *application) respondWithTokens(w http.ResponseWriter, r *http.Request, user *store.User, candidates []store.User) {
	accessToken, refreshToken, err := issueUserTokenPair(user)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	memberships, err := app.memberships(r.Context(), candidates)
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
		"businesses":         memberships,
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
