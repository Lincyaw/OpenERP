package identity

import (
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended" // Suspended due to payment/violation issues
	TenantStatusTrial     TenantStatus = "trial"     // Trial period
)

// TenantPlan represents the subscription plan of a tenant
type TenantPlan string

const (
	TenantPlanFree       TenantPlan = "free"
	TenantPlanBasic      TenantPlan = "basic"
	TenantPlanPro        TenantPlan = "pro"
	TenantPlanEnterprise TenantPlan = "enterprise"
)

// TenantConfig holds configurable settings for a tenant
type TenantConfig struct {
	MaxUsers      int    `json:"max_users"`      // Maximum number of users allowed
	MaxWarehouses int    `json:"max_warehouses"` // Maximum number of warehouses
	MaxProducts   int    `json:"max_products"`   // Maximum number of products
	Features      string `json:"features"`       // JSON object of enabled features
	Settings      string `json:"settings"`       // JSON object of tenant settings
	CostStrategy  string `json:"cost_strategy"`  // Default cost calculation strategy (fifo, weighted_average)
	Currency      string `json:"currency"`       // Default currency code
	Timezone      string `json:"timezone"`       // Tenant timezone
	Locale        string `json:"locale"`         // Tenant locale (e.g., zh-CN, en-US)
}

// DefaultTenantConfig returns the default configuration for a new tenant
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		MaxUsers:      5,
		MaxWarehouses: 3,
		MaxProducts:   1000,
		Features:      "{}",
		Settings:      "{}",
		CostStrategy:  "weighted_average",
		Currency:      "CNY",
		Timezone:      "Asia/Shanghai",
		Locale:        "zh-CN",
	}
}

