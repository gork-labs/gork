package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v76"
)

func TestProviderInfo_And_Marker(t *testing.T) {
	h := &Handler{}
	info := h.ProviderInfo()
	if info.Name == "" || info.Website == "" || info.DocsURL == "" {
		t.Fatalf("expected provider info fields to be populated: %#v", info)
	}

	// Ensure WebhookRequest implements the api.WebhookRequest marker method
	var req WebhookRequest
	// Call the marker method and assert it exists via reflection
	// (compile-time ensures presence; this line just executes it)
	req.WebhookRequest()

	// Also verify via reflection the method exists
	m, ok := reflect.TypeOf(req).MethodByName("WebhookRequest")
	if !ok || m.Type.NumIn() != 1 || m.Type.NumOut() != 0 {
		t.Fatalf("expected WebhookRequest marker method to exist with no outputs, got: %+v", m)
	}
}

func computeStripeSig(secret string, ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.%s", ts, string(body))))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestNewHandler_Responses_And_EventTypes(t *testing.T) {
	h := NewHandler("whsec_test").(*Handler)
	// Success response
	if resp, ok := h.SuccessResponse().(WebhookResponse); !ok || !resp.Body.Received {
		t.Fatalf("unexpected success response: %#v", resp)
	}
	// Error response
	errResp, ok := h.ErrorResponse(fmt.Errorf("boom")).(WebhookErrorResponse)
	if !ok || errResp.Body.Received || errResp.Body.Error == "" {
		t.Fatalf("unexpected error response: %#v", errResp)
	}
	// Default event types contain a known item
	if !h.IsValidEventType("payment_intent.succeeded") {
		t.Fatalf("expected default valid event type")
	}
	// Custom event types override
	h2 := NewHandler("whsec_test", "a.b", "c.d").(*Handler)
	if !h2.IsValidEventType("a.b") || h2.IsValidEventType("payment_intent.succeeded") {
		t.Fatalf("custom event types not respected")
	}
	if got := h2.GetValidEventTypes(); len(got) != 2 || got[0] != "a.b" || got[1] != "c.d" {
		t.Fatalf("unexpected custom event types: %v", got)
	}
}

func TestHasPrefix(t *testing.T) {
	if !hasPrefix("invoice.created", "invoice.") {
		t.Fatal("expected true")
	}
	if hasPrefix("charge.succeeded", "invoice.") {
		t.Fatal("expected false")
	}
}

