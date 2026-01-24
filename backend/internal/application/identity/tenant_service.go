package identity

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TenantService handles tenant management operations
type TenantService struct {
	tenantRepo identity.TenantRepository
	logger     *zap.Logger
}

// NewTenantService creates a new tenant service
func NewTenantService(
	tenantRepo identity.TenantRepository,
	logger *zap.Logger,
) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		logger:     logger,
	}
}

// CreateTenantInput contains input for creating a tenant
type CreateTenantInput struct {
	Code         string
	Name         string
	ShortName    string
	ContactName  string
	ContactPhone string
	ContactEmail string
	Address      string
	LogoURL      string
	Domain       string
	Plan         string
	Notes        string
	TrialDays    int // If > 0, creates a trial tenant
}

// UpdateTenantInput contains input for updating a tenant
type UpdateTenantInput struct {
	ID           uuid.UUID
	Name         *string
	ShortName    *string
	ContactName  *string
	ContactPhone *string
	ContactEmail *string
	Address      *string
	LogoURL      *string
	Domain       *string
	Notes        *string
}

// TenantConfigInput contains input for updating tenant configuration
type TenantConfigInput struct {
	MaxUsers      *int
	MaxWarehouses *int
	MaxProducts   *int
	CostStrategy  *string
	Currency      *string
	Timezone      *string
	Locale        *string
}

// TenantDTO represents tenant data transfer object
type TenantDTO struct {
	ID           uuid.UUID        `json:"id"`
	Code         string           `json:"code"`
	Name         string           `json:"name"`
	ShortName    string           `json:"short_name,omitempty"`
	Status       string           `json:"status"`
	Plan         string           `json:"plan"`
	ContactName  string           `json:"contact_name,omitempty"`
	ContactPhone string           `json:"contact_phone,omitempty"`
	ContactEmail string           `json:"contact_email,omitempty"`
	Address      string           `json:"address,omitempty"`
	LogoURL      string           `json:"logo_url,omitempty"`
	Domain       string           `json:"domain,omitempty"`
	ExpiresAt    *time.Time       `json:"expires_at,omitempty"`
	TrialEndsAt  *time.Time       `json:"trial_ends_at,omitempty"`
	Config       TenantConfigDTO  `json:"config"`
	Notes        string           `json:"notes,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// TenantConfigDTO represents tenant configuration
type TenantConfigDTO struct {
	MaxUsers      int    `json:"max_users"`
	MaxWarehouses int    `json:"max_warehouses"`
	MaxProducts   int    `json:"max_products"`
	CostStrategy  string `json:"cost_strategy"`
	Currency      string `json:"currency"`
	Timezone      string `json:"timezone"`
	Locale        string `json:"locale"`
}

// TenantFilter represents filter for querying tenants
type TenantFilter struct {
	Page     int
	PageSize int
	SortBy   string
	SortDir  string
	Keyword  string
	Status   string
	Plan     string
}

// ToSharedFilter converts TenantFilter to shared.Filter
func (f TenantFilter) ToSharedFilter() shared.Filter {
	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return shared.Filter{
		Page:     page,
		PageSize: pageSize,
		OrderBy:  f.SortBy,
		OrderDir: f.SortDir,
		Search:   f.Keyword,
	}
}

// TenantListResult represents paginated tenant list result
type TenantListResult struct {
	Tenants    []TenantDTO `json:"tenants"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// Create creates a new tenant
func (s *TenantService) Create(ctx context.Context, input CreateTenantInput) (*TenantDTO, error) {
	s.logger.Info("Creating new tenant",
		zap.String("code", input.Code),
		zap.String("name", input.Name))

	// Check if code already exists
	exists, err := s.tenantRepo.ExistsByCode(ctx, input.Code)
	if err != nil {
		s.logger.Error("Failed to check tenant code existence", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check code availability")
	}
	if exists {
		return nil, shared.NewDomainError("CODE_EXISTS", "Tenant code already exists")
	}

	// Check domain uniqueness if provided
	if input.Domain != "" {
		exists, err := s.tenantRepo.ExistsByDomain(ctx, input.Domain)
		if err != nil {
			s.logger.Error("Failed to check domain existence", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check domain availability")
		}
		if exists {
			return nil, shared.NewDomainError("DOMAIN_EXISTS", "Domain already exists")
		}
	}

	var tenant *identity.Tenant
	if input.TrialDays > 0 {
		tenant, err = identity.NewTrialTenant(input.Code, input.Name, input.TrialDays)
	} else {
		tenant, err = identity.NewTenant(input.Code, input.Name)
	}
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if input.ShortName != "" {
		tenant.ShortName = input.ShortName
	}
	if input.ContactName != "" || input.ContactPhone != "" || input.ContactEmail != "" {
		if err := tenant.SetContact(input.ContactName, input.ContactPhone, input.ContactEmail); err != nil {
			return nil, err
		}
	}
	if input.Address != "" {
		if err := tenant.SetAddress(input.Address); err != nil {
			return nil, err
		}
	}
	if input.LogoURL != "" {
		if err := tenant.SetLogoURL(input.LogoURL); err != nil {
			return nil, err
		}
	}
	if input.Domain != "" {
		if err := tenant.SetDomain(input.Domain); err != nil {
			return nil, err
		}
	}
	if input.Plan != "" {
		plan := identity.TenantPlan(input.Plan)
		if err := tenant.SetPlan(plan); err != nil {
			return nil, err
		}
	}
	if input.Notes != "" {
		tenant.SetNotes(input.Notes)
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to create tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to create tenant")
	}

	s.logger.Info("Tenant created successfully",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("code", tenant.Code))

	return toTenantDTO(tenant), nil
}

// GetByID retrieves a tenant by ID
func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}
	return toTenantDTO(tenant), nil
}

// GetByCode retrieves a tenant by code
func (s *TenantService) GetByCode(ctx context.Context, code string) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByCode(ctx, code)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant by code", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}
	return toTenantDTO(tenant), nil
}

