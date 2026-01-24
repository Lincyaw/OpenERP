package trade

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func createTestOrder(t *testing.T) *SalesOrder {
	tenantID := uuid.New()
	customerID := uuid.New()
	order, err := NewSalesOrder(tenantID, "SO-2024-001", customerID, "Test Customer")
	require.NoError(t, err)
	return order
}

func addTestItem(t *testing.T, order *SalesOrder, productName string, quantity float64, price float64) *SalesOrderItem {
	productID := uuid.New()
	unitPrice := valueobject.NewMoneyCNYFromFloat(price)
	item, err := order.AddItem(productID, productName, "SKU-001", "pcs", "pcs", decimal.NewFromFloat(quantity), decimal.NewFromInt(1), unitPrice)
	require.NoError(t, err)
	return item
}

// ============================================
// OrderStatus Tests
// ============================================

func TestOrderStatus_IsValid(t *testing.T) {
	tests := []struct {
		status  OrderStatus
		isValid bool
	}{
		{OrderStatusDraft, true},
		{OrderStatusConfirmed, true},
		{OrderStatusShipped, true},
		{OrderStatusCompleted, true},
		{OrderStatusCancelled, true},
		{OrderStatus("INVALID"), false},
		{OrderStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.status.IsValid())
		})
	}
}

func TestOrderStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from     OrderStatus
		to       OrderStatus
		canTrans bool
	}{
		// From DRAFT
		{OrderStatusDraft, OrderStatusConfirmed, true},
		{OrderStatusDraft, OrderStatusCancelled, true},
		{OrderStatusDraft, OrderStatusShipped, false},
		{OrderStatusDraft, OrderStatusCompleted, false},
		// From CONFIRMED
		{OrderStatusConfirmed, OrderStatusShipped, true},
		{OrderStatusConfirmed, OrderStatusCancelled, true},
		{OrderStatusConfirmed, OrderStatusDraft, false},
		{OrderStatusConfirmed, OrderStatusCompleted, false},
		// From SHIPPED
		{OrderStatusShipped, OrderStatusCompleted, true},
		{OrderStatusShipped, OrderStatusCancelled, false},
		{OrderStatusShipped, OrderStatusDraft, false},
		{OrderStatusShipped, OrderStatusConfirmed, false},
		// From COMPLETED (terminal)
		{OrderStatusCompleted, OrderStatusDraft, false},
		{OrderStatusCompleted, OrderStatusConfirmed, false},
		{OrderStatusCompleted, OrderStatusShipped, false},
		{OrderStatusCompleted, OrderStatusCancelled, false},
		// From CANCELLED (terminal)
		{OrderStatusCancelled, OrderStatusDraft, false},
		{OrderStatusCancelled, OrderStatusConfirmed, false},
		{OrderStatusCancelled, OrderStatusShipped, false},
		{OrderStatusCancelled, OrderStatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			assert.Equal(t, tt.canTrans, tt.from.CanTransitionTo(tt.to))
		})
	}
}

func TestOrderStatus_String(t *testing.T) {
	assert.Equal(t, "DRAFT", OrderStatusDraft.String())
	assert.Equal(t, "CONFIRMED", OrderStatusConfirmed.String())
}

// ============================================
// NewSalesOrder Tests
// ============================================

