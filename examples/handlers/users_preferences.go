package handlers

import (
	"context"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/gork-labs/gork/pkg/unions"
)

// NotificationChannel represents the channel through which notifications are sent.
type NotificationChannel string

const (
	// EmailNotificationChannel tells that notifications are sent via email.
	EmailNotificationChannel NotificationChannel = "email"

	// SMSNotificationChannel tells that notifications are sent via SMS.
	SMSNotificationChannel NotificationChannel = "sms"

	// PushNotificationChannel tells that notifications are sent via push notifications (app required).
	PushNotificationChannel NotificationChannel = "push"
)

// UpdateUserPreferencesRequest represents the request for updating user preferences.
type UpdateUserPreferencesRequest struct {
	// UserID is the ID of the user whose preferences are being updated
	UserID string `openapi:"'userID',in:path"`

	// Preferences contains the user's updated preferences
	// Payment methods are set in the "paymentMethods" field
	// PaymentMethod contains the user's payment method
	PaymentMethod unions.Union2[BankPaymentMethod, CreditCardPaymentMethod] `json:"paymentMethod"`

	// PrimaryNotificationChannel is the user's preferred notification channel
	PrimaryNotificationChannel NotificationChannel `json:"primaryNotificationChannel"`
}

// UpdateUserPreferences handles user preferences update requests.
func UpdateUserPreferences(_ context.Context, _ *UpdateUserPreferencesRequest) (*api.NoContentResponse, error) {
	// Handle user preferences update logic here
	return &api.NoContentResponse{}, nil
}
