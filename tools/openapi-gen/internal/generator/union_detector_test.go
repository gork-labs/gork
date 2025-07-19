package generator

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectUnionType(t *testing.T) {
	tests := []struct {
		name         string
		typeName     string
		wantIsUnion  bool
		wantSize     int
		wantTypes    []string
	}{
		// Union2, Union3, Union4 tests
		{
			name:        "Union2 simple types",
			typeName:    "Union2[string, int]",
			wantIsUnion: true,
			wantSize:    2,
			wantTypes:   []string{"string", "int"},
		},
		{
			name:        "Union3 with packages",
			typeName:    "Union3[models.User, models.Admin, models.Guest]",
			wantIsUnion: true,
			wantSize:    3,
			wantTypes:   []string{"models.User", "models.Admin", "models.Guest"},
		},
		{
			name:        "Union4 with mixed types",
			typeName:    "Union4[string, int, *User, []byte]",
			wantIsUnion: true,
			wantSize:    4,
			wantTypes:   []string{"string", "int", "*User", "[]byte"},
		},
		{
			name:        "unions.Union2 with package prefix",
			typeName:    "unions.Union2[CreditCard, BankTransfer]",
			wantIsUnion: true,
			wantSize:    2,
			wantTypes:   []string{"CreditCard", "BankTransfer"},
		},
		{
			name:        "Union3 with nested generics",
			typeName:    "Union3[User, Admin, Union2[Guest, Anonymous]]",
			wantIsUnion: true,
			wantSize:    3,
			wantTypes:   []string{"User", "Admin", "Union2[Guest, Anonymous]"},
		},

		// Non-union types
		{
			name:        "regular type",
			typeName:    "User",
			wantIsUnion: false,
		},
		{
			name:        "generic but not union",
			typeName:    "List[string]",
			wantIsUnion: false,
		},
		{
			name:        "slice type",
			typeName:    "[]User",
			wantIsUnion: false,
		},
		{
			name:        "map type",
			typeName:    "map[string]interface{}",
			wantIsUnion: false,
		},
		{
			name:        "invalid union number",
			typeName:    "Union5[A, B, C, D, E]",
			wantIsUnion: false,
		},
		{
			name:        "invalid union format",
			typeName:    "Union2(string, int)",
			wantIsUnion: false,
		},
		{
			name:        "empty type name",
			typeName:    "",
			wantIsUnion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectUnionType(tt.typeName)
			
			assert.Equal(t, tt.wantIsUnion, info.IsUnion, "IsUnion mismatch")
			
			if tt.wantIsUnion {
				assert.Equal(t, tt.typeName, info.FullTypeName, "FullTypeName mismatch")
				assert.Equal(t, tt.wantSize, info.UnionSize, "UnionSize mismatch")
				assert.Equal(t, tt.wantTypes, info.UnionTypes, "UnionTypes mismatch")
			} else {
				// For non-union types, FullTypeName should be empty
				assert.Empty(t, info.FullTypeName, "FullTypeName should be empty for non-union types")
			}
		})
	}
}

