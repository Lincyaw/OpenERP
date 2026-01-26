// Package integration provides integration tests for order-inventory interactions.
// This file tests the critical business flow: order confirmation locks inventory,
// order shipment deducts inventory, and order cancellation releases inventory.
package integration

import (
	"context"
	"testing"
	"time"

	inventoryapp "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/application/trade"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	domaintrade "github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// OrderInventoryTestSetup provides test infrastructure for order-inventory integration tests
type OrderInventoryTestSetup struct {
	DB               *TestDB
	InventoryRepo    inventory.InventoryItemRepository
	LockRepo         inventory.StockLockRepository
	BatchRepo        inventory.StockBatchRepository
	TransactionRepo  inventory.InventoryTransactionRepository
	InventoryService *inventoryapp.InventoryService
	Logger           *zap.Logger
	TenantID         uuid.UUID
	WarehouseID      uuid.UUID
	ProductID        uuid.UUID
}

// NewOrderInventoryTestSetup creates test infrastructure with real database
func NewOrderInventoryTestSetup(t *testing.T) *OrderInventoryTestSetup {
	t.Helper()

	testDB := NewTestDB(t)

	// Create repositories
	inventoryRepo := persistence.NewGormInventoryItemRepository(testDB.DB)
	lockRepo := persistence.NewGormStockLockRepository(testDB.DB)
	batchRepo := persistence.NewGormStockBatchRepository(testDB.DB)
	transactionRepo := persistence.NewGormInventoryTransactionRepository(testDB.DB)

	// Create service
	inventoryService := inventoryapp.NewInventoryService(
		inventoryRepo,
		batchRepo,
		lockRepo,
		transactionRepo,
	)

	// Create test data
	tenantID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()

	testDB.CreateTestTenantWithUUID(tenantID)
	testDB.CreateTestWarehouse(tenantID, warehouseID)
	testDB.CreateTestProduct(tenantID, productID)

	return &OrderInventoryTestSetup{
		DB:               testDB,
		InventoryRepo:    inventoryRepo,
		LockRepo:         lockRepo,
		BatchRepo:        batchRepo,
		TransactionRepo:  transactionRepo,
		InventoryService: inventoryService,
		Logger:           zap.NewNop(),
		TenantID:         tenantID,
		WarehouseID:      warehouseID,
		ProductID:        productID,
	}
}

// CreateInventoryWithStock creates an inventory item with initial stock
func (s *OrderInventoryTestSetup) CreateInventoryWithStock(t *testing.T, quantity float64) *inventory.InventoryItem {
	t.Helper()
	ctx := context.Background()

	item, err := inventory.NewInventoryItem(s.TenantID, s.WarehouseID, s.ProductID)
	require.NoError(t, err)

	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
	err = item.IncreaseStock(decimal.NewFromFloat(quantity), unitCost, nil)
	require.NoError(t, err)

	err = s.InventoryRepo.Save(ctx, item)
	require.NoError(t, err)

	return item
}

// CreateAdditionalProduct creates an additional product for multi-item tests
func (s *OrderInventoryTestSetup) CreateAdditionalProduct(t *testing.T) uuid.UUID {
	t.Helper()
	productID := uuid.New()
	s.DB.CreateTestProduct(s.TenantID, productID)
	return productID
}

// CreateInventoryForProduct creates inventory for a specific product
func (s *OrderInventoryTestSetup) CreateInventoryForProduct(t *testing.T, productID uuid.UUID, quantity float64) *inventory.InventoryItem {
	t.Helper()
	ctx := context.Background()

	item, err := inventory.NewInventoryItem(s.TenantID, s.WarehouseID, productID)
	require.NoError(t, err)

	unitCost := valueobject.NewMoneyCNY(decimal.NewFromFloat(10.00))
	err = item.IncreaseStock(decimal.NewFromFloat(quantity), unitCost, nil)
	require.NoError(t, err)

	err = s.InventoryRepo.Save(ctx, item)
	require.NoError(t, err)

	return item
}

// ==================== Order Confirmation -> Stock Locking Tests ====================

func TestOrderInventory_ConfirmLockStock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewOrderInventoryTestSetup(t)
	ctx := context.Background()

	t.Run("confirm order locks stock", func(t *testing.T) {
		// Setup: Create inventory with 100 units
		setup.CreateInventoryWithStock(t, 100)

		// Create a confirmed event (simulating order confirmation)
		orderID := uuid.New()
		orderNumber := "SO-2024-00001"

		event := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   setup.ProductID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(30),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(3000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(3000.00),
			PayableAmount: decimal.NewFromFloat(3000.00),
		}

		// Create and execute handler
		handler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify: Stock should be locked
		item, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)

		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(70)), "Available should be 100-30=70")
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(30)), "Locked should be 30")

		// Verify: Lock record should exist
		locks, err := setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID.String())
		require.NoError(t, err)
		assert.Len(t, locks, 1)
		assert.True(t, locks[0].Quantity.Equal(decimal.NewFromInt(30)))
		assert.False(t, locks[0].Released)
		assert.False(t, locks[0].Consumed)
	})

	t.Run("confirm order with multiple items locks all stock", func(t *testing.T) {
		// Create additional products with inventory
		productID2 := setup.CreateAdditionalProduct(t)
		productID3 := setup.CreateAdditionalProduct(t)

		setup.CreateInventoryForProduct(t, productID2, 50)
		setup.CreateInventoryForProduct(t, productID3, 80)

		orderID := uuid.New()
		orderNumber := "SO-2024-00002"

		event := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID2,
					ProductName: "Product 2",
					ProductCode: "PROD-002",
					Quantity:    decimal.NewFromInt(20),
					UnitPrice:   decimal.NewFromFloat(50.00),
					Amount:      decimal.NewFromFloat(1000.00),
					Unit:        "pcs",
				},
				{
					ItemID:      uuid.New(),
					ProductID:   productID3,
					ProductName: "Product 3",
					ProductCode: "PROD-003",
					Quantity:    decimal.NewFromInt(40),
					UnitPrice:   decimal.NewFromFloat(30.00),
					Amount:      decimal.NewFromFloat(1200.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(2200.00),
			PayableAmount: decimal.NewFromFloat(2200.00),
		}

		handler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify product 2 inventory
		item2, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID2)
		require.NoError(t, err)
		assert.True(t, item2.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(30)), "Product 2 available should be 50-20=30")
		assert.True(t, item2.LockedQuantity.Amount().Equal(decimal.NewFromFloat(20)), "Product 2 locked should be 20")

		// Verify product 3 inventory
		item3, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID3)
		require.NoError(t, err)
		assert.True(t, item3.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(40)), "Product 3 available should be 80-40=40")
		assert.True(t, item3.LockedQuantity.Amount().Equal(decimal.NewFromFloat(40)), "Product 3 locked should be 40")

		// Verify 2 lock records
		locks, err := setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID.String())
		require.NoError(t, err)
		assert.Len(t, locks, 2)
	})

	t.Run("confirm order fails with insufficient stock", func(t *testing.T) {
		// Create product with only 10 units
		insufficientProductID := setup.CreateAdditionalProduct(t)
		setup.CreateInventoryForProduct(t, insufficientProductID, 10)

		orderID := uuid.New()

		event := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  "SO-2024-00003",
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   insufficientProductID,
					ProductName: "Low Stock Product",
					ProductCode: "PROD-LOW",
					Quantity:    decimal.NewFromInt(50), // More than available
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(5000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(5000.00),
			PayableAmount: decimal.NewFromFloat(5000.00),
		}

		handler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := handler.Handle(ctx, event)

		// Should fail due to insufficient stock
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lock")

		// Verify no stock was locked (inventory unchanged)
		item, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, insufficientProductID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(10)), "Available should remain 10")
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Locked should remain 0")
	})
}

