package billing

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// UsageRecord represents an immutable record of a single usage event.
// Once created, usage records cannot be modified - corrections must be made with new records.
// This ensures a complete audit trail of all usage events.
type UsageRecord struct {
	shared.BaseEntity
	TenantID    uuid.UUID  // The tenant this usage belongs to
	UsageType   UsageType  // Type of usage being recorded
	Quantity    int64      // Amount of usage (always positive)
	Unit        UsageUnit  // Unit of measurement
	RecordedAt  time.Time  // When the usage occurred
	PeriodStart time.Time  // Start of the billing period
	PeriodEnd   time.Time  // End of the billing period
	SourceType  string     // Source of the usage event (e.g., "sales_order", "api_request")
	SourceID    string     // ID of the source entity (optional)
	Metadata    Metadata   // Additional context about the usage
	UserID      *uuid.UUID // User who triggered the usage (optional)
	IPAddress   string     // IP address of the request (for API calls)
	UserAgent   string     // User agent of the request (for API calls)
}

// Metadata holds additional context about a usage record
type Metadata map[string]any

// NewUsageRecord creates a new usage record with validation
func NewUsageRecord(
	tenantID uuid.UUID,
	usageType UsageType,
	quantity int64,
	periodStart time.Time,
	periodEnd time.Time,
) (*UsageRecord, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}
	if !usageType.IsValid() {
		return nil, shared.NewDomainError("INVALID_USAGE_TYPE", "Invalid usage type")
	}
	if quantity < 0 {
		return nil, shared.NewDomainError("INVALID_QUANTITY", "Quantity cannot be negative")
	}
	if periodEnd.Before(periodStart) {
		return nil, shared.NewDomainError("INVALID_PERIOD", "Period end cannot be before period start")
	}

	now := time.Now()
	return &UsageRecord{
		BaseEntity:  shared.NewBaseEntity(),
		TenantID:    tenantID,
		UsageType:   usageType,
		Quantity:    quantity,
		Unit:        usageType.Unit(),
		RecordedAt:  now,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Metadata:    make(Metadata),
	}, nil
}

// NewUsageRecordSimple creates a usage record for the current month
func NewUsageRecordSimple(
	tenantID uuid.UUID,
	usageType UsageType,
	quantity int64,
) (*UsageRecord, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return NewUsageRecord(tenantID, usageType, quantity, periodStart, periodEnd)
}

// WithSource sets the source information for the usage record
func (r *UsageRecord) WithSource(sourceType, sourceID string) *UsageRecord {
	r.SourceType = sourceType
	r.SourceID = sourceID
	return r
}

// WithUser sets the user who triggered the usage
func (r *UsageRecord) WithUser(userID uuid.UUID) *UsageRecord {
	r.UserID = &userID
	return r
}

// WithRequestInfo sets request information for API call tracking
func (r *UsageRecord) WithRequestInfo(ipAddress, userAgent string) *UsageRecord {
	r.IPAddress = ipAddress
	r.UserAgent = userAgent
	return r
}

// WithMetadata adds metadata to the usage record
func (r *UsageRecord) WithMetadata(key string, value any) *UsageRecord {
	if r.Metadata == nil {
		r.Metadata = make(Metadata)
	}
	r.Metadata[key] = value
	return r
}

// WithRecordedAt sets a custom recorded time (useful for batch imports)
func (r *UsageRecord) WithRecordedAt(recordedAt time.Time) *UsageRecord {
	r.RecordedAt = recordedAt
	return r
}

// IsInPeriod returns true if the given time falls within this record's billing period
func (r *UsageRecord) IsInPeriod(t time.Time) bool {
	return !t.Before(r.PeriodStart) && !t.After(r.PeriodEnd)
}

// GetFormattedQuantity returns the quantity formatted with its unit
func (r *UsageRecord) GetFormattedQuantity() string {
	return r.Unit.FormatValue(r.Quantity)
}

