package identity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewAuditLogFilter(t *testing.T) {
	filter := NewAuditLogFilter()

	assert.Equal(t, 1, filter.Page)
	assert.Equal(t, 20, filter.PageSize)
	assert.Equal(t, "created_at", filter.SortBy)
	assert.Equal(t, "desc", filter.SortOrder)
}

func TestAuditLogFilter_WithAdminUserID(t *testing.T) {
	adminUserID := uuid.New()
	filter := NewAuditLogFilter().WithAdminUserID(adminUserID)

	assert.NotNil(t, filter.AdminUserID)
	assert.Equal(t, adminUserID, *filter.AdminUserID)
}

func TestAuditLogFilter_WithAction(t *testing.T) {
	filter := NewAuditLogFilter().WithAction(AuditActionTenantCreate)

	assert.NotNil(t, filter.Action)
	assert.Equal(t, AuditActionTenantCreate, *filter.Action)
}

func TestAuditLogFilter_WithActions(t *testing.T) {
	actions := []AuditAction{AuditActionTenantCreate, AuditActionTenantUpdate}
	filter := NewAuditLogFilter().WithActions(actions...)

	assert.Equal(t, actions, filter.Actions)
}

func TestAuditLogFilter_WithTargetType(t *testing.T) {
	filter := NewAuditLogFilter().WithTargetType(AuditTargetTenant)

	assert.NotNil(t, filter.TargetType)
	assert.Equal(t, AuditTargetTenant, *filter.TargetType)
}

func TestAuditLogFilter_WithTargetID(t *testing.T) {
	targetID := uuid.New()
	filter := NewAuditLogFilter().WithTargetID(targetID)

	assert.NotNil(t, filter.TargetID)
	assert.Equal(t, targetID, *filter.TargetID)
}

func TestAuditLogFilter_WithTimeRange(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	filter := NewAuditLogFilter().WithTimeRange(start, end)

	assert.NotNil(t, filter.StartTime)
	assert.NotNil(t, filter.EndTime)
	assert.Equal(t, start, *filter.StartTime)
	assert.Equal(t, end, *filter.EndTime)
}

func TestAuditLogFilter_WithStartTime(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	filter := NewAuditLogFilter().WithStartTime(start)

	assert.NotNil(t, filter.StartTime)
	assert.Equal(t, start, *filter.StartTime)
	assert.Nil(t, filter.EndTime)
}

func TestAuditLogFilter_WithEndTime(t *testing.T) {
	end := time.Now()
	filter := NewAuditLogFilter().WithEndTime(end)

	assert.Nil(t, filter.StartTime)
	assert.NotNil(t, filter.EndTime)
	assert.Equal(t, end, *filter.EndTime)
}

func TestAuditLogFilter_WithIPAddress(t *testing.T) {
	filter := NewAuditLogFilter().WithIPAddress("192.168.1.1")

	assert.Equal(t, "192.168.1.1", filter.IPAddress)
}

func TestAuditLogFilter_WithPagination(t *testing.T) {
	filter := NewAuditLogFilter().WithPagination(3, 50)

	assert.Equal(t, 3, filter.Page)
	assert.Equal(t, 50, filter.PageSize)
}

func TestAuditLogFilter_WithSorting(t *testing.T) {
	filter := NewAuditLogFilter().WithSorting("action", "asc")

	assert.Equal(t, "action", filter.SortBy)
	assert.Equal(t, "asc", filter.SortOrder)
}

func TestAuditLogFilter_Offset(t *testing.T) {
	t.Run("calculates offset correctly", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(3, 20)
		assert.Equal(t, 40, filter.Offset())
	})

	t.Run("returns 0 for page 1", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(1, 20)
		assert.Equal(t, 0, filter.Offset())
	})

	t.Run("returns 0 for page 0", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(0, 20)
		assert.Equal(t, 0, filter.Offset())
	})

	t.Run("returns 0 for negative page", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(-1, 20)
		assert.Equal(t, 0, filter.Offset())
	})
}

func TestAuditLogFilter_Limit(t *testing.T) {
	t.Run("returns page size", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(1, 50)
		assert.Equal(t, 50, filter.Limit())
	})

	t.Run("returns default for 0 page size", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(1, 0)
		assert.Equal(t, 20, filter.Limit())
	})

	t.Run("returns default for negative page size", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(1, -10)
		assert.Equal(t, 20, filter.Limit())
	})

	t.Run("caps at 100", func(t *testing.T) {
		filter := NewAuditLogFilter().WithPagination(1, 200)
		assert.Equal(t, 100, filter.Limit())
	})
}

func TestAuditLogFilter_GetSortBy(t *testing.T) {
	t.Run("returns valid sort field", func(t *testing.T) {
		validFields := []string{"created_at", "action", "target_type", "admin_user_id"}
		for _, field := range validFields {
			filter := NewAuditLogFilter().WithSorting(field, "asc")
			assert.Equal(t, field, filter.GetSortBy())
		}
	})

	t.Run("returns default for invalid field", func(t *testing.T) {
		filter := NewAuditLogFilter().WithSorting("invalid_field", "asc")
		assert.Equal(t, "created_at", filter.GetSortBy())
	})
}

func TestAuditLogFilter_GetSortOrder(t *testing.T) {
	t.Run("returns asc when specified", func(t *testing.T) {
		filter := NewAuditLogFilter().WithSorting("created_at", "asc")
		assert.Equal(t, "asc", filter.GetSortOrder())
	})

	t.Run("returns desc for any other value", func(t *testing.T) {
		filter := NewAuditLogFilter().WithSorting("created_at", "invalid")
		assert.Equal(t, "desc", filter.GetSortOrder())
	})

	t.Run("returns desc by default", func(t *testing.T) {
		filter := NewAuditLogFilter()
		assert.Equal(t, "desc", filter.GetSortOrder())
	})
}

func TestAuditLogFilter_Chaining(t *testing.T) {
	adminUserID := uuid.New()
	targetID := uuid.New()
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	filter := NewAuditLogFilter().
		WithAdminUserID(adminUserID).
		WithAction(AuditActionTenantCreate).
		WithTargetType(AuditTargetTenant).
		WithTargetID(targetID).
		WithTimeRange(start, end).
		WithIPAddress("192.168.1.1").
		WithPagination(2, 30).
		WithSorting("action", "asc")

	assert.Equal(t, &adminUserID, filter.AdminUserID)
	assert.Equal(t, AuditActionTenantCreate, *filter.Action)
	assert.Equal(t, AuditTargetTenant, *filter.TargetType)
	assert.Equal(t, &targetID, filter.TargetID)
	assert.Equal(t, start, *filter.StartTime)
	assert.Equal(t, end, *filter.EndTime)
	assert.Equal(t, "192.168.1.1", filter.IPAddress)
	assert.Equal(t, 2, filter.Page)
	assert.Equal(t, 30, filter.PageSize)
	assert.Equal(t, "action", filter.SortBy)
	assert.Equal(t, "asc", filter.SortOrder)
}