// ==================== Order Shipment -> Stock Deduction Tests ====================

func TestOrderInventory_ShipDeductStock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewOrderInventoryTestSetup(t)
	ctx := context.Background()

	t.Run("ship order deducts locked stock", func(t *testing.T) {
		// Setup: Create inventory and lock stock (simulate confirmed order)
		setup.CreateInventoryWithStock(t, 100)

		orderID := uuid.New()
		orderNumber := "SO-2024-SHIP-001"

		// First, confirm order to lock stock
		confirmEvent := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   setup.ProductID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(25),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(2500.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(2500.00),
			PayableAmount: decimal.NewFromFloat(2500.00),
		}

		confirmHandler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := confirmHandler.Handle(ctx, confirmEvent)
		require.NoError(t, err)

		// Verify intermediate state: stock locked
		itemAfterLock, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)
		assert.True(t, itemAfterLock.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(75)))
		assert.True(t, itemAfterLock.LockedQuantity.Amount().Equal(decimal.NewFromFloat(25)))

		// Now ship the order
		shippedEvent := &domaintrade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderShipped,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   setup.ProductID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(25),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(2500.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(2500.00),
			PayableAmount: decimal.NewFromFloat(2500.00),
		}

		shippedHandler := trade.NewSalesOrderShippedHandler(setup.InventoryService, setup.Logger)
		err = shippedHandler.Handle(ctx, shippedEvent)
		require.NoError(t, err)

		// Verify: Locked stock should be deducted (consumed)
		itemAfterShip, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)

		assert.True(t, itemAfterShip.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(75)), "Available should remain 75")
		assert.True(t, itemAfterShip.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Locked should be 0 after deduction")

		// Verify lock is marked as consumed
		locks, err := setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID.String())
		require.NoError(t, err)
		assert.Len(t, locks, 1)
		assert.True(t, locks[0].Consumed, "Lock should be marked as consumed")
	})

	t.Run("ship order with multiple items deducts all", func(t *testing.T) {
		// Create products with inventory
		productID2 := setup.CreateAdditionalProduct(t)
		productID3 := setup.CreateAdditionalProduct(t)

		setup.CreateInventoryForProduct(t, productID2, 60)
		setup.CreateInventoryForProduct(t, productID3, 90)

		orderID := uuid.New()
		orderNumber := "SO-2024-SHIP-002"

		// Confirm order first
		confirmEvent := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID2,
					ProductName: "Product 2",
					ProductCode: "PROD-002",
					Quantity:    decimal.NewFromInt(15),
					UnitPrice:   decimal.NewFromFloat(50.00),
					Amount:      decimal.NewFromFloat(750.00),
					Unit:        "pcs",
				},
				{
					ItemID:      uuid.New(),
					ProductID:   productID3,
					ProductName: "Product 3",
					ProductCode: "PROD-003",
					Quantity:    decimal.NewFromInt(35),
					UnitPrice:   decimal.NewFromFloat(30.00),
					Amount:      decimal.NewFromFloat(1050.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(1800.00),
			PayableAmount: decimal.NewFromFloat(1800.00),
		}

		confirmHandler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := confirmHandler.Handle(ctx, confirmEvent)
		require.NoError(t, err)

		// Ship order
		shippedEvent := &domaintrade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderShipped,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID2,
					ProductName: "Product 2",
					ProductCode: "PROD-002",
					Quantity:    decimal.NewFromInt(15),
					UnitPrice:   decimal.NewFromFloat(50.00),
					Amount:      decimal.NewFromFloat(750.00),
					Unit:        "pcs",
				},
				{
					ItemID:      uuid.New(),
					ProductID:   productID3,
					ProductName: "Product 3",
					ProductCode: "PROD-003",
					Quantity:    decimal.NewFromInt(35),
					UnitPrice:   decimal.NewFromFloat(30.00),
					Amount:      decimal.NewFromFloat(1050.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(1800.00),
			PayableAmount: decimal.NewFromFloat(1800.00),
		}

		shippedHandler := trade.NewSalesOrderShippedHandler(setup.InventoryService, setup.Logger)
		err = shippedHandler.Handle(ctx, shippedEvent)
		require.NoError(t, err)

		// Verify both products have stock deducted
		item2, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID2)
		require.NoError(t, err)
		assert.True(t, item2.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(45)), "Product 2: Available should be 60-15=45")
		assert.True(t, item2.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Product 2: Locked should be 0")

		item3, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID3)
		require.NoError(t, err)
		assert.True(t, item3.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(55)), "Product 3: Available should be 90-35=55")
		assert.True(t, item3.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Product 3: Locked should be 0")
	})
}

