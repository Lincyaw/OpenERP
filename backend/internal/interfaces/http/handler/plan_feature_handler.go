package handler

import (
	"log/slog"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PlanFeatureHandler handles plan feature management HTTP requests
type PlanFeatureHandler struct {
	BaseHandler
	tenantRepo identity.TenantRepository
}

// NewPlanFeatureHandler creates a new plan feature handler
func NewPlanFeatureHandler(tenantRepo identity.TenantRepository) *PlanFeatureHandler {
	return &PlanFeatureHandler{
		tenantRepo: tenantRepo,
	}
}

// ============================================================================
// Request/Response DTOs
// ============================================================================

// PlanResponse represents a subscription plan
//
//	@Description	Subscription plan information
type PlanResponse struct {
	Code        string `json:"code" example:"basic"`
	Name        string `json:"name" example:"Basic Plan"`
	Description string `json:"description" example:"Basic plan with essential features"`
}

// PlanListResponse represents a list of plans
//
//	@Description	List of available subscription plans
type PlanListResponse struct {
	Plans []PlanResponse `json:"plans"`
}

// PlanFeatureResponse represents a feature for a plan
//
//	@Description	Feature configuration for a subscription plan
type PlanFeatureResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FeatureKey  string `json:"feature_key" example:"multi_warehouse"`
	Enabled     bool   `json:"enabled" example:"true"`
	Limit       *int   `json:"limit,omitempty" example:"1000"`
	Description string `json:"description" example:"Multiple warehouse management"`
}

// PlanFeaturesResponse represents features for a plan
//
//	@Description	List of features for a subscription plan
type PlanFeaturesResponse struct {
	Plan     string                `json:"plan" example:"basic"`
	Features []PlanFeatureResponse `json:"features"`
}

// UpdatePlanFeatureRequest represents a request to update a single feature
//
//	@Description	Request to update a feature configuration
type UpdatePlanFeatureRequest struct {
	FeatureKey string `json:"feature_key" binding:"required" example:"multi_warehouse"`
	Enabled    bool   `json:"enabled" example:"true"`
	Limit      *int   `json:"limit,omitempty" example:"1000"`
}

// UpdatePlanFeaturesRequest represents a request to update plan features
//
//	@Description	Request to update multiple feature configurations for a plan
type UpdatePlanFeaturesRequest struct {
	Features []UpdatePlanFeatureRequest `json:"features" binding:"required,dive"`
}

// TenantFeatureResponse represents a feature available to a tenant
//
//	@Description	Feature available to the current tenant
type TenantFeatureResponse struct {
	FeatureKey  string `json:"feature_key" example:"multi_warehouse"`
	Enabled     bool   `json:"enabled" example:"true"`
	Limit       *int   `json:"limit,omitempty" example:"1000"`
	Description string `json:"description" example:"Multiple warehouse management"`
}

// TenantFeaturesResponse represents all features available to a tenant
//
//	@Description	List of features available to the current tenant
type TenantFeaturesResponse struct {
	TenantID string                  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Plan     string                  `json:"plan" example:"basic"`
	Features []TenantFeatureResponse `json:"features"`
}

// ============================================================================
// Handlers
// ============================================================================

// ListPlans godoc
//
//	@ID				listPlans
//	@Summary		List all subscription plans
//	@Description	Get a list of all available subscription plans
//	@Tags			plan-features
//	@Produce		json
//	@Success		200	{object}	APIResponse[PlanListResponse]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/plans [get]
func (h *PlanFeatureHandler) ListPlans(c *gin.Context) {
	plans := []PlanResponse{
		{
			Code:        string(identity.TenantPlanFree),
			Name:        "Free Plan",
			Description: "Basic features for small businesses",
		},
		{
			Code:        string(identity.TenantPlanBasic),
			Name:        "Basic Plan",
			Description: "Essential features for growing businesses",
		},
		{
			Code:        string(identity.TenantPlanPro),
			Name:        "Pro Plan",
			Description: "Advanced features for professional use",
		},
		{
			Code:        string(identity.TenantPlanEnterprise),
			Name:        "Enterprise Plan",
			Description: "Full features with dedicated support",
		},
	}

	h.Success(c, PlanListResponse{Plans: plans})
}

// GetPlanFeatures godoc
//
//	@ID				getPlanFeatures
//	@Summary		Get features for a plan
//	@Description	Get all features and their configuration for a specific subscription plan
//	@Tags			plan-features
//	@Produce		json
//	@Param			plan	path		string	true	"Plan code"	Enums(free, basic, pro, enterprise)
//	@Success		200		{object}	APIResponse[PlanFeaturesResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/plans/{plan}/features [get]
func (h *PlanFeatureHandler) GetPlanFeatures(c *gin.Context) {
	planCode := c.Param("plan")
	if planCode == "" {
		h.BadRequest(c, "Plan code is required")
		return
	}

	// Validate plan code
	plan := identity.TenantPlan(planCode)
	if !isValidPlan(plan) {
		h.BadRequest(c, "Invalid plan code. Must be one of: free, basic, pro, enterprise")
		return
	}

	// Get default features for the plan
	features := identity.DefaultPlanFeatures(plan)

	// Convert to response
	featureResponses := make([]PlanFeatureResponse, len(features))
	for i, f := range features {
		featureResponses[i] = PlanFeatureResponse{
			ID:          f.ID.String(),
			FeatureKey:  string(f.FeatureKey),
			Enabled:     f.Enabled,
			Limit:       f.Limit,
			Description: f.Description,
		}
	}

	h.Success(c, PlanFeaturesResponse{
		Plan:     planCode,
		Features: featureResponses,
	})
}

