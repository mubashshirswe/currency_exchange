package types

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

type TransactionComplete struct {
	TransactionID      int64  `json:"transactionID"`
	DeliveredUserId    int64  `json:"delivered_user_id"`
	RecievedServiceFee any    `json:"received_service_fee"`
	ServiceFeeAmount   int64  `json:"received_service_fee_amount"`
	ServiceFeeCurrency string `json:"received_service_fee_currency"`
	ServiceFeeDetails  string `json:"received_service_fee_details"`
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
