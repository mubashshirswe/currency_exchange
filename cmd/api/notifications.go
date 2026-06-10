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
	callerID := r.Context().Value(UserKey).(int64)

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
		if targetUserID != callerID {
			caller, err := app.currentUser(r)
			if err != nil {
				app.internalServerError(w, r, err)
				return
			}
			if !app.isAdminUser(caller) {
				app.unauthorizedErrorResponse(w, r, errors.New("admin required"))
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
