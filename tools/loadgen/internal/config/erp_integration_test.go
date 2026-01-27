package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestERPConfigLoads verifies that the ERP configuration file loads and validates correctly.
func TestERPConfigLoads(t *testing.T) {
	// Find the configs directory relative to this test file
	configPath := findERPConfigPath(t)

	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err, "ERP config should load without errors")

	// Basic assertions
	assert.Equal(t, "ERP Load Test", cfg.Name)
	assert.Equal(t, "http://localhost:8080", cfg.Target.BaseURL)
	assert.Equal(t, "v1", cfg.Target.APIVersion)
}

// TestERPConfigHasAllDomains verifies that endpoints exist for all ERP domains.
func TestERPConfigHasAllDomains(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	expectedDomains := []string{
		"auth",
		"catalog",
		"partner",
		"inventory",
		"trade",
		"finance",
		"reports",
		"system",
	}

	for _, domain := range expectedDomains {
		endpoints := cfg.GetEndpointsByTag(domain)
		assert.NotEmpty(t, endpoints, "Domain %s should have at least one endpoint", domain)
	}
}

// TestERPConfigWarmupFill verifies that warmup fill types have corresponding producers.
func TestERPConfigWarmupFill(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	// Each warmup fill type should have at least one producer endpoint
	for _, fillType := range cfg.Warmup.Fill {
		producers := cfg.GetProducerEndpoints(fillType)
		assert.NotEmpty(t, producers,
			"Warmup fill type %s should have at least one producer endpoint", fillType)
	}
}

// TestERPConfigSemanticConsistency verifies that consumed types have producers.
func TestERPConfigSemanticConsistency(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	// Collect all consumed semantic types
	consumedTypes := make(map[circuit.SemanticType][]string) // type -> endpoint names
	for _, ep := range cfg.Endpoints {
		for _, ct := range ep.Consumes {
			consumedTypes[ct] = append(consumedTypes[ct], ep.Name)
		}
	}

	// Each consumed type should have at least one producer
	for consumedType, consumers := range consumedTypes {
		producers := cfg.GetProducerEndpoints(consumedType)
		assert.NotEmpty(t, producers,
			"Semantic type %s consumed by %v should have at least one producer",
			consumedType, consumers)
	}
}

// TestERPConfigWeightDistribution verifies reasonable weight distribution.
func TestERPConfigWeightDistribution(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	totalWeight := cfg.TotalWeight()
	assert.Greater(t, totalWeight, 0, "Total weight should be greater than 0")

	// Count read vs write operations
	readWeight := 0
	writeWeight := 0
	for _, ep := range cfg.GetEnabledEndpoints() {
		for _, tag := range ep.Tags {
			if tag == "read" {
				readWeight += ep.Weight
				break
			}
			if tag == "write" {
				writeWeight += ep.Weight
				break
			}
		}
	}

	// Read operations should dominate (realistic workload)
	if readWeight > 0 && writeWeight > 0 {
		readRatio := float64(readWeight) / float64(readWeight+writeWeight)
		assert.Greater(t, readRatio, 0.6,
			"Read operations should be at least 60%% of the workload, got %.2f%%", readRatio*100)
	}
}

// TestERPConfigScenarios verifies that scenarios reference valid endpoints.
func TestERPConfigScenarios(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	for _, scenario := range cfg.Scenarios {
		for _, epName := range scenario.Endpoints {
			ep := cfg.GetEndpointByName(epName)
			assert.NotNil(t, ep,
				"Scenario %s references non-existent endpoint %s", scenario.Name, epName)
		}
	}
}

// TestERPConfigEndpointPaths verifies that endpoint paths follow ERP API conventions.
func TestERPConfigEndpointPaths(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	for _, ep := range cfg.Endpoints {
		// Path should start with /
		assert.True(t, len(ep.Path) > 0 && ep.Path[0] == '/',
			"Endpoint %s path should start with /", ep.Name)

		// Path should not have double slashes
		assert.NotContains(t, ep.Path, "//",
			"Endpoint %s path should not have double slashes", ep.Name)
	}
}

