package identity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLog(t *testing.T) {
	adminUserID := uuid.New()

	t.Run("creates audit log with required fields", func(t *testing.T) {
		log, err := NewAuditLog(adminUserID, AuditActionTenantCreate, AuditTargetTenant).Build()

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, log.ID)
		assert.Equal(t, adminUserID, log.AdminUserID)
		assert.Equal(t, AuditActionTenantCreate, log.Action)
		assert.Equal(t, AuditTargetTenant, log.TargetType)
		assert.Nil(t, log.TargetID)
		assert.Nil(t, log.OldValue)
		assert.Nil(t, log.NewValue)
		assert.Empty(t, log.IPAddress)
		assert.Empty(t, log.UserAgent)
		assert.False(t, log.CreatedAt.IsZero())
	})

	t.Run("creates audit log with all optional fields", func(t *testing.T) {
		targetID := uuid.New()
		oldValue := map[string]interface{}{"name": "Old Name"}
		newValue := map[string]interface{}{"name": "New Name"}

		log, err := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithTargetID(targetID).
			WithOldValue(oldValue).
			WithNewValue(newValue).
			WithIPAddress("192.168.1.1").
			WithUserAgent("Mozilla/5.0").
			Build()

		require.NoError(t, err)
		assert.Equal(t, &targetID, log.TargetID)
		assert.Equal(t, oldValue, log.OldValue)
		assert.Equal(t, newValue, log.NewValue)
		assert.Equal(t, "192.168.1.1", log.IPAddress)
		assert.Equal(t, "Mozilla/5.0", log.UserAgent)
	})

	t.Run("fails with empty admin user ID", func(t *testing.T) {
		_, err := NewAuditLog(uuid.Nil, AuditActionTenantCreate, AuditTargetTenant).Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Admin user ID cannot be empty")
	})

	t.Run("fails with empty action", func(t *testing.T) {
		_, err := NewAuditLog(adminUserID, "", AuditTargetTenant).Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Action cannot be empty")
	})

	t.Run("fails with empty target type", func(t *testing.T) {
		_, err := NewAuditLog(adminUserID, AuditActionTenantCreate, "").Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Target type cannot be empty")
	})

	t.Run("fails with invalid action", func(t *testing.T) {
		_, err := NewAuditLog(adminUserID, "INVALID_ACTION", AuditTargetTenant).Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid audit action")
	})

	t.Run("fails with invalid target type", func(t *testing.T) {
		_, err := NewAuditLog(adminUserID, AuditActionTenantCreate, "invalid_target").Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid target type")
	})
}

func TestAuditLogBuilder_WithOldValueFromEntity(t *testing.T) {
	adminUserID := uuid.New()

	t.Run("serializes struct to map", func(t *testing.T) {
		type TestEntity struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		entity := TestEntity{Name: "Test", Value: 42}

		log, err := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithOldValueFromEntity(entity).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, log.OldValue)
		assert.Equal(t, "Test", log.OldValue["name"])
		assert.Equal(t, float64(42), log.OldValue["value"]) // JSON numbers are float64
	})

	t.Run("handles nil entity", func(t *testing.T) {
		log, err := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithOldValueFromEntity(nil).
			Build()

		require.NoError(t, err)
		assert.Nil(t, log.OldValue)
	})
}

func TestAuditLogBuilder_WithNewValueFromEntity(t *testing.T) {
	adminUserID := uuid.New()

	t.Run("serializes struct to map", func(t *testing.T) {
		type TestEntity struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		entity := TestEntity{Name: "Test", Value: 42}

		log, err := NewAuditLog(adminUserID, AuditActionTenantCreate, AuditTargetTenant).
			WithNewValueFromEntity(entity).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, log.NewValue)
		assert.Equal(t, "Test", log.NewValue["name"])
		assert.Equal(t, float64(42), log.NewValue["value"])
	})
}

func TestAuditAction_IsValid(t *testing.T) {
	validActions := []AuditAction{
		AuditActionTenantCreate,
		AuditActionTenantUpdate,
		AuditActionTenantDelete,
		AuditActionTenantSuspend,
		AuditActionTenantActivate,
		AuditActionSubscriptionChange,
		AuditActionQuotaUpdate,
		AuditActionUserCreate,
		AuditActionUserUpdate,
		AuditActionUserDelete,
		AuditActionUserSuspend,
		AuditActionUserActivate,
		AuditActionRoleCreate,
		AuditActionRoleUpdate,
		AuditActionRoleDelete,
		AuditActionPermissionGrant,
		AuditActionPermissionRevoke,
		AuditActionSystemConfigChange,
	}

	for _, action := range validActions {
		t.Run(string(action), func(t *testing.T) {
			assert.True(t, action.IsValid())
		})
	}

	t.Run("invalid action", func(t *testing.T) {
		assert.False(t, AuditAction("INVALID").IsValid())
	})
}

