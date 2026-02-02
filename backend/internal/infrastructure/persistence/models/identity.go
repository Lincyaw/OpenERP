package models

import (
	"encoding/json"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// UserModel is the persistence model for the User domain entity.
type UserModel struct {
	TenantAggregateModel
	Username           string              `gorm:"type:varchar(100);not null"`
	Email              string              `gorm:"type:varchar(200)"`
	Phone              string              `gorm:"type:varchar(50)"`
	PasswordHash       string              `gorm:"type:varchar(255);not null"`
	DisplayName        string              `gorm:"type:varchar(200)"`
	Avatar             string              `gorm:"type:varchar(500)"`
	Status             identity.UserStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	DepartmentID       *uuid.UUID          `gorm:"type:uuid;index"`
	LastLoginAt        *time.Time          `gorm:"index"`
	LastLoginIP        string              `gorm:"type:varchar(45)"`
	FailedAttempts     int                 `gorm:"not null;default:0"`
	LockedUntil        *time.Time
	PasswordChangedAt  *time.Time
	MustChangePassword bool   `gorm:"not null;default:false"`
	Notes              string `gorm:"type:text"`
}

// TableName returns the table name for GORM
func (UserModel) TableName() string {
	return "users"
}

// ToDomain converts the persistence model to a domain User entity.
// Note: RoleIDs must be loaded separately by the repository.
func (m *UserModel) ToDomain() *identity.User {
	user := &identity.User{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID:  m.TenantID,
			CreatedBy: m.CreatedBy,
		},
		Username:           m.Username,
		Email:              m.Email,
		Phone:              m.Phone,
		PasswordHash:       m.PasswordHash,
		DisplayName:        m.DisplayName,
		Avatar:             m.Avatar,
		Status:             m.Status,
		DepartmentID:       m.DepartmentID,
		RoleIDs:            make([]uuid.UUID, 0), // Loaded separately
		LastLoginAt:        m.LastLoginAt,
		LastLoginIP:        m.LastLoginIP,
		FailedAttempts:     m.FailedAttempts,
		LockedUntil:        m.LockedUntil,
		PasswordChangedAt:  m.PasswordChangedAt,
		MustChangePassword: m.MustChangePassword,
		Notes:              m.Notes,
	}
	return user
}

// FromDomain populates the persistence model from a domain User entity.
func (m *UserModel) FromDomain(u *identity.User) {
	m.FromDomainTenantAggregateRoot(u.TenantAggregateRoot)
	m.Username = u.Username
	m.Email = u.Email
	m.Phone = u.Phone
	m.PasswordHash = u.PasswordHash
	m.DisplayName = u.DisplayName
	m.Avatar = u.Avatar
	m.Status = u.Status
	m.DepartmentID = u.DepartmentID
	m.LastLoginAt = u.LastLoginAt
	m.LastLoginIP = u.LastLoginIP
	m.FailedAttempts = u.FailedAttempts
	m.LockedUntil = u.LockedUntil
	m.PasswordChangedAt = u.PasswordChangedAt
	m.MustChangePassword = u.MustChangePassword
	m.Notes = u.Notes
}

// UserModelFromDomain creates a new persistence model from a domain User entity.
func UserModelFromDomain(u *identity.User) *UserModel {
	m := &UserModel{}
	m.FromDomain(u)
	return m
}

// UserRoleModel is the persistence model for the UserRole relationship.
type UserRoleModel struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time `gorm:"not null"`
}

// TableName returns the table name for GORM
func (UserRoleModel) TableName() string {
	return "user_roles"
}

// ToDomain converts the persistence model to a domain UserRole.
func (m *UserRoleModel) ToDomain() identity.UserRole {
	return identity.UserRole{
		UserID:    m.UserID,
		RoleID:    m.RoleID,
		TenantID:  m.TenantID,
		CreatedAt: m.CreatedAt,
	}
}

