package identity

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates user with valid username and password", func(t *testing.T) {
		user, err := NewUser(tenantID, "testuser", "Password123")

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, tenantID, user.TenantID)
		assert.Equal(t, "testuser", user.Username)
		assert.NotEmpty(t, user.PasswordHash)
		assert.Equal(t, UserStatusPending, user.Status)
		assert.Empty(t, user.RoleIDs)
		assert.NotNil(t, user.PasswordChangedAt)

		// Should have domain event
		events := user.GetDomainEvents()
		assert.Len(t, events, 1)
		_, ok := events[0].(*UserCreatedEvent)
		assert.True(t, ok)
	})

	t.Run("normalizes username to lowercase", func(t *testing.T) {
		user, err := NewUser(tenantID, "TestUser", "Password123")

		require.NoError(t, err)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("trims username whitespace", func(t *testing.T) {
		user, err := NewUser(tenantID, "  testuser  ", "Password123")

		require.NoError(t, err)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("fails with empty username", func(t *testing.T) {
		_, err := NewUser(tenantID, "", "Password123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails with short username", func(t *testing.T) {
		_, err := NewUser(tenantID, "ab", "Password123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 3 characters")
	})

	t.Run("fails with invalid username characters", func(t *testing.T) {
		_, err := NewUser(tenantID, "test@user", "Password123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only contain letters")
	})

	t.Run("fails with empty password", func(t *testing.T) {
		_, err := NewUser(tenantID, "testuser", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails with short password", func(t *testing.T) {
		_, err := NewUser(tenantID, "testuser", "Pass1")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 8 characters")
	})

	t.Run("fails with password without letters", func(t *testing.T) {
		_, err := NewUser(tenantID, "testuser", "12345678")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one letter")
	})

	t.Run("fails with password without numbers", func(t *testing.T) {
		_, err := NewUser(tenantID, "testuser", "Password")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one letter and one number")
	})
}

func TestNewActiveUser(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates active user", func(t *testing.T) {
		user, err := NewActiveUser(tenantID, "testuser", "Password123")

		require.NoError(t, err)
		assert.Equal(t, UserStatusActive, user.Status)
	})
}

func TestUser_SetEmail(t *testing.T) {
	tenantID := uuid.New()
	user, _ := NewUser(tenantID, "testuser", "Password123")
	user.ClearDomainEvents()

	t.Run("sets valid email", func(t *testing.T) {
		err := user.SetEmail("test@example.com")

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("normalizes email to lowercase", func(t *testing.T) {
		err := user.SetEmail("Test@Example.COM")

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("allows empty email", func(t *testing.T) {
		err := user.SetEmail("")

		require.NoError(t, err)
		assert.Empty(t, user.Email)
	})

	t.Run("fails with invalid email format", func(t *testing.T) {
		err := user.SetEmail("invalid-email")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid email")
	})
}

func TestUser_SetPhone(t *testing.T) {
	tenantID := uuid.New()
	user, _ := NewUser(tenantID, "testuser", "Password123")

	t.Run("sets valid phone", func(t *testing.T) {
		err := user.SetPhone("+86 138 1234 5678")

		require.NoError(t, err)
		assert.Equal(t, "+86 138 1234 5678", user.Phone)
	})

	t.Run("allows empty phone", func(t *testing.T) {
		err := user.SetPhone("")

		require.NoError(t, err)
		assert.Empty(t, user.Phone)
	})
}

func TestUser_SetDisplayName(t *testing.T) {
	tenantID := uuid.New()
	user, _ := NewUser(tenantID, "testuser", "Password123")

	t.Run("sets display name", func(t *testing.T) {
		err := user.SetDisplayName("Test User")

		require.NoError(t, err)
		assert.Equal(t, "Test User", user.DisplayName)
	})
}

func TestUser_PasswordOperations(t *testing.T) {
	tenantID := uuid.New()

	t.Run("verifies correct password", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		assert.True(t, user.VerifyPassword("Password123"))
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		assert.False(t, user.VerifyPassword("WrongPassword1"))
	})

	t.Run("changes password with correct old password", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		user.ClearDomainEvents()

		err := user.ChangePassword("Password123", "NewPassword456")

		require.NoError(t, err)
		assert.True(t, user.VerifyPassword("NewPassword456"))
		assert.False(t, user.VerifyPassword("Password123"))

		// Should have password changed event
		events := user.GetDomainEvents()
		assert.Len(t, events, 1)
		_, ok := events[0].(*UserPasswordChangedEvent)
		assert.True(t, ok)
	})

	t.Run("fails to change password with wrong old password", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		err := user.ChangePassword("WrongPassword1", "NewPassword456")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incorrect")
	})

	t.Run("sets password without old password check", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		err := user.SetPassword("NewPassword456")

		require.NoError(t, err)
		assert.True(t, user.VerifyPassword("NewPassword456"))
	})

	t.Run("force password change flag", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		assert.False(t, user.MustChangePassword)

		user.ForcePasswordChange()

		assert.True(t, user.MustChangePassword)

		// Setting password clears the flag
		err := user.SetPassword("NewPassword456")
		require.NoError(t, err)
		assert.False(t, user.MustChangePassword)
	})
}

func TestUser_RoleOperations(t *testing.T) {
	tenantID := uuid.New()
	roleID1 := uuid.New()
	roleID2 := uuid.New()

	t.Run("assigns role", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		user.ClearDomainEvents()

		err := user.AssignRole(roleID1)

		require.NoError(t, err)
		assert.Len(t, user.RoleIDs, 1)
		assert.Equal(t, roleID1, user.RoleIDs[0])

		// Should have role assigned event
		events := user.GetDomainEvents()
		assert.Len(t, events, 1)
		event, ok := events[0].(*UserRoleAssignedEvent)
		assert.True(t, ok)
		assert.Equal(t, roleID1, event.RoleID)
	})

	t.Run("fails to assign empty role ID", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		err := user.AssignRole(uuid.Nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails to assign duplicate role", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		_ = user.AssignRole(roleID1)

		err := user.AssignRole(roleID1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already has this role")
	})

	t.Run("removes role", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		_ = user.AssignRole(roleID1)
		_ = user.AssignRole(roleID2)
		user.ClearDomainEvents()

		err := user.RemoveRole(roleID1)

		require.NoError(t, err)
		assert.Len(t, user.RoleIDs, 1)
		assert.Equal(t, roleID2, user.RoleIDs[0])

		// Should have role removed event
		events := user.GetDomainEvents()
		assert.Len(t, events, 1)
		event, ok := events[0].(*UserRoleRemovedEvent)
		assert.True(t, ok)
		assert.Equal(t, roleID1, event.RoleID)
	})

	t.Run("fails to remove role not assigned", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		err := user.RemoveRole(roleID1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have this role")
	})

	t.Run("sets all roles", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		_ = user.AssignRole(uuid.New())

		err := user.SetRoles([]uuid.UUID{roleID1, roleID2})

		require.NoError(t, err)
		assert.Len(t, user.RoleIDs, 2)
	})

	t.Run("deduplicates roles when setting", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		err := user.SetRoles([]uuid.UUID{roleID1, roleID1, roleID2})

		require.NoError(t, err)
		assert.Len(t, user.RoleIDs, 2)
	})

	t.Run("has role check", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		_ = user.AssignRole(roleID1)

		assert.True(t, user.HasRole(roleID1))
		assert.False(t, user.HasRole(roleID2))
	})
}

