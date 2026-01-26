package partner

import (
	"context"

	"github.com/google/uuid"
)

// CustomerLevelRepository defines the interface for customer level persistence
type CustomerLevelRepository interface {
	// FindByID finds a customer level by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*CustomerLevelRecord, error)

	// FindByIDForTenant finds a customer level by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*CustomerLevelRecord, error)

	// FindByCode finds a customer level by its code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*CustomerLevelRecord, error)

	// FindAllForTenant finds all customer levels for a tenant (sorted by sort_order)
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID) ([]*CustomerLevelRecord, error)

	// FindActiveForTenant finds all active customer levels for a tenant
	FindActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]*CustomerLevelRecord, error)

	// FindDefaultForTenant finds the default customer level for a tenant
	FindDefaultForTenant(ctx context.Context, tenantID uuid.UUID) (*CustomerLevelRecord, error)

	// Save creates or updates a customer level
	Save(ctx context.Context, record *CustomerLevelRecord) error

	// Delete deletes a customer level by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a customer level within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// ExistsByCode checks if a customer level with the given code exists in the tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// CountCustomersWithLevel counts customers using a specific level code
	CountCustomersWithLevel(ctx context.Context, tenantID uuid.UUID, levelCode string) (int64, error)

	// CountCustomersByLevelCodes counts customers grouped by level codes in a single query
	// Returns a map of level code -> customer count
	CountCustomersByLevelCodes(ctx context.Context, tenantID uuid.UUID, codes []string) (map[string]int64, error)

	// InitializeDefaultLevels creates the default customer levels for a new tenant
	InitializeDefaultLevels(ctx context.Context, tenantID uuid.UUID) error
}
