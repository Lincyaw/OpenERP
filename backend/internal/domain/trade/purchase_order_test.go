package trade

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers for PurchaseOrder
func createTestPurchaseOrder(t *testing.T) *PurchaseOrder {
	tenantID := uuid.New()
	supplierID := uuid.New()
	order, err := NewPurchaseOrder(tenantID, "PO-2024-001", supplierID, "Test Supplier")
	require.NoError(t, err)
	return order
}

func addTestPurchaseOrderItem(t *testing.T, order *PurchaseOrder, productName string, quantity float64, cost float64) *PurchaseOrderItem {
	productID := uuid.New()
	unitCost := valueobject.NewMoneyCNYFromFloat(cost)
	item, err := order.AddItem(productID, productName, "SKU-001", "pcs", "pcs", decimal.NewFromFloat(quantity), decimal.NewFromInt(1), unitCost)
	require.NoError(t, err)
	return item
}

// ============================================
// PurchaseOrderStatus Tests
// ============================================

func TestPurchaseOrderStatus_IsValid(t *testing.T) {
	tests := []struct {
		status  PurchaseOrderStatus
		isValid bool
	}{
		{PurchaseOrderStatusDraft, true},
		{PurchaseOrderStatusConfirmed, true},
		{PurchaseOrderStatusPartialReceived, true},
		{PurchaseOrderStatusCompleted, true},
		{PurchaseOrderStatusCancelled, true},
		{PurchaseOrderStatus("INVALID"), false},
		{PurchaseOrderStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.status.IsValid())
		})
	}
}

func TestPurchaseOrderStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from     PurchaseOrderStatus
		to       PurchaseOrderStatus
		canTrans bool
	}{
		// From DRAFT
		{PurchaseOrderStatusDraft, PurchaseOrderStatusConfirmed, true},
		{PurchaseOrderStatusDraft, PurchaseOrderStatusCancelled, true},
		{PurchaseOrderStatusDraft, PurchaseOrderStatusPartialReceived, false},
		{PurchaseOrderStatusDraft, PurchaseOrderStatusCompleted, false},
		// From CONFIRMED
		{PurchaseOrderStatusConfirmed, PurchaseOrderStatusPartialReceived, true},
		{PurchaseOrderStatusConfirmed, PurchaseOrderStatusCompleted, true},
		{PurchaseOrderStatusConfirmed, PurchaseOrderStatusCancelled, true},
		{PurchaseOrderStatusConfirmed, PurchaseOrderStatusDraft, false},
		// From PARTIAL_RECEIVED
		{PurchaseOrderStatusPartialReceived, PurchaseOrderStatusPartialReceived, true},
		{PurchaseOrderStatusPartialReceived, PurchaseOrderStatusCompleted, true},
		{PurchaseOrderStatusPartialReceived, PurchaseOrderStatusCancelled, false}, // Cannot cancel after receiving
		{PurchaseOrderStatusPartialReceived, PurchaseOrderStatusDraft, false},
		{PurchaseOrderStatusPartialReceived, PurchaseOrderStatusConfirmed, false},
		// From COMPLETED (terminal)
		{PurchaseOrderStatusCompleted, PurchaseOrderStatusDraft, false},
		{PurchaseOrderStatusCompleted, PurchaseOrderStatusConfirmed, false},
		{PurchaseOrderStatusCompleted, PurchaseOrderStatusPartialReceived, false},
		{PurchaseOrderStatusCompleted, PurchaseOrderStatusCancelled, false},
		// From CANCELLED (terminal)
		{PurchaseOrderStatusCancelled, PurchaseOrderStatusDraft, false},
		{PurchaseOrderStatusCancelled, PurchaseOrderStatusConfirmed, false},
		{PurchaseOrderStatusCancelled, PurchaseOrderStatusPartialReceived, false},
		{PurchaseOrderStatusCancelled, PurchaseOrderStatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			assert.Equal(t, tt.canTrans, tt.from.CanTransitionTo(tt.to))
		})
	}
}

