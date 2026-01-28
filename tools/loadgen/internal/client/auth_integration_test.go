package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoginAuth tests login-based authentication flow.
func TestLoginAuth(t *testing.T) {
	// Create test server that simulates ERP auth endpoints
	mux := http.NewServeMux()

	// Login endpoint
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req LoginRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Check credentials
		if req.Username == "admin" && req.Password == "admin123" {
			resp := LoginResponse{
				Data: struct {
					Token TokenResponse `json:"token"`
				}{
					Token: TokenResponse{
						AccessToken:          "test-access-token",
						RefreshToken:         "test-refresh-token",
						AccessTokenExpiresAt: time.Now().Add(1 * time.Hour),
						TokenType:            "Bearer",
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid credentials"}`))
		}
	})

	// Protected endpoint
	mux.HandleFunc("/api/protected", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "unauthorized"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "authenticated"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test successful login
	t.Run("SuccessfulLogin", func(t *testing.T) {
		targetCfg := config.TargetConfig{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
		}

		authCfg := &config.AuthConfig{
			Type: "login",
			Login: &config.LoginConfig{
				Endpoint: "/auth/login",
				Username: "admin",
				Password: "admin123",
			},
		}

		client, err := NewClient(targetCfg, authCfg, nil)
		require.NoError(t, err)
		defer client.auth.Stop()

		// Verify authentication
		assert.True(t, client.auth.IsAuthenticated())
		assert.Equal(t, "test-access-token", client.auth.GetAccessToken())

		// Test authenticated request
		resp, err := client.Get(context.Background(), "/api/protected", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, `{"message": "authenticated"}`, string(resp.Body))
	})

	// Test failed login
	t.Run("FailedLogin", func(t *testing.T) {
		targetCfg := config.TargetConfig{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
		}

		authCfg := &config.AuthConfig{
			Type: "login",
			Login: &config.LoginConfig{
				Endpoint: "/auth/login",
				Username: "wrong",
				Password: "wrong",
			},
		}

		_, err := NewClient(targetCfg, authCfg, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "login failed")
	})
}

// TestAuthTypes tests different authentication types.
func TestAuthTypes(t *testing.T) {
	// Test server that checks different auth methods
	mux := http.NewServeMux()

	mux.HandleFunc("/basic-auth", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "user" || password != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"auth": "basic"}`))
	})

	mux.HandleFunc("/bearer-auth", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer static-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"auth": "bearer"}`))
	})

	mux.HandleFunc("/apikey-auth", func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "my-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"auth": "apikey"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	// Test Basic Auth
	t.Run("BasicAuth", func(t *testing.T) {
		authCfg := &config.AuthConfig{
			Type: "basic",
			Login: &config.LoginConfig{
				Username: "user",
				Password: "pass",
			},
		}

		client, err := NewClient(targetCfg, authCfg, nil)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/basic-auth", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, `{"auth": "basic"}`, string(resp.Body))
	})

	// Test Bearer Auth
	t.Run("BearerAuth", func(t *testing.T) {
		authCfg := &config.AuthConfig{
			Type: "bearer",
			Bearer: &config.BearerConfig{
				Token: "static-token",
			},
		}

		client, err := NewClient(targetCfg, authCfg, nil)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/bearer-auth", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, `{"auth": "bearer"}`, string(resp.Body))
	})

	// Test API Key Auth
	t.Run("APIKeyAuth", func(t *testing.T) {
		authCfg := &config.AuthConfig{
			Type: "api_key",
			APIKey: &config.APIKeyConfig{
				Key:    "my-api-key",
				Header: "X-API-Key",
			},
		}

		client, err := NewClient(targetCfg, authCfg, nil)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/apikey-auth", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, `{"auth": "apikey"}`, string(resp.Body))
	})
}

