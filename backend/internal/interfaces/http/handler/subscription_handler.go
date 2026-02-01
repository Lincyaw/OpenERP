package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionHandler handles subscription-related HTTP requests
type SubscriptionHandler struct {
	BaseHandler
	tenantRepo      identity.TenantRepository
	planFeatureRepo identity.PlanFeatureRepository
	userRepo        identity.UserRepository
	warehouseRepo   WarehouseCounter
	productRepo     ProductCounter
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(
	tenantRepo identity.TenantRepository,
	planFeatureRepo identity.PlanFeatureRepository,
	userRepo identity.UserRepository,
	warehouseRepo WarehouseCounter,
	productRepo ProductCounter,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		tenantRepo:      tenantRepo,
		planFeatureRepo: planFeatureRepo,
		userRepo:        userRepo,
		warehouseRepo:   warehouseRepo,
		productRepo:     productRepo,
	}
}

// ============================================================================
// Request/Response DTOs
// ============================================================================

// SubscriptionQuotaResponse represents a single quota with usage information
//
//	@Description	Quota item with current usage and remaining capacity
type SubscriptionQuotaResponse struct {
	Type        string `json:"type" example:"users"`
	Limit       int64  `json:"limit" example:"10"`
	Used        int64  `json:"used" example:"5"`
	Remaining   int64  `json:"remaining" example:"5"`
	Unit        string `json:"unit" example:"count"`
	ResetPeriod string `json:"reset_period" example:"never"`
}

// SubscriptionFeaturesResponse represents enabled features for the subscription
//
//	@Description	Map of feature keys to their enabled status
type SubscriptionFeaturesResponse map[string]bool

// CurrentSubscriptionResponse represents the current tenant subscription
//
//	@Description	Complete subscription information for the current tenant
type CurrentSubscriptionResponse struct {
	PlanID      string                       `json:"plan_id" example:"basic"`
	PlanName    string                       `json:"plan_name" example:"基础版"`
	Status      string                       `json:"status" example:"active"`
	PeriodStart *string                      `json:"period_start,omitempty" example:"2024-01-01T00:00:00Z"`
	PeriodEnd   *string                      `json:"period_end,omitempty" example:"2024-02-01T00:00:00Z"`
	TrialEndsAt *string                      `json:"trial_ends_at,omitempty" example:"2024-01-15T00:00:00Z"`
	Quotas      []SubscriptionQuotaResponse  `json:"quotas"`
	Features    SubscriptionFeaturesResponse `json:"features"`
}

// ============================================================================
// Handlers
// ============================================================================

