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
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Path is the output file path (for file outputs).
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// ReportInterval is how often to print progress reports.
	// Default: 10s
	ReportInterval time.Duration `yaml:"reportInterval,omitempty" json:"reportInterval,omitempty"`

	// Verbose enables verbose output.
	Verbose bool `yaml:"verbose,omitempty" json:"verbose,omitempty"`
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

	// Apply output defaults
	if c.Output.ReportInterval == 0 {
		c.Output.ReportInterval = 10 * time.Second
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
