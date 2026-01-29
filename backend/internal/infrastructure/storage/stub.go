// Package storage provides object storage implementations for file operations.
package storage

import (
	"context"
	"errors"
	"time"

	catalogapp "github.com/erp/backend/internal/application/catalog"
)

// StubObjectStorage is a placeholder implementation of ObjectStorageService.
// It returns stub/empty values for all operations.
// Use this for development until a real storage backend (S3, RustFS, etc.) is implemented.
type StubObjectStorage struct {
	// BaseURL is the base URL for generating upload/download URLs
	// Defaults to "https://storage.example.com" if not set
	BaseURL string
}

// NewStubObjectStorage creates a new StubObjectStorage
func NewStubObjectStorage() *StubObjectStorage {
	return &StubObjectStorage{
		BaseURL: "https://storage.example.com",
	}
}

// Ensure StubObjectStorage implements ObjectStorageService
var _ catalogapp.ObjectStorageService = (*StubObjectStorage)(nil)

// GenerateUploadURL generates a stub presigned URL for uploading a file
func (s *StubObjectStorage) GenerateUploadURL(
	ctx context.Context,
	storageKey, contentType string,
	expiresIn time.Duration,
) (string, time.Time, error) {
	if storageKey == "" {
		return "", time.Time{}, errors.New("storage key is required")
	}

	expiresAt := time.Now().Add(expiresIn)
	url := s.BaseURL + "/upload/" + storageKey + "?expires=" + expiresAt.Format(time.RFC3339)

	return url, expiresAt, nil
}

// GenerateDownloadURL generates a stub presigned URL for downloading a file
func (s *StubObjectStorage) GenerateDownloadURL(
	ctx context.Context,
	storageKey string,
	expiresIn time.Duration,
) (string, time.Time, error) {
	if storageKey == "" {
		return "", time.Time{}, errors.New("storage key is required")
	}

	expiresAt := time.Now().Add(expiresIn)
	url := s.BaseURL + "/download/" + storageKey + "?expires=" + expiresAt.Format(time.RFC3339)

	return url, expiresAt, nil
}

// DeleteObject is a no-op stub that always succeeds
func (s *StubObjectStorage) DeleteObject(ctx context.Context, storageKey string) error {
	if storageKey == "" {
		return errors.New("storage key is required")
	}
	// No-op: In stub mode, we don't actually delete anything
	return nil
}

// ObjectExists always returns true in stub mode
// This allows the upload confirmation flow to work during development
func (s *StubObjectStorage) ObjectExists(ctx context.Context, storageKey string) (bool, error) {
	if storageKey == "" {
		return false, errors.New("storage key is required")
	}
	// In stub mode, always return true to allow confirmation flow
	return true, nil
}
