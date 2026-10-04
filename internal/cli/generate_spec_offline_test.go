package cli

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestGenerateSpecCallsValidatorOnlyWhenValidateOnline(t *testing.T) {
	originalClient := defaultValidatorClient
	defaultValidatorClient = &MockValidatorClient{CallError: errors.New("validator called")}
	defer func() { defaultValidatorClient = originalClient }()

	config := &GenerateConfig{OutputPath: filepath.Join(t.TempDir(), "openapi.json"), Title: "API", Version: "1.0.0"}
	if err := GenerateSpec(config); err != nil {
		t.Fatalf("expected no validator call by default, got %v", err)
	}

	config.ValidateOnline = true
	if err := GenerateSpec(config); err == nil {
		t.Error("expected the validator call with ValidateOnline")
	}
}
