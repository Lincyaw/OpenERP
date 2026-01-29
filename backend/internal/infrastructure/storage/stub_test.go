package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStubObjectStorage(t *testing.T) {
	s := NewStubObjectStorage()
	require.NotNil(t, s)
	assert.Equal(t, "https://storage.example.com", s.BaseURL)
}

func TestStubObjectStorage_GenerateUploadURL(t *testing.T) {
	s := NewStubObjectStorage()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		url, expiresAt, err := s.GenerateUploadURL(ctx, "test/key/file.jpg", "image/jpeg", 15*time.Minute)
		require.NoError(t, err)
		assert.Contains(t, url, "https://storage.example.com/upload/test/key/file.jpg")
		assert.True(t, expiresAt.After(time.Now()))
	})

	t.Run("empty storage key", func(t *testing.T) {
		_, _, err := s.GenerateUploadURL(ctx, "", "image/jpeg", 15*time.Minute)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage key is required")
	})
}

func TestStubObjectStorage_GenerateDownloadURL(t *testing.T) {
	s := NewStubObjectStorage()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		url, expiresAt, err := s.GenerateDownloadURL(ctx, "test/key/file.jpg", 1*time.Hour)
		require.NoError(t, err)
		assert.Contains(t, url, "https://storage.example.com/download/test/key/file.jpg")
		assert.True(t, expiresAt.After(time.Now()))
	})

	t.Run("empty storage key", func(t *testing.T) {
		_, _, err := s.GenerateDownloadURL(ctx, "", 1*time.Hour)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage key is required")
	})
}

func TestStubObjectStorage_DeleteObject(t *testing.T) {
	s := NewStubObjectStorage()
	ctx := context.Background()

	t.Run("success - no-op", func(t *testing.T) {
		err := s.DeleteObject(ctx, "test/key/file.jpg")
		require.NoError(t, err)
	})

	t.Run("empty storage key", func(t *testing.T) {
		err := s.DeleteObject(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage key is required")
	})
}

func TestStubObjectStorage_ObjectExists(t *testing.T) {
	s := NewStubObjectStorage()
	ctx := context.Background()

	t.Run("always returns true for valid key", func(t *testing.T) {
		exists, err := s.ObjectExists(ctx, "test/key/file.jpg")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("empty storage key", func(t *testing.T) {
		exists, err := s.ObjectExists(ctx, "")
		require.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "storage key is required")
	})
}
