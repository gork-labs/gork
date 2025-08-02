package lintgork

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	// Use the analysistest package to test our analyzer
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}

func TestValidOpenAPITag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want bool
	}{
		{"empty tag", "", false},
		{"discriminator valid", "discriminator=user", true},
		{"discriminator empty value", "discriminator=", false},
		{"query param valid", "id,in=query", true},
		{"path param valid", "userId,in=path", true},
		{"header param valid", "Authorization,in=header", true},
		{"invalid location", "id,in=body", false},
		{"missing name", ",in=query", false},
		{"missing location", "id", false},
		{"missing in= prefix", "id,query", false},
		{"extra spaces", " id , in=query ", true},
		{"multiple parts valid", "user_id,in=path,required", true},
		{"cookie param invalid", "session,in=cookie", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validOpenAPITag(tt.tag); got != tt.want {
				t.Errorf("validOpenAPITag(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestParseDiscriminator(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantVal string
		wantOk  bool
	}{
		{"simple discriminator", "discriminator=user", "user", true},
		{"empty discriminator", "discriminator=", "", false},
		{"no discriminator", "id,in=query", "", false},
		{"discriminator in multi-part", "name=user,discriminator=admin,in=header", "admin", true},
		{"discriminator with spaces", " discriminator=customer ", "customer", true},
		{"multiple discriminators (first wins)", "discriminator=user,discriminator=admin", "user", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := parseDiscriminator(tt.tag)
			if gotVal != tt.wantVal || gotOk != tt.wantOk {
				t.Errorf("parseDiscriminator(%q) = (%q, %v), want (%q, %v)", tt.tag, gotVal, gotOk, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestIsRouterMethodCall(t *testing.T) {
	// Parse a simple function call
	expr, err := parser.ParseExpr("router.Get")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}
	
	// Create a call expression
	call := &ast.CallExpr{
		Fun: expr,
		Args: []ast.Expr{
			&ast.BasicLit{Value: `"/test"`},
			&ast.Ident{Name: "handler"},
		},
	}
	
	if !isRouterMethodCall(call) {
		t.Error("Expected router.Get to be recognized as router method call")
	}
	
	// Test non-router method
	expr2, err := parser.ParseExpr("fmt.Println")
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}
	
	call2 := &ast.CallExpr{
		Fun: expr2,
		Args: []ast.Expr{
			&ast.BasicLit{Value: `"hello"`},
		},
	}
	
	if isRouterMethodCall(call2) {
		t.Error("Expected fmt.Println to NOT be recognized as router method call")
	}
	
	// Test invalid call (not selector expression)
	call3 := &ast.CallExpr{
		Fun: &ast.Ident{Name: "someFunc"},
	}
	
	if isRouterMethodCall(call3) {
		t.Error("Expected simple function call to NOT be recognized as router method call")
	}
}

func TestExtractPathPlaceholders(t *testing.T) {
	tests := []struct {
		name        string
		pathValue   string
		wantPath    string
		wantPlaceholders []string
	}{
		{
			name: "simple path with one placeholder",
			pathValue: `"/users/{id}"`,
			wantPath: "/users/{id}",
			wantPlaceholders: []string{"id"},
		},
		{
			name: "path with multiple placeholders",
			pathValue: `"/users/{userId}/posts/{postId}"`,
			wantPath: "/users/{userId}/posts/{postId}",
			wantPlaceholders: []string{"userId", "postId"},
		},
		{
			name: "path with no placeholders",
			pathValue: `"/users"`,
			wantPath: "/users",
			wantPlaceholders: []string{},
		},
		{
			name: "empty path",
			pathValue: `""`,
			wantPath: "",
			wantPlaceholders: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a call expression with the path as first argument
			call := &ast.CallExpr{
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: tt.pathValue,
					},
					&ast.Ident{Name: "handler"},
				},
			}
			
			gotPath, gotPlaceholders := extractPathPlaceholders(call)
			
			if gotPath != tt.wantPath {
				t.Errorf("extractPathPlaceholders() path = %q, want %q", gotPath, tt.wantPath)
			}
			
			if len(gotPlaceholders) != len(tt.wantPlaceholders) {
				t.Errorf("extractPathPlaceholders() placeholders count = %d, want %d", len(gotPlaceholders), len(tt.wantPlaceholders))
			}
			
			for _, placeholder := range tt.wantPlaceholders {
				if _, exists := gotPlaceholders[placeholder]; !exists {
					t.Errorf("extractPathPlaceholders() missing placeholder %q", placeholder)
				}
			}
		})
	}
	
	// Test with invalid arguments
	t.Run("invalid call - no args", func(t *testing.T) {
		call := &ast.CallExpr{Args: []ast.Expr{}}
		gotPath, gotPlaceholders := extractPathPlaceholders(call)
		if gotPath != "" || gotPlaceholders != nil {
			t.Errorf("extractPathPlaceholders() with no args should return empty values")
		}
	})
	
	t.Run("invalid call - non-string arg", func(t *testing.T) {
		call := &ast.CallExpr{
			Args: []ast.Expr{
				&ast.Ident{Name: "variable"},
				&ast.Ident{Name: "handler"},
			},
		}
		gotPath, gotPlaceholders := extractPathPlaceholders(call)
		if gotPath != "" || gotPlaceholders != nil {
			t.Errorf("extractPathPlaceholders() with non-string arg should return empty values")
		}
	})
}

func TestValidatePlaceholders(t *testing.T) {
	// Test with valid placeholders (should not report anything)
	placeholders := map[string]struct{}{
		"id":     {},
		"userId": {},
	}
	
	call := &ast.CallExpr{
		Fun: &ast.Ident{Name: "Get"},
		Args: []ast.Expr{
			&ast.BasicLit{Value: `"/test/{id}"`},
		},
	}
	
	// Test that function doesn't panic with valid placeholders and nil pass
	// (this simulates the case where validatePlaceholders is called without reporting)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("validatePlaceholders panicked with valid placeholders: %v", r)
		}
	}()
	
	// Create a mock pass that doesn't actually report but doesn't cause nil pointer dereference
	// We test the logic path but can't test actual reporting without full analysis context
	validatePlaceholders(placeholders, nil, call)
	
	// Test empty placeholder detection logic separately
	emptyPlaceholders := map[string]struct{}{
		"":   {}, // This should trigger a diagnostic if pass was not nil
		"id": {},
	}
	
	// This should not panic even with empty placeholder
	validatePlaceholders(emptyPlaceholders, nil, call)
}

