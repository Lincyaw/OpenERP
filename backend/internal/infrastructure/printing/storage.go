package printing

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PDFStorage defines the interface for storing and retrieving PDF files
type PDFStorage interface {
	// Store saves a PDF file and returns its URL/path
	Store(ctx context.Context, req *StoreRequest) (*StoreResult, error)
	// Get retrieves a PDF file by its path
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	// Delete removes a PDF file
	Delete(ctx context.Context, path string) error
	// CleanupOlderThan removes files older than the specified duration
	CleanupOlderThan(ctx context.Context, age time.Duration) (int, error)
	// GetURL returns the accessible URL for a stored PDF
	GetURL(path string) string
}

// StoreRequest contains the parameters for storing a PDF
type StoreRequest struct {
	// TenantID for multi-tenant isolation
	TenantID uuid.UUID
	// JobID is the print job identifier
	JobID uuid.UUID
	// PDFData is the raw PDF content
	PDFData []byte
}

// StoreResult contains the result of storing a PDF
type StoreResult struct {
	// Path is the storage path (relative to base)
	Path string
	// URL is the accessible URL for the PDF
	URL string
	// Size is the file size in bytes
	Size int64
}

// FileSystemStorageConfig contains configuration for file system storage
type FileSystemStorageConfig struct {
	// BasePath is the root directory for PDF storage
	// Default: /data/prints
	BasePath string
	// BaseURL is the URL prefix for accessing PDFs
	// Example: https://erp.example.com/api/v1/prints
	BaseURL string
	// RetentionDays is how long to keep PDFs (0 = forever)
	RetentionDays int
	// Logger for operations
	Logger *zap.Logger
}

// FileSystemStorage stores PDFs on the local file system
type FileSystemStorage struct {
	config *FileSystemStorageConfig
	logger *zap.Logger
}

// NewFileSystemStorage creates a new file system based PDF storage
func NewFileSystemStorage(config *FileSystemStorageConfig) (*FileSystemStorage, error) {
	if config == nil {
		config = &FileSystemStorageConfig{}
	}

	// Set defaults
	if config.BasePath == "" {
		config.BasePath = "/data/prints"
	}
	if config.BaseURL == "" {
		config.BaseURL = "/api/v1/prints"
	}

	// Ensure base directory exists
	if err := os.MkdirAll(config.BasePath, 0755); err != nil {
		return nil, NewRenderError(ErrCodeStorageFailed,
			fmt.Sprintf("failed to create storage directory: %s", config.BasePath), err)
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &FileSystemStorage{
		config: config,
		logger: logger,
	}, nil
}

// Store saves a PDF file to the file system
// Path structure: {base}/{tenant_id}/{year}/{month}/{job_id}.pdf
func (s *FileSystemStorage) Store(ctx context.Context, req *StoreRequest) (*StoreResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, NewRenderError(ErrCodeStorageFailed, "operation cancelled", ctx.Err())
	default:
	}

	if req == nil {
		return nil, NewRenderError(ErrCodeStorageFailed, "store request is nil", nil)
	}
	if req.TenantID == uuid.Nil {
		return nil, NewRenderError(ErrCodeStorageFailed, "tenant ID is required", nil)
	}
	if req.JobID == uuid.Nil {
		return nil, NewRenderError(ErrCodeStorageFailed, "job ID is required", nil)
	}
	if len(req.PDFData) == 0 {
		return nil, NewRenderError(ErrCodeStorageFailed, "PDF data is empty", nil)
	}

	// Build directory path: {base}/{tenant_id}/{year}/{month}/
	now := time.Now()
	dirPath := filepath.Join(
		s.config.BasePath,
		req.TenantID.String(),
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
	)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, NewRenderError(ErrCodeStorageFailed, "failed to create directory", err)
	}

	// Build file path
	fileName := req.JobID.String() + ".pdf"
	filePath := filepath.Join(dirPath, fileName)

	// Write file
	if err := os.WriteFile(filePath, req.PDFData, 0644); err != nil {
		return nil, NewRenderError(ErrCodeStorageFailed, "failed to write PDF file", err)
	}

	// Calculate relative path for URL
	relativePath := filepath.Join(
		req.TenantID.String(),
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fileName,
	)

	url := s.GetURL(relativePath)

	s.logger.Info("PDF stored",
		zap.String("path", filePath),
		zap.Int("size", len(req.PDFData)),
		zap.String("url", url))

	return &StoreResult{
		Path: relativePath,
		URL:  url,
		Size: int64(len(req.PDFData)),
	}, nil
}