func TestNewSalesOrder(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("creates order with valid inputs", func(t *testing.T) {
		order, err := NewSalesOrder(tenantID, "SO-2024-001", customerID, "Test Customer")
		require.NoError(t, err)
		require.NotNil(t, order)

		assert.Equal(t, tenantID, order.TenantID)
		assert.Equal(t, "SO-2024-001", order.OrderNumber)
		assert.Equal(t, customerID, order.CustomerID)
		assert.Equal(t, "Test Customer", order.CustomerName)
		assert.Equal(t, OrderStatusDraft, order.Status)
		assert.Empty(t, order.Items)
		assert.True(t, order.TotalAmount.IsZero())
		assert.True(t, order.DiscountAmount.IsZero())
		assert.True(t, order.PayableAmount.IsZero())
		assert.Nil(t, order.WarehouseID)
		assert.NotEmpty(t, order.ID)
		assert.Equal(t, 1, order.GetVersion())
	})

	t.Run("publishes SalesOrderCreated event", func(t *testing.T) {
		order, err := NewSalesOrder(tenantID, "SO-2024-002", customerID, "Test Customer")
		require.NoError(t, err)

		events := order.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeSalesOrderCreated, events[0].EventType())

		event, ok := events[0].(*SalesOrderCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, order.ID, event.OrderID)
		assert.Equal(t, order.OrderNumber, event.OrderNumber)
		assert.Equal(t, order.CustomerID, event.CustomerID)
	})

	t.Run("fails with empty order number", func(t *testing.T) {
		_, err := NewSalesOrder(tenantID, "", customerID, "Test Customer")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Order number cannot be empty")
	})

	t.Run("fails with order number too long", func(t *testing.T) {
		longNumber := string(make([]byte, 51))
		_, err := NewSalesOrder(tenantID, longNumber, customerID, "Test Customer")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with nil customer ID", func(t *testing.T) {
		_, err := NewSalesOrder(tenantID, "SO-2024-001", uuid.Nil, "Test Customer")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Customer ID cannot be empty")
	})

	t.Run("fails with empty customer name", func(t *testing.T) {
		_, err := NewSalesOrder(tenantID, "SO-2024-001", customerID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Customer name cannot be empty")
	})
}

// ============================================
// SalesOrderItem Tests
// ============================================

func TestNewSalesOrderItem(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()

	t.Run("creates item with valid inputs", func(t *testing.T) {
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		item, err := NewSalesOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)
		require.NoError(t, err)
		require.NotNil(t, item)

		assert.Equal(t, orderID, item.OrderID)
		assert.Equal(t, productID, item.ProductID)
		assert.Equal(t, "Test Product", item.ProductName)
		assert.Equal(t, "SKU-001", item.ProductCode)
		assert.Equal(t, "pcs", item.Unit)
		assert.True(t, item.Quantity.Equal(decimal.NewFromFloat(10)))
		assert.True(t, item.UnitPrice.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, item.Amount.Equal(decimal.NewFromFloat(1000.00))) // 10 * 100
	})

	t.Run("fails with nil product ID", func(t *testing.T) {
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewSalesOrderItem(orderID, uuid.Nil, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Product ID cannot be empty")
	})

	t.Run("fails with empty product name", func(t *testing.T) {
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewSalesOrderItem(orderID, productID, "", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Product name cannot be empty")
	})

	t.Run("fails with zero quantity", func(t *testing.T) {
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewSalesOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.Zero, decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be positive")
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewSalesOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(-5), decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Quantity must be positive")
	})

	t.Run("fails with negative price", func(t *testing.T) {
		unitPrice := valueobject.NewMoneyCNYFromFloat(-10.00)
		_, err := NewSalesOrderItem(orderID, productID, "Test Product", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unit price cannot be negative")
	})

	t.Run("fails with empty unit", func(t *testing.T) {
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		_, err := NewSalesOrderItem(orderID, productID, "Test Product", "SKU-001", "", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unit cannot be empty")
	})
}

func TestSalesOrderItem_UpdateQuantity(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()
	unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
	item, _ := NewSalesOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)

	t.Run("updates quantity and recalculates amount", func(t *testing.T) {
		err := item.UpdateQuantity(decimal.NewFromFloat(20))
		require.NoError(t, err)

		assert.True(t, item.Quantity.Equal(decimal.NewFromFloat(20)))
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
}

func TestSalesOrderItem_UpdateUnitPrice(t *testing.T) {
	orderID := uuid.New()
	productID := uuid.New()
	unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
	item, _ := NewSalesOrderItem(orderID, productID, "Test", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)

	t.Run("updates price and recalculates amount", func(t *testing.T) {
		newPrice := valueobject.NewMoneyCNYFromFloat(150.00)
		err := item.UpdateUnitPrice(newPrice)
		require.NoError(t, err)

		assert.True(t, item.UnitPrice.Equal(decimal.NewFromFloat(150.00)))
		assert.True(t, item.Amount.Equal(decimal.NewFromFloat(1500.00))) // 10 * 150
	})

	t.Run("fails with negative price", func(t *testing.T) {
		newPrice := valueobject.NewMoneyCNYFromFloat(-10.00)
		err := item.UpdateUnitPrice(newPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unit price cannot be negative")
	})
}

// ============================================
// AddItem Tests
// ============================================

func TestSalesOrder_AddItem(t *testing.T) {
	t.Run("adds item and calculates totals", func(t *testing.T) {
		order := createTestOrder(t)
		order.ClearDomainEvents()

		productID := uuid.New()
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		item, err := order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)
		require.NoError(t, err)
		require.NotNil(t, item)

		assert.Len(t, order.Items, 1)
		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, order.PayableAmount.Equal(decimal.NewFromFloat(1000.00)))
	})

	t.Run("adds multiple items", func(t *testing.T) {
		order := createTestOrder(t)

		addTestItem(t, order, "Product 1", 10, 100.00) // 1000
		addTestItem(t, order, "Product 2", 5, 200.00)  // 1000

		assert.Len(t, order.Items, 2)
		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(2000.00)))
	})

	t.Run("fails to add duplicate product", func(t *testing.T) {
		order := createTestOrder(t)
		productID := uuid.New()
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)

		_, err := order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)
		require.NoError(t, err)

		_, err = order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(5), decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists in order")
	})

	t.Run("fails when order is not in draft status", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)

		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		productID := uuid.New()
		unitPrice := valueobject.NewMoneyCNYFromFloat(50.00)
		_, err := order.AddItem(productID, "Product 2", "SKU-002", "pcs", "pcs", decimal.NewFromFloat(5), decimal.NewFromInt(1), unitPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-draft order")
	})
}