// FromDomain populates the persistence model from a domain UserRole.
func (m *UserRoleModel) FromDomain(ur identity.UserRole) {
	m.UserID = ur.UserID
	m.RoleID = ur.RoleID
	m.TenantID = ur.TenantID
	m.CreatedAt = ur.CreatedAt
}

// TenantModel is the persistence model for the Tenant domain entity.
type TenantModel struct {
	AggregateModel
	Code      string                `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name      string                `gorm:"type:varchar(200);not null"`
	ShortName string                `gorm:"type:varchar(100)"`
	Status    identity.TenantStatus `gorm:"type:varchar(20);not null;default:'active'"`
	Plan      identity.TenantPlan   `gorm:"type:varchar(20);not null;default:'free'"`
	// Scheduled plan change fields (for downgrades)
	ScheduledPlan            *string    `gorm:"column:scheduled_plan;type:varchar(20)"`
	ScheduledPlanEffectiveAt *time.Time `gorm:"column:scheduled_plan_effective_at"`
	ContactName              string     `gorm:"type:varchar(100)"`
	ContactPhone             string     `gorm:"type:varchar(50)"`
	ContactEmail             string     `gorm:"type:varchar(200)"`
	Address                  string     `gorm:"type:text"`
	LogoURL                  string     `gorm:"type:varchar(500)"`
	Domain                   string     `gorm:"type:varchar(200);uniqueIndex"`
	ExpiresAt                *time.Time `gorm:"index"`
	TrialEndsAt              *time.Time
	// Embedded config fields
	ConfigMaxUsers      int    `gorm:"column:config_max_users;not null;default:5"`
	ConfigMaxWarehouses int    `gorm:"column:config_max_warehouses;not null;default:3"`
	ConfigMaxProducts   int    `gorm:"column:config_max_products;not null;default:1000"`
	ConfigFeatures      string `gorm:"column:config_features;type:jsonb;default:'{}'"`
	ConfigSettings      string `gorm:"column:config_settings;type:jsonb;default:'{}'"`
	ConfigCostStrategy  string `gorm:"column:config_cost_strategy;type:varchar(50);default:'weighted_average'"`
	ConfigCurrency      string `gorm:"column:config_currency;type:varchar(10);default:'CNY'"`
	ConfigTimezone      string `gorm:"column:config_timezone;type:varchar(50);default:'Asia/Shanghai'"`
	ConfigLocale        string `gorm:"column:config_locale;type:varchar(20);default:'zh-CN'"`
	Notes               string `gorm:"type:text"`
	// Stripe billing fields
	StripeCustomerID     string `gorm:"column:stripe_customer_id;type:varchar(255);index"`
	StripeSubscriptionID string `gorm:"column:stripe_subscription_id;type:varchar(255);index"`
	// Suspension-related fields
	SuspensionReason      string     `gorm:"column:suspension_reason;type:text"`
	SuspendedAt           *time.Time `gorm:"column:suspended_at"`
	ScheduledReactivateAt *time.Time `gorm:"column:scheduled_reactivate_at;index"`
}

// TableName returns the table name for GORM
func (TenantModel) TableName() string {
	return "tenants"
}

// ToDomain converts the persistence model to a domain Tenant entity.
func (m *TenantModel) ToDomain() *identity.Tenant {
	tenant := &identity.Tenant{
		BaseAggregateRoot: shared.BaseAggregateRoot{
			BaseEntity: shared.BaseEntity{
				ID:        m.ID,
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
			},
			Version: m.Version,
		},
		Code:                     m.Code,
		Name:                     m.Name,
		ShortName:                m.ShortName,
		Status:                   m.Status,
		Plan:                     m.Plan,
		ScheduledPlanEffectiveAt: m.ScheduledPlanEffectiveAt,
		ContactName:              m.ContactName,
		ContactPhone:             m.ContactPhone,
		ContactEmail:             m.ContactEmail,
		Address:                  m.Address,
		LogoURL:                  m.LogoURL,
		Domain:                   m.Domain,
		ExpiresAt:                m.ExpiresAt,
		TrialEndsAt:              m.TrialEndsAt,
		Config: identity.TenantConfig{
			MaxUsers:      m.ConfigMaxUsers,
			MaxWarehouses: m.ConfigMaxWarehouses,
			MaxProducts:   m.ConfigMaxProducts,
			Features:      m.ConfigFeatures,
			Settings:      m.ConfigSettings,
			CostStrategy:  m.ConfigCostStrategy,
			Currency:      m.ConfigCurrency,
			Timezone:      m.ConfigTimezone,
			Locale:        m.ConfigLocale,
		},
		Notes:                 m.Notes,
		StripeCustomerID:      m.StripeCustomerID,
		StripeSubscriptionID:  m.StripeSubscriptionID,
		SuspensionReason:      m.SuspensionReason,
		SuspendedAt:           m.SuspendedAt,
		ScheduledReactivateAt: m.ScheduledReactivateAt,
	}

	// Convert scheduled plan string to TenantPlan pointer
	if m.ScheduledPlan != nil && *m.ScheduledPlan != "" {
		plan := identity.TenantPlan(*m.ScheduledPlan)
		tenant.ScheduledPlan = &plan
	}

	return tenant
}

// FromDomain populates the persistence model from a domain Tenant entity.
func (m *TenantModel) FromDomain(t *identity.Tenant) {
	m.FromDomainAggregateRoot(t.BaseAggregateRoot)
	m.Code = t.Code
	m.Name = t.Name
	m.ShortName = t.ShortName
	m.Status = t.Status
	m.Plan = t.Plan
	// Convert scheduled plan pointer to string pointer
	if t.ScheduledPlan != nil {
		planStr := string(*t.ScheduledPlan)
		m.ScheduledPlan = &planStr
	} else {
		m.ScheduledPlan = nil
	}
	m.ScheduledPlanEffectiveAt = t.ScheduledPlanEffectiveAt
	m.ContactName = t.ContactName
	m.ContactPhone = t.ContactPhone
	m.ContactEmail = t.ContactEmail
	m.Address = t.Address
	m.LogoURL = t.LogoURL
	m.Domain = t.Domain
	m.ExpiresAt = t.ExpiresAt
	m.TrialEndsAt = t.TrialEndsAt
	m.ConfigMaxUsers = t.Config.MaxUsers
	m.ConfigMaxWarehouses = t.Config.MaxWarehouses
	m.ConfigMaxProducts = t.Config.MaxProducts
	m.ConfigFeatures = t.Config.Features
	m.ConfigSettings = t.Config.Settings
	m.ConfigCostStrategy = t.Config.CostStrategy
	m.ConfigCurrency = t.Config.Currency
	m.ConfigTimezone = t.Config.Timezone
	m.ConfigLocale = t.Config.Locale
	m.Notes = t.Notes
	m.StripeCustomerID = t.StripeCustomerID
	m.StripeSubscriptionID = t.StripeSubscriptionID
	m.SuspensionReason = t.SuspensionReason
	m.SuspendedAt = t.SuspendedAt
	m.ScheduledReactivateAt = t.ScheduledReactivateAt
}

// TenantModelFromDomain creates a new persistence model from a domain Tenant entity.
func TenantModelFromDomain(t *identity.Tenant) *TenantModel {
	m := &TenantModel{}
	m.FromDomain(t)
	return m
}

// RoleModel is the persistence model for the Role domain entity.
type RoleModel struct {
	TenantAggregateModel
	Code         string `gorm:"type:varchar(50);not null"`
	Name         string `gorm:"type:varchar(100);not null"`
	Description  string `gorm:"type:text"`
	IsSystemRole bool   `gorm:"not null;default:false"`
	IsEnabled    bool   `gorm:"not null;default:true"`
	SortOrder    int    `gorm:"not null;default:0"`
}

// TableName returns the table name for GORM
func (RoleModel) TableName() string {
	return "roles"
}

// ToDomain converts the persistence model to a domain Role entity.
// Note: Permissions and DataScopes must be loaded separately by the repository.
func (m *RoleModel) ToDomain() *identity.Role {
	return &identity.Role{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID:  m.TenantID,
			CreatedBy: m.CreatedBy,
		},
		Code:         m.Code,
		Name:         m.Name,
		Description:  m.Description,
		IsSystemRole: m.IsSystemRole,
		IsEnabled:    m.IsEnabled,
		SortOrder:    m.SortOrder,
		Permissions:  make([]identity.Permission, 0),
		DataScopes:   make([]identity.DataScope, 0),
	}
}

// FromDomain populates the persistence model from a domain Role entity.
func (m *RoleModel) FromDomain(r *identity.Role) {
	m.FromDomainTenantAggregateRoot(r.TenantAggregateRoot)
	m.Code = r.Code
	m.Name = r.Name
	m.Description = r.Description
	m.IsSystemRole = r.IsSystemRole
	m.IsEnabled = r.IsEnabled
	m.SortOrder = r.SortOrder
}

// RoleModelFromDomain creates a new persistence model from a domain Role entity.
func RoleModelFromDomain(r *identity.Role) *RoleModel {
	m := &RoleModel{}
	m.FromDomain(r)
	return m
}

// RolePermissionModel is the persistence model for role permissions.
type RolePermissionModel struct {
	RoleID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Code        string    `gorm:"type:varchar(100);primaryKey"`
	Resource    string    `gorm:"type:varchar(50);not null;index"`
	Action      string    `gorm:"type:varchar(50);not null"`
	Description string    `gorm:"type:varchar(200)"`
	CreatedAt   time.Time `gorm:"not null"`
}

// TableName returns the table name for GORM
func (RolePermissionModel) TableName() string {
	return "role_permissions"
}

// ToDomain converts the persistence model to a domain Permission.
func (m *RolePermissionModel) ToDomain() identity.Permission {
	return identity.Permission{
		Code:        m.Code,
		Resource:    m.Resource,
		Action:      m.Action,
		Description: m.Description,
	}
}

// FromDomain populates the persistence model from a domain Permission.
func (m *RolePermissionModel) FromDomain(roleID, tenantID uuid.UUID, p identity.Permission) {
	m.RoleID = roleID
	m.TenantID = tenantID
	m.Code = p.Code
	m.Resource = p.Resource
	m.Action = p.Action
	m.Description = p.Description
	m.CreatedAt = time.Now()
}

// RoleDataScopeModel is the persistence model for role data scopes.
type RoleDataScopeModel struct {
	RoleID      uuid.UUID              `gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID              `gorm:"type:uuid;not null;index"`
	Resource    string                 `gorm:"type:varchar(50);primaryKey"`
	ScopeType   identity.DataScopeType `gorm:"type:varchar(20);not null"`
	ScopeField  string                 `gorm:"type:varchar(50)"`
	ScopeValues string                 `gorm:"type:text"` // JSON array
	Description string                 `gorm:"type:varchar(200)"`
	CreatedAt   time.Time              `gorm:"not null"`
}