// TestERPLoginIntegration tests login to actual ERP system.
// To run this test: ERP_TEST_ENABLED=true go test -v -run TestERPLoginIntegration
func TestERPLoginIntegration(t *testing.T) {
	// Skip if ERP testing is not enabled
	if os.Getenv("ERP_TEST_ENABLED") != "true" {
		t.Skip("Skipping ERP integration test - set ERP_TEST_ENABLED=true to run")
	}

	erpHost := os.Getenv("ERP_HOST")
	if erpHost == "" {
		erpHost = "http://localhost:8080"
	}

	erpUser := os.Getenv("ERP_USER")
	if erpUser == "" {
		erpUser = "admin"
	}

	erpPassword := os.Getenv("ERP_PASSWORD")
	if erpPassword == "" {
		erpPassword = "admin123"
	}

	targetCfg := config.TargetConfig{
		BaseURL:    erpHost,
		APIVersion: "", // Don't auto-prefix API version
		Timeout:    10 * time.Second,
	}

	authCfg := &config.AuthConfig{
		Type: "login",
		Login: &config.LoginConfig{
			Endpoint: "/api/v1/auth/login",
			Username: erpUser,
			Password: erpPassword,
		},
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)
	defer client.auth.Stop()

	// Verify authentication
	assert.True(t, client.auth.IsAuthenticated())
	assert.NotEmpty(t, client.auth.GetAccessToken())
	t.Logf("Got access token: %s...", client.auth.GetAccessToken()[:50])

	// Test authenticated request to get current user
	resp, err := client.Get(context.Background(), "/api/v1/identity/auth/me", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	var result struct {
		Data struct {
			User struct {
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
			} `json:"user"`
		} `json:"data"`
	}

	err = json.Unmarshal(resp.Body, &result)
	require.NoError(t, err)
	assert.Equal(t, erpUser, result.Data.User.Username)
	t.Logf("Logged in as: %s (%s)", result.Data.User.Username, result.Data.User.DisplayName)
}

// TestTokenRefresh tests the token refresh functionality.
func TestTokenRefresh(t *testing.T) {
	// Create test server that simulates token refresh
	callCount := 0
	mux := http.NewServeMux()

	// Login endpoint
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := LoginResponse{
			Data: struct {
				Token TokenResponse `json:"token"`
			}{
				Token: TokenResponse{
					AccessToken:          "initial-token",
					RefreshToken:         "initial-refresh-token",
					AccessTokenExpiresAt: time.Now().Add(1 * time.Hour),
					TokenType:            "Bearer",
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	// Refresh endpoint
	mux.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := LoginResponse{
			Data: struct {
				Token TokenResponse `json:"token"`
			}{
				Token: TokenResponse{
					AccessToken:          "refreshed-token",
					RefreshToken:         "new-refresh-token",
					AccessTokenExpiresAt: time.Now().Add(1 * time.Hour),
					TokenType:            "Bearer",
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	authCfg := &config.AuthConfig{
		Type: "login",
		Login: &config.LoginConfig{
			Endpoint:        "/auth/login",
			Username:        "admin",
			Password:        "admin123",
			RefreshEndpoint: "/auth/refresh",
			RefreshInterval: 100 * time.Millisecond,
		},
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)
	defer client.auth.Stop()

	// Wait for auto-refresh to trigger
	time.Sleep(250 * time.Millisecond)

	// Stop to prevent further refreshes
	client.auth.Stop()

	// Verify refresh was called at least once
	assert.GreaterOrEqual(t, callCount, 2, "Expected at least login + refresh calls")
}

// TestAuthNoneType tests the none authentication type.
func TestAuthNoneType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no auth header is set
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	authCfg := &config.AuthConfig{
		Type: "none",
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)

	resp, err := client.Get(context.Background(), "/test", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestAuthUnsupportedType tests unsupported authentication type.
func TestAuthUnsupportedType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	authCfg := &config.AuthConfig{
		Type: "unsupported",
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)

	// Unsupported type error happens during request, not client creation
	_, err = client.Get(context.Background(), "/test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported auth type")
}

// TestIsAuthenticatedForDifferentTypes tests IsAuthenticated for various auth types.
func TestIsAuthenticatedForDifferentTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	t.Run("bearer auth is authenticated", func(t *testing.T) {
		authCfg := &config.AuthConfig{
			Type: "bearer",
			Bearer: &config.BearerConfig{
				Token: "test-token",
			},
		}

		client, err := NewClient(targetCfg, authCfg, nil)
		require.NoError(t, err)
		assert.True(t, client.GetAuthManager().IsAuthenticated())
	})

	t.Run("api_key auth is authenticated", func(t *testing.T) {
		authCfg := &config.AuthConfig{
			Type: "api_key",
			APIKey: &config.APIKeyConfig{
				Key: "test-key",
			},
		}

		client, err := NewClient(targetCfg, authCfg, nil)
		require.NoError(t, err)
		assert.True(t, client.GetAuthManager().IsAuthenticated())
	})

	t.Run("basic auth is authenticated", func(t *testing.T) {
		authCfg := &config.AuthConfig{
			Type: "basic",
			Login: &config.LoginConfig{
				Username: "user",
				Password: "pass",
			},
		}

		client, err := NewClient(targetCfg, authCfg, nil)
		require.NoError(t, err)
		assert.True(t, client.GetAuthManager().IsAuthenticated())
	})
}
