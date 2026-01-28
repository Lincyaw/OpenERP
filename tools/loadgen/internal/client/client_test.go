package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientCreation tests basic client creation.
func TestClientCreation(t *testing.T) {
	targetCfg := config.TargetConfig{
		BaseURL:    "http://localhost:8080",
		APIVersion: "v1",
		Timeout:    30 * time.Second,
	}

	client, err := NewClient(targetCfg, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080", client.GetBaseURL())
}

// TestClientWithAuth tests client creation with authentication.
func TestClientWithAuth(t *testing.T) {
	targetCfg := config.TargetConfig{
		BaseURL: "http://localhost:8080",
	}

	authCfg := &config.AuthConfig{
		Type: "bearer",
		Bearer: &config.BearerConfig{
			Token: "test-token",
		},
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.GetAuthManager())
}

// TestBasicRequest tests basic HTTP request execution.
func TestBasicRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/test", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// Create client
	targetCfg := config.TargetConfig{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	}

	client, err := NewClient(targetCfg, nil, nil)
	require.NoError(t, err)

	// Execute request
	resp, err := client.Get(context.Background(), "/test", nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `{"message": "success"}`, string(resp.Body))
}

// TestRequestWithQueryParams tests request with query parameters.
func TestRequestWithQueryParams(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "value1", r.URL.Query().Get("param1"))
		assert.Equal(t, "value2", r.URL.Query().Get("param2"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create client
	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(targetCfg, nil, nil)
	require.NoError(t, err)

	// Execute request with query params
	queryParams := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}
	resp, err := client.Get(context.Background(), "/test", queryParams)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestPostRequest tests POST request with body.
func TestPostRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "test", body["name"])

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "123", "name": "test"}`))
	}))
	defer server.Close()

	// Create client
	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(targetCfg, nil, nil)
	require.NoError(t, err)

	// Execute POST request
	body := map[string]interface{}{
		"name": "test",
	}
	resp, err := client.Post(context.Background(), "/items", body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, `{"id": "123", "name": "test"}`, string(resp.Body))
}

// TestRetryLogic tests retry behavior on failure.
func TestRetryLogic(t *testing.T) {
	attempts := 0

	// Create test server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	// Create client with retry config
	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	retryCfg := RetryConfig{
		MaxRetries: 3,
		RetryDelay: 100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Multiplier: 2.0,
		ShouldRetry: func(resp *http.Response, err error) bool {
			return resp.StatusCode >= 500
		},
	}

	client, err := NewClient(targetCfg, nil, &retryCfg)
	require.NoError(t, err)

	// Execute request
	resp, err := client.Get(context.Background(), "/test", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, attempts)
}

// TestBearerAuth tests bearer token authentication.
func TestBearerAuth(t *testing.T) {
	// Create test server that checks auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "unauthorized"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	// Create client with bearer auth
	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	authCfg := &config.AuthConfig{
		Type: "bearer",
		Bearer: &config.BearerConfig{
			Token: "test-token",
		},
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)

	// Execute request
	resp, err := client.Get(context.Background(), "/protected", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `{"authenticated": true}`, string(resp.Body))
}

// TestAPIKeyAuth tests API key authentication.
func TestAPIKeyAuth(t *testing.T) {
	// Create test server that checks API key
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "secret-key" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid api key"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	// Create client with API key auth
	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	authCfg := &config.AuthConfig{
		Type: "api_key",
		APIKey: &config.APIKeyConfig{
			Key:    "secret-key",
			Header: "X-API-Key",
		},
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)

	// Execute request
	resp, err := client.Get(context.Background(), "/protected", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestBasicAuth tests basic authentication.
func TestBasicAuth(t *testing.T) {
	// Create test server that checks basic auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "admin" || password != "admin123" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid credentials"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	// Create client with basic auth
	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	authCfg := &config.AuthConfig{
		Type: "basic",
		Login: &config.LoginConfig{
			Username: "admin",
			Password: "admin123",
		},
	}

	client, err := NewClient(targetCfg, authCfg, nil)
	require.NoError(t, err)

	// Execute request
	resp, err := client.Get(context.Background(), "/protected", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestTimeout tests request timeout.
func TestTimeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with no retries
	retryCfg := RetryConfig{
		MaxRetries: 0,
		ShouldRetry: func(resp *http.Response, err error) bool {
			return false // Never retry
		},
	}

	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 1 * time.Second,
	}

	client, err := NewClient(targetCfg, nil, &retryCfg)
	require.NoError(t, err)

	// Execute request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resp, err := client.Get(ctx, "/slow", nil)
	assert.Error(t, err)
	assert.NotNil(t, resp)
}

// TestConcurrentRequests tests concurrent request execution.
func TestConcurrentRequests(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"request": "` + r.URL.Path + `"}`))
	}))
	defer server.Close()

	// Create client
	targetCfg := config.TargetConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(targetCfg, nil, nil)
	require.NoError(t, err)

	// Execute concurrent requests
	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			path := fmt.Sprintf("/request-%d", id)
			resp, err := client.Get(context.Background(), path, nil)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, fmt.Sprintf(`{"request": "%s"}`, path), string(resp.Body))
			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < numRequests; i++ {
		<-done
	}
}