package payment

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// WechatPayConfig contains configuration for WeChat Pay API v3
type WechatPayConfig struct {
	// MchID is the merchant ID
	MchID string
	// AppID is the WeChat application ID
	AppID string
	// APIKey is the API v3 key (32 bytes)
	APIKey string
	// SerialNo is the certificate serial number
	SerialNo string
	// PrivateKey is the merchant private key for signing
	PrivateKey *rsa.PrivateKey
	// WechatCert is the WeChat platform certificate for verification
	WechatCert *x509.Certificate
	// WechatCertSerialNo is the WeChat platform certificate serial number
	WechatCertSerialNo string
	// IsSandbox indicates whether to use sandbox environment
	IsSandbox bool
	// NotifyURL is the default callback URL for payment notifications
	NotifyURL string
	// RefundNotifyURL is the default callback URL for refund notifications
	RefundNotifyURL string
}

// Errors for configuration validation
var (
	ErrWechatMissingMchID      = errors.New("wechat: missing merchant ID")
	ErrWechatMissingAppID      = errors.New("wechat: missing app ID")
	ErrWechatMissingAPIKey     = errors.New("wechat: missing API key")
	ErrWechatInvalidAPIKey     = errors.New("wechat: API key must be 32 bytes")
	ErrWechatMissingSerialNo   = errors.New("wechat: missing certificate serial number")
	ErrWechatMissingPrivateKey = errors.New("wechat: missing private key")
	ErrWechatInvalidPrivateKey = errors.New("wechat: invalid private key format")
	ErrWechatMissingWechatCert = errors.New("wechat: missing WeChat platform certificate")
	ErrWechatInvalidWechatCert = errors.New("wechat: invalid WeChat platform certificate")
	ErrWechatMissingNotifyURL  = errors.New("wechat: missing notify URL")
)

// Validate validates the configuration
func (c *WechatPayConfig) Validate() error {
	if c.MchID == "" {
		return ErrWechatMissingMchID
	}
	if c.AppID == "" {
		return ErrWechatMissingAppID
	}
	if c.APIKey == "" {
		return ErrWechatMissingAPIKey
	}
	if len(c.APIKey) != 32 {
		return ErrWechatInvalidAPIKey
	}
	if c.SerialNo == "" {
		return ErrWechatMissingSerialNo
	}
	if c.PrivateKey == nil {
		return ErrWechatMissingPrivateKey
	}
	if c.NotifyURL == "" {
		return ErrWechatMissingNotifyURL
	}
	return nil
}

// WechatPayConfigBuilder helps build WechatPayConfig
type WechatPayConfigBuilder struct {
	config WechatPayConfig
	err    error
}

// NewWechatPayConfigBuilder creates a new config builder
func NewWechatPayConfigBuilder() *WechatPayConfigBuilder {
	return &WechatPayConfigBuilder{}
}

// SetMchID sets the merchant ID
func (b *WechatPayConfigBuilder) SetMchID(mchID string) *WechatPayConfigBuilder {
	b.config.MchID = mchID
	return b
}

// SetAppID sets the app ID
func (b *WechatPayConfigBuilder) SetAppID(appID string) *WechatPayConfigBuilder {
	b.config.AppID = appID
	return b
}

// SetAPIKey sets the API v3 key
func (b *WechatPayConfigBuilder) SetAPIKey(apiKey string) *WechatPayConfigBuilder {
	b.config.APIKey = apiKey
	return b
}

// SetSerialNo sets the certificate serial number
func (b *WechatPayConfigBuilder) SetSerialNo(serialNo string) *WechatPayConfigBuilder {
	b.config.SerialNo = serialNo
	return b
}

// SetPrivateKeyFromPEM sets the private key from PEM string
func (b *WechatPayConfigBuilder) SetPrivateKeyFromPEM(pemStr string) *WechatPayConfigBuilder {
	if b.err != nil {
		return b
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		b.err = ErrWechatInvalidPrivateKey
		return b
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1 format
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			b.err = fmt.Errorf("%w: %v", ErrWechatInvalidPrivateKey, err)
			return b
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		b.err = ErrWechatInvalidPrivateKey
		return b
	}

	b.config.PrivateKey = rsaKey
	return b
}

// SetPrivateKeyFromFile sets the private key from a file
func (b *WechatPayConfigBuilder) SetPrivateKeyFromFile(path string) *WechatPayConfigBuilder {
	if b.err != nil {
		return b
	}

	data, err := os.ReadFile(path)
	if err != nil {
		b.err = fmt.Errorf("wechat: failed to read private key file: %w", err)
		return b
	}

	return b.SetPrivateKeyFromPEM(string(data))
}

// SetWechatCertFromPEM sets the WeChat platform certificate from PEM string
func (b *WechatPayConfigBuilder) SetWechatCertFromPEM(pemStr string, serialNo string) *WechatPayConfigBuilder {
	if b.err != nil {
		return b
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		b.err = ErrWechatInvalidWechatCert
		return b
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		b.err = fmt.Errorf("%w: %v", ErrWechatInvalidWechatCert, err)
		return b
	}

	b.config.WechatCert = cert
	b.config.WechatCertSerialNo = serialNo
	return b
}

// SetWechatCertFromFile sets the WeChat platform certificate from a file
func (b *WechatPayConfigBuilder) SetWechatCertFromFile(path string, serialNo string) *WechatPayConfigBuilder {
	if b.err != nil {
		return b
	}

	data, err := os.ReadFile(path)
	if err != nil {
		b.err = fmt.Errorf("wechat: failed to read certificate file: %w", err)
		return b
	}

	return b.SetWechatCertFromPEM(string(data), serialNo)
}

// SetIsSandbox sets whether to use sandbox environment
func (b *WechatPayConfigBuilder) SetIsSandbox(isSandbox bool) *WechatPayConfigBuilder {
	b.config.IsSandbox = isSandbox
	return b
}

// SetNotifyURL sets the default notify URL
func (b *WechatPayConfigBuilder) SetNotifyURL(url string) *WechatPayConfigBuilder {
	b.config.NotifyURL = url
	return b
}

// SetRefundNotifyURL sets the refund notify URL
func (b *WechatPayConfigBuilder) SetRefundNotifyURL(url string) *WechatPayConfigBuilder {
	b.config.RefundNotifyURL = url
	return b
}

// Build builds the config and validates it
func (b *WechatPayConfigBuilder) Build() (*WechatPayConfig, error) {
	if b.err != nil {
		return nil, b.err
	}

	if err := b.config.Validate(); err != nil {
		return nil, err
	}

	return &b.config, nil
}
