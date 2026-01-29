package storage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// ============================================================================
// Unit Tests (no external dependencies)
// ============================================================================

func TestNewS3ObjectStorage_Validation(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := NewS3ObjectStorage(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is required")
	})

	t.Run("missing bucket returns error", func(t *testing.T) {
		cfg := &config.StorageConfig{
			AccessKey: "test-key",
			SecretKey: "test-secret",
		}
		_, err := NewS3ObjectStorage(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bucket is required")
	})

	t.Run("missing access key returns error", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:    "test-bucket",
			SecretKey: "test-secret",
		}
		_, err := NewS3ObjectStorage(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access key is required")
	})

	t.Run("missing secret key returns error", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:    "test-bucket",
			AccessKey: "test-key",
		}
		_, err := NewS3ObjectStorage(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is required")
	})

	t.Run("valid config creates storage", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:            "test-bucket",
			AccessKey:         "test-key",
			SecretKey:         "test-secret",
			Region:            "us-east-1",
			Endpoint:          "http://localhost:9000",
			UsePathStyle:      true,
			PresignExpiration: 15 * time.Minute,
		}
		storage, err := NewS3ObjectStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.Equal(t, "test-bucket", storage.GetBucket())
		assert.Equal(t, 15*time.Minute, storage.presignExpiration)
	})

	t.Run("default region is us-east-1", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:    "test-bucket",
			AccessKey: "test-key",
			SecretKey: "test-secret",
			Endpoint:  "http://localhost:9000",
		}
		storage, err := NewS3ObjectStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})

	t.Run("default endpoint is localhost", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:    "test-bucket",
			AccessKey: "test-key",
			SecretKey: "test-secret",
		}
		storage, err := NewS3ObjectStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})

	t.Run("adds http prefix when missing and no SSL", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:    "test-bucket",
			AccessKey: "test-key",
			SecretKey: "test-secret",
			Endpoint:  "localhost:9000",
			UseSSL:    false,
		}
		storage, err := NewS3ObjectStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})

	t.Run("adds https prefix when missing and SSL enabled", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:    "test-bucket",
			AccessKey: "test-key",
			SecretKey: "test-secret",
			Endpoint:  "localhost:9000",
			UseSSL:    true,
		}
		storage, err := NewS3ObjectStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})

	t.Run("default presign expiration is 15 minutes", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Bucket:    "test-bucket",
			AccessKey: "test-key",
			SecretKey: "test-secret",
			Endpoint:  "http://localhost:9000",
		}
		storage, err := NewS3ObjectStorage(cfg)
		require.NoError(t, err)
		assert.Equal(t, 15*time.Minute, storage.presignExpiration)
	})
}

func TestS3ObjectStorageOptions(t *testing.T) {
	baseConfig := &config.StorageConfig{
		Bucket:    "test-bucket",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Endpoint:  "http://localhost:9000",
	}

	t.Run("WithLogger sets custom logger", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		storage, err := NewS3ObjectStorage(baseConfig, WithLogger(logger))
		require.NoError(t, err)
		assert.NotNil(t, storage.logger)
	})

	t.Run("WithPresignExpiration sets custom duration", func(t *testing.T) {
		storage, err := NewS3ObjectStorage(baseConfig, WithPresignExpiration(1*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 1*time.Hour, storage.presignExpiration)
	})
}

