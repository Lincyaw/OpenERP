package featureflag

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// FlagOverride represents a user or tenant-specific override for a feature flag
// It is an entity that belongs to the FeatureFlag aggregate
type FlagOverride struct {
	shared.BaseEntity
	FlagKey    string             `json:"flag_key"`
	TargetType OverrideTargetType `json:"target_type"`
	TargetID   uuid.UUID          `json:"target_id"`
	Value      FlagValue          `json:"value"`
	Reason     string             `json:"reason,omitempty"`
	ExpiresAt  *time.Time         `json:"expires_at,omitempty"`
	CreatedBy  *uuid.UUID         `json:"created_by,omitempty"`
}

// NewFlagOverride creates a new flag override
func NewFlagOverride(
	flagKey string,
	targetType OverrideTargetType,
	targetID uuid.UUID,
	value FlagValue,
	reason string,
	expiresAt *time.Time,
	createdBy *uuid.UUID,
) (*FlagOverride, error) {
	if flagKey == "" {
		return nil, shared.NewDomainError("INVALID_FLAG_KEY", "Flag key cannot be empty")
	}
	if !targetType.IsValid() {
		return nil, shared.NewDomainError("INVALID_TARGET_TYPE", "Invalid override target type")
	}
	if targetID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TARGET_ID", "Target ID cannot be empty")
	}
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, shared.NewDomainError("INVALID_EXPIRES_AT", "Expiration time cannot be in the past")
	}

	return &FlagOverride{
		BaseEntity: shared.NewBaseEntity(),
		FlagKey:    flagKey,
		TargetType: targetType,
		TargetID:   targetID,
		Value:      value,
		Reason:     reason,
		ExpiresAt:  expiresAt,
		CreatedBy:  createdBy,
	}, nil
}

// GetFlagKey returns the flag key
func (o *FlagOverride) GetFlagKey() string {
	return o.FlagKey
}

// GetTargetType returns the target type
func (o *FlagOverride) GetTargetType() OverrideTargetType {
	return o.TargetType
}

// GetTargetID returns the target ID
func (o *FlagOverride) GetTargetID() uuid.UUID {
	return o.TargetID
}

// GetValue returns the override value
func (o *FlagOverride) GetValue() FlagValue {
	return o.Value
}

// GetReason returns the reason for the override
func (o *FlagOverride) GetReason() string {
	return o.Reason
}

// GetExpiresAt returns the expiration time
func (o *FlagOverride) GetExpiresAt() *time.Time {
	return o.ExpiresAt
}

// GetCreatedBy returns the user who created the override
func (o *FlagOverride) GetCreatedBy() *uuid.UUID {
	return o.CreatedBy
}

// IsExpired returns true if the override has expired
func (o *FlagOverride) IsExpired() bool {
	if o.ExpiresAt == nil {
		return false
	}
	return o.ExpiresAt.Before(time.Now())
}

// IsActive returns true if the override is active (not expired)
func (o *FlagOverride) IsActive() bool {
	return !o.IsExpired()
}

// IsForUser returns true if this is a user-level override
func (o *FlagOverride) IsForUser() bool {
	return o.TargetType == OverrideTargetTypeUser
}

// IsForTenant returns true if this is a tenant-level override
func (o *FlagOverride) IsForTenant() bool {
	return o.TargetType == OverrideTargetTypeTenant
}

// MatchesTarget returns true if the override matches the given target
func (o *FlagOverride) MatchesTarget(targetType OverrideTargetType, targetID uuid.UUID) bool {
	return o.TargetType == targetType && o.TargetID == targetID
}

// UpdateValue updates the override value
func (o *FlagOverride) UpdateValue(value FlagValue) {
	o.Value = value
	o.UpdatedAt = time.Now()
}

// UpdateReason updates the override reason
func (o *FlagOverride) UpdateReason(reason string) {
	o.Reason = reason
	o.UpdatedAt = time.Now()
}

// UpdateExpiresAt updates the expiration time
func (o *FlagOverride) UpdateExpiresAt(expiresAt *time.Time) error {
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return shared.NewDomainError("INVALID_EXPIRES_AT", "Expiration time cannot be in the past")
	}
	o.ExpiresAt = expiresAt
	o.UpdatedAt = time.Now()
	return nil
}

// Extend extends the expiration time by the given duration
func (o *FlagOverride) Extend(duration time.Duration) error {
	if duration <= 0 {
		return shared.NewDomainError("INVALID_DURATION", "Duration must be positive")
	}
	var newExpiresAt time.Time
	if o.ExpiresAt != nil {
		newExpiresAt = o.ExpiresAt.Add(duration)
	} else {
		newExpiresAt = time.Now().Add(duration)
	}
	o.ExpiresAt = &newExpiresAt
	o.UpdatedAt = time.Now()
	return nil
}
