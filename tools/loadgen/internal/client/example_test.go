package client

import (
	"fmt"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
)

// Example_client demonstrates how to use the HTTP client to login to the ERP system.
func Example_client() {
	// This example shows how to configure and use the HTTP client
	// Note: This is a demonstration - actual ERP connection would require a running server

	// Configure target system
	targetCfg := config.TargetConfig{
		BaseURL: "http://localhost:8080",
		Timeout: 30 * time.Second,
	}

	// Configure authentication
	authCfg := &config.AuthConfig{
		Type: "login",
		Login: &config.LoginConfig{
			Endpoint: "/auth/login",
			Username: "admin",
			Password: "admin123",
		},
	}

	// Create client (this would fail without a running server)
	client, err := NewClient(targetCfg, authCfg, nil)
	if err != nil {
		// In a real scenario, handle the error appropriately
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer client.auth.Stop()

	// Verify authentication
	if client.auth.IsAuthenticated() {
		fmt.Println("Successfully logged in to ERP system")
		fmt.Printf("Access token: %s\n", client.auth.GetAccessToken())
	}

	// Output: Failed to create client: creating auth manager: initial login failed: login request failed: Post "http://localhost:8080/auth/login": dial tcp [::1]:8080: connect: connection refused
}