// GetCurrentSubscription godoc
//
//	@ID				getCurrentSubscription
//	@Summary		Get current tenant subscription
//	@Description	Get complete subscription information for the current tenant including plan, quotas, and features
//	@Tags			billing
//	@Produce		json
//	@Success		200	{object}	APIResponse[CurrentSubscriptionResponse]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/billing/subscription/current [get]
func (h *SubscriptionHandler) GetCurrentSubscription(c *gin.Context) {
	// Get tenant ID from JWT context
	tenantIDStr := middleware.GetJWTTenantID(c)
	if tenantIDStr == "" {
		h.Unauthorized(c, "Tenant ID not found in token")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID format")
		return
	}

	// Get tenant information
	tenant, err := h.tenantRepo.FindByID(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			h.NotFound(c, "Tenant not found")
			return
		}
		slog.Error("failed to retrieve tenant", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve tenant information")
		return
	}

	// Build subscription response
	response, err := h.buildSubscriptionResponse(c.Request.Context(), tenant)
	if err != nil {
		slog.Error("failed to build subscription response", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve subscription information")
		return
	}

	h.Success(c, response)
}

// ============================================================================
// Helper functions
// ============================================================================

// buildSubscriptionResponse builds the complete subscription response
func (h *SubscriptionHandler) buildSubscriptionResponse(ctx context.Context, tenant *identity.Tenant) (*CurrentSubscriptionResponse, error) {
	// Get quotas
	quotas, err := h.getSubscriptionQuotas(ctx, tenant)
	if err != nil {
		return nil, err
	}

	// Get features
	features, err := h.getSubscriptionFeatures(ctx, tenant.Plan)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &CurrentSubscriptionResponse{
		PlanID:   string(tenant.Plan),
		PlanName: getPlanDisplayName(tenant.Plan),
		Status:   getSubscriptionStatus(tenant),
		Quotas:   quotas,
		Features: features,
	}

	// Set period dates if subscription has expiration
	if tenant.ExpiresAt != nil {
		// Calculate period start (assume monthly billing, period start is 1 month before expiry)
		periodStart := tenant.ExpiresAt.AddDate(0, -1, 0).Format(time.RFC3339)
		periodEnd := tenant.ExpiresAt.Format(time.RFC3339)
		response.PeriodStart = &periodStart
		response.PeriodEnd = &periodEnd
	}

	// Set trial end date if in trial
	if tenant.TrialEndsAt != nil {
		trialEnds := tenant.TrialEndsAt.Format(time.RFC3339)
		response.TrialEndsAt = &trialEnds
	}

	return response, nil
}

// getSubscriptionQuotas retrieves quota information for the tenant
func (h *SubscriptionHandler) getSubscriptionQuotas(ctx context.Context, tenant *identity.Tenant) ([]SubscriptionQuotaResponse, error) {
	// Count users
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Count warehouses
	warehouseCount, err := h.warehouseRepo.CountForTenant(ctx, tenant.ID, shared.Filter{})
	if err != nil {
		return nil, err
	}

	// Count products
	productCount, err := h.productRepo.CountForTenant(ctx, tenant.ID, shared.Filter{})
	if err != nil {
		return nil, err
	}

	// Check if enterprise plan (unlimited)
	isEnterprise := tenant.Plan == identity.TenantPlanEnterprise

	quotas := []SubscriptionQuotaResponse{
		{
			Type:        "users",
			Limit:       getQuotaLimit(int64(tenant.Config.MaxUsers), isEnterprise),
			Used:        userCount,
			Remaining:   calculateQuotaRemaining(userCount, int64(tenant.Config.MaxUsers), isEnterprise),
			Unit:        "count",
			ResetPeriod: "never",
		},
		{
			Type:        "warehouses",
			Limit:       getQuotaLimit(int64(tenant.Config.MaxWarehouses), isEnterprise),
			Used:        warehouseCount,
			Remaining:   calculateQuotaRemaining(warehouseCount, int64(tenant.Config.MaxWarehouses), isEnterprise),
			Unit:        "count",
			ResetPeriod: "never",
		},
		{
			Type:        "products",
			Limit:       getQuotaLimit(int64(tenant.Config.MaxProducts), isEnterprise),
			Used:        productCount,
			Remaining:   calculateQuotaRemaining(productCount, int64(tenant.Config.MaxProducts), isEnterprise),
			Unit:        "count",
			ResetPeriod: "never",
		},
	}

	return quotas, nil
}

// getSubscriptionFeatures retrieves enabled features for the plan
func (h *SubscriptionHandler) getSubscriptionFeatures(ctx context.Context, plan identity.TenantPlan) (SubscriptionFeaturesResponse, error) {
	features := make(SubscriptionFeaturesResponse)

	// Try to get features from repository first
	if h.planFeatureRepo != nil {
		planFeatures, err := h.planFeatureRepo.FindByPlan(ctx, plan)
		if err == nil && len(planFeatures) > 0 {
			for _, f := range planFeatures {
				features[string(f.FeatureKey)] = f.Enabled
			}
			return features, nil
		}
		// Log warning but continue with defaults
		if err != nil {
			slog.Warn("failed to get plan features from repository, using defaults", "plan", plan, "error", err)
		}
	}

	// Fall back to default plan features
	defaultFeatures := identity.DefaultPlanFeatures(plan)
	for _, f := range defaultFeatures {
		features[string(f.FeatureKey)] = f.Enabled
	}

	return features, nil
}

// getPlanDisplayName returns the display name for a plan
func getPlanDisplayName(plan identity.TenantPlan) string {
	switch plan {
	case identity.TenantPlanFree:
		return "免费版"
	case identity.TenantPlanBasic:
		return "基础版"
	case identity.TenantPlanPro:
		return "专业版"
	case identity.TenantPlanEnterprise:
		return "企业版"
	default:
		return string(plan)
	}
}

// getSubscriptionStatus returns the subscription status based on tenant state
func getSubscriptionStatus(tenant *identity.Tenant) string {
	switch tenant.Status {
	case identity.TenantStatusActive:
		return "active"
	case identity.TenantStatusInactive:
		return "cancelled"
	case identity.TenantStatusSuspended:
		return "expired"
	case identity.TenantStatusTrial:
		return "trial"
	default:
		return string(tenant.Status)
	}
}

// getQuotaLimit returns the quota limit, -1 for unlimited
func getQuotaLimit(limit int64, isUnlimited bool) int64 {
	if isUnlimited {
		return -1
	}
	return limit
}

// calculateQuotaRemaining calculates remaining quota
func calculateQuotaRemaining(used, limit int64, isUnlimited bool) int64 {
	if isUnlimited {
		return -1 // -1 indicates unlimited
	}
	remaining := limit - used
	if remaining < 0 {
		return 0
	}
	return remaining
}