// TableName returns the table name for GORM
func (RoleDataScopeModel) TableName() string {
	return "role_data_scopes"
}

// ToDomain converts the persistence model to a domain DataScope.
// Note: ScopeValues JSON parsing must be handled by the repository.
func (m *RoleDataScopeModel) ToDomain() identity.DataScope {
	return identity.DataScope{
		Resource:    m.Resource,
		ScopeType:   m.ScopeType,
		ScopeField:  m.ScopeField,
		ScopeValues: make([]string, 0), // Parsed from JSON by repository
		Description: m.Description,
	}
}

// FromDomain populates the persistence model from a domain DataScope.
// Note: ScopeValues must be JSON-encoded by the repository.
func (m *RoleDataScopeModel) FromDomain(roleID, tenantID uuid.UUID, ds identity.DataScope, scopeValuesJSON string) {
	m.RoleID = roleID
	m.TenantID = tenantID
	m.Resource = ds.Resource
	m.ScopeType = ds.ScopeType
	m.ScopeField = ds.ScopeField
	m.ScopeValues = scopeValuesJSON
	m.Description = ds.Description
	m.CreatedAt = time.Now()
}

// DepartmentModel is the persistence model for the Department domain entity.
type DepartmentModel struct {
	TenantAggregateModel
	Code        string                    `gorm:"type:varchar(50);not null"`
	Name        string                    `gorm:"type:varchar(200);not null"`
	Description string                    `gorm:"type:text"`
	ParentID    *uuid.UUID                `gorm:"type:uuid;index"`
	Path        string                    `gorm:"type:varchar(1000);not null;index"`
	Level       int                       `gorm:"not null;default:0"`
	SortOrder   int                       `gorm:"not null;default:0"`
	ManagerID   *uuid.UUID                `gorm:"type:uuid;index"`
	Status      identity.DepartmentStatus `gorm:"type:varchar(20);not null;default:'active'"`
	Metadata    string                    `gorm:"type:jsonb;default:'{}'"`
}