// ============================================
// UpdateItem Tests
// ============================================

func TestSalesOrder_UpdateItemQuantity(t *testing.T) {
	t.Run("updates item quantity", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00) // 1000

		err := order.UpdateItemQuantity(item.ID, decimal.NewFromFloat(20))
		require.NoError(t, err)

		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(2000.00)))
		assert.True(t, order.Items[0].Quantity.Equal(decimal.NewFromFloat(20)))
	})

	t.Run("fails with non-existent item", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)

		err := order.UpdateItemQuantity(uuid.New(), decimal.NewFromFloat(20))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})

	t.Run("fails when order is not in draft status", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()

		err := order.UpdateItemQuantity(item.ID, decimal.NewFromFloat(20))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-draft order")
	})
}

func TestSalesOrder_UpdateItemPrice(t *testing.T) {
	t.Run("updates item price", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00) // 1000

		newPrice := valueobject.NewMoneyCNYFromFloat(150.00)
		err := order.UpdateItemPrice(item.ID, newPrice)
		require.NoError(t, err)

		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(1500.00)))
	})

	t.Run("fails with non-existent item", func(t *testing.T) {
		order := createTestOrder(t)

		newPrice := valueobject.NewMoneyCNYFromFloat(150.00)
		err := order.UpdateItemPrice(uuid.New(), newPrice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})
}

// ============================================
// RemoveItem Tests
// ============================================

