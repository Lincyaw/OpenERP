package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
func TestERPLoginIntegration(t *testing.T) {
	// Skip if ERP is not running
	t.Skip("Skipping ERP integration test - requires running ERP system")

	targetCfg := config.TargetConfig{
		BaseURL: "http://localhost:8080",
		Timeout: 10 * time.Second,
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
	assert.NotEmpty(t, client.auth.GetAccessToken())

	// Test authenticated request to get current user
	resp, err := client.Get(context.Background(), "/auth/me", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	var result struct {
		Data struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}

	err = json.Unmarshal(resp.Body, &result)
	require.NoError(t, err)
	assert.Equal(t, "admin", result.Data.User.Username)
}