// Package config provides configuration structures for the load generator.
// The main Config struct ties together all loadgen components.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/loadctrl"
	"github.com/example/erp/tools/loadgen/internal/warmup"
	"github.com/example/erp/tools/loadgen/internal/workflow"
	"gopkg.in/yaml.v3"
)

// Errors returned by the config package.
var (
	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("config: invalid configuration")
	// ErrConfigNotFound is returned when the config file is not found.
	ErrConfigNotFound = errors.New("config: configuration file not found")
)

// Config is the root configuration structure for the load generator.
type Config struct {
	// Name is a descriptive name for this configuration.
	Name string `yaml:"name" json:"name"`

	// Description provides additional context about the configuration.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Version is the configuration schema version.
	Version string `yaml:"version" json:"version"`

	// Target is the base URL of the target system.
	Target TargetConfig `yaml:"target" json:"target"`

	// Auth configures authentication for the target system.
	Auth AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`

	// Duration is the total duration of the load test.
	// Default: 5m
	Duration time.Duration `yaml:"duration" json:"duration"`

	// Warmup configures the warmup phase.
	Warmup warmup.Config `yaml:"warmup,omitempty" json:"warmup,omitempty"`

	// Endpoints defines the API endpoints and their configurations.
	Endpoints []EndpointConfig `yaml:"endpoints" json:"endpoints"`

	// Scenarios defines named scenarios that group endpoints.
	Scenarios []ScenarioConfig `yaml:"scenarios,omitempty" json:"scenarios,omitempty"`

	// TrafficShaper configures traffic patterns.
	TrafficShaper loadctrl.ShaperConfig `yaml:"trafficShaper" json:"trafficShaper"`

	// RateLimiter configures the rate limiter.
	RateLimiter loadctrl.RateLimiterConfig `yaml:"rateLimiter,omitempty" json:"rateLimiter,omitempty"`

	// WorkerPool configures the worker pool.
	WorkerPool loadctrl.WorkerPoolConfig `yaml:"workerPool,omitempty" json:"workerPool,omitempty"`

	// Controller configures the load controller.
	Controller loadctrl.LoadControllerConfig `yaml:"controller,omitempty" json:"controller,omitempty"`

	// Backpressure configures backpressure handling.
	Backpressure loadctrl.BackpressureConfig `yaml:"backpressure,omitempty" json:"backpressure,omitempty"`

	// Metrics configures metrics collection.
	Metrics loadctrl.MetricsConfig `yaml:"metrics,omitempty" json:"metrics,omitempty"`

	// Output configures output and reporting.
	Output OutputConfig `yaml:"output,omitempty" json:"output,omitempty"`

	// DataGenerators configures data generators for semantic types.
	// The key is the semantic type (e.g., "common.code", "common.name").
	DataGenerators map[string]GeneratorConfig `yaml:"dataGenerators,omitempty" json:"dataGenerators,omitempty"`

	// SemanticOverrides allows explicit semantic type assignments.
	// Keys can be:
	// - Exact field name: "customer_id"
	// - Endpoint-scoped: "/customers:id"
	// - Tag-scoped: "customers:id"
	// - Wildcard: "*.customer_id"
	SemanticOverrides map[string]string `yaml:"semanticOverrides,omitempty" json:"semanticOverrides,omitempty"`

	// InferenceConfig configures the semantic type inference engine.
	InferenceConfig *InferenceConfig `yaml:"inference,omitempty" json:"inference,omitempty"`

	// Workflows defines business workflows for sequential API call execution.
	// Workflows are executed as complete sequences, with parameter passing between steps.
	Workflows map[string]workflow.Definition `yaml:"workflows,omitempty" json:"workflows,omitempty"`

	// Assertions configures SLO assertions for validating test results.
	// Assertions are evaluated at the end of the test.
	Assertions AssertionConfig `yaml:"assertions,omitempty" json:"assertions,omitempty"`
}

// InferenceConfig configures the semantic type inference engine.
type InferenceConfig struct {
	// Enabled controls whether inference is enabled.
	// Default: true
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// MinConfidence is the minimum confidence threshold (0.0-1.0).
	// Inferences below this threshold are rejected.
	// Default: 0.7
	MinConfidence float64 `yaml:"minConfidence,omitempty" json:"minConfidence,omitempty"`

	// DryRun outputs inference results without applying them.
	// Default: false
	DryRun bool `yaml:"dryRun,omitempty" json:"dryRun,omitempty"`
}

// TargetConfig holds target system configuration.
type TargetConfig struct {
	// BaseURL is the base URL of the target system (e.g., "http://localhost:8080").
	BaseURL string `yaml:"baseURL" json:"baseURL"`

	// APIVersion is the API version prefix (e.g., "v1").
	// Default: "v1"
	APIVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`

	// Timeout is the request timeout.
	// Default: 30s
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// TLSSkipVerify skips TLS certificate verification (for testing only).
	TLSSkipVerify bool `yaml:"tlsSkipVerify,omitempty" json:"tlsSkipVerify,omitempty"`

	// Headers are additional headers to include in all requests.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	// Type is the authentication type: "none", "basic", "bearer", "api_key".
	Type string `yaml:"type" json:"type"`

	// Login configures login-based authentication.
	Login *LoginConfig `yaml:"login,omitempty" json:"login,omitempty"`

	// APIKey configures API key authentication.
	APIKey *APIKeyConfig `yaml:"apiKey,omitempty" json:"apiKey,omitempty"`

	// Bearer configures static bearer token authentication.
	Bearer *BearerConfig `yaml:"bearer,omitempty" json:"bearer,omitempty"`
}