// Get retrieves a PDF file by its relative path
func (s *FileSystemStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, NewRenderError(ErrCodeStorageFailed, "operation cancelled", ctx.Err())
	default:
	}

	// Sanitize path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || containsDotDot(path) { // Check raw path for ".."
		s.logger.Warn("blocked potentially malicious path",
			zap.String("path", path),
			zap.String("cleanPath", cleanPath))
		return nil, NewRenderError(ErrCodeStorageFailed, "invalid path", nil)
	}

	fullPath := filepath.Join(s.config.BasePath, cleanPath)

	// Additional security: verify the resolved path is still under BasePath
	absBase, err := filepath.Abs(s.config.BasePath)
	if err != nil {
		return nil, NewRenderError(ErrCodeStorageFailed, "failed to resolve base path", err)
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, NewRenderError(ErrCodeStorageFailed, "failed to resolve file path", err)
	}
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		s.logger.Warn("path escape attempt blocked",
			zap.String("path", path),
			zap.String("absPath", absPath),
			zap.String("absBase", absBase))
		return nil, NewRenderError(ErrCodeStorageFailed, "invalid path", nil)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewRenderError(ErrCodeStorageFailed, "PDF not found", err)
		}
		return nil, NewRenderError(ErrCodeStorageFailed, "failed to open PDF file", err)
	}

	return file, nil
}

// Delete removes a PDF file
func (s *FileSystemStorage) Delete(ctx context.Context, path string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return NewRenderError(ErrCodeStorageFailed, "operation cancelled", ctx.Err())
	default:
	}

	// Sanitize path
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || containsDotDot(path) { // Check raw path for ".."
		s.logger.Warn("blocked potentially malicious path",
			zap.String("path", path),
			zap.String("cleanPath", cleanPath))
		return NewRenderError(ErrCodeStorageFailed, "invalid path", nil)
	}

	fullPath := filepath.Join(s.config.BasePath, cleanPath)

	// Additional security: verify the resolved path is still under BasePath
	absBase, err := filepath.Abs(s.config.BasePath)
	if err != nil {
		return NewRenderError(ErrCodeStorageFailed, "failed to resolve base path", err)
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return NewRenderError(ErrCodeStorageFailed, "failed to resolve file path", err)
	}
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		s.logger.Warn("path escape attempt blocked",
			zap.String("path", path),
			zap.String("absPath", absPath),
			zap.String("absBase", absBase))
		return NewRenderError(ErrCodeStorageFailed, "invalid path", nil)
	}

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, not an error
		}
		return NewRenderError(ErrCodeStorageFailed, "failed to delete PDF file", err)
	}

	s.logger.Info("PDF deleted", zap.String("path", path))
	return nil
}

// CleanupOlderThan removes files older than the specified duration
func (s *FileSystemStorage) CleanupOlderThan(ctx context.Context, age time.Duration) (int, error) {
	cutoff := time.Now().Add(-age)
	deletedCount := 0

	err := filepath.Walk(s.config.BasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Only process PDF files
		if info.IsDir() || filepath.Ext(path) != ".pdf" {
			return nil
		}

		// Check modification time
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				deletedCount++
				s.logger.Debug("deleted old PDF", zap.String("path", path))
			}
		}

		return nil
	})

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		return deletedCount, NewRenderError(ErrCodeStorageFailed, "cleanup walk failed", err)
	}

	s.logger.Info("cleanup completed",
		zap.Int("deleted", deletedCount),
		zap.Duration("age", age))

	return deletedCount, nil
}

// GetURL returns the accessible URL for a stored PDF
func (s *FileSystemStorage) GetURL(path string) string {
	// Clean the path and convert to URL format (forward slashes)
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	return fmt.Sprintf("%s/%s", s.config.BaseURL, cleanPath)
}

// containsDotDot checks if a path contains ".." components
func containsDotDot(path string) bool {
	// Split by both forward and backward slashes for cross-platform support
	// Use raw string splitting to detect ".." before any path normalization
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == filepath.Separator
	})
	return slices.Contains(parts, "..")
}

// splitPath splits a path into components (using filepath.Split for proper parsing)
func splitPath(path string) []string {
	// Use strings.Split for simple path component extraction
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == filepath.Separator
	})
	// Filter out empty parts
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// Ensure FileSystemStorage implements PDFStorage
var _ PDFStorage = (*FileSystemStorage)(nil)
