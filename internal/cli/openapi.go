package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gork-labs/gork/pkg/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newOpenAPICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "OpenAPI related utilities",
	}
	cmd.AddCommand(newGenerateCommand())
	return cmd
}

func newGenerateCommand() *cobra.Command {
	var config GenerateConfig

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate an OpenAPI specification",
		RunE: func(_ *cobra.Command, _ []string) error {
			return GenerateSpec(&config)
		},
	}

	cmd.Flags().StringVar(&config.BuildPath, "build", "", "Path to main package to build with '-tags openapi'")
	cmd.Flags().StringVar(&config.SourcePath, "source", ".", "Directory containing Go source code for documentation extraction")
	cmd.Flags().StringVar(&config.OutputPath, "output", "openapi.json", "Path to output file or '-' for stdout")
	cmd.Flags().StringVar(&config.Title, "title", "API", "API title")
	cmd.Flags().StringVar(&config.Version, "version", "0.1.0", "API version")
	cmd.Flags().StringVar(&config.Format, "format", "json", "Output format: json or yaml")
	cmd.Flags().StringVar(&config.ConfigPath, "config", "", "Path to .gork.yml config file")

	return cmd
}

// GenerateConfig holds configuration for OpenAPI generation.
type GenerateConfig struct {
	BuildPath  string
	SourcePath string
	OutputPath string
	Title      string
	Version    string
	Format     string
	ConfigPath string
}

// GenerateSpec generates an OpenAPI specification based on the provided configuration.
func GenerateSpec(config *GenerateConfig) error {
	if err := loadConfigFile(config); err != nil {
		return err
	}

	spec, err := generateBaseSpec(config)
	if err != nil {
		return err
	}

	if err := enrichWithDocs(spec, config.SourcePath); err != nil {
		return err
	}

	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("spec validation failed: %w", err)
	}

	return writeOutput(spec, config)
}

func loadConfigFile(config *GenerateConfig) error {
	if config.ConfigPath == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Clean(config.ConfigPath))
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

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// Apply config values if flags weren't set
	if config.BuildPath == "" {
		config.BuildPath = cfg.OpenAPI.Build
	}
	if config.SourcePath == "" {
		config.SourcePath = cfg.OpenAPI.Source
	}
	if config.OutputPath == "openapi.json" && cfg.OpenAPI.Output != "" {
		config.OutputPath = cfg.OpenAPI.Output
	}
	if config.Title == "API" && cfg.OpenAPI.Title != "" {
		config.Title = cfg.OpenAPI.Title
	}
	if config.Version == "0.1.0" && cfg.OpenAPI.Version != "" {
		config.Version = cfg.OpenAPI.Version
	}

	return nil
}

func generateBaseSpec(config *GenerateConfig) (*api.OpenAPISpec, error) {
	if config.BuildPath == "" {
		return &api.OpenAPISpec{
			OpenAPI:    "3.1.0",
			Info:       api.Info{Title: config.Title, Version: config.Version},
			Paths:      map[string]*api.PathItem{},
			Components: &api.Components{Schemas: map[string]*api.Schema{}},
		}, nil
	}
	return buildAndExtract(config.BuildPath)
}

// BuildRunner allows dependency injection for testing
type BuildRunner interface {
	CreateTemp(pattern string) (*os.File, error)
	BuildCommand(outputPath, buildPath string) error
	RunCommand(exePath string) ([]byte, error)
}

// DefaultBuildRunner implements BuildRunner using real OS commands
type DefaultBuildRunner struct{}

func (r *DefaultBuildRunner) CreateTemp(pattern string) (*os.File, error) {
	return os.CreateTemp("", pattern)
}

func (r *DefaultBuildRunner) BuildCommand(outputPath, buildPath string) error {
	cmd := exec.Command("go", "build", "-tags", "openapi", "-o", outputPath, buildPath) // #nosec G204
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *DefaultBuildRunner) RunCommand(exePath string) ([]byte, error) {
	var out bytes.Buffer
	cmd := exec.Command(exePath) // #nosec G204
	cmd.Env = append(os.Environ(), "GORK_EXPORT=1")
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return out.Bytes(), err
}

var defaultBuildRunner BuildRunner = &DefaultBuildRunner{}

func buildAndExtract(buildPath string) (*api.OpenAPISpec, error) {
	return buildAndExtractWithRunner(buildPath, defaultBuildRunner)
}

func buildAndExtractWithRunner(buildPath string, runner BuildRunner) (*api.OpenAPISpec, error) {
	tmpExe, err := runner.CreateTemp("gork-build-*")
	if err != nil {
		return nil, fmt.Errorf("create temp exe: %w", err)
	}
	_ = tmpExe.Close()
	defer func() { _ = os.Remove(tmpExe.Name()) }()

	if err := runner.BuildCommand(tmpExe.Name(), buildPath); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	output, err := runner.RunCommand(tmpExe.Name())
	if err != nil {
		return nil, fmt.Errorf("run generated binary: %w", err)
	}

	var spec api.OpenAPISpec
	if err := json.Unmarshal(output, &spec); err != nil {
		return nil, fmt.Errorf("parse spec json: %w", err)
	}
	return &spec, nil
}

