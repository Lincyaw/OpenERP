package featureflag

import (
	"maps"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// FlagAuditLog represents an audit log entry for feature flag changes.
// It captures who did what, when, and the details of the change.
type FlagAuditLog struct {
	shared.BaseEntity
	FlagKey   string         `json:"flag_key"`
	Action    AuditAction    `json:"action"`
	OldValue  map[string]any `json:"old_value,omitempty"`
	NewValue  map[string]any `json:"new_value,omitempty"`
	UserID    *uuid.UUID     `json:"user_id,omitempty"`
	IPAddress string         `json:"ip_address,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
}

// NewFlagAuditLog creates a new flag audit log entry
func NewFlagAuditLog(
	flagKey string,
	action AuditAction,
	oldValue, newValue map[string]any,
	userID *uuid.UUID,
	ipAddress, userAgent string,
) (*FlagAuditLog, error) {
	if flagKey == "" {
		return nil, shared.NewDomainError("INVALID_FLAG_KEY", "Flag key cannot be empty")
	}
	if !action.IsValid() {
		return nil, shared.NewDomainError("INVALID_ACTION", "Invalid audit action")
	}

	return &FlagAuditLog{
		BaseEntity: shared.NewBaseEntity(),
		FlagKey:    flagKey,
		Action:     action,
		OldValue:   oldValue,
		NewValue:   newValue,
		UserID:     userID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}, nil
}

// GetFlagKey returns the flag key
func (l *FlagAuditLog) GetFlagKey() string {
	return l.FlagKey
}

// GetAction returns the audit action
func (l *FlagAuditLog) GetAction() AuditAction {
	return l.Action
}

// GetOldValue returns the old value
func (l *FlagAuditLog) GetOldValue() map[string]any {
	if l.OldValue == nil {
		return make(map[string]any)
	}
	// Return a copy to maintain immutability
	result := make(map[string]any, len(l.OldValue))
	maps.Copy(result, l.OldValue)
	return result
}

// GetNewValue returns the new value
func (l *FlagAuditLog) GetNewValue() map[string]any {
	if l.NewValue == nil {
		return make(map[string]any)
	}
	// Return a copy to maintain immutability
	result := make(map[string]any, len(l.NewValue))
	maps.Copy(result, l.NewValue)
	return result
}

// GetUserID returns the user ID who performed the action
func (l *FlagAuditLog) GetUserID() *uuid.UUID {
	return l.UserID
}

// GetIPAddress returns the IP address from where the action was performed
func (l *FlagAuditLog) GetIPAddress() string {
	return l.IPAddress
}

// GetUserAgent returns the user agent string
func (l *FlagAuditLog) GetUserAgent() string {
	return l.UserAgent
}

// GetTimestamp returns when the audit log was created
func (l *FlagAuditLog) GetTimestamp() time.Time {
	return l.CreatedAt
}