func TestParseTypeParameters(t *testing.T) {
	tests := []struct {
		name   string
		params string
		want   []string
	}{
		{
			name:   "simple types",
			params: "string, int, bool",
			want:   []string{"string", "int", "bool"},
		},
		{
			name:   "types with spaces",
			params: " string , int , bool ",
			want:   []string{"string", "int", "bool"},
		},
		{
			name:   "nested generics",
			params: "List[string], Map[string, int], User",
			want:   []string{"List[string]", "Map[string, int]", "User"},
		},
		{
			name:   "deeply nested",
			params: "Result[List[Map[string, User]], Error]",
			want:   []string{"Result[List[Map[string, User]], Error]"},
		},
		{
			name:   "mixed brackets and angles",
			params: "List<string>, Map[string, int], Vector<float64>",
			want:   []string{"List<string>", "Map[string, int]", "Vector<float64>"},
		},
		{
			name:   "complex nested with multiple levels",
			params: "Option[Result[List[Map[string, Value]], Error]], Fallback",
			want:   []string{"Option[Result[List[Map[string, Value]], Error]]", "Fallback"},
		},
		{
			name:   "single type",
			params: "string",
			want:   []string{"string"},
		},
		{
			name:   "empty params",
			params: "",
			want:   []string{},
		},
		{
			name:   "pointer types",
			params: "*User, *Admin, User",
			want:   []string{"*User", "*Admin", "User"},
		},
		{
			name:   "slice types",
			params: "[]string, []int, User",
			want:   []string{"[]string", "[]int", "User"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTypeParameters(tt.params)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple identifier",
			code:     "User",
			expected: "User",
		},
		{
			name:     "qualified name",
			code:     "models.User",
			expected: "models.User",
		},
		{
			name:     "pointer type",
			code:     "*User",
			expected: "*User",
		},
		{
			name:     "slice type",
			code:     "[]string",
			expected: "[]string",
		},
		{
			name:     "map type",
			code:     "map[string]int",
			expected: "map[string]int",
		},
		{
			name:     "single generic",
			code:     "Option[string]",
			expected: "Option[string]",
		},
		{
			name:     "multiple generics",
			code:     "Union3[A, B, C]",
			expected: "Union3[A, B, C]",
		},
		{
			name:     "qualified generic",
			code:     "unions.Union2[string, int]",
			expected: "unions.Union2[string, int]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			
			result := getTypeName(expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsUnionField(t *testing.T) {
	tests := []struct {
		name        string
		fieldCode   string
		wantIsUnion bool
		wantIsOneOf bool
	}{
		{
			name:        "Union2 field",
			fieldCode:   "Data unions.Union2[string, int]",
			wantIsUnion: true,
		},
		{
			name:        "regular field",
			fieldCode:   "Name string",
			wantIsUnion: false,
		},
		{
			name:        "generic but not union field",
			fieldCode:   "Items []string",
			wantIsUnion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse as struct field
			structCode := "struct { " + tt.fieldCode + " }"
			expr, err := parser.ParseExpr(structCode)
			require.NoError(t, err)
			
			structType := expr.(*ast.StructType)
			field := structType.Fields.List[0]
			
			info := IsUnionField(field)
			assert.Equal(t, tt.wantIsUnion, info.IsUnion)
		})
	}
}

func TestExtractUnionMemberTypes(t *testing.T) {
	tests := []struct {
		name      string
		unionInfo UnionInfo
		imports   map[string]string
		expected  []string
	}{
		{
			name: "simple types",
			unionInfo: UnionInfo{
				IsUnion:    true,
				UnionTypes: []string{"string", "int", "bool"},
			},
			imports:  map[string]string{},
			expected: []string{"string", "int", "bool"},
		},
		{
			name: "pointer types",
			unionInfo: UnionInfo{
				IsUnion:    true,
				UnionTypes: []string{"*User", "*Admin", "Guest"},
			},
			imports:  map[string]string{},
			expected: []string{"*User", "*Admin", "Guest"},
		},
		{
			name: "qualified types with imports",
			unionInfo: UnionInfo{
				IsUnion:    true,
				UnionTypes: []string{"models.User", "auth.Admin"},
			},
			imports: map[string]string{
				"models": "github.com/example/models",
				"auth":   "github.com/example/auth",
			},
			expected: []string{"models.User", "auth.Admin"},
		},
		{
			name: "mixed pointer and qualified types",
			unionInfo: UnionInfo{
				IsUnion:    true,
				UnionTypes: []string{"*models.User", "auth.Admin", "*Guest"},
			},
			imports: map[string]string{
				"models": "github.com/example/models",
				"auth":   "github.com/example/auth",
			},
			expected: []string{"*models.User", "auth.Admin", "*Guest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractUnionMemberTypes(tt.unionInfo, tt.imports)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDiscriminatorFromTag(t *testing.T) {
	tests := []struct {
		name                 string
		fieldCode            string
		wantDiscriminator    string
		wantHasDiscriminator bool
	}{
		{
			name:                 "field with discriminator tag",
			fieldCode:            `Data interface{} ` + "`openapi:\"discriminator:type\"`",
			wantDiscriminator:    "type",
			wantHasDiscriminator: true,
		},
		{
			name:                 "field with discriminator and other tags",
			fieldCode:            `Data interface{} ` + "`json:\"data\" openapi:\"discriminator:kind;example:test\"`",
			wantDiscriminator:    "kind",
			wantHasDiscriminator: true,
		},
		{
			name:                 "field without discriminator",
			fieldCode:            `Data interface{} ` + "`json:\"data\"`",
			wantDiscriminator:    "",
			wantHasDiscriminator: false,
		},
		{
			name:                 "field without any tags",
			fieldCode:            "Data interface{}",
			wantDiscriminator:    "",
			wantHasDiscriminator: false,
		},
		{
			name:                 "field with openapi tag but no discriminator",
			fieldCode:            `Data interface{} ` + "`openapi:\"example:test\"`",
			wantDiscriminator:    "",
			wantHasDiscriminator: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse as struct field
			structCode := "struct { " + tt.fieldCode + " }"
			expr, err := parser.ParseExpr(structCode)
			require.NoError(t, err)
			
			structType := expr.(*ast.StructType)
			field := structType.Fields.List[0]
			
			discriminator, hasDiscriminator := GetDiscriminatorFromTag(field)
			assert.Equal(t, tt.wantDiscriminator, discriminator)
			assert.Equal(t, tt.wantHasDiscriminator, hasDiscriminator)
		})
	}
}

func TestIsSubsetType(t *testing.T) {
	// Create mock type definitions
	typeMap := map[string]ExtractedType{
		"User": {
			Name: "User",
			Fields: []ExtractedField{
				{Name: "ID", Type: "string"},
				{Name: "Name", Type: "string"},
				{Name: "Email", Type: "string"},
			},
		},
		"PublicUser": {
			Name: "PublicUser", 
			Fields: []ExtractedField{
				{Name: "ID", Type: "string"},
				{Name: "Name", Type: "string"},
			},
		},
		"Admin": {
			Name: "Admin",
			Fields: []ExtractedField{
				{Name: "ID", Type: "string"},
				{Name: "Role", Type: "string"},
			},
		},
		"SuperUser": {
			Name: "SuperUser",
			Fields: []ExtractedField{
				{Name: "ID", Type: "string"},
				{Name: "Name", Type: "string"},
				{Name: "Email", Type: "string"},
				{Name: "Permissions", Type: "[]string"},
				{Name: "Level", Type: "int"},
			},
		},
	}

	tests := []struct {
		name     string
		typeA    string
		typeB    string
		expected bool
	}{
		{
			name:     "PublicUser is subset of User",
			typeA:    "PublicUser",
			typeB:    "User",
			expected: true,
		},
		{
			name:     "User is subset of SuperUser",
			typeA:    "User", 
			typeB:    "SuperUser",
			expected: true,
		},
		{
			name:     "Admin is not subset of User",
			typeA:    "Admin",
			typeB:    "User",
			expected: false,
		},
		{
			name:     "User is not subset of PublicUser",
			typeA:    "User",
			typeB:    "PublicUser",
			expected: false,
		},
		{
			name:     "User is not subset of itself",
			typeA:    "User",
			typeB:    "User",
			expected: false,
		},
		{
			name:     "pointer types",
			typeA:    "*PublicUser",
			typeB:    "*User",
			expected: true,
		},
		{
			name:     "unknown type A",
			typeA:    "Unknown",
			typeB:    "User",
			expected: false,
		},
		{
			name:     "unknown type B",
			typeA:    "User",
			typeB:    "Unknown",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSubsetType(tt.typeA, tt.typeB, typeMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}


func TestRegexPatterns(t *testing.T) {
	t.Run("unionTypeRegex", func(t *testing.T) {
		validUnions := []string{
			"Union2[A, B]",
			"Union3[A, B, C]", 
			"Union4[A, B, C, D]",
			"unions.Union2[string, int]",
			"unions.Union3[A, B, C]",
		}

		invalidUnions := []string{
			"Union1[A]",
			"Union5[A, B, C, D, E]",
			"Union[A, B]",
			"UnionX[A, B]",
			"Union2(A, B)",
			"union2[A, B]",
		}

		for _, valid := range validUnions {
			t.Run("valid: "+valid, func(t *testing.T) {
				matches := unionTypeRegex.FindStringSubmatch(valid)
				assert.Len(t, matches, 3, "Should match union pattern: %s", valid)
			})
		}

		for _, invalid := range invalidUnions {
			t.Run("invalid: "+invalid, func(t *testing.T) {
				matches := unionTypeRegex.FindStringSubmatch(invalid)
				assert.Len(t, matches, 0, "Should not match union pattern: %s", invalid)
			})
		}
	})
}