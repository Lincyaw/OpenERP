package billing

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// UsageQuota defines usage limits for a specific usage type and subscription plan.
// Quotas can be configured at the plan level (default) or overridden for specific tenants.
type UsageQuota struct {
	shared.BaseAggregateRoot
	PlanID        string        // Subscription plan ID (e.g., "free", "basic", "pro", "enterprise")
	TenantID      *uuid.UUID    // Optional tenant-specific override (nil = plan default)
	UsageType     UsageType     // Type of usage being limited
	Limit         int64         // Maximum allowed usage (-1 = unlimited)
	Unit          UsageUnit     // Unit of measurement
	ResetPeriod   ResetPeriod   // When the quota resets
	SoftLimit     *int64        // Optional soft limit for warnings (nil = no soft limit)
	OveragePolicy OveragePolicy // What happens when quota is exceeded
	Description   string        // Human-readable description
	IsActive      bool          // Whether this quota is currently active
}

// OveragePolicy defines what happens when a quota is exceeded
type OveragePolicy string

const (
	// OveragePolicyBlock blocks further usage when quota is exceeded
	OveragePolicyBlock OveragePolicy = "BLOCK"

	// OveragePolicyWarn allows usage but sends warnings
	OveragePolicyWarn OveragePolicy = "WARN"

	// OveragePolicyCharge allows usage and charges for overage
	OveragePolicyCharge OveragePolicy = "CHARGE"

	// OveragePolicyThrottle reduces service quality/speed
	OveragePolicyThrottle OveragePolicy = "THROTTLE"
)

// String returns the string representation of OveragePolicy
func (o OveragePolicy) String() string {
	return string(o)
}

// IsValid returns true if the overage policy is valid
func (o OveragePolicy) IsValid() bool {
	switch o {
	case OveragePolicyBlock, OveragePolicyWarn, OveragePolicyCharge, OveragePolicyThrottle:
		return true
	}
	return false
}

// NewUsageQuota creates a new usage quota for a plan
func NewUsageQuota(
	planID string,
	usageType UsageType,
	limit int64,
	resetPeriod ResetPeriod,
) (*UsageQuota, error) {
	if planID == "" {
		return nil, shared.NewDomainError("INVALID_PLAN", "Plan ID cannot be empty")
	}
	if !usageType.IsValid() {
		return nil, shared.NewDomainError("INVALID_USAGE_TYPE", "Invalid usage type")
	}
	if limit < -1 {
		return nil, shared.NewDomainError("INVALID_LIMIT", "Limit must be -1 (unlimited) or non-negative")
	}
	if !resetPeriod.IsValid() {
		return nil, shared.NewDomainError("INVALID_RESET_PERIOD", "Invalid reset period")
	}

	return &UsageQuota{
		BaseAggregateRoot: shared.NewBaseAggregateRoot(),
		PlanID:            planID,
		UsageType:         usageType,
		Limit:             limit,
		Unit:              usageType.Unit(),
		ResetPeriod:       resetPeriod,
		OveragePolicy:     OveragePolicyBlock, // Default to blocking
		IsActive:          true,
	}, nil
}

// NewTenantUsageQuota creates a tenant-specific quota override
func NewTenantUsageQuota(
	tenantID uuid.UUID,
	planID string,
	usageType UsageType,
	limit int64,
	resetPeriod ResetPeriod,
) (*UsageQuota, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}

	quota, err := NewUsageQuota(planID, usageType, limit, resetPeriod)
	if err != nil {
		return nil, err
	}

	quota.TenantID = &tenantID
	return quota, nil
}

// WithSoftLimit sets a soft limit for warnings
func (q *UsageQuota) WithSoftLimit(softLimit int64) *UsageQuota {
	if softLimit >= 0 && (q.Limit == -1 || softLimit < q.Limit) {
		q.SoftLimit = &softLimit
	}
	return q
}

// WithOveragePolicy sets the overage policy
func (q *UsageQuota) WithOveragePolicy(policy OveragePolicy) *UsageQuota {
	if policy.IsValid() {
		q.OveragePolicy = policy
	}
	return q
}

