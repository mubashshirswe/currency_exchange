package main

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/env"
	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// JWTUserMiddleware — tokenni tekshiradi va so'rov kontekstiga tenant ma'lumotini
// (user, business, company, role) joylaydi. Business DB'dagi userdan olinadi —
// token ichidagi qiymatga ishonilmaydi.
func (app *application) JWTUserMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := userIDFromToken(GetTokenFromRequest(r), accessTokenType)
			if err != nil {
				log.Printf("failed to validate token: %v", err)
				app.unauthorizedErrorResponse(w, r, err)
				return
			}

			user, err := app.GetUser(r.Context(), userID)
			if err != nil {
				log.Printf("failed to get user by id %v", err)
				app.unauthorizedErrorResponse(w, r, err)
				return
			}

			if user.BusinessId == 0 {
				app.unauthorizedErrorResponse(w, r, errNoBusiness)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, user.ID)
			ctx = context.WithValue(ctx, BusinessKey, user.BusinessId)
			ctx = context.WithValue(ctx, CompanyKey, user.CompanyId)
			ctx = context.WithValue(ctx, RoleKey, user.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OwnerOnlyMiddleware — faqat business egasi (ROLE_OWNER) o'tadigan yo'llar uchun.
func (app *application) OwnerOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := app.requireOwner(w, r); !ok {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUser resolves a user through the cache, falling back to the database.
// Cache failures are logged and treated as a miss: a flaky Redis must never
// turn a valid token into a 401, which the clients read as "session expired".
func (app *application) GetUser(ctx context.Context, id int64) (*store.User, error) {
	if user, err := app.cacheStore.Users.Get(ctx, id); err != nil {
		log.Printf("user cache read failed for id %d: %v", id, err)
	} else if user != nil && user.BusinessId != 0 {
		return user, nil
	}

	user, err := app.store.Users.GetById(ctx, &id)
	if err != nil {
		return nil, err
	}

	if err := app.cacheStore.Users.Set(ctx, user); err != nil {
		log.Printf("user cache write failed for id %d: %v", id, err)
	}

	return user, nil
}

// PlatformAdminMiddleware — business va kompaniya ochish/o'zgartirish faqat
// platforma operatori uchun: mijoz ilovalari bu yo'llarga umuman chiqmaydi.
// Kalit ADMIN_API_KEY env'idan olinadi va X-Admin-Key sarlavhasi bilan
// solishtiriladi. Kalit sozlanmagan bo'lsa endpoint o'chirilgan hisoblanadi.
func (app *application) PlatformAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := env.GetString("ADMIN_API_KEY", "")
		if expected == "" {
			log.Printf("admin endpoint disabled: ADMIN_API_KEY is not set")
			writeError(w, http.StatusServiceUnavailable, "ADMIN API DISABLED")
			return
		}

		provided := r.Header.Get("X-Admin-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			log.Printf("admin key rejected: %s path: %s", r.Method, r.URL.Path)
			writeError(w, http.StatusUnauthorized, "INVALID ADMIN KEY")
			return
		}

		next.ServeHTTP(w, r)
	})
}
