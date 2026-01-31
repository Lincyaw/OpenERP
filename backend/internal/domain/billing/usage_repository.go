package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UsageRecordRepository defines the interface for persisting and querying usage records
type UsageRecordRepository interface {
	// Save persists a new usage record
	Save(ctx context.Context, record *UsageRecord) error

	// SaveBatch persists multiple usage records in a single transaction
	SaveBatch(ctx context.Context, records []*UsageRecord) error

	// FindByID retrieves a usage record by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*UsageRecord, error)

	// FindByTenant retrieves all usage records for a tenant within a time range
	FindByTenant(ctx context.Context, tenantID uuid.UUID, filter UsageRecordFilter) ([]*UsageRecord, error)

	// FindByTenantAndType retrieves usage records for a tenant and specific usage type
	FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType UsageType, filter UsageRecordFilter) ([]*UsageRecord, error)

	// CountByTenant counts usage records for a tenant within a time range
	CountByTenant(ctx context.Context, tenantID uuid.UUID, filter UsageRecordFilter) (int64, error)

	// SumByTenantAndType calculates total usage for a tenant and type within a time range
	SumByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType UsageType, start, end time.Time) (int64, error)

	// GetAggregatedUsage returns aggregated usage data for a tenant
	GetAggregatedUsage(ctx context.Context, tenantID uuid.UUID, usageType UsageType, start, end time.Time, groupBy AggregationPeriod) ([]UsageAggregation, error)

	// DeleteOlderThan removes usage records older than the specified time (for data retention)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// UsageRecordFilter defines filtering options for usage record queries
type UsageRecordFilter struct {
	StartTime  *time.Time  // Filter records from this time
	EndTime    *time.Time  // Filter records until this time
	UsageTypes []UsageType // Filter by specific usage types
	SourceType string      // Filter by source type
	UserID     *uuid.UUID  // Filter by user
	Page       int         // Page number (1-based)
	PageSize   int         // Number of records per page
	OrderBy    string      // Field to order by
	OrderDir   string      // Order direction (asc/desc)
}

// DefaultUsageRecordFilter returns a filter with default values
func DefaultUsageRecordFilter() UsageRecordFilter {
	return UsageRecordFilter{
		Page:     1,
		PageSize: 100,
		OrderBy:  "recorded_at",
		OrderDir: "desc",
	}
}

// WithTimeRange sets the time range for the filter
func (f UsageRecordFilter) WithTimeRange(start, end time.Time) UsageRecordFilter {
	f.StartTime = &start
	f.EndTime = &end
	return f
}

// WithUsageTypes sets the usage types filter
func (f UsageRecordFilter) WithUsageTypes(types ...UsageType) UsageRecordFilter {
	f.UsageTypes = types
	return f
}

// WithPagination sets pagination options
func (f UsageRecordFilter) WithPagination(page, pageSize int) UsageRecordFilter {
	f.Page = page
	f.PageSize = pageSize
	return f
}

// AggregationPeriod defines the time period for aggregating usage data
type AggregationPeriod string

const (
	// AggregationPeriodHour aggregates by hour
	AggregationPeriodHour AggregationPeriod = "HOUR"

	// AggregationPeriodDay aggregates by day
	AggregationPeriodDay AggregationPeriod = "DAY"

	// AggregationPeriodWeek aggregates by week
	AggregationPeriodWeek AggregationPeriod = "WEEK"

	// AggregationPeriodMonth aggregates by month
	AggregationPeriodMonth AggregationPeriod = "MONTH"
)

// String returns the string representation of AggregationPeriod
func (a AggregationPeriod) String() string {
	return string(a)
}

// IsValid returns true if the aggregation period is valid
func (a AggregationPeriod) IsValid() bool {
	switch a {
	case AggregationPeriodHour, AggregationPeriodDay, AggregationPeriodWeek, AggregationPeriodMonth:
		return true
	}
	return false
}

// UsageAggregation represents aggregated usage data for a time period
type UsageAggregation struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	TotalUsage  int64
	RecordCount int64
	MinUsage    int64
	MaxUsage    int64
	AvgUsage    float64
}

