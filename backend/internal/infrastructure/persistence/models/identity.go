package models

import (
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
	Code         string                `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name         string                `gorm:"type:varchar(200);not null"`
	ShortName    string                `gorm:"type:varchar(100)"`
	Status       identity.TenantStatus `gorm:"type:varchar(20);not null;default:'active'"`
	Plan         identity.TenantPlan   `gorm:"type:varchar(20);not null;default:'free'"`
	ContactName  string                `gorm:"type:varchar(100)"`
	ContactPhone string                `gorm:"type:varchar(50)"`
	ContactEmail string                `gorm:"type:varchar(200)"`
	Address      string                `gorm:"type:text"`
	LogoURL      string                `gorm:"type:varchar(500)"`
	Domain       string                `gorm:"type:varchar(200);uniqueIndex"`
	ExpiresAt    *time.Time            `gorm:"index"`
	TrialEndsAt  *time.Time
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
}

// TableName returns the table name for GORM
func (TenantModel) TableName() string {
	return "tenants"
}

// ToDomain converts the persistence model to a domain Tenant entity.
func (m *TenantModel) ToDomain() *identity.Tenant {
	return &identity.Tenant{
		BaseAggregateRoot: shared.BaseAggregateRoot{
			BaseEntity: shared.BaseEntity{
				ID:        m.ID,
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
			},
			Version: m.Version,
		},
		Code:         m.Code,
		Name:         m.Name,
		ShortName:    m.ShortName,
		Status:       m.Status,
		Plan:         m.Plan,
		ContactName:  m.ContactName,
		ContactPhone: m.ContactPhone,
		ContactEmail: m.ContactEmail,
		Address:      m.Address,
		LogoURL:      m.LogoURL,
		Domain:       m.Domain,
		ExpiresAt:    m.ExpiresAt,
		TrialEndsAt:  m.TrialEndsAt,
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
		Notes:                m.Notes,
		StripeCustomerID:     m.StripeCustomerID,
		StripeSubscriptionID: m.StripeSubscriptionID,
	}
}

// FromDomain populates the persistence model from a domain Tenant entity.
func (m *TenantModel) FromDomain(t *identity.Tenant) {
	m.FromDomainAggregateRoot(t.BaseAggregateRoot)
	m.Code = t.Code
	m.Name = t.Name
	m.ShortName = t.ShortName
	m.Status = t.Status
	m.Plan = t.Plan
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
