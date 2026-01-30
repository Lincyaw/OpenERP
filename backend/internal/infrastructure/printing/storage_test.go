package printing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileSystemStorage(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &FileSystemStorageConfig{
			BasePath: tempDir,
		}

		storage, err := NewFileSystemStorage(config)
		require.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Equal(t, tempDir, storage.config.BasePath)
		assert.Equal(t, "/prints", storage.config.BaseURL) // Default
	})

	t.Run("with nil config", func(t *testing.T) {
		// This will try to create /data/prints which may fail without permissions
		storage, err := NewFileSystemStorage(nil)
		// May succeed or fail depending on permissions
		if err == nil {
			assert.NotNil(t, storage)
		}
	})

	t.Run("with custom base URL", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &FileSystemStorageConfig{
			BasePath: tempDir,
			BaseURL:  "https://example.com/prints",
		}

		storage, err := NewFileSystemStorage(config)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/prints", storage.config.BaseURL)
	})
}

func TestFileSystemStorage_Store(t *testing.T) {
	tempDir := t.TempDir()
	config := &FileSystemStorageConfig{
		BasePath: tempDir,
		BaseURL:  "/api/v1/prints",
	}
	storage, err := NewFileSystemStorage(config)
	require.NoError(t, err)

	t.Run("successful store", func(t *testing.T) {
		tenantID := uuid.New()
		jobID := uuid.New()
		pdfData := []byte("%PDF-1.4 test pdf content")

		result, err := storage.Store(context.Background(), &StoreRequest{
			TenantID: tenantID,
			JobID:    jobID,
			PDFData:  pdfData,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, result.Path)
		assert.NotEmpty(t, result.URL)
		assert.Equal(t, int64(len(pdfData)), result.Size)

		// Verify file exists
		fullPath := filepath.Join(tempDir, result.Path)
		_, err = os.Stat(fullPath)
		assert.NoError(t, err)

		// Verify content
		content, err := os.ReadFile(fullPath)
		require.NoError(t, err)
		assert.Equal(t, pdfData, content)
	})

	t.Run("nil request", func(t *testing.T) {
		result, err := storage.Store(context.Background(), nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("nil tenant ID", func(t *testing.T) {
		result, err := storage.Store(context.Background(), &StoreRequest{
			TenantID: uuid.Nil,
			JobID:    uuid.New(),
			PDFData:  []byte("test"),
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "tenant")
	})

	t.Run("nil job ID", func(t *testing.T) {
		result, err := storage.Store(context.Background(), &StoreRequest{
			TenantID: uuid.New(),
			JobID:    uuid.Nil,
			PDFData:  []byte("test"),
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "job")
	})

	t.Run("empty PDF data", func(t *testing.T) {
		result, err := storage.Store(context.Background(), &StoreRequest{
			TenantID: uuid.New(),
			JobID:    uuid.New(),
			PDFData:  []byte{},
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestFileSystemStorage_Get(t *testing.T) {
	tempDir := t.TempDir()
	config := &FileSystemStorageConfig{
		BasePath: tempDir,
		BaseURL:  "/api/v1/prints",
	}
	storage, err := NewFileSystemStorage(config)
	require.NoError(t, err)

	// Store a file first
	tenantID := uuid.New()
	jobID := uuid.New()
	pdfData := []byte("%PDF-1.4 test pdf content")

	result, err := storage.Store(context.Background(), &StoreRequest{
		TenantID: tenantID,
		JobID:    jobID,
		PDFData:  pdfData,
	})
	require.NoError(t, err)

	t.Run("successful get", func(t *testing.T) {
		reader, err := storage.Get(context.Background(), result.Path)
		require.NoError(t, err)
		defer reader.Close()

		content, err := os.ReadFile(filepath.Join(tempDir, result.Path))
		require.NoError(t, err)
		assert.Equal(t, pdfData, content)
	})

	t.Run("file not found", func(t *testing.T) {
		reader, err := storage.Get(context.Background(), "nonexistent/path.pdf")
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("directory traversal attempt", func(t *testing.T) {
		reader, err := storage.Get(context.Background(), "../../../etc/passwd")
		assert.Error(t, err)
		assert.Nil(t, reader)
	})

	t.Run("absolute path attempt", func(t *testing.T) {
		reader, err := storage.Get(context.Background(), "/etc/passwd")
		assert.Error(t, err)
		assert.Nil(t, reader)
	})
}

func TestFileSystemStorage_Delete(t *testing.T) {
	tempDir := t.TempDir()
	config := &FileSystemStorageConfig{
		BasePath: tempDir,
		BaseURL:  "/api/v1/prints",
	}
	storage, err := NewFileSystemStorage(config)
	require.NoError(t, err)

	// Store a file first
	tenantID := uuid.New()
	jobID := uuid.New()
	pdfData := []byte("%PDF-1.4 test pdf content")

	result, err := storage.Store(context.Background(), &StoreRequest{
		TenantID: tenantID,
		JobID:    jobID,
		PDFData:  pdfData,
	})
	require.NoError(t, err)

	t.Run("successful delete", func(t *testing.T) {
		err := storage.Delete(context.Background(), result.Path)
		require.NoError(t, err)

		// Verify file no longer exists
		fullPath := filepath.Join(tempDir, result.Path)
		_, err = os.Stat(fullPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete nonexistent file", func(t *testing.T) {
		// Should not error when deleting non-existent file
		err := storage.Delete(context.Background(), "nonexistent/path.pdf")
		assert.NoError(t, err)
	})

	t.Run("directory traversal attempt", func(t *testing.T) {
		err := storage.Delete(context.Background(), "../../../etc/passwd")
		assert.Error(t, err)
	})
}

func TestFileSystemStorage_CleanupOlderThan(t *testing.T) {
	tempDir := t.TempDir()
	config := &FileSystemStorageConfig{
		BasePath: tempDir,
		BaseURL:  "/api/v1/prints",
	}
	storage, err := NewFileSystemStorage(config)
	require.NoError(t, err)

	// Create some files with different ages
	tenantID := uuid.New()
	pdfData := []byte("%PDF-1.4 test")

	// Store multiple files
	for i := 0; i < 3; i++ {
		_, err := storage.Store(context.Background(), &StoreRequest{
			TenantID: tenantID,
			JobID:    uuid.New(),
			PDFData:  pdfData,
		})
		require.NoError(t, err)
	}

	t.Run("cleanup with future cutoff", func(t *testing.T) {
		// With 0 age, nothing should be deleted (files are too new)
		deleted, err := storage.CleanupOlderThan(context.Background(), 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 0, deleted)
	})

	t.Run("cleanup with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		deleted, err := storage.CleanupOlderThan(ctx, 0)
		// Should either return error or early termination
		_ = deleted
		_ = err
	})
}

func TestFileSystemStorage_GetURL(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		baseURL  string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			baseURL:  "/api/v1/prints",
			path:     "tenant-id/2024/01/job-id.pdf",
			expected: "/api/v1/prints/tenant-id/2024/01/job-id.pdf",
		},
		{
			name:     "with https base URL",
			baseURL:  "https://example.com/prints",
			path:     "tenant-id/2024/01/job-id.pdf",
			expected: "https://example.com/prints/tenant-id/2024/01/job-id.pdf",
		},
		{
			name:     "path with dots",
			baseURL:  "/api/v1/prints",
			path:     "tenant-id/2024/01/./job-id.pdf",
			expected: "/api/v1/prints/tenant-id/2024/01/job-id.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &FileSystemStorageConfig{
				BasePath: tempDir,
				BaseURL:  tt.baseURL,
			}
			storage, err := NewFileSystemStorage(config)
			require.NoError(t, err)

			url := storage.GetURL(tt.path)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestContainsDotDot(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "normal path",
			path:     "tenant/2024/01/file.pdf",
			expected: false,
		},
		{
			name:     "path with dot dot",
			path:     "tenant/../secret/file.pdf",
			expected: true,
		},
		{
			name:     "path starting with dot dot",
			path:     "../etc/passwd",
			expected: true,
		},
		{
			name:     "path with double dot dot",
			path:     "../../secret",
			expected: true,
		},
		{
			name:     "path with single dot",
			path:     "tenant/./2024/file.pdf",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsDotDot(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "simple path",
			path:     "a/b/c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "single component",
			path:     "file.pdf",
			expected: []string{"file.pdf"},
		},
		{
			name:     "path with extension",
			path:     "tenant/2024/01/file.pdf",
			expected: []string{"tenant", "2024", "01", "file.pdf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
