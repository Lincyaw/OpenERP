package partner

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// SupplierRepository defines the interface for supplier persistence
type SupplierRepository interface {
	// FindByID finds a supplier by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*Supplier, error)

	// FindByIDForTenant finds a supplier by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*Supplier, error)

	// FindByCode finds a supplier by its code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Supplier, error)

	// FindByPhone finds a supplier by phone number within a tenant
	FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*Supplier, error)

	// FindByEmail finds a supplier by email within a tenant
	FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*Supplier, error)

	// FindAll finds all suppliers matching the filter
	FindAll(ctx context.Context, filter shared.Filter) ([]Supplier, error)

	// FindAllForTenant finds all suppliers for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Supplier, error)

	// FindByType finds suppliers by type (manufacturer/distributor/retailer/service)
	FindByType(ctx context.Context, tenantID uuid.UUID, supplierType SupplierType, filter shared.Filter) ([]Supplier, error)

	// FindByStatus finds suppliers by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status SupplierStatus, filter shared.Filter) ([]Supplier, error)

	// FindActive finds all active suppliers for a tenant
	FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Supplier, error)

	// FindByIDs finds multiple suppliers by their IDs
	FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]Supplier, error)

	// FindByCodes finds multiple suppliers by their codes
	FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]Supplier, error)

	// FindWithOutstandingBalance finds suppliers with accounts payable balance > 0
	FindWithOutstandingBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Supplier, error)

	// FindOverCreditLimit finds suppliers whose balance exceeds their credit limit
	FindOverCreditLimit(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Supplier, error)

	// Save creates or updates a supplier
	Save(ctx context.Context, supplier *Supplier) error

	// SaveBatch creates or updates multiple suppliers
	SaveBatch(ctx context.Context, suppliers []*Supplier) error

	// Delete deletes a supplier
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a supplier within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// Count counts suppliers matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountForTenant counts suppliers for a tenant
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByType counts suppliers by type for a tenant
	CountByType(ctx context.Context, tenantID uuid.UUID, supplierType SupplierType) (int64, error)

	// CountByStatus counts suppliers by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status SupplierStatus) (int64, error)

	// ExistsByCode checks if a supplier with the given code exists in the tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// ExistsByPhone checks if a supplier with the given phone exists in the tenant
	ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error)

	// ExistsByEmail checks if a supplier with the given email exists in the tenant
	ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
}