// UsageQuotaRepository defines the interface for persisting and querying usage quotas
type UsageQuotaRepository interface {
	// Save persists a usage quota
	Save(ctx context.Context, quota *UsageQuota) error

	// Update updates an existing usage quota
	Update(ctx context.Context, quota *UsageQuota) error

	// FindByID retrieves a usage quota by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*UsageQuota, error)

	// FindByPlan retrieves all quotas for a subscription plan
	FindByPlan(ctx context.Context, planID string) ([]*UsageQuota, error)

	// FindByPlanAndType retrieves a specific quota for a plan and usage type
	FindByPlanAndType(ctx context.Context, planID string, usageType UsageType) (*UsageQuota, error)

	// FindByTenant retrieves tenant-specific quota overrides
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*UsageQuota, error)

	// FindByTenantAndType retrieves a tenant-specific quota for a usage type
	FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType UsageType) (*UsageQuota, error)

	// FindEffectiveQuota retrieves the effective quota for a tenant and usage type
	// (tenant override if exists, otherwise plan default)
	FindEffectiveQuota(ctx context.Context, tenantID uuid.UUID, planID string, usageType UsageType) (*UsageQuota, error)

	// FindAllEffectiveQuotas retrieves all effective quotas for a tenant
	FindAllEffectiveQuotas(ctx context.Context, tenantID uuid.UUID, planID string) ([]*UsageQuota, error)

	// Delete removes a usage quota
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByTenant removes all tenant-specific quota overrides
	DeleteByTenant(ctx context.Context, tenantID uuid.UUID) error
}

// UsageMeterRepository defines the interface for caching and querying usage meters
type UsageMeterRepository interface {
	// GetMeter retrieves a cached usage meter for a tenant and type
	GetMeter(ctx context.Context, tenantID uuid.UUID, usageType UsageType, periodStart, periodEnd time.Time) (*UsageMeter, error)

	// SetMeter caches a usage meter
	SetMeter(ctx context.Context, meter *UsageMeter, ttl time.Duration) error

	// InvalidateMeter removes a cached meter
	InvalidateMeter(ctx context.Context, tenantID uuid.UUID, usageType UsageType) error

	// InvalidateAllMeters removes all cached meters for a tenant
	InvalidateAllMeters(ctx context.Context, tenantID uuid.UUID) error

	// GetSummary retrieves a cached usage summary for a tenant
	GetSummary(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*UsageSummary, error)

	// SetSummary caches a usage summary
	SetSummary(ctx context.Context, summary *UsageSummary, ttl time.Duration) error

	// CalculateMeter calculates a fresh usage meter from usage records
	CalculateMeter(ctx context.Context, tenantID uuid.UUID, usageType UsageType, periodStart, periodEnd time.Time) (*UsageMeter, error)

	// CalculateSummary calculates a fresh usage summary from usage records
	CalculateSummary(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*UsageSummary, error)
}

// UsageService defines the interface for usage metering operations
type UsageService interface {
	// RecordUsage records a single usage event
	RecordUsage(ctx context.Context, record *UsageRecord) error

	// RecordUsageBatch records multiple usage events
	RecordUsageBatch(ctx context.Context, records []*UsageRecord) error

	// GetCurrentUsage retrieves current usage for a tenant and type in the current billing period
	GetCurrentUsage(ctx context.Context, tenantID uuid.UUID, usageType UsageType) (*UsageMeter, error)

	// GetUsageSummary retrieves a summary of all usage for a tenant
	GetUsageSummary(ctx context.Context, tenantID uuid.UUID) (*UsageSummary, error)

	// CheckQuota checks if a tenant can consume the specified amount
	CheckQuota(ctx context.Context, tenantID uuid.UUID, usageType UsageType, amount int64) (QuotaCheckResult, error)

	// GetQuotaStatus retrieves the quota status for all usage types
	GetQuotaStatus(ctx context.Context, tenantID uuid.UUID) (map[UsageType]QuotaCheckResult, error)

	// GetUsageTrend retrieves usage trend data for analytics
	GetUsageTrend(ctx context.Context, tenantID uuid.UUID, usageType UsageType, periods int, period AggregationPeriod) (*UsageTrend, error)
}

// UsageEventPublisher defines the interface for publishing usage-related events
type UsageEventPublisher interface {
	// PublishQuotaWarning publishes an event when usage approaches quota limit
	PublishQuotaWarning(ctx context.Context, tenantID uuid.UUID, result QuotaCheckResult) error

	// PublishQuotaExceeded publishes an event when usage exceeds quota limit
	PublishQuotaExceeded(ctx context.Context, tenantID uuid.UUID, result QuotaCheckResult) error

	// PublishUsageRecorded publishes an event when usage is recorded
	PublishUsageRecorded(ctx context.Context, record *UsageRecord) error
}