// List retrieves a paginated list of tenants
func (s *TenantService) List(ctx context.Context, filter TenantFilter) (*TenantListResult, error) {
	sharedFilter := filter.ToSharedFilter()

	var tenants []identity.Tenant
	var total int64
	var err error

	if filter.Status != "" {
		status := identity.TenantStatus(filter.Status)
		tenants, err = s.tenantRepo.FindByStatus(ctx, status, sharedFilter)
		if err != nil {
			s.logger.Error("Failed to list tenants by status", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list tenants")
		}
		total, err = s.tenantRepo.CountByStatus(ctx, status)
	} else if filter.Plan != "" {
		plan := identity.TenantPlan(filter.Plan)
		tenants, err = s.tenantRepo.FindByPlan(ctx, plan, sharedFilter)
		if err != nil {
			s.logger.Error("Failed to list tenants by plan", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list tenants")
		}
		total, err = s.tenantRepo.CountByPlan(ctx, plan)
	} else {
		tenants, err = s.tenantRepo.FindAll(ctx, sharedFilter)
		if err != nil {
			s.logger.Error("Failed to list tenants", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list tenants")
		}
		total, err = s.tenantRepo.Count(ctx, sharedFilter)
	}

	if err != nil {
		s.logger.Error("Failed to count tenants", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to count tenants")
	}

	// Calculate total pages
	pageSize := sharedFilter.PageSize
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	tenantDTOs := make([]TenantDTO, len(tenants))
	for i, tenant := range tenants {
		tenantDTOs[i] = *toTenantDTO(&tenant)
	}

	return &TenantListResult{
		Tenants:    tenantDTOs,
		Total:      total,
		Page:       sharedFilter.Page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Update updates a tenant's information
func (s *TenantService) Update(ctx context.Context, input UpdateTenantInput) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, input.ID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Update basic info
	if input.Name != nil || input.ShortName != nil {
		name := tenant.Name
		shortName := tenant.ShortName
		if input.Name != nil {
			name = *input.Name
		}
		if input.ShortName != nil {
			shortName = *input.ShortName
		}
		if err := tenant.Update(name, shortName); err != nil {
			return nil, err
		}
	}

	// Update contact info
	if input.ContactName != nil || input.ContactPhone != nil || input.ContactEmail != nil {
		contactName := tenant.ContactName
		contactPhone := tenant.ContactPhone
		contactEmail := tenant.ContactEmail
		if input.ContactName != nil {
			contactName = *input.ContactName
		}
		if input.ContactPhone != nil {
			contactPhone = *input.ContactPhone
		}
		if input.ContactEmail != nil {
			contactEmail = *input.ContactEmail
		}
		if err := tenant.SetContact(contactName, contactPhone, contactEmail); err != nil {
			return nil, err
		}
	}

	if input.Address != nil {
		if err := tenant.SetAddress(*input.Address); err != nil {
			return nil, err
		}
	}

	if input.LogoURL != nil {
		if err := tenant.SetLogoURL(*input.LogoURL); err != nil {
			return nil, err
		}
	}

	if input.Domain != nil {
		// Check domain uniqueness if changed
		if *input.Domain != tenant.Domain && *input.Domain != "" {
			exists, err := s.tenantRepo.ExistsByDomain(ctx, *input.Domain)
			if err != nil {
				return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check domain availability")
			}
			if exists {
				return nil, shared.NewDomainError("DOMAIN_EXISTS", "Domain already exists")
			}
		}
		if err := tenant.SetDomain(*input.Domain); err != nil {
			return nil, err
		}
	}

	if input.Notes != nil {
		tenant.SetNotes(*input.Notes)
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to update tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update tenant")
	}

	s.logger.Info("Tenant updated", zap.String("tenant_id", input.ID.String()))

	return toTenantDTO(tenant), nil
}

// UpdateConfig updates a tenant's configuration
func (s *TenantService) UpdateConfig(ctx context.Context, id uuid.UUID, input TenantConfigInput) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	config := tenant.Config
	if input.MaxUsers != nil {
		config.MaxUsers = *input.MaxUsers
	}
	if input.MaxWarehouses != nil {
		config.MaxWarehouses = *input.MaxWarehouses
	}
	if input.MaxProducts != nil {
		config.MaxProducts = *input.MaxProducts
	}
	if input.CostStrategy != nil {
		config.CostStrategy = *input.CostStrategy
	}
	if input.Currency != nil {
		config.Currency = *input.Currency
	}
	if input.Timezone != nil {
		config.Timezone = *input.Timezone
	}
	if input.Locale != nil {
		config.Locale = *input.Locale
	}

	if err := tenant.UpdateConfig(config); err != nil {
		return nil, err
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to update tenant config", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update tenant config")
	}

	s.logger.Info("Tenant config updated", zap.String("tenant_id", id.String()))

	return toTenantDTO(tenant), nil
}

// SetPlan updates a tenant's subscription plan
func (s *TenantService) SetPlan(ctx context.Context, id uuid.UUID, plan string) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	tenantPlan := identity.TenantPlan(plan)
	if err := tenant.SetPlan(tenantPlan); err != nil {
		return nil, err
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to update tenant plan", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update tenant plan")
	}

	s.logger.Info("Tenant plan updated",
		zap.String("tenant_id", id.String()),
		zap.String("plan", plan))

	return toTenantDTO(tenant), nil
}

// Activate activates a tenant
func (s *TenantService) Activate(ctx context.Context, id uuid.UUID) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	if err := tenant.Activate(); err != nil {
		return nil, err
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to activate tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to activate tenant")
	}

	s.logger.Info("Tenant activated", zap.String("tenant_id", id.String()))

	return toTenantDTO(tenant), nil
}

