package service

import (
	"testing"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// Toshkent yaratgan, Namangan yakunlagan va xizmat haqini olgan tranzaksiya:
// javob ikkala kompaniya uchun ham bir xil — haqni Namangan olgani ko'rinadi.
func TestAttachServiceFeeOwnerFeeTakenAtComplete(t *testing.T) {
	companies := []store.Company{
		{ID: 1, Name: "Toshkent"},
		{ID: 2, Name: "Namangan"},
	}
	deliveredUserID := int64(9)
	feeCompanyID := int64(2)

	tran := store.Transaction{
		ReceivedCompanyId:   1,
		DeliveredCompanyId:  2,
		DeliveredUserId:     &deliveredUserID,
		ServiceFeeAmount:    50000,
		ServiceFeeCurrency:  "SUM",
		ServiceFeeCompanyId: &feeCompanyID,
		Status:              store.STATUS_COMPLETED,
	}

	res := map[string]interface{}{}
	attachServiceFeeOwner(res, companies, tran)

	if res["service_fee_company_id"] != int64(2) {
		t.Fatalf("service_fee_company_id = %v, want 2", res["service_fee_company_id"])
	}
	if res["service_fee_company"] != "Namangan" {
		t.Fatalf("service_fee_company = %v, want Namangan", res["service_fee_company"])
	}
}

// Xizmat haqi yaratishda kiritilgan: haq qabul qiluvchi kompaniyada qoladi.
func TestAttachServiceFeeOwnerFeeTakenAtCreate(t *testing.T) {
	companies := []store.Company{
		{ID: 1, Name: "Toshkent"},
		{ID: 2, Name: "Namangan"},
	}
	feeCompanyID := int64(1)

	tran := store.Transaction{
		ReceivedCompanyId:   1,
		DeliveredCompanyId:  2,
		ServiceFeeAmount:    30000,
		ServiceFeeCompanyId: &feeCompanyID,
		Status:              store.STATUS_CREATED,
	}

	res := map[string]interface{}{}
	attachServiceFeeOwner(res, companies, tran)

	if res["service_fee_company_id"] != int64(1) {
		t.Fatalf("service_fee_company_id = %v, want 1", res["service_fee_company_id"])
	}
	if res["service_fee_company"] != "Toshkent" {
		t.Fatalf("service_fee_company = %v, want Toshkent", res["service_fee_company"])
	}
}

// Eski yozuvda transaction_service_fees qatori yo'q (masalan v1 `complete`):
// yakunlangan bo'lsa yetkazuvchi, aks holda qabul qiluvchi kompaniya.
func TestServiceFeeOwnerFallbackWithoutFeeRow(t *testing.T) {
	deliveredUserID := int64(9)

	completed := store.Transaction{
		ReceivedCompanyId:  1,
		DeliveredCompanyId: 2,
		DeliveredUserId:    &deliveredUserID,
		ServiceFeeAmount:   50000,
		Status:             store.STATUS_COMPLETED,
	}
	if got := serviceFeeOwnerCompanyID(completed); got != 2 {
		t.Fatalf("serviceFeeOwnerCompanyID(completed) = %d, want 2", got)
	}

	open := store.Transaction{
		ReceivedCompanyId:  1,
		DeliveredCompanyId: 2,
		ServiceFeeAmount:   50000,
		Status:             store.STATUS_CREATED,
	}
	if got := serviceFeeOwnerCompanyID(open); got != 1 {
		t.Fatalf("serviceFeeOwnerCompanyID(open) = %d, want 1", got)
	}
}

func TestServiceFeeOwnerEmptyWithoutFee(t *testing.T) {
	res := map[string]interface{}{}
	attachServiceFeeOwner(res, []store.Company{{ID: 1, Name: "Toshkent"}}, store.Transaction{
		ReceivedCompanyId:  1,
		DeliveredCompanyId: 2,
	})

	if res["service_fee_company_id"] != int64(0) {
		t.Fatalf("service_fee_company_id = %v, want 0", res["service_fee_company_id"])
	}
	if res["service_fee_company"] != "" {
		t.Fatalf("service_fee_company = %v, want empty", res["service_fee_company"])
	}
}
