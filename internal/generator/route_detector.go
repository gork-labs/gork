package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// RouteDetector detects route registrations in Go source files
type RouteDetector struct {
	fileSet *token.FileSet
}

// NewRouteDetector creates a new route detector
func NewRouteDetector() *RouteDetector {
	return &RouteDetector{
		fileSet: token.NewFileSet(),
	}
}

// DetectRoutesFromFile detects routes from a Go source file
func (rd *RouteDetector) DetectRoutesFromFile(filename string) ([]ExtractedRoute, error) {
	src, err := parser.ParseFile(rd.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}
	
	var routes []ExtractedRoute
	
	ast.Inspect(src, func(node ast.Node) bool {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		
		// Try different router patterns
		if route := rd.detectGinRoute(callExpr); route != nil {
			routes = append(routes, *route)
		} else if route := rd.detectEchoRoute(callExpr); route != nil {
			routes = append(routes, *route)
		} else if route := rd.detectGorillaMuxRoute(callExpr); route != nil {
			routes = append(routes, *route)
		} else if route := rd.detectChiRoute(callExpr); route != nil {
			routes = append(routes, *route)
		} else if route := rd.detectFiberRoute(callExpr); route != nil {
			routes = append(routes, *route)
		} else if route := rd.detectStdLibRoute(callExpr); route != nil {
			routes = append(routes, *route)
		}
		
		return true
	})
	
	return routes, nil
}

// detectGinRoute detects Gin router patterns: router.GET("/path", handler)
func (rd *RouteDetector) detectGinRoute(call *ast.CallExpr) *ExtractedRoute {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) < 2 {
		return nil
	}
	
	method := strings.ToUpper(sel.Sel.Name)
	if !isHTTPMethod(method) {
		return nil
	}
	
	// Extract path
	path := extractStringLiteral(call.Args[0])
	if path == "" {
		return nil
	}
	
	// Extract handler
	handler := extractHandlerName(call.Args[1])
	if handler == "" {
		return nil
	}
	
	// Convert Gin path params (:id) to OpenAPI format ({id})
	path = convertPathParams(path, "gin")
	
	return &ExtractedRoute{
		Method:      method,
		Path:        path,
		HandlerName: handler,
	}
}

// detectEchoRoute detects Echo router patterns: e.GET("/path", handler)
func (rd *RouteDetector) detectEchoRoute(call *ast.CallExpr) *ExtractedRoute {
	// Same pattern as Gin
	return rd.detectGinRoute(call)
}

// detectGorillaMuxRoute detects Gorilla Mux patterns: router.HandleFunc("/path", handler).Methods("GET")
func (rd *RouteDetector) detectGorillaMuxRoute(call *ast.CallExpr) *ExtractedRoute {
	// Check for .Methods() call
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Methods" || len(call.Args) < 1 {
		return nil
	}
	
	// Get the method
	method := extractStringLiteral(call.Args[0])
	if method == "" || !isHTTPMethod(method) {
		return nil
	}
	
	// Check if X is a call to HandleFunc
	handleCall, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return nil
	}
	
	handleSel, ok := handleCall.Fun.(*ast.SelectorExpr)
	if !ok || handleSel.Sel.Name != "HandleFunc" || len(handleCall.Args) < 2 {
		return nil
	}
	
	// Extract path and handler
	path := extractStringLiteral(handleCall.Args[0])
	handler := extractHandlerName(handleCall.Args[1])
	
	if path == "" || handler == "" {
		return nil
	}
	
	// Gorilla Mux uses {param} format
	path = convertPathParams(path, "gorilla")
	
	return &ExtractedRoute{
		Method:      strings.ToUpper(method),
		Path:        path,
		HandlerName: handler,
	}
}

// detectChiRoute detects Chi router patterns: r.Get("/path", handler)
func (rd *RouteDetector) detectChiRoute(call *ast.CallExpr) *ExtractedRoute {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) < 2 {
		return nil
	}
	
	// Chi uses Get, Post, etc. (capitalized)
	method := strings.ToUpper(sel.Sel.Name)
	if !isHTTPMethod(method) {
		return nil
	}
	
	// Extract path and handler
	path := extractStringLiteral(call.Args[0])
	handler := extractHandlerName(call.Args[1])
	
	if path == "" || handler == "" {
		return nil
	}
	
	// Chi uses {param} format
	path = convertPathParams(path, "chi")
	
	return &ExtractedRoute{
		Method:      method,
		Path:        path,
		HandlerName: handler,
	}
}

