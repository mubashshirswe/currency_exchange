package main

import (
	"context"
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
}

type RefreshTokenPayload struct {
	RefreshToken string `json:"refresh_token"`
}

func (app *application) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload UserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		Username:  payload.Username,
		Phone:     payload.Phone,
		Role:      payload.Role,
		Password:  payload.Password,
		CompanyId: payload.CompanyId,
	}

	if err := app.store.Users.Create(r.Context(), user); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.store.Balances.Create(context.Background(), &store.Balance{
		Balance:   0,
		UserId:    user.ID,
		CompanyId: user.CompanyId,
		InOutLay:  0,
		OutInLay:  0,
		Currency:  "USD",
	}); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := app.store.Balances.Create(context.Background(), &store.Balance{
		Balance:   0,
		UserId:    user.ID,
		CompanyId: user.CompanyId,
		InOutLay:  0,
		OutInLay:  0,
		Currency:  "SUM",
	}); err != nil {
		app.badRequestResponse(w, r, err)
		return
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

	user := &store.User{
		Phone:    payload.Phone,
		Password: payload.Password,
	}

	if err := app.store.Users.Login(r.Context(), user); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.respondWithTokens(w, r, user)
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
	accessToken, refreshToken, err := issueTokenPair(user.ID)
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

func (app *application) GetAllUserHandler(w http.ResponseWriter, r *http.Request) {
	users, err := app.store.Users.GetAll(r.Context())
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
	id := getIDFromContext(r)
	var payload UserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		ID:        id,
		Username:  payload.Username,
		Role:      payload.Role,
		Password:  payload.Password,
		Avatar:    payload.Avatar,
		CompanyId: payload.CompanyId,
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
	id := getIDFromContext(r)

	if err := app.store.Users.Delete(r.Context(), &id); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, "DELETED"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
