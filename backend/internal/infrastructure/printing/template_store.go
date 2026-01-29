package printing

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// TemplateStore manages static print templates.
// It supports loading from an external directory (for customization)
// with fallback to embedded templates.
type TemplateStore struct {
	externalDir string
	templates   []StaticTemplate
	mu          sync.RWMutex
}

// StaticTemplate represents a static print template with loaded content
type StaticTemplate struct {
	ID          string // Generated stable ID based on doc type and paper size
	DocType     printing.DocType
	Name        string
	Description string
	PaperSize   printing.PaperSize
	Orientation printing.Orientation
	Margins     printing.Margins
	Content     string // Loaded HTML content
	IsDefault   bool
}

// TemplateStoreConfig configures the template store
type TemplateStoreConfig struct {
	// ExternalDir is the directory to load templates from.
	// If empty or directory doesn't exist, embedded templates are used.
	ExternalDir string
}

// NewTemplateStore creates a new template store
func NewTemplateStore(config *TemplateStoreConfig) (*TemplateStore, error) {
	store := &TemplateStore{}

	if config != nil && config.ExternalDir != "" {
		store.externalDir = config.ExternalDir
	}

	// Load all templates
	if err := store.loadTemplates(); err != nil {
		return nil, err
	}

	return store, nil
}

// loadTemplates loads all templates from external dir or embedded
func (s *TemplateStore) loadTemplates() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaults := GetDefaultTemplates()
	s.templates = make([]StaticTemplate, 0, len(defaults))

	for _, dt := range defaults {
		content, err := s.loadTemplateContent(dt.FilePath)
		if err != nil {
			return fmt.Errorf("failed to load template %s: %w", dt.Name, err)
		}

		// Generate a stable ID based on doc type, paper size, and orientation
		id := generateTemplateID(dt.DocType, dt.PaperSize, dt.Orientation)

		s.templates = append(s.templates, StaticTemplate{
			ID:          id,
			DocType:     dt.DocType,
			Name:        dt.Name,
			Description: dt.Description,
			PaperSize:   dt.PaperSize,
			Orientation: dt.Orientation,
			Margins:     dt.Margins,
			Content:     content,
			IsDefault:   dt.IsDefault,
		})
	}

	return nil
}

// loadTemplateContent loads template content from external dir or embedded
func (s *TemplateStore) loadTemplateContent(embeddedPath string) (string, error) {
	// Try external directory first
	if s.externalDir != "" {
		// Extract filename from embedded path (e.g., "templates/sales_delivery_a4.html" -> "sales_delivery_a4.html")
		filename := filepath.Base(embeddedPath)
		externalPath := filepath.Join(s.externalDir, filename)

		if content, err := os.ReadFile(externalPath); err == nil {
			return string(content), nil
		}
		// Fall through to embedded if external not found
	}

	// Use embedded template
	return LoadTemplateContent(embeddedPath)
}

// GetByID returns a template by its ID
func (s *TemplateStore) GetByID(id string) *StaticTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.templates {
		if s.templates[i].ID == id {
			return &s.templates[i]
		}
	}
	return nil
}

// GetByDocType returns all templates for a document type
func (s *TemplateStore) GetByDocType(docType printing.DocType) []StaticTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []StaticTemplate
	for _, t := range s.templates {
		if t.DocType == docType {
			result = append(result, t)
		}
	}
	return result
}

// GetDefault returns the default template for a document type
func (s *TemplateStore) GetDefault(docType printing.DocType) *StaticTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.templates {
		if s.templates[i].DocType == docType && s.templates[i].IsDefault {
			return &s.templates[i]
		}
	}
	return nil
}

// GetAll returns all templates
func (s *TemplateStore) GetAll() []StaticTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]StaticTemplate, len(s.templates))
	copy(result, s.templates)
	return result
}

// Reload reloads all templates from disk/embedded
func (s *TemplateStore) Reload() error {
	return s.loadTemplates()
}

// generateTemplateID generates a stable UUID v5 based on doc type, paper size, and orientation
// This ensures the same template always has the same ID
func generateTemplateID(docType printing.DocType, paperSize printing.PaperSize, orientation printing.Orientation) string {
	// Use a fixed namespace UUID for template IDs
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // URL namespace
	name := fmt.Sprintf("print-template:%s:%s:%s", docType, paperSize, orientation)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

// ToPrintTemplate converts a StaticTemplate to domain PrintTemplate for rendering
func (t *StaticTemplate) ToPrintTemplate() *printing.PrintTemplate {
	id := uuid.MustParse(t.ID)
	return &printing.PrintTemplate{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID: id,
				},
			},
		},
		DocumentType: t.DocType,
		Name:         t.Name,
		Description:  t.Description,
		Content:      t.Content,
		PaperSize:    t.PaperSize,
		Orientation:  t.Orientation,
		Margins:      t.Margins,
		IsDefault:    t.IsDefault,
		Status:       printing.TemplateStatusActive,
	}
}
