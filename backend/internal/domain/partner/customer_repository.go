package partner

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// CustomerRepository defines the interface for customer persistence
type CustomerRepository interface {
	// FindByID finds a customer by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)

	// FindByIDForTenant finds a customer by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error)

	// FindByCode finds a customer by its code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Customer, error)

	// FindByPhone finds a customer by phone number within a tenant
	FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*Customer, error)

	// FindByEmail finds a customer by email within a tenant
	FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*Customer, error)

	// FindAll finds all customers matching the filter
	FindAll(ctx context.Context, filter shared.Filter) ([]Customer, error)

	// FindAllForTenant finds all customers for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Customer, error)

	// FindByType finds customers by type (individual/organization)
	FindByType(ctx context.Context, tenantID uuid.UUID, customerType CustomerType, filter shared.Filter) ([]Customer, error)

	// FindByLevel finds customers by tier level
	FindByLevel(ctx context.Context, tenantID uuid.UUID, level CustomerLevel, filter shared.Filter) ([]Customer, error)

	// FindByStatus finds customers by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status CustomerStatus, filter shared.Filter) ([]Customer, error)

	// FindActive finds all active customers for a tenant
	FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Customer, error)

	// FindByIDs finds multiple customers by their IDs
	FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]Customer, error)

	// FindByCodes finds multiple customers by their codes
	FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]Customer, error)

	// FindWithPositiveBalance finds customers with prepaid balance > 0
	FindWithPositiveBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Customer, error)

	// Save creates or updates a customer
	Save(ctx context.Context, customer *Customer) error

	// SaveWithLock saves a customer with optimistic locking (version check)
	// Returns error if the version has changed (concurrent modification)
	SaveWithLock(ctx context.Context, customer *Customer) error

	// SaveBatch creates or updates multiple customers
	SaveBatch(ctx context.Context, customers []*Customer) error

	// Delete deletes a customer
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a customer within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// Count counts customers matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountForTenant counts customers for a tenant
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByType counts customers by type for a tenant
	CountByType(ctx context.Context, tenantID uuid.UUID, customerType CustomerType) (int64, error)

	// CountByLevel counts customers by level for a tenant
	CountByLevel(ctx context.Context, tenantID uuid.UUID, level CustomerLevel) (int64, error)

	// CountByStatus counts customers by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status CustomerStatus) (int64, error)

	// ExistsByCode checks if a customer with the given code exists in the tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// ExistsByPhone checks if a customer with the given phone exists in the tenant
	ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error)

	// ExistsByEmail checks if a customer with the given email exists in the tenant
	ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
}
