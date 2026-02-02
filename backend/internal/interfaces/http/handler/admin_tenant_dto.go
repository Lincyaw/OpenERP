package handler

import (
	"time"

	appIdentity "github.com/erp/backend/internal/application/identity"
	"github.com/google/uuid"
)

// ChangePlanRequest represents a plan change request
//
//	@Description Request body for changing tenant subscription plan
type ChangePlanRequest struct {
	Plan   string `json:"plan" binding:"required,oneof=free basic pro enterprise" example:"enterprise"`
	Reason string `json:"reason,omitempty" example:"Customer upgraded after successful pilot program"`
}

// ChangePlanResponse represents a plan change response
//
//	@Description Response for plan change operation with effective date information
type ChangePlanResponse struct {
	Tenant       AdminTenantResponse `json:"tenant"`
	ChangeType   string              `json:"change_type" example:"immediate"`
	EffectiveAt  time.Time           `json:"effective_at" example:"2024-02-01T00:00:00Z"`
	PreviousPlan string              `json:"previous_plan" example:"pro"`
	NewPlan      string              `json:"new_plan" example:"enterprise"`
}

// UpdateQuotaRequest represents a quota update request
//
//	@Description Request body for updating tenant quota limits
type UpdateQuotaRequest struct {
	MaxUsers      *int   `json:"max_users,omitempty" binding:"omitempty,min=1" example:"100"`
	MaxWarehouses *int   `json:"max_warehouses,omitempty" binding:"omitempty,min=1" example:"20"`
	MaxProducts   *int   `json:"max_products,omitempty" binding:"omitempty,min=1" example:"10000"`
	Reason        string `json:"reason,omitempty" example:"Enterprise customer requires additional capacity"`
}

// AdminTenantResponse represents admin tenant data
//
//	@Description Complete tenant information for admin view
type AdminTenantResponse struct {
	ID                       uuid.UUID                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code                     string                    `json:"code" example:"ACME001"`
	Name                     string                    `json:"name" example:"Acme Corporation"`
	ShortName                string                    `json:"short_name,omitempty" example:"Acme"`
	Status                   string                    `json:"status" example:"active"`
	Plan                     string                    `json:"plan" example:"pro"`
	ScheduledPlan            *string                   `json:"scheduled_plan,omitempty" example:"enterprise"`
	ScheduledPlanEffectiveAt *time.Time                `json:"scheduled_plan_effective_at,omitempty" example:"2024-03-01T00:00:00Z"`
	ContactName              string                    `json:"contact_name,omitempty" example:"John Smith"`
	ContactPhone             string                    `json:"contact_phone,omitempty" example:"+1-555-123-4567"`
	ContactEmail             string                    `json:"contact_email,omitempty" example:"john.smith@acme.com"`
	Address                  string                    `json:"address,omitempty" example:"123 Business Ave, Suite 100, New York, NY 10001"`
	LogoURL                  string                    `json:"logo_url,omitempty" example:"https://cdn.example.com/logos/acme.png"`
	Domain                   string                    `json:"domain,omitempty" example:"acme.erp.example.com"`
	ExpiresAt                *time.Time                `json:"expires_at,omitempty" example:"2025-02-01T00:00:00Z"`
	TrialEndsAt              *time.Time                `json:"trial_ends_at,omitempty" example:"2024-02-15T00:00:00Z"`
	Config                   AdminTenantConfigResponse `json:"config"`
	Notes                    string                    `json:"notes,omitempty" example:"Enterprise customer, signed 2-year contract"`
	Statistics               *AdminTenantStatsResponse `json:"statistics,omitempty"`
	CreatedAt                time.Time                 `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt                time.Time                 `json:"updated_at" example:"2024-02-01T14:20:00Z"`
}

// TenantConfigResponse represents tenant configuration
//
//	@Description Tenant configuration including quotas and regional settings
type AdminTenantConfigResponse struct {
	MaxUsers      int    `json:"max_users" example:"50"`
	MaxWarehouses int    `json:"max_warehouses" example:"10"`
	MaxProducts   int    `json:"max_products" example:"5000"`
	CostStrategy  string `json:"cost_strategy" example:"weighted_average"`
	Currency      string `json:"currency" example:"USD"`
	Timezone      string `json:"timezone" example:"America/New_York"`
	Locale        string `json:"locale" example:"en-US"`
}

// TenantStatsResponse represents tenant usage statistics
//
//	@Description Current usage statistics for a tenant
type AdminTenantStatsResponse struct {
	UserCount      int64 `json:"user_count" example:"25"`
	WarehouseCount int64 `json:"warehouse_count" example:"3"`
	ProductCount   int64 `json:"product_count" example:"1500"`
	OrderCount     int64 `json:"order_count" example:"3200"`
}

// SubscriptionHistoryQuery represents query parameters for subscription history
//
//	@Description Query parameters for subscription history endpoint
type SubscriptionHistoryQuery struct {
	Page       int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"`
	ChangeType string `form:"change_type" binding:"omitempty,oneof=plan_upgrade plan_downgrade quota_update" example:"plan_upgrade"`
}