func TestPurchaseOrderStatus_CanReceive(t *testing.T) {
	tests := []struct {
		status     PurchaseOrderStatus
		canReceive bool
	}{
		{PurchaseOrderStatusDraft, false},
		{PurchaseOrderStatusConfirmed, true},
		{PurchaseOrderStatusPartialReceived, true},
		{PurchaseOrderStatusCompleted, false},
		{PurchaseOrderStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.canReceive, tt.status.CanReceive())
		})
	}
}

func TestPurchaseOrderStatus_String(t *testing.T) {
	assert.Equal(t, "DRAFT", PurchaseOrderStatusDraft.String())
	assert.Equal(t, "CONFIRMED", PurchaseOrderStatusConfirmed.String())
	assert.Equal(t, "PARTIAL_RECEIVED", PurchaseOrderStatusPartialReceived.String())
}

// ============================================
// NewPurchaseOrder Tests
// ============================================

func TestNewPurchaseOrder(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()

	t.Run("creates order with valid inputs", func(t *testing.T) {
		order, err := NewPurchaseOrder(tenantID, "PO-2024-001", supplierID, "Test Supplier")
		require.NoError(t, err)
		require.NotNil(t, order)

		assert.Equal(t, tenantID, order.TenantID)
		assert.Equal(t, "PO-2024-001", order.OrderNumber)
		assert.Equal(t, supplierID, order.SupplierID)
		assert.Equal(t, "Test Supplier", order.SupplierName)
		assert.Equal(t, PurchaseOrderStatusDraft, order.Status)
		assert.Empty(t, order.Items)
		assert.True(t, order.TotalAmount.IsZero())
		assert.True(t, order.DiscountAmount.IsZero())
		assert.True(t, order.PayableAmount.IsZero())
		assert.Nil(t, order.WarehouseID)
		assert.NotEmpty(t, order.ID)
		assert.Equal(t, 1, order.GetVersion())
	})

	t.Run("publishes PurchaseOrderCreated event", func(t *testing.T) {
		order, err := NewPurchaseOrder(tenantID, "PO-2024-002", supplierID, "Test Supplier")
		require.NoError(t, err)

		events := order.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypePurchaseOrderCreated, events[0].EventType())

		event, ok := events[0].(*PurchaseOrderCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, order.ID, event.OrderID)
		assert.Equal(t, order.OrderNumber, event.OrderNumber)
		assert.Equal(t, order.SupplierID, event.SupplierID)
	})

	t.Run("fails with empty order number", func(t *testing.T) {
		_, err := NewPurchaseOrder(tenantID, "", supplierID, "Test Supplier")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Order number cannot be empty")
	})

	t.Run("fails with order number too long", func(t *testing.T) {
		longNumber := string(make([]byte, 51))
		_, err := NewPurchaseOrder(tenantID, longNumber, supplierID, "Test Supplier")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with nil supplier ID", func(t *testing.T) {
		_, err := NewPurchaseOrder(tenantID, "PO-2024-001", uuid.Nil, "Test Supplier")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Supplier ID cannot be empty")
	})

	t.Run("fails with empty supplier name", func(t *testing.T) {
		_, err := NewPurchaseOrder(tenantID, "PO-2024-001", supplierID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Supplier name cannot be empty")
	})
}

// ============================================
// PurchaseOrderItem Tests
// ============================================

