package service

import (
	"testing"

	"github.com/mubashshir3767/currencyExchange/internal/store"
)

// Toshkent (kompaniya 1) Namangan (kompaniya 2) olgan xizmat haqini ko'rmaydi:
// summa, valyuta, izoh va haqni olgan kompaniya javobdan chiqib ketadi.
func TestMaskServiceFeeForViewerHidesOtherCompanyFee(t *testing.T) {
	rows := []map[string]interface{}{
		{
			"has_service_fee":        true,
			"service_fee":            "50000 SUM",
			"service_fee_amount":     int64(50000),
			"service_fee_currency":   "SUM",
			"service_fee_details":    "yo'l kira",
			"service_fee_company_id": int64(2),
			"service_fee_company":    "Namangan",
		},
	}

	MaskServiceFeeForViewer(rows, 1)

	res := rows[0]
	if res["has_service_fee"] != false {
		t.Fatalf("has_service_fee = %v, want false", res["has_service_fee"])
	}
	if res["service_fee_amount"] != int64(0) {
		t.Fatalf("service_fee_amount = %v, want 0", res["service_fee_amount"])
	}
	for _, key := range []string{"service_fee", "service_fee_currency", "service_fee_details", "service_fee_company"} {
		if res[key] != "" {
			t.Fatalf("%s = %v, want empty", key, res[key])
		}
	}
	if res["service_fee_company_id"] != int64(0) {
		t.Fatalf("service_fee_company_id = %v, want 0", res["service_fee_company_id"])
	}
}

// Namangan o'zi olgan haqni ko'radi.
func TestMaskServiceFeeForViewerKeepsOwnFee(t *testing.T) {
	rows := []map[string]interface{}{
		{
			"has_service_fee":        true,
			"service_fee":            "50000 SUM",
			"service_fee_amount":     int64(50000),
			"service_fee_currency":   "SUM",
			"service_fee_details":    "yo'l kira",
			"service_fee_company_id": int64(2),
			"service_fee_company":    "Namangan",
		},
	}

	MaskServiceFeeForViewer(rows, 2)

	if rows[0]["service_fee_amount"] != int64(50000) {
		t.Fatalf("service_fee_amount = %v, want 50000", rows[0]["service_fee_amount"])
	}
	if rows[0]["service_fee_company"] != "Namangan" {
		t.Fatalf("service_fee_company = %v, want Namangan", rows[0]["service_fee_company"])
	}
}

// GetByCompanyId javobi oxiridagi meta yozuv (xizmat haqi maydonlari yo'q)
// o'zgarmaydi.
func TestMaskServiceFeeForViewerSkipsMetaRow(t *testing.T) {
	rows := []map[string]interface{}{
		{"get_currencies": map[string]int64{"USD": 10}},
	}

	MaskServiceFeeForViewer(rows, 1)

	if _, ok := rows[0]["service_fee_amount"]; ok {
		t.Fatal("meta yozuvga xizmat haqi maydonlari qo'shildi")
	}
}

// Xom Transaction ro'yxatida ham maskalash kompaniya bo'yicha ishlaydi.
func TestMaskServiceFeeInTransactions(t *testing.T) {
	feeCompanyID := int64(2)
	trans := []store.Transaction{
		{
			ReceivedCompanyId:   1,
			DeliveredCompanyId:  2,
			ServiceFeeAmount:    50000,
			ServiceFeeCurrency:  "SUM",
			ServiceFeeDetails:   "yo'l kira",
			ServiceFeeCompanyId: &feeCompanyID,
		},
		{
			ReceivedCompanyId:   1,
			DeliveredCompanyId:  2,
			ServiceFeeAmount:    30000,
			ServiceFeeCurrency:  "SUM",
			ServiceFeeCompanyId: &[]int64{1}[0],
		},
	}

	MaskServiceFeeInTransactions(trans, 1)

	if trans[0].ServiceFeeAmount != 0 || trans[0].ServiceFeeCompanyId != nil {
		t.Fatalf("Namangan haqi maskalanmadi: %+v", trans[0])
	}
	if trans[0].ServiceFeeCurrency != "" || trans[0].ServiceFeeDetails != "" {
		t.Fatalf("valyuta/izoh maskalanmadi: %+v", trans[0])
	}
	if trans[1].ServiceFeeAmount != 30000 {
		t.Fatalf("o'z haqi o'chib ketdi: %+v", trans[1])
	}
}
