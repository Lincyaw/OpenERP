package featureflag

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewFlagOverride(t *testing.T) {
	userID := uuid.New()
	targetID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	t.Run("valid user override", func(t *testing.T) {
		override, err := NewFlagOverride(
			"test_flag",
			OverrideTargetTypeUser,
			targetID,
			NewBooleanFlagValue(true),
			"Testing",
			&expiresAt,
			&userID,
		)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, override.ID)
		assert.Equal(t, "test_flag", override.FlagKey)
		assert.Equal(t, OverrideTargetTypeUser, override.TargetType)
		assert.Equal(t, targetID, override.TargetID)
		assert.True(t, override.Value.Enabled)
		assert.Equal(t, "Testing", override.Reason)
		assert.Equal(t, &expiresAt, override.ExpiresAt)
		assert.Equal(t, &userID, override.CreatedBy)
	})

	t.Run("valid tenant override", func(t *testing.T) {
		override, err := NewFlagOverride(
			"test_flag",
			OverrideTargetTypeTenant,
			targetID,
			NewBooleanFlagValue(false),
			"",
			nil,
			nil,
		)
		assert.NoError(t, err)
		assert.Equal(t, OverrideTargetTypeTenant, override.TargetType)
		assert.Nil(t, override.ExpiresAt)
		assert.Nil(t, override.CreatedBy)
	})

	t.Run("empty flag key", func(t *testing.T) {
		_, err := NewFlagOverride("", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
		assert.Error(t, err)
	})

	t.Run("invalid target type", func(t *testing.T) {
		_, err := NewFlagOverride("test_flag", OverrideTargetType("invalid"), targetID, NewBooleanFlagValue(true), "", nil, nil)
		assert.Error(t, err)
	})

	t.Run("empty target ID", func(t *testing.T) {
		_, err := NewFlagOverride("test_flag", OverrideTargetTypeUser, uuid.Nil, NewBooleanFlagValue(true), "", nil, nil)
		assert.Error(t, err)
	})

	t.Run("past expiration", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		_, err := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", &pastTime, nil)
		assert.Error(t, err)
	})
}

func TestFlagOverride_Getters(t *testing.T) {
	targetID := uuid.New()
	userID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	override, _ := NewFlagOverride(
		"test_flag",
		OverrideTargetTypeUser,
		targetID,
		NewBooleanFlagValue(true),
		"Test reason",
		&expiresAt,
		&userID,
	)

	assert.Equal(t, "test_flag", override.GetFlagKey())
	assert.Equal(t, OverrideTargetTypeUser, override.GetTargetType())
	assert.Equal(t, targetID, override.GetTargetID())
	assert.True(t, override.GetValue().Enabled)
	assert.Equal(t, "Test reason", override.GetReason())
	assert.Equal(t, &expiresAt, override.GetExpiresAt())
	assert.Equal(t, &userID, override.GetCreatedBy())
}

func TestFlagOverride_IsExpired(t *testing.T) {
	targetID := uuid.New()

	t.Run("not expired", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", &futureTime, nil)
		assert.False(t, override.IsExpired())
	})

	t.Run("no expiration", func(t *testing.T) {
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
		assert.False(t, override.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
		pastTime := time.Now().Add(-1 * time.Hour)
		override.ExpiresAt = &pastTime
		assert.True(t, override.IsExpired())
	})
}

func TestFlagOverride_IsActive(t *testing.T) {
	targetID := uuid.New()

	t.Run("active", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", &futureTime, nil)
		assert.True(t, override.IsActive())
	})

	t.Run("expired", func(t *testing.T) {
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
		pastTime := time.Now().Add(-1 * time.Hour)
		override.ExpiresAt = &pastTime
		assert.False(t, override.IsActive())
	})
}

func TestFlagOverride_IsForUser(t *testing.T) {
	targetID := uuid.New()

	userOverride, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
	assert.True(t, userOverride.IsForUser())
	assert.False(t, userOverride.IsForTenant())

	tenantOverride, _ := NewFlagOverride("test_flag", OverrideTargetTypeTenant, targetID, NewBooleanFlagValue(true), "", nil, nil)
	assert.False(t, tenantOverride.IsForUser())
	assert.True(t, tenantOverride.IsForTenant())
}

func TestFlagOverride_MatchesTarget(t *testing.T) {
	targetID := uuid.New()
	otherID := uuid.New()

	override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)

	assert.True(t, override.MatchesTarget(OverrideTargetTypeUser, targetID))
	assert.False(t, override.MatchesTarget(OverrideTargetTypeTenant, targetID))
	assert.False(t, override.MatchesTarget(OverrideTargetTypeUser, otherID))
}

func TestFlagOverride_UpdateValue(t *testing.T) {
	targetID := uuid.New()
	override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)

	originalUpdatedAt := override.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	override.UpdateValue(NewBooleanFlagValue(false))
	assert.False(t, override.Value.Enabled)
	assert.True(t, override.UpdatedAt.After(originalUpdatedAt))
}

func TestFlagOverride_UpdateReason(t *testing.T) {
	targetID := uuid.New()
	override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)

	override.UpdateReason("New reason")
	assert.Equal(t, "New reason", override.Reason)
}

func TestFlagOverride_UpdateExpiresAt(t *testing.T) {
	targetID := uuid.New()
	override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)

	futureTime := time.Now().Add(48 * time.Hour)
	err := override.UpdateExpiresAt(&futureTime)
	assert.NoError(t, err)
	assert.Equal(t, &futureTime, override.ExpiresAt)

	// Set to nil
	err = override.UpdateExpiresAt(nil)
	assert.NoError(t, err)
	assert.Nil(t, override.ExpiresAt)

	// Past time should fail
	pastTime := time.Now().Add(-1 * time.Hour)
	err = override.UpdateExpiresAt(&pastTime)
	assert.Error(t, err)
}

func TestFlagOverride_Extend(t *testing.T) {
	targetID := uuid.New()

	t.Run("extend with existing expiration", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", &futureTime, nil)

		err := override.Extend(24 * time.Hour)
		assert.NoError(t, err)
		assert.True(t, override.ExpiresAt.After(futureTime))
	})

	t.Run("extend without existing expiration", func(t *testing.T) {
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)

		err := override.Extend(24 * time.Hour)
		assert.NoError(t, err)
		assert.NotNil(t, override.ExpiresAt)
		assert.True(t, override.ExpiresAt.After(time.Now()))
	})

	t.Run("zero duration", func(t *testing.T) {
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
		err := override.Extend(0)
		assert.Error(t, err)
	})

	t.Run("negative duration", func(t *testing.T) {
		override, _ := NewFlagOverride("test_flag", OverrideTargetTypeUser, targetID, NewBooleanFlagValue(true), "", nil, nil)
		err := override.Extend(-1 * time.Hour)
		assert.Error(t, err)
	})
}