// ==================== Order Cancellation -> Stock Release Tests ====================

func TestOrderInventory_CancelReleaseStock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewOrderInventoryTestSetup(t)
	ctx := context.Background()

	t.Run("cancel confirmed order releases locked stock", func(t *testing.T) {
		// Setup: Create inventory and lock stock
		setup.CreateInventoryWithStock(t, 100)

		orderID := uuid.New()
		orderNumber := "SO-2024-CANCEL-001"

		// First, confirm order to lock stock
		confirmEvent := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   setup.ProductID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(40),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(4000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(4000.00),
			PayableAmount: decimal.NewFromFloat(4000.00),
		}

		confirmHandler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := confirmHandler.Handle(ctx, confirmEvent)
		require.NoError(t, err)

		// Verify stock is locked
		itemAfterLock, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)
		assert.True(t, itemAfterLock.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(60)))
		assert.True(t, itemAfterLock.LockedQuantity.Amount().Equal(decimal.NewFromFloat(40)))

		// Cancel the order
		cancelEvent := &domaintrade.SalesOrderCancelledEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderCancelled,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:     orderID,
			OrderNumber: orderNumber,
			CustomerID:  uuid.New(),
			WarehouseID: &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   setup.ProductID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(40),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(4000.00),
					Unit:        "pcs",
				},
			},
			CancelReason: "Customer cancelled",
			WasConfirmed: true, // Important: must be true to release locks
		}

		cancelHandler := trade.NewSalesOrderCancelledHandler(setup.InventoryService, setup.Logger)
		err = cancelHandler.Handle(ctx, cancelEvent)
		require.NoError(t, err)

		// Verify: Locked stock should be released back to available
		itemAfterCancel, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)

		assert.True(t, itemAfterCancel.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(100)), "Available should be restored to 100")
		assert.True(t, itemAfterCancel.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Locked should be 0 after release")

		// Verify lock is marked as released
		locks, err := setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID.String())
		require.NoError(t, err)
		assert.Len(t, locks, 1)
		assert.True(t, locks[0].Released, "Lock should be marked as released")
	})

	t.Run("cancel draft order does not affect inventory", func(t *testing.T) {
		// Create fresh inventory
		productID := setup.CreateAdditionalProduct(t)
		setup.CreateInventoryForProduct(t, productID, 50)

		orderID := uuid.New()

		// Cancel without prior confirmation (draft order)
		cancelEvent := &domaintrade.SalesOrderCancelledEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderCancelled,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:     orderID,
			OrderNumber: "SO-2024-CANCEL-002",
			CustomerID:  uuid.New(),
			WarehouseID: nil, // Draft orders typically don't have warehouse set
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(20),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(2000.00),
					Unit:        "pcs",
				},
			},
			CancelReason: "Customer cancelled",
			WasConfirmed: false, // Draft order - not confirmed
		}

		cancelHandler := trade.NewSalesOrderCancelledHandler(setup.InventoryService, setup.Logger)
		err := cancelHandler.Handle(ctx, cancelEvent)
		require.NoError(t, err) // Should succeed without touching inventory

		// Verify inventory unchanged
		item, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(50)), "Available should remain 50")
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Locked should remain 0")
	})

	t.Run("cancel order with multiple items releases all locks", func(t *testing.T) {
		// Create products with inventory
		productID2 := setup.CreateAdditionalProduct(t)
		productID3 := setup.CreateAdditionalProduct(t)

		setup.CreateInventoryForProduct(t, productID2, 70)
		setup.CreateInventoryForProduct(t, productID3, 100)

		orderID := uuid.New()
		orderNumber := "SO-2024-CANCEL-003"

		// Confirm order first
		confirmEvent := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID2,
					ProductName: "Product 2",
					ProductCode: "PROD-002",
					Quantity:    decimal.NewFromInt(30),
					UnitPrice:   decimal.NewFromFloat(50.00),
					Amount:      decimal.NewFromFloat(1500.00),
					Unit:        "pcs",
				},
				{
					ItemID:      uuid.New(),
					ProductID:   productID3,
					ProductName: "Product 3",
					ProductCode: "PROD-003",
					Quantity:    decimal.NewFromInt(50),
					UnitPrice:   decimal.NewFromFloat(30.00),
					Amount:      decimal.NewFromFloat(1500.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(3000.00),
			PayableAmount: decimal.NewFromFloat(3000.00),
		}

		confirmHandler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := confirmHandler.Handle(ctx, confirmEvent)
		require.NoError(t, err)

		// Cancel order
		cancelEvent := &domaintrade.SalesOrderCancelledEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderCancelled,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:     orderID,
			OrderNumber: orderNumber,
			CustomerID:  uuid.New(),
			WarehouseID: &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID2,
					ProductName: "Product 2",
					ProductCode: "PROD-002",
					Quantity:    decimal.NewFromInt(30),
					UnitPrice:   decimal.NewFromFloat(50.00),
					Amount:      decimal.NewFromFloat(1500.00),
					Unit:        "pcs",
				},
				{
					ItemID:      uuid.New(),
					ProductID:   productID3,
					ProductName: "Product 3",
					ProductCode: "PROD-003",
					Quantity:    decimal.NewFromInt(50),
					UnitPrice:   decimal.NewFromFloat(30.00),
					Amount:      decimal.NewFromFloat(1500.00),
					Unit:        "pcs",
				},
			},
			CancelReason: "Stock issue",
			WasConfirmed: true,
		}

		cancelHandler := trade.NewSalesOrderCancelledHandler(setup.InventoryService, setup.Logger)
		err = cancelHandler.Handle(ctx, cancelEvent)
		require.NoError(t, err)

		// Verify both products have stock released
		item2, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID2)
		require.NoError(t, err)
		assert.True(t, item2.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(70)), "Product 2: Available should be restored to 70")
		assert.True(t, item2.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Product 2: Locked should be 0")

		item3, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID3)
		require.NoError(t, err)
		assert.True(t, item3.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(100)), "Product 3: Available should be restored to 100")
		assert.True(t, item3.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "Product 3: Locked should be 0")
	})
}