func TestNewPurchaseOrderItem(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()

	t.Run("creates item with valid inputs", func(t *testing.T) {
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		item, err := NewPurchaseOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		require.NoError(t, err)
		require.NotNil(t, item)

		assert.Equal(t, orderID, item.OrderID)
		assert.Equal(t, productID, item.ProductID)
		assert.Equal(t, "Test Product", item.ProductName)
		assert.Equal(t, "SKU-001", item.ProductCode)
		assert.Equal(t, "pcs", item.Unit)
		assert.True(t, item.OrderedQuantity.Equal(decimal.NewFromFloat(10)))
		assert.True(t, item.ReceivedQuantity.IsZero())
		assert.True(t, item.UnitCost.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, item.Amount.Equal(decimal.NewFromFloat(1000.00))) // 10 * 100
	})

	t.Run("fails with nil product ID", func(t *testing.T) {
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewPurchaseOrderItem(orderID, uuid.Nil, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Product ID cannot be empty")
	})

	t.Run("fails with empty product name", func(t *testing.T) {
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewPurchaseOrderItem(orderID, productID, "", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Product name cannot be empty")
	})

	t.Run("fails with zero quantity", func(t *testing.T) {
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewPurchaseOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.Zero, decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be positive")
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewPurchaseOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(-5), decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be positive")
	})

	t.Run("fails with negative cost", func(t *testing.T) {
		unitCost := valueobject.NewMoneyCNYFromFloat(-10.00)
		_, err := NewPurchaseOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unit cost cannot be negative")
	})

	t.Run("fails with empty unit", func(t *testing.T) {
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewPurchaseOrderItem(orderID, productID, "Test Product", "SKU-001", "", "", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unit cannot be empty")
	})
}

func TestPurchaseOrderItem_UpdateQuantity(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()
	unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
	item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)

	t.Run("updates quantity and recalculates amount", func(t *testing.T) {
		err := item.UpdateQuantity(decimal.NewFromFloat(20))
		require.NoError(t, err)

		assert.True(t, item.OrderedQuantity.Equal(decimal.NewFromFloat(20)))
		assert.True(t, item.Amount.Equal(decimal.NewFromFloat(2000.00))) // 20 * 100
	})

	t.Run("fails with zero quantity", func(t *testing.T) {
		err := item.UpdateQuantity(decimal.Zero)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be positive")
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		err := item.UpdateQuantity(decimal.NewFromFloat(-5))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be positive")
	})

	t.Run("fails when new quantity is less than received quantity", func(t *testing.T) {
		item2, _ := NewPurchaseOrderItem(orderID, uuid.New(), "Test2", "SKU-002", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		item2.ReceivedQuantity = decimal.NewFromFloat(8)

		err := item2.UpdateQuantity(decimal.NewFromFloat(5)) // Less than received (8)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be less than received quantity")
	})
}

func TestPurchaseOrderItem_UpdateUnitCost(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()
	unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
	item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)

	t.Run("updates cost and recalculates amount", func(t *testing.T) {
		newCost := valueobject.NewMoneyCNYFromFloat(150.00)
		err := item.UpdateUnitCost(newCost)
		require.NoError(t, err)

		assert.True(t, item.UnitCost.Equal(decimal.NewFromFloat(150.00)))
		assert.True(t, item.Amount.Equal(decimal.NewFromFloat(1500.00))) // 10 * 150
	})

	t.Run("fails with negative cost", func(t *testing.T) {
		newCost := valueobject.NewMoneyCNYFromFloat(-10.00)
		err := item.UpdateUnitCost(newCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unit cost cannot be negative")
	})
}

func TestPurchaseOrderItem_ReceiveMethods(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()
	unitCost := valueobject.NewMoneyCNYFromFloat(100.00)

	t.Run("RemainingQuantity returns correct value", func(t *testing.T) {
		item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		assert.True(t, item.RemainingQuantity().Equal(decimal.NewFromFloat(10)))

		item.ReceivedQuantity = decimal.NewFromFloat(3)
		assert.True(t, item.RemainingQuantity().Equal(decimal.NewFromFloat(7)))
	})

	t.Run("IsFullyReceived returns correct value", func(t *testing.T) {
		item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		assert.False(t, item.IsFullyReceived())

		item.ReceivedQuantity = decimal.NewFromFloat(10)
		assert.True(t, item.IsFullyReceived())

		item.ReceivedQuantity = decimal.NewFromFloat(11) // Over-received (edge case)
		assert.True(t, item.IsFullyReceived())
	})

	t.Run("CanReceive returns correct value", func(t *testing.T) {
		item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		assert.True(t, item.CanReceive())

		item.ReceivedQuantity = decimal.NewFromFloat(10)
		assert.False(t, item.CanReceive())
	})

	t.Run("AddReceivedQuantity works correctly", func(t *testing.T) {
		item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)

		err := item.AddReceivedQuantity(decimal.NewFromFloat(5))
		require.NoError(t, err)
		assert.True(t, item.ReceivedQuantity.Equal(decimal.NewFromFloat(5)))

		err = item.AddReceivedQuantity(decimal.NewFromFloat(3))
		require.NoError(t, err)
		assert.True(t, item.ReceivedQuantity.Equal(decimal.NewFromFloat(8)))
	})

	t.Run("AddReceivedQuantity fails with zero quantity", func(t *testing.T) {
		item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)

		err := item.AddReceivedQuantity(decimal.Zero)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be positive")
	})

	t.Run("AddReceivedQuantity fails when exceeding ordered quantity", func(t *testing.T) {
		item, _ := NewPurchaseOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		item.ReceivedQuantity = decimal.NewFromFloat(8)

		err := item.AddReceivedQuantity(decimal.NewFromFloat(5)) // Would make total 13, but ordered only 10
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot receive")
	})
}

// ============================================
// AddItem Tests
// ============================================

func TestPurchaseOrder_AddItem(t *testing.T) {
	t.Run("adds item and calculates totals", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		order.ClearDomainEvents()

		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		item, err := order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		require.NoError(t, err)
		require.NotNil(t, item)

		assert.Len(t, order.Items, 1)
		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, order.PayableAmount.Equal(decimal.NewFromFloat(1000.00)))
	})

	t.Run("adds multiple items", func(t *testing.T) {
		order := createTestPurchaseOrder(t)

		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // 1000
		addTestPurchaseOrderItem(t, order, "Product 2", 5, 200.00)  // 1000

		assert.Len(t, order.Items, 2)
		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(2000.00)))
	})

	t.Run("fails to add duplicate product", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)

		_, err := order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		require.NoError(t, err)

		_, err = order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(5), decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists in order")
	})

	t.Run("fails when order is not in draft status", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)

		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(50.00)
		_, err := order.AddItem(productID, "Product 2", "SKU-002", "pcs", "pcs", decimal.NewFromFloat(5), decimal.NewFromInt(1), unitCost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-draft order")
	})
}

