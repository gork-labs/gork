// Package fiber provides Fiber framework adapter for the gork toolkit.
package fiber

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gork-labs/gork/pkg/api"
	"gopkg.in/yaml.v3"
)

// YAMLMarshaler allows dependency injection for testing.
type YAMLMarshaler func(interface{}) ([]byte, error)

var defaultYAMLMarshaler YAMLMarshaler = yaml.Marshal

// DocsHandler creates a Fiber handler for serving OpenAPI documentation.
// This is a helper function for custom documentation serving if needed.
func DocsHandler(spec *api.OpenAPISpec, config api.DocsConfig) fiber.Handler {
	return DocsHandlerWithMarshaler(spec, config, defaultYAMLMarshaler)
}

// DocsHandlerWithMarshaler allows injecting custom YAML marshaler for testing.
func DocsHandlerWithMarshaler(spec *api.OpenAPISpec, config api.DocsConfig, marshaler YAMLMarshaler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		switch path {
		case config.OpenAPIPath:
			// Serve the OpenAPI JSON specification
			c.Set("Content-Type", "application/json")
			return c.JSON(spec)

		case config.OpenAPIPath + ".yaml":
			// Serve the OpenAPI YAML specification
			yamlData, err := marshaler(spec)
			if err != nil {
				return c.Status(http.StatusInternalServerError).SendString("Error generating YAML")
			}
			c.Set("Content-Type", "application/yaml")
			return c.SendString(string(yamlData))

		default:
			return c.Status(http.StatusNotFound).SendString("Not Found")
		}
	}
}