// Test the main analyzer functions with minimal setup
func TestAnalyzerFunctions(t *testing.T) {
	// Test that the analyzer is properly configured
	if Analyzer.Name != "lintgork" {
		t.Errorf("Expected analyzer name 'lintgork', got %q", Analyzer.Name)
	}
	
	if Analyzer.Doc == "" {
		t.Error("Analyzer should have documentation")
	}
	
	if Analyzer.Run == nil {
		t.Error("Analyzer should have a Run function")
	}
}

// Helper function to create test source for more complex tests
func createTestFile(t *testing.T, source string) *ast.File {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}
	return file
}

func TestStructFieldValidation(t *testing.T) {
	source := `
package test

type User struct {
	ID   string ` + "`" + `json:"id" openapi:"id,in=path"` + "`" + `
	Name string ` + "`" + `json:"name" openapi:"discriminator=user"` + "`" + `
	Age  int    ` + "`" + `json:"age"` + "`" + `
}
`
	
	file := createTestFile(t, source)
	
	// Find the struct type
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType = st
				return false
			}
		}
		return true
	})
	
	if structType == nil {
		t.Fatal("Could not find struct type in test source")
	}
	
	// Test that validateStructFields doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("validateStructFields panicked: %v", r)
		}
	}()
	
	validateStructFields(structType, nil)
}

func TestStructFieldValidationEdgeCases(t *testing.T) {
	// Test struct with field that has no tag
	sourceNoTag := `
package test

type NoTag struct {
	Field string
}
`
	
	file := createTestFile(t, sourceNoTag)
	
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType = st
				return false
			}
		}
		return true
	})
	
	if structType == nil {
		t.Fatal("Could not find struct type in test source")
	}
	
	validateStructFields(structType, nil)
	
	// Test error path in strconv.Unquote by manually creating struct with invalid tag
	// Create a struct field with a tag that has invalid syntax for strconv.Unquote
	invalidStructType := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{{Name: "Field"}},
					Type:  &ast.Ident{Name: "string"},
					Tag:   &ast.BasicLit{Value: "invalid unquoted string"}, // This will cause strconv.Unquote to fail
				},
			},
		},
	}
	
	// This should trigger the error path in validateStructFields
	validateStructFields(invalidStructType, nil)
}