func TestAuditTargetType_IsValid(t *testing.T) {
	validTargets := []AuditTargetType{
		AuditTargetTenant,
		AuditTargetUser,
		AuditTargetRole,
		AuditTargetPermission,
		AuditTargetSubscription,
		AuditTargetQuota,
		AuditTargetSystemConfig,
	}

	for _, target := range validTargets {
		t.Run(string(target), func(t *testing.T) {
			assert.True(t, target.IsValid())
		})
	}

	t.Run("invalid target type", func(t *testing.T) {
		assert.False(t, AuditTargetType("invalid").IsValid())
	})
}

func TestAuditLog_GetOldValueJSON(t *testing.T) {
	adminUserID := uuid.New()

	t.Run("returns JSON for non-nil value", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithOldValue(map[string]interface{}{"name": "Test"}).
			Build()

		json, err := log.GetOldValueJSON()

		require.NoError(t, err)
		assert.Contains(t, string(json), "name")
		assert.Contains(t, string(json), "Test")
	})

	t.Run("returns nil for nil value", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantCreate, AuditTargetTenant).Build()

		json, err := log.GetOldValueJSON()

		require.NoError(t, err)
		assert.Nil(t, json)
	})
}

func TestAuditLog_GetNewValueJSON(t *testing.T) {
	adminUserID := uuid.New()

	t.Run("returns JSON for non-nil value", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantCreate, AuditTargetTenant).
			WithNewValue(map[string]interface{}{"name": "Test"}).
			Build()

		json, err := log.GetNewValueJSON()

		require.NoError(t, err)
		assert.Contains(t, string(json), "name")
		assert.Contains(t, string(json), "Test")
	})

	t.Run("returns nil for nil value", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantCreate, AuditTargetTenant).Build()

		json, err := log.GetNewValueJSON()

		require.NoError(t, err)
		assert.Nil(t, json)
	})
}

func TestAuditLog_HasChanges(t *testing.T) {
	adminUserID := uuid.New()

	t.Run("returns true when old value exists", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithOldValue(map[string]interface{}{"name": "Old"}).
			Build()

		assert.True(t, log.HasChanges())
	})

	t.Run("returns true when new value exists", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantCreate, AuditTargetTenant).
			WithNewValue(map[string]interface{}{"name": "New"}).
			Build()

		assert.True(t, log.HasChanges())
	})

	t.Run("returns false when no values exist", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantDelete, AuditTargetTenant).Build()

		assert.False(t, log.HasChanges())
	})
}

func TestAuditLog_GetChangedFields(t *testing.T) {
	adminUserID := uuid.New()

	t.Run("returns changed fields", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithOldValue(map[string]interface{}{"name": "Old", "status": "active"}).
			WithNewValue(map[string]interface{}{"name": "New", "status": "active"}).
			Build()

		changed := log.GetChangedFields()

		assert.Contains(t, changed, "name")
		assert.NotContains(t, changed, "status")
	})

	t.Run("returns added fields", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithOldValue(map[string]interface{}{"name": "Test"}).
			WithNewValue(map[string]interface{}{"name": "Test", "description": "New field"}).
			Build()

		changed := log.GetChangedFields()

		assert.Contains(t, changed, "description")
	})

	t.Run("returns removed fields", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantUpdate, AuditTargetTenant).
			WithOldValue(map[string]interface{}{"name": "Test", "description": "Old field"}).
			WithNewValue(map[string]interface{}{"name": "Test"}).
			Build()

		changed := log.GetChangedFields()

		assert.Contains(t, changed, "description")
	})

	t.Run("returns nil when no values", func(t *testing.T) {
		log, _ := NewAuditLog(adminUserID, AuditActionTenantDelete, AuditTargetTenant).Build()

		changed := log.GetChangedFields()

		assert.Nil(t, changed)
	})
}

func TestAuditAction_String(t *testing.T) {
	assert.Equal(t, "TENANT_CREATE", AuditActionTenantCreate.String())
	assert.Equal(t, "TENANT_UPDATE", AuditActionTenantUpdate.String())
}

func TestAuditTargetType_String(t *testing.T) {
	assert.Equal(t, "tenant", AuditTargetTenant.String())
	assert.Equal(t, "user", AuditTargetUser.String())
}
