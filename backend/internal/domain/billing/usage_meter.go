package billing

import (
	"time"

	"github.com/google/uuid"
)

// UsageMeter is a value object that represents aggregated usage statistics
// for a tenant over a specific time period. It provides a snapshot of usage
// that can be used for billing calculations, quota enforcement, and reporting.
type UsageMeter struct {
	TenantID    uuid.UUID // The tenant this meter belongs to
	UsageType   UsageType // Type of usage being metered
	Unit        UsageUnit // Unit of measurement
	PeriodStart time.Time // Start of the metering period
	PeriodEnd   time.Time // End of the metering period
	TotalUsage  int64     // Total usage in the period
	RecordCount int64     // Number of usage records in the period
	PeakUsage   int64     // Peak usage value (for countable resources)
	AverageRate float64   // Average usage rate per day
	LastUpdated time.Time // When this meter was last calculated
	QuotaLimit  *int64    // Optional quota limit for comparison
	QuotaUsed   float64   // Percentage of quota used (0-100+)
}

// NewUsageMeter creates a new usage meter
func NewUsageMeter(
	tenantID uuid.UUID,
	usageType UsageType,
	periodStart time.Time,
	periodEnd time.Time,
) *UsageMeter {
	return &UsageMeter{
		TenantID:    tenantID,
		UsageType:   usageType,
		Unit:        usageType.Unit(),
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		LastUpdated: time.Now(),
	}
}

// NewUsageMeterForCurrentMonth creates a usage meter for the current billing month
func NewUsageMeterForCurrentMonth(tenantID uuid.UUID, usageType UsageType) *UsageMeter {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return NewUsageMeter(tenantID, usageType, periodStart, periodEnd)
}

// WithTotalUsage sets the total usage
func (m *UsageMeter) WithTotalUsage(total int64) *UsageMeter {
	m.TotalUsage = total
	m.calculateQuotaUsed()
	return m
}

// WithRecordCount sets the record count
func (m *UsageMeter) WithRecordCount(count int64) *UsageMeter {
	m.RecordCount = count
	return m
}

// WithPeakUsage sets the peak usage
func (m *UsageMeter) WithPeakUsage(peak int64) *UsageMeter {
	m.PeakUsage = peak
	return m
}

// WithQuotaLimit sets the quota limit for comparison
func (m *UsageMeter) WithQuotaLimit(limit int64) *UsageMeter {
	m.QuotaLimit = &limit
	m.calculateQuotaUsed()
	return m
}

// calculateQuotaUsed calculates the percentage of quota used
func (m *UsageMeter) calculateQuotaUsed() {
	if m.QuotaLimit != nil && *m.QuotaLimit > 0 {
		m.QuotaUsed = float64(m.TotalUsage) / float64(*m.QuotaLimit) * 100
	} else {
		m.QuotaUsed = 0
	}
}

// CalculateAverageRate calculates the average daily usage rate
func (m *UsageMeter) CalculateAverageRate() *UsageMeter {
	days := m.PeriodEnd.Sub(m.PeriodStart).Hours() / 24
	if days > 0 {
		m.AverageRate = float64(m.TotalUsage) / days
	}
	return m
}

// IsOverQuota returns true if usage exceeds the quota limit
func (m *UsageMeter) IsOverQuota() bool {
	return m.QuotaLimit != nil && m.TotalUsage > *m.QuotaLimit
}

// IsNearQuota returns true if usage is at or above the given threshold percentage
func (m *UsageMeter) IsNearQuota(thresholdPercent float64) bool {
	return m.QuotaUsed >= thresholdPercent
}

