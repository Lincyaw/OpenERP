package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MaxFindByTagsLimit is the maximum number of tags allowed in FindByTags query
// to prevent excessive query complexity and potential denial-of-service attacks
const MaxFindByTagsLimit = 50

// FeatureFlagSortFields contains allowed sort fields for feature flags
var FeatureFlagSortFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"key":        true,
	"name":       true,
	"type":       true,
	"status":     true,
}

// FlagOverrideSortFields contains allowed sort fields for flag overrides
var FlagOverrideSortFields = map[string]bool{
	"id":          true,
	"created_at":  true,
	"updated_at":  true,
	"flag_key":    true,
	"target_type": true,
	"expires_at":  true,
}

// FlagAuditLogSortFields contains allowed sort fields for flag audit logs
var FlagAuditLogSortFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"flag_key":   true,
	"action":     true,
}

// GormFeatureFlagRepository implements FeatureFlagRepository using GORM
type GormFeatureFlagRepository struct {
	db *gorm.DB
}

// NewGormFeatureFlagRepository creates a new GormFeatureFlagRepository
func NewGormFeatureFlagRepository(db *gorm.DB) *GormFeatureFlagRepository {
	return &GormFeatureFlagRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *GormFeatureFlagRepository) WithTx(tx *gorm.DB) *GormFeatureFlagRepository {
	return &GormFeatureFlagRepository{db: tx}
}

// Create creates a new feature flag
func (r *GormFeatureFlagRepository) Create(ctx context.Context, flag *featureflag.FeatureFlag) error {
	model := models.FeatureFlagModelFromDomain(flag)
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoNothing: true,
		}).
		Create(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.NewDomainError("FLAG_KEY_EXISTS", "Feature flag with this key already exists")
	}
	return nil
}