// TestERPConfigHTTPMethods verifies that HTTP methods are valid.
func TestERPConfigHTTPMethods(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	validMethods := map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"PATCH":  true,
		"DELETE": true,
	}

	for _, ep := range cfg.Endpoints {
		assert.True(t, validMethods[ep.Method],
			"Endpoint %s has invalid HTTP method: %s", ep.Name, ep.Method)
	}
}

// TestERPConfigAuth verifies authentication configuration.
func TestERPConfigAuth(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	assert.Equal(t, "bearer", cfg.Auth.Type)
	require.NotNil(t, cfg.Auth.Login)
	assert.Equal(t, "/auth/login", cfg.Auth.Login.Endpoint)
	assert.NotEmpty(t, cfg.Auth.Login.Username)
	assert.NotEmpty(t, cfg.Auth.Login.Password)
	assert.NotEmpty(t, cfg.Auth.Login.TokenPath)
}

// TestERPConfigTrafficShaper verifies traffic shaper configuration.
func TestERPConfigTrafficShaper(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	assert.NotEmpty(t, cfg.TrafficShaper.Type)
	assert.Greater(t, cfg.TrafficShaper.BaseQPS, 0.0)
}

// TestERPConfigWorkerPool verifies worker pool configuration.
func TestERPConfigWorkerPool(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	assert.Greater(t, cfg.WorkerPool.MinSize, 0)
	assert.Greater(t, cfg.WorkerPool.MaxSize, cfg.WorkerPool.MinSize)
}

// TestERPConfigMinimumEndpointCoverage verifies essential endpoints exist.
func TestERPConfigMinimumEndpointCoverage(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	essentialEndpoints := []string{
		// Auth
		"auth.login",
		"auth.me",

		// Catalog
		"catalog.products.list",
		"catalog.products.get",
		"catalog.categories.list",

		// Partner
		"partner.customers.list",
		"partner.suppliers.list",
		"partner.warehouses.list",

		// Inventory
		"inventory.items.list",

		// Trade
		"trade.sales_orders.list",
		"trade.purchase_orders.list",

		// Finance
		"finance.receivables.list",
		"finance.payables.list",

		// System
		"system.ping",
	}

	for _, epName := range essentialEndpoints {
		ep := cfg.GetEndpointByName(epName)
		assert.NotNil(t, ep, "Essential endpoint %s should exist", epName)
	}
}

// TestERPConfigProducerEndpointsExist verifies that producer endpoints are properly tagged.
func TestERPConfigProducerEndpointsExist(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	// Check that we have at least some producer endpoints
	producerCount := 0
	for _, ep := range cfg.Endpoints {
		if len(ep.Produces) > 0 {
			producerCount++
		}
	}

	assert.Greater(t, producerCount, 5,
		"Should have at least 5 producer endpoints for warmup")
}

// TestERPConfigEndpointNaming verifies endpoint naming convention.
func TestERPConfigEndpointNaming(t *testing.T) {
	configPath := findERPConfigPath(t)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	for _, ep := range cfg.Endpoints {
		// Names should use dot notation for hierarchy
		assert.Contains(t, ep.Name, ".",
			"Endpoint name %s should use dot notation (e.g., domain.resource.action)", ep.Name)

		// Names should not contain spaces or special characters (other than dots and underscores)
		for _, c := range ep.Name {
			validChar := (c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '.' ||
				c == '_'
			if !validChar {
				t.Errorf("Endpoint name %s contains invalid character '%c'", ep.Name, c)
			}
		}
	}
}

// findERPConfigPath finds the ERP config file path.
func findERPConfigPath(t *testing.T) string {
	t.Helper()

	// Try multiple possible paths
	paths := []string{
		"../../configs/erp.yaml",
		"../../../configs/erp.yaml",
		"configs/erp.yaml",
	}

	// Get current working directory
	cwd, _ := os.Getwd()

	for _, p := range paths {
		fullPath := filepath.Join(cwd, p)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	// Try from GOPATH or module root
	moduleRoot := findModuleRoot(cwd)
	if moduleRoot != "" {
		configPath := filepath.Join(moduleRoot, "tools", "loadgen", "configs", "erp.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	t.Skip("Could not find ERP config file - skipping integration test")
	return ""
}

// findModuleRoot finds the Go module root by looking for go.mod.
func findModuleRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
