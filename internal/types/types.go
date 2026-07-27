package types

import "strings"

type BalanceRecordPayload struct {
	ID               int64  `json:"id"`
	ReceivedMoney    int64  `json:"received_money"`
	ReceivedCurrency string `json:"received_currency"`
	SelledMoney      int64  `json:"selled_money"`
	SelledCurrency   string `json:"selled_currency"`
	UserId           int64  `json:"user_id"`
	CompanyID        *int64 `json:"company_id"`
	Details          string `json:"details"`
}

// CompanyBalanceRecordPayload — kompaniya balansiga kirim/chiqim uchun payload.
// Eski BalanceRecordPayload'dan ALOHIDA: faqat company_balances'ga ta'sir qiladi.
// UserId va CompanyID server tomonda JWT'dan to'ldiriladi.
type CompanyBalanceRecordPayload struct {
	ReceivedMoney    int64  `json:"received_money"`
	ReceivedCurrency string `json:"received_currency"`
	SelledMoney      int64  `json:"selled_money"`
	SelledCurrency   string `json:"selled_currency"`
	UserId           int64  `json:"user_id"`
	CompanyID        int64  `json:"company_id"`
	Details          string `json:"details"`
}

// TransactionAccept — 3 bosqichli oqimning o'rta bosqichi (qabul qilish) payload'i.
// Balansga ta'sir qilmaydi, shuning uchun summa/xizmat haqi maydonlari yo'q.
type TransactionAccept struct {
	TransactionID int64 `json:"transactionID"`
}

type TransactionComplete struct {
	TransactionID      int64  `json:"transactionID"`
	DeliveredUserId    int64  `json:"delivered_user_id"`
	RecievedServiceFee any    `json:"received_service_fee"`
	ServiceFeeAmount   int64  `json:"received_service_fee_amount"`
	ServiceFeeCurrency string `json:"received_service_fee_currency"`
	ServiceFeeDetails  string `json:"received_service_fee_details"`

	// Qarzdorlik (debt) ma'lumotlari — yuborilsa, yakunlashda debt bo'limiga ham yozuv tushadi.
	DebtorID     int64  `json:"debtor_id"`
	DebtFullName string `json:"full_name"`
	DebtPhone    string `json:"phone"`
	DebtDetails  string `json:"debt_details"`
	DebtType     int    `json:"debt_type"`
}

// DebtInput — transaction yakunlash payload'idagi qarzdorlik maydonlari.
func (t TransactionComplete) DebtInput() TransactionDebtInput {
	return TransactionDebtInput{
		DebtorID: t.DebtorID,
		FullName: t.DebtFullName,
		Phone:    t.DebtPhone,
		Details:  t.DebtDetails,
		Type:     t.DebtType,
	}
}

// TransactionDebtInput — transaction yaratish/yakunlashda qo'shib yuboriladigan
// qarzdorlik ma'lumotlari. To'ldirilgan bo'lsa, transaction bilan bir tranzaksiyada
// debtor + debt yozuvlari yaratiladi.
type TransactionDebtInput struct {
	DebtorID int64
	FullName string
	Phone    string
	Details  string
	Type     int
}

// Enabled — qarz yozuvi yaratilishi kerakmi (debtor_id yoki full_name yuborilganmi).
func (d TransactionDebtInput) Enabled() bool {
	return d.DebtorID > 0 || strings.TrimSpace(d.FullName) != ""
}

type ReceivedIncomes struct {
	ReceivedAmount   int64  `json:"received_amount"`
	ReceivedCurrency string `json:"received_currency"`
}

type DeliveredOutcomes struct {
	DeliveredAmount   int64  `json:"delivered_amount"`
	DeliveredCurrency string `json:"delivered_currency"`
}

type Pagination struct {
	Page                int         `json:"page"`
	Limit               int         `json:"limit"`
	OrderBy             string      `json:"order_by"`
	Data                interface{} `json:"data"`
	Offset              int         `json:"offset"`
	TaskId              int         `json:"taskId"`
	ProductCollectionId int         `json:"productCollectionId"`
	UserId              int64       `json:"user_id"`
	Language            *string     `json:"language"`
}

const (
	TYPE_SELL = 1
	TYPE_BUY  = 2
)