// Update updates an existing feature flag with optimistic locking
func (r *GormFeatureFlagRepository) Update(ctx context.Context, flag *featureflag.FeatureFlag) error {
	// Check version matches before incrementing
	currentVersion := flag.Version

	// Increment version
	flag.IncrementVersion()

	model := models.FeatureFlagModelFromDomain(flag)

	// Use optimistic locking: only update if version matches
	result := r.db.WithContext(ctx).
		Model(&models.FeatureFlagModel{}).
		Where("id = ? AND version = ?", flag.ID, currentVersion).
		Updates(map[string]any{
			"name":          model.Name,
			"description":   model.Description,
			"status":        model.Status,
			"default_value": model.DefaultValueJSON,
			"rules":         model.RulesJSON,
			"tags":          model.TagsJSON,
			"updated_by":    model.UpdatedBy,
			"version":       model.Version,
			"updated_at":    model.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Check if flag exists
		var count int64
		r.db.WithContext(ctx).Model(&models.FeatureFlagModel{}).Where("id = ?", flag.ID).Count(&count)
		if count == 0 {
			return shared.ErrNotFound
		}
		return shared.NewDomainError("OPTIMISTIC_LOCK_FAILED", "Feature flag was modified by another transaction")
	}
	return nil
}

// FindByKey finds a feature flag by its unique key
func (r *GormFeatureFlagRepository) FindByKey(ctx context.Context, key string) (*featureflag.FeatureFlag, error) {
	var model models.FeatureFlagModel
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByID finds a feature flag by its ID
func (r *GormFeatureFlagRepository) FindByID(ctx context.Context, id uuid.UUID) (*featureflag.FeatureFlag, error) {
	var model models.FeatureFlagModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAll retrieves all feature flags with optional filtering
func (r *GormFeatureFlagRepository) FindAll(ctx context.Context, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	var flagModels []models.FeatureFlagModel

	query := r.db.WithContext(ctx).Model(&models.FeatureFlagModel{})
	query = r.applyFilter(query, filter)

	if err := query.Find(&flagModels).Error; err != nil {
		return nil, err
	}

	flags := make([]featureflag.FeatureFlag, len(flagModels))
	for i, model := range flagModels {
		flags[i] = *model.ToDomain()
	}
	return flags, nil
}

// FindByStatus finds all feature flags with a specific status
func (r *GormFeatureFlagRepository) FindByStatus(ctx context.Context, status featureflag.FlagStatus, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	var flagModels []models.FeatureFlagModel

	query := r.db.WithContext(ctx).Model(&models.FeatureFlagModel{}).
		Where("status = ?", status)
	query = r.applyFilter(query, filter)

	if err := query.Find(&flagModels).Error; err != nil {
		return nil, err
	}

	flags := make([]featureflag.FeatureFlag, len(flagModels))
	for i, model := range flagModels {
		flags[i] = *model.ToDomain()
	}
	return flags, nil
}

// FindByTags finds all feature flags that have any of the specified tags
func (r *GormFeatureFlagRepository) FindByTags(ctx context.Context, tags []string, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	if len(tags) == 0 {
		return []featureflag.FeatureFlag{}, nil
	}

	// Validate tags array size to prevent excessive query complexity
	if len(tags) > MaxFindByTagsLimit {
		return nil, shared.NewDomainError("TOO_MANY_TAGS",
			"FindByTags is limited to a maximum of 50 tags")
	}

	var flagModels []models.FeatureFlagModel

	// Use PostgreSQL JSONB containment operator to check if tags array contains any of the specified tags
	// The ?| operator checks if the array contains any of the specified elements
	query := r.db.WithContext(ctx).Model(&models.FeatureFlagModel{}).
		Where("tags ?| ?", tags)
	query = r.applyFilter(query, filter)

	if err := query.Find(&flagModels).Error; err != nil {
		return nil, err
	}

	flags := make([]featureflag.FeatureFlag, len(flagModels))
	for i, model := range flagModels {
		flags[i] = *model.ToDomain()
	}
	return flags, nil
}

// FindByType finds all feature flags of a specific type
func (r *GormFeatureFlagRepository) FindByType(ctx context.Context, flagType featureflag.FlagType, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	var flagModels []models.FeatureFlagModel

	query := r.db.WithContext(ctx).Model(&models.FeatureFlagModel{}).
		Where("type = ?", flagType)
	query = r.applyFilter(query, filter)

	if err := query.Find(&flagModels).Error; err != nil {
		return nil, err
	}

	flags := make([]featureflag.FeatureFlag, len(flagModels))
	for i, model := range flagModels {
		flags[i] = *model.ToDomain()
	}
	return flags, nil
}

// FindEnabled finds all enabled feature flags
func (r *GormFeatureFlagRepository) FindEnabled(ctx context.Context, filter shared.Filter) ([]featureflag.FeatureFlag, error) {
	return r.FindByStatus(ctx, featureflag.FlagStatusEnabled, filter)
}

// Delete deletes a feature flag by its key
func (r *GormFeatureFlagRepository) Delete(ctx context.Context, key string) error {
	result := r.db.WithContext(ctx).Delete(&models.FeatureFlagModel{}, "key = ?", key)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// ExistsByKey checks if a feature flag with the given key exists
func (r *GormFeatureFlagRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.FeatureFlagModel{}).Where("key = ?", key).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count counts feature flags matching the filter
func (r *GormFeatureFlagRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.FeatureFlagModel{})
	query = r.applyFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts feature flags by status
func (r *GormFeatureFlagRepository) CountByStatus(ctx context.Context, status featureflag.FlagStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.FeatureFlagModel{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyFilter applies filter options to the query
func (r *GormFeatureFlagRepository) applyFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, FeatureFlagSortFields, "")
		if sortField != "" {
			sortOrder := ValidateSortOrder(filter.OrderDir)
			query = query.Order(sortField + " " + sortOrder)
		} else {
			query = query.Order("created_at DESC")
		}
	} else {
		query = query.Order("created_at DESC")
	}

	return query
}

// applyFilterWithoutPagination applies filter options without pagination
func (r *GormFeatureFlagRepository) applyFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	// Apply search
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("key ILIKE ? OR name ILIKE ? OR description ILIKE ?", search, search, search)
	}

	// Apply custom filters
	for key, value := range filter.Filters {
		switch key {
		case "status":
			query = query.Where("status = ?", value)
		case "type":
			query = query.Where("type = ?", value)
		case "enabled":
			if value == true {
				query = query.Where("status = ?", featureflag.FlagStatusEnabled)
			}
		case "archived":
			if value == true {
				query = query.Where("status = ?", featureflag.FlagStatusArchived)
			} else {
				query = query.Where("status != ?", featureflag.FlagStatusArchived)
			}
		}
	}

	return query
}

// Ensure GormFeatureFlagRepository implements FeatureFlagRepository
var _ featureflag.FeatureFlagRepository = (*GormFeatureFlagRepository)(nil)

// ======================================================================
// GormFlagOverrideRepository
// ======================================================================

// GormFlagOverrideRepository implements FlagOverrideRepository using GORM
type GormFlagOverrideRepository struct {
	db *gorm.DB
}

// NewGormFlagOverrideRepository creates a new GormFlagOverrideRepository
func NewGormFlagOverrideRepository(db *gorm.DB) *GormFlagOverrideRepository {
	return &GormFlagOverrideRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *GormFlagOverrideRepository) WithTx(tx *gorm.DB) *GormFlagOverrideRepository {
	return &GormFlagOverrideRepository{db: tx}
}

// Create creates a new flag override
func (r *GormFlagOverrideRepository) Create(ctx context.Context, override *featureflag.FlagOverride) error {
	model := models.FlagOverrideModelFromDomain(override)
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "flag_key"}, {Name: "target_type"}, {Name: "target_id"}},
			DoNothing: true,
		}).
		Create(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.NewDomainError("OVERRIDE_EXISTS", "Override for this flag/target combination already exists")
	}
	return nil
}

