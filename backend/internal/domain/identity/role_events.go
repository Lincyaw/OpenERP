package identity

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Aggregate type constant for Role
const AggregateTypeRole = "Role"

// Role domain event types
const (
	EventTypeRoleCreated           = "RoleCreated"
	EventTypeRoleUpdated           = "RoleUpdated"
	EventTypeRoleDeleted           = "RoleDeleted"
	EventTypeRoleEnabled           = "RoleEnabled"
	EventTypeRoleDisabled          = "RoleDisabled"
	EventTypeRolePermissionGranted = "RolePermissionGranted"
	EventTypeRolePermissionRevoked = "RolePermissionRevoked"
	EventTypeRoleDataScopeChanged  = "RoleDataScopeChanged"
	EventTypeRoleUsersChanged      = "RoleUsersChanged"
)

// RoleCreatedEvent is published when a new role is created
type RoleCreatedEvent struct {
	shared.BaseDomainEvent
	Code         string `json:"code"`
	Name         string `json:"name"`
	IsSystemRole bool   `json:"is_system_role"`
}

// NewRoleCreatedEvent creates a new RoleCreatedEvent
func NewRoleCreatedEvent(role *Role) *RoleCreatedEvent {
	return &RoleCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeRoleCreated, AggregateTypeRole, role.TenantID, role.ID),
		Code:            role.Code,
		Name:            role.Name,
		IsSystemRole:    role.IsSystemRole,
	}
}

// RoleUpdatedEvent is published when a role is updated
type RoleUpdatedEvent struct {
	shared.BaseDomainEvent
	Code string `json:"code"`
	Name string `json:"name"`
}

// NewRoleUpdatedEvent creates a new RoleUpdatedEvent
func NewRoleUpdatedEvent(role *Role) *RoleUpdatedEvent {
	return &RoleUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeRoleUpdated, AggregateTypeRole, role.TenantID, role.ID),
		Code:            role.Code,
		Name:            role.Name,
	}
}

// RoleDeletedEvent is published when a role is deleted
type RoleDeletedEvent struct {
	shared.BaseDomainEvent
	Code string `json:"code"`
}

// NewRoleDeletedEvent creates a new RoleDeletedEvent
func NewRoleDeletedEvent(role *Role) *RoleDeletedEvent {
	return &RoleDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeRoleDeleted, AggregateTypeRole, role.TenantID, role.ID),
		Code:            role.Code,
	}
}

// RoleEnabledEvent is published when a role is enabled
type RoleEnabledEvent struct {
	shared.BaseDomainEvent
	Code string `json:"code"`
}

// NewRoleEnabledEvent creates a new RoleEnabledEvent
func NewRoleEnabledEvent(role *Role) *RoleEnabledEvent {
	return &RoleEnabledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeRoleEnabled, AggregateTypeRole, role.TenantID, role.ID),
		Code:            role.Code,
	}
}

// RoleDisabledEvent is published when a role is disabled
type RoleDisabledEvent struct {
	shared.BaseDomainEvent
	Code string `json:"code"`
}

// NewRoleDisabledEvent creates a new RoleDisabledEvent
func NewRoleDisabledEvent(role *Role) *RoleDisabledEvent {
	return &RoleDisabledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeRoleDisabled, AggregateTypeRole, role.TenantID, role.ID),
		Code:            role.Code,
	}
}

// RolePermissionGrantedEvent is published when a permission is granted to a role
type RolePermissionGrantedEvent struct {
	shared.BaseDomainEvent
	RoleCode           string `json:"role_code"`
	PermissionCode     string `json:"permission_code"`
	PermissionResource string `json:"permission_resource"`
	PermissionAction   string `json:"permission_action"`
}

// NewRolePermissionGrantedEvent creates a new RolePermissionGrantedEvent
func NewRolePermissionGrantedEvent(role *Role, perm Permission) *RolePermissionGrantedEvent {
	return &RolePermissionGrantedEvent{
		BaseDomainEvent:    shared.NewBaseDomainEvent(EventTypeRolePermissionGranted, AggregateTypeRole, role.TenantID, role.ID),
		RoleCode:           role.Code,
		PermissionCode:     perm.Code,
		PermissionResource: perm.Resource,
		PermissionAction:   perm.Action,
	}
}

// RolePermissionRevokedEvent is published when a permission is revoked from a role
type RolePermissionRevokedEvent struct {
	shared.BaseDomainEvent
	RoleCode           string `json:"role_code"`
	PermissionCode     string `json:"permission_code"`
	PermissionResource string `json:"permission_resource"`
	PermissionAction   string `json:"permission_action"`
}

// NewRolePermissionRevokedEvent creates a new RolePermissionRevokedEvent
func NewRolePermissionRevokedEvent(role *Role, perm Permission) *RolePermissionRevokedEvent {
	return &RolePermissionRevokedEvent{
		BaseDomainEvent:    shared.NewBaseDomainEvent(EventTypeRolePermissionRevoked, AggregateTypeRole, role.TenantID, role.ID),
		RoleCode:           role.Code,
		PermissionCode:     perm.Code,
		PermissionResource: perm.Resource,
		PermissionAction:   perm.Action,
	}
}

// RoleDataScopeChangedEvent is published when a data scope is changed for a role
type RoleDataScopeChangedEvent struct {
	shared.BaseDomainEvent
	RoleCode  string        `json:"role_code"`
	Resource  string        `json:"resource"`
	ScopeType DataScopeType `json:"scope_type"`
}

// NewRoleDataScopeChangedEvent creates a new RoleDataScopeChangedEvent
func NewRoleDataScopeChangedEvent(role *Role, ds DataScope) *RoleDataScopeChangedEvent {
	return &RoleDataScopeChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeRoleDataScopeChanged, AggregateTypeRole, role.TenantID, role.ID),
		RoleCode:        role.Code,
		Resource:        ds.Resource,
		ScopeType:       ds.ScopeType,
	}
}

// RoleUsersChangedEvent is published when users assigned to a role changes
// This is a summary event that can be published after batch role assignment
type RoleUsersChangedEvent struct {
	shared.BaseDomainEvent
	RoleCode  string      `json:"role_code"`
	UserIDs   []uuid.UUID `json:"user_ids"`
	ChangedAt time.Time   `json:"changed_at"`
}

// NewRoleUsersChangedEvent creates a new RoleUsersChangedEvent
func NewRoleUsersChangedEvent(role *Role, userIDs []uuid.UUID) *RoleUsersChangedEvent {
	return &RoleUsersChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeRoleUsersChanged, AggregateTypeRole, role.TenantID, role.ID),
		RoleCode:        role.Code,
		UserIDs:         userIDs,
		ChangedAt:       time.Now(),
	}
}
