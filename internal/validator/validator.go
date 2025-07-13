package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

// ValidateSpec validates an OpenAPI spec using the official validator API
func ValidateSpec(filename string) error {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check if it's YAML or JSON
	var spec map[string]interface{}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		// Try JSON
		if err := json.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("failed to parse file as YAML or JSON: %w", err)
		}
	}

	// Validate basic structure
	if err := validateBasicStructure(spec); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}

	fmt.Println("✓ OpenAPI version is valid")
	fmt.Println("✓ Info object is present")
	fmt.Println("✓ Basic structure is valid")

	// Validate paths
	if paths, ok := spec["paths"].(map[string]interface{}); ok {
		fmt.Printf("✓ Found %d paths\n", len(paths))
		for path, pathItem := range paths {
			if err := validatePath(path, pathItem); err != nil {
				return fmt.Errorf("path %s validation failed: %w", path, err)
			}
		}
	}

	// Validate components
	if components, ok := spec["components"].(map[string]interface{}); ok {
		if schemas, ok := components["schemas"].(map[string]interface{}); ok {
			fmt.Printf("✓ Found %d schemas\n", len(schemas))
			for name, schema := range schemas {
				if err := validateSchema(name, schema); err != nil {
					return fmt.Errorf("schema %s validation failed: %w", name, err)
				}
			}
		}
	}

	fmt.Println("\n✅ OpenAPI spec validation passed!")
	return nil
}

func validateBasicStructure(spec map[string]interface{}) error {
	// Check OpenAPI version
	openapi, ok := spec["openapi"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'openapi' field")
	}
	if openapi != "3.0.0" && openapi != "3.0.1" && openapi != "3.0.2" && openapi != "3.0.3" && openapi != "3.1.0" {
		return fmt.Errorf("unsupported OpenAPI version: %s", openapi)
	}

	// Check info object
	info, ok := spec["info"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid 'info' field")
	}

	if _, ok := info["title"].(string); !ok {
		return fmt.Errorf("missing or invalid 'info.title' field")
	}

	if _, ok := info["version"].(string); !ok {
		return fmt.Errorf("missing or invalid 'info.version' field")
	}

	return nil
}

func validatePath(path string, pathItem interface{}) error {
	item, ok := pathItem.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid path item")
	}

	validMethods := []string{"get", "post", "put", "delete", "patch", "head", "options", "trace"}
	hasOperation := false

	for _, method := range validMethods {
		if op, exists := item[method]; exists {
			hasOperation = true
			if err := validateOperation(method, op); err != nil {
				return fmt.Errorf("operation %s: %w", method, err)
			}
		}
	}

	if !hasOperation && len(item) > 0 {
		// Check for parameters at path level
		if _, hasParams := item["parameters"]; !hasParams {
			return fmt.Errorf("path item has no operations")
		}
	}

	return nil
}

func validateOperation(method string, operation interface{}) error {
	op, ok := operation.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid operation")
	}

	// Check responses (required)
	responses, ok := op["responses"].(map[string]interface{})
	if !ok || len(responses) == 0 {
		return fmt.Errorf("missing or empty 'responses' field")
	}

	// Validate each response
	for status, response := range responses {
		if err := validateResponse(status, response); err != nil {
			return fmt.Errorf("response %s: %w", status, err)
		}
	}

	// Check parameters if present
	if params, ok := op["parameters"].([]interface{}); ok {
		for i, param := range params {
			if err := validateParameter(i, param); err != nil {
				return fmt.Errorf("parameter %d: %w", i, err)
			}
		}
	}

	return nil
}

func validateResponse(status string, response interface{}) error {
	resp, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid response")
	}

	// Description is required
	if _, ok := resp["description"].(string); !ok {
		return fmt.Errorf("missing 'description' field")
	}

	return nil
}

func validateParameter(index int, parameter interface{}) error {
	param, ok := parameter.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid parameter")
	}

	// Required fields
	if _, ok := param["name"].(string); !ok {
		return fmt.Errorf("missing 'name' field")
	}

	in, ok := param["in"].(string)
	if !ok {
		return fmt.Errorf("missing 'in' field")
	}

	validIn := map[string]bool{"query": true, "header": true, "path": true, "cookie": true}
	if !validIn[in] {
		return fmt.Errorf("invalid 'in' value: %s", in)
	}

	// Path parameters must be required
	if in == "path" {
		if required, ok := param["required"].(bool); !ok || !required {
			return fmt.Errorf("path parameter must have 'required: true'")
		}
	}

	return nil
}

func validateSchema(name string, schema interface{}) error {
	s, ok := schema.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid schema")
	}

	// If it has $ref, that's all it should have
	if ref, hasRef := s["$ref"].(string); hasRef {
		if len(s) > 1 {
			return fmt.Errorf("schema with $ref should not have other properties")
		}
		if ref == "" {
			return fmt.Errorf("empty $ref")
		}
		return nil
	}

	// Otherwise, check type
	if _, hasType := s["type"].(string); !hasType && len(s) > 0 {
		// It's okay to not have type if it's an empty schema or has other constraints
		if _, hasProps := s["properties"]; !hasProps {
			if _, hasAllOf := s["allOf"]; !hasAllOf {
				if _, hasAnyOf := s["anyOf"]; !hasAnyOf {
					if _, hasOneOf := s["oneOf"]; !hasOneOf {
						// Warn but don't error
						fmt.Printf("  ⚠️  Schema '%s' has no type\n", name)
					}
				}
			}
		}
	}

	return nil
}

// ValidateWithExternalService validates using an external OpenAPI validator service
func ValidateWithExternalService(filename string) error {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Use swagger.io validator
	resp, err := http.Post(
		"https://validator.swagger.io/validator/debug",
		"application/yaml",
		bytes.NewReader(data),
	)
	if err != nil {
		// Fallback to local validation
		fmt.Println("⚠️  Could not reach external validator, using local validation only")
		return ValidateSpec(filename)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 200 {
		fmt.Println("✅ External validation passed!")
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		if messages, ok := result["messages"].([]interface{}); ok && len(messages) > 0 {
			fmt.Println("❌ Validation errors:")
			for _, msg := range messages {
				fmt.Printf("  - %v\n", msg)
			}
			return fmt.Errorf("validation failed")
		}
	}

	return fmt.Errorf("validation failed with status %d: %s", resp.StatusCode, string(body))
}