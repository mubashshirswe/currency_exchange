package main

import (
	"context"
	"log"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

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

			ctx := context.WithValue(r.Context(), UserKey, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser resolves a user through the cache, falling back to the database.
// Cache failures are logged and treated as a miss: a flaky Redis must never
// turn a valid token into a 401, which the clients read as "session expired".
func (app *application) GetUser(ctx context.Context, id int64) (*store.User, error) {
	if user, err := app.cacheStore.Users.Get(ctx, id); err != nil {
		log.Printf("user cache read failed for id %d: %v", id, err)
	} else if user != nil {
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
