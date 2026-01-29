// Package storage provides object storage implementations for file operations.
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	catalogapp "github.com/erp/backend/internal/application/catalog"
	infraconfig "github.com/erp/backend/internal/infrastructure/config"
	"go.uber.org/zap"
)

// Ensure S3ObjectStorage implements ObjectStorageService
var _ catalogapp.ObjectStorageService = (*S3ObjectStorage)(nil)

// S3ObjectStorage implements ObjectStorageService using AWS S3 SDK v2.
// It is compatible with any S3-compatible storage (AWS S3, RustFS, MinIO, etc.)
type S3ObjectStorage struct {
	client            *s3.Client
	presignClient     *s3.PresignClient
	bucket            string
	presignExpiration time.Duration
	logger            *zap.Logger
}

// S3ObjectStorageOption is a functional option for configuring S3ObjectStorage
type S3ObjectStorageOption func(*S3ObjectStorage)

// WithLogger sets a custom logger for S3ObjectStorage
func WithLogger(logger *zap.Logger) S3ObjectStorageOption {
	return func(s *S3ObjectStorage) {
		s.logger = logger
	}
}

// WithPresignExpiration sets a custom presign expiration duration
func WithPresignExpiration(d time.Duration) S3ObjectStorageOption {
	return func(s *S3ObjectStorage) {
		s.presignExpiration = d
	}
}

// NewS3ObjectStorage creates a new S3ObjectStorage from configuration.
// It supports any S3-compatible storage backend (AWS S3, RustFS, MinIO, etc.)
func NewS3ObjectStorage(cfg *infraconfig.StorageConfig, opts ...S3ObjectStorageOption) (*S3ObjectStorage, error) {
	if cfg == nil {
		return nil, errors.New("storage configuration is required")
	}

	// Validate required configuration
	if cfg.Bucket == "" {
		return nil, errors.New("storage bucket is required")
	}
	if cfg.AccessKey == "" {
		return nil, errors.New("storage access key is required")
	}
	if cfg.SecretKey == "" {
		return nil, errors.New("storage secret key is required")
	}

	// Build endpoint URL
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:9000" // RustFS default
	}

	// Ensure endpoint has protocol
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if cfg.UseSSL {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}

	// Validate endpoint URL
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid storage endpoint: %w", err)
	}

	// Create AWS SDK config with custom credentials and endpoint
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"", // session token (not used for static credentials)
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create S3 client with path-style addressing and custom endpoint
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.BaseEndpoint = aws.String(endpoint)
	})

	// Create presign client
	presignClient := s3.NewPresignClient(client)

	// Create storage instance
	storage := &S3ObjectStorage{
		client:            client,
		presignClient:     presignClient,
		bucket:            cfg.Bucket,
		presignExpiration: cfg.PresignExpiration,
		logger:            zap.NewNop(),
	}

	// Apply options
	for _, opt := range opts {
		opt(storage)
	}

	// Set default presign expiration if not set
	if storage.presignExpiration == 0 {
		storage.presignExpiration = 15 * time.Minute
	}

	return storage, nil
}

// EnsureBucket creates the bucket if it doesn't exist.
// Call this during application startup to ensure the bucket is ready.
func (s *S3ObjectStorage) EnsureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		// Bucket exists
		return nil
	}

	// Check if error is because bucket doesn't exist
	var notFound *types.NotFound
	var noSuchBucket *types.NoSuchBucket
	if !errors.As(err, &notFound) && !errors.As(err, &noSuchBucket) {
		// Some other error
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	// Create bucket
	s.logger.Info("Creating storage bucket", zap.String("bucket", s.bucket))
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		// Ignore "BucketAlreadyOwnedByYou" error (race condition)
		var alreadyOwned *types.BucketAlreadyOwnedByYou
		if errors.As(err, &alreadyOwned) {
			return nil
		}
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	s.logger.Info("Storage bucket created successfully", zap.String("bucket", s.bucket))
	return nil
}

// GenerateUploadURL generates a presigned URL for uploading a file.
// The URL is valid for the configured presignExpiration duration.
func (s *S3ObjectStorage) GenerateUploadURL(
	ctx context.Context,
	storageKey, contentType string,
	expiresIn time.Duration,
) (string, time.Time, error) {
	if storageKey == "" {
		return "", time.Time{}, errors.New("storage key is required")
	}

	// Use provided expiration or default
	if expiresIn <= 0 {
		expiresIn = s.presignExpiration
	}

	// Generate presigned PUT URL
	presignReq, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(storageKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	expiresAt := time.Now().Add(expiresIn)
	return presignReq.URL, expiresAt, nil
}

// GenerateDownloadURL generates a presigned URL for downloading a file.
// The URL is valid for the configured presignExpiration duration.
func (s *S3ObjectStorage) GenerateDownloadURL(
	ctx context.Context,
	storageKey string,
	expiresIn time.Duration,
) (string, time.Time, error) {
	if storageKey == "" {
		return "", time.Time{}, errors.New("storage key is required")
	}

	// Use provided expiration or default
	if expiresIn <= 0 {
		expiresIn = s.presignExpiration
	}

	// Generate presigned GET URL
	presignReq, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(storageKey),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate download URL: %w", err)
	}

	expiresAt := time.Now().Add(expiresIn)
	return presignReq.URL, expiresAt, nil
}

// DeleteObject deletes an object from storage.
func (s *S3ObjectStorage) DeleteObject(ctx context.Context, storageKey string) error {
	if storageKey == "" {
		return errors.New("storage key is required")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// ObjectExists checks if an object exists in storage.
func (s *S3ObjectStorage) ObjectExists(ctx context.Context, storageKey string) (bool, error) {
	if storageKey == "" {
		return false, errors.New("storage key is required")
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		// Check if error is "not found"
		var notFound *types.NotFound
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &notFound) || errors.As(err, &noSuchKey) {
			return false, nil
		}
		// Also check for S3 API error code "NotFound"
		// Some S3-compatible services return this differently
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// Upload uploads data directly to storage (for internal use).
// For user uploads, prefer using GenerateUploadURL with presigned URLs.
func (s *S3ObjectStorage) Upload(ctx context.Context, storageKey string, data []byte, contentType string) error {
	if storageKey == "" {
		return errors.New("storage key is required")
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(storageKey),
		Body:        strings.NewReader(string(data)),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

// GetBucket returns the bucket name
func (s *S3ObjectStorage) GetBucket() string {
	return s.bucket
}