// Deactivate deactivates a tenant
func (s *TenantService) Deactivate(ctx context.Context, id uuid.UUID) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	if err := tenant.Deactivate(); err != nil {
		return nil, err
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to deactivate tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to deactivate tenant")
	}

	s.logger.Info("Tenant deactivated", zap.String("tenant_id", id.String()))

	return toTenantDTO(tenant), nil
}

// Suspend suspends a tenant
func (s *TenantService) Suspend(ctx context.Context, id uuid.UUID) (*TenantDTO, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	if err := tenant.Suspend(); err != nil {
		return nil, err
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to suspend tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to suspend tenant")
	}

	s.logger.Info("Tenant suspended", zap.String("tenant_id", id.String()))

	return toTenantDTO(tenant), nil
}

// Delete deletes a tenant
func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Only inactive tenants can be deleted
	if !tenant.IsInactive() {
		return shared.NewDomainError("TENANT_NOT_INACTIVE", "Only inactive tenants can be deleted")
	}

	if err := s.tenantRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete tenant", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to delete tenant")
	}

	s.logger.Info("Tenant deleted", zap.String("tenant_id", id.String()))

	return nil
}

// Count returns the total number of tenants
func (s *TenantService) Count(ctx context.Context) (int64, error) {
	return s.tenantRepo.Count(ctx, shared.DefaultFilter())
}

