package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewCommand returns the "openapi" root command with subcommands attached.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "OpenAPI related utilities",
	}
	cmd.AddCommand(newGenerateCmd())
	return cmd
}

func newGenerateCmd() *cobra.Command {
	var (
		buildPath  string
		sourcePath string
		outputPath string
		title      string
		version    string
		format     string
		configPath string
	)

	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate an OpenAPI specification",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration file if provided
			if configPath != "" {
				if err := applyConfig(configPath, &buildPath, &sourcePath, &outputPath, &title, &version); err != nil {
					return err
				}
			}
			// 1. Generate base spec from runtime build (if buildPath provided)
			var spec *api.OpenAPISpec
			if buildPath != "" {
				var err error
				spec, err = buildAndExtractSpec(buildPath)
				if err != nil {
					return err
				}
			}

			// Fallback to empty spec when build step is skipped or failed
			if spec == nil {
				spec = &api.OpenAPISpec{
					OpenAPI:    "3.1.0",
					Info:       api.Info{Title: title, Version: version},
					Paths:      map[string]*api.PathItem{},
					Components: &api.Components{Schemas: map[string]*api.Schema{}},
				}
			}

			// 2. Parse docs and enrich
			if sourcePath != "" {
				extractor := api.NewDocExtractor()
				if err := extractor.ParseDirectory(sourcePath); err != nil {
					return fmt.Errorf("failed to parse source: %w", err)
				}
				api.EnhanceOpenAPISpecWithDocs(spec, extractor)
			}

			// Apply title/version overrides
			spec.Info.Title = title
			spec.Info.Version = version

			if outputPath == "-" {
				return writeSpec(os.Stdout, format, spec)
			}

			f, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer f.Close()
			return writeSpec(f, format, spec)
		},
	}

	c.Flags().StringVar(&buildPath, "build", "", "Path to main package to build with '-tags openapi'")
	c.Flags().StringVar(&sourcePath, "source", ".", "Directory containing Go source code for documentation extraction")
	c.Flags().StringVar(&outputPath, "output", "openapi.json", "Path to output file or '-' for stdout")
	c.Flags().StringVar(&title, "title", "API", "API title")
	c.Flags().StringVar(&version, "version", "0.1.0", "API version")
	c.Flags().StringVar(&format, "format", "json", "Output format: json or yaml")
	c.Flags().StringVar(&configPath, "config", "", "Path to .gork.yml config file")

	return c
}

// buildAndExtractSpec compiles the target program with the "openapi" build
// tag, executes the resulting binary and captures its stdout which is expected
// to contain a JSON OpenAPI 3.1 specification.
//
// The function returns the parsed spec or an error.
func buildAndExtractSpec(buildPath string) (*api.OpenAPISpec, error) {
	tmpExe, err := os.CreateTemp("", "gork-build-*")
	if err != nil {
		return nil, fmt.Errorf("create temp exe: %w", err)
	}
	tmpExe.Close()
	defer os.Remove(tmpExe.Name())

	// Build the binary with the openapi tag so that applications which include
	// a `//go:build openapi` gated init() can enable export mode.
	cmdBuild := exec.Command("go", "build", "-tags", "openapi", "-o", tmpExe.Name(), buildPath)
	cmdBuild.Stdout = os.Stdout
	cmdBuild.Stderr = os.Stderr
	if err := cmdBuild.Run(); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	// Execute the binary and capture its stdout.
	var out bytes.Buffer
	cmdRun := exec.Command(tmpExe.Name())
	cmdRun.Env = append(os.Environ(), "GORK_EXPORT=1")
	cmdRun.Stdout = &out
	cmdRun.Stderr = os.Stderr
	if err := cmdRun.Run(); err != nil {
		return nil, fmt.Errorf("run generated binary: %w", err)
	}

	var spec api.OpenAPISpec
	if err := json.Unmarshal(out.Bytes(), &spec); err != nil {
		return nil, fmt.Errorf("parse spec json: %w", err)
	}
	return &spec, nil
}

func writeSpec(w *os.File, format string, spec *api.OpenAPISpec) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(spec)
	case "yaml", "yml":
		data, err := yaml.Marshal(spec)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// applyConfig loads YAML config and overrides empty flag values.
func applyConfig(path string, buildPath, sourcePath, outputPath, title, version *string) error {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var cfg struct {
		OpenAPI struct {
			Build   string `yaml:"build"`
			Source  string `yaml:"source"`
			Output  string `yaml:"output"`
			Title   string `yaml:"title"`
			Version string `yaml:"version"`
		} `yaml:"openapi"`
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if *buildPath == "" {
		*buildPath = cfg.OpenAPI.Build
	}
	if *sourcePath == "" {
		*sourcePath = cfg.OpenAPI.Source
	}
	if *outputPath == "openapi.json" { // default value
		if cfg.OpenAPI.Output != "" {
			*outputPath = cfg.OpenAPI.Output
		}
	}
	if *title == "API" {
		if cfg.OpenAPI.Title != "" {
			*title = cfg.OpenAPI.Title
		}
	}
	if *version == "0.1.0" {
		if cfg.OpenAPI.Version != "" {
			*version = cfg.OpenAPI.Version
		}
	}
	return nil
}
