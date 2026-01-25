package payment

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWechatPayConfigBuilder_Build(t *testing.T) {
	// Generate a test private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to PEM format
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	t.Run("complete config", func(t *testing.T) {
		config, err := NewWechatPayConfigBuilder().
			SetMchID("1234567890").
			SetAppID("wx1234567890abcdef").
			SetAPIKey("12345678901234567890123456789012").
			SetSerialNo("ABCDEF1234567890").
			SetPrivateKeyFromPEM(string(privateKeyPEM)).
			SetNotifyURL("https://example.com/notify").
			SetRefundNotifyURL("https://example.com/refund-notify").
			SetIsSandbox(true).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "1234567890", config.MchID)
		assert.Equal(t, "wx1234567890abcdef", config.AppID)
		assert.Equal(t, "12345678901234567890123456789012", config.APIKey)
		assert.Equal(t, "ABCDEF1234567890", config.SerialNo)
		assert.NotNil(t, config.PrivateKey)
		assert.Equal(t, "https://example.com/notify", config.NotifyURL)
		assert.Equal(t, "https://example.com/refund-notify", config.RefundNotifyURL)
		assert.True(t, config.IsSandbox)
	})

	t.Run("missing MchID", func(t *testing.T) {
		_, err := NewWechatPayConfigBuilder().
			SetAppID("wx1234567890abcdef").
			SetAPIKey("12345678901234567890123456789012").
			SetSerialNo("ABCDEF1234567890").
			SetPrivateKeyFromPEM(string(privateKeyPEM)).
			SetNotifyURL("https://example.com/notify").
			Build()

		assert.ErrorIs(t, err, ErrWechatMissingMchID)
	})

	t.Run("invalid private key PEM", func(t *testing.T) {
		_, err := NewWechatPayConfigBuilder().
			SetMchID("1234567890").
			SetAppID("wx1234567890abcdef").
			SetAPIKey("12345678901234567890123456789012").
			SetSerialNo("ABCDEF1234567890").
			SetPrivateKeyFromPEM("invalid-pem-data").
			SetNotifyURL("https://example.com/notify").
			Build()

		assert.ErrorIs(t, err, ErrWechatInvalidPrivateKey)
	})

	t.Run("error propagation", func(t *testing.T) {
		// Test that errors are propagated through the builder chain
		builder := NewWechatPayConfigBuilder()
		builder.SetPrivateKeyFromPEM("invalid")

		// Further operations should be no-ops
		builder.SetMchID("test")

		_, err := builder.Build()
		assert.ErrorIs(t, err, ErrWechatInvalidPrivateKey)
	})
}

func TestWechatPayConfigBuilder_SetPrivateKeyFromFile(t *testing.T) {
	// Generate a test private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to PEM format
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	t.Run("valid file", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		keyFile := filepath.Join(tmpDir, "private_key.pem")
		err := os.WriteFile(keyFile, privateKeyPEM, 0600)
		require.NoError(t, err)

		config, err := NewWechatPayConfigBuilder().
			SetMchID("1234567890").
			SetAppID("wx1234567890abcdef").
			SetAPIKey("12345678901234567890123456789012").
			SetSerialNo("ABCDEF1234567890").
			SetPrivateKeyFromFile(keyFile).
			SetNotifyURL("https://example.com/notify").
			Build()

		require.NoError(t, err)
		assert.NotNil(t, config.PrivateKey)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := NewWechatPayConfigBuilder().
			SetPrivateKeyFromFile("/non/existent/file.pem").
			Build()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read private key file")
	})
}

func TestWechatPayConfigBuilder_PKCS8PrivateKey(t *testing.T) {
	// Generate a test private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to PKCS8 format
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	config, err := NewWechatPayConfigBuilder().
		SetMchID("1234567890").
		SetAppID("wx1234567890abcdef").
		SetAPIKey("12345678901234567890123456789012").
		SetSerialNo("ABCDEF1234567890").
		SetPrivateKeyFromPEM(string(privateKeyPEM)).
		SetNotifyURL("https://example.com/notify").
		Build()

	require.NoError(t, err)
	assert.NotNil(t, config.PrivateKey)
}

func TestGenerateNonceStr(t *testing.T) {
	// Test that nonce strings are unique
	nonce1 := generateNonceStr()
	nonce2 := generateNonceStr()

	assert.NotEmpty(t, nonce1)
	assert.NotEmpty(t, nonce2)
	assert.NotEqual(t, nonce1, nonce2)
	assert.Len(t, nonce1, 32) // hex encoded 16 bytes = 32 chars
}
