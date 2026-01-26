package integration

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// ProductMapping Entity
// ---------------------------------------------------------------------------

// ProductMapping represents the mapping between a local product and a platform product.
// This is an Entity (not Aggregate Root) as it doesn't have its own lifecycle events,
// but it has identity and is mutable.
type ProductMapping struct {
	// ID is the unique identifier of this mapping
	ID uuid.UUID
	// TenantID is the tenant this mapping belongs to
	TenantID uuid.UUID
	// LocalProductID is our internal product ID
	LocalProductID uuid.UUID
	// PlatformCode identifies which platform this mapping is for
	PlatformCode PlatformCode
	// PlatformProductID is the product ID on the platform
	PlatformProductID string
	// PlatformProductName is the product name on the platform (for reference)
	PlatformProductName string
	// PlatformCategoryID is the category ID on the platform
	PlatformCategoryID string
	// SKUMappings contains the SKU-level mappings
	SKUMappings []SKUMapping
	// IsActive indicates if this mapping is currently active
	IsActive bool
	// SyncEnabled indicates if auto-sync is enabled for this mapping
	SyncEnabled bool
	// LastSyncAt is when this mapping was last synced
	LastSyncAt *time.Time
	// LastSyncStatus is the result of the last sync
	LastSyncStatus SyncStatus
	// LastSyncError contains any error from last sync
	LastSyncError string
	// CreatedAt is when this mapping was created
	CreatedAt time.Time
	// UpdatedAt is when this mapping was last updated
	UpdatedAt time.Time
}