// UsageRecordBuilder provides a fluent interface for building usage records
type UsageRecordBuilder struct {
	record *UsageRecord
	err    error
}

// NewUsageRecordBuilder creates a new usage record builder
func NewUsageRecordBuilder(
	tenantID uuid.UUID,
	usageType UsageType,
	quantity int64,
) *UsageRecordBuilder {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	record, err := NewUsageRecord(tenantID, usageType, quantity, periodStart, periodEnd)
	return &UsageRecordBuilder{record: record, err: err}
}

// WithPeriod sets the billing period
func (b *UsageRecordBuilder) WithPeriod(start, end time.Time) *UsageRecordBuilder {
	if b.err != nil {
		return b
	}
	if end.Before(start) {
		b.err = shared.NewDomainError("INVALID_PERIOD", "Period end cannot be before period start")
		return b
	}
	b.record.PeriodStart = start
	b.record.PeriodEnd = end
	return b
}

// WithSource sets the source information
func (b *UsageRecordBuilder) WithSource(sourceType, sourceID string) *UsageRecordBuilder {
	if b.err != nil {
		return b
	}
	b.record.WithSource(sourceType, sourceID)
	return b
}

// WithUser sets the user
func (b *UsageRecordBuilder) WithUser(userID uuid.UUID) *UsageRecordBuilder {
	if b.err != nil {
		return b
	}
	b.record.WithUser(userID)
	return b
}

// WithRequestInfo sets request information
func (b *UsageRecordBuilder) WithRequestInfo(ipAddress, userAgent string) *UsageRecordBuilder {
	if b.err != nil {
		return b
	}
	b.record.WithRequestInfo(ipAddress, userAgent)
	return b
}

// WithMetadata adds metadata
func (b *UsageRecordBuilder) WithMetadata(key string, value any) *UsageRecordBuilder {
	if b.err != nil {
		return b
	}
	b.record.WithMetadata(key, value)
	return b
}

// WithRecordedAt sets the recorded time
func (b *UsageRecordBuilder) WithRecordedAt(recordedAt time.Time) *UsageRecordBuilder {
	if b.err != nil {
		return b
	}
	b.record.WithRecordedAt(recordedAt)
	return b
}

// Build returns the built usage record or an error
func (b *UsageRecordBuilder) Build() (*UsageRecord, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.record, nil
}

// CreateAPICallRecord is a helper to create an API call usage record
func CreateAPICallRecord(tenantID uuid.UUID, endpoint string, userID *uuid.UUID, ipAddress, userAgent string) (*UsageRecord, error) {
	record, err := NewUsageRecordSimple(tenantID, UsageTypeAPICalls, 1)
	if err != nil {
		return nil, err
	}

	record.WithSource("api_request", endpoint)
	record.WithRequestInfo(ipAddress, userAgent)
	record.WithMetadata("endpoint", endpoint)

	if userID != nil {
		record.WithUser(*userID)
	}

	return record, nil
}

// CreateStorageRecord is a helper to create a storage usage record
func CreateStorageRecord(tenantID uuid.UUID, bytes int64, sourceType, sourceID string) (*UsageRecord, error) {
	record, err := NewUsageRecordSimple(tenantID, UsageTypeStorageBytes, bytes)
	if err != nil {
		return nil, err
	}

	record.WithSource(sourceType, sourceID)
	return record, nil
}

// CreateOrderRecord is a helper to create an order creation usage record
func CreateOrderRecord(tenantID uuid.UUID, orderType, orderID string, userID uuid.UUID) (*UsageRecord, error) {
	record, err := NewUsageRecordSimple(tenantID, UsageTypeOrdersCreated, 1)
	if err != nil {
		return nil, err
	}

	record.WithSource(orderType, orderID)
	record.WithUser(userID)
	record.WithMetadata("order_type", orderType)

	return record, nil
}
