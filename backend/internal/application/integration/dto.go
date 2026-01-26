package integration

import (
	"time"

	"github.com/erp/backend/internal/domain/integration"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Product Mapping DTOs
// ---------------------------------------------------------------------------

// ProductMappingResponse represents a product mapping in API responses
type ProductMappingResponse struct {
	ID                  uuid.UUID                `json:"id"`
	TenantID            uuid.UUID                `json:"tenant_id"`
	LocalProductID      uuid.UUID                `json:"local_product_id"`
	PlatformCode        integration.PlatformCode `json:"platform_code"`
	PlatformDisplayName string                   `json:"platform_display_name"`
	PlatformProductID   string                   `json:"platform_product_id"`
	PlatformProductName string                   `json:"platform_product_name,omitempty"`
	PlatformCategoryID  string                   `json:"platform_category_id,omitempty"`
	SKUMappings         []SKUMappingResponse     `json:"sku_mappings"`
	IsActive            bool                     `json:"is_active"`
	SyncEnabled         bool                     `json:"sync_enabled"`
	LastSyncAt          *time.Time               `json:"last_sync_at,omitempty"`
	LastSyncStatus      integration.SyncStatus   `json:"last_sync_status"`
	LastSyncError       string                   `json:"last_sync_error,omitempty"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
}

// SKUMappingResponse represents a SKU mapping in API responses
type SKUMappingResponse struct {
	LocalSKUID      uuid.UUID `json:"local_sku_id"`
	PlatformSkuID   string    `json:"platform_sku_id"`
	PlatformSkuName string    `json:"platform_sku_name,omitempty"`
	IsActive        bool      `json:"is_active"`
}

// ProductMappingListResponse represents a product mapping in list responses (lighter)
type ProductMappingListResponse struct {
	ID                  uuid.UUID                `json:"id"`
	LocalProductID      uuid.UUID                `json:"local_product_id"`
	PlatformCode        integration.PlatformCode `json:"platform_code"`
	PlatformDisplayName string                   `json:"platform_display_name"`
	PlatformProductID   string                   `json:"platform_product_id"`
	PlatformProductName string                   `json:"platform_product_name,omitempty"`
	SKUCount            int                      `json:"sku_count"`
	IsActive            bool                     `json:"is_active"`
	SyncEnabled         bool                     `json:"sync_enabled"`
	LastSyncStatus      integration.SyncStatus   `json:"last_sync_status"`
	LastSyncAt          *time.Time               `json:"last_sync_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CreateProductMappingRequest represents a request to create a product mapping
type CreateProductMappingRequest struct {
	LocalProductID      uuid.UUID                `json:"local_product_id" validate:"required"`
	PlatformCode        integration.PlatformCode `json:"platform_code" validate:"required"`
	PlatformProductID   string                   `json:"platform_product_id" validate:"required"`
	PlatformProductName string                   `json:"platform_product_name,omitempty"`
	PlatformCategoryID  string                   `json:"platform_category_id,omitempty"`
}

// UpdateProductMappingRequest represents a request to update a product mapping
type UpdateProductMappingRequest struct {
	PlatformProductID   *string `json:"platform_product_id,omitempty"`
	PlatformProductName *string `json:"platform_product_name,omitempty"`
	PlatformCategoryID  *string `json:"platform_category_id,omitempty"`
	IsActive            *bool   `json:"is_active,omitempty"`
	SyncEnabled         *bool   `json:"sync_enabled,omitempty"`
}

// AddSKUMappingRequest represents a request to add a SKU mapping
type AddSKUMappingRequest struct {
	LocalSKUID      uuid.UUID `json:"local_sku_id" validate:"required"`
	PlatformSkuID   string    `json:"platform_sku_id" validate:"required"`
	PlatformSkuName string    `json:"platform_sku_name,omitempty"`
}

// BatchCreateMappingRequest represents a request to create mappings in batch
type BatchCreateMappingRequest struct {
	Mappings []CreateProductMappingRequest `json:"mappings" validate:"required,min=1,max=100"`
}

// BatchMappingIDsRequest represents a request with multiple mapping IDs
type BatchMappingIDsRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=100"`
}

// ProductMappingListFilter represents filter options for listing mappings
type ProductMappingListFilter struct {
	PlatformCode   string    `form:"platform_code"`
	IsActive       *bool     `form:"is_active"`
	SyncEnabled    *bool     `form:"sync_enabled"`
	LastSyncStatus string    `form:"last_sync_status"`
	LocalProductID uuid.UUID `form:"local_product_id"`
	Search         string    `form:"search"`
	Page           int       `form:"page"`
	PageSize       int       `form:"page_size"`
}

// ---------------------------------------------------------------------------
// Conversion functions
// ---------------------------------------------------------------------------

// ToProductMappingResponse converts a domain ProductMapping to a response DTO
func ToProductMappingResponse(m *integration.ProductMapping) ProductMappingResponse {
	skuResponses := make([]SKUMappingResponse, len(m.SKUMappings))
	for i, sku := range m.SKUMappings {
		skuResponses[i] = SKUMappingResponse{
			LocalSKUID:      sku.LocalSKUID,
			PlatformSkuID:   sku.PlatformSkuID,
			PlatformSkuName: sku.PlatformSkuName,
			IsActive:        sku.IsActive,
		}
	}

	return ProductMappingResponse{
		ID:                  m.ID,
		TenantID:            m.TenantID,
		LocalProductID:      m.LocalProductID,
		PlatformCode:        m.PlatformCode,
		PlatformDisplayName: m.PlatformCode.DisplayName(),
		PlatformProductID:   m.PlatformProductID,
		PlatformProductName: m.PlatformProductName,
		PlatformCategoryID:  m.PlatformCategoryID,
		SKUMappings:         skuResponses,
		IsActive:            m.IsActive,
		SyncEnabled:         m.SyncEnabled,
		LastSyncAt:          m.LastSyncAt,
		LastSyncStatus:      m.LastSyncStatus,
		LastSyncError:       m.LastSyncError,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

// ToProductMappingListResponse converts a domain ProductMapping to a list response DTO
func ToProductMappingListResponse(m *integration.ProductMapping) ProductMappingListResponse {
	return ProductMappingListResponse{
		ID:                  m.ID,
		LocalProductID:      m.LocalProductID,
		PlatformCode:        m.PlatformCode,
		PlatformDisplayName: m.PlatformCode.DisplayName(),
		PlatformProductID:   m.PlatformProductID,
		PlatformProductName: m.PlatformProductName,
		SKUCount:            len(m.SKUMappings),
		IsActive:            m.IsActive,
		SyncEnabled:         m.SyncEnabled,
		LastSyncStatus:      m.LastSyncStatus,
		LastSyncAt:          m.LastSyncAt,
	}
}

// ToProductMappingListResponses converts a slice of domain ProductMappings to list response DTOs
func ToProductMappingListResponses(mappings []integration.ProductMapping) []ProductMappingListResponse {
	responses := make([]ProductMappingListResponse, len(mappings))
	for i := range mappings {
		responses[i] = ToProductMappingListResponse(&mappings[i])
	}
	return responses
}

// ToProductMappingResponses converts a slice of domain ProductMappings to response DTOs
func ToProductMappingResponses(mappings []integration.ProductMapping) []ProductMappingResponse {
	responses := make([]ProductMappingResponse, len(mappings))
	for i := range mappings {
		responses[i] = ToProductMappingResponse(&mappings[i])
	}
	return responses
}

// ToDomainFilter converts a list filter to domain filter
func (f ProductMappingListFilter) ToDomainFilter() integration.ProductMappingFilter {
	filter := integration.ProductMappingFilter{
		IsActive:    f.IsActive,
		SyncEnabled: f.SyncEnabled,
		Page:        f.Page,
		PageSize:    f.PageSize,
	}

	// Convert platform code
	if f.PlatformCode != "" {
		pc := integration.PlatformCode(f.PlatformCode)
		if pc.IsValid() {
			filter.PlatformCode = &pc
		}
	}

	// Convert sync status
	if f.LastSyncStatus != "" {
		status := integration.SyncStatus(f.LastSyncStatus)
		if status.IsValid() {
			filter.LastSyncStatus = &status
		}
	}

	// Convert local product ID filter
	if f.LocalProductID != uuid.Nil {
		filter.LocalProductIDs = []uuid.UUID{f.LocalProductID}
	}

	// Search keyword
	filter.SearchKeyword = f.Search

	return filter
}