// ==================== End-to-End Flow Tests ====================

func TestOrderInventory_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewOrderInventoryTestSetup(t)
	ctx := context.Background()

	t.Run("complete order flow: draft -> confirm -> ship", func(t *testing.T) {
		// Create inventory with 200 units
		setup.CreateInventoryWithStock(t, 200)

		orderID := uuid.New()
		orderNumber := "SO-2024-FULL-001"

		// Initial state: 200 available, 0 locked
		item, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(200)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)))

		// Step 1: Confirm order - locks 50 units
		confirmEvent := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Full Flow Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   setup.ProductID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(50),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(5000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(5000.00),
			PayableAmount: decimal.NewFromFloat(5000.00),
		}

		confirmHandler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err = confirmHandler.Handle(ctx, confirmEvent)
		require.NoError(t, err)

		// State after confirm: 150 available, 50 locked
		item, err = setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(150)), "After confirm: Available=150")
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(50)), "After confirm: Locked=50")

		// Step 2: Ship order - deducts 50 locked units
		shippedEvent := &domaintrade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderShipped,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   uuid.New(),
			CustomerName: "Full Flow Customer",
			WarehouseID:  setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   setup.ProductID,
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(50),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(5000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(5000.00),
			PayableAmount: decimal.NewFromFloat(5000.00),
		}

		shippedHandler := trade.NewSalesOrderShippedHandler(setup.InventoryService, setup.Logger)
		err = shippedHandler.Handle(ctx, shippedEvent)
		require.NoError(t, err)

		// Final state: 150 available, 0 locked (50 consumed)
		item, err = setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, setup.ProductID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(150)), "After ship: Available=150")
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(0)), "After ship: Locked=0")
	})

	t.Run("concurrent orders lock different stock", func(t *testing.T) {
		// Create fresh inventory with 100 units
		productID := setup.CreateAdditionalProduct(t)
		setup.CreateInventoryForProduct(t, productID, 100)

		// First order: 30 units
		orderID1 := uuid.New()
		confirmEvent1 := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID1,
				setup.TenantID,
			),
			OrderID:      orderID1,
			OrderNumber:  "SO-2024-CONCURRENT-001",
			CustomerID:   uuid.New(),
			CustomerName: "Customer 1",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID,
					ProductName: "Concurrent Product",
					ProductCode: "PROD-CONC",
					Quantity:    decimal.NewFromInt(30),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(3000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(3000.00),
			PayableAmount: decimal.NewFromFloat(3000.00),
		}

		confirmHandler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := confirmHandler.Handle(ctx, confirmEvent1)
		require.NoError(t, err)

		// After order 1: 70 available, 30 locked
		item, err := setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(70)))
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(30)))

		// Second order: 40 units
		orderID2 := uuid.New()
		confirmEvent2 := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID2,
				setup.TenantID,
			),
			OrderID:      orderID2,
			OrderNumber:  "SO-2024-CONCURRENT-002",
			CustomerID:   uuid.New(),
			CustomerName: "Customer 2",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   productID,
					ProductName: "Concurrent Product",
					ProductCode: "PROD-CONC",
					Quantity:    decimal.NewFromInt(40),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(4000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(4000.00),
			PayableAmount: decimal.NewFromFloat(4000.00),
		}

		err = confirmHandler.Handle(ctx, confirmEvent2)
		require.NoError(t, err)

		// After order 2: 30 available, 70 locked
		item, err = setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(30)), "After 2 orders: Available=30")
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(70)), "After 2 orders: Locked=70")

		// Verify both orders have separate locks
		locks1, err := setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID1.String())
		require.NoError(t, err)
		assert.Len(t, locks1, 1)

		locks2, err := setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID2.String())
		require.NoError(t, err)
		assert.Len(t, locks2, 1)

		// Ship order 1 - only that lock consumed
		shippedEvent1 := &domaintrade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderShipped,
				domaintrade.AggregateTypeSalesOrder,
				orderID1,
				setup.TenantID,
			),
			OrderID:     orderID1,
			OrderNumber: "SO-2024-CONCURRENT-001",
			CustomerID:  uuid.New(),
			WarehouseID: setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:    uuid.New(),
					ProductID: productID,
					Quantity:  decimal.NewFromInt(30),
				},
			},
		}

		shippedHandler := trade.NewSalesOrderShippedHandler(setup.InventoryService, setup.Logger)
		err = shippedHandler.Handle(ctx, shippedEvent1)
		require.NoError(t, err)

		// After shipping order 1: 30 available, 40 locked (order 2's lock)
		item, err = setup.InventoryRepo.FindByWarehouseAndProduct(ctx, setup.TenantID, setup.WarehouseID, productID)
		require.NoError(t, err)
		assert.True(t, item.AvailableQuantity.Amount().Equal(decimal.NewFromFloat(30)), "After ship order1: Available=30")
		assert.True(t, item.LockedQuantity.Amount().Equal(decimal.NewFromFloat(40)), "After ship order1: Locked=40 (order2)")

		// Verify order 1's lock is consumed, order 2's lock is still active
		locks1, err = setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID1.String())
		require.NoError(t, err)
		assert.True(t, locks1[0].Consumed, "Order 1 lock should be consumed")

		locks2, err = setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID2.String())
		require.NoError(t, err)
		assert.False(t, locks2[0].Consumed, "Order 2 lock should NOT be consumed")
		assert.False(t, locks2[0].Released, "Order 2 lock should NOT be released")
	})
}

