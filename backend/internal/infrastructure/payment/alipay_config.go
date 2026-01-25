package payment

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// AlipayConfig contains configuration for Alipay Open Platform API
type AlipayConfig struct {
	// AppID is the Alipay application ID
	AppID string
	// PrivateKey is the application private key for signing requests
	PrivateKey *rsa.PrivateKey
	// AlipayPublicKey is Alipay's public key for verifying responses/callbacks
	AlipayPublicKey *rsa.PublicKey
	// AlipayRootCert is the Alipay root certificate (optional, for cert mode)
	AlipayRootCert *x509.Certificate
	// AlipayCertSN is the Alipay certificate serial number (for cert mode)
	AlipayCertSN string
	// AppCertSN is the application certificate serial number (for cert mode)
	AppCertSN string
	// IsSandbox indicates whether to use sandbox environment
	IsSandbox bool
	// SignType is the signature algorithm (RSA2 recommended)
	SignType string
	// NotifyURL is the default callback URL for payment notifications
	NotifyURL string
	// ReturnURL is the default return URL after payment
	ReturnURL string
}

// Errors for configuration validation
var (
	ErrAlipayMissingAppID      = errors.New("alipay: missing app ID")
	ErrAlipayMissingPrivateKey = errors.New("alipay: missing private key")
	ErrAlipayInvalidPrivateKey = errors.New("alipay: invalid private key format")
	ErrAlipayMissingPublicKey  = errors.New("alipay: missing Alipay public key")
	ErrAlipayInvalidPublicKey  = errors.New("alipay: invalid Alipay public key format")
	ErrAlipayMissingNotifyURL  = errors.New("alipay: missing notify URL")
	ErrAlipayInvalidSignType   = errors.New("alipay: invalid sign type, must be RSA2 or RSA")
)

// Validate validates the configuration
func (c *AlipayConfig) Validate() error {
	if c.AppID == "" {
		return ErrAlipayMissingAppID
	}
	if c.PrivateKey == nil {
		return ErrAlipayMissingPrivateKey
	}
	if c.AlipayPublicKey == nil {
		return ErrAlipayMissingPublicKey
	}
	if c.SignType == "" {
		c.SignType = "RSA2" // Default to RSA2
	}
	if c.SignType != "RSA2" && c.SignType != "RSA" {
		return ErrAlipayInvalidSignType
	}
	if c.NotifyURL == "" {
		return ErrAlipayMissingNotifyURL
	}
	return nil
}

// AlipayConfigBuilder helps build AlipayConfig
type AlipayConfigBuilder struct {
	config AlipayConfig
	err    error
}

// NewAlipayConfigBuilder creates a new config builder
func NewAlipayConfigBuilder() *AlipayConfigBuilder {
	return &AlipayConfigBuilder{
		config: AlipayConfig{
			SignType: "RSA2", // Default to RSA2
		},
	}
}

// SetAppID sets the app ID
func (b *AlipayConfigBuilder) SetAppID(appID string) *AlipayConfigBuilder {
	b.config.AppID = appID
	return b
}

// SetPrivateKeyFromPEM sets the private key from PEM string
func (b *AlipayConfigBuilder) SetPrivateKeyFromPEM(pemStr string) *AlipayConfigBuilder {
	if b.err != nil {
		return b
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		b.err = ErrAlipayInvalidPrivateKey
		return b
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1 format
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			b.err = fmt.Errorf("%w: %v", ErrAlipayInvalidPrivateKey, err)
			return b
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		b.err = ErrAlipayInvalidPrivateKey
		return b
	}

	b.config.PrivateKey = rsaKey
	return b
}

// SetPrivateKeyFromFile sets the private key from a file
func (b *AlipayConfigBuilder) SetPrivateKeyFromFile(path string) *AlipayConfigBuilder {
	if b.err != nil {
		return b
	}

	data, err := os.ReadFile(path)
	if err != nil {
		b.err = fmt.Errorf("alipay: failed to read private key file: %w", err)
		return b
	}

	return b.SetPrivateKeyFromPEM(string(data))
}

// SetAlipayPublicKeyFromPEM sets the Alipay public key from PEM string
func (b *AlipayConfigBuilder) SetAlipayPublicKeyFromPEM(pemStr string) *AlipayConfigBuilder {
	if b.err != nil {
		return b
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		b.err = ErrAlipayInvalidPublicKey
		return b
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		b.err = fmt.Errorf("%w: %v", ErrAlipayInvalidPublicKey, err)
		return b
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		b.err = ErrAlipayInvalidPublicKey
		return b
	}

	b.config.AlipayPublicKey = rsaKey
	return b
}

// SetAlipayPublicKeyFromFile sets the Alipay public key from a file
func (b *AlipayConfigBuilder) SetAlipayPublicKeyFromFile(path string) *AlipayConfigBuilder {
	if b.err != nil {
		return b
	}

	data, err := os.ReadFile(path)
	if err != nil {
		b.err = fmt.Errorf("alipay: failed to read public key file: %w", err)
		return b
	}

	return b.SetAlipayPublicKeyFromPEM(string(data))
}

// SetIsSandbox sets whether to use sandbox environment
func (b *AlipayConfigBuilder) SetIsSandbox(isSandbox bool) *AlipayConfigBuilder {
	b.config.IsSandbox = isSandbox
	return b
}

// SetSignType sets the signature type (RSA2 or RSA)
func (b *AlipayConfigBuilder) SetSignType(signType string) *AlipayConfigBuilder {
	b.config.SignType = signType
	return b
}

// SetNotifyURL sets the default notify URL
func (b *AlipayConfigBuilder) SetNotifyURL(url string) *AlipayConfigBuilder {
	b.config.NotifyURL = url
	return b
}

// SetReturnURL sets the default return URL
func (b *AlipayConfigBuilder) SetReturnURL(url string) *AlipayConfigBuilder {
	b.config.ReturnURL = url
	return b
}

// SetAlipayRootCertFromPEM sets the Alipay root certificate from PEM string
func (b *AlipayConfigBuilder) SetAlipayRootCertFromPEM(pemStr string, certSN string) *AlipayConfigBuilder {
	if b.err != nil {
		return b
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		b.err = errors.New("alipay: invalid root certificate format")
		return b
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		b.err = fmt.Errorf("alipay: failed to parse root certificate: %w", err)
		return b
	}

	b.config.AlipayRootCert = cert
	b.config.AlipayCertSN = certSN
	return b
}

// SetAppCertSN sets the application certificate serial number
func (b *AlipayConfigBuilder) SetAppCertSN(certSN string) *AlipayConfigBuilder {
	b.config.AppCertSN = certSN
	return b
}

// Build builds the config and validates it
func (b *AlipayConfigBuilder) Build() (*AlipayConfig, error) {
	if b.err != nil {
		return nil, b.err
	}

	if err := b.config.Validate(); err != nil {
		return nil, err
	}

	return &b.config, nil
}
