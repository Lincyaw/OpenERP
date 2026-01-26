package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/erp/backend/internal/domain/integration"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormProductMappingRepository implements ProductMappingRepository using GORM
type GormProductMappingRepository struct {
	db *gorm.DB
}

// NewGormProductMappingRepository creates a new GormProductMappingRepository
func NewGormProductMappingRepository(db *gorm.DB) *GormProductMappingRepository {
	return &GormProductMappingRepository{db: db}
}

// ---------------------------------------------------------------------------
// ProductMappingReader implementation
// ---------------------------------------------------------------------------

// FindByID finds a mapping by its ID
func (r *GormProductMappingRepository) FindByID(ctx context.Context, id uuid.UUID) (*integration.ProductMapping, error) {
	var model models.ProductMappingModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, integration.ErrMappingNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a mapping by ID within a specific tenant (safer for multi-tenant apps)
func (r *GormProductMappingRepository) FindByIDForTenant(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*integration.ProductMapping, error) {
	var model models.ProductMappingModel
	if err := r.db.WithContext(ctx).First(&model, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, integration.ErrMappingNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByLocalProduct finds mappings for a local product
func (r *GormProductMappingRepository) FindByLocalProduct(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID) ([]integration.ProductMapping, error) {
	var mappingModels []models.ProductMappingModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND local_product_id = ?", tenantID, localProductID).
		Order("platform_code ASC").
		Find(&mappingModels).Error; err != nil {
		return nil, err
	}

	mappings := make([]integration.ProductMapping, len(mappingModels))
	for i, model := range mappingModels {
		mappings[i] = *model.ToDomain()
	}
	return mappings, nil
}

// FindByLocalProductAndPlatform finds a specific mapping
func (r *GormProductMappingRepository) FindByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode integration.PlatformCode) (*integration.ProductMapping, error) {
	var model models.ProductMappingModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND local_product_id = ? AND platform_code = ?", tenantID, localProductID, platformCode).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, integration.ErrMappingNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByPlatformProduct finds a mapping by platform product ID
func (r *GormProductMappingRepository) FindByPlatformProduct(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, platformProductID string) (*integration.ProductMapping, error) {
	var model models.ProductMappingModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND platform_code = ? AND platform_product_id = ?", tenantID, platformCode, platformProductID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, integration.ErrMappingNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByPlatformSku finds a mapping by platform SKU ID
func (r *GormProductMappingRepository) FindByPlatformSku(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, platformSkuID string) (*integration.ProductMapping, error) {
	var mappingModels []models.ProductMappingModel
	// Safely construct JSON criteria using json.Marshal to prevent injection
	type skuCriteria struct {
		PlatformSkuID string `json:"platform_sku_id"`
	}
	criteria := []skuCriteria{{PlatformSkuID: platformSkuID}}
	criteriaJSON, err := json.Marshal(criteria)
	if err != nil {
		return nil, err
	}

	// Search in JSONB for the SKU mapping
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND platform_code = ? AND sku_mappings::jsonb @> ?",
			tenantID, platformCode, string(criteriaJSON)).
		Find(&mappingModels).Error; err != nil {
		return nil, err
	}

	if len(mappingModels) == 0 {
		return nil, integration.ErrMappingNotFound
	}

	// Return the first matching mapping
	return mappingModels[0].ToDomain(), nil
}

// ---------------------------------------------------------------------------
// ProductMappingFinder implementation
// ---------------------------------------------------------------------------

// FindAll finds all mappings for a tenant with optional filters
func (r *GormProductMappingRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter integration.ProductMappingFilter) ([]integration.ProductMapping, error) {
	var mappingModels []models.ProductMappingModel
	query := r.applyFilter(r.db.WithContext(ctx).Model(&models.ProductMappingModel{}).Where("tenant_id = ?", tenantID), filter)

	if err := query.Find(&mappingModels).Error; err != nil {
		return nil, err
	}

	mappings := make([]integration.ProductMapping, len(mappingModels))
	for i, model := range mappingModels {
		mappings[i] = *model.ToDomain()
	}
	return mappings, nil
}

// FindActiveForPlatform finds all active mappings for a platform
func (r *GormProductMappingRepository) FindActiveForPlatform(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) ([]integration.ProductMapping, error) {
	var mappingModels []models.ProductMappingModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND platform_code = ? AND is_active = ?", tenantID, platformCode, true).
		Order("created_at DESC").
		Find(&mappingModels).Error; err != nil {
		return nil, err
	}

	mappings := make([]integration.ProductMapping, len(mappingModels))
	for i, model := range mappingModels {
		mappings[i] = *model.ToDomain()
	}
	return mappings, nil
}

// FindSyncEnabled finds all mappings with sync enabled
func (r *GormProductMappingRepository) FindSyncEnabled(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode) ([]integration.ProductMapping, error) {
	var mappingModels []models.ProductMappingModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND platform_code = ? AND is_active = ? AND sync_enabled = ?", tenantID, platformCode, true, true).
		Order("created_at DESC").
		Find(&mappingModels).Error; err != nil {
		return nil, err
	}

	mappings := make([]integration.ProductMapping, len(mappingModels))
	for i, model := range mappingModels {
		mappings[i] = *model.ToDomain()
	}
	return mappings, nil
}

// Count counts mappings matching the filter
func (r *GormProductMappingRepository) Count(ctx context.Context, tenantID uuid.UUID, filter integration.ProductMappingFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.ProductMappingModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByLocalProductAndPlatform checks if a mapping exists
func (r *GormProductMappingRepository) ExistsByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode integration.PlatformCode) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductMappingModel{}).
		Where("tenant_id = ? AND local_product_id = ? AND platform_code = ?", tenantID, localProductID, platformCode).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByPlatformProduct checks if a mapping exists for a platform product
func (r *GormProductMappingRepository) ExistsByPlatformProduct(ctx context.Context, tenantID uuid.UUID, platformCode integration.PlatformCode, platformProductID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductMappingModel{}).
		Where("tenant_id = ? AND platform_code = ? AND platform_product_id = ?", tenantID, platformCode, platformProductID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---------------------------------------------------------------------------
// ProductMappingWriter implementation
// ---------------------------------------------------------------------------

// Save creates or updates a mapping
func (r *GormProductMappingRepository) Save(ctx context.Context, mapping *integration.ProductMapping) error {
	model := models.ProductMappingModelFromDomain(mapping)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveBatch creates or updates multiple mappings
func (r *GormProductMappingRepository) SaveBatch(ctx context.Context, mappings []*integration.ProductMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	mappingModels := make([]*models.ProductMappingModel, len(mappings))
	for i, m := range mappings {
		mappingModels[i] = models.ProductMappingModelFromDomain(m)
	}

	return r.db.WithContext(ctx).Save(mappingModels).Error
}

// Delete deletes a mapping
func (r *GormProductMappingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ProductMappingModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return integration.ErrMappingNotFound
	}
	return nil
}

// DeleteByLocalProduct deletes all mappings for a local product
func (r *GormProductMappingRepository) DeleteByLocalProduct(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Delete(&models.ProductMappingModel{}, "tenant_id = ? AND local_product_id = ?", tenantID, localProductID).Error
}

// DeleteByLocalProductAndPlatform deletes a specific mapping
func (r *GormProductMappingRepository) DeleteByLocalProductAndPlatform(ctx context.Context, tenantID uuid.UUID, localProductID uuid.UUID, platformCode integration.PlatformCode) error {
	result := r.db.WithContext(ctx).
		Delete(&models.ProductMappingModel{}, "tenant_id = ? AND local_product_id = ? AND platform_code = ?", tenantID, localProductID, platformCode)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return integration.ErrMappingNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Filter helpers
// ---------------------------------------------------------------------------

// applyFilter applies filter options to the query
func (r *GormProductMappingRepository) applyFilter(query *gorm.DB, filter integration.ProductMappingFilter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Default ordering
	query = query.Order("created_at DESC")

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormProductMappingRepository) applyFilterWithoutPagination(query *gorm.DB, filter integration.ProductMappingFilter) *gorm.DB {
	// Filter by platform code
	if filter.PlatformCode != nil && filter.PlatformCode.IsValid() {
		query = query.Where("platform_code = ?", *filter.PlatformCode)
	}

	// Filter by active status
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Filter by sync enabled
	if filter.SyncEnabled != nil {
		query = query.Where("sync_enabled = ?", *filter.SyncEnabled)
	}

	// Filter by last sync status
	if filter.LastSyncStatus != nil && filter.LastSyncStatus.IsValid() {
		query = query.Where("last_sync_status = ?", *filter.LastSyncStatus)
	}

	// Filter by local product IDs
	if len(filter.LocalProductIDs) > 0 {
		query = query.Where("local_product_id IN ?", filter.LocalProductIDs)
	}

	// Search in product name (escape LIKE special characters)
	if filter.SearchKeyword != "" {
		escaped := escapeLikePattern(filter.SearchKeyword)
		searchPattern := "%" + escaped + "%"
		query = query.Where("platform_product_name ILIKE ? OR platform_product_id ILIKE ?", searchPattern, searchPattern)
	}

	return query
}

// ---------------------------------------------------------------------------
// Additional batch operations
// ---------------------------------------------------------------------------

// FindByLocalProductIDs finds mappings for multiple local products
func (r *GormProductMappingRepository) FindByLocalProductIDs(ctx context.Context, tenantID uuid.UUID, localProductIDs []uuid.UUID, platformCode integration.PlatformCode) ([]integration.ProductMapping, error) {
	if len(localProductIDs) == 0 {
		return []integration.ProductMapping{}, nil
	}

	var mappingModels []models.ProductMappingModel
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND local_product_id IN ?", tenantID, localProductIDs)

	if platformCode.IsValid() {
		query = query.Where("platform_code = ?", platformCode)
	}

	if err := query.Find(&mappingModels).Error; err != nil {
		return nil, err
	}

	mappings := make([]integration.ProductMapping, len(mappingModels))
	for i, model := range mappingModels {
		mappings[i] = *model.ToDomain()
	}
	return mappings, nil
}

// UpdateSyncStatus updates the sync status for multiple mappings
func (r *GormProductMappingRepository) UpdateSyncStatus(ctx context.Context, ids []uuid.UUID, status integration.SyncStatus, errorMsg string) error {
	if len(ids) == 0 {
		return nil
	}

	updates := map[string]any{
		"last_sync_status": status,
		"last_sync_error":  errorMsg,
		"last_sync_at":     gorm.Expr("NOW()"),
		"updated_at":       gorm.Expr("NOW()"),
	}

	return r.db.WithContext(ctx).
		Model(&models.ProductMappingModel{}).
		Where("id IN ?", ids).
		Updates(updates).Error
}

// GetMappingStats returns statistics about mappings for a tenant
func (r *GormProductMappingRepository) GetMappingStats(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Total count
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductMappingModel{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	// Active count
	var active int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductMappingModel{}).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Count(&active).Error; err != nil {
		return nil, err
	}
	stats["active"] = active

	// Sync enabled count
	var syncEnabled int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductMappingModel{}).
		Where("tenant_id = ? AND sync_enabled = ?", tenantID, true).
		Count(&syncEnabled).Error; err != nil {
		return nil, err
	}
	stats["sync_enabled"] = syncEnabled

	// Failed sync count
	var failed int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductMappingModel{}).
		Where("tenant_id = ? AND last_sync_status = ?", tenantID, integration.SyncStatusFailed).
		Count(&failed).Error; err != nil {
		return nil, err
	}
	stats["failed"] = failed

	return stats, nil
}

// Ensure GormProductMappingRepository implements ProductMappingRepository
var _ integration.ProductMappingRepository = (*GormProductMappingRepository)(nil)

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// escapeLikePattern escapes special characters in LIKE patterns
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
