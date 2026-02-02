package identity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantStatusChangeType_IsValid(t *testing.T) {
	t.Run("valid change types", func(t *testing.T) {
		assert.True(t, TenantStatusChangeTypeSuspend.IsValid())
		assert.True(t, TenantStatusChangeTypeActivate.IsValid())
		assert.True(t, TenantStatusChangeTypeDeactivate.IsValid())
	})

	t.Run("invalid change type", func(t *testing.T) {
		invalidType := TenantStatusChangeType("invalid")
		assert.False(t, invalidType.IsValid())
	})
}

func TestTenantStatusChangeType_String(t *testing.T) {
	assert.Equal(t, "suspend", TenantStatusChangeTypeSuspend.String())
	assert.Equal(t, "activate", TenantStatusChangeTypeActivate.String())
	assert.Equal(t, "deactivate", TenantStatusChangeTypeDeactivate.String())
}

func TestNewTenantStatusHistory(t *testing.T) {
	tenantID := uuid.New()
	changedByUserID := uuid.New()

	t.Run("creates suspension history", func(t *testing.T) {
		reactivateAt := time.Now().Add(24 * time.Hour)
		history, err := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeSuspend).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			WithReason("Payment overdue").
			WithScheduledReactivateAt(&reactivateAt).
			WithIPAddress("127.0.0.1").
			WithUserAgent("test-agent").
			Build()

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, history.ID)
		assert.Equal(t, tenantID, history.TenantID)
		assert.Equal(t, changedByUserID, history.ChangedByUserID)
		assert.Equal(t, TenantStatusChangeTypeSuspend, history.ChangeType)
		assert.Equal(t, TenantStatusActive, history.OldStatus)
		assert.Equal(t, TenantStatusSuspended, history.NewStatus)
		assert.Equal(t, "Payment overdue", history.Reason)
		assert.NotNil(t, history.ScheduledReactivateAt)
		assert.Equal(t, "127.0.0.1", history.IPAddress)
		assert.Equal(t, "test-agent", history.UserAgent)
	})

	t.Run("creates activation history", func(t *testing.T) {
		history, err := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeActivate).
			WithStatusChange(TenantStatusSuspended, TenantStatusActive).
			WithIPAddress("127.0.0.1").
			Build()

		require.NoError(t, err)
		assert.Equal(t, TenantStatusChangeTypeActivate, history.ChangeType)
		assert.Equal(t, TenantStatusSuspended, history.OldStatus)
		assert.Equal(t, TenantStatusActive, history.NewStatus)
		assert.Empty(t, history.Reason)
		assert.Nil(t, history.ScheduledReactivateAt)
	})

	t.Run("fails with empty tenant ID", func(t *testing.T) {
		_, err := NewTenantStatusHistory(uuid.Nil, changedByUserID, TenantStatusChangeTypeSuspend).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Tenant ID cannot be empty")
	})

	t.Run("fails with empty changed by user ID", func(t *testing.T) {
		_, err := NewTenantStatusHistory(tenantID, uuid.Nil, TenantStatusChangeTypeSuspend).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Changed by user ID cannot be empty")
	})

	t.Run("fails with invalid change type", func(t *testing.T) {
		_, err := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeType("invalid")).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid status change type")
	})

	t.Run("fails without status change", func(t *testing.T) {
		_, err := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeSuspend).
			Build()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Status change requires both old and new status")
	})
}

func TestTenantStatusHistory_IsSuspension(t *testing.T) {
	tenantID := uuid.New()
	changedByUserID := uuid.New()

	t.Run("returns true for suspension", func(t *testing.T) {
		history, _ := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeSuspend).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			Build()

		assert.True(t, history.IsSuspension())
	})

	t.Run("returns false for activation", func(t *testing.T) {
		history, _ := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeActivate).
			WithStatusChange(TenantStatusSuspended, TenantStatusActive).
			Build()

		assert.False(t, history.IsSuspension())
	})
}

func TestTenantStatusHistory_IsActivation(t *testing.T) {
	tenantID := uuid.New()
	changedByUserID := uuid.New()

	t.Run("returns true for activation", func(t *testing.T) {
		history, _ := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeActivate).
			WithStatusChange(TenantStatusSuspended, TenantStatusActive).
			Build()

		assert.True(t, history.IsActivation())
	})

	t.Run("returns false for suspension", func(t *testing.T) {
		history, _ := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeSuspend).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			Build()

		assert.False(t, history.IsActivation())
	})
}

func TestTenantStatusHistory_HasScheduledReactivation(t *testing.T) {
	tenantID := uuid.New()
	changedByUserID := uuid.New()

	t.Run("returns true when scheduled", func(t *testing.T) {
		reactivateAt := time.Now().Add(24 * time.Hour)
		history, _ := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeSuspend).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			WithScheduledReactivateAt(&reactivateAt).
			Build()

		assert.True(t, history.HasScheduledReactivation())
	})

	t.Run("returns false when not scheduled", func(t *testing.T) {
		history, _ := NewTenantStatusHistory(tenantID, changedByUserID, TenantStatusChangeTypeSuspend).
			WithStatusChange(TenantStatusActive, TenantStatusSuspended).
			Build()

		assert.False(t, history.HasScheduledReactivation())
	})
}
