package lintgork

import (
	"go/ast"
	"go/token"
	"go/types"
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
		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, fld := range st.Fields.List {
				if fld.Tag == nil {
					continue
				}
				tagVal, err := strconv.Unquote(fld.Tag.Value)
				if err != nil {
					return true
				}
				tag := reflect.StructTag(tagVal)
				if openapiTag, ok := tag.Lookup("openapi"); ok {
					if !validOpenAPITag(openapiTag) {
						pass.Reportf(fld.Tag.Pos(), "invalid openapi tag %q: expect '<name>,in=<query|path|header>' or 'discriminator=<value>'", openapiTag)
					}
				}
			}
			return false
		})
	}
	collectPathParamDiagnostics(pass)

	// Duplicate discriminator value detection
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
			for _, fld := range st.Fields.List {
				if fld.Tag == nil {
					continue
				}
				tagVal, err := strconv.Unquote(fld.Tag.Value)
				if err != nil {
					continue
				}
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
			return false
		})
	}
	return nil, nil
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

var pathVarRegexpLint = regexp.MustCompile(`\{([^{}]+)\}`)

func collectPathParamDiagnostics(pass *analysis.Pass) {
	inspect := func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		// Rough match of router registration methods
		switch sel.Sel.Name {
		case "Get", "Post", "Put", "Patch", "Delete":
		default:
			return true
		}
		// first arg path string literal
		pathLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || pathLit.Kind != token.STRING {
			return true
		}
		pathStr, err := strconv.Unquote(pathLit.Value)
		if err != nil {
			return true
		}
		placeholders := map[string]struct{}{}
		for _, m := range pathVarRegexpLint.FindAllStringSubmatch(pathStr, -1) {
			if len(m) > 1 {
				placeholders[m[1]] = struct{}{}
			}
		}
		if len(placeholders) == 0 {
			return true
		}
		// second arg handler identifier
		handlerIdent, ok := call.Args[1].(*ast.Ident)
		if !ok {
			return true
		}
		obj := pass.TypesInfo.ObjectOf(handlerIdent)
		if obj == nil {
			return true
		}
		sig, ok := obj.Type().(*types.Signature)
		if !ok || sig.Params().Len() < 2 {
			return true
		}
		reqType := sig.Params().At(1).Type()
		named, ok := reqType.(*types.Named)
		if !ok {
			return true
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			return true
		}
		// collect path param names from struct tags
		have := map[string]struct{}{}
		for i := 0; i < st.NumFields(); i++ {
			tag := st.Tag(i)
			name, loc, ok := parseOpenAPITagForLint(tag)
			if ok && loc == "path" {
				have[name] = struct{}{}
			}
		}
		for p := range placeholders {
			if _, ok := have[p]; !ok {
				pass.Reportf(pathLit.Pos(), "path parameter %q in route %s is not declared in request struct %s", p, pathStr, named.Obj().Name())
			}
		}
		// reverse check: struct declares path param not present in url
		for n := range have {
			if _, ok := placeholders[n]; !ok {
				pass.Reportf(pathLit.Pos(), "struct %s declares path param %q not present in route %s", named.Obj().Name(), n, pathStr)
			}
		}
		return true
	}
	for _, f := range pass.Files {
		ast.Inspect(f, inspect)
	}
}

func parseOpenAPITagForLint(tag string) (name, loc string, ok bool) {
	parts := strings.Split(tag, ",")
	if len(parts) < 2 {
		return "", "", false
	}
	name = strings.TrimSpace(parts[0])
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "in=") {
			loc = strings.TrimPrefix(p, "in=")
		}
	}
	if name == "" || loc == "" {
		return "", "", false
	}
	return name, loc, true
}

func parseDiscriminator(tag string) (string, bool) {
	if strings.HasPrefix(tag, "discriminator=") {
		val := strings.TrimPrefix(tag, "discriminator=")
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
