package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkParseFile(b *testing.B) {
	// Create test file
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	
	content := `package test

import "time"

type User struct {
	ID        string    ` + "`json:\"id\" validate:\"required,uuid\"`" + `
	Email     string    ` + "`json:\"email\" validate:\"required,email,max=255\"`" + `
	Username  string    ` + "`json:\"username\" validate:\"required,alphanum,min=3,max=50\"`" + `
	Age       *int      ` + "`json:\"age,omitempty\" validate:\"omitempty,gte=0,lte=150\"`" + `
	CreatedAt time.Time ` + "`json:\"createdAt\"`" + `
}

type CreateUserRequest struct {
	Email    string ` + "`json:\"email\" validate:\"required,email\"`" + `
	Username string ` + "`json:\"username\" validate:\"required,alphanum,min=3,max=50\"`" + `
	Age      *int   ` + "`json:\"age,omitempty\" validate:\"omitempty,gte=0,lte=150\"`" + `
}

type UserResponse struct {
	User *User ` + "`json:\"user\"`" + `
}
`
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		gen := New("Benchmark API", "1.0.0")
		err := gen.ParseDirectories([]string{tempDir})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseDirectory(b *testing.B) {
	// Use the examples directory for realistic benchmarking
	examplesDir := "../../examples"
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		gen := New("Benchmark API", "1.0.0")
		err := gen.ParseDirectories([]string{examplesDir})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidatorMapping(b *testing.B) {
	tests := []struct {
		name string
		tag  string
	}{
		{"simple", "required,email"},
		{"complex", "required,email,min=5,max=255,contains=@"},
		{"very_complex", "required,email,min=5,max=255,contains=@,excludes=test,startswith=user,endswith=.com"},
		{"numeric", "required,gte=0,lte=100,gt=5,lt=95"},
		{"array", "required,min=1,max=10,dive,alphanum,len=5"},
		{"cross_field", "required,eqfield=Password,nefield=Username,gtfield=MinValue"},
		{"conditional", "required_if=Type admin,required_unless=Status active,excluded_with=Phone"},
	}

	mapper := NewValidatorMapper()
	
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				schema := &Schema{Type: "string"}
				err := mapper.MapValidatorTags(tt.tag, schema, "string")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUnionDetection(b *testing.B) {
	testCases := []string{
		"Union2[string, int]",
		"Union3[models.User, models.Admin, models.Guest]",
		"Union4[A, B, C, D]",
		"OneOf[PaymentOptions]",
		"unions.Union2[CreditCard, BankTransfer]",
		"unions.OneOf[models.AuthOptions]",
		"RegularType",
		"[]string",
		"map[string]interface{}",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = DetectUnionType(tc)
		}
	}
}

func BenchmarkSchemaGeneration(b *testing.B) {
	// Create test files with different complexity levels
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "simple_struct",
			content: `package test

type SimpleStruct struct {
	ID   string ` + "`json:\"id\" validate:\"required,uuid\"`" + `
	Name string ` + "`json:\"name\" validate:\"required,max=100\"`" + `
	Age  int    ` + "`json:\"age\" validate:\"gte=0,lte=150\"`" + `
}`,
		},
		{
			name: "complex_struct",
			content: `package test

import "time"

type ComplexStruct struct {
	ID          string                 ` + "`json:\"id\" validate:\"required,uuid\"`" + `
	Email       string                 ` + "`json:\"email\" validate:\"required,email,max=255\"`" + `
	Username    string                 ` + "`json:\"username\" validate:\"required,alphanum,min=3,max=50\"`" + `
	Profile     UserProfile            ` + "`json:\"profile\" validate:\"required\"`" + `
	Tags        []string               ` + "`json:\"tags,omitempty\" validate:\"omitempty,dive,alphanum\"`" + `
	Metadata    map[string]interface{} ` + "`json:\"metadata,omitempty\"`" + `
	CreatedAt   time.Time              ` + "`json:\"createdAt\"`" + `
	UpdatedAt   *time.Time             ` + "`json:\"updatedAt,omitempty\"`" + `
	Preferences *UserPreferences       ` + "`json:\"preferences,omitempty\"`" + `
}

type UserProfile struct {
	FirstName   string  ` + "`json:\"firstName\" validate:\"required,alpha,min=2,max=50\"`" + `
	LastName    string  ` + "`json:\"lastName\" validate:\"required,alpha,min=2,max=50\"`" + `
	Bio         *string ` + "`json:\"bio,omitempty\" validate:\"omitempty,max=1000\"`" + `
	Avatar      *string ` + "`json:\"avatar,omitempty\" validate:\"omitempty,url\"`" + `
	PhoneNumber *string ` + "`json:\"phoneNumber,omitempty\" validate:\"omitempty,e164\"`" + `
}

type UserPreferences struct {
	Theme        string ` + "`json:\"theme\" validate:\"oneof=light dark auto\"`" + `
	Language     string ` + "`json:\"language\" validate:\"required,iso3166_1_alpha2\"`" + `
	Timezone     string ` + "`json:\"timezone\" validate:\"required\"`" + `
	Notifications bool  ` + "`json:\"notifications\"`" + `
}`,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Setup
			tempDir := b.TempDir()
			testFile := filepath.Join(tempDir, "test.go")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				b.Fatal(err)
			}

			gen := New("Benchmark API", "1.0.0")
			err = gen.ParseDirectories([]string{tempDir})
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Generate the full spec which includes schema generation
				_ = gen.Generate()
			}
		})
	}
}

