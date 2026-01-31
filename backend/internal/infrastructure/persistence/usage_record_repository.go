package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageRecordModel is the GORM model for usage records
type UsageRecordModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID  `gorm:"type:uuid;index;not null"`
	UsageType   string     `gorm:"type:varchar(50);not null"`
	Quantity    int64      `gorm:"not null"`
	Unit        string     `gorm:"type:varchar(20);not null;default:'requests'"`
	RecordedAt  time.Time  `gorm:"not null;default:now()"`
	PeriodStart time.Time  `gorm:"not null"`
	PeriodEnd   time.Time  `gorm:"not null"`
	SourceType  string     `gorm:"type:varchar(100)"`
	SourceID    string     `gorm:"type:varchar(255)"`
	Metadata    []byte     `gorm:"type:jsonb;default:'{}'"`
	UserID      *uuid.UUID `gorm:"type:uuid"`
	IPAddress   string     `gorm:"type:varchar(45)"`
	UserAgent   string     `gorm:"type:text"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
}

// TableName returns the table name for the model
func (UsageRecordModel) TableName() string {
	return "usage_records"
}

// ToEntity converts the model to a domain entity
func (m *UsageRecordModel) ToEntity() *billing.UsageRecord {
	usageType, _ := billing.ParseUsageType(m.UsageType)

	var metadata billing.Metadata
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(billing.Metadata)
	}

	return &billing.UsageRecord{
		BaseEntity: shared.BaseEntity{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		TenantID:    m.TenantID,
		UsageType:   usageType,
		Quantity:    m.Quantity,
		Unit:        billing.UsageUnit(m.Unit),
		RecordedAt:  m.RecordedAt,
		PeriodStart: m.PeriodStart,
		PeriodEnd:   m.PeriodEnd,
		SourceType:  m.SourceType,
		SourceID:    m.SourceID,
		Metadata:    metadata,
		UserID:      m.UserID,
		IPAddress:   m.IPAddress,
		UserAgent:   m.UserAgent,
	}
}

// UsageRecordModelFromEntity creates a model from a domain entity
func UsageRecordModelFromEntity(e *billing.UsageRecord) *UsageRecordModel {
	var metadataBytes []byte
	if e.Metadata != nil {
		metadataBytes, _ = json.Marshal(e.Metadata)
	} else {
		metadataBytes = []byte("{}")
	}

	return &UsageRecordModel{
		ID:          e.ID,
		TenantID:    e.TenantID,
		UsageType:   string(e.UsageType),
		Quantity:    e.Quantity,
		Unit:        string(e.Unit),
		RecordedAt:  e.RecordedAt,
		PeriodStart: e.PeriodStart,
		PeriodEnd:   e.PeriodEnd,
		SourceType:  e.SourceType,
		SourceID:    e.SourceID,
		Metadata:    metadataBytes,
		UserID:      e.UserID,
		IPAddress:   e.IPAddress,
		UserAgent:   e.UserAgent,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// UsageRecordRepository implements the billing.UsageRecordRepository interface
type UsageRecordRepository struct {
	db *gorm.DB
}

// NewUsageRecordRepository creates a new usage record repository
func NewUsageRecordRepository(db *gorm.DB) *UsageRecordRepository {
	return &UsageRecordRepository{db: db}
}

// Save persists a new usage record
func (r *UsageRecordRepository) Save(ctx context.Context, record *billing.UsageRecord) error {
	model := UsageRecordModelFromEntity(record)
	return r.db.WithContext(ctx).Create(model).Error
}

// SaveBatch persists multiple usage records in a single transaction
func (r *UsageRecordRepository) SaveBatch(ctx context.Context, records []*billing.UsageRecord) error {
	if len(records) == 0 {
		return nil
	}

	models := make([]*UsageRecordModel, len(records))
	for i, record := range records {
		models[i] = UsageRecordModelFromEntity(record)
	}

	return r.db.WithContext(ctx).CreateInBatches(models, 100).Error
}

// FindByID retrieves a usage record by its ID
func (r *UsageRecordRepository) FindByID(ctx context.Context, id uuid.UUID) (*billing.UsageRecord, error) {
	var model UsageRecordModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindByTenant retrieves all usage records for a tenant within a time range
func (r *UsageRecordRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageRecordFilter) ([]*billing.UsageRecord, error) {
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	query = r.applyFilter(query, filter)

	var models []UsageRecordModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	records := make([]*billing.UsageRecord, len(models))
	for i, model := range models {
		records[i] = model.ToEntity()
	}
	return records, nil
}

// FindByTenantAndType retrieves usage records for a tenant and specific usage type
func (r *UsageRecordRepository) FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, filter billing.UsageRecordFilter) ([]*billing.UsageRecord, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("usage_type = ?", string(usageType))
	query = r.applyFilter(query, filter)

	var models []UsageRecordModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	records := make([]*billing.UsageRecord, len(models))
	for i, model := range models {
		records[i] = model.ToEntity()
	}
	return records, nil
}

// CountByTenant counts usage records for a tenant within a time range
func (r *UsageRecordRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageRecordFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&UsageRecordModel{}).Where("tenant_id = ?", tenantID)
	query = r.applyFilterForCount(query, filter)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumByTenantAndType calculates total usage for a tenant and type within a time range
func (r *UsageRecordRepository) SumByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, start, end time.Time) (int64, error) {
	var result struct {
		Total int64
	}

	err := r.db.WithContext(ctx).
		Model(&UsageRecordModel{}).
		Select("COALESCE(SUM(quantity), 0) as total").
		Where("tenant_id = ?", tenantID).
		Where("usage_type = ?", string(usageType)).
		Where("recorded_at >= ?", start).
		Where("recorded_at <= ?", end).
		Scan(&result).Error

	if err != nil {
		return 0, err
	}
	return result.Total, nil
}

// GetAggregatedUsage returns aggregated usage data for a tenant
func (r *UsageRecordRepository) GetAggregatedUsage(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, start, end time.Time, groupBy billing.AggregationPeriod) ([]billing.UsageAggregation, error) {
	var dateFormat string
	switch groupBy {
	case billing.AggregationPeriodHour:
		dateFormat = "YYYY-MM-DD HH24:00:00"
	case billing.AggregationPeriodDay:
		dateFormat = "YYYY-MM-DD"
	case billing.AggregationPeriodWeek:
		dateFormat = "IYYY-IW" // ISO week
	case billing.AggregationPeriodMonth:
		dateFormat = "YYYY-MM"
	default:
		dateFormat = "YYYY-MM-DD"
	}

	type aggregationResult struct {
		PeriodKey   string
		TotalUsage  int64
		RecordCount int64
		MinUsage    int64
		MaxUsage    int64
		AvgUsage    float64
	}

	var results []aggregationResult
	err := r.db.WithContext(ctx).
		Model(&UsageRecordModel{}).
		Select(`
			TO_CHAR(recorded_at, ?) as period_key,
			SUM(quantity) as total_usage,
			COUNT(*) as record_count,
			MIN(quantity) as min_usage,
			MAX(quantity) as max_usage,
			AVG(quantity) as avg_usage
		`, dateFormat).
		Where("tenant_id = ?", tenantID).
		Where("usage_type = ?", string(usageType)).
		Where("recorded_at >= ?", start).
		Where("recorded_at <= ?", end).
		Group("period_key").
		Order("period_key ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	aggregations := make([]billing.UsageAggregation, len(results))
	for i, result := range results {
		periodStart, periodEnd := r.parsePeriodKey(result.PeriodKey, groupBy, start.Location())
		aggregations[i] = billing.UsageAggregation{
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			TotalUsage:  result.TotalUsage,
			RecordCount: result.RecordCount,
			MinUsage:    result.MinUsage,
			MaxUsage:    result.MaxUsage,
			AvgUsage:    result.AvgUsage,
		}
	}

	return aggregations, nil
}

// DeleteOlderThan removes usage records older than the specified time
func (r *UsageRecordRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("recorded_at < ?", before).
		Delete(&UsageRecordModel{})
	return result.RowsAffected, result.Error
}

// applyFilter applies filter options to a query
func (r *UsageRecordRepository) applyFilter(query *gorm.DB, filter billing.UsageRecordFilter) *gorm.DB {
	if filter.StartTime != nil {
		query = query.Where("recorded_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("recorded_at <= ?", *filter.EndTime)
	}
	if len(filter.UsageTypes) > 0 {
		types := make([]string, len(filter.UsageTypes))
		for i, t := range filter.UsageTypes {
			types[i] = string(t)
		}
		query = query.Where("usage_type IN ?", types)
	}
	if filter.SourceType != "" {
		query = query.Where("source_type = ?", filter.SourceType)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	// Apply ordering with validation to prevent SQL injection
	orderBy := validateUsageRecordSortField(filter.OrderBy)
	orderDir := validateSortOrder(filter.OrderDir)
	query = query.Order(orderBy + " " + orderDir)

	// Apply pagination
	if filter.PageSize > 0 {
		query = query.Limit(filter.PageSize)
		if filter.Page > 1 {
			query = query.Offset((filter.Page - 1) * filter.PageSize)
		}
	}

	return query
}

// applyFilterForCount applies filter options for count queries (no pagination/ordering)
func (r *UsageRecordRepository) applyFilterForCount(query *gorm.DB, filter billing.UsageRecordFilter) *gorm.DB {
	if filter.StartTime != nil {
		query = query.Where("recorded_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("recorded_at <= ?", *filter.EndTime)
	}
	if len(filter.UsageTypes) > 0 {
		types := make([]string, len(filter.UsageTypes))
		for i, t := range filter.UsageTypes {
			types[i] = string(t)
		}
		query = query.Where("usage_type IN ?", types)
	}
	if filter.SourceType != "" {
		query = query.Where("source_type = ?", filter.SourceType)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	return query
}

// parsePeriodKey parses a period key string into start and end times
func (r *UsageRecordRepository) parsePeriodKey(key string, groupBy billing.AggregationPeriod, loc *time.Location) (time.Time, time.Time) {
	var periodStart, periodEnd time.Time

	switch groupBy {
	case billing.AggregationPeriodHour:
		periodStart, _ = time.ParseInLocation("2006-01-02 15:00:00", key, loc)
		periodEnd = periodStart.Add(time.Hour).Add(-time.Nanosecond)
	case billing.AggregationPeriodDay:
		periodStart, _ = time.ParseInLocation("2006-01-02", key, loc)
		periodEnd = periodStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
	case billing.AggregationPeriodWeek:
		// Parse ISO week format (YYYY-WW)
		var year, week int
		_, _ = time.Parse("2006-01", key) // Just to get the format
		if _, err := time.Parse("2006-01", key); err == nil {
			// Fallback: treat as month
			periodStart, _ = time.ParseInLocation("2006-01", key, loc)
			periodEnd = periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
		} else {
			// Try to parse as ISO week
			n, _ := time.Parse("2006-02", key)
			year = n.Year()
			week = int(n.Month())
			periodStart = isoWeekStart(year, week, loc)
			periodEnd = periodStart.AddDate(0, 0, 7).Add(-time.Nanosecond)
		}
	case billing.AggregationPeriodMonth:
		periodStart, _ = time.ParseInLocation("2006-01", key, loc)
		periodEnd = periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
	default:
		periodStart, _ = time.ParseInLocation("2006-01-02", key, loc)
		periodEnd = periodStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}

	return periodStart, periodEnd
}

// isoWeekStart returns the start of an ISO week
func isoWeekStart(year, week int, loc *time.Location) time.Time {
	// Find January 4th of the year (always in week 1)
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, loc)
	// Find the Monday of that week
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, -(weekday - 1))
	// Add weeks
	return monday.AddDate(0, 0, (week-1)*7)
}

// usageRecordAllowedSortFields defines the allowed sort fields for usage records
var usageRecordAllowedSortFields = map[string]bool{
	"id":          true,
	"recorded_at": true,
	"created_at":  true,
	"updated_at":  true,
	"usage_type":  true,
	"quantity":    true,
	"source_type": true,
}

// validateUsageRecordSortField validates and returns a safe sort field
func validateUsageRecordSortField(field string) string {
	if field == "" {
		return "recorded_at"
	}
	if usageRecordAllowedSortFields[field] {
		return field
	}
	return "recorded_at"
}

// validateSortOrder validates and returns a safe sort order
func validateSortOrder(order string) string {
	switch order {
	case "asc", "ASC":
		return "asc"
	case "desc", "DESC":
		return "desc"
	default:
		return "desc"
	}
}

// Ensure UsageRecordRepository implements the interface
var _ billing.UsageRecordRepository = (*UsageRecordRepository)(nil)