// LoginConfig configures login-based authentication.
type LoginConfig struct {
	// Endpoint is the login endpoint path (e.g., "/auth/login").
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Method is the HTTP method (default: "POST").
	Method string `yaml:"method,omitempty" json:"method,omitempty"`

	// Username is the login username.
	Username string `yaml:"username" json:"username"`

	// Password is the login password.
	Password string `yaml:"password" json:"password"`

	// TokenPath is the JSONPath to extract the token from the response.
	// Default: "$.data.access_token"
	TokenPath string `yaml:"tokenPath,omitempty" json:"tokenPath,omitempty"`

	// RefreshEndpoint is the token refresh endpoint (optional).
	RefreshEndpoint string `yaml:"refreshEndpoint,omitempty" json:"refreshEndpoint,omitempty"`

	// RefreshInterval is how often to refresh the token.
	RefreshInterval time.Duration `yaml:"refreshInterval,omitempty" json:"refreshInterval,omitempty"`
}

// APIKeyConfig configures API key authentication.
type APIKeyConfig struct {
	// Key is the API key value.
	Key string `yaml:"key" json:"key"`

	// Header is the header name for the API key.
	// Default: "X-API-Key"
	Header string `yaml:"header,omitempty" json:"header,omitempty"`
}

// BearerConfig configures static bearer token authentication.
type BearerConfig struct {
	// Token is the bearer token value.
	Token string `yaml:"token" json:"token"`
}