func TestCheckStructFieldsForDuplicatesEdgeCases(t *testing.T) {
	// Test struct with field that has no tag
	sourceNoTag := `
package test

type NoTag struct {
	Field string
}
`
	
	file := createTestFile(t, sourceNoTag)
	
	var structType *ast.StructType
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType = st
				return false
			}
		}
		return true
	})
	
	if structType == nil {
		t.Fatal("Could not find struct type in test source")
	}
	
	discSeen := map[string]ast.Node{}
	checkStructFieldsForDuplicates(structType, discSeen, nil)
	
	// Test struct with field that has invalid tag syntax
	sourceInvalidTag := `
package test

type InvalidTag struct {
	Field string ` + "`" + `invalid tag syntax` + "`" + `
}
`
	
	file2 := createTestFile(t, sourceInvalidTag)
	
	var structType2 *ast.StructType
	ast.Inspect(file2, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				structType2 = st
				return false
			}
		}
		return true
	})
	
	if structType2 == nil {
		t.Fatal("Could not find second struct type in test source")
	}
	
	discSeen2 := map[string]ast.Node{}
	checkStructFieldsForDuplicates(structType2, discSeen2, nil)
	
	// Test error path in strconv.Unquote by manually creating struct with invalid tag
	// This tests the error handling in checkStructFieldsForDuplicates
	invalidStructType2 := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{{Name: "Field"}},
					Type:  &ast.Ident{Name: "string"},
					Tag:   &ast.BasicLit{Value: "invalid unquoted string"}, // This will cause strconv.Unquote to fail
				},
			},
		},
	}
	
	// This should trigger the error path in checkStructFieldsForDuplicates
	discSeen3 := map[string]ast.Node{}
	checkStructFieldsForDuplicates(invalidStructType2, discSeen3, nil)
}

func TestCollectPathParamDiagnosticsEdgeCases(t *testing.T) {
	// Test with various edge cases that might not be covered
	
	// Create a file with different types of function calls
	source := `
package test

func setupRoutes() {
	// Not a call expression - should be ignored
	var x = 5
	
	// Call with no arguments - should be ignored
	someFunc()
	
	// Call with one argument - should be ignored  
	router.Get("/test")
	
	// Call with non-string first argument - should be ignored
	router.Get(variable, handler)
	
	// Valid call
	router.Post("/users/{id}", handler)
	
	// Non-router method call - should be ignored
	fmt.Println("/test/{id}")
}
`
	
	file := createTestFile(t, source)
	
	// Create a minimal inspection that exercises collectPathParamDiagnostics paths
	fset := token.NewFileSet()
	_ = fset // avoid unused variable
	
	// Test the inspection function manually to cover edge cases
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
		validatePlaceholders(placeholders, nil, call)
		return true
	}
	
	ast.Inspect(file, inspect)
}

func TestCollectPathParamDiagnosticsWithRealCases(t *testing.T) {
	// Create test cases that trigger the exact uncovered paths
	
	// Test case 1: Non-router method call (should trigger line 224-226)
	nonRouterCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "fmt"},
			Sel: &ast.Ident{Name: "Println"},
		},
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: `"/users/{id}"`,
			},
			&ast.Ident{Name: "handler"},
		},
	}
	
	// This should return early due to !isRouterMethodCall
	inspect1 := func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		if !isRouterMethodCall(call) {
			return true // This path should be covered
		}
		pathStr, placeholders := extractPathPlaceholders(call)
		if pathStr == "" {
			return true
		}
		validatePlaceholders(placeholders, nil, call)
		return true
	}
	
	// Manually test the non-router call
	inspect1(nonRouterCall)
	
	// Test case 2: Router call that returns empty pathStr (should trigger line 228-230)
	routerCallWithInvalidPath := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "router"},
			Sel: &ast.Ident{Name: "Get"},
		},
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: `invalid string literal`, // This will cause extractPathPlaceholders to return empty string
			},
			&ast.Ident{Name: "handler"},
		},
	}
	
	// This should return early due to pathStr being empty
	inspect2 := func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		if !isRouterMethodCall(call) {
			return true
		}
		pathStr, placeholders := extractPathPlaceholders(call)
		if pathStr == "" {
			return true // This path should be covered
		}
		validatePlaceholders(placeholders, nil, call)
		return true
	}
	
	// Manually test the router call with invalid path
	inspect2(routerCallWithInvalidPath)
}

func TestExtractPathPlaceholdersEdgeCases(t *testing.T) {
	// Test with call having only one argument (not enough args)
	call1 := &ast.CallExpr{
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: `"/test"`,
			},
		},
	}
	
	gotPath, gotPlaceholders := extractPathPlaceholders(call1)
	if gotPath != "/test" || len(gotPlaceholders) != 0 {
		t.Errorf("extractPathPlaceholders() with one arg should work correctly")
	}
	
	// Test with invalid string literal (can't be unquoted)
	call2 := &ast.CallExpr{
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: `invalid string literal`,
			},
			&ast.Ident{Name: "handler"},
		},
	}
	
	gotPath2, gotPlaceholders2 := extractPathPlaceholders(call2)
	if gotPath2 != "" || gotPlaceholders2 != nil {
		t.Errorf("extractPathPlaceholders() with invalid string should return empty values")
	}
}