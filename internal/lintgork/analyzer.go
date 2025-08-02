// Package lintgork provides an OpenAPI tag linter for Gork projects.
package lintgork

import (
	"go/ast"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the gork OpenAPI tag linter.
var Analyzer = &analysis.Analyzer{
	Name: "lintgork",
	Doc:  "checks struct openapi tags for correct format",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		inspectStructFields(file, pass)
	}
	collectPathParamDiagnostics(pass)
	checkDuplicateDiscriminatorValues(pass)
	return nil, nil
}

func inspectStructFields(file *ast.File, pass *analysis.Pass) {
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		validateStructFields(st, pass)
		return false
	})
}

func validateStructFields(st *ast.StructType, pass *analysis.Pass) {
	for _, fld := range st.Fields.List {
		if fld.Tag == nil {
			continue
		}
		tagVal, err := strconv.Unquote(fld.Tag.Value)
		if err != nil {
			continue
		}
		validateOpenAPITag(tagVal, fld, pass)
	}
}

func validateOpenAPITag(tagVal string, fld *ast.Field, pass *analysis.Pass) {
	tag := reflect.StructTag(tagVal)
	if openapiTag, ok := tag.Lookup("openapi"); ok {
		if !validOpenAPITag(openapiTag) {
			pass.Reportf(fld.Tag.Pos(), "invalid openapi tag %q: expect '<name>,in=<query|path|header>' or 'discriminator=<value>'", openapiTag)
		}
	}
}

func checkDuplicateDiscriminatorValues(pass *analysis.Pass) {
	discSeen := map[string]ast.Node{}
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			checkStructFieldsForDuplicates(st, discSeen, pass)
			return false
		})
	}
}

func checkStructFieldsForDuplicates(st *ast.StructType, discSeen map[string]ast.Node, pass *analysis.Pass) {
	for _, fld := range st.Fields.List {
		if fld.Tag == nil {
			continue
		}
		tagVal, err := strconv.Unquote(fld.Tag.Value)
		if err != nil {
			continue
		}
		checkDiscriminatorTag(tagVal, fld, discSeen, pass)
	}
}

func checkDiscriminatorTag(tagVal string, fld *ast.Field, discSeen map[string]ast.Node, pass *analysis.Pass) {
	tag := reflect.StructTag(tagVal)
	if openapiTag, ok := tag.Lookup("openapi"); ok {
		if val, ok := parseDiscriminator(openapiTag); ok {
			if prev, dup := discSeen[val]; dup {
				pass.Reportf(fld.Tag.Pos(), "duplicate discriminator value %q also used at %s", val, pass.Fset.Position(prev.Pos()))
			} else {
				discSeen[val] = fld.Tag
			}
		}
	}
}

func validOpenAPITag(tag string) bool {
	if tag == "" {
		return false
	}
	// discriminator=val pattern
	if strings.HasPrefix(tag, "discriminator=") {
		val := strings.TrimPrefix(tag, "discriminator=")
		return val != ""
	}
	parts := strings.Split(tag, ",")
	if len(parts) < 2 {
		return false
	}
	name := strings.TrimSpace(parts[0])
	loc := ""
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "in=") {
			loc = strings.TrimPrefix(p, "in=")
		}
	}
	if name == "" || loc == "" {
		return false
	}
	switch loc {
	case "query", "path", "header":
		return true
	}
	return false
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
