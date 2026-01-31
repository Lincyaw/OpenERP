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

// UsageHandler handles tenant usage statistics HTTP requests
type UsageHandler struct {
	BaseHandler
	tenantRepo    identity.TenantRepository
	userRepo      identity.UserRepository
	warehouseRepo WarehouseCounter
	productRepo   ProductCounter
}

// WarehouseCounter interface for counting warehouses
type WarehouseCounter interface {
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)
}

// ProductCounter interface for counting products
type ProductCounter interface {
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)
}

// NewUsageHandler creates a new usage handler
func NewUsageHandler(
	tenantRepo identity.TenantRepository,
	userRepo identity.UserRepository,
	warehouseRepo WarehouseCounter,
	productRepo ProductCounter,
) *UsageHandler {
	return &UsageHandler{
		tenantRepo:    tenantRepo,
		userRepo:      userRepo,
		warehouseRepo: warehouseRepo,
		productRepo:   productRepo,
	}
}

// ============================================================================
// Request/Response DTOs
// ============================================================================

// UsageMetric represents a single usage metric with current value and limit
//
//	@Description	Usage metric with current value and quota limit
type UsageMetric struct {
	Name        string  `json:"name" example:"users"`
	DisplayName string  `json:"display_name" example:"Users"`
	Current     int64   `json:"current" example:"5"`
	Limit       int64   `json:"limit" example:"10"`
	Percentage  float64 `json:"percentage" example:"50.0"`
	Unit        string  `json:"unit" example:"count"`
}

// UsageSummaryResponse represents the current tenant usage summary
//
//	@Description	Current tenant usage summary with all metrics
type UsageSummaryResponse struct {
	TenantID    string        `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantName  string        `json:"tenant_name" example:"Acme Corp"`
	Plan        string        `json:"plan" example:"basic"`
	Metrics     []UsageMetric `json:"metrics"`
	LastUpdated string        `json:"last_updated" example:"2024-01-15T10:30:00Z"`
}

// UsageHistoryPoint represents a single point in usage history
//
//	@Description	Single data point in usage history
type UsageHistoryPoint struct {
	Date       string `json:"date" example:"2024-01-15"`
	Users      int64  `json:"users" example:"5"`
	Products   int64  `json:"products" example:"100"`
	Warehouses int64  `json:"warehouses" example:"2"`
}

// UsageHistoryResponse represents historical usage trends
//
//	@Description	Historical usage trends over time
type UsageHistoryResponse struct {
	TenantID   string              `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Period     string              `json:"period" example:"daily"`
	StartDate  string              `json:"start_date" example:"2024-01-01"`
	EndDate    string              `json:"end_date" example:"2024-01-15"`
	DataPoints []UsageHistoryPoint `json:"data_points"`
}

// QuotaItem represents a single quota with usage information
//
//	@Description	Quota item with current usage and remaining capacity
type QuotaItem struct {
	Resource    string `json:"resource" example:"users"`
	DisplayName string `json:"display_name" example:"Users"`
	Used        int64  `json:"used" example:"5"`
	Limit       int64  `json:"limit" example:"10"`
	Remaining   int64  `json:"remaining" example:"5"`
	IsUnlimited bool   `json:"is_unlimited" example:"false"`
}

// QuotasResponse represents all quotas for a tenant
//
//	@Description	All quotas and remaining capacity for a tenant
type QuotasResponse struct {
	TenantID string      `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Plan     string      `json:"plan" example:"basic"`
	Quotas   []QuotaItem `json:"quotas"`
}

// AdminUsageResponse represents usage data for admin view
//
//	@Description	Detailed usage data for admin viewing a specific tenant
type AdminUsageResponse struct {
	TenantID     string        `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantCode   string        `json:"tenant_code" example:"ACME"`
	TenantName   string        `json:"tenant_name" example:"Acme Corp"`
	Plan         string        `json:"plan" example:"basic"`
	Status       string        `json:"status" example:"active"`
	Metrics      []UsageMetric `json:"metrics"`
	Quotas       []QuotaItem   `json:"quotas"`
	CreatedAt    string        `json:"created_at" example:"2024-01-01T00:00:00Z"`
	LastActivity string        `json:"last_activity" example:"2024-01-15T10:30:00Z"`
}

// ============================================================================
// Handlers
// ============================================================================