// Tenant represents a tenant/organization in the multi-tenant system
// It is the aggregate root for tenant-related operations
type Tenant struct {
	shared.BaseAggregateRoot
	Code         string       `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name         string       `gorm:"type:varchar(200);not null"`
	ShortName    string       `gorm:"type:varchar(100)"`
	Status       TenantStatus `gorm:"type:varchar(20);not null;default:'active'"`
	Plan         TenantPlan   `gorm:"type:varchar(20);not null;default:'free'"`
	ContactName  string       `gorm:"type:varchar(100)"`
	ContactPhone string       `gorm:"type:varchar(50)"`
	ContactEmail string       `gorm:"type:varchar(200)"`
	Address      string       `gorm:"type:text"`
	LogoURL      string       `gorm:"type:varchar(500)"`
	Domain       string       `gorm:"type:varchar(200);uniqueIndex"` // Custom subdomain
	ExpiresAt    *time.Time   `gorm:"index"`                         // Subscription expiry date
	TrialEndsAt  *time.Time   // Trial period end date
	Config       TenantConfig `gorm:"embedded;embeddedPrefix:config_"`
	Notes        string       `gorm:"type:text"`
}

// TableName returns the table name for GORM
func (Tenant) TableName() string {
	return "tenants"
}

// NewTenant creates a new tenant with required fields
func NewTenant(code, name string) (*Tenant, error) {
	if err := validateTenantCode(code); err != nil {
		return nil, err
	}
	if err := validateTenantName(name); err != nil {
		return nil, err
	}

	tenant := &Tenant{
		BaseAggregateRoot: shared.NewBaseAggregateRoot(),
		Code:              strings.ToUpper(code),
		Name:              name,
		Status:            TenantStatusActive,
		Plan:              TenantPlanFree,
		Config:            DefaultTenantConfig(),
	}

	tenant.AddDomainEvent(NewTenantCreatedEvent(tenant))

	return tenant, nil
}

// NewTrialTenant creates a new tenant in trial status
func NewTrialTenant(code, name string, trialDays int) (*Tenant, error) {
	if trialDays <= 0 {
		return nil, shared.NewDomainError("INVALID_TRIAL_DAYS", "Trial days must be positive")
	}

	tenant, err := NewTenant(code, name)
	if err != nil {
		return nil, err
	}

	tenant.Status = TenantStatusTrial
	trialEnds := time.Now().AddDate(0, 0, trialDays)
	tenant.TrialEndsAt = &trialEnds

	return tenant, nil
}

// Update updates the tenant's basic information
func (t *Tenant) Update(name, shortName string) error {
	if err := validateTenantName(name); err != nil {
		return err
	}
	if shortName != "" && len(shortName) > 100 {
		return shared.NewDomainError("INVALID_SHORT_NAME", "Short name cannot exceed 100 characters")
	}

	t.Name = name
	t.ShortName = shortName
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewTenantUpdatedEvent(t))

	return nil
}

// SetContact sets the tenant's contact information
func (t *Tenant) SetContact(contactName, phone, email string) error {
	if contactName != "" && len(contactName) > 100 {
		return shared.NewDomainError("INVALID_CONTACT_NAME", "Contact name cannot exceed 100 characters")
	}
	if phone != "" && len(phone) > 50 {
		return shared.NewDomainError("INVALID_PHONE", "Phone cannot exceed 50 characters")
	}
	if email != "" && len(email) > 200 {
		return shared.NewDomainError("INVALID_EMAIL", "Email cannot exceed 200 characters")
	}

	t.ContactName = contactName
	t.ContactPhone = phone
	t.ContactEmail = email
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	return nil
}

// SetAddress sets the tenant's address
func (t *Tenant) SetAddress(address string) error {
	if address != "" && len(address) > 500 {
		return shared.NewDomainError("INVALID_ADDRESS", "Address cannot exceed 500 characters")
	}

	t.Address = address
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	return nil
}

// SetLogoURL sets the tenant's logo URL
func (t *Tenant) SetLogoURL(url string) error {
	if url != "" && len(url) > 500 {
		return shared.NewDomainError("INVALID_URL", "Logo URL cannot exceed 500 characters")
	}

	t.LogoURL = url
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	return nil
}

// SetDomain sets the tenant's custom domain/subdomain
func (t *Tenant) SetDomain(domain string) error {
	if domain != "" && len(domain) > 200 {
		return shared.NewDomainError("INVALID_DOMAIN", "Domain cannot exceed 200 characters")
	}
	if domain != "" {
		domain = strings.ToLower(strings.TrimSpace(domain))
	}

	t.Domain = domain
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	return nil
}

// SetPlan sets the tenant's subscription plan
func (t *Tenant) SetPlan(plan TenantPlan) error {
	if err := validateTenantPlan(plan); err != nil {
		return err
	}

	oldPlan := t.Plan
	t.Plan = plan
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	// If upgrading from trial, clear trial status
	if t.Status == TenantStatusTrial && plan != TenantPlanFree {
		t.Status = TenantStatusActive
		t.TrialEndsAt = nil
	}

	// Update config limits based on plan
	t.updateConfigForPlan(plan)

	t.AddDomainEvent(NewTenantPlanChangedEvent(t, oldPlan, plan))

	return nil
}

// updateConfigForPlan updates configuration limits based on the plan
func (t *Tenant) updateConfigForPlan(plan TenantPlan) {
	switch plan {
	case TenantPlanFree:
		t.Config.MaxUsers = 5
		t.Config.MaxWarehouses = 3
		t.Config.MaxProducts = 1000
	case TenantPlanBasic:
		t.Config.MaxUsers = 10
		t.Config.MaxWarehouses = 5
		t.Config.MaxProducts = 5000
	case TenantPlanPro:
		t.Config.MaxUsers = 50
		t.Config.MaxWarehouses = 20
		t.Config.MaxProducts = 50000
	case TenantPlanEnterprise:
		t.Config.MaxUsers = 9999
		t.Config.MaxWarehouses = 9999
		t.Config.MaxProducts = 999999
	}
}

// SetExpiration sets the subscription expiration date
func (t *Tenant) SetExpiration(expiresAt time.Time) {
	t.ExpiresAt = &expiresAt
	t.UpdatedAt = time.Now()
	t.IncrementVersion()
}

// ClearExpiration clears the expiration date (e.g., for lifetime plans)
func (t *Tenant) ClearExpiration() {
	t.ExpiresAt = nil
	t.UpdatedAt = time.Now()
	t.IncrementVersion()
}

// UpdateConfig updates the tenant's configuration
func (t *Tenant) UpdateConfig(config TenantConfig) error {
	if config.MaxUsers < 0 {
		return shared.NewDomainError("INVALID_MAX_USERS", "Max users cannot be negative")
	}
	if config.MaxWarehouses < 0 {
		return shared.NewDomainError("INVALID_MAX_WAREHOUSES", "Max warehouses cannot be negative")
	}
	if config.MaxProducts < 0 {
		return shared.NewDomainError("INVALID_MAX_PRODUCTS", "Max products cannot be negative")
	}

	t.Config = config
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	return nil
}

// SetNotes sets the tenant's notes
func (t *Tenant) SetNotes(notes string) {
	t.Notes = notes
	t.UpdatedAt = time.Now()
	t.IncrementVersion()
}

// Activate activates the tenant
func (t *Tenant) Activate() error {
	if t.Status == TenantStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Tenant is already active")
	}

	oldStatus := t.Status
	t.Status = TenantStatusActive
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewTenantStatusChangedEvent(t, oldStatus, TenantStatusActive))

	return nil
}

// Deactivate deactivates the tenant
func (t *Tenant) Deactivate() error {
	if t.Status == TenantStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Tenant is already inactive")
	}

	oldStatus := t.Status
	t.Status = TenantStatusInactive
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewTenantStatusChangedEvent(t, oldStatus, TenantStatusInactive))

	return nil
}

// Suspend suspends the tenant (e.g., due to payment issues)
func (t *Tenant) Suspend() error {
	if t.Status == TenantStatusSuspended {
		return shared.NewDomainError("ALREADY_SUSPENDED", "Tenant is already suspended")
	}

	oldStatus := t.Status
	t.Status = TenantStatusSuspended
	t.UpdatedAt = time.Now()
	t.IncrementVersion()

	t.AddDomainEvent(NewTenantStatusChangedEvent(t, oldStatus, TenantStatusSuspended))

	return nil
}

// ConvertFromTrial converts a trial tenant to a paid tenant
func (t *Tenant) ConvertFromTrial(plan TenantPlan) error {
	if t.Status != TenantStatusTrial {
		return shared.NewDomainError("NOT_TRIAL", "Tenant is not in trial status")
	}
	if plan == TenantPlanFree {
		return shared.NewDomainError("INVALID_PLAN", "Cannot convert to free plan from trial")
	}

	return t.SetPlan(plan)
}

// IsActive returns true if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsInactive returns true if the tenant is inactive
func (t *Tenant) IsInactive() bool {
	return t.Status == TenantStatusInactive
}

// IsSuspended returns true if the tenant is suspended
func (t *Tenant) IsSuspended() bool {
	return t.Status == TenantStatusSuspended
}

// IsTrial returns true if the tenant is in trial period
func (t *Tenant) IsTrial() bool {
	return t.Status == TenantStatusTrial
}

// IsTrialExpired returns true if the trial has expired
func (t *Tenant) IsTrialExpired() bool {
	if t.Status != TenantStatusTrial {
		return false
	}
	if t.TrialEndsAt == nil {
		return false
	}
	return time.Now().After(*t.TrialEndsAt)
}

// IsSubscriptionExpired returns true if the subscription has expired
func (t *Tenant) IsSubscriptionExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// CanAddUser returns true if the tenant can add more users
func (t *Tenant) CanAddUser(currentUserCount int) bool {
	return currentUserCount < t.Config.MaxUsers
}

// CanAddWarehouse returns true if the tenant can add more warehouses
func (t *Tenant) CanAddWarehouse(currentWarehouseCount int) bool {
	return currentWarehouseCount < t.Config.MaxWarehouses
}

// CanAddProduct returns true if the tenant can add more products
func (t *Tenant) CanAddProduct(currentProductCount int) bool {
	return currentProductCount < t.Config.MaxProducts
}

// GetID returns the tenant ID (implements a helper for getting UUID)
func (t *Tenant) GetTenantID() uuid.UUID {
	return t.ID
}

// Validation functions

func validateTenantCode(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_CODE", "Tenant code cannot be empty")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_CODE", "Tenant code cannot exceed 50 characters")
	}
	for _, r := range code {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return shared.NewDomainError("INVALID_CODE", "Tenant code can only contain letters, numbers, underscores, and hyphens")
		}
	}
	return nil
}

func validateTenantName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Tenant name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_NAME", "Tenant name cannot exceed 200 characters")
	}
	return nil
}

func validateTenantPlan(plan TenantPlan) error {
	switch plan {
	case TenantPlanFree, TenantPlanBasic, TenantPlanPro, TenantPlanEnterprise:
		return nil
	default:
		return shared.NewDomainError("INVALID_PLAN", "Invalid tenant plan")
	}
}
