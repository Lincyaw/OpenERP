package identity

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Aggregate type constant for User
const AggregateTypeUser = "User"

// User domain event types
const (
	EventTypeUserCreated         = "UserCreated"
	EventTypeUserDeactivated     = "UserDeactivated"
	EventTypeUserPasswordChanged = "UserPasswordChanged"
	EventTypeUserRoleAssigned    = "UserRoleAssigned"
	EventTypeUserRoleRemoved     = "UserRoleRemoved"
	EventTypeUserStatusChanged   = "UserStatusChanged"
)

// UserCreatedEvent is published when a user is created
type UserCreatedEvent struct {
	shared.BaseDomainEvent
	Username string     `json:"username"`
	Email    string     `json:"email"`
	Status   UserStatus `json:"status"`
}

// NewUserCreatedEvent creates a new UserCreatedEvent
func NewUserCreatedEvent(user *User) *UserCreatedEvent {
	return &UserCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeUserCreated, AggregateTypeUser, user.TenantID, user.ID),
		Username:        user.Username,
		Email:           user.Email,
		Status:          user.Status,
	}
}

// UserDeactivatedEvent is published when a user is deactivated
type UserDeactivatedEvent struct {
	shared.BaseDomainEvent
	Username string `json:"username"`
}

// NewUserDeactivatedEvent creates a new UserDeactivatedEvent
func NewUserDeactivatedEvent(user *User) *UserDeactivatedEvent {
	return &UserDeactivatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeUserDeactivated, AggregateTypeUser, user.TenantID, user.ID),
		Username:        user.Username,
	}
}

// UserPasswordChangedEvent is published when a user's password is changed
type UserPasswordChangedEvent struct {
	shared.BaseDomainEvent
	Username  string    `json:"username"`
	ChangedAt time.Time `json:"changed_at"`
}

// NewUserPasswordChangedEvent creates a new UserPasswordChangedEvent
func NewUserPasswordChangedEvent(user *User) *UserPasswordChangedEvent {
	changedAt := time.Now()
	if user.PasswordChangedAt != nil {
		changedAt = *user.PasswordChangedAt
	}
	return &UserPasswordChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeUserPasswordChanged, AggregateTypeUser, user.TenantID, user.ID),
		Username:        user.Username,
		ChangedAt:       changedAt,
	}
}

// UserRoleAssignedEvent is published when a role is assigned to a user
type UserRoleAssignedEvent struct {
	shared.BaseDomainEvent
	Username string    `json:"username"`
	RoleID   uuid.UUID `json:"role_id"`
}

// NewUserRoleAssignedEvent creates a new UserRoleAssignedEvent
func NewUserRoleAssignedEvent(user *User, roleID uuid.UUID) *UserRoleAssignedEvent {
	return &UserRoleAssignedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeUserRoleAssigned, AggregateTypeUser, user.TenantID, user.ID),
		Username:        user.Username,
		RoleID:          roleID,
	}
}

// UserRoleRemovedEvent is published when a role is removed from a user
type UserRoleRemovedEvent struct {
	shared.BaseDomainEvent
	Username string    `json:"username"`
	RoleID   uuid.UUID `json:"role_id"`
}

// NewUserRoleRemovedEvent creates a new UserRoleRemovedEvent
func NewUserRoleRemovedEvent(user *User, roleID uuid.UUID) *UserRoleRemovedEvent {
	return &UserRoleRemovedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeUserRoleRemoved, AggregateTypeUser, user.TenantID, user.ID),
		Username:        user.Username,
		RoleID:          roleID,
	}
}

// UserStatusChangedEvent is published when a user's status changes
type UserStatusChangedEvent struct {
	shared.BaseDomainEvent
	Username  string     `json:"username"`
	OldStatus UserStatus `json:"old_status"`
	NewStatus UserStatus `json:"new_status"`
}

// NewUserStatusChangedEvent creates a new UserStatusChangedEvent
func NewUserStatusChangedEvent(user *User, oldStatus, newStatus UserStatus) *UserStatusChangedEvent {
	return &UserStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeUserStatusChanged, AggregateTypeUser, user.TenantID, user.ID),
		Username:        user.Username,
		OldStatus:       oldStatus,
		NewStatus:       newStatus,
	}
}
