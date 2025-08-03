// Package lintgork provides a Convention Over Configuration linter for Gork projects.
package lintgork

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the gork Convention Over Configuration linter.
var Analyzer = &analysis.Analyzer{
	Name: "lintgork",
	Doc:  "checks struct gork tags and convention compliance",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// Convention Over Configuration linting only
		analyzeConventionStructure(file, pass)
	}
	collectPathParamDiagnostics(pass)
	return nil, nil
}

// ---------------- path param check ----------------

var pathVarRegexpLint = regexp.MustCompile(`\{([^{}]*)\}`)

func collectPathParamDiagnostics(pass *analysis.Pass) {
	inspect := func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		if !isRouterMethodCall(call) {
			return true
		}
		pathStr, placeholders := extractPathPlaceholders(call)
		if pathStr == "" {
			return true
		}
		validatePlaceholders(placeholders, pass, call)
		return true
	}
	for _, file := range pass.Files {
		ast.Inspect(file, inspect)
	}
}

func isRouterMethodCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "Get", "Post", "Put", "Patch", "Delete":
		return true
	}
	return false
}

func extractPathPlaceholders(call *ast.CallExpr) (string, map[string]struct{}) {
	if len(call.Args) == 0 {
		return "", nil
	}
	pathLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || pathLit.Kind != token.STRING {
		return "", nil
	}
	pathStr, err := strconv.Unquote(pathLit.Value)
	if err != nil {
		return "", nil
	}
	placeholders := map[string]struct{}{}
	for _, m := range pathVarRegexpLint.FindAllStringSubmatch(pathStr, -1) {
		if len(m) > 1 {
			placeholders[m[1]] = struct{}{}
		}
	}
	return pathStr, placeholders
}

func validatePlaceholders(placeholders map[string]struct{}, pass *analysis.Pass, call *ast.CallExpr) {
	// Example validation logic: Ensure placeholders are used correctly
	// This is a placeholder for actual validation logic
	for placeholder := range placeholders {
		if placeholder == "" && pass != nil {
			pass.Reportf(call.Pos(), "empty placeholder found in route")
		}
	}
}

func parseDiscriminator(tag string) (string, bool) {
	if strings.HasPrefix(tag, "discriminator=") {
		val := strings.TrimPrefix(tag, "discriminator=")
		// Handle cases where there might be additional comma-separated parts
		if idx := strings.Index(val, ","); idx != -1 {
			val = val[:idx]
		}
		return val, val != ""
	}
	// tag may have multiple segments
	parts := strings.Split(tag, ",")
	for _, p := range parts {
		if strings.HasPrefix(strings.TrimSpace(p), "discriminator=") {
			val := strings.TrimPrefix(strings.TrimSpace(p), "discriminator=")
			return val, val != ""
		}
	}
	return "", false
}
