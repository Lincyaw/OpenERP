package persistence

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	infraBilling "github.com/erp/backend/internal/infrastructure/billing"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageReportLogModel is the GORM model for usage report logs
type UsageReportLogModel struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID           uuid.UUID `gorm:"type:uuid;index;not null"`
	SubscriptionItemID string    `gorm:"type:varchar(255);not null"`
	UsageType          string    `gorm:"type:varchar(50);not null"`
	Quantity           int64     `gorm:"not null"`
	Timestamp          time.Time `gorm:"not null"`
	StripeRecordID     string    `gorm:"type:varchar(255)"`
	Status             string    `gorm:"type:varchar(20);not null;index"`
	ErrorMessage       string    `gorm:"type:text"`
	RetryCount         int       `gorm:"default:0"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for the model
func (UsageReportLogModel) TableName() string {
	return "usage_report_logs"
}

// ToEntity converts the model to a domain entity
func (m *UsageReportLogModel) ToEntity() *infraBilling.UsageReportLog {
	usageType, _ := billing.ParseUsageType(m.UsageType)
	return &infraBilling.UsageReportLog{
		ID:                 m.ID,
		TenantID:           m.TenantID,
		SubscriptionItemID: m.SubscriptionItemID,
		UsageType:          usageType,
		Quantity:           m.Quantity,
		Timestamp:          m.Timestamp,
		StripeRecordID:     m.StripeRecordID,
		Status:             infraBilling.UsageReportStatus(m.Status),
		ErrorMessage:       m.ErrorMessage,
		RetryCount:         m.RetryCount,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

// FromEntity creates a model from a domain entity
func UsageReportLogModelFromEntity(e *infraBilling.UsageReportLog) *UsageReportLogModel {
	return &UsageReportLogModel{
		ID:                 e.ID,
		TenantID:           e.TenantID,
		SubscriptionItemID: e.SubscriptionItemID,
		UsageType:          e.UsageType.String(),
		Quantity:           e.Quantity,
		Timestamp:          e.Timestamp,
		StripeRecordID:     e.StripeRecordID,
		Status:             e.Status.String(),
		ErrorMessage:       e.ErrorMessage,
		RetryCount:         e.RetryCount,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

// UsageReportLogRepository implements the UsageReportLogRepository interface
type UsageReportLogRepository struct {
	db *gorm.DB
}

// NewUsageReportLogRepository creates a new usage report log repository
func NewUsageReportLogRepository(db *gorm.DB) *UsageReportLogRepository {
	return &UsageReportLogRepository{db: db}
}

// Save persists a usage report log
func (r *UsageReportLogRepository) Save(ctx context.Context, log *infraBilling.UsageReportLog) error {
	model := UsageReportLogModelFromEntity(log)
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing usage report log
func (r *UsageReportLogRepository) Update(ctx context.Context, log *infraBilling.UsageReportLog) error {
	model := UsageReportLogModelFromEntity(log)
	return r.db.WithContext(ctx).Save(model).Error
}

// FindByID retrieves a usage report log by ID
func (r *UsageReportLogRepository) FindByID(ctx context.Context, id uuid.UUID) (*infraBilling.UsageReportLog, error) {
	var model UsageReportLogModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindPending retrieves all pending usage reports for retry
func (r *UsageReportLogRepository) FindPending(ctx context.Context, maxRetries int) ([]*infraBilling.UsageReportLog, error) {
	var models []UsageReportLogModel
	err := r.db.WithContext(ctx).
		Where("status IN (?, ?) AND retry_count < ?",
			infraBilling.UsageReportStatusPending,
			infraBilling.UsageReportStatusRetrying,
			maxRetries).
		Order("created_at ASC").
		Limit(100).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	logs := make([]*infraBilling.UsageReportLog, len(models))
	for i, model := range models {
		logs[i] = model.ToEntity()
	}
	return logs, nil
}

// FindByTenant retrieves usage report logs for a tenant
func (r *UsageReportLogRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]*infraBilling.UsageReportLog, error) {
	var models []UsageReportLogModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantID, start, end).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	logs := make([]*infraBilling.UsageReportLog, len(models))
	for i, model := range models {
		logs[i] = model.ToEntity()
	}
	return logs, nil
}

// FindByStatus retrieves usage report logs by status
func (r *UsageReportLogRepository) FindByStatus(ctx context.Context, status infraBilling.UsageReportStatus, limit int) ([]*infraBilling.UsageReportLog, error) {
	var models []UsageReportLogModel
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	logs := make([]*infraBilling.UsageReportLog, len(models))
	for i, model := range models {
		logs[i] = model.ToEntity()
	}
	return logs, nil
}

// MarkAsSuccess marks a usage report as successful
func (r *UsageReportLogRepository) MarkAsSuccess(ctx context.Context, id uuid.UUID, stripeRecordID string) error {
	return r.db.WithContext(ctx).
		Model(&UsageReportLogModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":           infraBilling.UsageReportStatusSuccess,
			"stripe_record_id": stripeRecordID,
			"error_message":    "",
			"updated_at":       time.Now(),
		}).Error
}

// MarkAsFailed marks a usage report as failed
func (r *UsageReportLogRepository) MarkAsFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	return r.db.WithContext(ctx).
		Model(&UsageReportLogModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        infraBilling.UsageReportStatusFailed,
			"error_message": errorMessage,
			"updated_at":    time.Now(),
		}).Error
}

// IncrementRetryCount increments the retry count for a usage report
func (r *UsageReportLogRepository) IncrementRetryCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&UsageReportLogModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"retry_count": gorm.Expr("retry_count + 1"),
			"status":      infraBilling.UsageReportStatusRetrying,
			"updated_at":  time.Now(),
		}).Error
}

// GetReportingSummary returns a summary of usage reporting for a tenant
func (r *UsageReportLogRepository) GetReportingSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*UsageReportingSummary, error) {
	var summary UsageReportingSummary
	summary.TenantID = tenantID
	summary.PeriodStart = start
	summary.PeriodEnd = end

	// Count by status
	type statusCount struct {
		Status string
		Count  int64
	}
	var counts []statusCount

	err := r.db.WithContext(ctx).
		Model(&UsageReportLogModel{}).
		Select("status, count(*) as count").
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantID, start, end).
		Group("status").
		Scan(&counts).Error
	if err != nil {
		return nil, err
	}

	for _, c := range counts {
		switch infraBilling.UsageReportStatus(c.Status) {
		case infraBilling.UsageReportStatusSuccess:
			summary.SuccessCount = c.Count
		case infraBilling.UsageReportStatusFailed:
			summary.FailedCount = c.Count
		case infraBilling.UsageReportStatusPending, infraBilling.UsageReportStatusRetrying:
			summary.PendingCount += c.Count
		case infraBilling.UsageReportStatusAbandoned:
			summary.AbandonedCount = c.Count
		}
		summary.TotalCount += c.Count
	}

	// Get total quantity reported
	var totalQuantity int64
	err = r.db.WithContext(ctx).
		Model(&UsageReportLogModel{}).
		Select("COALESCE(SUM(quantity), 0)").
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ? AND status = ?",
			tenantID, start, end, infraBilling.UsageReportStatusSuccess).
		Scan(&totalQuantity).Error
	if err != nil {
		return nil, err
	}
	summary.TotalQuantityReported = totalQuantity

	return &summary, nil
}

// UsageReportingSummary contains summary statistics for usage reporting
type UsageReportingSummary struct {
	TenantID              uuid.UUID
	PeriodStart           time.Time
	PeriodEnd             time.Time
	TotalCount            int64
	SuccessCount          int64
	FailedCount           int64
	PendingCount          int64
	AbandonedCount        int64
	TotalQuantityReported int64
}

// DeleteOldLogs deletes usage report logs older than the specified time
func (r *UsageReportLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ? AND status = ?", before, infraBilling.UsageReportStatusSuccess).
		Delete(&UsageReportLogModel{})
	return result.RowsAffected, result.Error
}

// Ensure UsageReportLogRepository implements the interface
var _ infraBilling.UsageReportLogRepository = (*UsageReportLogRepository)(nil)