// WithDescription sets the description
func (q *UsageQuota) WithDescription(description string) *UsageQuota {
	q.Description = description
	return q
}

// SetLimit updates the quota limit
func (q *UsageQuota) SetLimit(limit int64) error {
	if limit < -1 {
		return shared.NewDomainError("INVALID_LIMIT", "Limit must be -1 (unlimited) or non-negative")
	}
	q.Limit = limit
	q.UpdatedAt = time.Now()
	return nil
}

// SetSoftLimit updates the soft limit
func (q *UsageQuota) SetSoftLimit(softLimit *int64) error {
	if softLimit != nil && *softLimit < 0 {
		return shared.NewDomainError("INVALID_SOFT_LIMIT", "Soft limit cannot be negative")
	}
	if softLimit != nil && q.Limit != -1 && *softLimit >= q.Limit {
		return shared.NewDomainError("INVALID_SOFT_LIMIT", "Soft limit must be less than hard limit")
	}
	q.SoftLimit = softLimit
	q.UpdatedAt = time.Now()
	return nil
}

// Activate activates the quota
func (q *UsageQuota) Activate() {
	q.IsActive = true
	q.UpdatedAt = time.Now()
}

// Deactivate deactivates the quota
func (q *UsageQuota) Deactivate() {
	q.IsActive = false
	q.UpdatedAt = time.Now()
}

// IsUnlimited returns true if the quota has no limit
func (q *UsageQuota) IsUnlimited() bool {
	return q.Limit == -1
}

// IsTenantOverride returns true if this is a tenant-specific override
func (q *UsageQuota) IsTenantOverride() bool {
	return q.TenantID != nil
}

// CheckUsage checks if the given usage amount is within quota
func (q *UsageQuota) CheckUsage(currentUsage int64) QuotaCheckResult {
	result := QuotaCheckResult{
		UsageType:     q.UsageType,
		CurrentUsage:  currentUsage,
		Limit:         q.Limit,
		SoftLimit:     q.SoftLimit,
		OveragePolicy: q.OveragePolicy,
		IsUnlimited:   q.IsUnlimited(),
	}

	if !q.IsActive {
		result.Status = QuotaStatusInactive
		return result
	}

	if q.IsUnlimited() {
		result.Status = QuotaStatusOK
		return result
	}

	result.Remaining = q.Limit - currentUsage
	result.UsagePercent = float64(currentUsage) / float64(q.Limit) * 100

	switch {
	case currentUsage > q.Limit:
		result.Status = QuotaStatusExceeded
		result.Overage = currentUsage - q.Limit
	case q.SoftLimit != nil && currentUsage >= *q.SoftLimit:
		result.Status = QuotaStatusWarning
	default:
		result.Status = QuotaStatusOK
	}

	return result
}

// CanConsume checks if the given amount can be consumed without exceeding quota
func (q *UsageQuota) CanConsume(currentUsage, amount int64) bool {
	if !q.IsActive {
		return true // Inactive quotas don't block
	}
	if q.IsUnlimited() {
		return true
	}
	return currentUsage+amount <= q.Limit
}

// GetFormattedLimit returns the limit formatted with its unit
func (q *UsageQuota) GetFormattedLimit() string {
	if q.IsUnlimited() {
		return "Unlimited"
	}
	return q.Unit.FormatValue(q.Limit)
}

// GetFormattedSoftLimit returns the soft limit formatted with its unit
func (q *UsageQuota) GetFormattedSoftLimit() string {
	if q.SoftLimit == nil {
		return "N/A"
	}
	return q.Unit.FormatValue(*q.SoftLimit)
}

// QuotaStatus represents the status of quota usage
type QuotaStatus string

const (
	// QuotaStatusOK indicates usage is within normal limits
	QuotaStatusOK QuotaStatus = "OK"

	// QuotaStatusWarning indicates usage has reached the soft limit
	QuotaStatusWarning QuotaStatus = "WARNING"

	// QuotaStatusExceeded indicates usage has exceeded the hard limit
	QuotaStatusExceeded QuotaStatus = "EXCEEDED"

	// QuotaStatusInactive indicates the quota is not active
	QuotaStatusInactive QuotaStatus = "INACTIVE"
)