// ============================================
// RemoveItem Tests
// ============================================

func TestPurchaseOrder_RemoveItem(t *testing.T) {
	t.Run("removes item and recalculates totals", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		item1 := addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // 1000
		addTestPurchaseOrderItem(t, order, "Product 2", 5, 200.00)           // 1000

		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(2000.00)))

		err := order.RemoveItem(item1.ID)
		require.NoError(t, err)

		assert.Len(t, order.Items, 1)
		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
	})

	t.Run("fails with non-existent item", func(t *testing.T) {
		order := createTestPurchaseOrder(t)

		err := order.RemoveItem(uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})

	t.Run("fails when order is not in draft status", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		item := addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		err := order.RemoveItem(item.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-draft order")
	})
}

// ============================================
// Discount Tests
// ============================================

func TestPurchaseOrder_ApplyDiscount(t *testing.T) {
	t.Run("applies discount", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // 1000

		discount := valueobject.NewMoneyCNYFromFloat(100.00)
		err := order.ApplyDiscount(discount)
		require.NoError(t, err)

		assert.True(t, order.DiscountAmount.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, order.PayableAmount.Equal(decimal.NewFromFloat(900.00)))
	})

	t.Run("fails with discount exceeding total", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // 1000

		discount := valueobject.NewMoneyCNYFromFloat(1500.00)
		err := order.ApplyDiscount(discount)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed total amount")
	})

	t.Run("fails with negative discount", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)

		discount := valueobject.NewMoneyCNYFromFloat(-50.00)
		err := order.ApplyDiscount(discount)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("fails when order is not in draft status", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		discount := valueobject.NewMoneyCNYFromFloat(50.00)
		err := order.ApplyDiscount(discount)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-draft order")
	})
}

// ============================================
// Warehouse Tests
// ============================================

func TestPurchaseOrder_SetWarehouse(t *testing.T) {
	t.Run("sets warehouse in draft status", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		warehouseID := uuid.New()

		err := order.SetWarehouse(warehouseID)
		require.NoError(t, err)

		assert.NotNil(t, order.WarehouseID)
		assert.Equal(t, warehouseID, *order.WarehouseID)
	})

	t.Run("sets warehouse in confirmed status", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		warehouseID1 := uuid.New()
		order.SetWarehouse(warehouseID1)
		order.Confirm()

		warehouseID2 := uuid.New()
		err := order.SetWarehouse(warehouseID2)
		require.NoError(t, err)
		assert.Equal(t, warehouseID2, *order.WarehouseID)
	})

	t.Run("fails with nil warehouse ID", func(t *testing.T) {
		order := createTestPurchaseOrder(t)

		err := order.SetWarehouse(uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse ID cannot be empty")
	})

	t.Run("fails when order is completed", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		// Receive all goods to complete
		order.Receive([]ReceiveItem{{ProductID: order.Items[0].ProductID, Quantity: decimal.NewFromFloat(10)}})

		newWarehouseID := uuid.New()
		err := order.SetWarehouse(newWarehouseID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "current status")
	})
}