// EndpointConfig configures a single API endpoint.
type EndpointConfig struct {
	// Name is a unique identifier for this endpoint.
	Name string `yaml:"name" json:"name"`

	// Description provides additional context about the endpoint.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Path is the URL path (e.g., "/catalog/products").
	Path string `yaml:"path" json:"path"`

	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH).
	Method string `yaml:"method" json:"method"`

	// Weight determines how often this endpoint is called relative to others.
	// Higher weight = more frequent calls.
	// Default: 1
	Weight int `yaml:"weight,omitempty" json:"weight,omitempty"`

	// Tags categorize the endpoint (e.g., ["read", "catalog"]).
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// RequiresAuth indicates whether this endpoint requires authentication.
	// Default: true
	RequiresAuth *bool `yaml:"requiresAuth,omitempty" json:"requiresAuth,omitempty"`

	// Headers are additional headers for this endpoint.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// QueryParams are query parameters for GET requests.
	QueryParams map[string]ParameterConfig `yaml:"queryParams,omitempty" json:"queryParams,omitempty"`

	// PathParams are path parameters (e.g., {id} in /products/{id}).
	PathParams map[string]ParameterConfig `yaml:"pathParams,omitempty" json:"pathParams,omitempty"`

	// Body is the request body template (for POST/PUT/PATCH).
	Body string `yaml:"body,omitempty" json:"body,omitempty"`

	// BodyTemplate is a Go template for the request body.
	BodyTemplate string `yaml:"bodyTemplate,omitempty" json:"bodyTemplate,omitempty"`

	// ExpectedStatus is the expected HTTP status code.
	// Default: 200 for GET, 201 for POST, 204 for DELETE
	ExpectedStatus int `yaml:"expectedStatus,omitempty" json:"expectedStatus,omitempty"`

	// Produces defines values this endpoint produces (for warmup).
	Produces []ProducesConfig `yaml:"produces,omitempty" json:"produces,omitempty"`

	// Consumes defines semantic types this endpoint consumes.
	Consumes []circuit.SemanticType `yaml:"consumes,omitempty" json:"consumes,omitempty"`

	// DependsOn lists endpoints that must be called first (for warmup ordering).
	DependsOn []string `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`

	// Timeout is the endpoint-specific timeout (overrides target timeout).
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Disabled indicates whether this endpoint is disabled.
	Disabled bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// ParameterConfig configures a single parameter.
type ParameterConfig struct {
	// SemanticType links this parameter to the pool.
	// Example: "entity.product.id"
	SemanticType circuit.SemanticType `yaml:"semanticType,omitempty" json:"semanticType,omitempty"`

	// Value is a static value for this parameter.
	Value string `yaml:"value,omitempty" json:"value,omitempty"`

	// Generator specifies a data generator.
	Generator *GeneratorConfig `yaml:"generator,omitempty" json:"generator,omitempty"`
}

// GeneratorConfig configures a data generator.
type GeneratorConfig struct {
	// Type is the generator type: "faker", "random", "sequence", "pattern".
	Type string `yaml:"type" json:"type"`

	// Faker is faker-specific configuration.
	Faker *FakerConfig `yaml:"faker,omitempty" json:"faker,omitempty"`

	// Random is random generator configuration.
	Random *RandomConfig `yaml:"random,omitempty" json:"random,omitempty"`

	// Sequence is sequence generator configuration.
	Sequence *SequenceConfig `yaml:"sequence,omitempty" json:"sequence,omitempty"`

	// Pattern is pattern-based generator configuration.
	Pattern *PatternConfig `yaml:"pattern,omitempty" json:"pattern,omitempty"`
}

// FakerConfig configures the faker data generator.
type FakerConfig struct {
	// Type is the faker type: "name", "email", "phone", "address", "company", etc.
	Type string `yaml:"type" json:"type"`

	// Locale is the locale for generated data.
	// Default: "en"
	Locale string `yaml:"locale,omitempty" json:"locale,omitempty"`
}

// RandomConfig configures random value generation.
type RandomConfig struct {
	// Type is the value type: "int", "float", "string", "uuid", "bool".
	Type string `yaml:"type" json:"type"`

	// Min is the minimum value (for int/float).
	Min float64 `yaml:"min,omitempty" json:"min,omitempty"`

	// Max is the maximum value (for int/float).
	Max float64 `yaml:"max,omitempty" json:"max,omitempty"`

	// Length is the string length.
	Length int `yaml:"length,omitempty" json:"length,omitempty"`

	// Charset is the character set for strings.
	// Default: "alphanumeric"
	Charset string `yaml:"charset,omitempty" json:"charset,omitempty"`
}