// NewProductMapping creates a new product mapping
func NewProductMapping(
	tenantID uuid.UUID,
	localProductID uuid.UUID,
	platformCode PlatformCode,
	platformProductID string,
) (*ProductMapping, error) {
	if tenantID == uuid.Nil {
		return nil, ErrMappingInvalidTenantID
	}
	if localProductID == uuid.Nil {
		return nil, ErrMappingInvalidProductID
	}
	if !platformCode.IsValid() {
		return nil, ErrMappingInvalidPlatformCode
	}
	if platformProductID == "" {
		return nil, ErrMappingInvalidPlatformID
	}

	now := time.Now()
	return &ProductMapping{
		ID:                uuid.New(),
		TenantID:          tenantID,
		LocalProductID:    localProductID,
		PlatformCode:      platformCode,
		PlatformProductID: platformProductID,
		SKUMappings:       make([]SKUMapping, 0),
		IsActive:          true,
		SyncEnabled:       true,
		LastSyncStatus:    SyncStatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// Validate validates the product mapping
func (m *ProductMapping) Validate() error {
	if m.TenantID == uuid.Nil {
		return ErrMappingInvalidTenantID
	}
	if m.LocalProductID == uuid.Nil {
		return ErrMappingInvalidProductID
	}
	if !m.PlatformCode.IsValid() {
		return ErrMappingInvalidPlatformCode
	}
	if m.PlatformProductID == "" {
		return ErrMappingInvalidPlatformID
	}
	return nil
}

// AddSKUMapping adds a SKU mapping to this product mapping
func (m *ProductMapping) AddSKUMapping(localSKUID uuid.UUID, platformSkuID string) error {
	if localSKUID == uuid.Nil {
		return errors.New("integration: invalid local SKU ID")
	}
	if platformSkuID == "" {
		return errors.New("integration: invalid platform SKU ID")
	}

	// Check for duplicate
	for _, existing := range m.SKUMappings {
		if existing.LocalSKUID == localSKUID && existing.PlatformSkuID == platformSkuID {
			return nil // Already exists
		}
	}

	m.SKUMappings = append(m.SKUMappings, SKUMapping{
		LocalSKUID:    localSKUID,
		PlatformSkuID: platformSkuID,
		IsActive:      true,
	})
	m.UpdatedAt = time.Now()
	return nil
}

// RemoveSKUMapping removes a SKU mapping by platform SKU ID
func (m *ProductMapping) RemoveSKUMapping(platformSkuID string) {
	for i, mapping := range m.SKUMappings {
		if mapping.PlatformSkuID == platformSkuID {
			m.SKUMappings = append(m.SKUMappings[:i], m.SKUMappings[i+1:]...)
			m.UpdatedAt = time.Now()
			return
		}
	}
}

// GetLocalSKUID returns the local SKU ID for a platform SKU ID
func (m *ProductMapping) GetLocalSKUID(platformSkuID string) (uuid.UUID, bool) {
	for _, mapping := range m.SKUMappings {
		if mapping.PlatformSkuID == platformSkuID && mapping.IsActive {
			return mapping.LocalSKUID, true
		}
	}
	return uuid.Nil, false
}

// GetPlatformSkuID returns the platform SKU ID for a local SKU ID
func (m *ProductMapping) GetPlatformSkuID(localSKUID uuid.UUID) (string, bool) {
	for _, mapping := range m.SKUMappings {
		if mapping.LocalSKUID == localSKUID && mapping.IsActive {
			return mapping.PlatformSkuID, true
		}
	}
	return "", false
}

// Activate activates this mapping
func (m *ProductMapping) Activate() {
	m.IsActive = true
	m.UpdatedAt = time.Now()
}

// Deactivate deactivates this mapping
func (m *ProductMapping) Deactivate() {
	m.IsActive = false
	m.UpdatedAt = time.Now()
}

// EnableSync enables automatic synchronization
func (m *ProductMapping) EnableSync() {
	m.SyncEnabled = true
	m.UpdatedAt = time.Now()
}

// DisableSync disables automatic synchronization
func (m *ProductMapping) DisableSync() {
	m.SyncEnabled = false
	m.UpdatedAt = time.Now()
}

// RecordSyncSuccess records a successful sync
func (m *ProductMapping) RecordSyncSuccess() {
	now := time.Now()
	m.LastSyncAt = &now
	m.LastSyncStatus = SyncStatusSuccess
	m.LastSyncError = ""
	m.UpdatedAt = now
}

// RecordSyncFailure records a failed sync
func (m *ProductMapping) RecordSyncFailure(errMsg string) {
	now := time.Now()
	m.LastSyncAt = &now
	m.LastSyncStatus = SyncStatusFailed
	m.LastSyncError = errMsg
	m.UpdatedAt = now
}

// ---------------------------------------------------------------------------
// SKUMapping Value Object
// ---------------------------------------------------------------------------

// SKUMapping represents the mapping between a local SKU and a platform SKU
type SKUMapping struct {
	// LocalSKUID is our internal SKU/variant ID
	LocalSKUID uuid.UUID
	// PlatformSkuID is the SKU ID on the platform
	PlatformSkuID string
	// PlatformSkuName is the SKU name on the platform (for reference)
	PlatformSkuName string
	// IsActive indicates if this SKU mapping is active
	IsActive bool
}

// ---------------------------------------------------------------------------
// ProductMappingRepository Interface
// ---------------------------------------------------------------------------

// ProductMappingReader defines the interface for reading product mappings
type ProductMappingReader interface {
	// FindByID finds a mapping by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*ProductMapping, error)

	// FindByLocalProduct finds mappings for a local product
	// Returns all platform mappings for this product
	FindByLocalProduct(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID) ([]ProductMapping, error)

	// FindByLocalProductAndPlatform finds a specific mapping
	FindByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode PlatformCode) (*ProductMapping, error)

	// FindByPlatformProduct finds a mapping by platform product ID
	FindByPlatformProduct(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, platformProductID string) (*ProductMapping, error)

	// FindByPlatformSku finds a mapping by platform SKU ID
	FindByPlatformSku(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, platformSkuID string) (*ProductMapping, error)
}

// ProductMappingFinder defines the interface for searching product mappings
type ProductMappingFinder interface {
	// FindAll finds all mappings for a tenant with optional filters
	FindAll(ctx context.Context, tenantID uuid.UUID, filter ProductMappingFilter) ([]ProductMapping, error)

	// FindActiveForPlatform finds all active mappings for a platform
	FindActiveForPlatform(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) ([]ProductMapping, error)

	// FindSyncEnabled finds all mappings with sync enabled
	FindSyncEnabled(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) ([]ProductMapping, error)

	// Count counts mappings matching the filter
	Count(ctx context.Context, tenantID uuid.UUID, filter ProductMappingFilter) (int64, error)

	// ExistsByLocalProductAndPlatform checks if a mapping exists
	ExistsByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode PlatformCode) (bool, error)

	// ExistsByPlatformProduct checks if a mapping exists for a platform product
	ExistsByPlatformProduct(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, platformProductID string) (bool, error)
}