func TestUser_StatusOperations(t *testing.T) {
	tenantID := uuid.New()

	t.Run("activates pending user", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		assert.Equal(t, UserStatusPending, user.Status)
		user.ClearDomainEvents()

		err := user.Activate()

		require.NoError(t, err)
		assert.Equal(t, UserStatusActive, user.Status)

		// Should have status changed event
		events := user.GetDomainEvents()
		assert.Len(t, events, 1)
		event, ok := events[0].(*UserStatusChangedEvent)
		assert.True(t, ok)
		assert.Equal(t, UserStatusPending, event.OldStatus)
		assert.Equal(t, UserStatusActive, event.NewStatus)
	})

	t.Run("fails to activate already active user", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")

		err := user.Activate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})

	t.Run("deactivates user", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		user.ClearDomainEvents()

		err := user.Deactivate()

		require.NoError(t, err)
		assert.Equal(t, UserStatusDeactivated, user.Status)

		// Should have deactivated event and status changed event
		events := user.GetDomainEvents()
		assert.Len(t, events, 2)
	})

	t.Run("fails to deactivate already deactivated user", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		_ = user.Deactivate()

		err := user.Deactivate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already deactivated")
	})

	t.Run("locks user", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		user.ClearDomainEvents()

		err := user.Lock(time.Hour)

		require.NoError(t, err)
		assert.Equal(t, UserStatusLocked, user.Status)
		assert.NotNil(t, user.LockedUntil)
		assert.True(t, user.IsLocked())
	})

	t.Run("locks user indefinitely", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")

		err := user.Lock(0)

		require.NoError(t, err)
		assert.Equal(t, UserStatusLocked, user.Status)
		assert.Nil(t, user.LockedUntil)
		assert.True(t, user.IsLocked())
	})

	t.Run("cannot lock deactivated user", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		_ = user.Deactivate()

		err := user.Lock(time.Hour)

		assert.Error(t, err)
	})

	t.Run("unlocks user", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		_ = user.Lock(time.Hour)
		user.ClearDomainEvents()

		err := user.Unlock()

		require.NoError(t, err)
		assert.Equal(t, UserStatusActive, user.Status)
		assert.Nil(t, user.LockedUntil)
		assert.False(t, user.IsLocked())
	})

	t.Run("cannot unlock non-locked user", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")

		err := user.Unlock()

		assert.Error(t, err)
	})
}

