package persistence

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLogRepository implements the identity.AuditLogRepository interface
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new AuditLogRepository
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(ctx context.Context, log *identity.AuditLog) error {
	model, err := models.AdminAuditLogModelFromDomain(log)
	if err != nil {
		return fmt.Errorf("failed to convert audit log to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// FindByID finds an audit log by ID
func (r *AuditLogRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.AuditLog, error) {
	var model models.AdminAuditLogModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find audit log: %w", err)
	}

	return model.ToDomain(), nil
}

// FindAll returns audit logs with filtering and pagination
func (r *AuditLogRepository) FindAll(ctx context.Context, filter identity.AuditLogFilter) ([]*identity.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AdminAuditLogModel{})
	query = r.applyFilters(query, filter)

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Apply sorting and pagination
	query = r.applySorting(query, filter)
	query = query.Offset(filter.Offset()).Limit(filter.Limit())

	var modelList []models.AdminAuditLogModel
	if err := query.Find(&modelList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find audit logs: %w", err)
	}

	logs := make([]*identity.AuditLog, len(modelList))
	for i, m := range modelList {
		logs[i] = m.ToDomain()
	}

	return logs, total, nil
}

// FindByAdminUserID returns all audit logs for a specific admin user
func (r *AuditLogRepository) FindByAdminUserID(ctx context.Context, adminUserID uuid.UUID, filter identity.AuditLogFilter) ([]*identity.AuditLog, int64, error) {
	filter.AdminUserID = &adminUserID
	return r.FindAll(ctx, filter)
}

// FindByTargetID returns all audit logs for a specific target entity
func (r *AuditLogRepository) FindByTargetID(ctx context.Context, targetType identity.AuditTargetType, targetID uuid.UUID, filter identity.AuditLogFilter) ([]*identity.AuditLog, int64, error) {
	filter.TargetType = &targetType
	filter.TargetID = &targetID
	return r.FindAll(ctx, filter)
}

// Count returns the total number of audit logs matching the filter
func (r *AuditLogRepository) Count(ctx context.Context, filter identity.AuditLogFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AdminAuditLogModel{})
	query = r.applyFilters(query, filter)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// applyFilters applies the filter conditions to the query
func (r *AuditLogRepository) applyFilters(query *gorm.DB, filter identity.AuditLogFilter) *gorm.DB {
	if filter.AdminUserID != nil {
		query = query.Where("admin_user_id = ?", *filter.AdminUserID)
	}

	if filter.Action != nil {
		query = query.Where("action = ?", *filter.Action)
	}

	if len(filter.Actions) > 0 {
		query = query.Where("action IN ?", filter.Actions)
	}

	if filter.TargetType != nil {
		query = query.Where("target_type = ?", *filter.TargetType)
	}

	if filter.TargetID != nil {
		query = query.Where("target_id = ?", *filter.TargetID)
	}

	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}

	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	if filter.IPAddress != "" {
		query = query.Where("ip_address = ?", filter.IPAddress)
	}

	return query
}

// applySorting applies sorting to the query
func (r *AuditLogRepository) applySorting(query *gorm.DB, filter identity.AuditLogFilter) *gorm.DB {
	sortBy := filter.GetSortBy()
	sortOrder := filter.GetSortOrder()

	// Map sort field to column name
	columnMap := map[string]string{
		"created_at":    "created_at",
		"action":        "action",
		"target_type":   "target_type",
		"admin_user_id": "admin_user_id",
	}

	column, ok := columnMap[sortBy]
	if !ok {
		column = "created_at"
	}

	return query.Order(fmt.Sprintf("%s %s", column, sortOrder))
}

// Ensure AuditLogRepository implements identity.AuditLogRepository
var _ identity.AuditLogRepository = (*AuditLogRepository)(nil)
