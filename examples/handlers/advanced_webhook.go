package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gork-labs/gork/pkg/api"
	stripepkg "github.com/gork-labs/gork/pkg/webhooks/stripe"
	"github.com/stripe/stripe-go/v76"
)

// Advanced webhook handler example demonstrating:
// - Custom metadata validation
// - Database integration
// - Error handling
// - Logging and monitoring

// CustomPaymentMetadata extends the basic Stripe payment metadata with application-specific fields.
type CustomPaymentMetadata struct {
	UserID       string `json:"user_id" validate:"required,uuid4"`
	ProjectID    string `json:"project_id" validate:"required,uuid4"`
	OrderID      string `json:"order_id" validate:"required"`
	PlanType     string `json:"plan_type" validate:"required,oneof=basic premium enterprise"`
	BillingCycle string `json:"billing_cycle" validate:"oneof=monthly yearly"`
	CouponCode   string `json:"coupon_code" validate:"omitempty,alphanum,max=20"`
	ReferrerID   string `json:"referrer_id" validate:"omitempty,uuid4"`
}

// AdvancedWebhookHandlers demonstrates production-ready webhook handling.
type AdvancedWebhookHandlers struct {
	db     *sql.DB
	logger *log.Logger
}

// NewAdvancedWebhookHandlers creates handlers with database and logging support.
func NewAdvancedWebhookHandlers(db *sql.DB, logger *log.Logger) *AdvancedWebhookHandlers {
	return &AdvancedWebhookHandlers{db: db, logger: logger}
}

