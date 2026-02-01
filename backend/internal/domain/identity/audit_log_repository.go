package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditLogRepository defines the interface for audit log persistence
// Note: This repository only supports Create and Read operations.
// Audit logs are immutable and cannot be updated or deleted.
type AuditLogRepository interface {
	// Create creates a new audit log entry
	Create(ctx context.Context, log *AuditLog) error

	// FindByID finds an audit log by ID
	FindByID(ctx context.Context, id uuid.UUID) (*AuditLog, error)

	// FindAll returns audit logs with filtering and pagination
	FindAll(ctx context.Context, filter AuditLogFilter) ([]*AuditLog, int64, error)

	// FindByAdminUserID returns all audit logs for a specific admin user
	FindByAdminUserID(ctx context.Context, adminUserID uuid.UUID, filter AuditLogFilter) ([]*AuditLog, int64, error)

	// FindByTargetID returns all audit logs for a specific target entity
	FindByTargetID(ctx context.Context, targetType AuditTargetType, targetID uuid.UUID, filter AuditLogFilter) ([]*AuditLog, int64, error)

	// Count returns the total number of audit logs matching the filter
	Count(ctx context.Context, filter AuditLogFilter) (int64, error)
}

// AuditLogFilter contains filter options for querying audit logs
type AuditLogFilter struct {
	// Filter by admin user
	AdminUserID *uuid.UUID

	// Filter by action type
	Action *AuditAction

	// Filter by multiple actions
	Actions []AuditAction

	// Filter by target type
	TargetType *AuditTargetType

	// Filter by target ID
	TargetID *uuid.UUID

	// Filter by time range
	StartTime *time.Time
	EndTime   *time.Time

	// Filter by IP address
	IPAddress string

	// Pagination
	Page     int
	PageSize int

	// Sorting
	SortBy    string // "created_at" (default), "action", "target_type"
	SortOrder string // "asc" or "desc" (default)
}

// NewAuditLogFilter creates a new AuditLogFilter with default values
func NewAuditLogFilter() AuditLogFilter {
	return AuditLogFilter{
		Page:      1,
		PageSize:  20,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}

// WithAdminUserID sets the admin user ID filter
func (f AuditLogFilter) WithAdminUserID(adminUserID uuid.UUID) AuditLogFilter {
	f.AdminUserID = &adminUserID
	return f
}

// WithAction sets the action filter
func (f AuditLogFilter) WithAction(action AuditAction) AuditLogFilter {
	f.Action = &action
	return f
}

// WithActions sets multiple action filters
func (f AuditLogFilter) WithActions(actions ...AuditAction) AuditLogFilter {
	f.Actions = actions
	return f
}

// WithTargetType sets the target type filter
func (f AuditLogFilter) WithTargetType(targetType AuditTargetType) AuditLogFilter {
	f.TargetType = &targetType
	return f
}

// WithTargetID sets the target ID filter
func (f AuditLogFilter) WithTargetID(targetID uuid.UUID) AuditLogFilter {
	f.TargetID = &targetID
	return f
}

// WithTimeRange sets the time range filter
func (f AuditLogFilter) WithTimeRange(start, end time.Time) AuditLogFilter {
	f.StartTime = &start
	f.EndTime = &end
	return f
}

// WithStartTime sets the start time filter
func (f AuditLogFilter) WithStartTime(start time.Time) AuditLogFilter {
	f.StartTime = &start
	return f
}

// WithEndTime sets the end time filter
func (f AuditLogFilter) WithEndTime(end time.Time) AuditLogFilter {
	f.EndTime = &end
	return f
}

// WithIPAddress sets the IP address filter
func (f AuditLogFilter) WithIPAddress(ip string) AuditLogFilter {
	f.IPAddress = ip
	return f
}

// WithPagination sets pagination parameters
func (f AuditLogFilter) WithPagination(page, pageSize int) AuditLogFilter {
	f.Page = page
	f.PageSize = pageSize
	return f
}

// WithSorting sets sorting parameters
func (f AuditLogFilter) WithSorting(sortBy, sortOrder string) AuditLogFilter {
	f.SortBy = sortBy
	f.SortOrder = sortOrder
	return f
}

// Offset returns the offset for pagination
func (f AuditLogFilter) Offset() int {
	if f.Page <= 0 {
		return 0
	}
	return (f.Page - 1) * f.PageSize
}

// Limit returns the limit for pagination
func (f AuditLogFilter) Limit() int {
	if f.PageSize <= 0 {
		return 20
	}
	if f.PageSize > 100 {
		return 100
	}
	return f.PageSize
}

// GetSortBy returns the validated sort field
func (f AuditLogFilter) GetSortBy() string {
	switch f.SortBy {
	case "created_at", "action", "target_type", "admin_user_id":
		return f.SortBy
	default:
		return "created_at"
	}
}

// GetSortOrder returns the validated sort order
func (f AuditLogFilter) GetSortOrder() string {
	if f.SortOrder == "asc" {
		return "asc"
	}
	return "desc"
}
