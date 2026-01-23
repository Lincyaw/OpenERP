package partner

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// WarehouseRepository defines the interface for warehouse persistence
type WarehouseRepository interface {
	// FindByID finds a warehouse by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*Warehouse, error)

	// FindByIDForTenant finds a warehouse by ID within a tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*Warehouse, error)

	// FindByCode finds a warehouse by its code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Warehouse, error)

	// FindAll finds all warehouses matching the filter
	FindAll(ctx context.Context, filter shared.Filter) ([]Warehouse, error)

	// FindAllForTenant finds all warehouses for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Warehouse, error)

	// FindByType finds warehouses by type (physical/virtual/consign/transit)
	FindByType(ctx context.Context, tenantID uuid.UUID, warehouseType WarehouseType, filter shared.Filter) ([]Warehouse, error)

	// FindByStatus finds warehouses by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status WarehouseStatus, filter shared.Filter) ([]Warehouse, error)

	// FindActive finds all active warehouses for a tenant
	FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]Warehouse, error)

	// FindDefault finds the default warehouse for a tenant
	FindDefault(ctx context.Context, tenantID uuid.UUID) (*Warehouse, error)

	// FindByIDs finds multiple warehouses by their IDs
	FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]Warehouse, error)

	// FindByCodes finds multiple warehouses by their codes
	FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]Warehouse, error)

	// Save creates or updates a warehouse
	Save(ctx context.Context, warehouse *Warehouse) error

	// SaveBatch creates or updates multiple warehouses
	SaveBatch(ctx context.Context, warehouses []*Warehouse) error

	// Delete deletes a warehouse
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a warehouse within a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// Count counts warehouses matching the filter
	Count(ctx context.Context, filter shared.Filter) (int64, error)

	// CountForTenant counts warehouses for a tenant
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByType counts warehouses by type for a tenant
	CountByType(ctx context.Context, tenantID uuid.UUID, warehouseType WarehouseType) (int64, error)

	// CountByStatus counts warehouses by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status WarehouseStatus) (int64, error)

	// ExistsByCode checks if a warehouse with the given code exists in the tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// ClearDefault clears the default flag for all warehouses in a tenant
	ClearDefault(ctx context.Context, tenantID uuid.UUID) error
}
