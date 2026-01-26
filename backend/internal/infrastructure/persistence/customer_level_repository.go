package persistence

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormCustomerLevelRepository implements CustomerLevelRepository using GORM
type GormCustomerLevelRepository struct {
	db *gorm.DB
}

// NewGormCustomerLevelRepository creates a new GormCustomerLevelRepository
func NewGormCustomerLevelRepository(db *gorm.DB) *GormCustomerLevelRepository {
	return &GormCustomerLevelRepository{db: db}
}

// FindByID finds a customer level by its ID
func (r *GormCustomerLevelRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.CustomerLevelRecord, error) {
	var record partner.CustomerLevelRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &record, nil
}

// FindByIDForTenant finds a customer level by ID within a tenant
func (r *GormCustomerLevelRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.CustomerLevelRecord, error) {
	var record partner.CustomerLevelRecord
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &record, nil
}

// FindByCode finds a customer level by its code within a tenant
func (r *GormCustomerLevelRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.CustomerLevelRecord, error) {
	var record partner.CustomerLevelRecord
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, code).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &record, nil
}

// FindAllForTenant finds all customer levels for a tenant (sorted by sort_order)
func (r *GormCustomerLevelRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID) ([]*partner.CustomerLevelRecord, error) {
	var records []*partner.CustomerLevelRecord
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("sort_order ASC, name ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// FindActiveForTenant finds all active customer levels for a tenant
func (r *GormCustomerLevelRepository) FindActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]*partner.CustomerLevelRecord, error) {
	var records []*partner.CustomerLevelRecord
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("sort_order ASC, name ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// FindDefaultForTenant finds the default customer level for a tenant
func (r *GormCustomerLevelRepository) FindDefaultForTenant(ctx context.Context, tenantID uuid.UUID) (*partner.CustomerLevelRecord, error) {
	var record partner.CustomerLevelRecord
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_default = ? AND is_active = ?", tenantID, true, true).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If no default level found, try to get the first active level
			if err := r.db.WithContext(ctx).
				Where("tenant_id = ? AND is_active = ?", tenantID, true).
				Order("sort_order ASC").
				First(&record).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, shared.ErrNotFound
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &record, nil
}

// Save creates or updates a customer level
func (r *GormCustomerLevelRepository) Save(ctx context.Context, record *partner.CustomerLevelRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// Delete deletes a customer level by ID
func (r *GormCustomerLevelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&partner.CustomerLevelRecord{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// DeleteForTenant deletes a customer level within a tenant
func (r *GormCustomerLevelRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&partner.CustomerLevelRecord{}, "tenant_id = ? AND id = ?", tenantID, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// ExistsByCode checks if a customer level with the given code exists in the tenant
func (r *GormCustomerLevelRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&partner.CustomerLevelRecord{}).
		Where("tenant_id = ? AND code = ?", tenantID, code).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountCustomersWithLevel counts customers using a specific level code
func (r *GormCustomerLevelRepository) CountCustomersWithLevel(ctx context.Context, tenantID uuid.UUID, levelCode string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&partner.Customer{}).
		Where("tenant_id = ? AND level = ?", tenantID, levelCode).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountCustomersByLevelCodes counts customers grouped by level codes in a single query
func (r *GormCustomerLevelRepository) CountCustomersByLevelCodes(ctx context.Context, tenantID uuid.UUID, codes []string) (map[string]int64, error) {
	if len(codes) == 0 {
		return make(map[string]int64), nil
	}

	type levelCount struct {
		Level string
		Count int64
	}
	var results []levelCount

	if err := r.db.WithContext(ctx).
		Model(&partner.Customer{}).
		Select("level, count(*) as count").
		Where("tenant_id = ? AND level IN ?", tenantID, codes).
		Group("level").
		Find(&results).Error; err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Level] = r.Count
	}
	return counts, nil
}

// InitializeDefaultLevels creates the default customer levels for a new tenant
// Uses a transaction to prevent race conditions when multiple requests call this simultaneously
func (r *GormCustomerLevelRepository) InitializeDefaultLevels(ctx context.Context, tenantID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if tenant already has levels (within transaction for consistency)
		var count int64
		if err := tx.Model(&partner.CustomerLevelRecord{}).
			Where("tenant_id = ?", tenantID).
			Count(&count).Error; err != nil {
			return err
		}

		// Skip if tenant already has levels
		if count > 0 {
			return nil
		}

		// Create default levels
		records := partner.DefaultCustomerLevelRecords(tenantID)
		for _, record := range records {
			if err := tx.Create(record).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Ensure GormCustomerLevelRepository implements CustomerLevelRepository
var _ partner.CustomerLevelRepository = (*GormCustomerLevelRepository)(nil)
