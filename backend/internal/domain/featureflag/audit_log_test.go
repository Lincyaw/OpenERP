package featureflag

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlagAuditLog(t *testing.T) {
	userID := uuid.New()

	t.Run("valid audit log", func(t *testing.T) {
		oldValue := map[string]any{"enabled": false}
		newValue := map[string]any{"enabled": true}

		log, err := NewFlagAuditLog(
			"test_flag",
			AuditActionEnabled,
			oldValue,
			newValue,
			&userID,
			"192.168.1.1",
			"Mozilla/5.0",
		)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, log.ID)
		assert.Equal(t, "test_flag", log.FlagKey)
		assert.Equal(t, AuditActionEnabled, log.Action)
		assert.Equal(t, oldValue, log.OldValue)
		assert.Equal(t, newValue, log.NewValue)
		assert.Equal(t, &userID, log.UserID)
		assert.Equal(t, "192.168.1.1", log.IPAddress)
		assert.Equal(t, "Mozilla/5.0", log.UserAgent)
		assert.False(t, log.CreatedAt.IsZero())
	})

	t.Run("valid audit log with nil values", func(t *testing.T) {
		log, err := NewFlagAuditLog(
			"test_flag",
			AuditActionCreated,
			nil,
			map[string]any{"key": "value"},
			nil,
			"",
			"",
		)

		require.NoError(t, err)
		assert.Equal(t, "test_flag", log.FlagKey)
		assert.Nil(t, log.OldValue)
		assert.NotNil(t, log.NewValue)
		assert.Nil(t, log.UserID)
		assert.Empty(t, log.IPAddress)
		assert.Empty(t, log.UserAgent)
	})

	t.Run("empty flag key", func(t *testing.T) {
		_, err := NewFlagAuditLog(
			"",
			AuditActionCreated,
			nil,
			nil,
			nil,
			"",
			"",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Flag key cannot be empty")
	})

	t.Run("invalid action", func(t *testing.T) {
		_, err := NewFlagAuditLog(
			"test_flag",
			AuditAction("invalid"),
			nil,
			nil,
			nil,
			"",
			"",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid audit action")
	})

	t.Run("all valid audit actions", func(t *testing.T) {
		actions := AllAuditActions()
		for _, action := range actions {
			log, err := NewFlagAuditLog(
				"test_flag",
				action,
				nil,
				nil,
				nil,
				"",
				"",
			)
			require.NoError(t, err, "Action %s should be valid", action)
			assert.Equal(t, action, log.Action)
		}
	})
}

func TestFlagAuditLog_Getters(t *testing.T) {
	userID := uuid.New()
	oldValue := map[string]any{"status": "disabled"}
	newValue := map[string]any{"status": "enabled"}

	log, err := NewFlagAuditLog(
		"test_flag",
		AuditActionEnabled,
		oldValue,
		newValue,
		&userID,
		"10.0.0.1",
		"Test Agent",
	)
	require.NoError(t, err)

	// Test getters
	assert.Equal(t, "test_flag", log.GetFlagKey())
	assert.Equal(t, AuditActionEnabled, log.GetAction())
	assert.Equal(t, &userID, log.GetUserID())
	assert.Equal(t, "10.0.0.1", log.GetIPAddress())
	assert.Equal(t, "Test Agent", log.GetUserAgent())
	assert.Equal(t, log.CreatedAt, log.GetTimestamp())
}

func TestFlagAuditLog_GetOldValue_Immutability(t *testing.T) {
	oldValue := map[string]any{"key": "original"}

	log, err := NewFlagAuditLog(
		"test_flag",
		AuditActionUpdated,
		oldValue,
		nil,
		nil,
		"",
		"",
	)
	require.NoError(t, err)

	// Get a copy
	retrieved := log.GetOldValue()

	// Modify the copy
	retrieved["key"] = "modified"

	// Original should remain unchanged
	assert.Equal(t, "original", log.OldValue["key"])
}

func TestFlagAuditLog_GetNewValue_Immutability(t *testing.T) {
	newValue := map[string]any{"key": "original"}

	log, err := NewFlagAuditLog(
		"test_flag",
		AuditActionUpdated,
		nil,
		newValue,
		nil,
		"",
		"",
	)
	require.NoError(t, err)

	// Get a copy
	retrieved := log.GetNewValue()

	// Modify the copy
	retrieved["key"] = "modified"

	// Original should remain unchanged
	assert.Equal(t, "original", log.NewValue["key"])
}

func TestFlagAuditLog_GetOldValue_NilHandling(t *testing.T) {
	log, err := NewFlagAuditLog(
		"test_flag",
		AuditActionCreated,
		nil,
		nil,
		nil,
		"",
		"",
	)
	require.NoError(t, err)

	// Should return empty map, not nil
	result := log.GetOldValue()
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestFlagAuditLog_GetNewValue_NilHandling(t *testing.T) {
	log, err := NewFlagAuditLog(
		"test_flag",
		AuditActionCreated,
		nil,
		nil,
		nil,
		"",
		"",
	)
	require.NoError(t, err)

	// Should return empty map, not nil
	result := log.GetNewValue()
	assert.NotNil(t, result)
	assert.Empty(t, result)
}
