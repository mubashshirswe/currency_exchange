package notify

import "context"

// DeliveredUser sends FCM notifications for transaction lifecycle events.
type DeliveredUser interface {
	NotifyPendingDelivery(ctx context.Context, deliveredUserID *int64, txnID int64, phone, details string)
	NotifyDeliveryCompleted(ctx context.Context, deliveredUserID int64, txnID int64, details string)
	// NotifyNewOrderToCompany notifies all members of the company that accepted the order (delivered_company_id).
	NotifyNewOrderToCompany(ctx context.Context, companyID int64, txnID int64, phone, details string)
}

type NoopDeliveredUser struct{}

func (NoopDeliveredUser) NotifyPendingDelivery(context.Context, *int64, int64, string, string) {}

func (NoopDeliveredUser) NotifyDeliveryCompleted(context.Context, int64, int64, string) {}

func (NoopDeliveredUser) NotifyNewOrderToCompany(context.Context, int64, int64, string, string) {}
