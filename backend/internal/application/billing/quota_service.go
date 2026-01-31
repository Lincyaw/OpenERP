package billing

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QuotaExceededError represents an error when a quota limit is exceeded
type QuotaExceededError struct {
	UsageType    billing.UsageType
	CurrentUsage int64
	Limit        int64
	Message      string
}

// Error implements the error interface
func (e *QuotaExceededError) Error() string {
	return e.Message
}

// HTTPStatusCode returns the HTTP status code for this error (429 Too Many Requests)
func (e *QuotaExceededError) HTTPStatusCode() int {
	return http.StatusTooManyRequests
}

// NewQuotaExceededError creates a new QuotaExceededError
func NewQuotaExceededError(usageType billing.UsageType, currentUsage, limit int64) *QuotaExceededError {
	return &QuotaExceededError{
		UsageType:    usageType,
		CurrentUsage: currentUsage,
		Limit:        limit,
		Message: fmt.Sprintf(
			"Quota exceeded for %s: current usage %d exceeds limit %d",
			usageType.DisplayName(), currentUsage, limit,
		),
	}
}

// QuotaWarning represents a warning when approaching quota limit
type QuotaWarning struct {
	UsageType    billing.UsageType
	CurrentUsage int64
	Limit        int64
	SoftLimit    int64
	Percentage   float64
	Message      string
}

// QuotaCheckInput contains input for checking quota
type QuotaCheckInput struct {
	TenantID  uuid.UUID
	UsageType billing.UsageType
	Amount    int64 // Amount to be consumed (default 1)
}

// QuotaCheckResult contains the result of a quota check
type QuotaCheckResult struct {
	Allowed      bool                  // Whether the operation is allowed
	UsageType    billing.UsageType     // Type of usage checked
	CurrentUsage int64                 // Current usage amount
	Limit        int64                 // Hard limit (-1 for unlimited)
	SoftLimit    *int64                // Soft limit for warnings (nil if not set)
	Remaining    int64                 // Remaining quota
	Percentage   float64               // Usage percentage (0-100+)
	Status       billing.QuotaStatus   // OK, WARNING, EXCEEDED, INACTIVE
	Policy       billing.OveragePolicy // What happens when exceeded
	Warning      *QuotaWarning         // Warning if approaching limit
	Error        *QuotaExceededError   // Error if exceeded and blocked
}

// UsageSummaryDTO contains usage summary for a tenant
type UsageSummaryDTO struct {
	TenantID    uuid.UUID                 `json:"tenant_id"`
	PeriodStart time.Time                 `json:"period_start"`
	PeriodEnd   time.Time                 `json:"period_end"`
	Usages      map[string]UsageDetailDTO `json:"usages"`
	Warnings    []QuotaWarning            `json:"warnings,omitempty"`
	Exceeded    []string                  `json:"exceeded,omitempty"`
}

// UsageDetailDTO contains detailed usage information for a single type
type UsageDetailDTO struct {
	UsageType    string  `json:"usage_type"`
	DisplayName  string  `json:"display_name"`
	CurrentUsage int64   `json:"current_usage"`
	Limit        int64   `json:"limit"`
	SoftLimit    *int64  `json:"soft_limit,omitempty"`
	Remaining    int64   `json:"remaining"`
	Percentage   float64 `json:"percentage"`
	Status       string  `json:"status"`
	Unit         string  `json:"unit"`
	Formatted    string  `json:"formatted"`
}

// QuotaService handles quota checking and enforcement
type QuotaService struct {
	quotaRepo      billing.UsageQuotaRepository
	usageRepo      billing.UsageRecordRepository
	meterRepo      billing.UsageMeterRepository
	tenantRepo     identity.TenantRepository
	eventPublisher billing.UsageEventPublisher
	logger         *zap.Logger

	// Configuration
	defaultSoftLimitPercent float64 // Default soft limit as percentage of hard limit (e.g., 80%)
	cacheTTL                time.Duration
}