func enrichWithDocs(spec *api.OpenAPISpec, sourcePath string) error {
	if sourcePath == "" {
		return nil
	}
	extractor := api.NewDocExtractor()
	if err := extractor.ParseDirectory(sourcePath); err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}
	api.EnhanceOpenAPISpecWithDocs(spec, extractor)
	return nil
}

// HTTPClient interface for dependency injection
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}

// DefaultHTTPClient implements HTTPClient
type DefaultHTTPClient struct{}

func (c *DefaultHTTPClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return http.Post(url, contentType, body)
}

// ValidatorClient interface for dependency injection
type ValidatorClient interface {
	CallValidator(data []byte) ([]byte, int, error)
	MarshalJSON(v interface{}) ([]byte, error)
}

// DefaultValidatorClient implements ValidatorClient
type DefaultValidatorClient struct {
	httpClient HTTPClient
}

func NewDefaultValidatorClient() *DefaultValidatorClient {
	return &DefaultValidatorClient{httpClient: &DefaultHTTPClient{}}
}

func (c *DefaultValidatorClient) CallValidator(data []byte) ([]byte, int, error) {
	return c.callValidatorWithClient(data)
}

func (c *DefaultValidatorClient) MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c *DefaultValidatorClient) callValidatorWithClient(data []byte) ([]byte, int, error) {
	resp, err := c.httpClient.Post("https://validator.swagger.io/validator/debug", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, 0, fmt.Errorf("send to Swagger validator: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

var defaultValidatorClient ValidatorClient = NewDefaultValidatorClient()

func validateSpec(spec *api.OpenAPISpec) error {
	return validateSpecWithClient(spec, defaultValidatorClient)
}

func validateSpecWithClient(spec *api.OpenAPISpec, client ValidatorClient) error {
	data, err := client.MarshalJSON(spec)
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}

	body, statusCode, err := client.CallValidator(data)
	if err != nil {
		return err
	}

	return parseValidatorResponse(body, statusCode)
}


func parseValidatorResponse(body []byte, statusCode int) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf("validator returned status %d: %s", statusCode, string(body))
	}

	trimmed := bytes.TrimSpace(body)
	if bytes.Equal(trimmed, []byte("{}")) || len(trimmed) == 0 {
		return nil
	}

	if trimmed[0] == '{' {
		var result struct {
			Messages []struct {
				Level   string `json:"level"`
				Message string `json:"message"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(trimmed, &result); err == nil {
			for _, m := range result.Messages {
				if m.Level == "error" {
					return fmt.Errorf("swagger validator errors: %s", string(body))
				}
			}
			return nil
		}
	}

	if bytes.Contains(bytes.ToLower(trimmed), []byte("error")) && !bytes.Contains(trimmed, []byte("schemaValidationMessages: null")) {
		return fmt.Errorf("swagger validator returned errors: %s", string(body))
	}

	return nil
}

// FileSystem interface for dependency injection
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	Create(name string) (*os.File, error)
}

// DefaultFileSystem implements FileSystem
type DefaultFileSystem struct{}

func (fs *DefaultFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *DefaultFileSystem) Create(name string) (*os.File, error) {
	return os.Create(name)
}

var defaultFileSystem FileSystem = &DefaultFileSystem{}

func writeOutput(spec *api.OpenAPISpec, config *GenerateConfig) error {
	return writeOutputWithFS(spec, config, defaultFileSystem)
}

func writeOutputWithFS(spec *api.OpenAPISpec, config *GenerateConfig, fs FileSystem) error {
	if config.OutputPath == "-" {
		return writeSpec(os.Stdout, config.Format, spec)
	}

	outDir := filepath.Dir(config.OutputPath)
	if fi, err := fs.Stat(outDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("output directory %s does not exist — please create it first", outDir)
		}
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("output path %s is not a directory", outDir)
	}

	f, err := fs.Create(config.OutputPath) // #nosec G304
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return writeSpec(f, config.Format, spec)
}

// SpecWriter interface for dependency injection
type SpecWriter interface {
	MarshalYAML(v interface{}) ([]byte, error)
}

// DefaultSpecWriter implements SpecWriter
type DefaultSpecWriter struct{}

func (w *DefaultSpecWriter) MarshalYAML(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

var defaultSpecWriter SpecWriter = &DefaultSpecWriter{}

func writeSpec(w *os.File, format string, spec *api.OpenAPISpec) error {
	return writeSpecWithWriter(w, format, spec, defaultSpecWriter)
}

func writeSpecWithWriter(w *os.File, format string, spec *api.OpenAPISpec, writer SpecWriter) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(spec)
	case "yaml", "yml":
		data, err := writer.MarshalYAML(spec)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}