// UpdatePlanFeatures godoc
//
//	@ID				updatePlanFeatures
//	@Summary		Preview feature updates for a plan (read-only)
//	@Description	Validate and preview feature configuration changes for a subscription plan.
//	@Description	NOTE: This endpoint currently returns a preview of the changes but does NOT persist them.
//	@Description	Full persistence will be implemented when PlanFeatureRepository is available.
//	@Tags			plan-features
//	@Accept			json
//	@Produce		json
//	@Param			plan	path		string						true	"Plan code"	Enums(free, basic, pro, enterprise)
//	@Param			request	body		UpdatePlanFeaturesRequest	true	"Features to update"
//	@Success		200		{object}	APIResponse[PlanFeaturesResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/plans/{plan}/features [put]
func (h *PlanFeatureHandler) UpdatePlanFeatures(c *gin.Context) {
	planCode := c.Param("plan")
	if planCode == "" {
		h.BadRequest(c, "Plan code is required")
		return
	}

	// Validate plan code
	plan := identity.TenantPlan(planCode)
	if !isValidPlan(plan) {
		h.BadRequest(c, "Invalid plan code. Must be one of: free, basic, pro, enterprise")
		return
	}

	var req UpdatePlanFeaturesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if len(req.Features) == 0 {
		h.BadRequest(c, "At least one feature must be provided")
		return
	}

	// Validate all feature keys
	for _, f := range req.Features {
		if !identity.IsValidFeatureKey(identity.FeatureKey(f.FeatureKey)) {
			h.BadRequest(c, "Invalid feature key: "+f.FeatureKey)
			return
		}
		if f.Limit != nil && *f.Limit < 0 {
			h.BadRequest(c, "Feature limit cannot be negative for: "+f.FeatureKey)
			return
		}
	}

	// Get current features and apply updates
	// TODO: Implement persistence via PlanFeatureRepository when available.
	// Currently this endpoint validates input and returns a preview of changes
	// but does NOT persist them to the database.
	slog.Info("UpdatePlanFeatures preview requested",
		"plan", planCode,
		"feature_count", len(req.Features),
		"note", "Changes are not persisted - preview only")
	features := identity.DefaultPlanFeatures(plan)

	// Create a map for quick lookup
	featureMap := make(map[identity.FeatureKey]*identity.PlanFeature)
	for i := range features {
		featureMap[features[i].FeatureKey] = &features[i]
	}

	// Apply updates
	for _, update := range req.Features {
		key := identity.FeatureKey(update.FeatureKey)
		if pf, exists := featureMap[key]; exists {
			if update.Enabled {
				pf.Enable()
			} else {
				pf.Disable()
			}
			if update.Limit != nil {
				if err := pf.SetLimit(*update.Limit); err != nil {
					h.UnprocessableEntity(c, "INVALID_LIMIT", "Invalid limit for feature "+update.FeatureKey+": "+err.Error())
					return
				}
			} else {
				pf.ClearLimit()
			}
		}
	}

	// Convert to response
	featureResponses := make([]PlanFeatureResponse, len(features))
	for i, f := range features {
		featureResponses[i] = PlanFeatureResponse{
			ID:          f.ID.String(),
			FeatureKey:  string(f.FeatureKey),
			Enabled:     f.Enabled,
			Limit:       f.Limit,
			Description: f.Description,
		}
	}

	h.Success(c, PlanFeaturesResponse{
		Plan:     planCode,
		Features: featureResponses,
	})
}

// GetCurrentTenantFeatures godoc
//
//	@ID				getCurrentTenantFeatures
//	@Summary		Get current tenant's available features
//	@Description	Get all features available to the current tenant based on their subscription plan
//	@Tags			plan-features
//	@Produce		json
//	@Success		200	{object}	APIResponse[TenantFeaturesResponse]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/tenants/current/features [get]
func (h *PlanFeatureHandler) GetCurrentTenantFeatures(c *gin.Context) {
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

	// Get tenant to determine their plan
	tenant, err := h.tenantRepo.FindByID(c.Request.Context(), tenantID)
	if err != nil {
		if err == shared.ErrNotFound {
			h.NotFound(c, "Tenant not found")
			return
		}
		slog.Error("failed to retrieve tenant", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve tenant information")
		return
	}

	// Get features for the tenant's plan
	features := identity.DefaultPlanFeatures(tenant.Plan)

	// Filter to only enabled features and convert to response
	enabledFeatures := make([]TenantFeatureResponse, 0, len(features))
	for _, f := range features {
		enabledFeatures = append(enabledFeatures, TenantFeatureResponse{
			FeatureKey:  string(f.FeatureKey),
			Enabled:     f.Enabled,
			Limit:       f.Limit,
			Description: f.Description,
		})
	}

	h.Success(c, TenantFeaturesResponse{
		TenantID: tenantID.String(),
		Plan:     string(tenant.Plan),
		Features: enabledFeatures,
	})
}

// ============================================================================
// Helper functions
// ============================================================================

// isValidPlan checks if a plan code is valid
func isValidPlan(plan identity.TenantPlan) bool {
	switch plan {
	case identity.TenantPlanFree, identity.TenantPlanBasic, identity.TenantPlanPro, identity.TenantPlanEnterprise:
		return true
	default:
		return false
	}
}
