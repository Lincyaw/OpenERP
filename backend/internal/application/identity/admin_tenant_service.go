package identity

import (
	"context"
	"regexp"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AdminTenantService handles tenant management operations for super admins.
// This service provides CRUD operations that bypass tenant isolation.
// IMPORTANT: All methods must be protected by SuperAdminMiddleware.
type AdminTenantService struct {
	adminRepo           identity.AdminTenantRepository
	tenantRepo          identity.TenantRepository
	subscriptionHistory identity.SubscriptionHistoryRepository
	statusHistory       identity.TenantStatusHistoryRepository
	auditService        *AuditService
	logger              *zap.Logger
}

// NewAdminTenantService creates a new admin tenant service
func NewAdminTenantService(
	adminRepo identity.AdminTenantRepository,
	tenantRepo identity.TenantRepository,
	subscriptionHistory identity.SubscriptionHistoryRepository,
	statusHistory identity.TenantStatusHistoryRepository,
	auditService *AuditService,
	logger *zap.Logger,
) *AdminTenantService {
	return &AdminTenantService{
		adminRepo:           adminRepo,
		tenantRepo:          tenantRepo,
		subscriptionHistory: subscriptionHistory,
		statusHistory:       statusHistory,
		auditService:        auditService,
		logger:              logger,
	}
}

// AdminCreateTenantInput contains input for creating a tenant by super admin
type AdminCreateTenantInput struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	ShortName    string `json:"short_name,omitempty"`
	ContactName  string `json:"contact_name,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	Address      string `json:"address,omitempty"`
	Plan         string `json:"plan,omitempty"`
	TrialDays    int    `json:"trial_days,omitempty"` // If > 0, creates a trial tenant
	Notes        string `json:"notes,omitempty"`
}

// AdminUpdateTenantInput contains input for updating a tenant by super admin
type AdminUpdateTenantInput struct {
	Name         *string `json:"name,omitempty"`
	ShortName    *string `json:"short_name,omitempty"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	Address      *string `json:"address,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

// AdminTenantDTO represents tenant data for admin operations
type AdminTenantDTO struct {
	ID                       uuid.UUID       `json:"id"`
	Code                     string          `json:"code"`
	Name                     string          `json:"name"`
	ShortName                string          `json:"short_name,omitempty"`
	Status                   string          `json:"status"`
	Plan                     string          `json:"plan"`
	ScheduledPlan            *string         `json:"scheduled_plan,omitempty"`
	ScheduledPlanEffectiveAt *time.Time      `json:"scheduled_plan_effective_at,omitempty"`
	ContactName              string          `json:"contact_name,omitempty"`
	ContactPhone             string          `json:"contact_phone,omitempty"`
	ContactEmail             string          `json:"contact_email,omitempty"`
	Address                  string          `json:"address,omitempty"`
	LogoURL                  string          `json:"logo_url,omitempty"`
	Domain                   string          `json:"domain,omitempty"`
	ExpiresAt                *time.Time      `json:"expires_at,omitempty"`
	TrialEndsAt              *time.Time      `json:"trial_ends_at,omitempty"`
	Config                   TenantConfigDTO `json:"config"`
	Notes                    string          `json:"notes,omitempty"`
	// Suspension-related fields
	SuspensionReason      string               `json:"suspension_reason,omitempty"`
	SuspendedAt           *time.Time           `json:"suspended_at,omitempty"`
	ScheduledReactivateAt *time.Time           `json:"scheduled_reactivate_at,omitempty"`
	Statistics            *AdminTenantStatsDTO `json:"statistics,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

// AdminTenantStatsDTO represents usage statistics for a tenant
type AdminTenantStatsDTO struct {
	UserCount      int64 `json:"user_count"`
	WarehouseCount int64 `json:"warehouse_count"`
	ProductCount   int64 `json:"product_count"`
	OrderCount     int64 `json:"order_count"`
}

// AdminTenantListResult represents paginated tenant list result for admin
type AdminTenantListResult struct {
	Tenants    []AdminTenantDTO `json:"tenants"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// AdminTenantFilterInput contains filter options for admin tenant queries
type AdminTenantFilterInput struct {
	Page          int        `json:"page"`
	PageSize      int        `json:"page_size"`
	OrderBy       string     `json:"order_by,omitempty"`
	OrderDir      string     `json:"order_dir,omitempty"`
	Search        string     `json:"search,omitempty"`
	Status        *string    `json:"status,omitempty"`
	Plan          *string    `json:"plan,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}

// AuditContext contains context information for audit logging
type AuditContext struct {
	AdminUserID uuid.UUID
	IPAddress   string
	UserAgent   string
}

// SuspendTenantInput contains input for suspending a tenant
type SuspendTenantInput struct {
	TenantID              uuid.UUID  `json:"tenant_id"`
	Reason                string     `json:"reason"`                            // Reason for suspension
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty"` // Optional: auto-reactivate at this time
}

// TenantStatusDTO represents the current status of a tenant
type TenantStatusDTO struct {
	TenantID              uuid.UUID                `json:"tenant_id"`
	Status                string                   `json:"status"`
	SuspensionReason      string                   `json:"suspension_reason,omitempty"`
	SuspendedAt           *time.Time               `json:"suspended_at,omitempty"`
	ScheduledReactivateAt *time.Time               `json:"scheduled_reactivate_at,omitempty"`
	History               []TenantStatusHistoryDTO `json:"history,omitempty"`
}

// TenantStatusHistoryDTO represents a status change history entry
type TenantStatusHistoryDTO struct {
	ID                    uuid.UUID  `json:"id"`
	ChangeType            string     `json:"change_type"`
	OldStatus             string     `json:"old_status"`
	NewStatus             string     `json:"new_status"`
	Reason                string     `json:"reason,omitempty"`
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty"`
	ChangedByUserID       uuid.UUID  `json:"changed_by_user_id"`
	CreatedAt             time.Time  `json:"created_at"`
}

// TenantStatusHistoryFilterInput contains filter options for status history queries
type TenantStatusHistoryFilterInput struct {
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	ChangeType *string    `json:"change_type,omitempty"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
}

// TenantStatusHistoryListResult represents paginated status history results
type TenantStatusHistoryListResult struct {
	History    []TenantStatusHistoryDTO `json:"history"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

// CreateTenant creates a new tenant
func (s *AdminTenantService) CreateTenant(ctx context.Context, input AdminCreateTenantInput, auditCtx AuditContext) (*AdminTenantDTO, error) {
	s.logger.Info("Admin creating new tenant",
		zap.String("code", input.Code),
		zap.String("name", input.Name),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Validate input
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	// Check if code already exists
	exists, err := s.tenantRepo.ExistsByCode(ctx, input.Code)
	if err != nil {
		s.logger.Error("Failed to check tenant code existence", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check code availability")
	}
	if exists {
		return nil, shared.NewDomainError("CODE_EXISTS", "Tenant code already exists")
	}

	// Check name uniqueness
	if err := s.checkNameUniqueness(ctx, input.Name, nil); err != nil {
		return nil, err
	}

	// Create tenant
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

	// Record audit log
	if err := s.auditService.RecordTenantCreate(ctx, auditCtx.AdminUserID, tenant, auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for tenant creation", zap.Error(err))
		// Don't fail the operation for audit log failure
	}

	s.logger.Info("Tenant created successfully by admin",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("code", tenant.Code),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return toAdminTenantDTO(tenant, nil), nil
}

// UpdateTenant updates a tenant's information
func (s *AdminTenantService) UpdateTenant(ctx context.Context, id uuid.UUID, input AdminUpdateTenantInput, auditCtx AuditContext) (*AdminTenantDTO, error) {
	s.logger.Info("Admin updating tenant",
		zap.String("tenant_id", id.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Find tenant
	tenant, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Store old state for audit
	oldTenant := *tenant

	// Validate and apply updates
	if input.Name != nil {
		if err := s.validateName(*input.Name); err != nil {
			return nil, err
		}
		// Check name uniqueness (excluding current tenant)
		if err := s.checkNameUniqueness(ctx, *input.Name, &id); err != nil {
			return nil, err
		}
	}

	if input.ContactEmail != nil && *input.ContactEmail != "" {
		if err := s.validateEmail(*input.ContactEmail); err != nil {
			return nil, err
		}
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

	if input.Notes != nil {
		tenant.SetNotes(*input.Notes)
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to update tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update tenant")
	}

	// Record audit log
	if err := s.auditService.RecordTenantUpdate(ctx, auditCtx.AdminUserID, &oldTenant, tenant, auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for tenant update", zap.Error(err))
	}

	s.logger.Info("Tenant updated successfully by admin",
		zap.String("tenant_id", id.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return toAdminTenantDTO(tenant, nil), nil
}

// DeleteTenant performs a soft delete on a tenant
func (s *AdminTenantService) DeleteTenant(ctx context.Context, id uuid.UUID, auditCtx AuditContext) error {
	s.logger.Info("Admin deleting tenant (soft delete)",
		zap.String("tenant_id", id.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Find tenant
	tenant, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Prevent deletion of system tenant
	if tenant.Code == identity.SystemTenantCode {
		return shared.NewDomainError("CANNOT_DELETE_SYSTEM_TENANT", "Cannot delete system tenant")
	}

	// Store old state for audit
	oldTenant := *tenant

	// Soft delete: mark as inactive (deactivate)
	// This preserves data for potential recovery
	if err := tenant.Deactivate(); err != nil {
		// If already inactive, that's fine for deletion
		if tenant.Status != identity.TenantStatusInactive {
			return err
		}
	}

	// Save the deactivated tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to soft delete tenant", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to delete tenant")
	}

	// Record audit log
	if err := s.auditService.RecordTenantDelete(ctx, auditCtx.AdminUserID, &oldTenant, auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for tenant deletion", zap.Error(err))
	}

	s.logger.Info("Tenant soft deleted successfully by admin",
		zap.String("tenant_id", id.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return nil
}

// GetTenant retrieves a tenant by ID with usage statistics
func (s *AdminTenantService) GetTenant(ctx context.Context, id uuid.UUID) (*AdminTenantDTO, error) {
	s.logger.Debug("Admin getting tenant details",
		zap.String("tenant_id", id.String()))

	tenant, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// TODO: Get usage statistics from respective repositories
	// For now, return nil statistics - this can be enhanced later
	// when we have access to user, warehouse, product, and order repositories
	stats := &AdminTenantStatsDTO{
		UserCount:      0,
		WarehouseCount: 0,
		ProductCount:   0,
		OrderCount:     0,
	}

	return toAdminTenantDTO(tenant, stats), nil
}

// ListTenants retrieves a paginated list of tenants
func (s *AdminTenantService) ListTenants(ctx context.Context, input AdminTenantFilterInput) (*AdminTenantListResult, error) {
	s.logger.Debug("Admin listing tenants",
		zap.Int("page", input.Page),
		zap.Int("page_size", input.PageSize))

	// Build filter
	filter := identity.DefaultAdminTenantFilter()
	if input.Page > 0 {
		filter.Page = input.Page
	}
	if input.PageSize > 0 {
		filter.PageSize = input.PageSize
		if filter.PageSize > 100 {
			filter.PageSize = 100
		}
	}
	if input.OrderBy != "" {
		filter.OrderBy = input.OrderBy
	}
	if input.OrderDir != "" {
		filter.OrderDir = input.OrderDir
	}
	if input.Search != "" {
		filter.Search = input.Search
	}
	if input.Status != nil {
		status := identity.TenantStatus(*input.Status)
		filter.Status = &status
	}
	if input.Plan != nil {
		plan := identity.TenantPlan(*input.Plan)
		filter.PlanType = &plan
	}
	if input.CreatedAfter != nil {
		filter.CreatedAfter = input.CreatedAfter
	}
	if input.CreatedBefore != nil {
		filter.CreatedBefore = input.CreatedBefore
	}

	// Get tenants
	tenants, err := s.adminRepo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list tenants", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list tenants")
	}

	// Get total count
	total, err := s.adminRepo.Count(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to count tenants", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to count tenants")
	}

	// Calculate total pages
	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	// Convert to DTOs
	tenantDTOs := make([]AdminTenantDTO, len(tenants))
	for i, tenant := range tenants {
		tenantDTOs[i] = *toAdminTenantDTO(&tenant, nil)
	}

	return &AdminTenantListResult{
		Tenants:    tenantDTOs,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetStatistics returns aggregated statistics for all tenants
func (s *AdminTenantService) GetStatistics(ctx context.Context) (*identity.TenantStatistics, error) {
	s.logger.Debug("Admin getting tenant statistics")

	stats, err := s.adminRepo.GetStatistics(ctx)
	if err != nil {
		s.logger.Error("Failed to get tenant statistics", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get statistics")
	}

	return stats, nil
}

// ActivateTenant activates a tenant
func (s *AdminTenantService) ActivateTenant(ctx context.Context, id uuid.UUID, auditCtx AuditContext) (*AdminTenantDTO, error) {
	s.logger.Info("Admin activating tenant",
		zap.String("tenant_id", id.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	tenant, err := s.adminRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	oldTenant := *tenant
	oldStatus := tenant.Status

	if err := tenant.Activate(); err != nil {
		return nil, err
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to activate tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to activate tenant")
	}

	// Record status history
	if s.statusHistory != nil {
		historyBuilder := identity.NewTenantStatusHistory(tenant.ID, auditCtx.AdminUserID, identity.TenantStatusChangeTypeActivate).
			WithStatusChange(oldStatus, identity.TenantStatusActive).
			WithIPAddress(auditCtx.IPAddress).
			WithUserAgent(auditCtx.UserAgent)

		history, err := historyBuilder.Build()
		if err != nil {
			s.logger.Warn("Failed to build status history", zap.Error(err))
		} else {
			if err := s.statusHistory.Create(ctx, history); err != nil {
				s.logger.Warn("Failed to save status history", zap.Error(err))
			}
		}
	}

	// Record audit log
	if err := s.auditService.RecordTenantActivate(ctx, auditCtx.AdminUserID, &oldTenant, tenant, auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for tenant activation", zap.Error(err))
	}

	s.logger.Info("Tenant activated successfully by admin",
		zap.String("tenant_id", id.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return toAdminTenantDTO(tenant, nil), nil
}

// SuspendTenant suspends a tenant (simple version without reason)
func (s *AdminTenantService) SuspendTenant(ctx context.Context, id uuid.UUID, auditCtx AuditContext) (*AdminTenantDTO, error) {
	return s.SuspendTenantWithReason(ctx, SuspendTenantInput{
		TenantID: id,
		Reason:   "",
	}, auditCtx)
}

// SuspendTenantWithReason suspends a tenant with a reason and optional scheduled reactivation
func (s *AdminTenantService) SuspendTenantWithReason(ctx context.Context, input SuspendTenantInput, auditCtx AuditContext) (*AdminTenantDTO, error) {
	s.logger.Info("Admin suspending tenant",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("reason", input.Reason),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	tenant, err := s.adminRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Prevent suspension of system tenant
	if tenant.Code == identity.SystemTenantCode {
		return nil, shared.NewDomainError("CANNOT_SUSPEND_SYSTEM_TENANT", "Cannot suspend system tenant")
	}

	oldTenant := *tenant
	oldStatus := tenant.Status

	// Use the enhanced suspend method with reason and scheduled reactivation
	if err := tenant.SuspendWithReason(input.Reason, input.ScheduledReactivateAt); err != nil {
		return nil, err
	}

	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to suspend tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to suspend tenant")
	}

	// Record status history
	if s.statusHistory != nil {
		historyBuilder := identity.NewTenantStatusHistory(tenant.ID, auditCtx.AdminUserID, identity.TenantStatusChangeTypeSuspend).
			WithStatusChange(oldStatus, identity.TenantStatusSuspended).
			WithReason(input.Reason).
			WithScheduledReactivateAt(input.ScheduledReactivateAt).
			WithIPAddress(auditCtx.IPAddress).
			WithUserAgent(auditCtx.UserAgent)

		history, err := historyBuilder.Build()
		if err != nil {
			s.logger.Warn("Failed to build status history", zap.Error(err))
		} else {
			if err := s.statusHistory.Create(ctx, history); err != nil {
				s.logger.Warn("Failed to save status history", zap.Error(err))
			}
		}
	}

	// Record audit log
	if err := s.auditService.RecordTenantSuspend(ctx, auditCtx.AdminUserID, &oldTenant, tenant, auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for tenant suspension", zap.Error(err))
	}

	s.logger.Info("Tenant suspended successfully by admin",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("reason", input.Reason),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return toAdminTenantDTO(tenant, nil), nil
}

// GetTenantStatus retrieves the current status and history of a tenant
func (s *AdminTenantService) GetTenantStatus(ctx context.Context, tenantID uuid.UUID, historyFilter TenantStatusHistoryFilterInput) (*TenantStatusDTO, error) {
	s.logger.Debug("Admin getting tenant status",
		zap.String("tenant_id", tenantID.String()))

	tenant, err := s.adminRepo.FindByID(ctx, tenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	statusDTO := &TenantStatusDTO{
		TenantID:              tenant.ID,
		Status:                string(tenant.Status),
		SuspensionReason:      tenant.SuspensionReason,
		SuspendedAt:           tenant.SuspendedAt,
		ScheduledReactivateAt: tenant.ScheduledReactivateAt,
	}

	// Get status history if repository is available
	if s.statusHistory != nil {
		filter := identity.DefaultTenantStatusHistoryFilter()
		if historyFilter.Page > 0 {
			filter.Page = historyFilter.Page
		}
		if historyFilter.PageSize > 0 {
			filter.PageSize = historyFilter.PageSize
			if filter.PageSize > 100 {
				filter.PageSize = 100
			}
		}
		if historyFilter.ChangeType != nil {
			changeType := identity.TenantStatusChangeType(*historyFilter.ChangeType)
			filter.ChangeType = &changeType
		}
		if historyFilter.StartDate != nil {
			filter.StartDate = historyFilter.StartDate
		}
		if historyFilter.EndDate != nil {
			filter.EndDate = historyFilter.EndDate
		}

		histories, _, err := s.statusHistory.FindByTenantID(ctx, tenantID, filter)
		if err != nil {
			s.logger.Warn("Failed to get status history", zap.Error(err))
		} else {
			statusDTO.History = make([]TenantStatusHistoryDTO, len(histories))
			for i, h := range histories {
				statusDTO.History[i] = toTenantStatusHistoryDTO(&h)
			}
		}
	}

	return statusDTO, nil
}

// GetTenantStatusHistory retrieves paginated status history for a tenant
func (s *AdminTenantService) GetTenantStatusHistory(ctx context.Context, tenantID uuid.UUID, input TenantStatusHistoryFilterInput) (*TenantStatusHistoryListResult, error) {
	s.logger.Debug("Admin getting tenant status history",
		zap.String("tenant_id", tenantID.String()))

	// Verify tenant exists
	_, err := s.adminRepo.FindByID(ctx, tenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Return empty result if repository is not available
	if s.statusHistory == nil {
		return &TenantStatusHistoryListResult{
			History:    []TenantStatusHistoryDTO{},
			Total:      0,
			Page:       1,
			PageSize:   20,
			TotalPages: 0,
		}, nil
	}

	// Build filter
	filter := identity.DefaultTenantStatusHistoryFilter()
	if input.Page > 0 {
		filter.Page = input.Page
	}
	if input.PageSize > 0 {
		filter.PageSize = input.PageSize
		if filter.PageSize > 100 {
			filter.PageSize = 100
		}
	}
	if input.ChangeType != nil {
		changeType := identity.TenantStatusChangeType(*input.ChangeType)
		filter.ChangeType = &changeType
	}
	if input.StartDate != nil {
		filter.StartDate = input.StartDate
	}
	if input.EndDate != nil {
		filter.EndDate = input.EndDate
	}

	histories, total, err := s.statusHistory.FindByTenantID(ctx, tenantID, filter)
	if err != nil {
		s.logger.Error("Failed to get status history", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get status history")
	}

	// Convert to DTOs
	historyDTOs := make([]TenantStatusHistoryDTO, len(histories))
	for i, h := range histories {
		historyDTOs[i] = toTenantStatusHistoryDTO(&h)
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit()
	if int(total)%filter.Limit() > 0 {
		totalPages++
	}

	return &TenantStatusHistoryListResult{
		History:    historyDTOs,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.Limit(),
		TotalPages: totalPages,
	}, nil
}

// toTenantStatusHistoryDTO converts domain TenantStatusHistory to DTO
func toTenantStatusHistoryDTO(h *identity.TenantStatusHistory) TenantStatusHistoryDTO {
	return TenantStatusHistoryDTO{
		ID:                    h.ID,
		ChangeType:            string(h.ChangeType),
		OldStatus:             string(h.OldStatus),
		NewStatus:             string(h.NewStatus),
		Reason:                h.Reason,
		ScheduledReactivateAt: h.ScheduledReactivateAt,
		ChangedByUserID:       h.ChangedByUserID,
		CreatedAt:             h.CreatedAt,
	}
}

// Validation helpers

func (s *AdminTenantService) validateCreateInput(input AdminCreateTenantInput) error {
	if input.Code == "" {
		return shared.NewDomainError("INVALID_CODE", "Tenant code is required")
	}
	if input.Name == "" {
		return shared.NewDomainError("INVALID_NAME", "Tenant name is required")
	}
	if err := s.validateName(input.Name); err != nil {
		return err
	}
	if input.ContactEmail != "" {
		if err := s.validateEmail(input.ContactEmail); err != nil {
			return err
		}
	}
	if input.Plan != "" {
		if err := s.validatePlan(input.Plan); err != nil {
			return err
		}
	}
	return nil
}

func (s *AdminTenantService) validateName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Tenant name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_NAME", "Tenant name cannot exceed 200 characters")
	}
	return nil
}

func (s *AdminTenantService) validateEmail(email string) error {
	if len(email) > 200 {
		return shared.NewDomainError("INVALID_EMAIL", "Email cannot exceed 200 characters")
	}
	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return shared.NewDomainError("INVALID_EMAIL", "Invalid email format")
	}
	return nil
}

func (s *AdminTenantService) validatePlan(plan string) error {
	switch identity.TenantPlan(plan) {
	case identity.TenantPlanFree, identity.TenantPlanBasic, identity.TenantPlanPro, identity.TenantPlanEnterprise:
		return nil
	default:
		return shared.NewDomainError("INVALID_PLAN", "Invalid tenant plan")
	}
}

func (s *AdminTenantService) checkNameUniqueness(ctx context.Context, name string, excludeID *uuid.UUID) error {
	// Search for tenants with the same name
	filter := identity.AdminTenantFilter{
		Page:     1,
		PageSize: 10,
		Search:   name,
	}
	tenants, err := s.adminRepo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to check name uniqueness", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to check name availability")
	}

	for _, tenant := range tenants {
		// Exact match check (case-insensitive would require additional logic)
		if tenant.Name == name {
			// If we're updating, exclude the current tenant
			if excludeID != nil && tenant.ID == *excludeID {
				continue
			}
			return shared.NewDomainError("NAME_EXISTS", "Tenant name already exists")
		}
	}
	return nil
}

// toAdminTenantDTO converts domain Tenant to AdminTenantDTO
func toAdminTenantDTO(tenant *identity.Tenant, stats *AdminTenantStatsDTO) *AdminTenantDTO {
	dto := &AdminTenantDTO{
		ID:                       tenant.ID,
		Code:                     tenant.Code,
		Name:                     tenant.Name,
		ShortName:                tenant.ShortName,
		Status:                   string(tenant.Status),
		Plan:                     string(tenant.Plan),
		ScheduledPlanEffectiveAt: tenant.ScheduledPlanEffectiveAt,
		ContactName:              tenant.ContactName,
		ContactPhone:             tenant.ContactPhone,
		ContactEmail:             tenant.ContactEmail,
		Address:                  tenant.Address,
		LogoURL:                  tenant.LogoURL,
		Domain:                   tenant.Domain,
		ExpiresAt:                tenant.ExpiresAt,
		TrialEndsAt:              tenant.TrialEndsAt,
		Config: TenantConfigDTO{
			MaxUsers:      tenant.Config.MaxUsers,
			MaxWarehouses: tenant.Config.MaxWarehouses,
			MaxProducts:   tenant.Config.MaxProducts,
			CostStrategy:  tenant.Config.CostStrategy,
			Currency:      tenant.Config.Currency,
			Timezone:      tenant.Config.Timezone,
			Locale:        tenant.Config.Locale,
		},
		Notes:                 tenant.Notes,
		SuspensionReason:      tenant.SuspensionReason,
		SuspendedAt:           tenant.SuspendedAt,
		ScheduledReactivateAt: tenant.ScheduledReactivateAt,
		Statistics:            stats,
		CreatedAt:             tenant.CreatedAt,
		UpdatedAt:             tenant.UpdatedAt,
	}

	// Convert scheduled plan pointer to string pointer
	if tenant.ScheduledPlan != nil {
		planStr := string(*tenant.ScheduledPlan)
		dto.ScheduledPlan = &planStr
	}

	return dto
}

// ============================================================================
// Subscription Management
// ============================================================================

// ChangePlanInput contains input for changing a tenant's plan
type ChangePlanInput struct {
	TenantID uuid.UUID `json:"tenant_id"`
	NewPlan  string    `json:"new_plan"`
	Reason   string    `json:"reason,omitempty"`
}

// ChangePlanResult contains the result of a plan change
type ChangePlanResult struct {
	Tenant       *AdminTenantDTO `json:"tenant"`
	ChangeType   string          `json:"change_type"` // "immediate" or "scheduled"
	EffectiveAt  time.Time       `json:"effective_at"`
	PreviousPlan string          `json:"previous_plan"`
	NewPlan      string          `json:"new_plan"`
}

// UpdateQuotaInput contains input for updating tenant quotas
type UpdateQuotaInput struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	MaxUsers      *int      `json:"max_users,omitempty"`
	MaxWarehouses *int      `json:"max_warehouses,omitempty"`
	MaxProducts   *int      `json:"max_products,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

// SubscriptionHistoryDTO represents subscription history for API responses
type SubscriptionHistoryDTO struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	ChangeType      string     `json:"change_type"`
	OldPlan         string     `json:"old_plan,omitempty"`
	NewPlan         string     `json:"new_plan,omitempty"`
	OldQuota        *QuotaDTO  `json:"old_quota,omitempty"`
	NewQuota        *QuotaDTO  `json:"new_quota,omitempty"`
	EffectiveAt     time.Time  `json:"effective_at"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
	ChangedByUserID uuid.UUID  `json:"changed_by_user_id"`
	Reason          string     `json:"reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// QuotaDTO represents quota limits
type QuotaDTO struct {
	MaxUsers      int `json:"max_users"`
	MaxWarehouses int `json:"max_warehouses"`
	MaxProducts   int `json:"max_products"`
}

// SubscriptionHistoryListResult represents paginated history results
type SubscriptionHistoryListResult struct {
	History    []SubscriptionHistoryDTO `json:"history"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
}

// SubscriptionHistoryFilterInput contains filter options for subscription history queries
type SubscriptionHistoryFilterInput struct {
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	ChangeType *string    `json:"change_type,omitempty"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
}

// ChangePlan changes a tenant's subscription plan
// Upgrades take effect immediately, downgrades are scheduled for the end of the billing cycle
func (s *AdminTenantService) ChangePlan(ctx context.Context, input ChangePlanInput, auditCtx AuditContext) (*ChangePlanResult, error) {
	s.logger.Info("Admin changing tenant plan",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("new_plan", input.NewPlan),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Validate input
	if err := s.validatePlan(input.NewPlan); err != nil {
		return nil, err
	}

	// Get tenant
	tenant, err := s.adminRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Prevent changes to system tenant
	if tenant.Code == identity.SystemTenantCode {
		return nil, shared.NewDomainError("CANNOT_MODIFY_SYSTEM_TENANT", "Cannot modify system tenant subscription")
	}

	newPlan := identity.TenantPlan(input.NewPlan)
	oldPlan := tenant.Plan

	// Check if plan is the same
	if newPlan.IsSamePlan(oldPlan) {
		return nil, shared.NewDomainError("SAME_PLAN", "New plan is the same as current plan")
	}

	var result *ChangePlanResult
	var historyBuilder *identity.SubscriptionHistoryBuilder

	if newPlan.IsUpgradeFrom(oldPlan) {
		// Upgrade: apply immediately
		if err := tenant.SetPlan(newPlan); err != nil {
			return nil, err
		}

		// Clear any scheduled downgrade
		if tenant.HasScheduledPlanChange() {
			_ = tenant.CancelScheduledPlanChange()
		}

		// Save tenant
		if err := s.tenantRepo.Save(ctx, tenant); err != nil {
			s.logger.Error("Failed to save tenant after plan upgrade", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to save tenant")
		}

		// Create history record
		historyBuilder = identity.NewSubscriptionHistory(tenant.ID, auditCtx.AdminUserID, identity.SubscriptionChangeTypePlanUpgrade).
			WithPlanChange(oldPlan, newPlan).
			WithEffectiveAt(time.Now())

		result = &ChangePlanResult{
			Tenant:       toAdminTenantDTO(tenant, nil),
			ChangeType:   "immediate",
			EffectiveAt:  time.Now(),
			PreviousPlan: string(oldPlan),
			NewPlan:      string(newPlan),
		}

		// Publish domain event
		tenant.AddDomainEvent(identity.NewSubscriptionUpgradedEvent(tenant, oldPlan, newPlan))

	} else {
		// Downgrade: schedule for end of billing cycle
		// Default to 30 days from now if no expiration is set
		effectiveAt := time.Now().AddDate(0, 0, 30)
		if tenant.ExpiresAt != nil && tenant.ExpiresAt.After(time.Now()) {
			effectiveAt = *tenant.ExpiresAt
		}

		if err := tenant.SchedulePlanDowngrade(newPlan, effectiveAt); err != nil {
			return nil, err
		}

		// Save tenant
		if err := s.tenantRepo.Save(ctx, tenant); err != nil {
			s.logger.Error("Failed to save tenant after scheduling downgrade", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to save tenant")
		}

		// Create history record
		scheduledAt := time.Now()
		historyBuilder = identity.NewSubscriptionHistory(tenant.ID, auditCtx.AdminUserID, identity.SubscriptionChangeTypePlanDowngrade).
			WithPlanChange(oldPlan, newPlan).
			WithEffectiveAt(effectiveAt).
			WithScheduledAt(scheduledAt)

		result = &ChangePlanResult{
			Tenant:       toAdminTenantDTO(tenant, nil),
			ChangeType:   "scheduled",
			EffectiveAt:  effectiveAt,
			PreviousPlan: string(oldPlan),
			NewPlan:      string(newPlan),
		}

		// Publish domain event
		tenant.AddDomainEvent(identity.NewSubscriptionDowngradeScheduledEvent(tenant, oldPlan, newPlan, effectiveAt))
	}

	// Add reason if provided
	if input.Reason != "" {
		historyBuilder.WithReason(input.Reason)
	}

	// Save subscription history
	history, err := historyBuilder.Build()
	if err != nil {
		s.logger.Error("Failed to build subscription history", zap.Error(err))
	} else if s.subscriptionHistory != nil {
		if err := s.subscriptionHistory.Create(ctx, history); err != nil {
			s.logger.Warn("Failed to save subscription history", zap.Error(err))
		}
	}

	// Record audit log
	if err := s.auditService.RecordSubscriptionChange(ctx, auditCtx.AdminUserID, tenant.ID,
		map[string]interface{}{"plan": string(oldPlan)},
		map[string]interface{}{"plan": string(newPlan), "change_type": result.ChangeType},
		auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for plan change", zap.Error(err))
	}

	s.logger.Info("Tenant plan change processed",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("old_plan", string(oldPlan)),
		zap.String("new_plan", string(newPlan)),
		zap.String("change_type", result.ChangeType),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return result, nil
}

// UpdateQuota updates a tenant's quota limits
// Changes take effect immediately. Existing resources over quota are not deleted but new additions are blocked.
func (s *AdminTenantService) UpdateQuota(ctx context.Context, input UpdateQuotaInput, auditCtx AuditContext) (*AdminTenantDTO, error) {
	s.logger.Info("Admin updating tenant quota",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Get tenant
	tenant, err := s.adminRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Prevent changes to system tenant
	if tenant.Code == identity.SystemTenantCode {
		return nil, shared.NewDomainError("CANNOT_MODIFY_SYSTEM_TENANT", "Cannot modify system tenant quota")
	}

	// Store old quota for history
	oldQuota := tenant.GetCurrentQuota()

	// Apply quota updates
	maxUsers := tenant.Config.MaxUsers
	maxWarehouses := tenant.Config.MaxWarehouses
	maxProducts := tenant.Config.MaxProducts

	if input.MaxUsers != nil {
		if *input.MaxUsers < 1 {
			return nil, shared.NewDomainError("INVALID_MAX_USERS", "Max users must be at least 1")
		}
		maxUsers = *input.MaxUsers
	}
	if input.MaxWarehouses != nil {
		if *input.MaxWarehouses < 1 {
			return nil, shared.NewDomainError("INVALID_MAX_WAREHOUSES", "Max warehouses must be at least 1")
		}
		maxWarehouses = *input.MaxWarehouses
	}
	if input.MaxProducts != nil {
		if *input.MaxProducts < 1 {
			return nil, shared.NewDomainError("INVALID_MAX_PRODUCTS", "Max products must be at least 1")
		}
		maxProducts = *input.MaxProducts
	}

	// Apply custom quota
	if err := tenant.SetCustomQuota(maxUsers, maxWarehouses, maxProducts); err != nil {
		return nil, err
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to save tenant after quota update", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to save tenant")
	}

	// Get new quota
	newQuota := tenant.GetCurrentQuota()

	// Create subscription history record
	historyBuilder := identity.NewSubscriptionHistory(tenant.ID, auditCtx.AdminUserID, identity.SubscriptionChangeTypeQuotaUpdate).
		WithQuotaChange(oldQuota, newQuota).
		WithEffectiveAt(time.Now())

	if input.Reason != "" {
		historyBuilder.WithReason(input.Reason)
	}

	history, err := historyBuilder.Build()
	if err != nil {
		s.logger.Error("Failed to build subscription history", zap.Error(err))
	} else if s.subscriptionHistory != nil {
		if err := s.subscriptionHistory.Create(ctx, history); err != nil {
			s.logger.Warn("Failed to save subscription history", zap.Error(err))
		}
	}

	// Publish domain event
	tenant.AddDomainEvent(identity.NewQuotaUpdatedEvent(tenant, oldQuota, newQuota))

	// Record audit log
	if err := s.auditService.RecordQuotaUpdate(ctx, auditCtx.AdminUserID, tenant.ID,
		map[string]interface{}{"max_users": oldQuota.MaxUsers, "max_warehouses": oldQuota.MaxWarehouses, "max_products": oldQuota.MaxProducts},
		map[string]interface{}{"max_users": newQuota.MaxUsers, "max_warehouses": newQuota.MaxWarehouses, "max_products": newQuota.MaxProducts},
		auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for quota update", zap.Error(err))
	}

	s.logger.Info("Tenant quota updated",
		zap.String("tenant_id", input.TenantID.String()),
		zap.Int("max_users", newQuota.MaxUsers),
		zap.Int("max_warehouses", newQuota.MaxWarehouses),
		zap.Int("max_products", newQuota.MaxProducts),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return toAdminTenantDTO(tenant, nil), nil
}

// GetSubscriptionHistory retrieves subscription change history for a tenant
func (s *AdminTenantService) GetSubscriptionHistory(ctx context.Context, tenantID uuid.UUID, input SubscriptionHistoryFilterInput) (*SubscriptionHistoryListResult, error) {
	s.logger.Debug("Admin getting subscription history",
		zap.String("tenant_id", tenantID.String()))

	// Verify tenant exists
	_, err := s.adminRepo.FindByID(ctx, tenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Build filter
	filter := identity.DefaultSubscriptionHistoryFilter()
	if input.Page > 0 {
		filter.Page = input.Page
	}
	if input.PageSize > 0 {
		filter.PageSize = input.PageSize
		if filter.PageSize > 100 {
			filter.PageSize = 100
		}
	}
	if input.ChangeType != nil {
		changeType := identity.SubscriptionChangeType(*input.ChangeType)
		filter.ChangeType = &changeType
	}
	if input.StartDate != nil {
		filter.StartDate = input.StartDate
	}
	if input.EndDate != nil {
		filter.EndDate = input.EndDate
	}

	// Query history
	if s.subscriptionHistory == nil {
		return &SubscriptionHistoryListResult{
			History:    []SubscriptionHistoryDTO{},
			Total:      0,
			Page:       filter.Page,
			PageSize:   filter.Limit(),
			TotalPages: 0,
		}, nil
	}

	histories, total, err := s.subscriptionHistory.FindByTenantID(ctx, tenantID, filter)
	if err != nil {
		s.logger.Error("Failed to get subscription history", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get subscription history")
	}

	// Convert to DTOs
	historyDTOs := make([]SubscriptionHistoryDTO, len(histories))
	for i, h := range histories {
		historyDTOs[i] = toSubscriptionHistoryDTO(&h)
	}

	// Calculate total pages
	totalPages := int(total) / filter.Limit()
	if int(total)%filter.Limit() > 0 {
		totalPages++
	}

	return &SubscriptionHistoryListResult{
		History:    historyDTOs,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.Limit(),
		TotalPages: totalPages,
	}, nil
}

// CancelScheduledPlanChange cancels a pending plan downgrade
func (s *AdminTenantService) CancelScheduledPlanChange(ctx context.Context, tenantID uuid.UUID, auditCtx AuditContext) (*AdminTenantDTO, error) {
	s.logger.Info("Admin canceling scheduled plan change",
		zap.String("tenant_id", tenantID.String()),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	// Get tenant
	tenant, err := s.adminRepo.FindByID(ctx, tenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Verify there's a scheduled change
	if !tenant.HasScheduledPlanChange() {
		return nil, shared.NewDomainError("NO_SCHEDULED_CHANGE", "No scheduled plan change to cancel")
	}

	// Store scheduled plan info for audit
	scheduledPlan := *tenant.ScheduledPlan

	// Cancel the scheduled change
	if err := tenant.CancelScheduledPlanChange(); err != nil {
		return nil, err
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		s.logger.Error("Failed to save tenant after canceling scheduled change", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to save tenant")
	}

	// Record audit log
	if err := s.auditService.RecordSubscriptionChange(ctx, auditCtx.AdminUserID, tenant.ID,
		map[string]interface{}{"scheduled_plan": string(scheduledPlan)},
		map[string]interface{}{"scheduled_plan": nil, "action": "canceled"},
		auditCtx.IPAddress, auditCtx.UserAgent); err != nil {
		s.logger.Warn("Failed to record audit log for canceling scheduled change", zap.Error(err))
	}

	s.logger.Info("Scheduled plan change canceled",
		zap.String("tenant_id", tenantID.String()),
		zap.String("canceled_plan", string(scheduledPlan)),
		zap.String("admin_user_id", auditCtx.AdminUserID.String()))

	return toAdminTenantDTO(tenant, nil), nil
}

// toSubscriptionHistoryDTO converts domain SubscriptionHistory to DTO
func toSubscriptionHistoryDTO(h *identity.SubscriptionHistory) SubscriptionHistoryDTO {
	dto := SubscriptionHistoryDTO{
		ID:              h.ID,
		TenantID:        h.TenantID,
		ChangeType:      string(h.ChangeType),
		OldPlan:         string(h.OldPlan),
		NewPlan:         string(h.NewPlan),
		EffectiveAt:     h.EffectiveAt,
		ScheduledAt:     h.ScheduledAt,
		ChangedByUserID: h.ChangedByUserID,
		Reason:          h.Reason,
		CreatedAt:       h.CreatedAt,
	}

	if h.OldQuota != nil {
		dto.OldQuota = &QuotaDTO{
			MaxUsers:      h.OldQuota.MaxUsers,
			MaxWarehouses: h.OldQuota.MaxWarehouses,
			MaxProducts:   h.OldQuota.MaxProducts,
		}
	}

	if h.NewQuota != nil {
		dto.NewQuota = &QuotaDTO{
			MaxUsers:      h.NewQuota.MaxUsers,
			MaxWarehouses: h.NewQuota.MaxWarehouses,
			MaxProducts:   h.NewQuota.MaxProducts,
		}
	}

	return dto
}
