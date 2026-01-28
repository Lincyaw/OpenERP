// Package client provides HTTP client functionality for the load generator.
// It includes authentication handling, retry logic, and response parsing.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
)

// Client is the main HTTP client for the load generator.
type Client struct {
	httpClient    *http.Client
	baseURL       string
	apiVersion    string
	headers       map[string]string
	auth          *AuthManager
	retryConfig   RetryConfig
	mu            sync.RWMutex
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries  int
	RetryDelay  time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	ShouldRetry func(resp *http.Response, err error) bool
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		ShouldRetry: func(resp *http.Response, err error) bool {
			if err != nil {
				return true
			}
			// Retry on 5xx errors and 429 (Too Many Requests)
			return resp.StatusCode >= 500 || resp.StatusCode == 429
		},
	}
}

// NewClient creates a new HTTP client for the load generator.
func NewClient(cfg config.TargetConfig, authCfg *config.AuthConfig, retryCfg *RetryConfig) (*Client, error) {
	// Validate configuration
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	if _, err := url.Parse(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	if retryCfg == nil {
		defaultCfg := DefaultRetryConfig()
		retryCfg = &defaultCfg
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	client := &Client{
		httpClient: httpClient,
		baseURL:    cfg.BaseURL,
		apiVersion: cfg.APIVersion,
		headers:    make(map[string]string),
		retryConfig: *retryCfg,
	}

	// Set default headers
	client.headers["Content-Type"] = "application/json"
	client.headers["Accept"] = "application/json"
	client.headers["User-Agent"] = "ERP-LoadGen/1.0"

	// Add custom headers from config
	for k, v := range cfg.Headers {
		client.headers[k] = v
	}

	// Initialize auth manager
	if authCfg != nil && authCfg.Type != "none" {
		authManager, err := NewAuthManager(client, authCfg)
		if err != nil {
			return nil, fmt.Errorf("creating auth manager: %w", err)
		}
		client.auth = authManager
	}

	return client, nil
}

// Request represents an HTTP request to be executed.
type Request struct {
	Method      string
	Path        string
	QueryParams map[string]string
	Headers     map[string]string
	Body        interface{}
	Timeout     time.Duration
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
	Error      error
}

// Do executes an HTTP request with retry logic.
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	// Build URL
	u, err := c.buildURL(req.Path, req.QueryParams)
	if err != nil {
		return nil, fmt.Errorf("building URL: %w", err)
	}

	// Marshal body if present
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Execute with retries
	var lastResp *Response
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			delay := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), bodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating HTTP request: %w", err)
		}

		// Set headers
		c.setHeaders(httpReq, req.Headers)

		// Add authentication if available
		if c.auth != nil {
			if err := c.auth.Authenticate(httpReq); err != nil {
				return nil, fmt.Errorf("authenticating request: %w", err)
			}
		}

		// Execute request
		start := time.Now()
		httpResp, err := c.httpClient.Do(httpReq)
		duration := time.Since(start)

		// Build response
		resp := &Response{
			Duration: duration,
			Error:    err,
		}

		if httpResp != nil {
			resp.StatusCode = httpResp.StatusCode
			resp.Headers = httpResp.Header
		}

		// Read body if no error
		if err == nil && httpResp.Body != nil {
			resp.Body, err = io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			if err != nil {
				resp.Error = fmt.Errorf("reading response body: %w", err)
			}
		}

		lastResp = resp
		lastErr = resp.Error

		// Check if we should retry
		if attempt < c.retryConfig.MaxRetries && c.retryConfig.ShouldRetry(httpResp, err) {
			continue
		}

		// Return response and error (if any)
		if err != nil {
			return resp, err
		}
		return resp, nil
	}

	return lastResp, lastErr
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, queryParams map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:      http.MethodGet,
		Path:        path,
		QueryParams: queryParams,
	})
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, Request{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	})
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, Request{
		Method: http.MethodDelete,
		Path:   path,
	})
}

// buildURL builds a complete URL from path and query parameters.
func (c *Client) buildURL(path string, queryParams map[string]string) (*url.URL, error) {
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Add API version if configured
	if c.apiVersion != "" && !strings.HasPrefix(path, "/"+c.apiVersion) {
		path = "/" + c.apiVersion + path
	}

	// Parse base URL
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}

	// Resolve path
	u, err := baseURL.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Add query parameters
	if len(queryParams) > 0 {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	return u, nil
}

// setHeaders sets headers on the request.
func (c *Client) setHeaders(req *http.Request, customHeaders map[string]string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Set default headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Set custom headers
	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}
}

// calculateBackoff calculates the backoff delay for the given attempt.
func (c *Client) calculateBackoff(attempt int) time.Duration {
	delay := float64(c.retryConfig.RetryDelay) * math.Pow(c.retryConfig.Multiplier, float64(attempt-1))
	if delay > float64(c.retryConfig.MaxDelay) {
		delay = float64(c.retryConfig.MaxDelay)
	}
	// Add jitter (±25%)
	jitter := delay * 0.25
	delay = delay + (rand.Float64()*2-1)*jitter
	return time.Duration(delay)
}

// SetHeader sets a default header for all requests.
func (c *Client) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// GetBaseURL returns the client's base URL.
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetAuthManager returns the authentication manager.
func (c *Client) GetAuthManager() *AuthManager {
	return c.auth
}

// Close closes the HTTP client.
func (c *Client) Close() error {
	// Nothing to close for standard http.Client
	return nil
}