// String returns the string representation of QuotaStatus
func (s QuotaStatus) String() string {
	return string(s)
}

// QuotaCheckResult represents the result of checking usage against a quota
type QuotaCheckResult struct {
	UsageType     UsageType
	Status        QuotaStatus
	CurrentUsage  int64
	Limit         int64
	SoftLimit     *int64
	Remaining     int64
	Overage       int64
	UsagePercent  float64
	OveragePolicy OveragePolicy
	IsUnlimited   bool
}

// IsAllowed returns true if the usage is allowed based on the overage policy
func (r QuotaCheckResult) IsAllowed() bool {
	switch r.Status {
	case QuotaStatusOK, QuotaStatusWarning, QuotaStatusInactive:
		return true
	case QuotaStatusExceeded:
		// Only block if policy is BLOCK
		return r.OveragePolicy != OveragePolicyBlock
	}
	return false
}

// ShouldWarn returns true if a warning should be sent
func (r QuotaCheckResult) ShouldWarn() bool {
	return r.Status == QuotaStatusWarning || r.Status == QuotaStatusExceeded
}

// ShouldCharge returns true if overage charges apply
func (r QuotaCheckResult) ShouldCharge() bool {
	return r.Status == QuotaStatusExceeded && r.OveragePolicy == OveragePolicyCharge
}

// GetMessage returns a human-readable message about the quota status
func (r QuotaCheckResult) GetMessage() string {
	switch r.Status {
	case QuotaStatusOK:
		if r.IsUnlimited {
			return "Usage is unlimited"
		}
		return "Usage is within quota"
	case QuotaStatusWarning:
		return "Usage is approaching quota limit"
	case QuotaStatusExceeded:
		return "Usage has exceeded quota limit"
	case QuotaStatusInactive:
		return "Quota is not active"
	default:
		return "Unknown quota status"
	}
}

// QuotaSet represents a collection of quotas for a plan or tenant
type QuotaSet struct {
	PlanID   string
	TenantID *uuid.UUID
	Quotas   map[UsageType]*UsageQuota
}

// NewQuotaSet creates a new quota set for a plan
func NewQuotaSet(planID string) *QuotaSet {
	return &QuotaSet{
		PlanID: planID,
		Quotas: make(map[UsageType]*UsageQuota),
	}
}

// NewTenantQuotaSet creates a new quota set for a specific tenant
func NewTenantQuotaSet(planID string, tenantID uuid.UUID) *QuotaSet {
	return &QuotaSet{
		PlanID:   planID,
		TenantID: &tenantID,
		Quotas:   make(map[UsageType]*UsageQuota),
	}
}

// AddQuota adds a quota to the set
func (s *QuotaSet) AddQuota(quota *UsageQuota) *QuotaSet {
	s.Quotas[quota.UsageType] = quota
	return s
}

// GetQuota returns the quota for a specific usage type
func (s *QuotaSet) GetQuota(usageType UsageType) *UsageQuota {
	return s.Quotas[usageType]
}

// HasQuota returns true if a quota exists for the usage type
func (s *QuotaSet) HasQuota(usageType UsageType) bool {
	_, exists := s.Quotas[usageType]
	return exists
}

// CheckAllUsage checks all quotas against the provided usage map
func (s *QuotaSet) CheckAllUsage(usage map[UsageType]int64) map[UsageType]QuotaCheckResult {
	results := make(map[UsageType]QuotaCheckResult)
	for usageType, quota := range s.Quotas {
		currentUsage := usage[usageType]
		results[usageType] = quota.CheckUsage(currentUsage)
	}
	return results
}

// GetExceededQuotas returns all quotas that are exceeded
func (s *QuotaSet) GetExceededQuotas(usage map[UsageType]int64) []*UsageQuota {
	var exceeded []*UsageQuota
	for usageType, quota := range s.Quotas {
		currentUsage := usage[usageType]
		result := quota.CheckUsage(currentUsage)
		if result.Status == QuotaStatusExceeded {
			exceeded = append(exceeded, quota)
		}
	}
	return exceeded
}
