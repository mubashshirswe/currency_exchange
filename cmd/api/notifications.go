package main

import (
	"errors"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/notify"
)

type sendUserNotificationPayload struct {
	UserID *int64            `json:"user_id"`
	Title  string            `json:"title" validate:"required"`
	Body   string            `json:"body" validate:"required"`
	Data   map[string]string `json:"data"`
}

// SendUserNotificationHandler — userning barcha FCM tokenlariga push yuboradi.
// user_id berilmasa joriy foydalanuvchiga; boshqa user uchun faqat admin.
func (app *application) SendUserNotificationHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}
	callerID := t.UserID

	var payload sendUserNotificationPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	targetUserID := callerID
	if payload.UserID != nil {
		targetUserID = *payload.UserID
		// Boshqa userga push — faqat business egasi va faqat o'z businessi ichida.
		if targetUserID != callerID {
			if !t.IsOwner() {
				app.forbiddenResponse(w, r, errors.New("admin required"))
				return
			}
			if err := app.authorizeUser(r, t, targetUserID); err != nil {
				app.handleScopeError(w, r, err)
				return
			}
		}
	}

	result, err := app.pusher.SendToUser(
		r.Context(),
		targetUserID,
		payload.Title,
		payload.Body,
		payload.Data,
	)
	if err != nil {
		if errors.Is(err, notify.ErrFCMDisabled) {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, result); err != nil {
		app.internalServerError(w, r, err)
	}
}