// QuotaServiceConfig contains configuration for QuotaService
type QuotaServiceConfig struct {
	DefaultSoftLimitPercent float64
	CacheTTL                time.Duration
}

// DefaultQuotaServiceConfig returns default configuration
func DefaultQuotaServiceConfig() QuotaServiceConfig {
	return QuotaServiceConfig{
		DefaultSoftLimitPercent: 80.0,
		CacheTTL:                5 * time.Minute,
	}
}

// NewQuotaService creates a new QuotaService
func NewQuotaService(
	quotaRepo billing.UsageQuotaRepository,
	usageRepo billing.UsageRecordRepository,
	meterRepo billing.UsageMeterRepository,
	tenantRepo identity.TenantRepository,
	eventPublisher billing.UsageEventPublisher,
	logger *zap.Logger,
	config QuotaServiceConfig,
) *QuotaService {
	return &QuotaService{
		quotaRepo:               quotaRepo,
		usageRepo:               usageRepo,
		meterRepo:               meterRepo,
		tenantRepo:              tenantRepo,
		eventPublisher:          eventPublisher,
		logger:                  logger,
		defaultSoftLimitPercent: config.DefaultSoftLimitPercent,
		cacheTTL:                config.CacheTTL,
	}
}

// CheckQuota checks if a tenant can consume the specified amount of a resource
// Returns QuotaCheckResult with allowed=true if within quota, or appropriate warnings/errors
func (s *QuotaService) CheckQuota(ctx context.Context, input QuotaCheckInput) (*QuotaCheckResult, error) {
	if input.TenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}
	if !input.UsageType.IsValid() {
		return nil, shared.NewDomainError("INVALID_USAGE_TYPE", "Invalid usage type")
	}
	if input.Amount <= 0 {
		input.Amount = 1 // Default to checking for 1 unit
	}

	s.logger.Debug("Checking quota",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("usage_type", string(input.UsageType)),
		zap.Int64("amount", input.Amount))

	// Get tenant to determine plan
	tenant, err := s.tenantRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Get effective quota for this tenant and usage type
	quota, err := s.quotaRepo.FindEffectiveQuota(ctx, input.TenantID, string(tenant.Plan), input.UsageType)
	if err != nil {
		// If no quota defined, allow by default (unlimited)
		if err == shared.ErrNotFound {
			s.logger.Debug("No quota defined, allowing unlimited usage",
				zap.String("tenant_id", input.TenantID.String()),
				zap.String("usage_type", string(input.UsageType)))
			return &QuotaCheckResult{
				Allowed:      true,
				UsageType:    input.UsageType,
				CurrentUsage: 0,
				Limit:        -1, // Unlimited
				Remaining:    -1,
				Percentage:   0,
				Status:       billing.QuotaStatusOK,
				Policy:       billing.OveragePolicyWarn,
			}, nil
		}
		s.logger.Error("Failed to find quota", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find quota")
	}

	// Get current usage for the billing period
	currentUsage, err := s.getCurrentUsage(ctx, input.TenantID, input.UsageType, quota.ResetPeriod)
	if err != nil {
		s.logger.Error("Failed to get current usage", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get current usage")
	}

	// Check quota with the amount to be consumed
	projectedUsage := currentUsage + input.Amount
	checkResult := quota.CheckUsage(projectedUsage)

	result := &QuotaCheckResult{
		Allowed:      checkResult.IsAllowed(),
		UsageType:    input.UsageType,
		CurrentUsage: currentUsage,
		Limit:        quota.Limit,
		SoftLimit:    quota.SoftLimit,
		Remaining:    checkResult.Remaining,
		Percentage:   checkResult.UsagePercent,
		Status:       checkResult.Status,
		Policy:       quota.OveragePolicy,
	}

	// Handle warnings (soft limit reached)
	if checkResult.ShouldWarn() {
		warning := &QuotaWarning{
			UsageType:    input.UsageType,
			CurrentUsage: currentUsage,
			Limit:        quota.Limit,
			Percentage:   checkResult.UsagePercent,
			Message: fmt.Sprintf(
				"%s usage is at %.1f%% of quota (%d/%d)",
				input.UsageType.DisplayName(),
				checkResult.UsagePercent,
				currentUsage,
				quota.Limit,
			),
		}
		if quota.SoftLimit != nil {
			warning.SoftLimit = *quota.SoftLimit
		}
		result.Warning = warning

		// Publish warning event asynchronously
		go func() {
			if s.eventPublisher != nil {
				if pubErr := s.eventPublisher.PublishQuotaWarning(context.Background(), input.TenantID, checkResult); pubErr != nil {
					s.logger.Warn("Failed to publish quota warning event", zap.Error(pubErr))
				}
			}
		}()
	}

	// Handle exceeded (hard limit reached with BLOCK policy)
	if checkResult.Status == billing.QuotaStatusExceeded && quota.OveragePolicy == billing.OveragePolicyBlock {
		result.Allowed = false
		result.Error = NewQuotaExceededError(input.UsageType, projectedUsage, quota.Limit)

		// Publish exceeded event asynchronously
		go func() {
			if s.eventPublisher != nil {
				if pubErr := s.eventPublisher.PublishQuotaExceeded(context.Background(), input.TenantID, checkResult); pubErr != nil {
					s.logger.Warn("Failed to publish quota exceeded event", zap.Error(pubErr))
				}
			}
		}()

		s.logger.Info("Quota exceeded, blocking operation",
			zap.String("tenant_id", input.TenantID.String()),
			zap.String("usage_type", string(input.UsageType)),
			zap.Int64("current_usage", currentUsage),
			zap.Int64("limit", quota.Limit))
	}

	return result, nil
}

// GetUsageSummary retrieves a summary of all usage for a tenant in the current billing period
func (s *QuotaService) GetUsageSummary(ctx context.Context, tenantID uuid.UUID, period billing.ResetPeriod) (*UsageSummaryDTO, error) {
	if tenantID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_TENANT", "Tenant ID cannot be empty")
	}

	s.logger.Debug("Getting usage summary",
		zap.String("tenant_id", tenantID.String()),
		zap.String("period", string(period)))

	// Get tenant to determine plan
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("TENANT_NOT_FOUND", "Tenant not found")
		}
		s.logger.Error("Failed to find tenant", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find tenant")
	}

	// Calculate period boundaries
	periodStart, periodEnd := s.calculatePeriodBoundaries(period)

	// Get all effective quotas for this tenant
	quotas, err := s.quotaRepo.FindAllEffectiveQuotas(ctx, tenantID, string(tenant.Plan))
	if err != nil {
		s.logger.Error("Failed to find quotas", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find quotas")
	}

	// Build quota map for quick lookup
	quotaMap := make(map[billing.UsageType]*billing.UsageQuota)
	for _, q := range quotas {
		quotaMap[q.UsageType] = q
	}

	// Get usage for all types
	summary := &UsageSummaryDTO{
		TenantID:    tenantID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Usages:      make(map[string]UsageDetailDTO),
		Warnings:    make([]QuotaWarning, 0),
		Exceeded:    make([]string, 0),
	}

	// Check each usage type that has a quota defined
	for usageType, quota := range quotaMap {
		currentUsage, err := s.getCurrentUsageForPeriod(ctx, tenantID, usageType, periodStart, periodEnd)
		if err != nil {
			s.logger.Warn("Failed to get usage for type",
				zap.String("usage_type", string(usageType)),
				zap.Error(err))
			continue
		}

		checkResult := quota.CheckUsage(currentUsage)

		detail := UsageDetailDTO{
			UsageType:    string(usageType),
			DisplayName:  usageType.DisplayName(),
			CurrentUsage: currentUsage,
			Limit:        quota.Limit,
			SoftLimit:    quota.SoftLimit,
			Remaining:    checkResult.Remaining,
			Percentage:   checkResult.UsagePercent,
			Status:       string(checkResult.Status),
			Unit:         string(quota.Unit),
			Formatted:    quota.Unit.FormatValue(currentUsage),
		}

		if quota.IsUnlimited() {
			detail.Remaining = -1
			detail.Percentage = 0
		}

		summary.Usages[string(usageType)] = detail

		// Collect warnings
		if checkResult.Status == billing.QuotaStatusWarning {
			warning := QuotaWarning{
				UsageType:    usageType,
				CurrentUsage: currentUsage,
				Limit:        quota.Limit,
				Percentage:   checkResult.UsagePercent,
				Message: fmt.Sprintf(
					"%s is approaching quota limit (%.1f%%)",
					usageType.DisplayName(),
					checkResult.UsagePercent,
				),
			}
			if quota.SoftLimit != nil {
				warning.SoftLimit = *quota.SoftLimit
			}
			summary.Warnings = append(summary.Warnings, warning)
		}

		// Collect exceeded
		if checkResult.Status == billing.QuotaStatusExceeded {
			summary.Exceeded = append(summary.Exceeded, string(usageType))
		}
	}

	return summary, nil
}

