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

// featureDisplayNames maps feature keys to their Chinese display names
// Package-level variable for performance (avoid recreating on each call)
var featureDisplayNames = map[identity.FeatureKey]string{
	identity.FeatureMultiWarehouse:    "多仓库管理",
	identity.FeatureBatchManagement:   "批次管理",
	identity.FeatureSerialTracking:    "序列号追踪",
	identity.FeatureMultiCurrency:     "多币种支持",
	identity.FeatureAdvancedReporting: "高级报表",
	identity.FeatureAPIAccess:         "API 访问",
	identity.FeatureCustomFields:      "自定义字段",
	identity.FeatureAuditLog:          "审计日志",
	identity.FeatureDataExport:        "数据导出",
	identity.FeatureDataImport:        "数据导入",
	identity.FeatureSalesOrders:       "销售订单",
	identity.FeaturePurchaseOrders:    "采购订单",
	identity.FeatureSalesReturns:      "销售退货",
	identity.FeaturePurchaseReturns:   "采购退货",
	identity.FeatureQuotations:        "报价单",
	identity.FeaturePriceManagement:   "价格管理",
	identity.FeatureDiscountRules:     "折扣规则",
	identity.FeatureCreditManagement:  "信用管理",
	identity.FeatureReceivables:       "应收账款",
	identity.FeaturePayables:          "应付账款",
	identity.FeatureReconciliation:    "账户对账",
	identity.FeatureExpenseTracking:   "费用追踪",
	identity.FeatureFinancialReports:  "财务报表",
	identity.FeatureWorkflowApproval:  "工作流审批",
	identity.FeatureNotifications:     "通知提醒",
	identity.FeatureIntegrations:      "第三方集成",
	identity.FeatureWhiteLabeling:     "白标定制",
	identity.FeaturePrioritySupport:   "优先支持",
	identity.FeatureDedicatedSupport:  "专属客服",
	identity.FeatureSLA:               "服务等级协议",
}

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

// PlanQuotaResponse represents a quota definition for a plan
//
//	@Description	Quota definition for a subscription plan
type PlanQuotaResponse struct {
	Type  string `json:"type" example:"users"`
	Limit int64  `json:"limit" example:"10"`
	Unit  string `json:"unit" example:"count"`
}

// SubscriptionPlanFeatureResponse represents a feature definition for a plan in the plans list
//
//	@Description	Feature definition for a subscription plan
type SubscriptionPlanFeatureResponse struct {
	Key         string `json:"key" example:"multi_warehouse"`
	Name        string `json:"name" example:"多仓库管理"`
	Description string `json:"description" example:"Multiple warehouse management"`
	Enabled     bool   `json:"enabled" example:"true"`
	Limit       *int   `json:"limit,omitempty" example:"10"`
}

// SubscriptionPlanResponse represents a subscription plan
//
//	@Description	Subscription plan with quotas and features
type SubscriptionPlanResponse struct {
	ID          string                            `json:"id" example:"basic"`
	Name        string                            `json:"name" example:"基础版"`
	Description string                            `json:"description" example:"适合小型团队"`
	Price       int64                             `json:"price" example:"199"`
	PriceUnit   string                            `json:"price_unit" example:"CNY/月"`
	Quotas      []PlanQuotaResponse               `json:"quotas"`
	Features    []SubscriptionPlanFeatureResponse `json:"features"`
	Highlighted bool                              `json:"highlighted" example:"false"`
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

// GetPlans godoc
//
//	@ID				getPlans
//	@Summary		Get available subscription plans
//	@Description	Get all available subscription plans with their quotas and features. This is a public endpoint - no authentication required.
//	@Tags			billing
//	@Produce		json
//	@Success		200	{object}	APIResponse[[]SubscriptionPlanResponse]
//	@Failure		500	{object}	dto.ErrorResponse
//	@Router			/billing/plans [get]
func (h *SubscriptionHandler) GetPlans(c *gin.Context) {
	// Set cache headers for CDN/browser caching (plans rarely change)
	c.Header("Cache-Control", "public, max-age=3600") // 1 hour
	c.Header("Vary", "Accept-Language")

	plans := h.buildAvailablePlans()
	h.Success(c, plans)
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

// buildAvailablePlans builds the list of available subscription plans
func (h *SubscriptionHandler) buildAvailablePlans() []SubscriptionPlanResponse {
	plans := []SubscriptionPlanResponse{
		h.buildPlanResponse(identity.TenantPlanFree, "免费版", "适合个人和小型团队", 0, false),
		h.buildPlanResponse(identity.TenantPlanBasic, "基础版", "适合成长中的企业", 199, false),
		h.buildPlanResponse(identity.TenantPlanPro, "专业版", "适合中型企业", 599, true),
		h.buildPlanResponse(identity.TenantPlanEnterprise, "企业版", "适合大型企业，按需定价", 0, false),
	}
	return plans
}

// buildPlanResponse builds a single plan response
func (h *SubscriptionHandler) buildPlanResponse(plan identity.TenantPlan, name, description string, price int64, highlighted bool) SubscriptionPlanResponse {
	// Get quotas for the plan
	quotas := getPlanQuotas(plan)

	// Get features for the plan
	features := getPlanFeatures(plan)

	// Determine price unit
	priceUnit := "CNY/月"
	if plan == identity.TenantPlanEnterprise {
		priceUnit = "按需定价"
	}

	return SubscriptionPlanResponse{
		ID:          string(plan),
		Name:        name,
		Description: description,
		Price:       price,
		PriceUnit:   priceUnit,
		Quotas:      quotas,
		Features:    features,
		Highlighted: highlighted,
	}
}

// getPlanQuotas returns the quota definitions for a plan
func getPlanQuotas(plan identity.TenantPlan) []PlanQuotaResponse {
	var maxUsers, maxWarehouses, maxProducts int64

	switch plan {
	case identity.TenantPlanFree:
		maxUsers = 5
		maxWarehouses = 3
		maxProducts = 1000
	case identity.TenantPlanBasic:
		maxUsers = 10
		maxWarehouses = 5
		maxProducts = 5000
	case identity.TenantPlanPro:
		maxUsers = 50
		maxWarehouses = 20
		maxProducts = 50000
	case identity.TenantPlanEnterprise:
		maxUsers = -1 // unlimited
		maxWarehouses = -1
		maxProducts = -1
	}

	return []PlanQuotaResponse{
		{Type: "users", Limit: maxUsers, Unit: "count"},
		{Type: "warehouses", Limit: maxWarehouses, Unit: "count"},
		{Type: "products", Limit: maxProducts, Unit: "count"},
	}
}

// getPlanFeatures returns the feature definitions for a plan
func getPlanFeatures(plan identity.TenantPlan) []SubscriptionPlanFeatureResponse {
	defaultFeatures := identity.DefaultPlanFeatures(plan)
	features := make([]SubscriptionPlanFeatureResponse, 0, len(defaultFeatures))

	for _, f := range defaultFeatures {
		features = append(features, SubscriptionPlanFeatureResponse{
			Key:         string(f.FeatureKey),
			Name:        getFeatureDisplayName(f.FeatureKey),
			Description: f.Description,
			Enabled:     f.Enabled,
			Limit:       f.Limit,
		})
	}

	return features
}

// getFeatureDisplayName returns the display name for a feature key
func getFeatureDisplayName(key identity.FeatureKey) string {
	if name, ok := featureDisplayNames[key]; ok {
		return name
	}
	return string(key)
}