// SubscriptionHistoryResponse represents a subscription history entry
//
//	@Description Single subscription change history entry
type SubscriptionHistoryResponse struct {
	ID              uuid.UUID      `json:"id" example:"550e8400-e29b-41d4-a716-446655440088"`
	TenantID        uuid.UUID      `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChangeType      string         `json:"change_type" example:"plan_upgrade"`
	OldPlan         string         `json:"old_plan,omitempty" example:"pro"`
	NewPlan         string         `json:"new_plan,omitempty" example:"enterprise"`
	OldQuota        *QuotaResponse `json:"old_quota,omitempty"`
	NewQuota        *QuotaResponse `json:"new_quota,omitempty"`
	EffectiveAt     time.Time      `json:"effective_at" example:"2024-02-01T00:00:00Z"`
	ScheduledAt     *time.Time     `json:"scheduled_at,omitempty" example:"2024-03-01T00:00:00Z"`
	ChangedByUserID uuid.UUID      `json:"changed_by_user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Reason          string         `json:"reason,omitempty" example:"Customer upgraded after successful pilot"`
	CreatedAt       time.Time      `json:"created_at" example:"2024-02-01T10:30:00Z"`
}

// QuotaResponse represents quota limits
//
//	@Description Tenant quota limits
type QuotaResponse struct {
	MaxUsers      int `json:"max_users" example:"50"`
	MaxWarehouses int `json:"max_warehouses" example:"10"`
	MaxProducts   int `json:"max_products" example:"5000"`
}

// SubscriptionHistoryListResponse represents paginated history
//
//	@Description Paginated list of subscription history entries
type SubscriptionHistoryListResponse struct {
	History    []SubscriptionHistoryResponse `json:"history"`
	Total      int64                         `json:"total" example:"25"`
	Page       int                           `json:"page" example:"1"`
	PageSize   int                           `json:"page_size" example:"20"`
	TotalPages int                           `json:"total_pages" example:"2"`
}

// toAdminTenantResponse converts application DTO to HTTP response
func toAdminTenantResponse(dto *appIdentity.AdminTenantDTO) AdminTenantResponse {
	response := AdminTenantResponse{
		ID:                       dto.ID,
		Code:                     dto.Code,
		Name:                     dto.Name,
		ShortName:                dto.ShortName,
		Status:                   dto.Status,
		Plan:                     dto.Plan,
		ScheduledPlan:            dto.ScheduledPlan,
		ScheduledPlanEffectiveAt: dto.ScheduledPlanEffectiveAt,
		ContactName:              dto.ContactName,
		ContactPhone:             dto.ContactPhone,
		ContactEmail:             dto.ContactEmail,
		Address:                  dto.Address,
		LogoURL:                  dto.LogoURL,
		Domain:                   dto.Domain,
		ExpiresAt:                dto.ExpiresAt,
		TrialEndsAt:              dto.TrialEndsAt,
		Config: AdminTenantConfigResponse{
			MaxUsers:      dto.Config.MaxUsers,
			MaxWarehouses: dto.Config.MaxWarehouses,
			MaxProducts:   dto.Config.MaxProducts,
			CostStrategy:  dto.Config.CostStrategy,
			Currency:      dto.Config.Currency,
			Timezone:      dto.Config.Timezone,
			Locale:        dto.Config.Locale,
		},
		Notes:     dto.Notes,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}

	if dto.Statistics != nil {
		response.Statistics = &AdminTenantStatsResponse{
			UserCount:      dto.Statistics.UserCount,
			WarehouseCount: dto.Statistics.WarehouseCount,
			ProductCount:   dto.Statistics.ProductCount,
			OrderCount:     dto.Statistics.OrderCount,
		}
	}

	return response
}

// toSubscriptionHistoryResponse converts application DTO to HTTP response
func toSubscriptionHistoryResponse(dto *appIdentity.SubscriptionHistoryDTO) SubscriptionHistoryResponse {
	response := SubscriptionHistoryResponse{
		ID:              dto.ID,
		TenantID:        dto.TenantID,
		ChangeType:      dto.ChangeType,
		OldPlan:         dto.OldPlan,
		NewPlan:         dto.NewPlan,
		EffectiveAt:     dto.EffectiveAt,
		ScheduledAt:     dto.ScheduledAt,
		ChangedByUserID: dto.ChangedByUserID,
		Reason:          dto.Reason,
		CreatedAt:       dto.CreatedAt,
	}

	if dto.OldQuota != nil {
		response.OldQuota = &QuotaResponse{
			MaxUsers:      dto.OldQuota.MaxUsers,
			MaxWarehouses: dto.OldQuota.MaxWarehouses,
			MaxProducts:   dto.OldQuota.MaxProducts,
		}
	}

	if dto.NewQuota != nil {
		response.NewQuota = &QuotaResponse{
			MaxUsers:      dto.NewQuota.MaxUsers,
			MaxWarehouses: dto.NewQuota.MaxWarehouses,
			MaxProducts:   dto.NewQuota.MaxProducts,
		}
	}

	return response
}
