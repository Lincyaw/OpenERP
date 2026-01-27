package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromBytes_MinimalConfig(t *testing.T) {
	yaml := `
name: "Test Config"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test.endpoint"
    path: "/test"
    method: "GET"
`
	cfg, err := LoadFromBytes([]byte(yaml))
	require.NoError(t, err)
	assert.Equal(t, "Test Config", cfg.Name)
	assert.Equal(t, "http://localhost:8080", cfg.Target.BaseURL)
	assert.Equal(t, "v1", cfg.Target.APIVersion)        // Default
	assert.Equal(t, 30*time.Second, cfg.Target.Timeout) // Default
	assert.Equal(t, 5*time.Minute, cfg.Duration)        // Default
	assert.Len(t, cfg.Endpoints, 1)
	assert.Equal(t, 1, cfg.Endpoints[0].Weight) // Default
}

func TestLoadFromBytes_FullConfig(t *testing.T) {
	yaml := `
name: "Full Test Config"
description: "A comprehensive test config"
version: "2.0"
target:
  baseURL: "http://localhost:8080"
  apiVersion: "v2"
  timeout: 60s
  tlsSkipVerify: true
  headers:
    X-Custom-Header: "test-value"
auth:
  type: "bearer"
  login:
    endpoint: "/auth/login"
    username: "admin"
    password: "secret"
    tokenPath: "$.token"
duration: 10m
warmup:
  iterations: 5
  minPoolSize: 3
  fill:
    - "entity.product.id"
    - "entity.customer.id"
trafficShaper:
  type: "step"
  baseQPS: 50
  step:
    steps:
      - qps: 10
        duration: 30s
      - qps: 50
        duration: 60s
rateLimiter:
  type: "token_bucket"
  qps: 100
  burstSize: 20
workerPool:
  minSize: 5
  maxSize: 50
endpoints:
  - name: "products.list"
    path: "/products"
    method: "GET"
    weight: 10
    tags: ["read", "catalog"]
    produces:
      - semanticType: "entity.product.id"
        jsonPath: "$.data[*].id"
        multiple: true
  - name: "products.create"
    path: "/products"
    method: "POST"
    weight: 2
    tags: ["write", "catalog"]
    consumes:
      - "entity.category.id"
    produces:
      - semanticType: "entity.product.id"
        jsonPath: "$.data.id"
scenarios:
  - name: "browse"
    endpoints: ["products.list"]
    weight: 5
`
	cfg, err := LoadFromBytes([]byte(yaml))
	require.NoError(t, err)

	assert.Equal(t, "Full Test Config", cfg.Name)
	assert.Equal(t, "2.0", cfg.Version)
	assert.Equal(t, "v2", cfg.Target.APIVersion)
	assert.Equal(t, 60*time.Second, cfg.Target.Timeout)
	assert.True(t, cfg.Target.TLSSkipVerify)
	assert.Equal(t, "test-value", cfg.Target.Headers["X-Custom-Header"])

	// Auth
	assert.Equal(t, "bearer", cfg.Auth.Type)
	assert.NotNil(t, cfg.Auth.Login)
	assert.Equal(t, "/auth/login", cfg.Auth.Login.Endpoint)

	// Duration
	assert.Equal(t, 10*time.Minute, cfg.Duration)

	// Warmup
	assert.Equal(t, 5, cfg.Warmup.Iterations)
	assert.Equal(t, 3, cfg.Warmup.MinPoolSize)
	assert.Len(t, cfg.Warmup.Fill, 2)

	// Traffic Shaper
	assert.Equal(t, "step", cfg.TrafficShaper.Type)
	assert.NotNil(t, cfg.TrafficShaper.Step)
	assert.Len(t, cfg.TrafficShaper.Step.Steps, 2)

	// Rate Limiter
	assert.Equal(t, "token_bucket", string(cfg.RateLimiter.Type))
	assert.Equal(t, 100.0, cfg.RateLimiter.QPS)

	// Worker Pool
	assert.Equal(t, 5, cfg.WorkerPool.MinSize)
	assert.Equal(t, 50, cfg.WorkerPool.MaxSize)

	// Endpoints
	assert.Len(t, cfg.Endpoints, 2)
	assert.Equal(t, "products.list", cfg.Endpoints[0].Name)
	assert.Equal(t, 10, cfg.Endpoints[0].Weight)
	assert.Contains(t, cfg.Endpoints[0].Tags, "read")

	// Scenarios
	assert.Len(t, cfg.Scenarios, 1)
	assert.Equal(t, "browse", cfg.Scenarios[0].Name)
}