func TestSalesOrder_RemoveItem(t *testing.T) {
	t.Run("removes item and recalculates totals", func(t *testing.T) {
		order := createTestOrder(t)
		item1 := addTestItem(t, order, "Product 1", 10, 100.00) // 1000
		addTestItem(t, order, "Product 2", 5, 200.00)           // 1000

		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(2000.00)))

		err := order.RemoveItem(item1.ID)
		require.NoError(t, err)

		assert.Len(t, order.Items, 1)
		assert.True(t, order.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
	})

	t.Run("fails with non-existent item", func(t *testing.T) {
		order := createTestOrder(t)

		err := order.RemoveItem(uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})

	t.Run("fails when order is not in draft status", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00)
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

func TestSalesOrder_ApplyDiscount(t *testing.T) {
	t.Run("applies discount", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00) // 1000

		discount := valueobject.NewMoneyCNYFromFloat(100.00)
		err := order.ApplyDiscount(discount)
		require.NoError(t, err)

		assert.True(t, order.DiscountAmount.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, order.PayableAmount.Equal(decimal.NewFromFloat(900.00)))
	})

	t.Run("fails with discount exceeding total", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00) // 1000

		discount := valueobject.NewMoneyCNYFromFloat(1500.00)
		err := order.ApplyDiscount(discount)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed total amount")
	})

	t.Run("fails with negative discount", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)

		discount := valueobject.NewMoneyCNYFromFloat(-50.00)
		err := order.ApplyDiscount(discount)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("fails when order is not in draft status", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
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

func TestSalesOrder_SetWarehouse(t *testing.T) {
	t.Run("sets warehouse in draft status", func(t *testing.T) {
		order := createTestOrder(t)
		warehouseID := uuid.New()

		err := order.SetWarehouse(warehouseID)
		require.NoError(t, err)

		assert.NotNil(t, order.WarehouseID)
		assert.Equal(t, warehouseID, *order.WarehouseID)
	})

	t.Run("sets warehouse in confirmed status", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID1 := uuid.New()
		order.SetWarehouse(warehouseID1)
		order.Confirm()

		warehouseID2 := uuid.New()
		err := order.SetWarehouse(warehouseID2)
		require.NoError(t, err)
		assert.Equal(t, warehouseID2, *order.WarehouseID)
	})

	t.Run("fails with nil warehouse ID", func(t *testing.T) {
		order := createTestOrder(t)

		err := order.SetWarehouse(uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse ID cannot be empty")
	})

	t.Run("fails when order is shipped", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.Ship()

		newWarehouseID := uuid.New()
		err := order.SetWarehouse(newWarehouseID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "current status")
	})
}

// ============================================
// Confirm Tests
// ============================================

func TestSalesOrder_Confirm(t *testing.T) {
	t.Run("confirms order with items", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		order.ClearDomainEvents()

		err := order.Confirm()
		require.NoError(t, err)

		assert.Equal(t, OrderStatusConfirmed, order.Status)
		assert.NotNil(t, order.ConfirmedAt)
		assert.True(t, order.IsConfirmed())
	})

	t.Run("publishes SalesOrderConfirmed event", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		order.ClearDomainEvents()

		order.Confirm()

		events := order.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeSalesOrderConfirmed, events[0].EventType())

		event, ok := events[0].(*SalesOrderConfirmedEvent)
		require.True(t, ok)
		assert.Equal(t, order.ID, event.OrderID)
		assert.Len(t, event.Items, 1)
	})

	t.Run("fails without items", func(t *testing.T) {
		order := createTestOrder(t)

		err := order.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "without items")
	})

	t.Run("fails when already confirmed", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()

		err := order.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CONFIRMED status")
	})

	t.Run("fails with zero payable amount", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00) // 1000
		order.ApplyDiscount(valueobject.NewMoneyCNYFromFloat(1000.00))

		err := order.Confirm()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be positive")
	})
}

// ============================================
// Ship Tests
// ============================================

func TestSalesOrder_Ship(t *testing.T) {
	t.Run("ships confirmed order", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()

		err := order.Ship()
		require.NoError(t, err)

		assert.Equal(t, OrderStatusShipped, order.Status)
		assert.NotNil(t, order.ShippedAt)
		assert.True(t, order.IsShipped())
	})

	t.Run("publishes SalesOrderShipped event", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()

		order.Ship()

		events := order.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeSalesOrderShipped, events[0].EventType())

		event, ok := events[0].(*SalesOrderShippedEvent)
		require.True(t, ok)
		assert.Equal(t, warehouseID, event.WarehouseID)
	})

	t.Run("fails without warehouse", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()

		err := order.Ship()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse must be set")
	})

	t.Run("fails when not confirmed", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)

		err := order.Ship()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DRAFT status")
	})
}

// ============================================
// Complete Tests
// ============================================

func TestSalesOrder_Complete(t *testing.T) {
	t.Run("completes shipped order", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.Ship()
		order.ClearDomainEvents()

		err := order.Complete()
		require.NoError(t, err)

		assert.Equal(t, OrderStatusCompleted, order.Status)
		assert.NotNil(t, order.CompletedAt)
		assert.True(t, order.IsCompleted())
		assert.True(t, order.IsTerminal())
	})

	t.Run("publishes SalesOrderCompleted event", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.Ship()
		order.ClearDomainEvents()

		order.Complete()

		events := order.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeSalesOrderCompleted, events[0].EventType())
	})

	t.Run("fails when not shipped", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()

		err := order.Complete()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CONFIRMED status")
	})
}