// ============================================
// Confirm Tests
// ============================================

func TestPurchaseOrder_Confirm(t *testing.T) {
	t.Run("confirms order with items", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		order.ClearDomainEvents()

		err := order.Confirm()
		require.NoError(t, err)

		assert.Equal(t, PurchaseOrderStatusConfirmed, order.Status)
		assert.NotNil(t, order.ConfirmedAt)
		assert.True(t, order.IsConfirmed())
	})

	t.Run("publishes PurchaseOrderConfirmed event", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		order.ClearDomainEvents()

		order.Confirm()

		events := order.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypePurchaseOrderConfirmed, events[0].EventType())

		event, ok := events[0].(*PurchaseOrderConfirmedEvent)
		require.True(t, ok)
		assert.Equal(t, order.ID, event.OrderID)
		assert.Len(t, event.Items, 1)
	})

	t.Run("fails without items", func(t *testing.T) {
		order := createTestPurchaseOrder(t)

		err := order.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "without items")
	})

	t.Run("fails when already confirmed", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()

		err := order.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CONFIRMED status")
	})

	t.Run("fails with zero payable amount", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // 1000
		order.ApplyDiscount(valueobject.NewMoneyCNYFromFloat(1000.00))

		err := order.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be positive")
	})
}

// ============================================
// Receive Tests
// ============================================

