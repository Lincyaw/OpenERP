package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/example/erp/tools/loadgen/internal/config"
)

const (
	// tokenRefreshBuffer is the time before token expiry to trigger refresh
	tokenRefreshBuffer = 30 * time.Second
)

// AuthManager handles authentication for the HTTP client.
type AuthManager struct {
	client            *Client
	config            *config.AuthConfig
	mu                sync.RWMutex
	accessToken       string
	refreshTokenValue string
	tokenExpiry       time.Time
	refreshTicker     *time.Ticker
	stopRefresh       chan struct{}
}

// LoginRequest represents the login request payload.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response payload.
type LoginResponse struct {
	Data struct {
		Token TokenResponse `json:"token"`
	} `json:"data"`
}

// TokenResponse represents the token data.
type TokenResponse struct {
	AccessToken          string    `json:"access_token"`
	RefreshToken         string    `json:"refresh_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
	TokenType            string    `json:"token_type"`
}

// RefreshTokenRequest represents the refresh token request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// NewAuthManager creates a new authentication manager.
func NewAuthManager(client *Client, authConfig *config.AuthConfig) (*AuthManager, error) {
	if authConfig == nil {
		return nil, fmt.Errorf("auth config is required")
	}

	am := &AuthManager{
		client:      client,
		config:      authConfig,
		stopRefresh: make(chan struct{}),
	}

	// Perform initial login if using login-based auth
	if authConfig.Type == "login" && authConfig.Login != nil {
		if err := am.login(); err != nil {
			return nil, fmt.Errorf("initial login failed: %w", err)
		}

		// Start token refresh if configured
		if authConfig.Login.RefreshEndpoint != "" && authConfig.Login.RefreshInterval > 0 {
			am.startTokenRefresh()
		}
	}

	return am, nil
}

// Authenticate adds authentication to the request.
func (am *AuthManager) Authenticate(req *http.Request) error {
	am.mu.RLock()
	defer am.mu.RUnlock()

	switch am.config.Type {
	case "none":
		// No authentication
		return nil

	case "basic":
		// Basic authentication
		if am.config.Login == nil {
			return fmt.Errorf("basic auth requires login config")
		}
		req.SetBasicAuth(am.config.Login.Username, am.config.Login.Password)
		return nil

	case "bearer":
		// Static bearer token
		if am.config.Bearer == nil {
			return fmt.Errorf("bearer auth requires bearer config")
		}
		req.Header.Set("Authorization", "Bearer "+am.config.Bearer.Token)
		return nil

	case "api_key":
		// API key authentication
		if am.config.APIKey == nil {
			return fmt.Errorf("api key auth requires api key config")
		}
		header := am.config.APIKey.Header
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, am.config.APIKey.Key)
		return nil

	case "login":
		// Login-based authentication with access token
		if am.accessToken == "" {
			return fmt.Errorf("no access token available")
		}

		// Check if token is expired
		if time.Now().After(am.tokenExpiry.Add(-tokenRefreshBuffer)) {
			// Token is about to expire, refresh it
			am.mu.RUnlock()
			am.mu.Lock()
			if time.Now().After(am.tokenExpiry.Add(-tokenRefreshBuffer)) {
				// Still expired after acquiring write lock
				if err := am.refreshToken(); err != nil {
					// Try to login again if refresh fails
					if err := am.login(); err != nil {
						am.mu.Unlock()
						am.mu.RLock()
						return fmt.Errorf("token refresh and login failed: %w", err)
					}
				}
			}
			am.mu.Unlock()
			am.mu.RLock()
		}

		req.Header.Set("Authorization", "Bearer "+am.accessToken)
		return nil

	default:
		return fmt.Errorf("unsupported auth type: %s", am.config.Type)
	}
}

// login performs the initial login.
func (am *AuthManager) login() error {
	if am.config.Login == nil {
		return fmt.Errorf("login config is required")
	}

	loginReq := LoginRequest{
		Username: am.config.Login.Username,
		Password: am.config.Login.Password,
	}

	method := am.config.Login.Method
	if method == "" {
		method = "POST"
	}

	resp, err := am.client.Do(context.Background(), Request{
		Method: method,
		Path:   am.config.Login.Endpoint,
		Body:   loginReq,
	})
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Parse response
	var loginResp LoginResponse
	if err := json.Unmarshal(resp.Body, &loginResp); err != nil {
		return fmt.Errorf("parsing login response: %w", err)
	}

	// Extract token using JSONPath if configured
	tokenPath := am.config.Login.TokenPath
	if tokenPath == "" {
		tokenPath = "$.data.access_token"
	}

	// For now, use the direct struct path
	am.accessToken = loginResp.Data.Token.AccessToken
	am.refreshTokenValue = loginResp.Data.Token.RefreshToken
	am.tokenExpiry = loginResp.Data.Token.AccessTokenExpiresAt

	return nil
}

// refreshToken refreshes the access token.
func (am *AuthManager) refreshToken() error {
	if am.config.Login == nil || am.config.Login.RefreshEndpoint == "" {
		return fmt.Errorf("refresh endpoint not configured")
	}

	if am.refreshTokenValue == "" {
		return fmt.Errorf("no refresh token available")
	}

	refreshReq := RefreshTokenRequest{
		RefreshToken: am.refreshTokenValue,
	}

	resp, err := am.client.Do(context.Background(), Request{
		Method: "POST",
		Path:   am.config.Login.RefreshEndpoint,
		Body:   refreshReq,
	})
	if err != nil {
		return fmt.Errorf("refresh token request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh token failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Parse response
	var refreshResp LoginResponse
	if err := json.Unmarshal(resp.Body, &refreshResp); err != nil {
		return fmt.Errorf("parsing refresh token response: %w", err)
	}

	// Update tokens
	am.accessToken = refreshResp.Data.Token.AccessToken
	if refreshResp.Data.Token.RefreshToken != "" {
		am.refreshTokenValue = refreshResp.Data.Token.RefreshToken
	}
	am.tokenExpiry = refreshResp.Data.Token.AccessTokenExpiresAt

	return nil
}

// startTokenRefresh starts the token refresh ticker.
func (am *AuthManager) startTokenRefresh() {
	if am.refreshTicker != nil {
		return
	}

	am.refreshTicker = time.NewTicker(am.config.Login.RefreshInterval)
	go func() {
		for {
			select {
			case <-am.refreshTicker.C:
				if err := am.refreshToken(); err != nil {
					// Log error but continue trying
					fmt.Printf("Token refresh failed: %v\n", err)
				}
			case <-am.stopRefresh:
				return
			}
		}
	}()
}

// Stop stops the authentication manager and token refresh.
func (am *AuthManager) Stop() {
	if am.refreshTicker != nil {
		am.refreshTicker.Stop()
		close(am.stopRefresh)
	}
}

// GetAccessToken returns the current access token.
func (am *AuthManager) GetAccessToken() string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.accessToken
}

// IsAuthenticated returns true if authenticated.
func (am *AuthManager) IsAuthenticated() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	switch am.config.Type {
	case "none":
		return true
	case "login":
		return am.accessToken != "" && time.Now().Before(am.tokenExpiry)
	case "basic", "bearer", "api_key":
		return true
	default:
		return false
	}
}