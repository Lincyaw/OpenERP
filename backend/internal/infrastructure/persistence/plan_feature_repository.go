package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormPlanFeatureRepository implements PlanFeatureRepository using GORM
type GormPlanFeatureRepository struct {
	db *gorm.DB
}

// NewGormPlanFeatureRepository creates a new GormPlanFeatureRepository
func NewGormPlanFeatureRepository(db *gorm.DB) *GormPlanFeatureRepository {
	return &GormPlanFeatureRepository{db: db}
}

// FindByID finds a plan feature by its ID
func (r *GormPlanFeatureRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.PlanFeature, error) {
	var model models.PlanFeatureModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByPlan finds all features for a specific plan
func (r *GormPlanFeatureRepository) FindByPlan(ctx context.Context, planID identity.TenantPlan) ([]identity.PlanFeature, error) {
	var featureModels []models.PlanFeatureModel
	if err := r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Order("feature_key ASC").
		Find(&featureModels).Error; err != nil {
		return nil, err
	}

	features := make([]identity.PlanFeature, len(featureModels))
	for i, model := range featureModels {
		features[i] = *model.ToDomain()
	}
	return features, nil
}

// FindByPlanAndFeature finds a specific feature for a plan
func (r *GormPlanFeatureRepository) FindByPlanAndFeature(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (*identity.PlanFeature, error) {
	var model models.PlanFeatureModel
	if err := r.db.WithContext(ctx).
		Where("plan_id = ? AND feature_key = ?", planID, featureKey).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindEnabledByPlan finds all enabled features for a plan
func (r *GormPlanFeatureRepository) FindEnabledByPlan(ctx context.Context, planID identity.TenantPlan) ([]identity.PlanFeature, error) {
	var featureModels []models.PlanFeatureModel
	if err := r.db.WithContext(ctx).
		Where("plan_id = ? AND enabled = ?", planID, true).
		Order("feature_key ASC").
		Find(&featureModels).Error; err != nil {
		return nil, err
	}

	features := make([]identity.PlanFeature, len(featureModels))
	for i, model := range featureModels {
		features[i] = *model.ToDomain()
	}
	return features, nil
}

// HasFeature checks if a plan has a specific feature enabled
func (r *GormPlanFeatureRepository) HasFeature(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PlanFeatureModel{}).
		Where("plan_id = ? AND feature_key = ? AND enabled = ?", planID, featureKey, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetFeatureLimit returns the limit for a feature in a plan (nil if unlimited or not found)
func (r *GormPlanFeatureRepository) GetFeatureLimit(ctx context.Context, planID identity.TenantPlan, featureKey identity.FeatureKey) (*int, error) {
	var model models.PlanFeatureModel
	if err := r.db.WithContext(ctx).
		Where("plan_id = ? AND feature_key = ?", planID, featureKey).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.Limit, nil
}

// Save creates or updates a plan feature
func (r *GormPlanFeatureRepository) Save(ctx context.Context, feature *identity.PlanFeature) error {
	model := models.PlanFeatureModelFromDomain(feature)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveBatch creates or updates multiple plan features in a transaction
func (r *GormPlanFeatureRepository) SaveBatch(ctx context.Context, features []identity.PlanFeature) error {
	if len(features) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, feature := range features {
			model := models.PlanFeatureModelFromDomain(&feature)
			if err := tx.Save(model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete deletes a plan feature
func (r *GormPlanFeatureRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.PlanFeatureModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteByPlan deletes all features for a plan
func (r *GormPlanFeatureRepository) DeleteByPlan(ctx context.Context, planID identity.TenantPlan) error {
	return r.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Delete(&models.PlanFeatureModel{}).Error
}

// SaveWithAuditLog saves a plan feature and records the change in the audit log
func (r *GormPlanFeatureRepository) SaveWithAuditLog(ctx context.Context, feature *identity.PlanFeature, changedBy *uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if feature exists
		var existingModel models.PlanFeatureModel
		err := tx.Where("plan_id = ? AND feature_key = ?", feature.PlanID, feature.FeatureKey).First(&existingModel).Error

		var changeType string
		var oldEnabled *bool
		var oldLimit *int

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// New feature
			changeType = "created"
		} else if err != nil {
			return err
		} else {
			// Existing feature - update
			changeType = "updated"
			oldEnabled = &existingModel.Enabled
			oldLimit = existingModel.Limit
		}

		// Save the feature
		model := models.PlanFeatureModelFromDomain(feature)
		if err := tx.Save(model).Error; err != nil {
			return err
		}

		// Create audit log entry
		auditLog := models.PlanFeatureChangeLogModel{
			ID:         uuid.New(),
			PlanID:     string(feature.PlanID),
			FeatureKey: string(feature.FeatureKey),
			ChangeType: changeType,
			OldEnabled: oldEnabled,
			NewEnabled: &feature.Enabled,
			OldLimit:   oldLimit,
			NewLimit:   feature.Limit,
			ChangedBy:  changedBy,
			ChangedAt:  time.Now(),
		}
		return tx.Create(&auditLog).Error
	})
}

// SaveBatchWithAuditLog saves multiple plan features and records changes in the audit log
func (r *GormPlanFeatureRepository) SaveBatchWithAuditLog(ctx context.Context, features []identity.PlanFeature, changedBy *uuid.UUID) error {
	if len(features) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, feature := range features {
			// Check if feature exists
			var existingModel models.PlanFeatureModel
			err := tx.Where("plan_id = ? AND feature_key = ?", feature.PlanID, feature.FeatureKey).First(&existingModel).Error

			var changeType string
			var oldEnabled *bool
			var oldLimit *int

			if errors.Is(err, gorm.ErrRecordNotFound) {
				changeType = "created"
			} else if err != nil {
				return err
			} else {
				changeType = "updated"
				oldEnabled = &existingModel.Enabled
				oldLimit = existingModel.Limit
			}

			// Save the feature
			model := models.PlanFeatureModelFromDomain(&feature)
			if err := tx.Save(model).Error; err != nil {
				return err
			}

			// Create audit log entry
			auditLog := models.PlanFeatureChangeLogModel{
				ID:         uuid.New(),
				PlanID:     string(feature.PlanID),
				FeatureKey: string(feature.FeatureKey),
				ChangeType: changeType,
				OldEnabled: oldEnabled,
				NewEnabled: &feature.Enabled,
				OldLimit:   oldLimit,
				NewLimit:   feature.Limit,
				ChangedBy:  changedBy,
				ChangedAt:  time.Now(),
			}
			if err := tx.Create(&auditLog).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Ensure GormPlanFeatureRepository implements PlanFeatureRepository
var _ identity.PlanFeatureRepository = (*GormPlanFeatureRepository)(nil)
