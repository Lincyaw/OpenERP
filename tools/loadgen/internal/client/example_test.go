package client

import (
	"fmt"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
)

// Example_clientCreation demonstrates how to configure the HTTP client.
func Example_clientCreation() {
	// This example shows how to configure and use the HTTP client
	// Note: This is a demonstration - actual ERP connection would require a running server

	// Configure target system
	targetCfg := config.TargetConfig{
		BaseURL:    "http://localhost:8080",
		APIVersion: "v1",
		Timeout:    30 * time.Second,
	}

	// Configure authentication with bearer token (doesn't require server)
	authCfg := &config.AuthConfig{
		Type: "bearer",
		Bearer: &config.BearerConfig{
			Token: "example-token",
		},
	}

	// Create client with bearer auth (no server needed)
	client, err := NewClient(targetCfg, authCfg, nil)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	fmt.Printf("Client created with base URL: %s\n", client.GetBaseURL())
	fmt.Printf("Auth manager initialized: %v\n", client.GetAuthManager() != nil)

	// Output:
	// Client created with base URL: http://localhost:8080
	// Auth manager initialized: true
}