// GetCurrentUsage godoc
//
//	@ID				getCurrentUsage
//	@Summary		Get current tenant usage summary
//	@Description	Get usage statistics for the current tenant including users, products, and warehouses
//	@Tags			usage
//	@Produce		json
//	@Success		200	{object}	APIResponse[UsageSummaryResponse]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/tenants/current/usage [get]
func (h *UsageHandler) GetCurrentUsage(c *gin.Context) {
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

	// Get current usage counts
	metrics, err := h.getUsageMetrics(c, tenantID, tenant)
	if err != nil {
		slog.Error("failed to get usage metrics", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve usage metrics")
		return
	}

	h.Success(c, UsageSummaryResponse{
		TenantID:    tenantID.String(),
		TenantName:  tenant.Name,
		Plan:        string(tenant.Plan),
		Metrics:     metrics,
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
	})
}

// GetUsageHistory godoc
//
//	@ID				getUsageHistory
//	@Summary		Get historical usage trends
//	@Description	Get historical usage data for the current tenant. Supports daily, weekly, and monthly periods.
//	@Tags			usage
//	@Produce		json
//	@Param			period		query		string	false	"Time period (daily, weekly, monthly)"	default(daily)	Enums(daily, weekly, monthly)
//	@Param			start_date	query		string	false	"Start date (YYYY-MM-DD)"				example(2024-01-01)
//	@Param			end_date	query		string	false	"End date (YYYY-MM-DD)"					example(2024-01-31)
//	@Success		200			{object}	APIResponse[UsageHistoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/tenants/current/usage/history [get]
func (h *UsageHandler) GetUsageHistory(c *gin.Context) {
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

	// Parse query parameters
	period := c.DefaultQuery("period", "daily")
	if period != "daily" && period != "weekly" && period != "monthly" {
		h.BadRequest(c, "Invalid period. Must be one of: daily, weekly, monthly")
		return
	}

	// Parse date range (default to last 30 days)
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		parsed, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			h.BadRequest(c, "Invalid start_date format. Use YYYY-MM-DD")
			return
		}
		startDate = parsed
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			h.BadRequest(c, "Invalid end_date format. Use YYYY-MM-DD")
			return
		}
		endDate = parsed
	}

	if startDate.After(endDate) {
		h.BadRequest(c, "start_date must be before end_date")
		return
	}

	// Generate historical data points
	// Note: This is a simplified implementation that returns current snapshot values
	// A production system would track historical usage in a dedicated table
	dataPoints, err := h.generateHistoryDataPoints(c, tenantID, period, startDate, endDate)
	if err != nil {
		slog.Error("failed to generate history data points", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve usage history")
		return
	}

	h.Success(c, UsageHistoryResponse{
		TenantID:   tenantID.String(),
		Period:     period,
		StartDate:  startDate.Format("2006-01-02"),
		EndDate:    endDate.Format("2006-01-02"),
		DataPoints: dataPoints,
	})
}