// Update updates an existing flag override
func (r *GormFlagOverrideRepository) Update(ctx context.Context, override *featureflag.FlagOverride) error {
	model := models.FlagOverrideModelFromDomain(override)
	result := r.db.WithContext(ctx).
		Model(&models.FlagOverrideModel{}).
		Where("id = ?", override.ID).
		Updates(map[string]any{
			"value":      model.ValueJSON,
			"reason":     model.Reason,
			"expires_at": model.ExpiresAt,
			"updated_at": model.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// FindByID finds a flag override by its ID
func (r *GormFlagOverrideRepository) FindByID(ctx context.Context, id uuid.UUID) (*featureflag.FlagOverride, error) {
	var model models.FlagOverrideModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByFlagKey finds all overrides for a specific flag
func (r *GormFlagOverrideRepository) FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	var overrideModels []models.FlagOverrideModel

	query := r.db.WithContext(ctx).Model(&models.FlagOverrideModel{}).
		Where("flag_key = ?", flagKey)
	query = r.applyOverrideFilter(query, filter)

	if err := query.Find(&overrideModels).Error; err != nil {
		return nil, err
	}

	overrides := make([]featureflag.FlagOverride, len(overrideModels))
	for i, model := range overrideModels {
		overrides[i] = *model.ToDomain()
	}
	return overrides, nil
}

// FindByTarget finds all overrides for a specific target (user or tenant)
func (r *GormFlagOverrideRepository) FindByTarget(ctx context.Context, targetType featureflag.OverrideTargetType, targetID uuid.UUID, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	var overrideModels []models.FlagOverrideModel

	query := r.db.WithContext(ctx).Model(&models.FlagOverrideModel{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID)
	query = r.applyOverrideFilter(query, filter)

	if err := query.Find(&overrideModels).Error; err != nil {
		return nil, err
	}

	overrides := make([]featureflag.FlagOverride, len(overrideModels))
	for i, model := range overrideModels {
		overrides[i] = *model.ToDomain()
	}
	return overrides, nil
}

// FindForEvaluation finds the most specific override for flag evaluation.
// Priority: user override > tenant override
// Only returns active (non-expired) overrides.
func (r *GormFlagOverrideRepository) FindForEvaluation(ctx context.Context, flagKey string, tenantID, userID *uuid.UUID) (*featureflag.FlagOverride, error) {
	now := time.Now()

	// First, try to find user-level override (highest priority)
	if userID != nil {
		var userOverride models.FlagOverrideModel
		err := r.db.WithContext(ctx).
			Where("flag_key = ? AND target_type = ? AND target_id = ?", flagKey, featureflag.OverrideTargetTypeUser, *userID).
			Where("expires_at IS NULL OR expires_at > ?", now).
			First(&userOverride).Error

		if err == nil {
			return userOverride.ToDomain(), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Then, try to find tenant-level override
	if tenantID != nil {
		var tenantOverride models.FlagOverrideModel
		err := r.db.WithContext(ctx).
			Where("flag_key = ? AND target_type = ? AND target_id = ?", flagKey, featureflag.OverrideTargetTypeTenant, *tenantID).
			Where("expires_at IS NULL OR expires_at > ?", now).
			First(&tenantOverride).Error

		if err == nil {
			return tenantOverride.ToDomain(), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// No override found
	return nil, nil
}

// FindByFlagKeyAndTarget finds a specific override by flag key and target
func (r *GormFlagOverrideRepository) FindByFlagKeyAndTarget(ctx context.Context, flagKey string, targetType featureflag.OverrideTargetType, targetID uuid.UUID) (*featureflag.FlagOverride, error) {
	var model models.FlagOverrideModel
	if err := r.db.WithContext(ctx).
		Where("flag_key = ? AND target_type = ? AND target_id = ?", flagKey, targetType, targetID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindExpired finds all expired overrides
func (r *GormFlagOverrideRepository) FindExpired(ctx context.Context, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	var overrideModels []models.FlagOverrideModel

	query := r.db.WithContext(ctx).Model(&models.FlagOverrideModel{}).
		Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now())
	query = r.applyOverrideFilter(query, filter)

	if err := query.Find(&overrideModels).Error; err != nil {
		return nil, err
	}

	overrides := make([]featureflag.FlagOverride, len(overrideModels))
	for i, model := range overrideModels {
		overrides[i] = *model.ToDomain()
	}
	return overrides, nil
}

// FindActive finds all active (non-expired) overrides
func (r *GormFlagOverrideRepository) FindActive(ctx context.Context, filter shared.Filter) ([]featureflag.FlagOverride, error) {
	var overrideModels []models.FlagOverrideModel

	query := r.db.WithContext(ctx).Model(&models.FlagOverrideModel{}).
		Where("expires_at IS NULL OR expires_at > ?", time.Now())
	query = r.applyOverrideFilter(query, filter)

	if err := query.Find(&overrideModels).Error; err != nil {
		return nil, err
	}

	overrides := make([]featureflag.FlagOverride, len(overrideModels))
	for i, model := range overrideModels {
		overrides[i] = *model.ToDomain()
	}
	return overrides, nil
}

// Delete deletes a flag override by its ID
func (r *GormFlagOverrideRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.FlagOverrideModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteByFlagKey deletes all overrides for a specific flag
func (r *GormFlagOverrideRepository) DeleteByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	result := r.db.WithContext(ctx).Delete(&models.FlagOverrideModel{}, "flag_key = ?", flagKey)
	return result.RowsAffected, result.Error
}

// DeleteExpired deletes all expired overrides and returns the count of deleted records
func (r *GormFlagOverrideRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Delete(&models.FlagOverrideModel{}, "expires_at IS NOT NULL AND expires_at <= ?", time.Now())
	return result.RowsAffected, result.Error
}

// Count counts overrides matching the filter
func (r *GormFlagOverrideRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.FlagOverrideModel{})
	query = r.applyOverrideFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByFlagKey counts overrides for a specific flag
func (r *GormFlagOverrideRepository) CountByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.FlagOverrideModel{}).Where("flag_key = ?", flagKey).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyOverrideFilter applies filter options to the query
func (r *GormFlagOverrideRepository) applyOverrideFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyOverrideFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, FlagOverrideSortFields, "")
		if sortField != "" {
			sortOrder := ValidateSortOrder(filter.OrderDir)
			query = query.Order(sortField + " " + sortOrder)
		} else {
			query = query.Order("created_at DESC")
		}
	} else {
		query = query.Order("created_at DESC")
	}

	return query
}

// applyOverrideFilterWithoutPagination applies filter options without pagination
func (r *GormFlagOverrideRepository) applyOverrideFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	for key, value := range filter.Filters {
		switch key {
		case "flag_key":
			query = query.Where("flag_key = ?", value)
		case "target_type":
			query = query.Where("target_type = ?", value)
		case "target_id":
			query = query.Where("target_id = ?", value)
		case "expired":
			if value == true {
				query = query.Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now())
			}
		case "active":
			if value == true {
				query = query.Where("expires_at IS NULL OR expires_at > ?", time.Now())
			}
		}
	}

	return query
}

// Ensure GormFlagOverrideRepository implements FlagOverrideRepository
var _ featureflag.FlagOverrideRepository = (*GormFlagOverrideRepository)(nil)

// ======================================================================
// GormFlagAuditLogRepository
// ======================================================================

// GormFlagAuditLogRepository implements FlagAuditLogRepository using GORM
type GormFlagAuditLogRepository struct {
	db *gorm.DB
}

// NewGormFlagAuditLogRepository creates a new GormFlagAuditLogRepository
func NewGormFlagAuditLogRepository(db *gorm.DB) *GormFlagAuditLogRepository {
	return &GormFlagAuditLogRepository{db: db}
}

// WithTx returns a new repository instance with the given transaction
func (r *GormFlagAuditLogRepository) WithTx(tx *gorm.DB) *GormFlagAuditLogRepository {
	return &GormFlagAuditLogRepository{db: tx}
}

// Create creates a new audit log entry
func (r *GormFlagAuditLogRepository) Create(ctx context.Context, log *featureflag.FlagAuditLog) error {
	model := models.FlagAuditLogModelFromDomain(log)
	return r.db.WithContext(ctx).Create(model).Error
}

// CreateBatch creates multiple audit log entries in a single transaction
func (r *GormFlagAuditLogRepository) CreateBatch(ctx context.Context, logs []*featureflag.FlagAuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	logModels := make([]*models.FlagAuditLogModel, len(logs))
	for i, log := range logs {
		logModels[i] = models.FlagAuditLogModelFromDomain(log)
	}

	return r.db.WithContext(ctx).Create(&logModels).Error
}

// FindByFlagKey finds audit logs for a specific flag with pagination
func (r *GormFlagAuditLogRepository) FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	var logModels []models.FlagAuditLogModel

	query := r.db.WithContext(ctx).Model(&models.FlagAuditLogModel{}).
		Where("flag_key = ?", flagKey)
	query = r.applyAuditLogFilter(query, filter)

	if err := query.Find(&logModels).Error; err != nil {
		return nil, err
	}

	logs := make([]featureflag.FlagAuditLog, len(logModels))
	for i, model := range logModels {
		logs[i] = *model.ToDomain()
	}
	return logs, nil
}

// FindByUserID finds audit logs for actions performed by a specific user
func (r *GormFlagAuditLogRepository) FindByUserID(ctx context.Context, userID uuid.UUID, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	var logModels []models.FlagAuditLogModel

	query := r.db.WithContext(ctx).Model(&models.FlagAuditLogModel{}).
		Where("user_id = ?", userID)
	query = r.applyAuditLogFilter(query, filter)

	if err := query.Find(&logModels).Error; err != nil {
		return nil, err
	}

	logs := make([]featureflag.FlagAuditLog, len(logModels))
	for i, model := range logModels {
		logs[i] = *model.ToDomain()
	}
	return logs, nil
}

// FindByAction finds audit logs for a specific action type
func (r *GormFlagAuditLogRepository) FindByAction(ctx context.Context, action featureflag.AuditAction, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	var logModels []models.FlagAuditLogModel

	query := r.db.WithContext(ctx).Model(&models.FlagAuditLogModel{}).
		Where("action = ?", action)
	query = r.applyAuditLogFilter(query, filter)

	if err := query.Find(&logModels).Error; err != nil {
		return nil, err
	}

	logs := make([]featureflag.FlagAuditLog, len(logModels))
	for i, model := range logModels {
		logs[i] = *model.ToDomain()
	}
	return logs, nil
}

// FindAll finds all audit logs with pagination
func (r *GormFlagAuditLogRepository) FindAll(ctx context.Context, filter shared.Filter) ([]featureflag.FlagAuditLog, error) {
	var logModels []models.FlagAuditLogModel

	query := r.db.WithContext(ctx).Model(&models.FlagAuditLogModel{})
	query = r.applyAuditLogFilter(query, filter)

	if err := query.Find(&logModels).Error; err != nil {
		return nil, err
	}

	logs := make([]featureflag.FlagAuditLog, len(logModels))
	for i, model := range logModels {
		logs[i] = *model.ToDomain()
	}
	return logs, nil
}

// Count counts audit logs matching the filter
func (r *GormFlagAuditLogRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.FlagAuditLogModel{})
	query = r.applyAuditLogFilterWithoutPagination(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByFlagKey counts audit logs for a specific flag
func (r *GormFlagAuditLogRepository) CountByFlagKey(ctx context.Context, flagKey string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.FlagAuditLogModel{}).Where("flag_key = ?", flagKey).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyAuditLogFilter applies filter options to the query
func (r *GormFlagAuditLogRepository) applyAuditLogFilter(query *gorm.DB, filter shared.Filter) *gorm.DB {
	query = r.applyAuditLogFilterWithoutPagination(query, filter)

	// Apply pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Apply ordering (audit logs default to newest first)
	if filter.OrderBy != "" {
		sortField := ValidateSortField(filter.OrderBy, FlagAuditLogSortFields, "")
		if sortField != "" {
			sortOrder := ValidateSortOrder(filter.OrderDir)
			query = query.Order(sortField + " " + sortOrder)
		} else {
			query = query.Order("created_at DESC")
		}
	} else {
		query = query.Order("created_at DESC")
	}

	return query
}

// applyAuditLogFilterWithoutPagination applies filter options without pagination
func (r *GormFlagAuditLogRepository) applyAuditLogFilterWithoutPagination(query *gorm.DB, filter shared.Filter) *gorm.DB {
	for key, value := range filter.Filters {
		switch key {
		case "flag_key":
			query = query.Where("flag_key = ?", value)
		case "user_id":
			query = query.Where("user_id = ?", value)
		case "action":
			query = query.Where("action = ?", value)
		}
	}

	return query
}

// Ensure GormFlagAuditLogRepository implements FlagAuditLogRepository
var _ featureflag.FlagAuditLogRepository = (*GormFlagAuditLogRepository)(nil)
