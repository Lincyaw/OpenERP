package ecommerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

// DouyinConfig holds configuration for Douyin (TikTok Shop) API integration
type DouyinConfig struct {
	// AppKey is the application key from Douyin open platform
	AppKey string
	// AppSecret is the application secret from Douyin open platform
	AppSecret string
	// AccessToken is the user's access token for API authorization
	AccessToken string
	// ShopID is the shop ID in Douyin platform
	ShopID string
	// APIBaseURL is the base URL for Douyin API (production or sandbox)
	APIBaseURL string
	// IsSandbox indicates if this is a sandbox environment
	IsSandbox bool
	// TimeoutSeconds is the HTTP request timeout
	TimeoutSeconds int
}

const (
	// DouyinProductionAPIURL is the production API endpoint
	DouyinProductionAPIURL = "https://openapi-fxg.jinritemai.com"
	// DouyinSandboxAPIURL is the sandbox API endpoint
	DouyinSandboxAPIURL = "https://openapi-sandbox.jinritemai.com"
)

// Errors for Douyin configuration
var (
	ErrDouyinConfigMissingAppKey      = errors.New("douyin: app key is required")
	ErrDouyinConfigMissingAppSecret   = errors.New("douyin: app secret is required")
	ErrDouyinConfigMissingAccessToken = errors.New("douyin: access token is required")
	ErrDouyinConfigMissingShopID      = errors.New("douyin: shop ID is required")
)

// NewDouyinConfig creates a new Douyin configuration with defaults
func NewDouyinConfig(appKey, appSecret, accessToken, shopID string) *DouyinConfig {
	return &DouyinConfig{
		AppKey:         appKey,
		AppSecret:      appSecret,
		AccessToken:    accessToken,
		ShopID:         shopID,
		APIBaseURL:     DouyinProductionAPIURL,
		IsSandbox:      false,
		TimeoutSeconds: 30,
	}
}

// NewSandboxDouyinConfig creates a new Douyin configuration for sandbox environment
func NewSandboxDouyinConfig(appKey, appSecret, accessToken, shopID string) *DouyinConfig {
	return &DouyinConfig{
		AppKey:         appKey,
		AppSecret:      appSecret,
		AccessToken:    accessToken,
		ShopID:         shopID,
		APIBaseURL:     DouyinSandboxAPIURL,
		IsSandbox:      true,
		TimeoutSeconds: 30,
	}
}

// Validate validates the Douyin configuration
func (c *DouyinConfig) Validate() error {
	if c.AppKey == "" {
		return ErrDouyinConfigMissingAppKey
	}
	if c.AppSecret == "" {
		return ErrDouyinConfigMissingAppSecret
	}
	if c.AccessToken == "" {
		return ErrDouyinConfigMissingAccessToken
	}
	if c.ShopID == "" {
		return ErrDouyinConfigMissingShopID
	}
	if c.APIBaseURL == "" {
		if c.IsSandbox {
			c.APIBaseURL = DouyinSandboxAPIURL
		} else {
			c.APIBaseURL = DouyinProductionAPIURL
		}
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 30
	}
	return nil
}

// Sign generates the signature for Douyin API request
// Douyin uses HMAC-SHA256 with format: method + param_json + timestamp + v + app_secret
// The signature process:
// 1. Sort all parameters (excluding sign)
// 2. Concatenate: app_secret + sorted_param_string + app_secret
// 3. Calculate HMAC-SHA256
func (c *DouyinConfig) Sign(method string, paramJSON string, timestamp string, v string) string {
	// Build sign string: app_secret + method + param_json + timestamp + v + app_secret
	var builder strings.Builder
	builder.WriteString(c.AppSecret)
	builder.WriteString(method)
	builder.WriteString(paramJSON)
	builder.WriteString(timestamp)
	builder.WriteString(v)
	builder.WriteString(c.AppSecret)

	// Calculate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(c.AppSecret))
	h.Write([]byte(builder.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// SignV2 generates signature using the new V2 signature method for Douyin API
// The V2 method sorts parameters and creates an HMAC-SHA256 hash
func (c *DouyinConfig) SignV2(params map[string]string) string {
	// Sort parameter keys
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sign string: app_secret + sorted_params + app_secret
	var builder strings.Builder
	builder.WriteString(c.AppSecret)
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(params[k])
	}
	builder.WriteString(c.AppSecret)

	// Calculate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(c.AppSecret))
	h.Write([]byte(builder.String()))
	return hex.EncodeToString(h.Sum(nil))
}
