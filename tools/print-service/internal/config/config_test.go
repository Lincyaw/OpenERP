package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromBytes_ValidConfig(t *testing.T) {
	yaml := `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/api/v1/printing/ws"
  tenant_id: "tenant-123"
  api_token: "token-abc"
  timeout: 30s
  reconnect_interval: 5s

printing:
  default_printer: "HP LaserJet"
  renderer: chromedp
  protocol: ipp
  ipp_server: "http://localhost:631"

logging:
  level: info
  file: ""

health:
  enabled: true
  port: 9999
  path: /health
`

	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	// Verify server config
	if cfg.Server.APIURL != "http://localhost:8080" {
		t.Errorf("Expected APIURL 'http://localhost:8080', got '%s'", cfg.Server.APIURL)
	}
	if cfg.Server.WebSocketURL != "ws://localhost:8080/api/v1/printing/ws" {
		t.Errorf("Expected WebSocketURL 'ws://localhost:8080/api/v1/printing/ws', got '%s'", cfg.Server.WebSocketURL)
	}
	if cfg.Server.TenantID != "tenant-123" {
		t.Errorf("Expected TenantID 'tenant-123', got '%s'", cfg.Server.TenantID)
	}
	if cfg.Server.APIToken != "token-abc" {
		t.Errorf("Expected APIToken 'token-abc', got '%s'", cfg.Server.APIToken)
	}
	if cfg.Server.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout 30s, got %v", cfg.Server.Timeout)
	}

	// Verify printing config
	if cfg.Printing.DefaultPrinter != "HP LaserJet" {
		t.Errorf("Expected DefaultPrinter 'HP LaserJet', got '%s'", cfg.Printing.DefaultPrinter)
	}
	if cfg.Printing.Renderer != "chromedp" {
		t.Errorf("Expected Renderer 'chromedp', got '%s'", cfg.Printing.Renderer)
	}
	if cfg.Printing.Protocol != "ipp" {
		t.Errorf("Expected Protocol 'ipp', got '%s'", cfg.Printing.Protocol)
	}

	// Verify health config
	if !cfg.Health.Enabled {
		t.Error("Expected Health.Enabled to be true")
	}
	if cfg.Health.Port != 9999 {
		t.Errorf("Expected Health.Port 9999, got %d", cfg.Health.Port)
	}
}

func TestLoadFromBytes_MissingRequired(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "missing api_url",
			yaml: `
server:
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "tenant-123"
  api_token: "token-abc"
printing:
  renderer: chromedp
  protocol: ipp
  ipp_server: "http://localhost:631"
logging:
  level: info
`,
			want: "server.api_url is required",
		},
		{
			name: "missing websocket_url",
			yaml: `
server:
  api_url: "http://localhost:8080"
  tenant_id: "tenant-123"
  api_token: "token-abc"
printing:
  renderer: chromedp
  protocol: ipp
  ipp_server: "http://localhost:631"
logging:
  level: info
`,
			want: "server.websocket_url is required",
		},
		{
			name: "missing tenant_id",
			yaml: `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/ws"
  api_token: "token-abc"
printing:
  renderer: chromedp
  protocol: ipp
  ipp_server: "http://localhost:631"
logging:
  level: info
`,
			want: "server.tenant_id is required",
		},
		{
			name: "missing api_token",
			yaml: `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "tenant-123"
printing:
  renderer: chromedp
  protocol: ipp
  ipp_server: "http://localhost:631"
logging:
  level: info
`,
			want: "server.api_token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFromBytes([]byte(tt.yaml))
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.want, err.Error())
			}
		})
	}
}

func TestLoadFromBytes_InvalidValues(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "invalid renderer",
			yaml: `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "tenant-123"
  api_token: "token-abc"
printing:
  renderer: invalid
  protocol: ipp
  ipp_server: "http://localhost:631"
logging:
  level: info
`,
			want: "printing.renderer must be one of",
		},
		{
			name: "invalid protocol",
			yaml: `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "tenant-123"
  api_token: "token-abc"
printing:
  renderer: chromedp
  protocol: invalid
logging:
  level: info
`,
			want: "printing.protocol must be one of",
		},
		{
			name: "invalid log level",
			yaml: `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "tenant-123"
  api_token: "token-abc"
printing:
  renderer: chromedp
  protocol: ipp
  ipp_server: "http://localhost:631"
logging:
  level: invalid
`,
			want: "logging.level must be one of",
		},
		{
			name: "ipp without server",
			yaml: `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "tenant-123"
  api_token: "token-abc"
printing:
  renderer: chromedp
  protocol: ipp
logging:
  level: info
`,
			want: "printing.ipp_server is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFromBytes([]byte(tt.yaml))
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.want, err.Error())
			}
		})
	}
}

func TestLoadFromBytes_Defaults(t *testing.T) {
	yaml := `
server:
  api_url: "http://localhost:8080"
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "tenant-123"
  api_token: "token-abc"
printing:
  ipp_server: "http://localhost:631"
`

	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	// Check defaults
	if cfg.Server.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout 30s, got %v", cfg.Server.Timeout)
	}
	if cfg.Server.ReconnectInterval != 5*time.Second {
		t.Errorf("Expected default ReconnectInterval 5s, got %v", cfg.Server.ReconnectInterval)
	}
	if cfg.Printing.Renderer != "chromedp" {
		t.Errorf("Expected default Renderer 'chromedp', got '%s'", cfg.Printing.Renderer)
	}
	if cfg.Printing.Protocol != "ipp" {
		t.Errorf("Expected default Protocol 'ipp', got '%s'", cfg.Printing.Protocol)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default Level 'info', got '%s'", cfg.Logging.Level)
	}
	if cfg.Health.Port != 9999 {
		t.Errorf("Expected default Health.Port 9999, got %d", cfg.Health.Port)
	}
	if cfg.Health.Path != "/health" {
		t.Errorf("Expected default Health.Path '/health', got '%s'", cfg.Health.Path)
	}
}

func TestLoadFromBytes_EnvExpansion(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_API_URL", "http://test-server:8080")
	os.Setenv("TEST_TENANT_ID", "test-tenant")
	defer os.Unsetenv("TEST_API_URL")
	defer os.Unsetenv("TEST_TENANT_ID")

	yaml := `
server:
  api_url: "${TEST_API_URL}"
  websocket_url: "ws://localhost:8080/ws"
  tenant_id: "${TEST_TENANT_ID}"
  api_token: "token-abc"
printing:
  ipp_server: "http://localhost:631"
`

	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	if cfg.Server.APIURL != "http://test-server:8080" {
		t.Errorf("Expected APIURL 'http://test-server:8080', got '%s'", cfg.Server.APIURL)
	}
	if cfg.Server.TenantID != "test-tenant" {
		t.Errorf("Expected TenantID 'test-tenant', got '%s'", cfg.Server.TenantID)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !contains(err.Error(), "configuration file not found") {
		t.Errorf("Expected 'configuration file not found' error, got '%s'", err.Error())
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.Server.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout 30s, got %v", cfg.Server.Timeout)
	}
	if cfg.Printing.Renderer != "chromedp" {
		t.Errorf("Expected default Renderer 'chromedp', got '%s'", cfg.Printing.Renderer)
	}
	if cfg.Health.Port != 9999 {
		t.Errorf("Expected default Health.Port 9999, got %d", cfg.Health.Port)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