// CheckQuotaForResourceCreation is a convenience method for checking quota before creating a resource
// Returns nil if allowed, QuotaExceededError if blocked, or other errors
func (s *QuotaService) CheckQuotaForResourceCreation(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType) error {
	result, err := s.CheckQuota(ctx, QuotaCheckInput{
		TenantID:  tenantID,
		UsageType: usageType,
		Amount:    1,
	})
	if err != nil {
		return err
	}

	if !result.Allowed {
		return result.Error
	}

	return nil
}

// CheckProductQuota checks if a tenant can create a new product
func (s *QuotaService) CheckProductQuota(ctx context.Context, tenantID uuid.UUID) error {
	return s.CheckQuotaForResourceCreation(ctx, tenantID, billing.UsageTypeProductsSKU)
}

// CheckUserQuota checks if a tenant can invite a new user
func (s *QuotaService) CheckUserQuota(ctx context.Context, tenantID uuid.UUID) error {
	return s.CheckQuotaForResourceCreation(ctx, tenantID, billing.UsageTypeActiveUsers)
}

// CheckOrderQuota checks if a tenant can create a new order
func (s *QuotaService) CheckOrderQuota(ctx context.Context, tenantID uuid.UUID) error {
	return s.CheckQuotaForResourceCreation(ctx, tenantID, billing.UsageTypeOrdersCreated)
}