// ==================== Stock Lock Expiry Tests ====================

func TestOrderInventory_LockExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewOrderInventoryTestSetup(t)
	ctx := context.Background()

	t.Run("lock has default expiry time", func(t *testing.T) {
		setup.CreateInventoryWithStock(t, 100)

		orderID := uuid.New()

		confirmEvent := &domaintrade.SalesOrderConfirmedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				domaintrade.EventTypeSalesOrderConfirmed,
				domaintrade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  "SO-2024-EXPIRY-001",
			CustomerID:   uuid.New(),
			CustomerName: "Test Customer",
			WarehouseID:  &setup.WarehouseID,
			Items: []domaintrade.SalesOrderItemInfo{
				{
					ItemID:    uuid.New(),
					ProductID: setup.ProductID,
					Quantity:  decimal.NewFromInt(20),
				},
			},
		}

		confirmHandler := trade.NewSalesOrderConfirmedHandler(setup.InventoryService, setup.Logger)
		err := confirmHandler.Handle(ctx, confirmEvent)
		require.NoError(t, err)

		// Verify lock has expiry time set (default is 30 minutes)
		locks, err := setup.InventoryService.GetLocksBySource(ctx, "SALES_ORDER", orderID.String())
		require.NoError(t, err)
		assert.Len(t, locks, 1)

		// Lock should expire in approximately 30 minutes
		expectedExpiry := time.Now().Add(30 * time.Minute)
		timeDiff := locks[0].ExpireAt.Sub(expectedExpiry)
		assert.True(t, timeDiff > -5*time.Second && timeDiff < 5*time.Second,
			"Lock expiry should be approximately 30 minutes from now, got diff: %v", timeDiff)
	})
}