// TableName returns the table name for GORM
func (DepartmentModel) TableName() string {
	return "departments"
}

// ToDomain converts the persistence model to a domain Department entity.
func (m *DepartmentModel) ToDomain() *identity.Department {
	dept := &identity.Department{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID:  m.TenantID,
			CreatedBy: m.CreatedBy,
		},
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		ParentID:    m.ParentID,
		Path:        m.Path,
		Level:       m.Level,
		SortOrder:   m.SortOrder,
		ManagerID:   m.ManagerID,
		Status:      m.Status,
		Metadata:    make(map[string]string), // Parsed from JSON by repository
	}
	return dept
}

// FromDomain populates the persistence model from a domain Department entity.
// Note: Metadata must be JSON-encoded by the repository.
func (m *DepartmentModel) FromDomain(d *identity.Department, metadataJSON string) {
	m.FromDomainTenantAggregateRoot(d.TenantAggregateRoot)
	m.Code = d.Code
	m.Name = d.Name
	m.Description = d.Description
	m.ParentID = d.ParentID
	m.Path = d.Path
	m.Level = d.Level
	m.SortOrder = d.SortOrder
	m.ManagerID = d.ManagerID
	m.Status = d.Status
	m.Metadata = metadataJSON
}

// DepartmentModelFromDomain creates a new persistence model from a domain Department entity.
func DepartmentModelFromDomain(d *identity.Department, metadataJSON string) *DepartmentModel {
	m := &DepartmentModel{}
	m.FromDomain(d, metadataJSON)
	return m
}