func TestUser_LoginOperations(t *testing.T) {
	tenantID := uuid.New()

	t.Run("records login success", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		user.FailedAttempts = 3

		user.RecordLoginSuccess("192.168.1.1")

		assert.NotNil(t, user.LastLoginAt)
		assert.Equal(t, "192.168.1.1", user.LastLoginIP)
		assert.Equal(t, 0, user.FailedAttempts)
	})

	t.Run("records login failure and locks after max attempts", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		maxAttempts := 5
		lockDuration := time.Hour

		for i := 0; i < 4; i++ {
			locked := user.RecordLoginFailure(maxAttempts, lockDuration)
			assert.False(t, locked)
			assert.Equal(t, i+1, user.FailedAttempts)
		}

		// Fifth attempt should lock
		locked := user.RecordLoginFailure(maxAttempts, lockDuration)
		assert.True(t, locked)
		assert.True(t, user.IsLocked())
	})

	t.Run("can login when active", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")

		assert.True(t, user.CanLogin())
	})

	t.Run("cannot login when deactivated", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		_ = user.Deactivate()

		assert.False(t, user.CanLogin())
	})

	t.Run("cannot login when locked", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		_ = user.Lock(time.Hour)

		assert.False(t, user.CanLogin())
	})

	t.Run("can login when lock expired", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		user.Status = UserStatusLocked
		pastTime := time.Now().Add(-time.Hour)
		user.LockedUntil = &pastTime

		// IsLocked should return false since lock expired
		assert.False(t, user.IsLocked())
		assert.True(t, user.CanLogin())
	})
}

func TestUser_StatusChecks(t *testing.T) {
	tenantID := uuid.New()

	t.Run("is active", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		assert.True(t, user.IsActive())
	})

	t.Run("is pending", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		assert.True(t, user.IsPending())
	})

	t.Run("is deactivated", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		_ = user.Deactivate()
		assert.True(t, user.IsDeactivated())
	})

	t.Run("is locked", func(t *testing.T) {
		user, _ := NewActiveUser(tenantID, "testuser", "Password123")
		_ = user.Lock(time.Hour)
		assert.True(t, user.IsLocked())
	})
}

func TestUser_GetDisplayNameOrUsername(t *testing.T) {
	tenantID := uuid.New()

	t.Run("returns display name when set", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")
		_ = user.SetDisplayName("Test User")

		assert.Equal(t, "Test User", user.GetDisplayNameOrUsername())
	})

	t.Run("returns username when display name not set", func(t *testing.T) {
		user, _ := NewUser(tenantID, "testuser", "Password123")

		assert.Equal(t, "testuser", user.GetDisplayNameOrUsername())
	})
}

func TestUserRole_TableName(t *testing.T) {
	ur := UserRole{}
	assert.Equal(t, "user_roles", ur.TableName())
}

func TestUser_TableName(t *testing.T) {
	u := User{}
	assert.Equal(t, "users", u.TableName())
}
