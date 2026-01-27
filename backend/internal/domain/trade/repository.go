package trade

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SalesOrderRepository defines the interface for sales order persistence
type SalesOrderRepository interface {
	// FindByID finds a sales order by ID
	FindByID(ctx context.Context, id uuid.UUID) (*SalesOrder, error)

	// SaveWithLockAndEvents saves with optimistic locking and persists domain events atomically
	// This implements the transactional outbox pattern - events are saved to the outbox table
	// in the same transaction as the aggregate, ensuring guaranteed event delivery
	SaveWithLockAndEvents(ctx context.Context, order *SalesOrder, events []shared.DomainEvent) error

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

	// CountIncompleteByCustomer counts incomplete (not COMPLETED or CANCELLED) orders for a customer
	// Used for validation before customer deletion
	CountIncompleteByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// ExistsByOrderNumber checks if an order number exists for a tenant
	ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error)

	// GenerateOrderNumber generates a unique order number for a tenant
	GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// PurchaseOrderRepository defines the interface for purchase order persistence
type PurchaseOrderRepository interface {
	// FindByID finds a purchase order by ID
	FindByID(ctx context.Context, id uuid.UUID) (*PurchaseOrder, error)

	// FindByIDForTenant finds a purchase order by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*PurchaseOrder, error)

	// FindByOrderNumber finds a purchase order by order number for a tenant
	FindByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*PurchaseOrder, error)

	// FindAllForTenant finds all purchase orders for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]PurchaseOrder, error)

	// FindBySupplier finds purchase orders for a supplier
	FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter shared.Filter) ([]PurchaseOrder, error)

	// FindByStatus finds purchase orders by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status PurchaseOrderStatus, filter shared.Filter) ([]PurchaseOrder, error)

	// FindByWarehouse finds purchase orders for a warehouse
	FindByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter shared.Filter) ([]PurchaseOrder, error)

	// FindPendingReceipt finds purchase orders pending receipt (CONFIRMED or PARTIAL_RECEIVED)
	FindPendingReceipt(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]PurchaseOrder, error)

	// Save creates or updates a purchase order
	Save(ctx context.Context, order *PurchaseOrder) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, order *PurchaseOrder) error

	// SaveWithLockAndEvents saves with optimistic locking and persists domain events atomically
	// This implements the transactional outbox pattern - events are saved to the outbox table
	// in the same transaction as the aggregate, ensuring guaranteed event delivery
	SaveWithLockAndEvents(ctx context.Context, order *PurchaseOrder, events []shared.DomainEvent) error

	// Delete deletes a purchase order (soft delete)
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a purchase order for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts purchase orders for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByStatus counts purchase orders by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status PurchaseOrderStatus) (int64, error)

	// CountBySupplier counts purchase orders for a supplier
	CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error)

	// CountPendingReceipt counts orders pending receipt for a tenant
	CountPendingReceipt(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// ExistsByOrderNumber checks if an order number exists for a tenant
	ExistsByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (bool, error)

	// GenerateOrderNumber generates a unique order number for a tenant
	GenerateOrderNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// SalesReturnRepository defines the interface for sales return persistence
type SalesReturnRepository interface {
	// FindByID finds a sales return by ID
	FindByID(ctx context.Context, id uuid.UUID) (*SalesReturn, error)

	// FindByIDForTenant finds a sales return by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*SalesReturn, error)

	// FindByReturnNumber finds a sales return by return number for a tenant
	FindByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*SalesReturn, error)

	// FindAllForTenant finds all sales returns for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]SalesReturn, error)

	// FindByCustomer finds sales returns for a customer
	FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter shared.Filter) ([]SalesReturn, error)

	// FindBySalesOrder finds sales returns for a sales order
	FindBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) ([]SalesReturn, error)

	// FindByStatus finds sales returns by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status ReturnStatus, filter shared.Filter) ([]SalesReturn, error)

	// FindPendingApproval finds sales returns pending approval
	FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]SalesReturn, error)

	// Save creates or updates a sales return
	Save(ctx context.Context, sr *SalesReturn) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, sr *SalesReturn) error

	// Delete deletes a sales return (soft delete)
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a sales return for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts sales returns for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByStatus counts sales returns by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status ReturnStatus) (int64, error)

	// CountByCustomer counts sales returns for a customer
	CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// CountBySalesOrder counts sales returns for a sales order
	CountBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) (int64, error)

	// CountPendingApproval counts returns pending approval for a tenant
	CountPendingApproval(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// ExistsByReturnNumber checks if a return number exists for a tenant
	ExistsByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (bool, error)

	// GenerateReturnNumber generates a unique return number for a tenant
	GenerateReturnNumber(ctx context.Context, tenantID uuid.UUID) (string, error)

	// GetReturnedQuantityByOrderItem returns the total quantity already returned for a specific sales order item
	// This only counts returns that are not cancelled or rejected (i.e., active returns)
	// Used to validate that total returned quantity does not exceed original order quantity
	GetReturnedQuantityByOrderItem(ctx context.Context, tenantID, salesOrderItemID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error)

	// GetReturnedQuantityByOrderItems returns the total quantity already returned for multiple sales order items
	// This is an optimized batch version for validating multiple items at once
	// Statuses excluded: CANCELLED, REJECTED
	GetReturnedQuantityByOrderItems(ctx context.Context, tenantID uuid.UUID, salesOrderItemIDs []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error)
}

// PurchaseReturnRepository defines the interface for purchase return persistence
type PurchaseReturnRepository interface {
	// FindByID finds a purchase return by ID
	FindByID(ctx context.Context, id uuid.UUID) (*PurchaseReturn, error)

	// FindByIDForTenant finds a purchase return by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*PurchaseReturn, error)

	// FindByReturnNumber finds a purchase return by return number for a tenant
	FindByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*PurchaseReturn, error)

	// FindAllForTenant finds all purchase returns for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]PurchaseReturn, error)

	// FindBySupplier finds purchase returns for a supplier
	FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter shared.Filter) ([]PurchaseReturn, error)

	// FindByPurchaseOrder finds purchase returns for a purchase order
	FindByPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID uuid.UUID) ([]PurchaseReturn, error)

	// FindByStatus finds purchase returns by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status PurchaseReturnStatus, filter shared.Filter) ([]PurchaseReturn, error)

	// FindPendingApproval finds purchase returns pending approval
	FindPendingApproval(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]PurchaseReturn, error)

	// Save creates or updates a purchase return
	Save(ctx context.Context, pr *PurchaseReturn) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, pr *PurchaseReturn) error

	// Delete deletes a purchase return (soft delete)
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant deletes a purchase return for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts purchase returns for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error)

	// CountByStatus counts purchase returns by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status PurchaseReturnStatus) (int64, error)

	// CountBySupplier counts purchase returns for a supplier
	CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error)

	// CountByPurchaseOrder counts purchase returns for a purchase order
	CountByPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID uuid.UUID) (int64, error)

	// CountPendingApproval counts returns pending approval for a tenant
	CountPendingApproval(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// ExistsByReturnNumber checks if a return number exists for a tenant
	ExistsByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (bool, error)

	// GenerateReturnNumber generates a unique return number for a tenant
	GenerateReturnNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}