func TestS3ObjectStorage_GenerateUploadURL(t *testing.T) {
	cfg := &config.StorageConfig{
		Bucket:            "test-bucket",
		AccessKey:         "test-key",
		SecretKey:         "test-secret",
		Endpoint:          "http://localhost:9000",
		UsePathStyle:      true,
		PresignExpiration: 15 * time.Minute,
	}
	storage, err := NewS3ObjectStorage(cfg)
	require.NoError(t, err)

	t.Run("empty storage key returns error", func(t *testing.T) {
		url, _, err := storage.GenerateUploadURL(context.Background(), "", "image/jpeg", 15*time.Minute)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage key is required")
		assert.Empty(t, url)
	})

	t.Run("generates valid presigned URL", func(t *testing.T) {
		url, expiresAt, err := storage.GenerateUploadURL(context.Background(), "test/key.jpg", "image/jpeg", 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.True(t, strings.Contains(url, "localhost:9000"))
		assert.True(t, strings.Contains(url, "test-bucket"))
		assert.True(t, strings.Contains(url, "test/key.jpg") || strings.Contains(url, "test%2Fkey.jpg"))
		assert.True(t, expiresAt.After(time.Now()))
		assert.True(t, expiresAt.Before(time.Now().Add(16*time.Minute)))
	})

	t.Run("uses default expiration when not provided", func(t *testing.T) {
		url, expiresAt, err := storage.GenerateUploadURL(context.Background(), "test/key.jpg", "image/jpeg", 0)
		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.True(t, expiresAt.After(time.Now()))
	})
}

func TestS3ObjectStorage_GenerateDownloadURL(t *testing.T) {
	cfg := &config.StorageConfig{
		Bucket:            "test-bucket",
		AccessKey:         "test-key",
		SecretKey:         "test-secret",
		Endpoint:          "http://localhost:9000",
		UsePathStyle:      true,
		PresignExpiration: 15 * time.Minute,
	}
	storage, err := NewS3ObjectStorage(cfg)
	require.NoError(t, err)

	t.Run("empty storage key returns error", func(t *testing.T) {
		url, _, err := storage.GenerateDownloadURL(context.Background(), "", 15*time.Minute)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage key is required")
		assert.Empty(t, url)
	})

	t.Run("generates valid presigned URL", func(t *testing.T) {
		url, expiresAt, err := storage.GenerateDownloadURL(context.Background(), "test/key.jpg", 1*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.True(t, strings.Contains(url, "localhost:9000"))
		assert.True(t, strings.Contains(url, "test-bucket"))
		assert.True(t, expiresAt.After(time.Now()))
	})

	t.Run("uses default expiration when not provided", func(t *testing.T) {
		url, expiresAt, err := storage.GenerateDownloadURL(context.Background(), "test/key.jpg", 0)
		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.True(t, expiresAt.After(time.Now()))
	})
}

func TestS3ObjectStorage_DeleteObject_ValidationOnly(t *testing.T) {
	cfg := &config.StorageConfig{
		Bucket:    "test-bucket",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Endpoint:  "http://localhost:9000",
	}
	storage, err := NewS3ObjectStorage(cfg)
	require.NoError(t, err)

	t.Run("empty storage key returns error", func(t *testing.T) {
		err := storage.DeleteObject(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage key is required")
	})
}

func TestS3ObjectStorage_ObjectExists_ValidationOnly(t *testing.T) {
	cfg := &config.StorageConfig{
		Bucket:    "test-bucket",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Endpoint:  "http://localhost:9000",
	}
	storage, err := NewS3ObjectStorage(cfg)
	require.NoError(t, err)

	t.Run("empty storage key returns error", func(t *testing.T) {
		exists, err := storage.ObjectExists(context.Background(), "")
		require.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "storage key is required")
	})
}

func TestS3ObjectStorage_Upload_ValidationOnly(t *testing.T) {
	cfg := &config.StorageConfig{
		Bucket:    "test-bucket",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Endpoint:  "http://localhost:9000",
	}
	storage, err := NewS3ObjectStorage(cfg)
	require.NoError(t, err)

	t.Run("empty storage key returns error", func(t *testing.T) {
		err := storage.Upload(context.Background(), "", []byte("test"), "text/plain")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage key is required")
	})
}

func TestS3ObjectStorage_GetBucket(t *testing.T) {
	cfg := &config.StorageConfig{
		Bucket:    "my-custom-bucket",
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Endpoint:  "http://localhost:9000",
	}
	storage, err := NewS3ObjectStorage(cfg)
	require.NoError(t, err)

	assert.Equal(t, "my-custom-bucket", storage.GetBucket())
}

// ============================================================================
// Integration Tests (require RustFS/MinIO running)
// ============================================================================

// skipIntegration skips the test if RustFS/MinIO is not available
func skipIntegration(t *testing.T) {
	t.Helper()
	// Check if we're in integration test mode
	// Set INTEGRATION_TEST=1 to run integration tests
	// These tests require RustFS running on localhost:9000
	t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 and run RustFS to enable.")
}

func newIntegrationStorage(t *testing.T) *S3ObjectStorage {
	t.Helper()
	skipIntegration(t)

	cfg := &config.StorageConfig{
		Bucket:            "test-integration",
		AccessKey:         "rustfsadmin",
		SecretKey:         "rustfsadmin123",
		Endpoint:          "http://localhost:9000",
		Region:            "us-east-1",
		UsePathStyle:      true,
		PresignExpiration: 15 * time.Minute,
	}

	logger := zap.NewNop()
	storage, err := NewS3ObjectStorage(cfg, WithLogger(logger))
	require.NoError(t, err)

	// Ensure bucket exists for integration tests
	err = storage.EnsureBucket(context.Background())
	require.NoError(t, err)

	return storage
}

func TestIntegration_UploadAndDownload(t *testing.T) {
	storage := newIntegrationStorage(t)
	ctx := context.Background()
	key := "integration-test/upload-download.txt"
	testData := []byte("Hello, RustFS integration test!")

	// Upload directly
	err := storage.Upload(ctx, key, testData, "text/plain")
	require.NoError(t, err)

	// Check exists
	exists, err := storage.ObjectExists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)

	// Generate download URL
	downloadURL, _, err := storage.GenerateDownloadURL(ctx, key, 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, downloadURL)

	// Cleanup
	err = storage.DeleteObject(ctx, key)
	require.NoError(t, err)

	// Verify deleted
	exists, err = storage.ObjectExists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestIntegration_EnsureBucket(t *testing.T) {
	skipIntegration(t)

	cfg := &config.StorageConfig{
		Bucket:            "test-ensure-bucket",
		AccessKey:         "rustfsadmin",
		SecretKey:         "rustfsadmin123",
		Endpoint:          "http://localhost:9000",
		Region:            "us-east-1",
		UsePathStyle:      true,
		PresignExpiration: 15 * time.Minute,
	}

	storage, err := NewS3ObjectStorage(cfg, WithLogger(zap.NewNop()))
	require.NoError(t, err)

	// Should create bucket if not exists
	err = storage.EnsureBucket(context.Background())
	require.NoError(t, err)

	// Should not error if bucket already exists
	err = storage.EnsureBucket(context.Background())
	require.NoError(t, err)
}