// ============================================
// Cancel Tests
// ============================================

func TestSalesOrder_Cancel(t *testing.T) {
	t.Run("cancels draft order", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		order.ClearDomainEvents()

		err := order.Cancel("Customer request")
		require.NoError(t, err)

		assert.Equal(t, OrderStatusCancelled, order.Status)
		assert.NotNil(t, order.CancelledAt)
		assert.Equal(t, "Customer request", order.CancelReason)
		assert.True(t, order.IsCancelled())
		assert.True(t, order.IsTerminal())
	})

	t.Run("cancels confirmed order with wasConfirmed flag", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()
		order.ClearDomainEvents()

		err := order.Cancel("Out of stock")
		require.NoError(t, err)

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*SalesOrderCancelledEvent)
		require.True(t, ok)
		assert.True(t, event.WasConfirmed) // Should release stock locks
	})

	t.Run("fails without reason", func(t *testing.T) {
		order := createTestOrder(t)

		err := order.Cancel("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cancel reason is required")
	})

	t.Run("fails when shipped", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.Ship()

		err := order.Cancel("Too late")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SHIPPED status")
	})

	t.Run("fails when already cancelled", func(t *testing.T) {
		order := createTestOrder(t)
		order.Cancel("First time")

		err := order.Cancel("Second time")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CANCELLED status")
	})
}

// ============================================
// Helper Methods Tests
// ============================================

func TestSalesOrder_HelperMethods(t *testing.T) {
	t.Run("ItemCount returns correct count", func(t *testing.T) {
		order := createTestOrder(t)
		assert.Equal(t, 0, order.ItemCount())

		addTestItem(t, order, "Product 1", 10, 100.00)
		assert.Equal(t, 1, order.ItemCount())

		addTestItem(t, order, "Product 2", 5, 50.00)
		assert.Equal(t, 2, order.ItemCount())
	})

	t.Run("TotalQuantity returns sum of item quantities", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		addTestItem(t, order, "Product 2", 5, 50.00)

		assert.True(t, order.TotalQuantity().Equal(decimal.NewFromFloat(15)))
	})

	t.Run("GetItem returns item by ID", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00)

		found := order.GetItem(item.ID)
		require.NotNil(t, found)
		assert.Equal(t, item.ID, found.ID)

		notFound := order.GetItem(uuid.New())
		assert.Nil(t, notFound)
	})

	t.Run("GetItemByProduct returns item by product ID", func(t *testing.T) {
		order := createTestOrder(t)
		productID := uuid.New()
		unitPrice := valueobject.NewMoneyCNYFromFloat(100.00)
		order.AddItem(productID, "Product 1", "SKU-001", "pcs", "pcs", decimal.NewFromFloat(10), decimal.NewFromInt(1), unitPrice)

		found := order.GetItemByProduct(productID)
		require.NotNil(t, found)
		assert.Equal(t, productID, found.ProductID)

		notFound := order.GetItemByProduct(uuid.New())
		assert.Nil(t, notFound)
	})

	t.Run("CanModify returns true only for draft", func(t *testing.T) {
		order := createTestOrder(t)
		assert.True(t, order.CanModify())

		addTestItem(t, order, "Product 1", 10, 100.00)
		order.Confirm()
		assert.False(t, order.CanModify())
	})

	t.Run("Money getters return correct values", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00) // 1000
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

func TestSalesOrder_StatusHelpers(t *testing.T) {
	order := createTestOrder(t)

	assert.True(t, order.IsDraft())
	assert.False(t, order.IsConfirmed())
	assert.False(t, order.IsShipped())
	assert.False(t, order.IsCompleted())
	assert.False(t, order.IsCancelled())
	assert.False(t, order.IsTerminal())

	addTestItem(t, order, "Product 1", 10, 100.00)
	order.Confirm()

	assert.False(t, order.IsDraft())
	assert.True(t, order.IsConfirmed())
	assert.False(t, order.IsTerminal())

	warehouseID := uuid.New()
	order.SetWarehouse(warehouseID)
	order.Ship()

	assert.True(t, order.IsShipped())
	assert.False(t, order.IsTerminal())

	order.Complete()

	assert.True(t, order.IsCompleted())
	assert.True(t, order.IsTerminal())
}