// GetRemainingQuota returns the remaining quota, or -1 if unlimited
func (m *UsageMeter) GetRemainingQuota() int64 {
	if m.QuotaLimit == nil {
		return -1 // Unlimited
	}
	remaining := *m.QuotaLimit - m.TotalUsage
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetFormattedTotalUsage returns the total usage formatted with its unit
func (m *UsageMeter) GetFormattedTotalUsage() string {
	return m.Unit.FormatValue(m.TotalUsage)
}

// GetFormattedQuotaLimit returns the quota limit formatted with its unit
func (m *UsageMeter) GetFormattedQuotaLimit() string {
	if m.QuotaLimit == nil {
		return "Unlimited"
	}
	return m.Unit.FormatValue(*m.QuotaLimit)
}

// GetDaysRemaining returns the number of days remaining in the billing period
func (m *UsageMeter) GetDaysRemaining() int {
	now := time.Now()
	if now.After(m.PeriodEnd) {
		return 0
	}
	return int(m.PeriodEnd.Sub(now).Hours() / 24)
}

// GetDaysElapsed returns the number of days elapsed in the billing period
func (m *UsageMeter) GetDaysElapsed() int {
	now := time.Now()
	if now.Before(m.PeriodStart) {
		return 0
	}
	if now.After(m.PeriodEnd) {
		return int(m.PeriodEnd.Sub(m.PeriodStart).Hours() / 24)
	}
	return int(now.Sub(m.PeriodStart).Hours() / 24)
}

// ProjectedUsage estimates the total usage by the end of the period
// based on current usage rate
func (m *UsageMeter) ProjectedUsage() int64 {
	daysElapsed := m.GetDaysElapsed()
	if daysElapsed == 0 {
		return m.TotalUsage
	}

	totalDays := int(m.PeriodEnd.Sub(m.PeriodStart).Hours() / 24)
	dailyRate := float64(m.TotalUsage) / float64(daysElapsed)
	return int64(dailyRate * float64(totalDays))
}

// WillExceedQuota returns true if projected usage will exceed the quota
func (m *UsageMeter) WillExceedQuota() bool {
	if m.QuotaLimit == nil {
		return false
	}
	return m.ProjectedUsage() > *m.QuotaLimit
}

// UsageSummary provides a summary of usage across multiple types
type UsageSummary struct {
	TenantID    uuid.UUID
	PeriodStart time.Time
	PeriodEnd   time.Time
	Meters      map[UsageType]*UsageMeter
	LastUpdated time.Time
}

// NewUsageSummary creates a new usage summary
func NewUsageSummary(tenantID uuid.UUID, periodStart, periodEnd time.Time) *UsageSummary {
	return &UsageSummary{
		TenantID:    tenantID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Meters:      make(map[UsageType]*UsageMeter),
		LastUpdated: time.Now(),
	}
}

// AddMeter adds a usage meter to the summary
func (s *UsageSummary) AddMeter(meter *UsageMeter) *UsageSummary {
	s.Meters[meter.UsageType] = meter
	s.LastUpdated = time.Now()
	return s
}

// GetMeter returns the meter for a specific usage type
func (s *UsageSummary) GetMeter(usageType UsageType) *UsageMeter {
	return s.Meters[usageType]
}

// GetOverQuotaTypes returns all usage types that are over quota
func (s *UsageSummary) GetOverQuotaTypes() []UsageType {
	var overQuota []UsageType
	for usageType, meter := range s.Meters {
		if meter.IsOverQuota() {
			overQuota = append(overQuota, usageType)
		}
	}
	return overQuota
}

// GetNearQuotaTypes returns all usage types near the quota threshold
func (s *UsageSummary) GetNearQuotaTypes(thresholdPercent float64) []UsageType {
	var nearQuota []UsageType
	for usageType, meter := range s.Meters {
		if meter.IsNearQuota(thresholdPercent) && !meter.IsOverQuota() {
			nearQuota = append(nearQuota, usageType)
		}
	}
	return nearQuota
}

// HasAnyOverQuota returns true if any usage type is over quota
func (s *UsageSummary) HasAnyOverQuota() bool {
	return len(s.GetOverQuotaTypes()) > 0
}

// UsageTrend represents usage trend data for analytics
type UsageTrend struct {
	TenantID   uuid.UUID
	UsageType  UsageType
	DataPoints []UsageDataPoint
}

// UsageDataPoint represents a single data point in a usage trend
type UsageDataPoint struct {
	Timestamp time.Time
	Value     int64
}

// NewUsageTrend creates a new usage trend
func NewUsageTrend(tenantID uuid.UUID, usageType UsageType) *UsageTrend {
	return &UsageTrend{
		TenantID:   tenantID,
		UsageType:  usageType,
		DataPoints: make([]UsageDataPoint, 0),
	}
}

// AddDataPoint adds a data point to the trend
func (t *UsageTrend) AddDataPoint(timestamp time.Time, value int64) *UsageTrend {
	t.DataPoints = append(t.DataPoints, UsageDataPoint{
		Timestamp: timestamp,
		Value:     value,
	})
	return t
}

// GetLatestValue returns the most recent data point value
func (t *UsageTrend) GetLatestValue() int64 {
	if len(t.DataPoints) == 0 {
		return 0
	}
	return t.DataPoints[len(t.DataPoints)-1].Value
}

// GetGrowthRate calculates the growth rate between first and last data points
func (t *UsageTrend) GetGrowthRate() float64 {
	if len(t.DataPoints) < 2 {
		return 0
	}
	first := t.DataPoints[0].Value
	last := t.DataPoints[len(t.DataPoints)-1].Value
	if first == 0 {
		return 0
	}
	return float64(last-first) / float64(first) * 100
}