// HandleAdvancedPaymentSuccess demonstrates advanced payment processing with:
// - Database transactions
// - Error handling and rollback
// - Audit logging
// - Business logic validation.
func (h *AdvancedWebhookHandlers) HandleAdvancedPaymentSuccess(ctx context.Context, _ *stripe.PaymentIntent, metadata *CustomPaymentMetadata) error {
	h.logger.Printf("Processing payment success for user %s, order %s", metadata.UserID, metadata.OrderID)

	// Start database transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		h.logger.Printf("Failed to start transaction: %v", err)
		return fmt.Errorf("database error: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Will be ignored if tx.Commit() succeeds

	// 1. Update payment status
	_, err = tx.ExecContext(ctx,
		"UPDATE payments SET status = $1, processed_at = NOW() WHERE order_id = $2",
		"completed", metadata.OrderID,
	)
	if err != nil {
		h.logger.Printf("Failed to update payment status: %v", err)
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// 2. Update user subscription if applicable
	if metadata.PlanType != "" {
		_, err = tx.ExecContext(ctx,
			"UPDATE subscriptions SET plan_type = $1, billing_cycle = $2, updated_at = NOW() WHERE user_id = $3",
			metadata.PlanType, metadata.BillingCycle, metadata.UserID,
		)
		if err != nil {
			h.logger.Printf("Failed to update subscription: %v", err)
			return fmt.Errorf("failed to update subscription: %w", err)
		}
	}

	// 3. Handle referral rewards if applicable
	if metadata.ReferrerID != "" {
		err = h.processReferralReward(ctx, tx, metadata.ReferrerID, metadata.UserID)
		if err != nil {
			h.logger.Printf("Failed to process referral reward: %v", err)
			// Don't fail the entire transaction for referral processing
			// Just log the error and continue
		}
	}

	// 4. Apply coupon tracking if used
	if metadata.CouponCode != "" {
		_, err = tx.ExecContext(ctx,
			"UPDATE coupon_usage SET used_at = NOW(), user_id = $1 WHERE code = $2 AND used_at IS NULL",
			metadata.UserID, metadata.CouponCode,
		)
		if err != nil {
			h.logger.Printf("Failed to update coupon usage: %v", err)
			// Continue - coupon tracking is not critical
		}
	}

	// 5. Create audit log entry
	_, err = tx.ExecContext(ctx,
		"INSERT INTO webhook_audit_log (event_type, user_id, order_id, metadata, processed_at) VALUES ($1, $2, $3, $4, NOW())",
		"payment_intent.succeeded", metadata.UserID, metadata.OrderID, fmt.Sprintf("%+v", metadata),
	)
	if err != nil {
		h.logger.Printf("Failed to create audit log: %v", err)
		// Continue - audit logging failure shouldn't block payment processing
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		h.logger.Printf("Failed to commit transaction: %v", err)
		return fmt.Errorf("failed to commit payment processing: %w", err)
	}

	h.logger.Printf("Successfully processed payment for user %s, order %s", metadata.UserID, metadata.OrderID)

	// Success; provider standard success response will be used
	return nil
}

// processReferralReward handles referral reward processing.
func (h *AdvancedWebhookHandlers) processReferralReward(ctx context.Context, tx *sql.Tx, referrerID, newUserID string) error {
	// Check if referrer exists and is eligible for rewards
	var referrerStatus string
	err := tx.QueryRowContext(ctx,
		"SELECT status FROM users WHERE id = $1",
		referrerID,
	).Scan(&referrerStatus)
	if err != nil {
		return fmt.Errorf("referrer not found: %w", err)
	}

	if referrerStatus != "active" {
		return fmt.Errorf("referrer not eligible for rewards")
	}

	// Add referral credit
	_, err = tx.ExecContext(ctx,
		"INSERT INTO referral_credits (referrer_id, referred_user_id, amount, created_at) VALUES ($1, $2, $3, NOW())",
		referrerID, newUserID, 10.00, // $10 referral credit
	)

	return err
}

// HandleFailedPaymentWithRetry demonstrates handling payment failures with retry logic.
func (h *AdvancedWebhookHandlers) HandleFailedPaymentWithRetry(ctx context.Context, _ *stripe.PaymentIntent, metadata *CustomPaymentMetadata) error {
	h.logger.Printf("Processing payment failure for user %s, order %s", metadata.UserID, metadata.OrderID)

	// Update payment status and increment retry count
	var retryCount int
	err := h.db.QueryRowContext(ctx,
		"UPDATE payments SET status = $1, failure_count = failure_count + 1, last_failure_at = NOW() WHERE order_id = $2 RETURNING failure_count",
		"failed", metadata.OrderID,
	).Scan(&retryCount)
	if err != nil {
		h.logger.Printf("Failed to update payment failure: %v", err)
		return fmt.Errorf("database error: %w", err)
	}

	// Decide follow-up actions (retry or permanent fail)

	// Determine next action based on retry count
	if retryCount >= 3 {
		// Mark order as permanently failed
		_, err = h.db.ExecContext(ctx,
			"UPDATE payments SET status = $1 WHERE order_id = $2",
			"permanently_failed", metadata.OrderID,
		)
		if err != nil {
			h.logger.Printf("Failed to mark payment as permanently failed: %v", err)
		}

		// Could emit metrics/alerts here

		// TODO: Send notification to user about payment failure
	} else {
		// Schedule retry
		// Could enqueue retry job here
		h.logger.Printf("Scheduling retry for order %s (attempt %d)", metadata.OrderID, retryCount+1)
		// TODO: Schedule retry job or send retry notification
	}

	h.logger.Printf("Payment failure processed for user %s, order %s, retry count: %d",
		metadata.UserID, metadata.OrderID, retryCount)

	return nil
}

// GetAdvancedWebhookHandler creates the HTTP handler with custom event types and enhanced error handling.
func (h *AdvancedWebhookHandlers) GetAdvancedWebhookHandler() http.HandlerFunc {
	// Create webhook handler with only the events we want to handle
	customEventTypes := []string{
		"payment_intent.succeeded",
		"payment_intent.payment_failed",
		"customer.subscription.updated",
		"invoice.payment_succeeded",
		"invoice.payment_failed",
	}

	handler := stripepkg.NewHandler("whsec_example", customEventTypes...)

	return api.WebhookHandlerFunc[stripepkg.WebhookRequest](
		handler,
		api.WithEventHandler[stripe.PaymentIntent, CustomPaymentMetadata]("payment_intent.succeeded", h.HandleAdvancedPaymentSuccess),
		api.WithEventHandler[stripe.PaymentIntent, CustomPaymentMetadata]("payment_intent.payment_failed", h.HandleFailedPaymentWithRetry),
	)
}

// Example usage with middleware for monitoring and error tracking:
//
// func (h *AdvancedWebhookHandlers) WithMonitoring(handler http.HandlerFunc) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         start := time.Now()
//
//         // Call the actual handler
//         handler(w, r)
//
//         // Log metrics
//         duration := time.Since(start)
//         h.logger.Printf("Webhook processed in %v", duration)
//
//         // Send metrics to monitoring service (e.g., Prometheus, DataDog)
//         // metrics.RecordWebhookDuration(duration)
//     }
// }