// CheckWarehouseQuota checks if a tenant can create a new warehouse
func (s *QuotaService) CheckWarehouseQuota(ctx context.Context, tenantID uuid.UUID) error {
	return s.CheckQuotaForResourceCreation(ctx, tenantID, billing.UsageTypeWarehouses)
}

// CheckCustomerQuota checks if a tenant can create a new customer
func (s *QuotaService) CheckCustomerQuota(ctx context.Context, tenantID uuid.UUID) error {
	return s.CheckQuotaForResourceCreation(ctx, tenantID, billing.UsageTypeCustomers)
}

// CheckSupplierQuota checks if a tenant can create a new supplier
func (s *QuotaService) CheckSupplierQuota(ctx context.Context, tenantID uuid.UUID) error {
	return s.CheckQuotaForResourceCreation(ctx, tenantID, billing.UsageTypeSuppliers)
}

// GetQuotaStatus retrieves the quota status for all usage types for a tenant
func (s *QuotaService) GetQuotaStatus(ctx context.Context, tenantID uuid.UUID) (map[billing.UsageType]QuotaCheckResult, error) {
	summary, err := s.GetUsageSummary(ctx, tenantID, billing.ResetPeriodMonthly)
	if err != nil {
		return nil, err
	}

	results := make(map[billing.UsageType]QuotaCheckResult)
	for usageTypeStr, detail := range summary.Usages {
		usageType := billing.UsageType(usageTypeStr)
		results[usageType] = QuotaCheckResult{
			Allowed:      detail.Status != string(billing.QuotaStatusExceeded),
			UsageType:    usageType,
			CurrentUsage: detail.CurrentUsage,
			Limit:        detail.Limit,
			SoftLimit:    detail.SoftLimit,
			Remaining:    detail.Remaining,
			Percentage:   detail.Percentage,
			Status:       billing.QuotaStatus(detail.Status),
		}
	}

	return results, nil
}

