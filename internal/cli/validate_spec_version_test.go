package cli

import (
	"errors"
	"testing"

	"github.com/gork-labs/gork/pkg/api"
)

func TestValidateSpecSkipsOpenAPI32(t *testing.T) {
	client := &MockValidatorClient{CallError: errors.New("validator called")}

	if err := validateSpecWithClient(&api.OpenAPISpec{OpenAPI: "3.2.0"}, client); err != nil {
		t.Errorf("expected no validator call for OpenAPI 3.2.0, got %v", err)
	}
	if err := validateSpecWithClient(&api.OpenAPISpec{OpenAPI: "3.1.0"}, client); err == nil {
		t.Error("expected the validator call for OpenAPI 3.1.0")
	}
}