func TestValidate_MissingName(t *testing.T) {
	yaml := `
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test"
    path: "/test"
    method: "GET"
`
	_, err := LoadFromBytes([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidate_MissingBaseURL(t *testing.T) {
	yaml := `
name: "Test"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test"
    path: "/test"
    method: "GET"
`
	_, err := LoadFromBytes([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "baseURL is required")
}

func TestValidate_NoEndpoints(t *testing.T) {
	yaml := `
name: "Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints: []
`
	_, err := LoadFromBytes([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one endpoint is required")
}

func TestValidate_DuplicateEndpointName(t *testing.T) {
	yaml := `
name: "Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test"
    path: "/test1"
    method: "GET"
  - name: "test"
    path: "/test2"
    method: "POST"
`
	_, err := LoadFromBytes([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate endpoint name")
}

func TestValidate_MissingEndpointPath(t *testing.T) {
	yaml := `
name: "Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test"
    method: "GET"
`
	_, err := LoadFromBytes([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestValidate_MissingEndpointMethod(t *testing.T) {
	yaml := `
name: "Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test"
    path: "/test"
`
	_, err := LoadFromBytes([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "method is required")
}

func TestGetEndpointByName(t *testing.T) {
	cfg := &Config{
		Endpoints: []EndpointConfig{
			{Name: "test1", Path: "/test1", Method: "GET"},
			{Name: "test2", Path: "/test2", Method: "POST"},
		},
	}

	ep := cfg.GetEndpointByName("test1")
	require.NotNil(t, ep)
	assert.Equal(t, "/test1", ep.Path)

	ep = cfg.GetEndpointByName("test2")
	require.NotNil(t, ep)
	assert.Equal(t, "/test2", ep.Path)

	ep = cfg.GetEndpointByName("nonexistent")
	assert.Nil(t, ep)
}

func TestGetProducerEndpoints(t *testing.T) {
	cfg := &Config{
		Endpoints: []EndpointConfig{
			{
				Name: "products.list",
				Produces: []ProducesConfig{
					{SemanticType: "entity.product.id"},
				},
			},
			{
				Name: "products.create",
				Produces: []ProducesConfig{
					{SemanticType: "entity.product.id"},
					{SemanticType: "ref.product.code"},
				},
			},
			{
				Name: "customers.list",
				Produces: []ProducesConfig{
					{SemanticType: "entity.customer.id"},
				},
			},
		},
	}

	producers := cfg.GetProducerEndpoints("entity.product.id")
	assert.Len(t, producers, 2)

	producers = cfg.GetProducerEndpoints("entity.customer.id")
	assert.Len(t, producers, 1)

	producers = cfg.GetProducerEndpoints("nonexistent")
	assert.Len(t, producers, 0)
}

func TestGetConsumerEndpoints(t *testing.T) {
	cfg := &Config{
		Endpoints: []EndpointConfig{
			{
				Name:     "products.get",
				Consumes: []circuit.SemanticType{"entity.product.id"},
			},
			{
				Name:     "products.update",
				Consumes: []circuit.SemanticType{"entity.product.id", "entity.category.id"},
			},
			{
				Name:     "customers.get",
				Consumes: []circuit.SemanticType{"entity.customer.id"},
			},
		},
	}

	consumers := cfg.GetConsumerEndpoints("entity.product.id")
	assert.Len(t, consumers, 2)

	consumers = cfg.GetConsumerEndpoints("entity.customer.id")
	assert.Len(t, consumers, 1)

	consumers = cfg.GetConsumerEndpoints("nonexistent")
	assert.Len(t, consumers, 0)
}

func TestGetEnabledEndpoints(t *testing.T) {
	cfg := &Config{
		Endpoints: []EndpointConfig{
			{Name: "ep1", Disabled: false},
			{Name: "ep2", Disabled: true},
			{Name: "ep3", Disabled: false},
		},
	}

	enabled := cfg.GetEnabledEndpoints()
	assert.Len(t, enabled, 2)
}

func TestGetEndpointsByTag(t *testing.T) {
	cfg := &Config{
		Endpoints: []EndpointConfig{
			{Name: "ep1", Tags: []string{"read", "catalog"}},
			{Name: "ep2", Tags: []string{"write", "catalog"}},
			{Name: "ep3", Tags: []string{"read", "inventory"}},
		},
	}

	matches := cfg.GetEndpointsByTag("read")
	assert.Len(t, matches, 2)

	matches = cfg.GetEndpointsByTag("catalog")
	assert.Len(t, matches, 2)

	matches = cfg.GetEndpointsByTag("write")
	assert.Len(t, matches, 1)

	matches = cfg.GetEndpointsByTag("nonexistent")
	assert.Len(t, matches, 0)
}

func TestTotalWeight(t *testing.T) {
	cfg := &Config{
		Endpoints: []EndpointConfig{
			{Name: "ep1", Weight: 10, Disabled: false},
			{Name: "ep2", Weight: 5, Disabled: true},
			{Name: "ep3", Weight: 15, Disabled: false},
		},
	}

	total := cfg.TotalWeight()
	assert.Equal(t, 25, total) // Disabled endpoints excluded
}

func TestLoadFromFile(t *testing.T) {
	// Create a temp file
	content := `
name: "File Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test"
    path: "/test"
    method: "GET"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "File Test", cfg.Name)
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestApplyDefaults_RequiresAuth(t *testing.T) {
	yaml := `
name: "Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "with_auth"
    path: "/test1"
    method: "GET"
  - name: "without_auth"
    path: "/test2"
    method: "GET"
    requiresAuth: false
`
	cfg, err := LoadFromBytes([]byte(yaml))
	require.NoError(t, err)

	// Default should be true
	require.NotNil(t, cfg.Endpoints[0].RequiresAuth)
	assert.True(t, *cfg.Endpoints[0].RequiresAuth)

	// Explicit false should be preserved
	require.NotNil(t, cfg.Endpoints[1].RequiresAuth)
	assert.False(t, *cfg.Endpoints[1].RequiresAuth)
}

func TestParameterConfig(t *testing.T) {
	yaml := `
name: "Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test"
    path: "/products/{id}"
    method: "GET"
    pathParams:
      id:
        semanticType: "entity.product.id"
    queryParams:
      page:
        value: "1"
      filter:
        generator:
          type: "random"
          random:
            type: "string"
            length: 10
`
	cfg, err := LoadFromBytes([]byte(yaml))
	require.NoError(t, err)

	ep := cfg.Endpoints[0]

	// Path params
	idParam, ok := ep.PathParams["id"]
	require.True(t, ok)
	assert.Equal(t, "entity.product.id", string(idParam.SemanticType))

	// Query params - static value
	pageParam, ok := ep.QueryParams["page"]
	require.True(t, ok)
	assert.Equal(t, "1", pageParam.Value)

	// Query params - generator
	filterParam, ok := ep.QueryParams["filter"]
	require.True(t, ok)
	require.NotNil(t, filterParam.Generator)
	assert.Equal(t, "random", filterParam.Generator.Type)
	require.NotNil(t, filterParam.Generator.Random)
	assert.Equal(t, "string", filterParam.Generator.Random.Type)
	assert.Equal(t, 10, filterParam.Generator.Random.Length)
}
