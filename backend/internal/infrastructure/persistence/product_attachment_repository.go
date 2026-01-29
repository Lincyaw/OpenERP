package persistence

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductAttachmentSortFields contains allowed sort fields for product attachments
var ProductAttachmentSortFields = map[string]bool{
	"id":           true,
	"created_at":   true,
	"updated_at":   true,
	"product_id":   true,
	"type":         true,
	"status":       true,
	"file_name":    true,
	"file_size":    true,
	"content_type": true,
	"sort_order":   true,
}

// GormProductAttachmentRepository implements ProductAttachmentRepository using GORM
type GormProductAttachmentRepository struct {
	db *gorm.DB
}

// NewGormProductAttachmentRepository creates a new GormProductAttachmentRepository
func NewGormProductAttachmentRepository(db *gorm.DB) *GormProductAttachmentRepository {
	return &GormProductAttachmentRepository{db: db}
}

// ==================== ProductAttachmentReader Interface ====================

// FindByID finds an attachment by its ID
func (r *GormProductAttachmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*catalog.ProductAttachment, error) {
	var model models.ProductAttachmentModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds an attachment by ID within a tenant
func (r *GormProductAttachmentRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*catalog.ProductAttachment, error) {
	var model models.ProductAttachmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDs finds multiple attachments by their IDs
func (r *GormProductAttachmentRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]catalog.ProductAttachment, error) {
	if len(ids) == 0 {
		return []catalog.ProductAttachment{}, nil
	}

	var attachmentModels []models.ProductAttachmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Order("sort_order ASC").
		Find(&attachmentModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	attachments := make([]catalog.ProductAttachment, len(attachmentModels))
	for i, model := range attachmentModels {
		attachments[i] = *model.ToDomain()
	}
	return attachments, nil
}

// ==================== ProductAttachmentFinder Interface ====================

// FindByProduct finds all attachments for a product
func (r *GormProductAttachmentRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID, filter shared.Filter) ([]catalog.ProductAttachment, error) {
	var attachmentModels []models.ProductAttachmentModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.ProductAttachmentModel{}).
			Where("tenant_id = ? AND product_id = ?", tenantID, productID),
		filter,
	)

	if err := query.Find(&attachmentModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	attachments := make([]catalog.ProductAttachment, len(attachmentModels))
	for i, model := range attachmentModels {
		attachments[i] = *model.ToDomain()
	}
	return attachments, nil
}

// FindByProductAndStatus finds attachments by product and status
func (r *GormProductAttachmentRepository) FindByProductAndStatus(ctx context.Context, tenantID, productID uuid.UUID, status catalog.AttachmentStatus, filter shared.Filter) ([]catalog.ProductAttachment, error) {
	var attachmentModels []models.ProductAttachmentModel
	query := r.applyFilter(
		r.db.WithContext(ctx).Model(&models.ProductAttachmentModel{}).
			Where("tenant_id = ? AND product_id = ? AND status = ?", tenantID, productID, status),
		filter,
	)

	if err := query.Find(&attachmentModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	attachments := make([]catalog.ProductAttachment, len(attachmentModels))
	for i, model := range attachmentModels {
		attachments[i] = *model.ToDomain()
	}
	return attachments, nil
}

// FindActiveByProduct finds all active attachments for a product
func (r *GormProductAttachmentRepository) FindActiveByProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]catalog.ProductAttachment, error) {
	var attachmentModels []models.ProductAttachmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ? AND status = ?", tenantID, productID, catalog.AttachmentStatusActive).
		Order("sort_order ASC, created_at ASC").
		Find(&attachmentModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	attachments := make([]catalog.ProductAttachment, len(attachmentModels))
	for i, model := range attachmentModels {
		attachments[i] = *model.ToDomain()
	}
	return attachments, nil
}

// FindMainImage finds the main image for a product (if any)
func (r *GormProductAttachmentRepository) FindMainImage(ctx context.Context, tenantID, productID uuid.UUID) (*catalog.ProductAttachment, error) {
	var model models.ProductAttachmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ? AND type = ? AND status = ?",
			tenantID, productID, catalog.AttachmentTypeMainImage, catalog.AttachmentStatusActive).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByType finds attachments by type for a product
func (r *GormProductAttachmentRepository) FindByType(ctx context.Context, tenantID, productID uuid.UUID, attachmentType catalog.AttachmentType) ([]catalog.ProductAttachment, error) {
	var attachmentModels []models.ProductAttachmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ? AND type = ? AND status = ?",
			tenantID, productID, attachmentType, catalog.AttachmentStatusActive).
		Order("sort_order ASC, created_at ASC").
		Find(&attachmentModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	attachments := make([]catalog.ProductAttachment, len(attachmentModels))
	for i, model := range attachmentModels {
		attachments[i] = *model.ToDomain()
	}
	return attachments, nil
}

// CountByProduct counts attachments for a product
func (r *GormProductAttachmentRepository) CountByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductAttachmentModel{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountActiveByProduct counts active attachments for a product
func (r *GormProductAttachmentRepository) CountActiveByProduct(ctx context.Context, tenantID, productID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductAttachmentModel{}).
		Where("tenant_id = ? AND product_id = ? AND status = ?", tenantID, productID, catalog.AttachmentStatusActive).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByStorageKey checks if an attachment with the given storage key exists
func (r *GormProductAttachmentRepository) ExistsByStorageKey(ctx context.Context, tenantID uuid.UUID, storageKey string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ProductAttachmentModel{}).
		Where("tenant_id = ? AND storage_key = ?", tenantID, storageKey).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ==================== ProductAttachmentWriter Interface ====================

// Save creates or updates an attachment
func (r *GormProductAttachmentRepository) Save(ctx context.Context, attachment *catalog.ProductAttachment) error {
	model := models.ProductAttachmentModelFromDomain(attachment)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveBatch creates or updates multiple attachments
func (r *GormProductAttachmentRepository) SaveBatch(ctx context.Context, attachments []*catalog.ProductAttachment) error {
	if len(attachments) == 0 {
		return nil
	}
	attachmentModels := make([]*models.ProductAttachmentModel, len(attachments))
	for i, a := range attachments {
		attachmentModels[i] = models.ProductAttachmentModelFromDomain(a)
	}
	return r.db.WithContext(ctx).Save(attachmentModels).Error
}

// Delete permanently deletes an attachment
func (r *GormProductAttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ProductAttachmentModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant permanently deletes an attachment within a tenant
func (r *GormProductAttachmentRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ProductAttachmentModel{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteByProduct permanently deletes all attachments for a product
func (r *GormProductAttachmentRepository) DeleteByProduct(ctx context.Context, tenantID, productID uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ProductAttachmentModel{}, "tenant_id = ? AND product_id = ?", tenantID, productID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ==================== Helper Methods ====================

// applyFilter applies filter options to the query
func (r *GormProductAttachmentRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering with whitelist validation to prevent SQL injection
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, ProductAttachmentSortFields, "")
		if sortField != "" {
			sortOrder := ValidateSortOrder(filter.OrderDir)
			query = query.Order(sortField + " " + sortOrder)
		} else {
			// Default ordering if invalid field
			query = query.Order("sort_order ASC, created_at ASC")
		}
	} else {
		// Default ordering
		query = query.Order("sort_order ASC, created_at ASC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormProductAttachmentRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search (searches file name)
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("file_name ILIKE ?", searchPattern)
	}

	// Apply additional filters
	for key, value := range filter.Filters {
		switch key {
		case "status":
			query = query.Where("status = ?", value)
		case "type":
			query = query.Where("type = ?", value)
		case "content_type":
			query = query.Where("content_type = ?", value)
		case "content_type_prefix":
			// Filter by MIME type prefix (e.g., "image/" to get all images)
			if prefix, ok := value.(string); ok {
				query = query.Where("content_type LIKE ?", prefix+"%")
			}
		case "min_file_size":
			query = query.Where("file_size >= ?", value)
		case "max_file_size":
			query = query.Where("file_size <= ?", value)
		case "uploaded_by":
			if value == nil {
				query = query.Where("uploaded_by IS NULL")
			} else {
				query = query.Where("uploaded_by = ?", value)
			}
		}
	}

	return query
}

// ==================== Additional Utility Methods ====================

// FindPendingOlderThan finds pending attachments older than the specified time
// This is useful for cleanup jobs to remove orphaned pending uploads
func (r *GormProductAttachmentRepository) FindPendingOlderThan(ctx context.Context, tenantID uuid.UUID, olderThan int64) ([]catalog.ProductAttachment, error) {
	var attachmentModels []models.ProductAttachmentModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ? AND created_at < NOW() - INTERVAL '1 second' * ?",
			tenantID, catalog.AttachmentStatusPending, olderThan).
		Order("created_at ASC").
		Find(&attachmentModels).Error; err != nil {
		return nil, err
	}

	// Convert to domain entities
	attachments := make([]catalog.ProductAttachment, len(attachmentModels))
	for i, model := range attachmentModels {
		attachments[i] = *model.ToDomain()
	}
	return attachments, nil
}

// GetMaxSortOrder gets the maximum sort order for a product's attachments
// This is useful when adding new attachments to determine the next sort order
func (r *GormProductAttachmentRepository) GetMaxSortOrder(ctx context.Context, tenantID, productID uuid.UUID) (int, error) {
	var maxOrder *int
	if err := r.db.WithContext(ctx).
		Model(&models.ProductAttachmentModel{}).
		Where("tenant_id = ? AND product_id = ? AND status = ?", tenantID, productID, catalog.AttachmentStatusActive).
		Select("MAX(sort_order)").
		Scan(&maxOrder).Error; err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return -1, nil // No existing attachments
	}
	return *maxOrder, nil
}

// Compile-time interface compliance checks
// GormProductAttachmentRepository implements the full ProductAttachmentRepository interface
// which is composed of ProductAttachmentReader, ProductAttachmentFinder, and ProductAttachmentWriter
var _ catalog.ProductAttachmentRepository = (*GormProductAttachmentRepository)(nil)
var _ catalog.ProductAttachmentReader = (*GormProductAttachmentRepository)(nil)
var _ catalog.ProductAttachmentFinder = (*GormProductAttachmentRepository)(nil)
var _ catalog.ProductAttachmentWriter = (*GormProductAttachmentRepository)(nil)
