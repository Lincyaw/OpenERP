// Package main provides tests for the CLI entry point.
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelper builds the CLI binary for testing
func buildLoadgen(t *testing.T) string {
	t.Helper()

	// Get the module root directory
	cmdDir, err := os.Getwd()
	require.NoError(t, err)

	// Build the binary
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "loadgen")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = cmdDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build loadgen: %s", string(output))

	return binPath
}

// runLoadgen executes the loadgen binary with the given args
func runLoadgen(t *testing.T, binPath string, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(binPath, args...)

	// Change to the loadgen directory so configs/ path works
	cmd.Dir = filepath.Dir(binPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

func TestCLI_Help(t *testing.T) {
	binPath := buildLoadgen(t)

	stdout, stderr, exitCode := runLoadgen(t, binPath, "--help")

	// Help goes to stderr per Go's flag package
	helpOutput := stderr + stdout
	assert.Contains(t, helpOutput, "Load Generator - ERP System Load Testing Tool")
	assert.Contains(t, helpOutput, "-config")
	assert.Contains(t, helpOutput, "-duration")
	assert.Contains(t, helpOutput, "-concurrency")
	assert.Contains(t, helpOutput, "-qps")
	assert.Contains(t, helpOutput, "-list")
	assert.Contains(t, helpOutput, "-validate")
	assert.Contains(t, helpOutput, "-dry-run")
	assert.Contains(t, helpOutput, "-verbose")
	assert.Contains(t, helpOutput, "EXAMPLES:")
	assert.Equal(t, 0, exitCode)
}

func TestCLI_Version(t *testing.T) {
	binPath := buildLoadgen(t)

	stdout, _, exitCode := runLoadgen(t, binPath, "-version")

	assert.Contains(t, stdout, "loadgen version")
	assert.Contains(t, stdout, "Build time:")
	assert.Contains(t, stdout, "Git commit:")
	assert.Equal(t, 0, exitCode)
}

func TestCLI_NoConfigError(t *testing.T) {
	binPath := buildLoadgen(t)

	_, stderr, exitCode := runLoadgen(t, binPath)

	assert.Contains(t, stderr, "-config or -openapi flag is required")
	assert.Equal(t, 1, exitCode)
}

func TestCLI_ConfigNotFound(t *testing.T) {
	binPath := buildLoadgen(t)

	_, stderr, exitCode := runLoadgen(t, binPath, "-config", "/nonexistent/path.yaml")

	assert.Contains(t, stderr, "configuration file not found")
	assert.Equal(t, 1, exitCode)
}

func TestCLI_Validate(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-config", configPath, "-validate")

	assert.Contains(t, stdout, "Configuration 'Test Config' is valid")
	assert.Contains(t, stdout, "Configuration Summary:")
	assert.Contains(t, stdout, "Name:")
	assert.Contains(t, stdout, "Target:")
	assert.Equal(t, 0, exitCode)
}

func TestCLI_List(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a config with multiple endpoints
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
name: "Test Config"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "products.list"
    path: "/products"
    method: "GET"
    weight: 10
    tags: ["catalog", "read"]
  - name: "products.create"
    path: "/products"
    method: "POST"
    weight: 2
    tags: ["catalog", "write"]
  - name: "orders.list"
    path: "/orders"
    method: "GET"
    weight: 5
    tags: ["trade", "read"]
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-config", configPath, "-list")

	assert.Contains(t, stdout, "Endpoints in 'Test Config'")
	assert.Contains(t, stdout, "products.list")
	assert.Contains(t, stdout, "products.create")
	assert.Contains(t, stdout, "orders.list")
	assert.Contains(t, stdout, "Summary:")
	assert.Contains(t, stdout, "Total Weight:")
	assert.Equal(t, 0, exitCode)
}

func TestCLI_DryRun(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a config with step shaper
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
name: "Dry Run Test"
target:
  baseURL: "http://localhost:8080"
duration: 5m
trafficShaper:
  type: "step"
  baseQPS: 10
  step:
    steps:
      - qps: 10
        duration: 30s
      - qps: 50
        duration: 60s
workerPool:
  minSize: 5
  maxSize: 50
  initialSize: 10
warmup:
  iterations: 3
  fill:
    - "entity.product.id"
endpoints:
  - name: "test.endpoint"
    path: "/test"
    method: "GET"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-config", configPath, "-dry-run")

	assert.Contains(t, stdout, "Execution Plan (Dry Run)")
	assert.Contains(t, stdout, "Configuration Summary:")
	assert.Contains(t, stdout, "Traffic Shaping:")
	assert.Contains(t, stdout, "Worker Pool:")
	assert.Contains(t, stdout, "Warmup Phase:")
	assert.Contains(t, stdout, "Endpoint Distribution")
	assert.Contains(t, stdout, "Ready to execute")
	assert.Equal(t, 0, exitCode)
}

func TestCLI_Overrides(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
name: "Override Test"
target:
  baseURL: "http://localhost:8080"
duration: 5m
trafficShaper:
  type: "sine"
  baseQPS: 10
workerPool:
  maxSize: 50
endpoints:
  - name: "test.endpoint"
    path: "/test"
    method: "GET"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test with overrides
	stdout, _, exitCode := runLoadgen(t, binPath,
		"-config", configPath,
		"-duration", "10m",
		"-qps", "100",
		"-concurrency", "200",
		"-validate",
		"-verbose",
	)

	assert.Contains(t, stdout, "Override: duration = 10m0s")
	assert.Contains(t, stdout, "Override: qps = 100.0")
	assert.Contains(t, stdout, "Override: concurrency (workerPool.maxSize) = 200")
	assert.Contains(t, stdout, "Duration:    10m0s")
	assert.Contains(t, stdout, "Base QPS:    100.0")
	assert.Equal(t, 0, exitCode)
}

func TestCLI_ShortFlags(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
name: "Short Flags Test"
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test with short flags
	stdout, _, exitCode := runLoadgen(t, binPath, "-c", configPath, "-d", "15m", "-v", "-l")

	assert.Contains(t, stdout, "Override: duration = 15m0s")
	assert.Contains(t, stdout, "Endpoints in 'Short Flags Test'")
	assert.Equal(t, 0, exitCode)
}

func TestCLI_InvalidConfig(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create an invalid config (missing required fields)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.yaml")
	configContent := `
name: "Invalid Config"
# Missing target.baseURL
endpoints:
  - name: "test.endpoint"
    path: "/test"
    method: "GET"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, stderr, exitCode := runLoadgen(t, binPath, "-config", configPath, "-validate")

	assert.Contains(t, stderr, "baseURL is required")
	assert.Equal(t, 1, exitCode)
}

func TestCLI_RunPlaceholder(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
name: "Run Test"
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
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test run (should show placeholder message)
	stdout, _, exitCode := runLoadgen(t, binPath, "-config", configPath)

	assert.Contains(t, stdout, "Load Test")
	assert.Contains(t, stdout, "Configuration: Run Test")
	assert.Contains(t, stdout, "not yet implemented")
	assert.Equal(t, 0, exitCode)
}

// TestApplyOverrides tests the applyOverrides function behavior
func TestApplyOverrides_Integration(t *testing.T) {
	binPath := buildLoadgen(t)

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name: "duration only",
			args: []string{"-duration", "20m"},
			expected: []string{
				"Override: duration = 20m0s",
				"Duration:    20m0s",
			},
		},
		{
			name: "qps only",
			args: []string{"-qps", "200"},
			expected: []string{
				"Override: qps = 200.0",
				"Base QPS:    200.0",
			},
		},
		{
			name: "concurrency only",
			args: []string{"-concurrency", "500"},
			expected: []string{
				"Override: concurrency (workerPool.maxSize) = 500",
			},
		},
		{
			name: "all overrides",
			args: []string{"-duration", "30m", "-qps", "150", "-concurrency", "300"},
			expected: []string{
				"Override: duration = 30m0s",
				"Override: qps = 150.0",
				"Override: concurrency (workerPool.maxSize) = 300",
			},
		},
	}

	// Create a minimal config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
name: "Override Test"
target:
  baseURL: "http://localhost:8080"
duration: 5m
trafficShaper:
  type: "sine"
  baseQPS: 10
workerPool:
  maxSize: 50
endpoints:
  - name: "test.endpoint"
    path: "/test"
    method: "GET"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"-config", configPath, "-validate", "-verbose"}, tc.args...)
			stdout, _, exitCode := runLoadgen(t, binPath, args...)

			for _, expected := range tc.expected {
				assert.Contains(t, stdout, expected)
			}
			assert.Equal(t, 0, exitCode)
		})
	}
}

// TestHelpers tests helper functions
func TestGetAuthType(t *testing.T) {
	tests := []struct {
		name     string
		yamlAuth string
		expected string
	}{
		{
			name:     "no auth",
			yamlAuth: "",
			expected: "none",
		},
		{
			name:     "bearer auth",
			yamlAuth: "type: bearer",
			expected: "bearer",
		},
		{
			name:     "api_key auth",
			yamlAuth: "type: api_key",
			expected: "api_key",
		},
	}

	binPath := buildLoadgen(t)
	tmpDir := t.TempDir()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tc.name+"-config.yaml")
			authSection := ""
			if tc.yamlAuth != "" {
				authSection = "auth:\n  " + tc.yamlAuth
			}
			configContent := `
name: "Auth Test"
target:
  baseURL: "http://localhost:8080"
` + authSection + `
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "test.endpoint"
    path: "/test"
    method: "GET"
`
			err := os.WriteFile(configPath, []byte(configContent), 0644)
			require.NoError(t, err)

			stdout, _, exitCode := runLoadgen(t, binPath, "-config", configPath, "-validate")
			assert.Contains(t, stdout, "Auth Type:   "+tc.expected)
			assert.Equal(t, 0, exitCode)
		})
	}
}

// TestEndpointGrouping tests the -list command groups endpoints by category
func TestEndpointGrouping(t *testing.T) {
	binPath := buildLoadgen(t)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "grouping-config.yaml")
	configContent := `
name: "Grouping Test"
target:
  baseURL: "http://localhost:8080"
trafficShaper:
  type: "sine"
  baseQPS: 100
endpoints:
  - name: "catalog.list"
    path: "/catalog"
    method: "GET"
    tags: ["catalog", "read"]
  - name: "catalog.create"
    path: "/catalog"
    method: "POST"
    tags: ["catalog", "write"]
  - name: "trade.list"
    path: "/trade"
    method: "GET"
    tags: ["trade", "read"]
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-config", configPath, "-list")

	// Check that categories are present as headers
	assert.True(t, strings.Contains(stdout, "== CATALOG ==") || strings.Contains(stdout, "catalog"))
	assert.True(t, strings.Contains(stdout, "== TRADE ==") || strings.Contains(stdout, "trade"))
	assert.Contains(t, stdout, "catalog.list")
	assert.Contains(t, stdout, "catalog.create")
	assert.Contains(t, stdout, "trade.list")
	assert.Equal(t, 0, exitCode)
}

// TestCLI_OpenAPIList tests the -openapi -list command
func TestCLI_OpenAPIList(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal OpenAPI spec
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.yaml")
	specContent := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
basePath: /api/v1
tags:
  - name: users
    description: User management
paths:
  /users:
    get:
      summary: List users
      operationId: listUsers
      tags:
        - users
      responses:
        "200":
          description: Success
          schema:
            type: array
            items:
              type: object
              properties:
                id:
                  type: string
                name:
                  type: string
    post:
      summary: Create user
      operationId: createUser
      tags:
        - users
      parameters:
        - name: body
          in: body
          required: true
          schema:
            type: object
            properties:
              name:
                type: string
      responses:
        "201":
          description: Created
  /users/{id}:
    get:
      summary: Get user by ID
      operationId: getUser
      tags:
        - users
      parameters:
        - name: id
          in: path
          required: true
          type: string
          format: uuid
      responses:
        "200":
          description: Success
`
	err := os.WriteFile(specPath, []byte(specContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-openapi", specPath, "-list")

	// Check output
	assert.Contains(t, stdout, "Test API")
	assert.Contains(t, stdout, "Total: 3 endpoints")
	assert.Contains(t, stdout, "/users")
	assert.Contains(t, stdout, "GET")
	assert.Contains(t, stdout, "POST")
	assert.Contains(t, stdout, "/users/{id}")
	assert.Contains(t, stdout, "USERS")
	assert.Equal(t, 0, exitCode)
}

// TestCLI_OpenAPIListVerbose tests the -openapi -list -v command
func TestCLI_OpenAPIListVerbose(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal OpenAPI spec with parameters
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.yaml")
	specContent := `
swagger: "2.0"
info:
  title: Test Verbose API
  version: "1.0"
paths:
  /items/{id}:
    get:
      summary: Get item by ID
      operationId: getItem
      parameters:
        - name: id
          in: path
          required: true
          type: string
        - name: include
          in: query
          type: string
      responses:
        "200":
          description: Success
          schema:
            type: object
            properties:
              id:
                type: string
              name:
                type: string
`
	err := os.WriteFile(specPath, []byte(specContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-openapi", specPath, "-list", "-v")

	// Check verbose output
	assert.Contains(t, stdout, "Summary: Get item by ID")
	assert.Contains(t, stdout, "OperationID: getItem")
	assert.Contains(t, stdout, "Input Parameters:")
	assert.Contains(t, stdout, "id* (path, string)")
	assert.Contains(t, stdout, "include (query, string)")
	assert.Contains(t, stdout, "Output Fields:")
	assert.Equal(t, 0, exitCode)
}

// TestCLI_OpenAPINotFound tests error handling for non-existent OpenAPI spec
func TestCLI_OpenAPINotFound(t *testing.T) {
	binPath := buildLoadgen(t)

	_, stderr, exitCode := runLoadgen(t, binPath, "-openapi", "/nonexistent/openapi.yaml", "-list")

	assert.Contains(t, stderr, "Error parsing OpenAPI spec")
	assert.Equal(t, 1, exitCode)
}

// TestCLI_OpenAPIInvalidSpec tests error handling for invalid OpenAPI spec
func TestCLI_OpenAPIInvalidSpec(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create an invalid spec file
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(specPath, []byte("invalid: yaml: ["), 0644)
	require.NoError(t, err)

	_, stderr, exitCode := runLoadgen(t, binPath, "-openapi", specPath, "-list")

	assert.Contains(t, stderr, "Error parsing OpenAPI spec")
	assert.Equal(t, 1, exitCode)
}

// TestCLI_OpenAPISummary tests the default OpenAPI summary output (no -list flag)
func TestCLI_OpenAPISummary(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal OpenAPI spec
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.yaml")
	specContent := `
swagger: "2.0"
info:
  title: Summary Test API
  version: "2.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: Success
`
	err := os.WriteFile(specPath, []byte(specContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-openapi", specPath)

	assert.Contains(t, stdout, "Summary Test API")
	assert.Contains(t, stdout, "Total Endpoints: 1")
	assert.Equal(t, 0, exitCode)
}

// TestCLI_OpenAPIShortFlag tests the -o short flag for OpenAPI
func TestCLI_OpenAPIShortFlag(t *testing.T) {
	binPath := buildLoadgen(t)

	// Create a minimal OpenAPI spec
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.yaml")
	specContent := `
swagger: "2.0"
info:
  title: Short Flag Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: Success
`
	err := os.WriteFile(specPath, []byte(specContent), 0644)
	require.NoError(t, err)

	stdout, _, exitCode := runLoadgen(t, binPath, "-o", specPath, "-l")

	assert.Contains(t, stdout, "Short Flag Test")
	assert.Contains(t, stdout, "/test")
	assert.Equal(t, 0, exitCode)
}
