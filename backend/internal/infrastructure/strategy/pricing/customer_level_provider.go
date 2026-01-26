package pricing

import (
	"context"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/google/uuid"
)

// GormCustomerLevelProvider implements CustomerLevelProvider using the CustomerLevelRepository.
// This provider allows the CustomerLevelPricingStrategy to look up discount rates dynamically
// from the database via the domain repository.
type GormCustomerLevelProvider struct {
	repo partner.CustomerLevelRepository
}

// NewGormCustomerLevelProvider creates a new GormCustomerLevelProvider.
func NewGormCustomerLevelProvider(repo partner.CustomerLevelRepository) *GormCustomerLevelProvider {
	return &GormCustomerLevelProvider{repo: repo}
}

// GetCustomerLevel retrieves the customer level by code for a tenant.
// Returns the CustomerLevel value object with full details (code, name, discountRate).
// Returns an error if the level is not found.
func (p *GormCustomerLevelProvider) GetCustomerLevel(
	ctx context.Context,
	tenantID uuid.UUID,
	levelCode string,
) (partner.CustomerLevel, error) {
	record, err := p.repo.FindByCode(ctx, tenantID, levelCode)
	if err != nil {
		return partner.CustomerLevel{}, err
	}
	return record.ToCustomerLevel(), nil
}

// GetAllCustomerLevels retrieves all active customer levels for a tenant.
// Returns an empty slice if no levels are configured.
func (p *GormCustomerLevelProvider) GetAllCustomerLevels(
	ctx context.Context,
	tenantID uuid.UUID,
) ([]partner.CustomerLevel, error) {
	records, err := p.repo.FindActiveForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	levels := make([]partner.CustomerLevel, 0, len(records))
	for _, r := range records {
		levels = append(levels, r.ToCustomerLevel())
	}
	return levels, nil
}

// Ensure GormCustomerLevelProvider implements CustomerLevelProvider
var _ CustomerLevelProvider = (*GormCustomerLevelProvider)(nil)