// ProductMappingWriter defines the interface for persisting product mappings
type ProductMappingWriter interface {
	// Save creates or updates a mapping
	Save(ctx context.Context, mapping *ProductMapping) error

	// SaveBatch creates or updates multiple mappings
	SaveBatch(ctx context.Context, mappings []*ProductMapping) error

	// Delete deletes a mapping
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByLocalProduct deletes all mappings for a local product
	DeleteByLocalProduct(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID) error

	// DeleteByLocalProductAndPlatform deletes a specific mapping
	DeleteByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode PlatformCode) error
}

// ProductMappingRepository defines the full interface for product mapping persistence
type ProductMappingRepository interface {
	ProductMappingReader
	ProductMappingFinder
	ProductMappingWriter
}

// ProductMappingFilter defines filter criteria for product mappings
type ProductMappingFilter struct {
	// PlatformCode filters by platform (optional)
	PlatformCode *PlatformCode
	// IsActive filters by active status (optional)
	IsActive *bool
	// SyncEnabled filters by sync enabled status (optional)
	SyncEnabled *bool
	// LastSyncStatus filters by last sync status (optional)
	LastSyncStatus *SyncStatus
	// LocalProductIDs filters by local product IDs (optional)
	LocalProductIDs []uuid.UUID
	// SearchKeyword searches in product names (optional)
	SearchKeyword string
	// Page number (1-indexed)
	Page int
	// Page size
	PageSize int
}

// ---------------------------------------------------------------------------
// ProductMappingService Interface
// ---------------------------------------------------------------------------

// ProductMappingService defines the application service interface for managing
// product mappings between local products and platform products.
type ProductMappingService interface {
	// ---------------------------------------------------------------------------
	// CRUD Operations
	// ---------------------------------------------------------------------------

	// CreateMapping creates a new product mapping
	CreateMapping(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode PlatformCode, platformProductID string) (*ProductMapping, error)

	// UpdateMapping updates an existing mapping
	UpdateMapping(ctx context.Context, mapping *ProductMapping) error

	// DeleteMapping deletes a mapping
	DeleteMapping(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error

	// GetMapping retrieves a mapping by ID
	GetMapping(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*ProductMapping, error)

	// ListMappings lists mappings with filtering
	ListMappings(ctx context.Context, tenantID uuid.UUID, filter ProductMappingFilter) ([]ProductMapping, int64, error)

	// ---------------------------------------------------------------------------
	// SKU Mapping Operations
	// ---------------------------------------------------------------------------

	// AddSKUMapping adds a SKU mapping to a product mapping
	AddSKUMapping(ctx context.Context, tenantID uuid.UUID, mappingID uuid.UUID, localSKUID uuid.UUID, platformSkuID string) error

	// RemoveSKUMapping removes a SKU mapping
	RemoveSKUMapping(ctx context.Context, tenantID uuid.UUID, mappingID uuid.UUID, platformSkuID string) error

	// ---------------------------------------------------------------------------
	// Lookup Operations
	// ---------------------------------------------------------------------------

	// GetLocalProductID returns the local product ID for a platform product
	GetLocalProductID(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, platformProductID string) (uuid.UUID, error)

	// GetLocalSKUID returns the local SKU ID for a platform SKU
	GetLocalSKUID(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, platformSkuID string) (uuid.UUID, error)

	// GetPlatformProductID returns the platform product ID for a local product
	GetPlatformProductID(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode PlatformCode) (string, error)

	// ---------------------------------------------------------------------------
	// Sync Operations
	// ---------------------------------------------------------------------------

	// EnableSync enables sync for a mapping
	EnableSync(ctx context.Context, tenantID uuid.UUID, mappingID uuid.UUID) error

	// DisableSync disables sync for a mapping
	DisableSync(ctx context.Context, tenantID uuid.UUID, mappingID uuid.UUID) error

	// GetMappingsForSync returns all mappings that need to be synced to a platform
	GetMappingsForSync(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) ([]ProductMapping, error)
}