// detectFiberRoute detects Fiber router patterns: app.Get("/path", handler)
func (rd *RouteDetector) detectFiberRoute(call *ast.CallExpr) *ExtractedRoute {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) < 2 {
		return nil
	}
	
	// Fiber uses Get, Post, etc. (capitalized)
	method := strings.ToUpper(sel.Sel.Name)
	if !isHTTPMethod(method) {
		return nil
	}
	
	// Extract path and handler
	path := extractStringLiteral(call.Args[0])
	handler := extractHandlerName(call.Args[1])
	
	if path == "" || handler == "" {
		return nil
	}
	
	// Fiber uses :param format
	path = convertPathParams(path, "fiber")
	
	return &ExtractedRoute{
		Method:      method,
		Path:        path,
		HandlerName: handler,
	}
}

// detectStdLibRoute detects standard library patterns: http.HandleFunc("/path", handler) or mux.HandleFunc("METHOD /path", api.HandlerFunc(...))
func (rd *RouteDetector) detectStdLibRoute(call *ast.CallExpr) *ExtractedRoute {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "HandleFunc" || len(call.Args) < 2 {
		return nil
	}
	
	// Check if it's http.HandleFunc or mux.HandleFunc
	var isHTTPPackage bool
	if pkg, ok := sel.X.(*ast.Ident); ok {
		isHTTPPackage = pkg.Name == "http" || strings.HasSuffix(pkg.Name, "mux")
	}
	
	if !isHTTPPackage {
		return nil
	}
	
	// Extract path - may contain method prefix like "POST /api/v1/login"
	pathStr := extractStringLiteral(call.Args[0])
	if pathStr == "" {
		return nil
	}
	
	// Parse method and path
	method := ""
	path := pathStr
	parts := strings.SplitN(pathStr, " ", 2)
	if len(parts) == 2 {
		possibleMethod := strings.ToUpper(parts[0])
		if isHTTPMethod(possibleMethod) {
			method = possibleMethod
			path = parts[1]
		}
	}
	
	// Extract handler - may be wrapped with api.HandlerFunc(...)
	handler, tags, security := rd.extractWrappedHandlerWithTags(call.Args[1])
	if handler == "" {
		handler = extractHandlerName(call.Args[1])
	}
	
	if handler == "" {
		return nil
	}
	
	// If no method specified, will be inferred from handler name
	return &ExtractedRoute{
		Method:      method,
		Path:        path,
		HandlerName: handler,
		Tags:        tags,
		Security:    security,
	}
}

// extractWrappedHandler extracts handler name from api.HandlerFunc(handlers.Login, ...)
func (rd *RouteDetector) extractWrappedHandler(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	
	// Check if it's api.HandlerFunc(...)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "HandlerFunc" {
		return ""
	}
	
	// Check the package is "api"
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "api" {
		return ""
	}
	
	// Extract the first argument which should be the handler function
	if len(call.Args) == 0 {
		return ""
	}
	
	return extractHandlerName(call.Args[0])
}

// extractWrappedHandlerWithTags extracts handler name and tags from api.HandlerFunc(handlers.Login, api.WithTags("auth"))
func (rd *RouteDetector) extractWrappedHandlerWithTags(expr ast.Expr) (handler string, tags []string, security []SecurityRequirement) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", nil, nil
	}
	
	// Check if it's api.HandlerFunc(...)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "HandlerFunc" {
		return "", nil, nil
	}
	
	// Check the package is "api"
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "api" {
		return "", nil, nil
	}
	
	// Extract the first argument which should be the handler function
	if len(call.Args) == 0 {
		return "", nil, nil
	}
	
	handler = extractHandlerName(call.Args[0])
	
	// Look for api.WithTags(...) and auth methods in the remaining arguments
	for i := 1; i < len(call.Args); i++ {
		if tagList := rd.extractWithTags(call.Args[i]); tagList != nil {
			tags = append(tags, tagList...)
		} else if sec := rd.extractSecurity(call.Args[i]); sec != nil {
			security = append(security, *sec)
		}
	}
	
	return handler, tags, security
}

