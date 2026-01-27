package featureflag

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// FeatureFlagRepository defines the interface for feature flag persistence.
//
// Feature flags are GLOBAL resources (not tenant-scoped). They control
// application behavior across the entire system. Tenant-specific behavior
// is achieved through FlagOverride entities.
type FeatureFlagRepository interface {
	// Create creates a new feature flag
	// Returns an error if a flag with the same key already exists
	Create(ctx context.Context, flag *FeatureFlag) error

	// Update updates an existing feature flag
	// Uses optimistic locking via the version field.
	// IMPORTANT: The caller must increment the version BEFORE calling Update.
	// The update will only succeed if the database version matches (flag.Version - 1).
	// Returns OPTIMISTIC_LOCK_FAILED error if version mismatch occurs.
	Update(ctx context.Context, flag *FeatureFlag) error

	// FindByKey finds a feature flag by its unique key
	// Returns shared.ErrNotFound if not found
	FindByKey(ctx context.Context, key string) (*FeatureFlag, error)

	// FindByID finds a feature flag by its ID
	// Returns shared.ErrNotFound if not found
	FindByID(ctx context.Context, id uuid.UUID) (*FeatureFlag, error)

	// FindAll retrieves all feature flags with optional filtering
	FindAll(ctx context.Context, filter shared.Filter) ([]FeatureFlag, error)

	// FindByStatus finds all feature flags with a specific status
	FindByStatus(ctx context.Context, status FlagStatus, filter shared.Filter) ([]FeatureFlag, error)

	// FindByTags finds all feature flags that have any of the specified tags
	FindByTags(ctx context.Context, tags []string, filter shared.Filter) ([]FeatureFlag, error)

	// FindByType finds all feature flags of a specific type
	FindByType(ctx context.Context, flagType FlagType, filter shared.Filter) ([]FeatureFlag, error)

	// FindEnabled finds all enabled feature flags
	FindEnabled(ctx context.Context, filter shared.Filter) ([]FeatureFlag, error)

	// Delete deletes a feature flag by its key
	// Note: Consider archiving instead of hard deletion in production
	Delete(ctx context.Context, key string) error

	// ExistsByKey checks if a feature flag with the given key exists
	ExistsByKey(ctx context.Context, key string) (bool, error)

	// Count counts feature flags matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountByStatus counts feature flags by status
	CountByStatus(ctx context.Context, status FlagStatus) (int64, error)
}

// FlagOverrideRepository defines the interface for flag override persistence.
//
// Overrides allow tenant-specific or user-specific feature flag values.
// They take precedence over the default flag value when evaluating flags.
type FlagOverrideRepository interface {
	// Create creates a new flag override
	// Returns an error if an override for the same flag/target combination exists
	Create(ctx context.Context, override *FlagOverride) error

	// Update updates an existing flag override
	Update(ctx context.Context, override *FlagOverride) error

	// FindByID finds a flag override by its ID
	// Returns shared.ErrNotFound if not found
	FindByID(ctx context.Context, id uuid.UUID) (*FlagOverride, error)

	// FindByFlagKey finds all overrides for a specific flag
	FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]FlagOverride, error)

	// FindByTarget finds all overrides for a specific target (user or tenant)
	FindByTarget(ctx context.Context, targetType OverrideTargetType, targetID uuid.UUID, filter shared.Filter) ([]FlagOverride, error)

	// FindForEvaluation finds the most specific override for flag evaluation.
	// Priority: user override > tenant override
	// Only returns active (non-expired) overrides.
	// Returns nil, nil if no override applies.
	FindForEvaluation(ctx context.Context, flagKey string, tenantID, userID *uuid.UUID) (*FlagOverride, error)

	// FindByFlagKeyAndTarget finds a specific override by flag key and target
	// Returns shared.ErrNotFound if not found
	FindByFlagKeyAndTarget(ctx context.Context, flagKey string, targetType OverrideTargetType, targetID uuid.UUID) (*FlagOverride, error)

	// FindExpired finds all expired overrides
	FindExpired(ctx context.Context, filter shared.Filter) ([]FlagOverride, error)

	// FindActive finds all active (non-expired) overrides
	FindActive(ctx context.Context, filter shared.Filter) ([]FlagOverride, error)

	// Delete deletes a flag override by its ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByFlagKey deletes all overrides for a specific flag
	DeleteByFlagKey(ctx context.Context, flagKey string) (int64, error)

	// DeleteExpired deletes all expired overrides and returns the count of deleted records
	DeleteExpired(ctx context.Context) (int64, error)

	// Count counts overrides matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountByFlagKey counts overrides for a specific flag
	CountByFlagKey(ctx context.Context, flagKey string) (int64, error)
}

// FlagAuditLogRepository defines the interface for flag audit log persistence.
//
// Audit logs provide an immutable record of all changes to feature flags
// and overrides. They support compliance, debugging, and rollback analysis.
type FlagAuditLogRepository interface {
	// Create creates a new audit log entry
	// Note: Audit logs are append-only; no update or delete operations
	Create(ctx context.Context, log *FlagAuditLog) error

	// CreateBatch creates multiple audit log entries in a single transaction
	CreateBatch(ctx context.Context, logs []*FlagAuditLog) error

	// FindByFlagKey finds audit logs for a specific flag with pagination
	FindByFlagKey(ctx context.Context, flagKey string, filter shared.Filter) ([]FlagAuditLog, error)

	// FindByUserID finds audit logs for actions performed by a specific user
	FindByUserID(ctx context.Context, userID uuid.UUID, filter shared.Filter) ([]FlagAuditLog, error)

	// FindByAction finds audit logs for a specific action type
	FindByAction(ctx context.Context, action AuditAction, filter shared.Filter) ([]FlagAuditLog, error)

	// FindAll finds all audit logs with pagination
	FindAll(ctx context.Context, filter shared.Filter) ([]FlagAuditLog, error)

	// Count counts audit logs matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountByFlagKey counts audit logs for a specific flag
	CountByFlagKey(ctx context.Context, flagKey string) (int64, error)
}

// FlagFilter extends shared.Filter with feature-flag-specific filters
type FlagFilter struct {
	shared.Filter
	Status   *FlagStatus
	Type     *FlagType
	Tags     []string
	Enabled  *bool
	Archived *bool
}

// OverrideFilter extends shared.Filter with override-specific filters
type OverrideFilter struct {
	shared.Filter
	FlagKey    string
	TargetType *OverrideTargetType
	TargetID   *uuid.UUID
	Expired    *bool
	Active     *bool
}

// AuditLogFilter extends shared.Filter with audit-log-specific filters
type AuditLogFilter struct {
	shared.Filter
	FlagKey string
	UserID  *uuid.UUID
	Action  *AuditAction
}