// SequenceConfig configures sequential value generation.
type SequenceConfig struct {
	// Prefix is added before the sequence number.
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`

	// Suffix is added after the sequence number.
	Suffix string `yaml:"suffix,omitempty" json:"suffix,omitempty"`

	// Start is the starting sequence number.
	// Default: 1
	Start int64 `yaml:"start,omitempty" json:"start,omitempty"`

	// Step is the increment between values.
	// Default: 1
	Step int64 `yaml:"step,omitempty" json:"step,omitempty"`

	// Padding is the minimum width with zero-padding.
	Padding int `yaml:"padding,omitempty" json:"padding,omitempty"`
}

// PatternConfig configures pattern-based value generation.
type PatternConfig struct {
	// Pattern is the template pattern.
	// Placeholders: {PREFIX}, {TIMESTAMP}, {RANDOM:N}, {UUID}, {DATE}, {TIME}, {DATETIME},
	// {INT:MIN:MAX}, {ALPHA:N}, {HEX:N}, {SEQUENCE}
	Pattern string `yaml:"pattern" json:"pattern"`

	// Prefix is the value to use for {PREFIX} placeholder.
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
}

// ProducesConfig defines what values an endpoint produces.
type ProducesConfig struct {
	// SemanticType is the semantic type of the produced value.
	SemanticType circuit.SemanticType `yaml:"semanticType" json:"semanticType"`

	// JSONPath is the path to extract the value from the response.
	// Default: "$.data.id"
	JSONPath string `yaml:"jsonPath,omitempty" json:"jsonPath,omitempty"`

	// Multiple indicates this produces multiple values (e.g., from a list).
	Multiple bool `yaml:"multiple,omitempty" json:"multiple,omitempty"`
}

// ScenarioConfig defines a named scenario grouping endpoints.
type ScenarioConfig struct {
	// Name is the scenario identifier.
	Name string `yaml:"name" json:"name"`

	// Description provides context about the scenario.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Endpoints lists the endpoint names in this scenario.
	Endpoints []string `yaml:"endpoints" json:"endpoints"`

	// Weight is the scenario weight for selection.
	// Default: 1
	Weight int `yaml:"weight,omitempty" json:"weight,omitempty"`

	// Sequential indicates endpoints should run in order.
	Sequential bool `yaml:"sequential,omitempty" json:"sequential,omitempty"`
}

// OutputConfig configures output and reporting.
type OutputConfig struct {
	// Type is the output type: "console", "json", "csv", "html".
	// Can be combined with comma separation: "console,json"
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Path is the output file path (for file outputs).
	// Deprecated: Use JSON.File instead.
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// ReportInterval is how often to print progress reports.
	// Default: 10s
	ReportInterval time.Duration `yaml:"reportInterval,omitempty" json:"reportInterval,omitempty"`

	// Verbose enables verbose output.
	Verbose bool `yaml:"verbose,omitempty" json:"verbose,omitempty"`

	// JSON configures JSON output.
	JSON JSONOutputConfig `yaml:"json,omitempty" json:"json,omitempty"`

	// Console configures console output.
	Console ConsoleOutputConfig `yaml:"console,omitempty" json:"console,omitempty"`

	// Prometheus configures Prometheus metrics export.
	Prometheus PrometheusOutputConfig `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
}

// JSONOutputConfig configures JSON report output.
type JSONOutputConfig struct {
	// Enabled enables JSON output.
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// File is the output file path.
	// Supports template variables:
	// - {{.Timestamp}} - Current timestamp in format YYYYMMDD-HHMMSS
	// - {{.Date}} - Current date in format YYYY-MM-DD
	// - {{.Time}} - Current time in format HHMMSS
	// Default: "./results/loadgen-{{.Timestamp}}.json"
	File string `yaml:"file,omitempty" json:"file,omitempty"`
}

// ConsoleOutputConfig configures console output.
type ConsoleOutputConfig struct {
	// Enabled enables console output. Default: true
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Interval is the refresh interval for real-time display.
	// Default: 500ms
	Interval time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
}

// PrometheusOutputConfig configures Prometheus metrics export.
type PrometheusOutputConfig struct {
	// Enabled enables Prometheus metrics export.
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Port is the HTTP port for the /metrics endpoint.
	// Default: 9090
	Port int `yaml:"port,omitempty" json:"port,omitempty"`

	// Path is the URL path for the metrics endpoint.
	// Default: /metrics
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
}