func BenchmarkFullGeneration(b *testing.B) {
	// Use different sized inputs
	tests := []struct {
		name string
		dir  string
	}{
		{"testdata_simple", "../../testdata/simple_api"},
		{"examples", "../../examples"},
	}

	for _, tt := range tests {
		// Check if directory exists
		if _, err := os.Stat(tt.dir); os.IsNotExist(err) {
			b.Skipf("Skipping %s: directory %s does not exist", tt.name, tt.dir)
			continue
		}

		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				gen := New("Benchmark API", "1.0.0")
				err := gen.ParseDirectories([]string{tt.dir})
				if err != nil {
					b.Fatal(err)
				}

				// Generate the spec
				_ = gen.Generate()
			}
		})
	}
}

func BenchmarkRouteDetection(b *testing.B) {
	// Create test file with various route patterns
	tempDir := b.TempDir()
	routeFile := filepath.Join(tempDir, "routes.go")
	
	content := `package test

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
)

func setupRoutes() {
	// Standard library
	http.HandleFunc("GET /users", handlers.ListUsers)
	http.HandleFunc("POST /users", handlers.CreateUser)
	http.HandleFunc("GET /users/{id}", handlers.GetUser)
	http.HandleFunc("PUT /users/{id}", handlers.UpdateUser)
	http.HandleFunc("DELETE /users/{id}", handlers.DeleteUser)
	
	// Gin routes
	r := gin.Default()
	r.GET("/gin/users", handlers.ListUsers)
	r.POST("/gin/users", handlers.CreateUser)
	r.GET("/gin/users/:id", handlers.GetUser)
	r.PUT("/gin/users/:id", handlers.UpdateUser)
	r.DELETE("/gin/users/:id", handlers.DeleteUser)
	
	// Gorilla Mux
	router := mux.NewRouter()
	router.HandleFunc("/mux/users", handlers.ListUsers).Methods("GET")
	router.HandleFunc("/mux/users", handlers.CreateUser).Methods("POST")
	router.HandleFunc("/mux/users/{id}", handlers.GetUser).Methods("GET")
	router.HandleFunc("/mux/users/{id}", handlers.UpdateUser).Methods("PUT")
	router.HandleFunc("/mux/users/{id}", handlers.DeleteUser).Methods("DELETE")
}`
	
	err := os.WriteFile(routeFile, []byte(content), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rd := NewRouteDetector()
		_, err := rd.DetectRoutesFromFile(routeFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTypeNameParsing(b *testing.B) {
	typeNames := []string{
		"string",
		"[]string",
		"map[string]int",
		"Union2[string, int]",
		"Union3[A, B, C]",
		"OneOf[PaymentOptions]",
		"unions.Union2[CreditCard, BankTransfer]",
		"unions.OneOf[models.AuthOptions]",
		"*User",
		"[]*User",
		"map[string]*User",
		"Option[Result[List[Map[string, User]], Error]]",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, typeName := range typeNames {
			_ = DetectUnionType(typeName)
		}
	}
}

func BenchmarkMemoryUsage(b *testing.B) {
	// This benchmark focuses on memory allocation patterns
	examplesDir := "../../examples"
	
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		b.Skip("Examples directory not found")
	}

	b.Run("parse_only", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			gen := New("Memory Benchmark API", "1.0.0")
			err := gen.ParseDirectories([]string{examplesDir})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("full_generation", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			gen := New("Memory Benchmark API", "1.0.0")
			err := gen.ParseDirectories([]string{examplesDir})
			if err != nil {
				b.Fatal(err)
			}

			// Generate the spec
			_ = gen.Generate()
		}
	})
}

// Benchmark with different validator tag complexities
func BenchmarkValidatorTagSizes(b *testing.B) {
	tags := []string{
		"required",
		"required,email",
		"required,email,max=255",
		"required,email,max=255,min=5",
		"required,email,max=255,min=5,contains=@",
		"required,email,max=255,min=5,contains=@,excludes=test,startswith=user,endswith=.com",
		"required,email,max=255,min=5,contains=@,excludes=test,startswith=user,endswith=.com,eqfield=Password,nefield=Username",
		"required,email,max=255,min=5,contains=@,excludes=test,startswith=user,endswith=.com,eqfield=Password,nefield=Username,required_if=Type admin,required_unless=Status active",
	}

	mapper := NewValidatorMapper()
	
	for i, tag := range tags {
		b.Run(fmt.Sprintf("tag_size_%d", i+1), func(b *testing.B) {
			b.ReportAllocs()
			
			for j := 0; j < b.N; j++ {
				schema := &Schema{Type: "string"}
				err := mapper.MapValidatorTags(tag, schema, "string")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark concurrent parsing (if we implement it)
func BenchmarkConcurrentParsing(b *testing.B) {
	if _, err := os.Stat("../../examples"); os.IsNotExist(err) {
		b.Skip("Examples directory not found")
	}

	b.Run("sequential", func(b *testing.B) {
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			gen := New("Concurrent Benchmark API", "1.0.0")
			err := gen.ParseDirectories([]string{"../../examples"})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Note: This would require implementing concurrent parsing
	// For now, we just have the sequential benchmark as a baseline
}