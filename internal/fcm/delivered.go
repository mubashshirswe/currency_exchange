package fcm

import (
	"context"
	"log"
	"strconv"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/mubashshir3767/currencyExchange/internal/notify"
	"github.com/mubashshir3767/currencyExchange/internal/store"
)

type DeliveredNotifier struct {
	store  store.Storage
	client *messaging.Client
}

func NewDeliveredNotifier(credsPath string, st store.Storage) (*DeliveredNotifier, error) {
	app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsFile(credsPath))
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}
	return &DeliveredNotifier{store: st, client: client}, nil
}

var (
	_ notify.DeliveredUser = (*DeliveredNotifier)(nil)
	_ notify.Pusher        = (*DeliveredNotifier)(nil)
)

func (n *DeliveredNotifier) NotifyPendingDelivery(ctx context.Context, deliveredUserID *int64, txnID int64, phone, details string) {
	if deliveredUserID == nil {
		return
	}
	n.sendToUser(ctx, *deliveredUserID, "Yangi tranzaksiya", "Yetkazib berish uchun yangi buyurtma", map[string]string{
		"type":           "transaction_pending",
		"transaction_id": strconv.FormatInt(txnID, 10),
		"phone":          phone,
		"details":        details,
	})
}

func (n *DeliveredNotifier) NotifyNewOrderToCompany(ctx context.Context, companyID int64, txnID int64, phone, details string) {
	if companyID == 0 {
		log.Printf("fcm NotifyNewOrderToCompany: skip companyID=0 txn=%d", txnID)
		return
	}
	tokens, err := n.store.UserSessions.FCMTokensByCompanyID(ctx, companyID)
	if err != nil {
		log.Printf("fcm NotifyNewOrderToCompany: tokens lookup failed company=%d txn=%d: %v", companyID, txnID, err)
		return
	}
	if len(tokens) == 0 {
		log.Printf("fcm NotifyNewOrderToCompany: no tokens for company=%d txn=%d", companyID, txnID)
		return
	}
	n.sendMulticast(ctx, tokens, "Yangi buyurtma", "Yangi buyurtma qabul qilindi", map[string]string{
		"type":           "transaction_pending",
		"transaction_id": strconv.FormatInt(txnID, 10),
		"phone":          phone,
		"details":        details,
	})
}

func (n *DeliveredNotifier) NotifyDeliveryCompleted(ctx context.Context, deliveredUserID int64, txnID int64, details string) {
	body := details
	if body == "" {
		body = "Tranzaksiya muvaffaqiyatli yakunlandi"
	}
	n.sendToUser(ctx, deliveredUserID, "Tranzaksiya yakunlandi", body, map[string]string{
		"type":           "transaction_completed",
		"transaction_id": strconv.FormatInt(txnID, 10),
		"details":        details,
	})
}

func (n *DeliveredNotifier) SendToUser(
	ctx context.Context,
	userID int64,
	title, body string,
	data map[string]string,
) (notify.SendResult, error) {
	tokens, err := n.store.UserSessions.FCMTokensByUserID(ctx, userID)
	if err != nil {
		return notify.SendResult{UserID: userID}, err
	}
	if len(tokens) == 0 {
		log.Printf("fcm SendToUser: no tokens for user=%d title=%q", userID, title)
		return notify.SendResult{UserID: userID}, nil
	}
	result, err := n.sendMulticast(ctx, tokens, title, body, data)
	result.UserID = userID
	return result, err
}

func (n *DeliveredNotifier) sendToUser(ctx context.Context, userID int64, title, body string, data map[string]string) {
	_, _ = n.SendToUser(ctx, userID, title, body, data)
}

func (n *DeliveredNotifier) sendMulticast(
	ctx context.Context,
	tokens []string,
	title, body string,
	data map[string]string,
) (notify.SendResult, error) {
	if len(tokens) == 0 {
		return notify.SendResult{}, nil
	}
	msg := &messaging.MulticastMessage{
		Tokens:       tokens,
		Notification: &messaging.Notification{Title: title, Body: body},
		Data:         data,
	}
	br, err := n.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		log.Printf("fcm SendEachForMulticast: %v", err)
		return notify.SendResult{TokenCount: len(tokens)}, err
	}
	log.Printf("fcm sent title=%q tokens=%d success=%d failure=%d", title, len(tokens), br.SuccessCount, br.FailureCount)

	var tokenErrors []notify.TokenError
	for i, resp := range br.Responses {
		if resp.Success {
			continue
		}
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Error()
		}
		log.Printf("fcm token failed preview=%s err=%s", tokenPreview(tokens[i]), errMsg)
		tokenErrors = append(tokenErrors, notify.TokenError{
			TokenPreview: tokenPreview(tokens[i]),
			Error:        errMsg,
		})
		if messaging.IsRegistrationTokenNotRegistered(resp.Error) || messaging.IsInvalidArgument(resp.Error) {
			if err := n.store.UserSessions.DeleteByFCMToken(ctx, tokens[i]); err != nil {
				log.Printf("fcm cleanup token: %v", err)
			}
		}
	}
	return notify.SendResult{
		TokenCount:   len(tokens),
		SuccessCount: br.SuccessCount,
		FailureCount: br.FailureCount,
		Errors:       tokenErrors,
	}, nil
}

func tokenPreview(token string) string {
	if len(token) <= 16 {
		return token
	}
	return token[:8] + "…" + token[len(token)-6:]
}