func TestPurchaseOrder_Receive(t *testing.T) {
	t.Run("receives partial goods", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()

		receivedInfos, err := order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(5)},
		})
		require.NoError(t, err)
		require.Len(t, receivedInfos, 1)

		assert.Equal(t, PurchaseOrderStatusPartialReceived, order.Status)
		assert.True(t, order.Items[0].ReceivedQuantity.Equal(decimal.NewFromFloat(5)))
	})

	t.Run("receives full goods and completes order", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()

		receivedInfos, err := order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(10)},
		})
		require.NoError(t, err)
		require.Len(t, receivedInfos, 1)

		assert.Equal(t, PurchaseOrderStatusCompleted, order.Status)
		assert.NotNil(t, order.CompletedAt)
		assert.True(t, order.IsCompleted())
	})

	t.Run("receives multiple items at once", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		product1ID := uuid.New()
		product2ID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(product1ID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		order.AddItem(product2ID, "Product 2", "SKU-002", "pcs", "pcs", decimal.NewFromFloat(5), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()

		receivedInfos, err := order.Receive([]ReceiveItem{
			{ProductID: product1ID, Quantity: decimal.NewFromFloat(10)},
			{ProductID: product2ID, Quantity: decimal.NewFromFloat(5)},
		})
		require.NoError(t, err)
		require.Len(t, receivedInfos, 2)

		assert.Equal(t, PurchaseOrderStatusCompleted, order.Status)
	})

	t.Run("receives in multiple batches", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		// First batch
		_, err := order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(3)},
		})
		require.NoError(t, err)
		assert.Equal(t, PurchaseOrderStatusPartialReceived, order.Status)

		// Second batch
		_, err = order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(4)},
		})
		require.NoError(t, err)
		assert.Equal(t, PurchaseOrderStatusPartialReceived, order.Status)

		// Final batch
		_, err = order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(3)},
		})
		require.NoError(t, err)
		assert.Equal(t, PurchaseOrderStatusCompleted, order.Status)
	})

	t.Run("publishes PurchaseOrderReceived event", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()

		order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(5), BatchNumber: "BATCH-001"},
		})

		events := order.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypePurchaseOrderReceived, events[0].EventType())

		event, ok := events[0].(*PurchaseOrderReceivedEvent)
		require.True(t, ok)
		assert.Len(t, event.ReceivedItems, 1)
		assert.Equal(t, "BATCH-001", event.ReceivedItems[0].BatchNumber)
		assert.False(t, event.IsFullyReceived)
	})

	t.Run("fails when not in receivable status", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)

		_, err := order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(5)},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DRAFT status")
	})

	t.Run("fails without warehouse", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		order.Confirm()

		_, err := order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(5)},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse must be set")
	})

	t.Run("fails with empty receive items", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		_, err := order.Receive([]ReceiveItem{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("fails with product not in order", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		_, err := order.Receive([]ReceiveItem{
			{ProductID: uuid.New(), Quantity: decimal.NewFromFloat(5)},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in order")
	})

	t.Run("fails when exceeding ordered quantity", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		_, err := order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(15)},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot receive")
	})

	t.Run("supports batch number and expiry date", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()

		expiryDate := time.Now().AddDate(1, 0, 0)
		receivedInfos, err := order.Receive([]ReceiveItem{
			{
				ProductID:   productID,
				Quantity:    decimal.NewFromFloat(5),
				BatchNumber: "BATCH-2024-001",
				ExpiryDate:  &expiryDate,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, "BATCH-2024-001", receivedInfos[0].BatchNumber)
		assert.NotNil(t, receivedInfos[0].ExpiryDate)
	})
}

// ============================================
// Cancel Tests
// ============================================

func TestPurchaseOrder_Cancel(t *testing.T) {
	t.Run("cancels draft order", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		order.ClearDomainEvents()

		err := order.Cancel("Supplier unavailable")
		require.NoError(t, err)

		assert.Equal(t, PurchaseOrderStatusCancelled, order.Status)
		assert.NotNil(t, order.CancelledAt)
		assert.Equal(t, "Supplier unavailable", order.CancelReason)
		assert.True(t, order.IsCancelled())
		assert.True(t, order.IsTerminal())
	})

	t.Run("cancels confirmed order with wasConfirmed flag", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()
		order.ClearDomainEvents()

		err := order.Cancel("Price changed")
		require.NoError(t, err)

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*PurchaseOrderCancelledEvent)
		require.True(t, ok)
		assert.True(t, event.WasConfirmed)
	})

	t.Run("fails without reason", func(t *testing.T) {
		order := createTestPurchaseOrder(t)

		err := order.Cancel("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cancel reason is required")
	})

	t.Run("fails after receiving goods", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.Receive([]ReceiveItem{
			{ProductID: productID, Quantity: decimal.NewFromFloat(5)},
		})

		err := order.Cancel("Too late")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PARTIAL_RECEIVED status")
	})

	t.Run("fails when already cancelled", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		order.Cancel("First time")

		err := order.Cancel("Second time")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CANCELLED status")
	})
}

// ============================================
// Helper Methods Tests
// ============================================

