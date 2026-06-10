package notify

import "context"

// TokenError — bitta token uchun FCM xatosi.
type TokenError struct {
	TokenPreview string `json:"token_preview"`
	Error        string `json:"error"`
}

// SendResult — bitta user uchun multicast natijasi.
type SendResult struct {
	UserID       int64        `json:"user_id"`
	TokenCount   int          `json:"tokens"`
	SuccessCount int          `json:"success"`
	FailureCount int          `json:"failure"`
	Errors       []TokenError `json:"errors,omitempty"`
}

// Pusher — userning barcha qurilmalariga push yuboradi.
type Pusher interface {
	SendToUser(ctx context.Context, userID int64, title, body string, data map[string]string) (SendResult, error)
}

type NoopPusher struct{}

func (NoopPusher) SendToUser(context.Context, int64, string, string, map[string]string) (SendResult, error) {
	return SendResult{}, ErrFCMDisabled
}
