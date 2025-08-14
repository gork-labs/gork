package foo

// WebhookRequest has the name expected by the generator so the provider
// can be derived from package path ending with /webhooks/foo.
type WebhookRequest struct {
	Headers struct {
		Sig string
	}
	Body []byte
}

// WebhookResponse and WebhookErrorResponse to exercise provider-specific naming.
type WebhookResponse struct{ Body struct{ Received bool } }
type WebhookErrorResponse struct {
	Body struct {
		Received bool
		Error    string
	}
}