func TestPurchaseOrder_HelperMethods(t *testing.T) {
	t.Run("ItemCount returns correct count", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		assert.Equal(t, 0, order.ItemCount())

		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		assert.Equal(t, 1, order.ItemCount())

		addTestPurchaseOrderItem(t, order, "Product 2", 5, 50.00)
		assert.Equal(t, 2, order.ItemCount())
	})

	t.Run("TotalOrderedQuantity returns sum of ordered quantities", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		addTestPurchaseOrderItem(t, order, "Product 2", 5, 50.00)

		assert.True(t, order.TotalOrderedQuantity().Equal(decimal.NewFromFloat(15)))
	})

	t.Run("TotalReceivedQuantity returns sum of received quantities", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.Receive([]ReceiveItem{{ProductID: productID, Quantity: decimal.NewFromFloat(5)}})

		assert.True(t, order.TotalReceivedQuantity().Equal(decimal.NewFromFloat(5)))
	})

	t.Run("TotalRemainingQuantity returns sum of remaining quantities", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.Receive([]ReceiveItem{{ProductID: productID, Quantity: decimal.NewFromFloat(3)}})

		assert.True(t, order.TotalRemainingQuantity().Equal(decimal.NewFromFloat(7)))
	})

	t.Run("ReceiveProgress returns correct percentage", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		assert.True(t, order.ReceiveProgress().Equal(decimal.Zero))

		order.Receive([]ReceiveItem{{ProductID: productID, Quantity: decimal.NewFromFloat(5)}})
		assert.True(t, order.ReceiveProgress().Equal(decimal.NewFromFloat(50)))

		order.Receive([]ReceiveItem{{ProductID: productID, Quantity: decimal.NewFromFloat(5)}})
		assert.True(t, order.ReceiveProgress().Equal(decimal.NewFromFloat(100)))
	})

	t.Run("GetItem returns item by ID", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		item := addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)

		found := order.GetItem(item.ID)
		require.NotNil(t, found)
		assert.Equal(t, item.ID, found.ID)

		notFound := order.GetItem(uuid.New())
		assert.Nil(t, notFound)
	})

	t.Run("GetItemByProduct returns item by product ID", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)

		found := order.GetItemByProduct(productID)
		require.NotNil(t, found)
		assert.Equal(t, productID, found.ProductID)

		notFound := order.GetItemByProduct(uuid.New())
		assert.Nil(t, notFound)
	})

	t.Run("GetReceivableItems returns items that can receive goods", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		product1ID := uuid.New()
		product2ID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(product1ID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		order.AddItem(product2ID, "Product 2", "SKU-002", "pcs", "pcs", decimal.NewFromFloat(5), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		// Receive all of product 1
		order.Receive([]ReceiveItem{{ProductID: product1ID, Quantity: decimal.NewFromFloat(10)}})

		receivable := order.GetReceivableItems()
		assert.Len(t, receivable, 1)
		assert.Equal(t, product2ID, receivable[0].ProductID)
	})

	t.Run("CanModify returns true only for draft", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		assert.True(t, order.CanModify())

		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()
		assert.False(t, order.CanModify())
	})

	t.Run("CanReceiveGoods returns correct value", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		assert.False(t, order.CanReceiveGoods())

		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()
		assert.True(t, order.CanReceiveGoods())
	})

	t.Run("Money getters return correct values", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // 1000
		order.ApplyDiscount(valueobject.NewMoneyCNYFromFloat(100.00))

		total := order.GetTotalAmountMoney()
		assert.True(t, total.Amount().Equal(decimal.NewFromFloat(1000.00)))

		discount := order.GetDiscountAmountMoney()
		assert.True(t, discount.Amount().Equal(decimal.NewFromFloat(100.00)))

		payable := order.GetPayableAmountMoney()
		assert.True(t, payable.Amount().Equal(decimal.NewFromFloat(900.00)))
	})
}

// ============================================
// Status Helper Tests
// ============================================

func TestPurchaseOrder_StatusHelpers(t *testing.T) {
	order := createTestPurchaseOrder(t)

	assert.True(t, order.IsDraft())
	assert.False(t, order.IsConfirmed())
	assert.False(t, order.IsPartialReceived())
	assert.False(t, order.IsCompleted())
	assert.False(t, order.IsCancelled())
	assert.False(t, order.IsTerminal())

	productID := uuid.New()
	unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
	order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
	warehouseID := uuid.New()
	order.SetWarehouse(warehouseID)
	order.Confirm()

	assert.False(t, order.IsDraft())
	assert.True(t, order.IsConfirmed())
	assert.False(t, order.IsTerminal())

	order.Receive([]ReceiveItem{{ProductID: productID, Quantity: decimal.NewFromFloat(5)}})

	assert.True(t, order.IsPartialReceived())
	assert.False(t, order.IsTerminal())

	order.Receive([]ReceiveItem{{ProductID: productID, Quantity: decimal.NewFromFloat(5)}})

	assert.True(t, order.IsCompleted())
	assert.True(t, order.IsTerminal())
}

// ============================================
// Event Tests
// ============================================