// GetStats returns tenant statistics
func (s *TenantService) GetStats(ctx context.Context) (*TenantStatsDTO, error) {
	activeCount, err := s.tenantRepo.CountByStatus(ctx, identity.TenantStatusActive)
	if err != nil {
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get stats")
	}

	trialCount, err := s.tenantRepo.CountByStatus(ctx, identity.TenantStatusTrial)
	if err != nil {
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get stats")
	}

	inactiveCount, err := s.tenantRepo.CountByStatus(ctx, identity.TenantStatusInactive)
	if err != nil {
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get stats")
	}

	suspendedCount, err := s.tenantRepo.CountByStatus(ctx, identity.TenantStatusSuspended)
	if err != nil {
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get stats")
	}

	total, err := s.tenantRepo.Count(ctx, shared.DefaultFilter())
	if err != nil {
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get stats")
	}

	return &TenantStatsDTO{
		Total:     total,
		Active:    activeCount,
		Trial:     trialCount,
		Inactive:  inactiveCount,
		Suspended: suspendedCount,
	}, nil
}

// TenantStatsDTO represents tenant statistics
type TenantStatsDTO struct {
	Total     int64 `json:"total"`
	Active    int64 `json:"active"`
	Trial     int64 `json:"trial"`
	Inactive  int64 `json:"inactive"`
	Suspended int64 `json:"suspended"`
}

// toTenantDTO converts domain Tenant to TenantDTO
func toTenantDTO(tenant *identity.Tenant) *TenantDTO {
	return &TenantDTO{
		ID:           tenant.ID,
		Code:         tenant.Code,
		Name:         tenant.Name,
		ShortName:    tenant.ShortName,
		Status:       string(tenant.Status),
		Plan:         string(tenant.Plan),
		ContactName:  tenant.ContactName,
		ContactPhone: tenant.ContactPhone,
		ContactEmail: tenant.ContactEmail,
		Address:      tenant.Address,
		LogoURL:      tenant.LogoURL,
		Domain:       tenant.Domain,
		ExpiresAt:    tenant.ExpiresAt,
		TrialEndsAt:  tenant.TrialEndsAt,
		Config: TenantConfigDTO{
			MaxUsers:      tenant.Config.MaxUsers,
			MaxWarehouses: tenant.Config.MaxWarehouses,
			MaxProducts:   tenant.Config.MaxProducts,
			CostStrategy:  tenant.Config.CostStrategy,
			Currency:      tenant.Config.Currency,
			Timezone:      tenant.Config.Timezone,
			Locale:        tenant.Config.Locale,
		},
		Notes:     tenant.Notes,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}
}
