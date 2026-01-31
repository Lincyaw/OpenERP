package persistence

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsageQuotaModel is the GORM model for usage quotas
type UsageQuotaModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	PlanID        string     `gorm:"type:varchar(50);not null"`
	TenantID      *uuid.UUID `gorm:"type:uuid;index"`
	UsageType     string     `gorm:"type:varchar(50);not null"`
	QuotaLimit    int64      `gorm:"column:quota_limit;not null;default:-1"`
	Unit          string     `gorm:"type:varchar(20);not null;default:'requests'"`
	ResetPeriod   string     `gorm:"type:varchar(20);not null;default:'MONTHLY'"`
	SoftLimit     *int64     `gorm:"column:soft_limit"`
	OveragePolicy string     `gorm:"type:varchar(20);not null;default:'BLOCK'"`
	Description   string     `gorm:"type:text"`
	IsActive      bool       `gorm:"not null;default:true"`
	Version       int        `gorm:"not null;default:1"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

// TableName returns the table name for the model
func (UsageQuotaModel) TableName() string {
	return "usage_quotas"
}

// ToEntity converts the model to a domain entity
func (m *UsageQuotaModel) ToEntity() *billing.UsageQuota {
	usageType, _ := billing.ParseUsageType(m.UsageType)

	return &billing.UsageQuota{
		BaseAggregateRoot: shared.BaseAggregateRoot{
			BaseEntity: shared.BaseEntity{
				ID:        m.ID,
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
			},
			Version: m.Version,
		},
		PlanID:        m.PlanID,
		TenantID:      m.TenantID,
		UsageType:     usageType,
		Limit:         m.QuotaLimit,
		Unit:          billing.UsageUnit(m.Unit),
		ResetPeriod:   billing.ResetPeriod(m.ResetPeriod),
		SoftLimit:     m.SoftLimit,
		OveragePolicy: billing.OveragePolicy(m.OveragePolicy),
		Description:   m.Description,
		IsActive:      m.IsActive,
	}
}

// UsageQuotaModelFromEntity creates a model from a domain entity
func UsageQuotaModelFromEntity(e *billing.UsageQuota) *UsageQuotaModel {
	return &UsageQuotaModel{
		ID:            e.ID,
		PlanID:        e.PlanID,
		TenantID:      e.TenantID,
		UsageType:     string(e.UsageType),
		QuotaLimit:    e.Limit,
		Unit:          string(e.Unit),
		ResetPeriod:   string(e.ResetPeriod),
		SoftLimit:     e.SoftLimit,
		OveragePolicy: string(e.OveragePolicy),
		Description:   e.Description,
		IsActive:      e.IsActive,
		Version:       e.Version,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

// UsageQuotaRepository implements the billing.UsageQuotaRepository interface
type UsageQuotaRepository struct {
	db *gorm.DB
}

// NewUsageQuotaRepository creates a new usage quota repository
func NewUsageQuotaRepository(db *gorm.DB) *UsageQuotaRepository {
	return &UsageQuotaRepository{db: db}
}

// Save persists a usage quota
func (r *UsageQuotaRepository) Save(ctx context.Context, quota *billing.UsageQuota) error {
	model := UsageQuotaModelFromEntity(quota)
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing usage quota
func (r *UsageQuotaRepository) Update(ctx context.Context, quota *billing.UsageQuota) error {
	model := UsageQuotaModelFromEntity(quota)
	return r.db.WithContext(ctx).Save(model).Error
}

// FindByID retrieves a usage quota by its ID
func (r *UsageQuotaRepository) FindByID(ctx context.Context, id uuid.UUID) (*billing.UsageQuota, error) {
	var model UsageQuotaModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindByPlan retrieves all quotas for a subscription plan
func (r *UsageQuotaRepository) FindByPlan(ctx context.Context, planID string) ([]*billing.UsageQuota, error) {
	var models []UsageQuotaModel
	err := r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Where("tenant_id IS NULL").
		Where("is_active = ?", true).
		Order("usage_type ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	quotas := make([]*billing.UsageQuota, len(models))
	for i, model := range models {
		quotas[i] = model.ToEntity()
	}
	return quotas, nil
}

// FindByPlanAndType retrieves a specific quota for a plan and usage type
func (r *UsageQuotaRepository) FindByPlanAndType(ctx context.Context, planID string, usageType billing.UsageType) (*billing.UsageQuota, error) {
	var model UsageQuotaModel
	err := r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Where("usage_type = ?", string(usageType)).
		Where("tenant_id IS NULL").
		Where("is_active = ?", true).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindByTenant retrieves tenant-specific quota overrides
func (r *UsageQuotaRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*billing.UsageQuota, error) {
	var models []UsageQuotaModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("is_active = ?", true).
		Order("usage_type ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	quotas := make([]*billing.UsageQuota, len(models))
	for i, model := range models {
		quotas[i] = model.ToEntity()
	}
	return quotas, nil
}

// FindByTenantAndType retrieves a tenant-specific quota for a usage type
func (r *UsageQuotaRepository) FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, usageType billing.UsageType) (*billing.UsageQuota, error) {
	var model UsageQuotaModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("usage_type = ?", string(usageType)).
		Where("is_active = ?", true).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindEffectiveQuota retrieves the effective quota for a tenant and usage type
// (tenant override if exists, otherwise plan default)
func (r *UsageQuotaRepository) FindEffectiveQuota(ctx context.Context, tenantID uuid.UUID, planID string, usageType billing.UsageType) (*billing.UsageQuota, error) {
	// First, try to find a tenant-specific override
	tenantQuota, err := r.FindByTenantAndType(ctx, tenantID, usageType)
	if err == nil {
		return tenantQuota, nil
	}
	if err != shared.ErrNotFound {
		return nil, err
	}

	// Fall back to plan default
	return r.FindByPlanAndType(ctx, planID, usageType)
}

// FindAllEffectiveQuotas retrieves all effective quotas for a tenant
func (r *UsageQuotaRepository) FindAllEffectiveQuotas(ctx context.Context, tenantID uuid.UUID, planID string) ([]*billing.UsageQuota, error) {
	// Get all plan defaults
	planQuotas, err := r.FindByPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Get all tenant overrides
	tenantQuotas, err := r.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Build a map of tenant overrides by usage type
	tenantOverrides := make(map[billing.UsageType]*billing.UsageQuota)
	for _, q := range tenantQuotas {
		tenantOverrides[q.UsageType] = q
	}

	// Merge: use tenant override if exists, otherwise use plan default
	effectiveQuotas := make([]*billing.UsageQuota, 0, len(planQuotas))
	for _, planQuota := range planQuotas {
		if override, exists := tenantOverrides[planQuota.UsageType]; exists {
			effectiveQuotas = append(effectiveQuotas, override)
		} else {
			effectiveQuotas = append(effectiveQuotas, planQuota)
		}
	}

	return effectiveQuotas, nil
}

// Delete removes a usage quota
func (r *UsageQuotaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&UsageQuotaModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteByTenant removes all tenant-specific quota overrides
func (r *UsageQuotaRepository) DeleteByTenant(ctx context.Context, tenantID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Delete(&UsageQuotaModel{}).Error
}

// Ensure UsageQuotaRepository implements the interface
var _ billing.UsageQuotaRepository = (*UsageQuotaRepository)(nil)