func TestPurchaseOrderEvents(t *testing.T) {
	t.Run("PurchaseOrderCreatedEvent has correct fields", func(t *testing.T) {
		order := createTestPurchaseOrder(t)

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*PurchaseOrderCreatedEvent)
		require.True(t, ok)

		assert.Equal(t, order.ID, event.OrderID)
		assert.Equal(t, order.OrderNumber, event.OrderNumber)
		assert.Equal(t, order.SupplierID, event.SupplierID)
		assert.Equal(t, order.SupplierName, event.SupplierName)
		assert.Equal(t, order.TenantID, event.TenantID())
		assert.Equal(t, EventTypePurchaseOrderCreated, event.EventType())
		assert.Equal(t, AggregateTypePurchaseOrder, event.AggregateType())
	})

	t.Run("PurchaseOrderConfirmedEvent includes all items", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)
		addTestPurchaseOrderItem(t, order, "Product 2", 5, 200.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.ClearDomainEvents()
		order.Confirm()

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*PurchaseOrderConfirmedEvent)
		require.True(t, ok)

		assert.Len(t, event.Items, 2)
		assert.Equal(t, &warehouseID, event.WarehouseID)
		assert.True(t, event.TotalAmount.Equal(order.TotalAmount))
		assert.True(t, event.PayableAmount.Equal(order.PayableAmount))
	})

	t.Run("PurchaseOrderReceivedEvent has warehouse ID and received items", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		productID := uuid.New()
		unitCost := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitCost)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()
		order.Receive([]ReceiveItem{{ProductID: productID, Quantity: decimal.NewFromFloat(5), BatchNumber: "B001"}})

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*PurchaseOrderReceivedEvent)
		require.True(t, ok)

		assert.Equal(t, warehouseID, event.WarehouseID)
		assert.Len(t, event.ReceivedItems, 1)
		assert.True(t, event.ReceivedItems[0].Quantity.Equal(decimal.NewFromFloat(5)))
		assert.Equal(t, "B001", event.ReceivedItems[0].BatchNumber)
	})

	t.Run("PurchaseOrderCancelledEvent has wasConfirmed flag", func(t *testing.T) {
		// Test cancelling draft order
		draftOrder := createTestPurchaseOrder(t)
		draftOrder.ClearDomainEvents()
		draftOrder.Cancel("Draft cancel")

		events := draftOrder.GetDomainEvents()
		event1, _ := events[0].(*PurchaseOrderCancelledEvent)
		assert.False(t, event1.WasConfirmed)

		// Test cancelling confirmed order
		confirmedOrder := createTestPurchaseOrder(t)
		addTestPurchaseOrderItem(t, confirmedOrder, "Product 1", 10, 100.00)
		confirmedOrder.Confirm()
		confirmedOrder.ClearDomainEvents()
		confirmedOrder.Cancel("Confirmed cancel")

		events = confirmedOrder.GetDomainEvents()
		event2, _ := events[0].(*PurchaseOrderCancelledEvent)
		assert.True(t, event2.WasConfirmed)
	})
}

// ============================================
// Edge Cases
// ============================================

func TestPurchaseOrder_EdgeCases(t *testing.T) {
	t.Run("discount adjustment when removing items", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		item1 := addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // 1000
		addTestPurchaseOrderItem(t, order, "Product 2", 5, 100.00)           // 500
		// Total: 1500

		order.ApplyDiscount(valueobject.NewMoneyCNYFromFloat(1000.00))
		// Payable: 500

		// Remove item 1, total becomes 500
		// Discount should be adjusted to not exceed new total
		order.RemoveItem(item1.ID)

		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(500.00)))
		assert.True(t, order.DiscountAmount.Equal(decimal.NewFromFloat(500.00))) // Adjusted
		assert.True(t, order.PayableAmount.IsZero())
	})

	t.Run("SetRemark updates order", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		originalVersion := order.GetVersion()

		order.SetRemark("Test remark")

		assert.Equal(t, "Test remark", order.Remark)
		assert.Equal(t, originalVersion+1, order.GetVersion())
	})

	t.Run("PurchaseOrderItem SetRemark", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		item := addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)

		item.SetRemark("Item specific note")
		assert.Equal(t, "Item specific note", item.Remark)
	})

	t.Run("PurchaseOrderItem GetAmountMoney", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		item := addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00) // Amount: 1000

		money := item.GetAmountMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(1000.00)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})

	t.Run("PurchaseOrderItem GetUnitCostMoney", func(t *testing.T) {
		order := createTestPurchaseOrder(t)
		item := addTestPurchaseOrderItem(t, order, "Product 1", 10, 100.00)

		money := item.GetUnitCostMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(100.00)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})
}
