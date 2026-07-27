package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// BusinessSettingsPayload — business sozlamalarini yangilash.
// transaction_flow: 1 = oddiy (yaratish => topshirish),
//                   2 = 3 bosqich (yaratish => qabul qilish => topshirish).
type BusinessSettingsPayload struct {
	TransactionFlow int64 `json:"transaction_flow"`
}

// GetMyBusinessSettingsHandler — joriy business sozlamalari (hamma o'qiy oladi:
// mobil ilova qaysi oqim yoqilganini bilishi kerak).
func (app *application) GetMyBusinessSettingsHandler(w http.ResponseWriter, r *http.Request) {
	t, ok := app.requireTenant(w, r)
	if !ok {
		return
	}

	settings, err := app.store.BusinessSettings.GetByBusinessID(r.Context(), t.BusinessID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, settings); err != nil {
		app.internalServerError(w, r, err)
	}
}

// UpdateMyBusinessSettingsHandler — sozlamani o'zgartirish; faqat business egasi.
func (app *application) UpdateMyBusinessSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var payload BusinessSettingsPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	t, ok := app.requireOwner(w, r)
	if !ok {
		return
	}

	if !store.ValidTransactionFlow(payload.TransactionFlow) {
		app.badRequestResponse(w, r, fmt.Errorf(
			"transaction_flow faqat %d (oddiy) yoki %d (3 bosqich) bo'lishi mumkin",
			store.TRANSACTION_FLOW_SIMPLE, store.TRANSACTION_FLOW_THREE_STAGE,
		))
		return
	}

	settings := &store.BusinessSettings{
		BusinessID:      t.BusinessID,
		TransactionFlow: payload.TransactionFlow,
	}
	if err := app.store.BusinessSettings.Upsert(r.Context(), settings); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.writeResponse(w, http.StatusOK, settings); err != nil {
		app.internalServerError(w, r, err)
	}
}

// requireAcceptedWhenThreeStage — 3 bosqichli oqimda yakunlashdan oldin buyurtma
// qabul qilingan bo'lishi shart. Oddiy oqimda hech narsa tekshirilmaydi.
func (app *application) requireAcceptedWhenThreeStage(r *http.Request, businessID, transactionID int64) error {
	settings, err := app.store.BusinessSettings.GetByBusinessID(r.Context(), businessID)
	if err != nil {
		return err
	}
	if !settings.IsThreeStage() {
		return nil
	}

	tran, err := app.store.Transactions.GetById(r.Context(), transactionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("BUYURTMA ALLAQACHON YAKUNLANGAN")
		}
		return err
	}
	if tran.Status != store.STATUS_ACCEPTED {
		return fmt.Errorf(types.TRANSACTION_NOT_ACCEPTED)
	}

	return nil
}
