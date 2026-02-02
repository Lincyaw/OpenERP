package identity

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// TenantStatusChangeType represents the type of status change
type TenantStatusChangeType string

const (
	TenantStatusChangeTypeSuspend    TenantStatusChangeType = "suspend"
	TenantStatusChangeTypeActivate   TenantStatusChangeType = "activate"
	TenantStatusChangeTypeDeactivate TenantStatusChangeType = "deactivate"
)

// IsValid checks if the change type is valid
func (t TenantStatusChangeType) IsValid() bool {
	switch t {
	case TenantStatusChangeTypeSuspend, TenantStatusChangeTypeActivate, TenantStatusChangeTypeDeactivate:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (t TenantStatusChangeType) String() string {
	return string(t)
}

// TenantStatusHistory records a tenant status change event
// This is an immutable entity that tracks all status changes for audit purposes
type TenantStatusHistory struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	ChangeType            TenantStatusChangeType
	OldStatus             TenantStatus
	NewStatus             TenantStatus
	Reason                string     // Reason for the status change (especially for suspension)
	ScheduledReactivateAt *time.Time // For suspensions: when auto-reactivation is scheduled
	ChangedByUserID       uuid.UUID  // Admin who made the change
	IPAddress             string     // IP address of the admin
	UserAgent             string     // User agent of the admin
	CreatedAt             time.Time
}

// TenantStatusHistoryBuilder provides a fluent interface for creating status history
type TenantStatusHistoryBuilder struct {
	history *TenantStatusHistory
}

// NewTenantStatusHistory creates a new status history builder
func NewTenantStatusHistory(tenantID, changedByUserID uuid.UUID, changeType TenantStatusChangeType) *TenantStatusHistoryBuilder {
	return &TenantStatusHistoryBuilder{
		history: &TenantStatusHistory{
			ID:              uuid.New(),
			TenantID:        tenantID,
			ChangeType:      changeType,
			ChangedByUserID: changedByUserID,
			CreatedAt:       time.Now(),
		},
	}
}

// WithStatusChange sets the status change details
func (b *TenantStatusHistoryBuilder) WithStatusChange(oldStatus, newStatus TenantStatus) *TenantStatusHistoryBuilder {
	b.history.OldStatus = oldStatus
	b.history.NewStatus = newStatus
	return b
}

// WithReason sets the reason for the change
func (b *TenantStatusHistoryBuilder) WithReason(reason string) *TenantStatusHistoryBuilder {
	b.history.Reason = reason
	return b
}

// WithScheduledReactivateAt sets the scheduled reactivation time
func (b *TenantStatusHistoryBuilder) WithScheduledReactivateAt(scheduledAt *time.Time) *TenantStatusHistoryBuilder {
	b.history.ScheduledReactivateAt = scheduledAt
	return b
}

// WithIPAddress sets the IP address
func (b *TenantStatusHistoryBuilder) WithIPAddress(ip string) *TenantStatusHistoryBuilder {
	b.history.IPAddress = ip
	return b
}

// WithUserAgent sets the user agent
func (b *TenantStatusHistoryBuilder) WithUserAgent(userAgent string) *TenantStatusHistoryBuilder {
	b.history.UserAgent = userAgent
	return b
}

// Build validates and returns the status history
func (b *TenantStatusHistoryBuilder) Build() (*TenantStatusHistory, error) {
	if b.history.TenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT_ID", "Tenant ID cannot be empty")
	}
	if b.history.ChangedByUserID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CHANGED_BY_USER_ID", "Changed by user ID cannot be empty")
	}
	if !b.history.ChangeType.IsValid() {
		return nil, shared.NewDomainError("INVALID_CHANGE_TYPE", "Invalid status change type")
	}
	if b.history.OldStatus == "" || b.history.NewStatus == "" {
		return nil, shared.NewDomainError("INVALID_STATUS_CHANGE", "Status change requires both old and new status")
	}

	return b.history, nil
}

// IsSuspension returns true if this is a suspension change
func (h *TenantStatusHistory) IsSuspension() bool {
	return h.ChangeType == TenantStatusChangeTypeSuspend
}

// IsActivation returns true if this is an activation change
func (h *TenantStatusHistory) IsActivation() bool {
	return h.ChangeType == TenantStatusChangeTypeActivate
}

// HasScheduledReactivation returns true if there's a scheduled reactivation
func (h *TenantStatusHistory) HasScheduledReactivation() bool {
	return h.ScheduledReactivateAt != nil
}
