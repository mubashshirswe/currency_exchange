package store

import (
	"context"
	"database/sql"
	"fmt"
)

// businessScopedTables — {id} bo'yicha keladigan resurslar. Handler guard'i
// resurs qaysi businessga tegishli ekanini shu ro'yxat orqali tekshiradi.
var businessScopedTables = map[string]bool{
	"transactions":             true,
	"exchanges":                true,
	"balances":                 true,
	"balance_records":          true,
	"company_balances":         true,
	"company_balance_records":  true,
	"soft_balances":            true,
	"soft_balance_records":     true,
	"debtors":                  true,
	"debts":                    true,
	"transaction_service_fees": true,
	"service_fee_settlements":  true,
	"companies":                true,
	"users":                    true,
}

// BusinessIDOf — resurs (jadval + id) qaysi businessga tegishli. Yozuv topilmasa
// sql.ErrNoRows qaytadi.
func (s *Storage) BusinessIDOf(ctx context.Context, table string, id int64) (int64, error) {
	if !businessScopedTables[table] {
		return 0, fmt.Errorf("unknown resource: %v", table)
	}

	var businessID sql.NullInt64
	query := fmt.Sprintf(`SELECT business_id FROM %s WHERE id = $1`, table)
	if err := s.DB.QueryRowContext(ctx, query, id).Scan(&businessID); err != nil {
		return 0, err
	}

	return businessID.Int64, nil
}

// Filtrlashga ruxsat etilgan ustunlar. Ustun nomi so'rov matniga qo'shilgani
// uchun (parametr sifatida bera bo'lmaydi) faqat shu ro'yxatdagilar o'tadi.
var (
	exchangeFilterFields = map[string]bool{
		"id":         true,
		"user_id":    true,
		"company_id": true,
		"status":     true,
	}

	balanceRecordFilterFields = map[string]bool{
		"id":             true,
		"user_id":        true,
		"company_id":     true,
		"balance_id":     true,
		"transaction_id": true,
		"exchange_id":    true,
		"debt_id":        true,
		"type":           true,
	}

	transactionFilterFields = map[string]bool{
		"id":                   true,
		"received_user_id":     true,
		"delivered_user_id":    true,
		"received_company_id":  true,
		"delivered_company_id": true,
	}
)

func checkFilterField(allowed map[string]bool, field string) error {
	if !allowed[field] {
		return fmt.Errorf("invalid field name: %v", field)
	}
	return nil
}