// AssertionConfig configures SLO assertions for load testing.
// Global assertions apply to all endpoints unless overridden.
type AssertionConfig struct {
	// Global assertions apply to overall test results.
	Global *GlobalAssertions `yaml:"global,omitempty" json:"global,omitempty"`

	// EndpointOverrides allows endpoint-specific assertion overrides.
	// Key is the endpoint name.
	EndpointOverrides map[string]EndpointAssertions `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`

	// ExitOnFailure causes the program to return a non-zero exit code
	// if any assertion fails. Default: true
	ExitOnFailure *bool `yaml:"exitOnFailure,omitempty" json:"exitOnFailure,omitempty"`
}

// GlobalAssertions defines SLO thresholds for the entire test.
type GlobalAssertions struct {
	// MaxErrorRate is the maximum allowed error rate (0.0 - 100.0).
	// An assertion fails if the actual error rate exceeds this value.
	// Example: 1.0 means max 1% error rate.
	MaxErrorRate *float64 `yaml:"maxErrorRate,omitempty" json:"maxErrorRate,omitempty"`

	// MinSuccessRate is the minimum required success rate (0.0 - 100.0).
	// An assertion fails if the actual success rate is below this value.
	// Example: 99.0 means minimum 99% success rate.
	MinSuccessRate *float64 `yaml:"minSuccessRate,omitempty" json:"minSuccessRate,omitempty"`

	// MaxP50Latency is the maximum allowed P50 (median) latency.
	// An assertion fails if the actual P50 latency exceeds this value.
	MaxP50Latency time.Duration `yaml:"maxP50Latency,omitempty" json:"maxP50Latency,omitempty"`

	// MaxP95Latency is the maximum allowed P95 latency.
	// An assertion fails if the actual P95 latency exceeds this value.
	MaxP95Latency time.Duration `yaml:"maxP95Latency,omitempty" json:"maxP95Latency,omitempty"`

	// MaxP99Latency is the maximum allowed P99 latency.
	// An assertion fails if the actual P99 latency exceeds this value.
	MaxP99Latency time.Duration `yaml:"maxP99Latency,omitempty" json:"maxP99Latency,omitempty"`

	// MaxAvgLatency is the maximum allowed average latency.
	// An assertion fails if the actual average latency exceeds this value.
	MaxAvgLatency time.Duration `yaml:"maxAvgLatency,omitempty" json:"maxAvgLatency,omitempty"`

	// MinThroughput is the minimum required throughput (requests per second).
	// An assertion fails if the actual QPS is below this value.
	MinThroughput *float64 `yaml:"minThroughput,omitempty" json:"minThroughput,omitempty"`
}

// EndpointAssertions defines SLO thresholds for a specific endpoint.
// These override global assertions for the specified endpoint.
type EndpointAssertions struct {
	// MaxErrorRate is the maximum allowed error rate (0.0 - 100.0).
	MaxErrorRate *float64 `yaml:"maxErrorRate,omitempty" json:"maxErrorRate,omitempty"`

	// MinSuccessRate is the minimum required success rate (0.0 - 100.0).
	MinSuccessRate *float64 `yaml:"minSuccessRate,omitempty" json:"minSuccessRate,omitempty"`

	// MaxP50Latency is the maximum allowed P50 (median) latency.
	MaxP50Latency time.Duration `yaml:"maxP50Latency,omitempty" json:"maxP50Latency,omitempty"`

	// MaxP95Latency is the maximum allowed P95 latency.
	MaxP95Latency time.Duration `yaml:"maxP95Latency,omitempty" json:"maxP95Latency,omitempty"`

	// MaxP99Latency is the maximum allowed P99 latency.
	MaxP99Latency time.Duration `yaml:"maxP99Latency,omitempty" json:"maxP99Latency,omitempty"`

	// MaxAvgLatency is the maximum allowed average latency.
	MaxAvgLatency time.Duration `yaml:"maxAvgLatency,omitempty" json:"maxAvgLatency,omitempty"`

	// MinThroughput is the minimum required throughput for this endpoint.
	MinThroughput *float64 `yaml:"minThroughput,omitempty" json:"minThroughput,omitempty"`

	// Disabled skips assertions for this endpoint entirely.
	Disabled bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return LoadFromBytes(data)
}

