package ecommerce

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

// TaobaoConfig holds configuration for Taobao/Tmall API integration
type TaobaoConfig struct {
	// AppKey is the application key from Taobao open platform
	AppKey string
	// AppSecret is the application secret from Taobao open platform
	AppSecret string
	// SessionKey is the user's access token (session key)
	SessionKey string
	// APIBaseURL is the base URL for Taobao API (production or sandbox)
	APIBaseURL string
	// IsSandbox indicates if this is a sandbox environment
	IsSandbox bool
	// TimeoutSeconds is the HTTP request timeout
	TimeoutSeconds int
}

const (
	// TaobaoProductionAPIURL is the production API endpoint
	TaobaoProductionAPIURL = "https://gw.api.taobao.com/router/rest"
	// TaobaoSandboxAPIURL is the sandbox API endpoint
	TaobaoSandboxAPIURL = "https://gw.api.tbsandbox.com/router/rest"
)

// Errors for Taobao configuration
var (
	ErrTaobaoConfigMissingAppKey     = errors.New("taobao: app key is required")
	ErrTaobaoConfigMissingAppSecret  = errors.New("taobao: app secret is required")
	ErrTaobaoConfigMissingSessionKey = errors.New("taobao: session key is required")
)

// NewTaobaoConfig creates a new Taobao configuration with defaults
func NewTaobaoConfig(appKey, appSecret, sessionKey string) *TaobaoConfig {
	return &TaobaoConfig{
		AppKey:         appKey,
		AppSecret:      appSecret,
		SessionKey:     sessionKey,
		APIBaseURL:     TaobaoProductionAPIURL,
		IsSandbox:      false,
		TimeoutSeconds: 30,
	}
}

// NewSandboxTaobaoConfig creates a new Taobao configuration for sandbox environment
func NewSandboxTaobaoConfig(appKey, appSecret, sessionKey string) *TaobaoConfig {
	return &TaobaoConfig{
		AppKey:         appKey,
		AppSecret:      appSecret,
		SessionKey:     sessionKey,
		APIBaseURL:     TaobaoSandboxAPIURL,
		IsSandbox:      true,
		TimeoutSeconds: 30,
	}
}

// Validate validates the Taobao configuration
func (c *TaobaoConfig) Validate() error {
	if c.AppKey == "" {
		return ErrTaobaoConfigMissingAppKey
	}
	if c.AppSecret == "" {
		return ErrTaobaoConfigMissingAppSecret
	}
	if c.SessionKey == "" {
		return ErrTaobaoConfigMissingSessionKey
	}
	if c.APIBaseURL == "" {
		if c.IsSandbox {
			c.APIBaseURL = TaobaoSandboxAPIURL
		} else {
			c.APIBaseURL = TaobaoProductionAPIURL
		}
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 30
	}
	return nil
}

// Sign generates the signature for Taobao API request.
// NOTE: This uses MD5 because it is required by Taobao's legacy API specification.
// MD5 is cryptographically weak but necessary for API compatibility.
// Taobao uses MD5(secret + sorted_params + secret) for signature.
func (c *TaobaoConfig) Sign(params map[string]string) string {
	// Sort parameter keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sign string: secret + key1value1key2value2... + secret
	var builder strings.Builder
	builder.WriteString(c.AppSecret)
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(params[k])
	}
	builder.WriteString(c.AppSecret)

	// Calculate MD5 hash
	hash := md5.Sum([]byte(builder.String()))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// SignHMAC generates HMAC-MD5 signature for Taobao API request (alternative method)
func (c *TaobaoConfig) SignHMAC(params map[string]string) string {
	// Sort parameter keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sign string: key1value1key2value2...
	var builder strings.Builder
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(params[k])
	}

	// Calculate HMAC-MD5
	h := hmac.New(md5.New, []byte(c.AppSecret))
	h.Write([]byte(builder.String()))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}
