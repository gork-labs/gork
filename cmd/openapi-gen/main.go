package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/example/openapi-gen/internal/generator"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	inputDirs   []string
	outputFile  string
	title       string
	version     string
	format      string
	routeFiles  []string
	description string
	generateUnionAccessors bool
	unionOutputFile       string
	colocatedGeneration   bool
)

var rootCmd = &cobra.Command{
	Use:   "openapi-gen",
	Short: "Generate OpenAPI 3.1.0 specs from Go code using validator tags",
	Long: `A code-first OpenAPI generator that extracts API documentation
from Go source code using go-playground/validator tags for validation constraints.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringSliceVarP(&inputDirs, "input", "i", []string{"."}, "Input directories to scan for Go files")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "openapi.json", "Output file path")
	rootCmd.Flags().StringVarP(&title, "title", "t", "API", "API title")
	rootCmd.Flags().StringVarP(&version, "version", "v", "1.0.0", "API version")
	rootCmd.Flags().StringVarP(&format, "format", "f", "json", "Output format (json or yaml)")
	rootCmd.Flags().StringSliceVarP(&routeFiles, "routes", "r", nil, "Route registration files")
	rootCmd.Flags().StringVarP(&description, "description", "d", "", "API description")
	rootCmd.Flags().BoolVar(&generateUnionAccessors, "generate-union-accessors", false, "Generate accessor methods for user-defined union types")
	rootCmd.Flags().StringVar(&unionOutputFile, "union-output", "", "Output file for generated union accessors (defaults to union_accessors.go in the same package)")
	rootCmd.Flags().BoolVar(&colocatedGeneration, "colocated", false, "Generate accessor files alongside source files with _goapi_gen.go suffix")
}

func run(cmd *cobra.Command, args []string) error {
	// Only use positional arguments if no input flags were provided
	if len(args) > 0 && !cmd.Flags().Changed("input") {
		inputDirs = args
	}
	
	// Create generator
	gen := generator.New(title, version)
	
	// Set description if provided
	if description != "" {
		gen.Generate().Info.Description = description
	}
	
	// Register any custom validators (these would be detected from code in a full implementation)
	// For demo purposes, registering some common ones
	gen.RegisterCustomValidator("username", "Username must be alphanumeric with optional underscores, no consecutive underscores")
	gen.RegisterCustomValidator("strongpassword", "Password must contain uppercase, lowercase, number, and special character")
	
	// Parse input directories (recursively)
	for _, dir := range inputDirs {
		// Expand directory if it contains wildcards
		if strings.Contains(dir, "*") {
			matches, err := filepath.Glob(dir)
			if err != nil {
				return fmt.Errorf("invalid pattern %s: %w", dir, err)
			}
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil && info.IsDir() {
					allDirs, err := getAllDirectories(match)
					if err != nil {
						return fmt.Errorf("failed to scan directory %s: %w", match, err)
					}
					if err := gen.ParseDirectories(allDirs); err != nil {
						return fmt.Errorf("failed to parse directory %s: %w", match, err)
					}
				}
			}
		} else {
			allDirs, err := getAllDirectories(dir)
			if err != nil {
				return fmt.Errorf("failed to scan directory %s: %w", dir, err)
			}
			if err := gen.ParseDirectories(allDirs); err != nil {
				return fmt.Errorf("failed to parse directory %s: %w", dir, err)
			}
		}
	}
	
	// Parse route files
	if len(routeFiles) > 0 {
		if err := gen.ParseRoutes(routeFiles); err != nil {
			return fmt.Errorf("failed to parse routes: %w", err)
		}
	} else {
		// Try to find route files automatically
		for _, dir := range inputDirs {
			routePatterns := []string{
				filepath.Join(dir, "routes.go"),
				filepath.Join(dir, "router.go"),
				filepath.Join(dir, "main.go"),
				filepath.Join(dir, "routes", "*.go"),
				filepath.Join(dir, "router", "*.go"),
			}
			
			for _, pattern := range routePatterns {
				matches, _ := filepath.Glob(pattern)
				routeFiles = append(routeFiles, matches...)
			}
		}
		
		if len(routeFiles) > 0 {
			fmt.Printf("Found route files: %v\n", routeFiles)
			if err := gen.ParseRoutes(routeFiles); err != nil {
				fmt.Printf("Warning: failed to parse some routes: %v\n", err)
			}
		}
	}
	
	
	// Generate OpenAPI spec
	spec := gen.Generate()
	
	// Create output directory if needed
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Write output
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(spec); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	case "yaml", "yml":
		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		if err := encoder.Encode(spec); err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (use json or yaml)", format)
	}
	
	fmt.Printf("OpenAPI spec generated successfully: %s\n", outputFile)
	
	// Print summary
	pathCount := len(spec.Paths)
	schemaCount := 0
	if spec.Components != nil && spec.Components.Schemas != nil {
		schemaCount = len(spec.Components.Schemas)
	}
	
	fmt.Printf("Summary: %d paths, %d schemas\n", pathCount, schemaCount)
	
	// Generate union accessors if requested
	if generateUnionAccessors {
		// Get extracted types
		types := gen.GetExtractedTypes()
		
		if colocatedGeneration {
			// Co-located generation mode
			unionsWithFiles := generator.ExtractUserDefinedUnionsWithFiles(types)
			
			if len(unionsWithFiles) > 0 {
				// Create accessor generator - package name will be set per file
				accessorGen := generator.NewUnionAccessorGenerator("")
				
				// Generate co-located files
				if err := accessorGen.GenerateColocatedFiles(unionsWithFiles); err != nil {
					return fmt.Errorf("failed to generate co-located union accessors: %w", err)
				}
				
				// Count unique files
				fileMap := make(map[string]bool)
				for _, uwf := range unionsWithFiles {
					outputFile := strings.TrimSuffix(uwf.SourceFile, ".go") + "_goapi_gen.go"
					fileMap[outputFile] = true
				}
				
				fmt.Printf("Generated union accessors for %d types in %d files\n", len(unionsWithFiles), len(fileMap))
			} else {
				fmt.Println("No user-defined union types found")
			}
		} else {
			// Single file generation mode
			unions := generator.ExtractUserDefinedUnions(types)
			
			if len(unions) > 0 {
				// Determine output file for union accessors
				unionOut := unionOutputFile
				if unionOut == "" {
					// Default to union_accessors.go in the first input directory
					if len(inputDirs) > 0 {
						unionOut = filepath.Join(inputDirs[0], "union_accessors.go")
					} else {
						unionOut = "union_accessors.go"
					}
				}
				
				// Get package name from the first union (they should all be in the same package)
				packageName := "main"
				if len(unions) > 0 && unions[0].PackageName != "" {
					packageName = unions[0].PackageName
				}
				
				// Create accessor generator
				accessorGen := generator.NewUnionAccessorGenerator(packageName)
				
				// Generate accessors file
				if err := accessorGen.GenerateFile(unions, unionOut); err != nil {
					return fmt.Errorf("failed to generate union accessors: %w", err)
				}
				
				fmt.Printf("Generated union accessors for %d types: %s\n", len(unions), unionOut)
			} else {
				fmt.Println("No user-defined union types found")
			}
		}
	}
	
	return nil
}

// getAllDirectories recursively finds all directories containing Go files
func getAllDirectories(root string) ([]string, error) {
	var dirs []string
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			// Check if directory contains Go files
			hasGoFiles, err := containsGoFiles(path)
			if err != nil {
				return err
			}
			
			if hasGoFiles {
				dirs = append(dirs, path)
			}
		}
		
		return nil
	})
	
	return dirs, err
}

// containsGoFiles checks if a directory contains any .go files
func containsGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true, nil
		}
	}
	
	return false, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}