// PlanFeatureModel is the persistence model for the PlanFeature domain entity.
type PlanFeatureModel struct {
	ID          uuid.UUID           `gorm:"type:uuid;primary_key"`
	PlanID      identity.TenantPlan `gorm:"column:plan_id;type:varchar(50);not null"`
	FeatureKey  string              `gorm:"column:feature_key;type:varchar(100);not null"`
	Enabled     bool                `gorm:"not null;default:false"`
	Limit       *int                `gorm:"column:feature_limit"`
	Description string              `gorm:"type:text"`
	CreatedAt   time.Time           `gorm:"not null"`
	UpdatedAt   time.Time           `gorm:"not null"`
}

// TableName returns the table name for GORM
func (PlanFeatureModel) TableName() string {
	return "plan_features"
}

// ToDomain converts the persistence model to a domain PlanFeature entity.
func (m *PlanFeatureModel) ToDomain() *identity.PlanFeature {
	return &identity.PlanFeature{
		ID:          m.ID,
		PlanID:      m.PlanID,
		FeatureKey:  identity.FeatureKey(m.FeatureKey),
		Enabled:     m.Enabled,
		Limit:       m.Limit,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain PlanFeature entity.
func (m *PlanFeatureModel) FromDomain(pf *identity.PlanFeature) {
	m.ID = pf.ID
	m.PlanID = pf.PlanID
	m.FeatureKey = string(pf.FeatureKey)
	m.Enabled = pf.Enabled
	m.Limit = pf.Limit
	m.Description = pf.Description
	m.CreatedAt = pf.CreatedAt
	m.UpdatedAt = pf.UpdatedAt
}

// PlanFeatureModelFromDomain creates a new persistence model from a domain PlanFeature entity.
func PlanFeatureModelFromDomain(pf *identity.PlanFeature) *PlanFeatureModel {
	m := &PlanFeatureModel{}
	m.FromDomain(pf)
	return m
}

// PlanFeatureChangeLogModel is the persistence model for audit logging plan feature changes.
type PlanFeatureChangeLogModel struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key"`
	PlanID     string     `gorm:"column:plan_id;type:varchar(50);not null"`
	FeatureKey string     `gorm:"column:feature_key;type:varchar(100);not null"`
	ChangeType string     `gorm:"column:change_type;type:varchar(20);not null"` // created, updated, deleted
	OldEnabled *bool      `gorm:"column:old_enabled"`
	NewEnabled *bool      `gorm:"column:new_enabled"`
	OldLimit   *int       `gorm:"column:old_limit"`
	NewLimit   *int       `gorm:"column:new_limit"`
	ChangedBy  *uuid.UUID `gorm:"column:changed_by;type:uuid"`
	ChangedAt  time.Time  `gorm:"column:changed_at;not null"`
}

// TableName returns the table name for GORM
func (PlanFeatureChangeLogModel) TableName() string {
	return "plan_feature_change_logs"
}

// AdminAuditLogModel is the persistence model for admin audit logs.
// This model represents immutable audit entries for super admin operations.
type AdminAuditLogModel struct {
	ID          uuid.UUID                `gorm:"type:uuid;primaryKey"`
	AdminUserID uuid.UUID                `gorm:"type:uuid;not null;index"`
	Action      identity.AuditAction     `gorm:"type:varchar(50);not null;index"`
	TargetType  identity.AuditTargetType `gorm:"type:varchar(50);not null"`
	TargetID    *uuid.UUID               `gorm:"type:uuid;index"`
	OldValue    *string                  `gorm:"type:jsonb"`
	NewValue    *string                  `gorm:"type:jsonb"`
	IPAddress   string                   `gorm:"type:varchar(45)"`
	UserAgent   string                   `gorm:"type:varchar(500)"`
	CreatedAt   time.Time                `gorm:"not null;index"`
}

// TableName returns the table name for GORM
func (AdminAuditLogModel) TableName() string {
	return "admin_audit_logs"
}

// ToDomain converts the persistence model to a domain AuditLog entity.
func (m *AdminAuditLogModel) ToDomain() *identity.AuditLog {
	log := &identity.AuditLog{
		ID:          m.ID,
		AdminUserID: m.AdminUserID,
		Action:      m.Action,
		TargetType:  m.TargetType,
		TargetID:    m.TargetID,
		IPAddress:   m.IPAddress,
		UserAgent:   m.UserAgent,
		CreatedAt:   m.CreatedAt,
	}

	// Parse OldValue JSON
	if m.OldValue != nil && *m.OldValue != "" {
		var oldVal map[string]interface{}
		if err := json.Unmarshal([]byte(*m.OldValue), &oldVal); err == nil {
			log.OldValue = oldVal
		}
	}

	// Parse NewValue JSON
	if m.NewValue != nil && *m.NewValue != "" {
		var newVal map[string]interface{}
		if err := json.Unmarshal([]byte(*m.NewValue), &newVal); err == nil {
			log.NewValue = newVal
		}
	}

	return log
}

// FromDomain populates the persistence model from a domain AuditLog entity.
func (m *AdminAuditLogModel) FromDomain(log *identity.AuditLog) error {
	m.ID = log.ID
	m.AdminUserID = log.AdminUserID
	m.Action = log.Action
	m.TargetType = log.TargetType
	m.TargetID = log.TargetID
	m.IPAddress = log.IPAddress
	m.UserAgent = log.UserAgent
	m.CreatedAt = log.CreatedAt

	// Serialize OldValue to JSON
	if log.OldValue != nil {
		oldJSON, err := json.Marshal(log.OldValue)
		if err != nil {
			return err
		}
		oldStr := string(oldJSON)
		m.OldValue = &oldStr
	}

	// Serialize NewValue to JSON
	if log.NewValue != nil {
		newJSON, err := json.Marshal(log.NewValue)
		if err != nil {
			return err
		}
		newStr := string(newJSON)
		m.NewValue = &newStr
	}

	return nil
}

// AdminAuditLogModelFromDomain creates a new persistence model from a domain AuditLog entity.
func AdminAuditLogModelFromDomain(log *identity.AuditLog) (*AdminAuditLogModel, error) {
	m := &AdminAuditLogModel{}
	if err := m.FromDomain(log); err != nil {
		return nil, err
	}
	return m, nil
}

// SubscriptionHistoryModel is the persistence model for subscription history.
type SubscriptionHistoryModel struct {
	ID              uuid.UUID                       `gorm:"type:uuid;primaryKey"`
	TenantID        uuid.UUID                       `gorm:"type:uuid;not null;index"`
	ChangeType      identity.SubscriptionChangeType `gorm:"column:change_type;type:varchar(30);not null"`
	OldPlan         *string                         `gorm:"column:old_plan;type:varchar(20)"`
	NewPlan         *string                         `gorm:"column:new_plan;type:varchar(20)"`
	OldQuota        *string                         `gorm:"column:old_quota;type:jsonb"`
	NewQuota        *string                         `gorm:"column:new_quota;type:jsonb"`
	EffectiveAt     time.Time                       `gorm:"column:effective_at;not null"`
	ScheduledAt     *time.Time                      `gorm:"column:scheduled_at"`
	ChangedByUserID uuid.UUID                       `gorm:"column:changed_by_user_id;type:uuid;not null"`
	Reason          string                          `gorm:"column:reason;type:text"`
	CreatedAt       time.Time                       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (SubscriptionHistoryModel) TableName() string {
	return "subscription_history"
}

// ToDomain converts the persistence model to a domain SubscriptionHistory entity.
func (m *SubscriptionHistoryModel) ToDomain() *identity.SubscriptionHistory {
	history := &identity.SubscriptionHistory{
		ID:              m.ID,
		TenantID:        m.TenantID,
		ChangeType:      m.ChangeType,
		EffectiveAt:     m.EffectiveAt,
		ScheduledAt:     m.ScheduledAt,
		ChangedByUserID: m.ChangedByUserID,
		Reason:          m.Reason,
		CreatedAt:       m.CreatedAt,
	}

	// Convert plan strings to TenantPlan
	if m.OldPlan != nil && *m.OldPlan != "" {
		plan := identity.TenantPlan(*m.OldPlan)
		history.OldPlan = plan
	}
	if m.NewPlan != nil && *m.NewPlan != "" {
		plan := identity.TenantPlan(*m.NewPlan)
		history.NewPlan = plan
	}

	// Parse quota JSON
	if m.OldQuota != nil && *m.OldQuota != "" {
		var quota identity.TenantQuota
		if err := json.Unmarshal([]byte(*m.OldQuota), &quota); err == nil {
			history.OldQuota = &quota
		}
	}
	if m.NewQuota != nil && *m.NewQuota != "" {
		var quota identity.TenantQuota
		if err := json.Unmarshal([]byte(*m.NewQuota), &quota); err == nil {
			history.NewQuota = &quota
		}
	}

	return history
}

// FromDomain populates the persistence model from a domain SubscriptionHistory entity.
func (m *SubscriptionHistoryModel) FromDomain(h *identity.SubscriptionHistory) error {
	m.ID = h.ID
	m.TenantID = h.TenantID
	m.ChangeType = h.ChangeType
	m.EffectiveAt = h.EffectiveAt
	m.ScheduledAt = h.ScheduledAt
	m.ChangedByUserID = h.ChangedByUserID
	m.Reason = h.Reason
	m.CreatedAt = h.CreatedAt

	// Convert plan to string pointer
	if h.OldPlan != "" {
		planStr := string(h.OldPlan)
		m.OldPlan = &planStr
	}
	if h.NewPlan != "" {
		planStr := string(h.NewPlan)
		m.NewPlan = &planStr
	}

	// Serialize quota to JSON
	if h.OldQuota != nil {
		quotaJSON, err := json.Marshal(h.OldQuota)
		if err != nil {
			return err
		}
		quotaStr := string(quotaJSON)
		m.OldQuota = &quotaStr
	}
	if h.NewQuota != nil {
		quotaJSON, err := json.Marshal(h.NewQuota)
		if err != nil {
			return err
		}
		quotaStr := string(quotaJSON)
		m.NewQuota = &quotaStr
	}

	return nil
}

// SubscriptionHistoryModelFromDomain creates a new persistence model from a domain SubscriptionHistory entity.
func SubscriptionHistoryModelFromDomain(h *identity.SubscriptionHistory) (*SubscriptionHistoryModel, error) {
	m := &SubscriptionHistoryModel{}
	if err := m.FromDomain(h); err != nil {
		return nil, err
	}
	return m, nil
}

// TenantStatusHistoryModel is the persistence model for tenant status history.
type TenantStatusHistoryModel struct {
	ID                    uuid.UUID                       `gorm:"type:uuid;primaryKey"`
	TenantID              uuid.UUID                       `gorm:"type:uuid;not null;index"`
	ChangeType            identity.TenantStatusChangeType `gorm:"column:change_type;type:varchar(30);not null"`
	OldStatus             identity.TenantStatus           `gorm:"column:old_status;type:varchar(20);not null"`
	NewStatus             identity.TenantStatus           `gorm:"column:new_status;type:varchar(20);not null"`
	Reason                string                          `gorm:"column:reason;type:text"`
	ScheduledReactivateAt *time.Time                      `gorm:"column:scheduled_reactivate_at"`
	ChangedByUserID       uuid.UUID                       `gorm:"column:changed_by_user_id;type:uuid;not null"`
	IPAddress             string                          `gorm:"column:ip_address;type:varchar(45)"`
	UserAgent             string                          `gorm:"column:user_agent;type:varchar(500)"`
	CreatedAt             time.Time                       `gorm:"not null;index"`
}

// TableName returns the table name for GORM
func (TenantStatusHistoryModel) TableName() string {
	return "tenant_status_history"
}

// ToDomain converts the persistence model to a domain TenantStatusHistory entity.
func (m *TenantStatusHistoryModel) ToDomain() *identity.TenantStatusHistory {
	return &identity.TenantStatusHistory{
		ID:                    m.ID,
		TenantID:              m.TenantID,
		ChangeType:            m.ChangeType,
		OldStatus:             m.OldStatus,
		NewStatus:             m.NewStatus,
		Reason:                m.Reason,
		ScheduledReactivateAt: m.ScheduledReactivateAt,
		ChangedByUserID:       m.ChangedByUserID,
		IPAddress:             m.IPAddress,
		UserAgent:             m.UserAgent,
		CreatedAt:             m.CreatedAt,
	}
}

// FromDomain populates the persistence model from a domain TenantStatusHistory entity.
func (m *TenantStatusHistoryModel) FromDomain(h *identity.TenantStatusHistory) {
	m.ID = h.ID
	m.TenantID = h.TenantID
	m.ChangeType = h.ChangeType
	m.OldStatus = h.OldStatus
	m.NewStatus = h.NewStatus
	m.Reason = h.Reason
	m.ScheduledReactivateAt = h.ScheduledReactivateAt
	m.ChangedByUserID = h.ChangedByUserID
	m.IPAddress = h.IPAddress
	m.UserAgent = h.UserAgent
	m.CreatedAt = h.CreatedAt
}

// TenantStatusHistoryModelFromDomain creates a new persistence model from a domain TenantStatusHistory entity.
func TenantStatusHistoryModelFromDomain(h *identity.TenantStatusHistory) *TenantStatusHistoryModel {
	m := &TenantStatusHistoryModel{}
	m.FromDomain(h)
	return m
}
