package handlers

import (
	"context"

	"github.com/example/openapi-gen/pkg/api"
	"github.com/example/openapi-gen/pkg/unions"
)

// NotificationChannel represents the channel through which notifications are sent
type NotificationChannel string

const (
	// EmailNotificationChannel tells that notifications are sent via email
	EmailNotificationChannel NotificationChannel = "email"

	// SMSNotificationChannel tells that notifications are sent via SMS
	SMSNotificationChannel NotificationChannel = "sms"

	// PushNotificationChannel tells that notifications are sent via push notifications (app required)
	PushNotificationChannel NotificationChannel = "push"
)

type UpdateUserPreferencesRequest struct {
	// UserId is the ID of the user whose preferences are being updated
	UserId string `openapi:"'userId',in:path"`

	// Preferences contains the user's updated preferences
	// Payment methods are set in the "paymentMethods" field
	PaymentMethod unions.Union2[BankPaymentMethod, CreditCardPaymentMethod] `json:"paymentMethod"`

	PrimaryNotificationChannel NotificationChannel `json:"primaryNotificationChannel"`
}

// UpdateUserPreferences handles user preferences update requests
func UpdateUserPreferences(ctx context.Context, req *UpdateUserPreferencesRequest) (*api.NoContentResponse, error) {
	// Handle user preferences update logic here
	return &api.NoContentResponse{}, nil
}