// ============================================
// Event Tests
// ============================================

func TestSalesOrderEvents(t *testing.T) {
	t.Run("SalesOrderCreatedEvent has correct fields", func(t *testing.T) {
		order := createTestOrder(t)

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*SalesOrderCreatedEvent)
		require.True(t, ok)

		assert.Equal(t, order.ID, event.OrderID)
		assert.Equal(t, order.OrderNumber, event.OrderNumber)
		assert.Equal(t, order.CustomerID, event.CustomerID)
		assert.Equal(t, order.CustomerName, event.CustomerName)
		assert.Equal(t, order.TenantID, event.TenantID())
		assert.Equal(t, EventTypeSalesOrderCreated, event.EventType())
		assert.Equal(t, AggregateTypeSalesOrder, event.AggregateType())
	})

	t.Run("SalesOrderConfirmedEvent includes all items", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		addTestItem(t, order, "Product 2", 5, 200.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.ClearDomainEvents()
		order.Confirm()

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*SalesOrderConfirmedEvent)
		require.True(t, ok)

		assert.Len(t, event.Items, 2)
		assert.Equal(t, &warehouseID, event.WarehouseID)
		assert.True(t, event.TotalAmount.Equal(order.TotalAmount))
		assert.True(t, event.PayableAmount.Equal(order.PayableAmount))
	})

	t.Run("SalesOrderShippedEvent has warehouse ID", func(t *testing.T) {
		order := createTestOrder(t)
		addTestItem(t, order, "Product 1", 10, 100.00)
		warehouseID := uuid.New()
		order.SetWarehouse(warehouseID)
		order.Confirm()
		order.ClearDomainEvents()
		order.Ship()

		events := order.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*SalesOrderShippedEvent)
		require.True(t, ok)

		assert.Equal(t, warehouseID, event.WarehouseID)
	})

	t.Run("SalesOrderCancelledEvent has wasConfirmed flag", func(t *testing.T) {
		// Test cancelling draft order
		draftOrder := createTestOrder(t)
		draftOrder.ClearDomainEvents()
		draftOrder.Cancel("Draft cancel")

		events := draftOrder.GetDomainEvents()
		event1, _ := events[0].(*SalesOrderCancelledEvent)
		assert.False(t, event1.WasConfirmed)

		// Test cancelling confirmed order
		confirmedOrder := createTestOrder(t)
		addTestItem(t, confirmedOrder, "Product 1", 10, 100.00)
		confirmedOrder.Confirm()
		confirmedOrder.ClearDomainEvents()
		confirmedOrder.Cancel("Confirmed cancel")

		events = confirmedOrder.GetDomainEvents()
		event2, _ := events[0].(*SalesOrderCancelledEvent)
		assert.True(t, event2.WasConfirmed)
	})
}

// ============================================
// Edge Cases
// ============================================

func TestSalesOrder_EdgeCases(t *testing.T) {
	t.Run("discount adjustment when removing items", func(t *testing.T) {
		order := createTestOrder(t)
		item1 := addTestItem(t, order, "Product 1", 10, 100.00) // 1000
		addTestItem(t, order, "Product 2", 5, 100.00)           // 500
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
		order := createTestOrder(t)
		originalVersion := order.GetVersion()

		order.SetRemark("Test remark")

		assert.Equal(t, "Test remark", order.Remark)
		assert.Equal(t, originalVersion+1, order.GetVersion())
	})

	t.Run("SalesOrderItem SetRemark", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00)

		item.SetRemark("Item specific note")
		assert.Equal(t, "Item specific note", item.Remark)
	})

	t.Run("SalesOrderItem GetAmountMoney", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00) // Amount: 1000

		money := item.GetAmountMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(1000.00)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})

	t.Run("SalesOrderItem GetUnitPriceMoney", func(t *testing.T) {
		order := createTestOrder(t)
		item := addTestItem(t, order, "Product 1", 10, 100.00)

		money := item.GetUnitPriceMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(100.00)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})
}