// LoadFromBytes loads configuration from YAML bytes.
func LoadFromBytes(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.ApplyDefaults()
	return &cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidConfig)
	}

	if c.Target.BaseURL == "" {
		return fmt.Errorf("%w: target.baseURL is required", ErrInvalidConfig)
	}

	if len(c.Endpoints) == 0 {
		return fmt.Errorf("%w: at least one endpoint is required", ErrInvalidConfig)
	}

	// Validate endpoints
	names := make(map[string]bool)
	for i, ep := range c.Endpoints {
		if ep.Name == "" {
			return fmt.Errorf("%w: endpoint[%d].name is required", ErrInvalidConfig, i)
		}
		if names[ep.Name] {
			return fmt.Errorf("%w: duplicate endpoint name: %s", ErrInvalidConfig, ep.Name)
		}
		names[ep.Name] = true

		if ep.Path == "" {
			return fmt.Errorf("%w: endpoint[%d].path is required", ErrInvalidConfig, i)
		}
		if ep.Method == "" {
			return fmt.Errorf("%w: endpoint[%d].method is required", ErrInvalidConfig, i)
		}
	}

	// Validate warmup config
	if err := c.Warmup.Validate(); err != nil {
		return fmt.Errorf("warmup config: %w", err)
	}

	// Validate workflows
	for name, def := range c.Workflows {
		if err := def.Validate(name); err != nil {
			return fmt.Errorf("workflow config: %w", err)
		}
	}

	return nil
}

// ApplyDefaults applies default values to unset fields.
func (c *Config) ApplyDefaults() {
	if c.Version == "" {
		c.Version = "1.0"
	}

	if c.Duration == 0 {
		c.Duration = 5 * time.Minute
	}

	if c.Target.APIVersion == "" {
		c.Target.APIVersion = "v1"
	}

	if c.Target.Timeout == 0 {
		c.Target.Timeout = 30 * time.Second
	}

	// Apply defaults to endpoints
	for i := range c.Endpoints {
		if c.Endpoints[i].Weight == 0 {
			c.Endpoints[i].Weight = 1
		}
		if c.Endpoints[i].RequiresAuth == nil {
			requiresAuth := true
			c.Endpoints[i].RequiresAuth = &requiresAuth
		}
	}

	// Apply defaults to scenarios
	for i := range c.Scenarios {
		if c.Scenarios[i].Weight == 0 {
			c.Scenarios[i].Weight = 1
		}
	}

	// Apply warmup defaults
	c.Warmup.ApplyDefaults()

	// Apply workflow defaults
	for name, def := range c.Workflows {
		def.ApplyDefaults(name)
		c.Workflows[name] = def
	}

	// Apply output defaults
	if c.Output.ReportInterval == 0 {
		c.Output.ReportInterval = 10 * time.Second
	}

	// Apply console output defaults
	if c.Output.Console.Enabled == nil {
		enabled := true
		c.Output.Console.Enabled = &enabled
	}
	if c.Output.Console.Interval == 0 {
		c.Output.Console.Interval = 500 * time.Millisecond
	}

	// Apply JSON output defaults
	if c.Output.JSON.Enabled && c.Output.JSON.File == "" {
		c.Output.JSON.File = "./results/loadgen-{{.Timestamp}}.json"
	}

	// Apply Prometheus output defaults
	if c.Output.Prometheus.Enabled {
		if c.Output.Prometheus.Port == 0 {
			c.Output.Prometheus.Port = 9090
		}
		if c.Output.Prometheus.Path == "" {
			c.Output.Prometheus.Path = "/metrics"
		}
	}

	// Apply assertion defaults
	if c.Assertions.ExitOnFailure == nil {
		exitOnFailure := true
		c.Assertions.ExitOnFailure = &exitOnFailure
	}
}

// GetEndpointByName returns an endpoint by name.
func (c *Config) GetEndpointByName(name string) *EndpointConfig {
	for i := range c.Endpoints {
		if c.Endpoints[i].Name == name {
			return &c.Endpoints[i]
		}
	}
	return nil
}