// GetQuotas godoc
//
//	@ID				getQuotas
//	@Summary		Get tenant quotas and remaining capacity
//	@Description	Get all quotas for the current tenant with used and remaining amounts
//	@Tags			usage
//	@Produce		json
//	@Success		200	{object}	APIResponse[QuotasResponse]
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/tenants/current/quotas [get]
func (h *UsageHandler) GetQuotas(c *gin.Context) {
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

	// Get current usage counts
	quotas, err := h.getQuotaItems(c, tenantID, tenant)
	if err != nil {
		slog.Error("failed to get quota items", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve quota information")
		return
	}

	h.Success(c, QuotasResponse{
		TenantID: tenantID.String(),
		Plan:     string(tenant.Plan),
		Quotas:   quotas,
	})
}

// GetTenantUsageByAdmin godoc
//
//	@ID				getTenantUsageByAdmin
//	@Summary		Get tenant usage (admin)
//	@Description	Admin endpoint to view usage statistics for a specific tenant
//	@Tags			usage
//	@Produce		json
//	@Param			id	path		string	true	"Tenant ID"
//	@Success		200	{object}	APIResponse[AdminUsageResponse]
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/tenants/{id}/usage [get]
func (h *UsageHandler) GetTenantUsageByAdmin(c *gin.Context) {
	// Parse tenant ID from path
	tenantIDStr := c.Param("id")
	if tenantIDStr == "" {
		h.BadRequest(c, "Tenant ID is required")
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

	// Get usage metrics
	metrics, err := h.getUsageMetrics(c, tenantID, tenant)
	if err != nil {
		slog.Error("failed to get usage metrics", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve usage metrics")
		return
	}

	// Get quota items
	quotas, err := h.getQuotaItems(c, tenantID, tenant)
	if err != nil {
		slog.Error("failed to get quota items", "tenant_id", tenantID, "error", err)
		h.InternalError(c, "Failed to retrieve quota information")
		return
	}

	h.Success(c, AdminUsageResponse{
		TenantID:     tenantID.String(),
		TenantCode:   tenant.Code,
		TenantName:   tenant.Name,
		Plan:         string(tenant.Plan),
		Status:       string(tenant.Status),
		Metrics:      metrics,
		Quotas:       quotas,
		CreatedAt:    tenant.CreatedAt.Format(time.RFC3339),
		LastActivity: tenant.UpdatedAt.Format(time.RFC3339),
	})
}

// ============================================================================
// Helper functions
// ============================================================================

// getUsageMetrics retrieves current usage metrics for a tenant
func (h *UsageHandler) getUsageMetrics(c *gin.Context, tenantID uuid.UUID, tenant *identity.Tenant) ([]UsageMetric, error) {
	ctx := c.Request.Context()

	// Count users (uses tenant from context)
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Count warehouses
	warehouseCount, err := h.warehouseRepo.CountForTenant(ctx, tenantID, shared.Filter{})
	if err != nil {
		return nil, err
	}

	// Count products
	productCount, err := h.productRepo.CountForTenant(ctx, tenantID, shared.Filter{})
	if err != nil {
		return nil, err
	}

	metrics := []UsageMetric{
		{
			Name:        "users",
			DisplayName: "Users",
			Current:     userCount,
			Limit:       int64(tenant.Config.MaxUsers),
			Percentage:  calculatePercentage(userCount, int64(tenant.Config.MaxUsers)),
			Unit:        "count",
		},
		{
			Name:        "warehouses",
			DisplayName: "Warehouses",
			Current:     warehouseCount,
			Limit:       int64(tenant.Config.MaxWarehouses),
			Percentage:  calculatePercentage(warehouseCount, int64(tenant.Config.MaxWarehouses)),
			Unit:        "count",
		},
		{
			Name:        "products",
			DisplayName: "Products",
			Current:     productCount,
			Limit:       int64(tenant.Config.MaxProducts),
			Percentage:  calculatePercentage(productCount, int64(tenant.Config.MaxProducts)),
			Unit:        "count",
		},
	}

	return metrics, nil
}

// getQuotaItems retrieves quota information for a tenant
func (h *UsageHandler) getQuotaItems(c *gin.Context, tenantID uuid.UUID, tenant *identity.Tenant) ([]QuotaItem, error) {
	ctx := c.Request.Context()

	// Count users (uses tenant from context)
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Count warehouses
	warehouseCount, err := h.warehouseRepo.CountForTenant(ctx, tenantID, shared.Filter{})
	if err != nil {
		return nil, err
	}

	// Count products
	productCount, err := h.productRepo.CountForTenant(ctx, tenantID, shared.Filter{})
	if err != nil {
		return nil, err
	}

	// Check if enterprise plan (unlimited)
	isEnterprise := tenant.Plan == identity.TenantPlanEnterprise

	quotas := []QuotaItem{
		{
			Resource:    "users",
			DisplayName: "Users",
			Used:        userCount,
			Limit:       int64(tenant.Config.MaxUsers),
			Remaining:   calculateRemaining(userCount, int64(tenant.Config.MaxUsers), isEnterprise),
			IsUnlimited: isEnterprise,
		},
		{
			Resource:    "warehouses",
			DisplayName: "Warehouses",
			Used:        warehouseCount,
			Limit:       int64(tenant.Config.MaxWarehouses),
			Remaining:   calculateRemaining(warehouseCount, int64(tenant.Config.MaxWarehouses), isEnterprise),
			IsUnlimited: isEnterprise,
		},
		{
			Resource:    "products",
			DisplayName: "Products",
			Used:        productCount,
			Limit:       int64(tenant.Config.MaxProducts),
			Remaining:   calculateRemaining(productCount, int64(tenant.Config.MaxProducts), isEnterprise),
			IsUnlimited: isEnterprise,
		},
	}

	return quotas, nil
}

// generateHistoryDataPoints generates historical data points
// Note: This is a simplified implementation. A production system would
// track historical usage in a dedicated table for accurate historical data.
func (h *UsageHandler) generateHistoryDataPoints(c *gin.Context, tenantID uuid.UUID, period string, startDate, endDate time.Time) ([]UsageHistoryPoint, error) {
	ctx := c.Request.Context()

	// Get current counts (used as baseline)
	userCount, err := h.userRepo.Count(ctx)
	if err != nil {
		slog.Error("failed to count users for history", "tenant_id", tenantID, "error", err)
		return nil, err
	}
	warehouseCount, err := h.warehouseRepo.CountForTenant(ctx, tenantID, shared.Filter{})
	if err != nil {
		slog.Error("failed to count warehouses for history", "tenant_id", tenantID, "error", err)
		return nil, err
	}
	productCount, err := h.productRepo.CountForTenant(ctx, tenantID, shared.Filter{})
	if err != nil {
		slog.Error("failed to count products for history", "tenant_id", tenantID, "error", err)
		return nil, err
	}

	var dataPoints []UsageHistoryPoint
	var step time.Duration

	switch period {
	case "weekly":
		step = 7 * 24 * time.Hour
	case "monthly":
		step = 30 * 24 * time.Hour
	default: // daily
		step = 24 * time.Hour
	}

	// Generate data points from start to end date
	// Using current values as the baseline (simplified approach)
	for date := startDate; !date.After(endDate); date = date.Add(step) {
		dataPoints = append(dataPoints, UsageHistoryPoint{
			Date:       date.Format("2006-01-02"),
			Users:      userCount,
			Products:   productCount,
			Warehouses: warehouseCount,
		})
	}

	return dataPoints, nil
}

// calculatePercentage calculates usage percentage
func calculatePercentage(current, limit int64) float64 {
	if limit <= 0 {
		return 0
	}
	percentage := float64(current) / float64(limit) * 100
	// Round to 2 decimal places
	return float64(int(percentage*100)) / 100
}

// calculateRemaining calculates remaining quota
func calculateRemaining(used, limit int64, isUnlimited bool) int64 {
	if isUnlimited {
		return -1 // -1 indicates unlimited
	}
	remaining := limit - used
	if remaining < 0 {
		return 0
	}
	return remaining
}
