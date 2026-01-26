package integration

import (
	"context"

	"github.com/erp/backend/internal/domain/integration"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// ProductMappingServiceImpl implements ProductMappingService interface
type ProductMappingServiceImpl struct {
	mappingRepo integration.ProductMappingRepository
}

// NewProductMappingService creates a new ProductMappingServiceImpl
func NewProductMappingService(mappingRepo integration.ProductMappingRepository) *ProductMappingServiceImpl {
	return &ProductMappingServiceImpl{
		mappingRepo: mappingRepo,
	}
}

// ---------------------------------------------------------------------------
// CRUD Operations
// ---------------------------------------------------------------------------

// CreateMapping creates a new product mapping
func (s *ProductMappingServiceImpl) CreateMapping(
	ctx context.Context,
	tenantID uuid.UUID,
	localProductID uuid.UUID,
	platformCode integration.PlatformCode,
	platformProductID string,
) (*integration.ProductMapping, error) {
	// Check if mapping already exists
	exists, err := s.mappingRepo.ExistsByLocalProductAndPlatform(ctx, tenantID, localProductID, platformCode)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, integration.ErrMappingAlreadyExists
	}

	// Check if platform product is already mapped
	exists, err = s.mappingRepo.ExistsByPlatformProduct(ctx, tenantID, platformCode, platformProductID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_MAPPED", "Platform product is already mapped to another local product")
	}

	// Create new mapping
	mapping, err := integration.NewProductMapping(tenantID, localProductID, platformCode, platformProductID)
	if err != nil {
		return nil, err
	}

	// Save mapping
	if err := s.mappingRepo.Save(ctx, mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

// UpdateMapping updates an existing mapping
func (s *ProductMappingServiceImpl) UpdateMapping(ctx context.Context, mapping *integration.ProductMapping) error {
	if err := mapping.Validate(); err != nil {
		return err
	}
	return s.mappingRepo.Save(ctx, mapping)
}

// DeleteMapping deletes a mapping
func (s *ProductMappingServiceImpl) DeleteMapping(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	// Verify the mapping exists and belongs to tenant
	mapping, err := s.mappingRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if mapping.TenantID != tenantID {
		return integration.ErrMappingNotFound
	}

	return s.mappingRepo.Delete(ctx, id)
}

// GetMapping retrieves a mapping by ID
func (s *ProductMappingServiceImpl) GetMapping(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*integration.ProductMapping, error) {
	mapping, err := s.mappingRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if mapping.TenantID != tenantID {
		return nil, integration.ErrMappingNotFound
	}
	return mapping, nil
}

// ListMappings lists mappings with filtering
func (s *ProductMappingServiceImpl) ListMappings(
	ctx context.Context,
	tenantID uuid.UUID,
	filter integration.ProductMappingFilter,
) ([]integration.ProductMapping, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	mappings, err := s.mappingRepo.FindAll(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.mappingRepo.Count(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}

	return mappings, count, nil
}

// ---------------------------------------------------------------------------
// SKU Mapping Operations
// ---------------------------------------------------------------------------

// AddSKUMapping adds a SKU mapping to a product mapping
func (s *ProductMappingServiceImpl) AddSKUMapping(
	ctx context.Context,
	tenantID uuid.UUID,
	mappingID uuid.UUID,
	localSKUID uuid.UUID,
	platformSkuID string,
) error {
	mapping, err := s.mappingRepo.FindByID(ctx, mappingID)
	if err != nil {
		return err
	}
	if mapping.TenantID != tenantID {
		return integration.ErrMappingNotFound
	}

	if err := mapping.AddSKUMapping(localSKUID, platformSkuID); err != nil {
		return err
	}

	return s.mappingRepo.Save(ctx, mapping)
}

// RemoveSKUMapping removes a SKU mapping
func (s *ProductMappingServiceImpl) RemoveSKUMapping(
	ctx context.Context,
	tenantID uuid.UUID,
	mappingID uuid.UUID,
	platformSkuID string,
) error {
	mapping, err := s.mappingRepo.FindByID(ctx, mappingID)
	if err != nil {
		return err
	}
	if mapping.TenantID != tenantID {
		return integration.ErrMappingNotFound
	}

	mapping.RemoveSKUMapping(platformSkuID)

	return s.mappingRepo.Save(ctx, mapping)
}

// ---------------------------------------------------------------------------
// Lookup Operations
// ---------------------------------------------------------------------------

// GetLocalProductID returns the local product ID for a platform product
func (s *ProductMappingServiceImpl) GetLocalProductID(
	ctx context.Context,
	tenantID uuid.UUID,
	platformCode integration.PlatformCode,
	platformProductID string,
) (uuid.UUID, error) {
	mapping, err := s.mappingRepo.FindByPlatformProduct(ctx, tenantID, platformCode, platformProductID)
	if err != nil {
		return uuid.Nil, err
	}
	return mapping.LocalProductID, nil
}

// GetLocalSKUID returns the local SKU ID for a platform SKU
func (s *ProductMappingServiceImpl) GetLocalSKUID(
	ctx context.Context,
	tenantID uuid.UUID,
	platformCode integration.PlatformCode,
	platformSkuID string,
) (uuid.UUID, error) {
	mapping, err := s.mappingRepo.FindByPlatformSku(ctx, tenantID, platformCode, platformSkuID)
	if err != nil {
		return uuid.Nil, err
	}

	localSKUID, found := mapping.GetLocalSKUID(platformSkuID)
	if !found {
		return uuid.Nil, integration.ErrMappingSkuMappingInvalid
	}

	return localSKUID, nil
}

// GetPlatformProductID returns the platform product ID for a local product
func (s *ProductMappingServiceImpl) GetPlatformProductID(
	ctx context.Context,
	tenantID uuid.UUID,
	localProductID uuid.UUID,
	platformCode integration.PlatformCode,
) (string, error) {
	mapping, err := s.mappingRepo.FindByLocalProductAndPlatform(ctx, tenantID, localProductID, platformCode)
	if err != nil {
		return "", err
	}
	return mapping.PlatformProductID, nil
}

// ---------------------------------------------------------------------------
// Sync Operations
// ---------------------------------------------------------------------------

// EnableSync enables sync for a mapping
func (s *ProductMappingServiceImpl) EnableSync(ctx context.Context, tenantID uuid.UUID, mappingID uuid.UUID) error {
	mapping, err := s.mappingRepo.FindByID(ctx, mappingID)
	if err != nil {
		return err
	}
	if mapping.TenantID != tenantID {
		return integration.ErrMappingNotFound
	}

	mapping.EnableSync()

	return s.mappingRepo.Save(ctx, mapping)
}

// DisableSync disables sync for a mapping
func (s *ProductMappingServiceImpl) DisableSync(ctx context.Context, tenantID uuid.UUID, mappingID uuid.UUID) error {
	mapping, err := s.mappingRepo.FindByID(ctx, mappingID)
	if err != nil {
		return err
	}
	if mapping.TenantID != tenantID {
		return integration.ErrMappingNotFound
	}

	mapping.DisableSync()

	return s.mappingRepo.Save(ctx, mapping)
}

// GetMappingsForSync returns all mappings that need to be synced to a platform
func (s *ProductMappingServiceImpl) GetMappingsForSync(
	ctx context.Context,
	tenantID uuid.UUID,
	platformCode integration.PlatformCode,
) ([]integration.ProductMapping, error) {
	return s.mappingRepo.FindSyncEnabled(ctx, tenantID, platformCode)
}

// ---------------------------------------------------------------------------
// Batch Operations
// ---------------------------------------------------------------------------

// CreateBatchMappings creates multiple product mappings in a batch
func (s *ProductMappingServiceImpl) CreateBatchMappings(
	ctx context.Context,
	tenantID uuid.UUID,
	requests []CreateMappingRequest,
) ([]CreateMappingResult, error) {
	results := make([]CreateMappingResult, len(requests))

	for i, req := range requests {
		result := CreateMappingResult{
			LocalProductID:    req.LocalProductID,
			PlatformCode:      req.PlatformCode,
			PlatformProductID: req.PlatformProductID,
		}

		mapping, err := s.CreateMapping(ctx, tenantID, req.LocalProductID, req.PlatformCode, req.PlatformProductID)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MappingID = mapping.ID
		}

		results[i] = result
	}

	return results, nil
}

// UpdateBatchSyncStatus updates sync status for multiple mappings
func (s *ProductMappingServiceImpl) UpdateBatchSyncStatus(
	ctx context.Context,
	tenantID uuid.UUID,
	ids []uuid.UUID,
	status integration.SyncStatus,
	errorMsg string,
) error {
	// Verify all mappings belong to tenant
	for _, id := range ids {
		mapping, err := s.mappingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if mapping.TenantID != tenantID {
			return integration.ErrMappingNotFound
		}
	}

	// Use repository's batch update if available
	// Otherwise update one by one
	for _, id := range ids {
		mapping, err := s.mappingRepo.FindByID(ctx, id)
		if err != nil {
			continue
		}

		switch status {
		case integration.SyncStatusSuccess:
			mapping.RecordSyncSuccess()
		case integration.SyncStatusFailed:
			mapping.RecordSyncFailure(errorMsg)
		default:
			// For other statuses, just update the status field
			mapping.LastSyncStatus = status
		}

		if err := s.mappingRepo.Save(ctx, mapping); err != nil {
			return err
		}
	}

	return nil
}

// ActivateMappings activates multiple mappings
func (s *ProductMappingServiceImpl) ActivateMappings(
	ctx context.Context,
	tenantID uuid.UUID,
	ids []uuid.UUID,
) error {
	for _, id := range ids {
		mapping, err := s.mappingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if mapping.TenantID != tenantID {
			return integration.ErrMappingNotFound
		}

		mapping.Activate()

		if err := s.mappingRepo.Save(ctx, mapping); err != nil {
			return err
		}
	}
	return nil
}

// DeactivateMappings deactivates multiple mappings
func (s *ProductMappingServiceImpl) DeactivateMappings(
	ctx context.Context,
	tenantID uuid.UUID,
	ids []uuid.UUID,
) error {
	for _, id := range ids {
		mapping, err := s.mappingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if mapping.TenantID != tenantID {
			return integration.ErrMappingNotFound
		}

		mapping.Deactivate()

		if err := s.mappingRepo.Save(ctx, mapping); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// DTOs
// ---------------------------------------------------------------------------

// CreateMappingRequest represents a request to create a mapping
type CreateMappingRequest struct {
	LocalProductID    uuid.UUID
	PlatformCode      integration.PlatformCode
	PlatformProductID string
}

// CreateMappingResult represents the result of creating a mapping
type CreateMappingResult struct {
	LocalProductID    uuid.UUID                `json:"local_product_id"`
	PlatformCode      integration.PlatformCode `json:"platform_code"`
	PlatformProductID string                   `json:"platform_product_id"`
	MappingID         uuid.UUID                `json:"mapping_id,omitempty"`
	Success           bool                     `json:"success"`
	Error             string                   `json:"error,omitempty"`
}

// Ensure ProductMappingServiceImpl implements ProductMappingService
var _ integration.ProductMappingService = (*ProductMappingServiceImpl)(nil)
