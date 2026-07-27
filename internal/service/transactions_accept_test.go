package service

import (
	"testing"
	"time"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

func TestIsOpenTransaction(t *testing.T) {
	cases := map[int64]bool{
		store.STATUS_CREATED:   true,
		store.STATUS_ACCEPTED:  true,
		store.STATUS_COMPLETED: false,
		store.STATUS_ARCHIVED:  false,
	}

	for status, want := range cases {
		if got := isOpenTransaction(status); got != want {
			t.Fatalf("isOpenTransaction(%d) = %v, want %v", status, got, want)
		}
	}
}

func TestAttachAcceptedInfoEmptyForSimpleFlow(t *testing.T) {
	res := map[string]interface{}{}
	attachAcceptedInfo(res, nil, store.Transaction{Status: store.STATUS_CREATED})

	if res["accepted_user_id"] != (*int64)(nil) {
		t.Fatalf("accepted_user_id = %v, want nil", res["accepted_user_id"])
	}
	if res["accepted_user"] != "" {
		t.Fatalf("accepted_user = %q, want empty", res["accepted_user"])
	}
	if res["accepted_at"] != "" {
		t.Fatalf("accepted_at = %q, want empty", res["accepted_at"])
	}
	if res["is_accepted"] != false {
		t.Fatalf("is_accepted = %v, want false", res["is_accepted"])
	}
}

func TestAttachAcceptedInfoResolvesUser(t *testing.T) {
	userID := int64(7)
	companyID := int64(3)
	acceptedAt := time.Date(2026, 7, 27, 10, 30, 0, 0, time.UTC)

	users := []store.User{{ID: 7, Username: "Ali"}, {ID: 8, Username: "Vali"}}
	tran := store.Transaction{
		Status:              store.STATUS_ACCEPTED,
		AcceptedUserId:      &userID,
		AcceptedCompanyId:   &companyID,
		AcceptedAt:          &acceptedAt,
		AcceptedAtFormatted: "2026-07-27 15:30:00",
	}

	res := map[string]interface{}{}
	attachAcceptedInfo(res, users, tran)

	if res["accepted_user"] != "Ali" {
		t.Fatalf("accepted_user = %v, want Ali", res["accepted_user"])
	}
	if res["accepted_user_id"] != &userID {
		t.Fatalf("accepted_user_id = %v, want %d", res["accepted_user_id"], userID)
	}
	if res["accepted_company_id"] != &companyID {
		t.Fatalf("accepted_company_id = %v, want %d", res["accepted_company_id"], companyID)
	}
	if res["accepted_at"] != "2026-07-27 15:30:00" {
		t.Fatalf("accepted_at = %v", res["accepted_at"])
	}
	if res["is_accepted"] != true {
		t.Fatalf("is_accepted = %v, want true", res["is_accepted"])
	}
}

// Qabul qilingan tranzaksiya hali ochiq: "yetkazilishi kerak" ro'yxatidan chiqmaydi.
func TestAcceptedTransactionStaysInProcessList(t *testing.T) {
	if !isOpenTransaction(TRANSACTION_STATUS_ACCEPTED) {
		t.Fatal("qabul qilingan tranzaksiya jarayondagi ro'yxatda qolishi kerak")
	}
	if TRANSACTION_STATUS_ACCEPTED != store.STATUS_ACCEPTED {
		t.Fatalf("service va store status qiymatlari mos emas: %d != %d",
			TRANSACTION_STATUS_ACCEPTED, store.STATUS_ACCEPTED)
	}
}
