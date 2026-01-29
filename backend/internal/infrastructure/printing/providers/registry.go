// Package providers implements DataProvider for loading document data from repositories.
// Each provider retrieves data for a specific document type for use in print templates.
package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/erp/backend/internal/domain/printing"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/google/uuid"
)

// DataProviderRegistry manages DataProvider implementations for different document types.
// It provides a centralized way to look up and use providers based on document type.
type DataProviderRegistry struct {
	mu        sync.RWMutex
	providers map[printing.DocType]infra.DataProvider
}

// NewDataProviderRegistry creates a new empty DataProviderRegistry.
func NewDataProviderRegistry() *DataProviderRegistry {
	return &DataProviderRegistry{
		providers: make(map[printing.DocType]infra.DataProvider),
	}
}

// Register adds a DataProvider to the registry.
// If a provider for the same DocType already exists, it will be replaced.
func (r *DataProviderRegistry) Register(provider infra.DataProvider) {
	if provider == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.GetDocType()] = provider
}

// GetProvider returns the DataProvider for the given DocType.
// Returns (nil, false) if no provider is registered for that type.
func (r *DataProviderRegistry) GetProvider(docType printing.DocType) (infra.DataProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[docType]
	return provider, ok
}

// LoadData loads document data using the appropriate provider for the document type.
// Returns an error if no provider is registered for the given docType.
func (r *DataProviderRegistry) LoadData(ctx context.Context, tenantID uuid.UUID, docType printing.DocType, documentID uuid.UUID) (*infra.DocumentData, error) {
	provider, ok := r.GetProvider(docType)
	if !ok {
		return nil, fmt.Errorf("no data provider registered for document type: %s", docType)
	}
	return provider.GetData(ctx, tenantID, documentID)
}

// HasProvider checks if a provider is registered for the given DocType.
func (r *DataProviderRegistry) HasProvider(docType printing.DocType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.providers[docType]
	return ok
}

// RegisteredTypes returns all document types that have registered providers.
func (r *DataProviderRegistry) RegisteredTypes() []printing.DocType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]printing.DocType, 0, len(r.providers))
	for docType := range r.providers {
		types = append(types, docType)
	}
	return types
}
