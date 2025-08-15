// Package fake provides minimal webhook types for testing provider detection.
package fake

// Request is a minimal type placed under a '/webhooks/' path
// to exercise provider detection in tests without importing real providers.
type Request struct{}