// extractWithTags extracts tags from api.WithTags("tag1", "tag2", ...)
func (rd *RouteDetector) extractWithTags(expr ast.Expr) []string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}
	
	// Check if it's api.WithTags(...)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "WithTags" {
		return nil
	}
	
	// Check the package is "api"
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "api" {
		return nil
	}
	
	// Extract all string arguments as tags
	var tags []string
	for _, arg := range call.Args {
		if tag := extractStringLiteral(arg); tag != "" {
			tags = append(tags, tag)
		}
	}
	
	return tags
}

// extractSecurity extracts security requirements from api.WithBearerTokenAuth(), api.WithAPIKeyAuth(), or api.WithBasicAuth()
func (rd *RouteDetector) extractSecurity(expr ast.Expr) *SecurityRequirement {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}
	
	// Check if it's an api auth method
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	
	// Check the package is "api"
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "api" {
		return nil
	}
	
	// Determine the security type based on the method name
	switch sel.Sel.Name {
	case "WithBasicAuth":
		return &SecurityRequirement{
			Type: "basic",
		}
	case "WithBearerTokenAuth":
		// Extract scopes if provided
		var scopes []string
		for _, arg := range call.Args {
			if scope := extractStringLiteral(arg); scope != "" {
				scopes = append(scopes, scope)
			}
		}
		return &SecurityRequirement{
			Type:   "bearer",
			Scopes: scopes,
		}
	case "WithAPIKeyAuth":
		return &SecurityRequirement{
			Type: "apiKey",
		}
	default:
		return nil
	}
}

// Helper functions

func isHTTPMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		return true
	default:
		return false
	}
}

func extractStringLiteral(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	// Remove quotes
	return strings.Trim(lit.Value, `"`)
}

func extractHandlerName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		// Handle package.Handler case
		if pkg, ok := e.X.(*ast.Ident); ok {
			return pkg.Name + "." + e.Sel.Name
		}
	case *ast.FuncLit:
		// Anonymous function
		return "anonymous"
	}
	return ""
}

// convertPathParams converts framework-specific path parameters to OpenAPI format
func convertPathParams(path, framework string) string {
	switch framework {
	case "gin", "echo", "fiber":
		// Convert :param to {param}
		re := regexp.MustCompile(`:(\w+)`)
		return re.ReplaceAllString(path, "{$1}")
	case "gorilla", "chi":
		// Already uses {param} format
		return path
	default:
		return path
	}
}

// ExtractPathParameters extracts parameter names from an OpenAPI-style path
func ExtractPathParameters(path string) []string {
	params := []string{}
	re := regexp.MustCompile(`\{(\w+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

// InferMethodFromHandler infers HTTP method from handler name
func InferMethodFromHandler(handlerName string) string {
	name := strings.ToLower(handlerName)
	
	// Handle qualified names like "handlers.GetUser"
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	
	// Remove common prefixes
	name = strings.TrimPrefix(name, "handle")
	name = strings.TrimPrefix(name, "handler")
	
	switch {
	case strings.HasPrefix(name, "get") || strings.HasPrefix(name, "list") || strings.HasPrefix(name, "fetch"):
		return "GET"
	case strings.HasPrefix(name, "create") || strings.HasPrefix(name, "post") || strings.HasPrefix(name, "add"):
		return "POST"
	case strings.HasPrefix(name, "update") || strings.HasPrefix(name, "put") || strings.HasPrefix(name, "edit"):
		return "PUT"
	case strings.HasPrefix(name, "patch") || strings.HasPrefix(name, "modify"):
		return "PATCH"
	case strings.HasPrefix(name, "delete") || strings.HasPrefix(name, "remove"):
		return "DELETE"
	default:
		// Default to POST for unknown patterns
		return "POST"
	}
}