// GetProducerEndpoints returns endpoints that produce the given semantic type.
func (c *Config) GetProducerEndpoints(semantic circuit.SemanticType) []*EndpointConfig {
	var producers []*EndpointConfig
	for i := range c.Endpoints {
		for _, p := range c.Endpoints[i].Produces {
			if p.SemanticType == semantic {
				producers = append(producers, &c.Endpoints[i])
				break
			}
		}
	}
	return producers
}

// GetConsumerEndpoints returns endpoints that consume the given semantic type.
func (c *Config) GetConsumerEndpoints(semantic circuit.SemanticType) []*EndpointConfig {
	var consumers []*EndpointConfig
	for i := range c.Endpoints {
		for _, ct := range c.Endpoints[i].Consumes {
			if ct == semantic {
				consumers = append(consumers, &c.Endpoints[i])
				break
			}
		}
	}
	return consumers
}

// GetEnabledEndpoints returns all non-disabled endpoints.
func (c *Config) GetEnabledEndpoints() []EndpointConfig {
	var enabled []EndpointConfig
	for _, ep := range c.Endpoints {
		if !ep.Disabled {
			enabled = append(enabled, ep)
		}
	}
	return enabled
}

// GetEndpointsByTag returns endpoints with the given tag.
func (c *Config) GetEndpointsByTag(tag string) []EndpointConfig {
	var matched []EndpointConfig
	for _, ep := range c.Endpoints {
		for _, t := range ep.Tags {
			if t == tag {
				matched = append(matched, ep)
				break
			}
		}
	}
	return matched
}

// TotalWeight returns the sum of all enabled endpoint weights.
func (c *Config) TotalWeight() int {
	total := 0
	for _, ep := range c.Endpoints {
		if !ep.Disabled {
			total += ep.Weight
		}
	}
	return total
}

// GetWorkflowConfig returns a workflow.Config from this config's workflows.
func (c *Config) GetWorkflowConfig() *workflow.Config {
	if len(c.Workflows) == 0 {
		return nil
	}
	return &workflow.Config{
		Workflows: c.Workflows,
	}
}

// GetEnabledWorkflows returns all non-disabled workflows.
func (c *Config) GetEnabledWorkflows() map[string]workflow.Definition {
	result := make(map[string]workflow.Definition)
	for name, def := range c.Workflows {
		if !def.Disabled {
			result[name] = def
		}
	}
	return result
}

// WorkflowTotalWeight returns the sum of all enabled workflow weights.
func (c *Config) WorkflowTotalWeight() int {
	total := 0
	for _, def := range c.Workflows {
		if !def.Disabled {
			weight := def.Weight
			if weight <= 0 {
				weight = 1
			}
			total += weight
		}
	}
	return total
}

// HasAssertions returns true if any assertions are configured.
func (c *Config) HasAssertions() bool {
	global := c.Assertions.Global

	// Check global assertions
	if global != nil {
		if global.MaxErrorRate != nil ||
			global.MinSuccessRate != nil ||
			global.MaxP50Latency > 0 ||
			global.MaxP95Latency > 0 ||
			global.MaxP99Latency > 0 ||
			global.MaxAvgLatency > 0 ||
			global.MinThroughput != nil {
			return true
		}
	}

	// Check endpoint assertions
	for _, assertions := range c.Assertions.EndpointOverrides {
		if assertions.Disabled {
			continue
		}
		if assertions.MaxErrorRate != nil ||
			assertions.MinSuccessRate != nil ||
			assertions.MaxP50Latency > 0 ||
			assertions.MaxP95Latency > 0 ||
			assertions.MaxP99Latency > 0 ||
			assertions.MaxAvgLatency > 0 ||
			assertions.MinThroughput != nil {
			return true
		}
	}

	return false
}

// ShouldExitOnAssertionFailure returns whether the program should exit
// with non-zero exit code when assertions fail.
func (c *Config) ShouldExitOnAssertionFailure() bool {
	if c.Assertions.ExitOnFailure == nil {
		return true // Default to true
	}
	return *c.Assertions.ExitOnFailure
}
