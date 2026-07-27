package store

import (
	"context"
	"database/sql"
	"fmt"
)

// Tranzaksiya ish oqimi (business_settings.transaction_flow).
const (
	// TRANSACTION_FLOW_SIMPLE — yaratish => topshirish (eski holat).
	TRANSACTION_FLOW_SIMPLE = 1
	// TRANSACTION_FLOW_THREE_STAGE — yaratish => qabul qilish => topshirish.
	// Qabul qilish bosqichi balansga ta'sir qilmaydi.
	TRANSACTION_FLOW_THREE_STAGE = 2
)

// BusinessSettings — bitta business (tenant) sozlamalari.
type BusinessSettings struct {
	BusinessID      int64  `json:"business_id"`
	TransactionFlow int64  `json:"transaction_flow"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// IsThreeStage — tranzaksiya 3 bosqichli oqimda ishlaydimi.
func (s BusinessSettings) IsThreeStage() bool {
	return s.TransactionFlow == TRANSACTION_FLOW_THREE_STAGE
}

// ValidTransactionFlow — mijozdan kelgan qiymatni tekshiradi.
func ValidTransactionFlow(flow int64) bool {
	return flow == TRANSACTION_FLOW_SIMPLE || flow == TRANSACTION_FLOW_THREE_STAGE
}

type BusinessSettingsStorage struct {
	db DBTX
}

func NewBusinessSettingsStorage(db DBTX) *BusinessSettingsStorage {
	return &BusinessSettingsStorage{db: db}
}

// GetByBusinessID — sozlamalar; yozuv bo'lmasa standart (SIMPLE) qaytadi,
// shunda eski businesslar uchun alohida migratsiya kerak bo'lmaydi.
func (s *BusinessSettingsStorage) GetByBusinessID(ctx context.Context, businessID int64) (*BusinessSettings, error) {
	query := `SELECT business_id, transaction_flow, created_at, updated_at
				FROM business_settings WHERE business_id = $1`

	settings := &BusinessSettings{}
	err := s.db.QueryRowContext(ctx, query, businessID).Scan(
		&settings.BusinessID,
		&settings.TransactionFlow,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return &BusinessSettings{
			BusinessID:      businessID,
			TransactionFlow: TRANSACTION_FLOW_SIMPLE,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// Upsert — sozlamani yozadi yoki yangilaydi.
func (s *BusinessSettingsStorage) Upsert(ctx context.Context, settings *BusinessSettings) error {
	if !ValidTransactionFlow(settings.TransactionFlow) {
		return fmt.Errorf("noto'g'ri transaction_flow: %d", settings.TransactionFlow)
	}

	query := `
		INSERT INTO business_settings (business_id, transaction_flow)
		VALUES ($1, $2)
		ON CONFLICT (business_id) DO UPDATE
			SET transaction_flow = EXCLUDED.transaction_flow,
				updated_at = now()
		RETURNING business_id, transaction_flow, created_at, updated_at`

	return s.db.QueryRowContext(ctx, query, settings.BusinessID, settings.TransactionFlow).Scan(
		&settings.BusinessID,
		&settings.TransactionFlow,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
}