// getCurrentUsage gets the current usage for a tenant and usage type based on reset period
func (s *QuotaService) getCurrentUsage(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, resetPeriod billing.ResetPeriod) (int64, error) {
	periodStart, periodEnd := s.calculatePeriodBoundaries(resetPeriod)
	return s.getCurrentUsageForPeriod(ctx, tenantID, usageType, periodStart, periodEnd)
}

// getCurrentUsageForPeriod gets the current usage for a specific time period
func (s *QuotaService) getCurrentUsageForPeriod(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType, periodStart, periodEnd time.Time) (int64, error) {
	// For countable resources (users, products, warehouses), we need current count
	// For accumulative resources (API calls, orders), we need sum over period
	if usageType.IsCountable() {
		return s.getCountableResourceUsage(ctx, tenantID, usageType)
	}

	// Try to get from cache first
	if s.meterRepo != nil {
		meter, err := s.meterRepo.GetMeter(ctx, tenantID, usageType, periodStart, periodEnd)
		if err == nil && meter != nil {
			return meter.TotalUsage, nil
		}
	}

	// Calculate from usage records
	return s.usageRepo.SumByTenantAndType(ctx, tenantID, usageType, periodStart, periodEnd)
}

// getCountableResourceUsage gets the current count of countable resources
// This queries the actual resource tables rather than usage records
func (s *QuotaService) getCountableResourceUsage(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType) (int64, error) {
	// For countable resources, we need to query the actual resource count
	// This is typically done through the usage meter repository which can calculate from source
	if s.meterRepo != nil {
		now := time.Now()
		periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

		meter, err := s.meterRepo.CalculateMeter(ctx, tenantID, usageType, periodStart, periodEnd)
		if err == nil && meter != nil {
			return meter.TotalUsage, nil
		}
	}

	// Fallback: return 0 if we can't determine the count
	// In production, this should be implemented to query actual resource counts
	s.logger.Warn("Could not determine countable resource usage, returning 0",
		zap.String("tenant_id", tenantID.String()),
		zap.String("usage_type", string(usageType)))
	return 0, nil
}

// calculatePeriodBoundaries calculates the start and end times for a billing period
func (s *QuotaService) calculatePeriodBoundaries(resetPeriod billing.ResetPeriod) (time.Time, time.Time) {
	now := time.Now()
	var periodStart, periodEnd time.Time

	switch resetPeriod {
	case billing.ResetPeriodDaily:
		periodStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 1).Add(-time.Nanosecond)

	case billing.ResetPeriodWeekly:
		// Start from Monday
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		daysFromMonday := weekday - 1
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-daysFromMonday, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 7).Add(-time.Nanosecond)

	case billing.ResetPeriodMonthly:
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	case billing.ResetPeriodYearly:
		periodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(1, 0, 0).Add(-time.Nanosecond)

	case billing.ResetPeriodNever:
		// For lifetime limits, use a very old start date
		periodStart = time.Date(2000, 1, 1, 0, 0, 0, 0, now.Location())
		periodEnd = time.Date(2100, 12, 31, 23, 59, 59, 999999999, now.Location())

	default:
		// Default to monthly
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
	}

	return periodStart, periodEnd
}
