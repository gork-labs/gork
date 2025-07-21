package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// DocsConfig holds basic configuration for documentation UI.
// This is intentionally minimal to keep the initial implementation lightweight.
// Additional fields can be added later without breaking callers.
type DocsConfig struct {
	// Title shown in the browser tab / UI header.
	Title string
	// OpenAPIPath under which the generated OpenAPI document is served. Must start with "/".
	// Defaults to "/openapi.json".
	OpenAPIPath string
	// SpecFile points to a pre-generated OpenAPI 3.1 JSON (or YAML) file on
	// disk. When set, DocsRoute will load the specification from this file
	// once during server start-up and serve it at runtime instead of
	// generating a fresh spec on every request. This allows enrichment from
	// build-time tooling (e.g. doc comments).
	//
	// If the file cannot be read or parsed the router falls back to runtime
	// generation so that documentation is still available albeit without the
	// additional metadata.
	SpecFile string
	// UITemplate holds the HTML page used to render the documentation UI. The
	// template must contain the placeholders {{.Title}}, {{.OpenAPIPath}} and
	// {{.BasePath}} which are replaced at runtime. Predefined templates are
	// provided – StoplightUITemplate (default), SwaggerUITemplate and
	// RedocUITemplate – but callers can supply any custom template string.
	UITemplate UITemplate
}

// UITemplate represents an HTML page template for serving API documentation.
// It is defined as a distinct type to avoid accidental mix-ups with regular
// strings and to make the purpose explicit.
type UITemplate string

// defaultDocsConfig returns a DocsConfig populated with sensible defaults.
func defaultDocsConfig() DocsConfig {
	return DocsConfig{
		Title:       "API Documentation",
		OpenAPIPath: "/openapi.json",
		SpecFile:    "",
		UITemplate:  StoplightUITemplate,
	}
}

// DocsRoute registers (1) an endpoint that serves the generated OpenAPI spec in
// JSON format and (2) a catch-all route that serves a minimal HTML page loading
// the chosen documentation UI from a public CDN. The implementation purposefully
// trades customisability for a small footprint so that users can benefit from
// documentation immediately while we iterate on a more sophisticated solution.
func (r *TypedRouter[T]) DocsRoute(path string, cfg ...DocsConfig) {
	// Prepare configuration.
	conf := defaultDocsConfig()
	if len(cfg) > 0 {
		conf = cfg[0]
		// Apply defaults for zero values so that callers can omit fields.
		if conf.Title == "" {
			conf.Title = "API Documentation"
		}
		if conf.OpenAPIPath == "" {
			conf.OpenAPIPath = "/openapi.json"
		}
		if conf.UITemplate == "" {
			conf.UITemplate = StoplightUITemplate
		}
	}

	// Normalise docs base path to always end with "/*".
	if !strings.HasSuffix(path, "/*") {
		if strings.HasSuffix(path, "/") {
			path += "*"
		} else {
			path += "/*"
		}
	}
	basePath := strings.TrimSuffix(path, "/*")

	// ---------------------------------------------------------------------
	// 1) Register the OpenAPI spec endpoint via the strongly-typed helpers so
	//    that the route appears in the registry like any other API route. We
	//    mount it under the docs base path to keep all documentation assets
	//    grouped together (e.g. /docs/openapi.json).
	// ---------------------------------------------------------------------

	// Build final OpenAPI endpoint path (e.g. /docs/openapi.json).
	openapiPath := basePath + conf.OpenAPIPath

	// Ensure HTML template receives correct absolute path.
	confWithFullPath := conf
	confWithFullPath.OpenAPIPath = openapiPath

	// Prepare static spec if SpecFile is provided.
	var staticSpec *OpenAPISpec
	if conf.SpecFile != "" {
		if b, err := os.ReadFile(conf.SpecFile); err == nil {
			var tmp OpenAPISpec
			// Try JSON first.
			if err := json.Unmarshal(b, &tmp); err == nil {
				staticSpec = &tmp
			} else if yamlErr := yaml.Unmarshal(b, &tmp); yamlErr == nil {
				staticSpec = &tmp
			}
		}
	}

	type emptyReq struct{}

	r.Get(openapiPath, func(ctx context.Context, _ emptyReq) (*OpenAPISpec, error) {
		if staticSpec != nil {
			return staticSpec, nil
		}
		spec := GenerateOpenAPI(r.registry)
		return spec, nil
	})

	// ---------------------------------------------------------------------
	// 2) Register the UI route directly with the underlying router because the
	//    generic handler does not follow the usual request/response contract.
	// ---------------------------------------------------------------------
	if r.registerFn != nil {
		r.registerFn(http.MethodGet, basePath+"/*", r.createDocsHandler(basePath, confWithFullPath), nil)
	}
}

// createDocsHandler returns an http.HandlerFunc that serves a pre-rendered HTML
// page loading one of the supported documentation UIs from a CDN.
func (r *TypedRouter[T]) createDocsHandler(basePath string, cfg DocsConfig) http.HandlerFunc {
	// Use the provided UI template.
	htmlTmpl := string(cfg.UITemplate)

	// Inject runtime values.
	replacer := strings.NewReplacer(
		"{{.Title}}", cfg.Title,
		"{{.OpenAPIPath}}", cfg.OpenAPIPath,
		"{{.BasePath}}", basePath,
	)
	html := replacer.Replace(htmlTmpl)

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// We intentionally ignore the write error – serving docs must never
		// crash the application.
		_, _ = w.Write([]byte(html))
	}
}

// -----------------------------------------------------------------------------
// Built-in UI templates
// -----------------------------------------------------------------------------

// StoplightUITemplate is the default UI powered by Stoplight Elements.
const StoplightUITemplate UITemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css">
</head>
<body style="margin:0;padding:0;">
    <elements-api apiDescriptionUrl="{{.OpenAPIPath}}" router="hash" layout="sidebar" />
</body>
</html>`

// SwaggerUITemplate exposes the popular Swagger UI.
const SwaggerUITemplate UITemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "{{.OpenAPIPath}}",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.SwaggerUIStandalonePreset
                ],
                layout: "BaseLayout"
            });
        }
    </script>
</body>
</html>`

// RedocUITemplate uses Redoc to render the documentation.
const RedocUITemplate UITemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
    <redoc spec-url="{{.OpenAPIPath}}"></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`
