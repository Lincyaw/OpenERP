package identity

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// SubscriptionChangeType represents the type of subscription change
type SubscriptionChangeType string

const (
	SubscriptionChangeTypePlanUpgrade   SubscriptionChangeType = "plan_upgrade"
	SubscriptionChangeTypePlanDowngrade SubscriptionChangeType = "plan_downgrade"
	SubscriptionChangeTypeQuotaUpdate   SubscriptionChangeType = "quota_update"
)

// IsValid checks if the change type is valid
func (t SubscriptionChangeType) IsValid() bool {
	switch t {
	case SubscriptionChangeTypePlanUpgrade, SubscriptionChangeTypePlanDowngrade, SubscriptionChangeTypeQuotaUpdate:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (t SubscriptionChangeType) String() string {
	return string(t)
}

// TenantQuota represents quota limits for a tenant
// This is a value object used for tracking quota changes
type TenantQuota struct {
	MaxUsers      int `json:"max_users"`
	MaxWarehouses int `json:"max_warehouses"`
	MaxProducts   int `json:"max_products"`
}

// NewTenantQuota creates a new TenantQuota from TenantConfig
func NewTenantQuota(config TenantConfig) TenantQuota {
	return TenantQuota{
		MaxUsers:      config.MaxUsers,
		MaxWarehouses: config.MaxWarehouses,
		MaxProducts:   config.MaxProducts,
	}
}

// Equals checks if two quotas are equal
func (q TenantQuota) Equals(other TenantQuota) bool {
	return q.MaxUsers == other.MaxUsers &&
		q.MaxWarehouses == other.MaxWarehouses &&
		q.MaxProducts == other.MaxProducts
}

// SubscriptionHistory records a subscription change event
// This is an immutable entity that tracks all subscription changes for audit purposes
type SubscriptionHistory struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	ChangeType      SubscriptionChangeType
	OldPlan         TenantPlan
	NewPlan         TenantPlan
	OldQuota        *TenantQuota // nil for plan-only changes
	NewQuota        *TenantQuota // nil for plan-only changes
	EffectiveAt     time.Time    // When the change takes/took effect
	ScheduledAt     *time.Time   // For downgrades: when it was scheduled (nil for immediate changes)
	ChangedByUserID uuid.UUID    // Admin who made the change
	Reason          string       // Optional reason for the change
	CreatedAt       time.Time
}

// SubscriptionHistoryBuilder provides a fluent interface for creating subscription history
type SubscriptionHistoryBuilder struct {
	history *SubscriptionHistory
}

// NewSubscriptionHistory creates a new subscription history builder
func NewSubscriptionHistory(tenantID, changedByUserID uuid.UUID, changeType SubscriptionChangeType) *SubscriptionHistoryBuilder {
	return &SubscriptionHistoryBuilder{
		history: &SubscriptionHistory{
			ID:              uuid.New(),
			TenantID:        tenantID,
			ChangeType:      changeType,
			ChangedByUserID: changedByUserID,
			EffectiveAt:     time.Now(),
			CreatedAt:       time.Now(),
		},
	}
}

// WithPlanChange sets the plan change details
func (b *SubscriptionHistoryBuilder) WithPlanChange(oldPlan, newPlan TenantPlan) *SubscriptionHistoryBuilder {
	b.history.OldPlan = oldPlan
	b.history.NewPlan = newPlan
	return b
}

// WithQuotaChange sets the quota change details
func (b *SubscriptionHistoryBuilder) WithQuotaChange(oldQuota, newQuota TenantQuota) *SubscriptionHistoryBuilder {
	b.history.OldQuota = &oldQuota
	b.history.NewQuota = &newQuota
	return b
}

// WithEffectiveAt sets when the change takes effect
func (b *SubscriptionHistoryBuilder) WithEffectiveAt(effectiveAt time.Time) *SubscriptionHistoryBuilder {
	b.history.EffectiveAt = effectiveAt
	return b
}

// WithScheduledAt sets when the change was scheduled (for downgrades)
func (b *SubscriptionHistoryBuilder) WithScheduledAt(scheduledAt time.Time) *SubscriptionHistoryBuilder {
	b.history.ScheduledAt = &scheduledAt
	return b
}

// WithReason sets the reason for the change
func (b *SubscriptionHistoryBuilder) WithReason(reason string) *SubscriptionHistoryBuilder {
	b.history.Reason = reason
	return b
}

// Build validates and returns the subscription history
func (b *SubscriptionHistoryBuilder) Build() (*SubscriptionHistory, error) {
	if b.history.TenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT_ID", "Tenant ID cannot be empty")
	}
	if b.history.ChangedByUserID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CHANGED_BY_USER_ID", "Changed by user ID cannot be empty")
	}
	if !b.history.ChangeType.IsValid() {
		return nil, shared.NewDomainError("INVALID_CHANGE_TYPE", "Invalid subscription change type")
	}

	// Validate based on change type
	switch b.history.ChangeType {
	case SubscriptionChangeTypePlanUpgrade, SubscriptionChangeTypePlanDowngrade:
		if b.history.OldPlan == "" || b.history.NewPlan == "" {
			return nil, shared.NewDomainError("INVALID_PLAN_CHANGE", "Plan change requires both old and new plan")
		}
		if b.history.OldPlan == b.history.NewPlan {
			return nil, shared.NewDomainError("INVALID_PLAN_CHANGE", "Old and new plan cannot be the same")
		}
	case SubscriptionChangeTypeQuotaUpdate:
		if b.history.OldQuota == nil || b.history.NewQuota == nil {
			return nil, shared.NewDomainError("INVALID_QUOTA_CHANGE", "Quota change requires both old and new quota")
		}
	}

	return b.history, nil
}

// IsScheduled returns true if this is a scheduled change (not yet effective)
func (h *SubscriptionHistory) IsScheduled() bool {
	return h.ScheduledAt != nil && h.EffectiveAt.After(time.Now())
}

// IsPlanChange returns true if this is a plan change (upgrade or downgrade)
func (h *SubscriptionHistory) IsPlanChange() bool {
	return h.ChangeType == SubscriptionChangeTypePlanUpgrade || h.ChangeType == SubscriptionChangeTypePlanDowngrade
}

// IsQuotaChange returns true if this is a quota change
func (h *SubscriptionHistory) IsQuotaChange() bool {
	return h.ChangeType == SubscriptionChangeTypeQuotaUpdate
}