func TestStripeWebhookRequest_Validate(t *testing.T) {
	var r WebhookRequest
	// Missing signature
	if err := r.Validate(context.Background()); err == nil {
		t.Fatal("expected error for missing signature")
	}
	// Empty body
	r.Headers.StripeSignature = "x"
	if err := r.Validate(context.Background()); err == nil {
		t.Fatal("expected error for empty body")
	}
	// Valid minimal
	r.Body = []byte("{}")
	if err := r.Validate(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewWebhookContext(t *testing.T) {
	ctx := NewWebhookContext("secret")
	if got, ok := ctx.Value(webhookSecretKey).(string); !ok || got != "secret" {
		t.Fatalf("unexpected context value: %v", got)
	}
}

func TestHandler_ParseRequest_Success_And_InvalidSignature(t *testing.T) {
	secret := "whsec_123"
	// Build a minimal Stripe event payload
	pi := &stripe.PaymentIntent{ID: "pi_123"}
	raw, _ := json.Marshal(pi)
	sev := stripe.Event{
		ID:   "evt_123",
		Type: stripe.EventTypePaymentIntentSucceeded,
		Data: &stripe.EventData{Raw: raw},
	}
	body, _ := json.Marshal(sev)
	ts := time.Now().Unix()
	sig := computeStripeSig(secret, ts, body)
	sigHeader := fmt.Sprintf("t=%d,v1=%s", ts, sig)

	h := NewHandler(secret).(*Handler)
	// Widen tolerance to be safe in CI
	h.tolerance = 30 * time.Minute
	event, err := h.ParseRequest(WebhookRequest{Body: body, Headers: struct {
		StripeSignature string "gork:\"Stripe-Signature\" validate:\"required\""
		ContentType     string "gork:\"Content-Type\""
	}{StripeSignature: sigHeader}})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if event.Type != string(stripe.EventTypePaymentIntentSucceeded) {
		t.Fatalf("unexpected type: %s", event.Type)
	}
	if event.ProviderObject == nil {
		t.Fatalf("expected provider object")
	}

	// Invalid signature should fail
	_, err = h.ParseRequest(WebhookRequest{Body: body, Headers: struct {
		StripeSignature string "gork:\"Stripe-Signature\" validate:\"required\""
		ContentType     string "gork:\"Content-Type\""
	}{StripeSignature: "t=1,v1=deadbeef"}})
	if err == nil {
		t.Fatalf("expected error for invalid signature")
	}
}

func TestHandler_ParseRequest_MappingBranches(t *testing.T) {
	secret := "whsec_branch"
	h := NewHandler(secret).(*Handler)
	h.tolerance = 30 * time.Minute

	// helper to sign body
	sign := func(ts int64, body []byte) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(fmt.Sprintf("%d.%s", ts, string(body))))
		return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
	}

	t.Run("subscription mapping with metadata", func(t *testing.T) {
		sub := &stripe.Subscription{ID: "sub_123", Metadata: map[string]string{"user_id": "u1"}}
		raw, _ := json.Marshal(sub)
		sev := stripe.Event{ID: "evt_sub", Type: stripe.EventTypeCustomerSubscriptionCreated, Data: &stripe.EventData{Raw: raw}}
		body, _ := json.Marshal(sev)
		ev, err := h.ParseRequest(WebhookRequest{Body: body, Headers: struct {
			StripeSignature string "gork:\"Stripe-Signature\" validate:\"required\""
			ContentType     string "gork:\"Content-Type\""
		}{StripeSignature: sign(time.Now().Unix(), body)}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Type != string(stripe.EventTypeCustomerSubscriptionCreated) {
			t.Fatalf("unexpected type: %s", ev.Type)
		}
		if _, ok := ev.ProviderObject.(*stripe.Subscription); !ok {
			t.Fatalf("expected subscription provider object")
		}
		if len(ev.UserMetaJSON) == 0 {
			t.Fatalf("expected user metadata JSON")
		}
	})

	t.Run("invoice mapping with metadata", func(t *testing.T) {
		inv := &stripe.Invoice{ID: "in_123", Metadata: map[string]string{"user_id": "u2"}}
		raw, _ := json.Marshal(inv)
		sev := stripe.Event{ID: "evt_inv", Type: stripe.EventTypeInvoiceCreated, Data: &stripe.EventData{Raw: raw}}
		body, _ := json.Marshal(sev)
		ev, err := h.ParseRequest(WebhookRequest{Body: body, Headers: struct {
			StripeSignature string "gork:\"Stripe-Signature\" validate:\"required\""
			ContentType     string "gork:\"Content-Type\""
		}{StripeSignature: sign(time.Now().Unix(), body)}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Type != string(stripe.EventTypeInvoiceCreated) {
			t.Fatalf("unexpected type: %s", ev.Type)
		}
		if _, ok := ev.ProviderObject.(*stripe.Invoice); !ok {
			t.Fatalf("expected invoice provider object")
		}
		if len(ev.UserMetaJSON) == 0 {
			t.Fatalf("expected user metadata JSON")
		}
	})

	t.Run("default mapping returns event", func(t *testing.T) {
		sev := stripe.Event{ID: "evt_unknown", Type: "unknown.event", Data: &stripe.EventData{Raw: []byte(`{}`)}}
		body, _ := json.Marshal(sev)
		ev, err := h.ParseRequest(WebhookRequest{Body: body, Headers: struct {
			StripeSignature string "gork:\"Stripe-Signature\" validate:\"required\""
			ContentType     string "gork:\"Content-Type\""
		}{StripeSignature: sign(time.Now().Unix(), body)}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Type != "unknown.event" {
			t.Fatalf("unexpected type: %s", ev.Type)
		}
		if _, ok := ev.ProviderObject.(*stripe.Event); !ok {
			t.Fatalf("expected raw event provider object")
		}
		if ev.UserMetaJSON != nil {
			t.Fatalf("expected nil user meta for default mapping")
		}
	})

	t.Run("payment_intent mapping with metadata", func(t *testing.T) {
		pi := &stripe.PaymentIntent{ID: "pi_meta", Metadata: map[string]string{"k": "v"}}
		raw, _ := json.Marshal(pi)
		sev := stripe.Event{ID: "evt_pi", Type: stripe.EventTypePaymentIntentSucceeded, Data: &stripe.EventData{Raw: raw}}
		body, _ := json.Marshal(sev)
		ev, err := h.ParseRequest(WebhookRequest{Body: body, Headers: struct {
			StripeSignature string "gork:\"Stripe-Signature\" validate:\"required\""
			ContentType     string "gork:\"Content-Type\""
		}{StripeSignature: sign(time.Now().Unix(), body)}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := ev.ProviderObject.(*stripe.PaymentIntent); !ok {
			t.Fatalf("expected payment intent provider object")
		}
		if len(ev.UserMetaJSON) == 0 {
			t.Fatalf("expected user metadata JSON for payment intent")
		}
	})
}

func TestTypes_UtilityCoverage(t *testing.T) {
	// RequestValidationError.Error
	e := (&RequestValidationError{Errors: []string{"a", "b"}}).Error()
	if e == "" {
		t.Fatalf("expected error string")
	}

	// WebhookRequest marker method
	var req WebhookRequest
	req.WebhookRequest()
}
