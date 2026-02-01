package identity

import (
	"encoding/json"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// AuditAction represents the type of admin action being audited
type AuditAction string

const (
	// Tenant actions
	AuditActionTenantCreate   AuditAction = "TENANT_CREATE"
	AuditActionTenantUpdate   AuditAction = "TENANT_UPDATE"
	AuditActionTenantDelete   AuditAction = "TENANT_DELETE"
	AuditActionTenantSuspend  AuditAction = "TENANT_SUSPEND"
	AuditActionTenantActivate AuditAction = "TENANT_ACTIVATE"

	// Subscription actions
	AuditActionSubscriptionChange AuditAction = "SUBSCRIPTION_CHANGE"
	AuditActionQuotaUpdate        AuditAction = "QUOTA_UPDATE"

	// User actions
	AuditActionUserCreate   AuditAction = "USER_CREATE"
	AuditActionUserUpdate   AuditAction = "USER_UPDATE"
	AuditActionUserDelete   AuditAction = "USER_DELETE"
	AuditActionUserSuspend  AuditAction = "USER_SUSPEND"
	AuditActionUserActivate AuditAction = "USER_ACTIVATE"

	// Role actions
	AuditActionRoleCreate AuditAction = "ROLE_CREATE"
	AuditActionRoleUpdate AuditAction = "ROLE_UPDATE"
	AuditActionRoleDelete AuditAction = "ROLE_DELETE"

	// Permission actions
	AuditActionPermissionGrant  AuditAction = "PERMISSION_GRANT"
	AuditActionPermissionRevoke AuditAction = "PERMISSION_REVOKE"

	// System config actions
	AuditActionSystemConfigChange AuditAction = "SYSTEM_CONFIG_CHANGE"
)

// AuditTargetType represents the type of entity being audited
type AuditTargetType string

const (
	AuditTargetTenant       AuditTargetType = "tenant"
	AuditTargetUser         AuditTargetType = "user"
	AuditTargetRole         AuditTargetType = "role"
	AuditTargetPermission   AuditTargetType = "permission"
	AuditTargetSubscription AuditTargetType = "subscription"
	AuditTargetQuota        AuditTargetType = "quota"
	AuditTargetSystemConfig AuditTargetType = "system_config"
)

// AuditLog represents an immutable audit log entry for admin operations
// This is NOT an aggregate root as it cannot be modified after creation
type AuditLog struct {
	ID          uuid.UUID
	AdminUserID uuid.UUID
	Action      AuditAction
	TargetType  AuditTargetType
	TargetID    *uuid.UUID
	OldValue    map[string]interface{}
	NewValue    map[string]interface{}
	IPAddress   string
	UserAgent   string
	CreatedAt   time.Time
}

// AuditLogBuilder provides a fluent interface for creating audit logs
type AuditLogBuilder struct {
	log *AuditLog
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(adminUserID uuid.UUID, action AuditAction, targetType AuditTargetType) *AuditLogBuilder {
	return &AuditLogBuilder{
		log: &AuditLog{
			ID:          uuid.New(),
			AdminUserID: adminUserID,
			Action:      action,
			TargetType:  targetType,
			CreatedAt:   time.Now(),
		},
	}
}

// WithTargetID sets the target entity ID
func (b *AuditLogBuilder) WithTargetID(targetID uuid.UUID) *AuditLogBuilder {
	b.log.TargetID = &targetID
	return b
}

// WithOldValue sets the old value (state before the action)
func (b *AuditLogBuilder) WithOldValue(oldValue map[string]interface{}) *AuditLogBuilder {
	b.log.OldValue = oldValue
	return b
}

// WithNewValue sets the new value (state after the action)
func (b *AuditLogBuilder) WithNewValue(newValue map[string]interface{}) *AuditLogBuilder {
	b.log.NewValue = newValue
	return b
}

// WithOldValueFromEntity sets the old value from any entity (serializes to map)
func (b *AuditLogBuilder) WithOldValueFromEntity(entity interface{}) *AuditLogBuilder {
	if entity == nil {
		return b
	}
	data, err := json.Marshal(entity)
	if err != nil {
		return b
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return b
	}
	b.log.OldValue = m
	return b
}

// WithNewValueFromEntity sets the new value from any entity (serializes to map)
func (b *AuditLogBuilder) WithNewValueFromEntity(entity interface{}) *AuditLogBuilder {
	if entity == nil {
		return b
	}
	data, err := json.Marshal(entity)
	if err != nil {
		return b
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return b
	}
	b.log.NewValue = m
	return b
}

// WithIPAddress sets the IP address of the admin user
func (b *AuditLogBuilder) WithIPAddress(ip string) *AuditLogBuilder {
	b.log.IPAddress = ip
	return b
}

// WithUserAgent sets the user agent string
func (b *AuditLogBuilder) WithUserAgent(userAgent string) *AuditLogBuilder {
	b.log.UserAgent = userAgent
	return b
}

// Build validates and returns the audit log entry
func (b *AuditLogBuilder) Build() (*AuditLog, error) {
	if b.log.AdminUserID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_ADMIN_USER_ID", "Admin user ID cannot be empty")
	}
	if b.log.Action == "" {
		return nil, shared.NewDomainError("INVALID_ACTION", "Action cannot be empty")
	}
	if b.log.TargetType == "" {
		return nil, shared.NewDomainError("INVALID_TARGET_TYPE", "Target type cannot be empty")
	}
	if !b.log.Action.IsValid() {
		return nil, shared.NewDomainError("INVALID_ACTION", "Invalid audit action: "+string(b.log.Action))
	}
	if !b.log.TargetType.IsValid() {
		return nil, shared.NewDomainError("INVALID_TARGET_TYPE", "Invalid target type: "+string(b.log.TargetType))
	}

	return b.log, nil
}

// IsValid checks if the action is a valid audit action
func (a AuditAction) IsValid() bool {
	switch a {
	case AuditActionTenantCreate, AuditActionTenantUpdate, AuditActionTenantDelete,
		AuditActionTenantSuspend, AuditActionTenantActivate,
		AuditActionSubscriptionChange, AuditActionQuotaUpdate,
		AuditActionUserCreate, AuditActionUserUpdate, AuditActionUserDelete,
		AuditActionUserSuspend, AuditActionUserActivate,
		AuditActionRoleCreate, AuditActionRoleUpdate, AuditActionRoleDelete,
		AuditActionPermissionGrant, AuditActionPermissionRevoke,
		AuditActionSystemConfigChange:
		return true
	default:
		return false
	}
}

// IsValid checks if the target type is valid
func (t AuditTargetType) IsValid() bool {
	switch t {
	case AuditTargetTenant, AuditTargetUser, AuditTargetRole,
		AuditTargetPermission, AuditTargetSubscription,
		AuditTargetQuota, AuditTargetSystemConfig:
		return true
	default:
		return false
	}
}

// String returns the string representation of the action
func (a AuditAction) String() string {
	return string(a)
}

// String returns the string representation of the target type
func (t AuditTargetType) String() string {
	return string(t)
}

// GetOldValueJSON returns the old value as JSON bytes
func (l *AuditLog) GetOldValueJSON() ([]byte, error) {
	if l.OldValue == nil {
		return nil, nil
	}
	return json.Marshal(l.OldValue)
}

// GetNewValueJSON returns the new value as JSON bytes
func (l *AuditLog) GetNewValueJSON() ([]byte, error) {
	if l.NewValue == nil {
		return nil, nil
	}
	return json.Marshal(l.NewValue)
}

// HasChanges returns true if there are differences between old and new values
func (l *AuditLog) HasChanges() bool {
	return l.OldValue != nil || l.NewValue != nil
}

// GetChangedFields returns a list of fields that changed between old and new values
func (l *AuditLog) GetChangedFields() []string {
	if l.OldValue == nil && l.NewValue == nil {
		return nil
	}

	changedFields := make([]string, 0)
	allKeys := make(map[string]bool)

	// Collect all keys from both maps
	for k := range l.OldValue {
		allKeys[k] = true
	}
	for k := range l.NewValue {
		allKeys[k] = true
	}

	// Compare values for each key
	for k := range allKeys {
		oldVal, oldExists := l.OldValue[k]
		newVal, newExists := l.NewValue[k]

		if !oldExists || !newExists {
			changedFields = append(changedFields, k)
			continue
		}

		// Simple comparison (may not work for nested objects)
		oldJSON, _ := json.Marshal(oldVal)
		newJSON, _ := json.Marshal(newVal)
		if string(oldJSON) != string(newJSON) {
			changedFields = append(changedFields, k)
		}
	}

	return changedFields
}
