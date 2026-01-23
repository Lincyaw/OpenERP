package trade

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// SalesOrderRepository defines the interface for sales order persistence
type SalesOrderRepository interface {
	// FindByID finds a sales order by ID
	FindByID(ctx context.Context, id uuid.UUID) (*SalesOrder, error)

	// FindByIDForTenant finds a sales order by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*SalesOrder, error)

	// FindByOrderNumber finds a sales order by order number for a tenant
	FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*SalesOrder, error)

	// FindAllForTenant finds all sales orders for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]SalesOrder, error)

	// FindByCustomer finds sales orders for a customer
	FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]SalesOrder, error)

	// FindByStatus finds sales orders by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status OrderStatus, filter shared.Filter) ([]SalesOrder, error)

	// FindByWarehouse finds sales orders for a warehouse
	FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]SalesOrder, error)

	// Save creates or updates a sales order
	Save(ctx context.Context, order *SalesOrder) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, order *SalesOrder) error

	// Delete deletes a sales order (soft delete)
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a sales order for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts sales orders for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByStatus counts sales orders by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status OrderStatus) (int64, error)

	// CountByCustomer counts sales orders for a customer
	CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// ExistsByOrderNumber checks if an order number exists for a tenant
	ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error)

	// GenerateOrderNumber generates a unique order number for a tenant
	GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}
