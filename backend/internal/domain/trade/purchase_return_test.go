package trade

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test purchase order for return
func createTestPurchaseOrderForReturn(t *testing.T) *PurchaseOrder {
	tenantID := uuid.New()
	supplierID := uuid.New()

	order, err := NewPurchaseOrder(tenantID, "PO-20260124-001", supplierID, "Test Supplier")
	require.NoError(t, err)

	// Add items
	_, err = order.AddItem(
		uuid.New(), "Product A", "PROD-A", "个", "个", decimal.NewFromInt(10), decimal.NewFromInt(1), valueobject.NewMoneyCNY(decimal.NewFromInt(50)),
	)
	require.NoError(t, err)

	_, err = order.AddItem(
		uuid.New(), "Product B", "PROD-B", "箱", "箱", decimal.NewFromInt(5), decimal.NewFromInt(1), valueobject.NewMoneyCNY(decimal.NewFromInt(100)),
	)
	require.NoError(t, err)

	// Set warehouse and confirm
	warehouseID := uuid.New()
	require.NoError(t, order.SetWarehouse(warehouseID))
	require.NoError(t, order.Confirm())

	// Receive all items
	receiveItems := []ReceiveItem{
		{ProductID: order.Items[0].ProductID, Quantity: decimal.NewFromInt(10)},
		{ProductID: order.Items[1].ProductID, Quantity: decimal.NewFromInt(5)},
	}
	_, err = order.Receive(receiveItems)
	require.NoError(t, err)

	return order
}

// Helper function to create a partially received purchase order
func createPartiallyReceivedPurchaseOrder(t *testing.T) *PurchaseOrder {
	tenantID := uuid.New()
	supplierID := uuid.New()

	order, err := NewPurchaseOrder(tenantID, "PO-20260124-002", supplierID, "Test Supplier")
	require.NoError(t, err)

	_, err = order.AddItem(
		uuid.New(), "Product A", "PROD-A", "个", "个", decimal.NewFromInt(10), decimal.NewFromInt(1), valueobject.NewMoneyCNY(decimal.NewFromInt(50)),
	)
	require.NoError(t, err)

	warehouseID := uuid.New()
	require.NoError(t, order.SetWarehouse(warehouseID))
	require.NoError(t, order.Confirm())

	// Partially receive
	receiveItems := []ReceiveItem{
		{ProductID: order.Items[0].ProductID, Quantity: decimal.NewFromInt(6)},
	}
	_, err = order.Receive(receiveItems)
	require.NoError(t, err)

	return order
}

func TestNewPurchaseReturn(t *testing.T) {
	t.Run("creates purchase return from completed order", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)

		pr, err := NewPurchaseReturn(order.TenantID, "PR-20260124-001", order)
		require.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "PR-20260124-001", pr.ReturnNumber)
		assert.Equal(t, order.ID, pr.PurchaseOrderID)
		assert.Equal(t, order.OrderNumber, pr.PurchaseOrderNumber)
		assert.Equal(t, order.SupplierID, pr.SupplierID)
		assert.Equal(t, order.SupplierName, pr.SupplierName)
		assert.Equal(t, PurchaseReturnStatusDraft, pr.Status)
		assert.Equal(t, 0, len(pr.Items))
		assert.True(t, pr.TotalRefund.IsZero())
	})

	t.Run("creates purchase return from partially received order", func(t *testing.T) {
		order := createPartiallyReceivedPurchaseOrder(t)

		pr, err := NewPurchaseReturn(order.TenantID, "PR-001", order)
		require.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, PurchaseReturnStatusDraft, pr.Status)
	})

	t.Run("fails with empty return number", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)

		pr, err := NewPurchaseReturn(order.TenantID, "", order)
		assert.Nil(t, pr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Return number cannot be empty")
	})

	t.Run("fails with nil order", func(t *testing.T) {
		pr, err := NewPurchaseReturn(uuid.New(), "PR-001", nil)
		assert.Nil(t, pr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Purchase order cannot be nil")
	})

	t.Run("fails with draft order", func(t *testing.T) {
		tenantID := uuid.New()
		order, _ := NewPurchaseOrder(tenantID, "PO-001", uuid.New(), "Supplier")

		pr, err := NewPurchaseReturn(tenantID, "PR-001", order)
		assert.Nil(t, pr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "orders with received goods")
	})

	t.Run("fails with confirmed but not received order", func(t *testing.T) {
		tenantID := uuid.New()
		order, _ := NewPurchaseOrder(tenantID, "PO-001", uuid.New(), "Supplier")
		order.AddItem(uuid.New(), "Product", "PROD", "个", "个", decimal.NewFromInt(10), decimal.NewFromInt(1), valueobject.NewMoneyCNY(decimal.NewFromInt(50)))
		order.Confirm()

		pr, err := NewPurchaseReturn(tenantID, "PR-001", order)
		assert.Nil(t, pr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "orders with received goods")
	})
}

func TestPurchaseReturn_AddItem(t *testing.T) {
	t.Run("adds item successfully", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		item, err := pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		require.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, 1, len(pr.Items))
		assert.Equal(t, decimal.NewFromInt(3), item.ReturnQuantity)
		assert.Equal(t, decimal.NewFromInt(150), item.RefundAmount) // 3 * 50
		assert.Equal(t, decimal.NewFromInt(150), pr.TotalRefund)
	})

	t.Run("fails with duplicate item", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		_, err := pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		require.NoError(t, err)

		_, err = pr.AddItem(&order.Items[0], decimal.NewFromInt(2))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("fails when return quantity exceeds received quantity", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		_, err := pr.AddItem(&order.Items[0], decimal.NewFromInt(15)) // Received is 10
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceed received quantity")
	})

	t.Run("fails in non-draft status", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()

		_, err := pr.AddItem(&order.Items[1], decimal.NewFromInt(2))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-draft")
	})
}

func TestPurchaseReturn_UpdateItemQuantity(t *testing.T) {
	t.Run("updates quantity successfully", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		item, _ := pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.UpdateItemQuantity(item.ID, decimal.NewFromInt(5))
		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(5), pr.Items[0].ReturnQuantity)
		assert.Equal(t, decimal.NewFromInt(250), pr.TotalRefund) // 5 * 50
	})

	t.Run("fails with invalid quantity", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		item, _ := pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.UpdateItemQuantity(item.ID, decimal.NewFromInt(0))
		assert.Error(t, err)
	})

	t.Run("fails with item not found", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		err := pr.UpdateItemQuantity(uuid.New(), decimal.NewFromInt(5))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestPurchaseReturn_RemoveItem(t *testing.T) {
	t.Run("removes item successfully", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		item, _ := pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.RemoveItem(item.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, len(pr.Items))
		assert.True(t, pr.TotalRefund.IsZero())
	})

	t.Run("fails with item not found", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		err := pr.RemoveItem(uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestPurchaseReturn_StatusTransitions(t *testing.T) {
	t.Run("submit transitions to pending", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.Submit()
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusPending, pr.Status)
		assert.NotNil(t, pr.SubmittedAt)
	})

	t.Run("submit fails without items", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		err := pr.Submit()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "without items")
	})

	t.Run("approve transitions to approved", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()

		approverID := uuid.New()
		err := pr.Approve(approverID, "Approved - valid return")
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusApproved, pr.Status)
		assert.NotNil(t, pr.ApprovedAt)
		assert.Equal(t, &approverID, pr.ApprovedBy)
		assert.Equal(t, "Approved - valid return", pr.ApprovalNote)
	})

	t.Run("reject transitions to rejected", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()

		rejecterID := uuid.New()
		err := pr.Reject(rejecterID, "Invalid return request")
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusRejected, pr.Status)
		assert.NotNil(t, pr.RejectedAt)
		assert.Equal(t, &rejecterID, pr.RejectedBy)
		assert.Equal(t, "Invalid return request", pr.RejectionReason)
	})

	t.Run("reject fails without reason", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()

		err := pr.Reject(uuid.New(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reason is required")
	})

	t.Run("ship transitions to shipped", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")

		shipperID := uuid.New()
		err := pr.Ship(shipperID, "Shipping notes", "TRACK123")
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusShipped, pr.Status)
		assert.NotNil(t, pr.ShippedAt)
		assert.Equal(t, &shipperID, pr.ShippedBy)
		assert.Equal(t, "Shipping notes", pr.ShippingNote)
		assert.Equal(t, "TRACK123", pr.TrackingNumber)

		// Check items are marked as shipped
		assert.Equal(t, decimal.NewFromInt(3), pr.Items[0].ShippedQuantity)
		assert.NotNil(t, pr.Items[0].ShippedAt)
	})

	t.Run("ship fails without warehouse", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.WarehouseID = nil // Clear warehouse
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")

		err := pr.Ship(uuid.New(), "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse must be set")
	})

	t.Run("complete transitions to completed", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.Ship(uuid.New(), "", "")

		err := pr.Complete()
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusCompleted, pr.Status)
		assert.NotNil(t, pr.CompletedAt)
	})

	t.Run("complete fails from approved (must ship first)", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")

		err := pr.Complete()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "APPROVED")
	})

	t.Run("cancel works from draft", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.Cancel("Changed mind")
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusCancelled, pr.Status)
	})

	t.Run("cancel works from pending", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()

		err := pr.Cancel("Supplier resolved issue")
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusCancelled, pr.Status)
	})

	t.Run("cancel works from approved", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")

		err := pr.Cancel("Cannot ship at this time")
		require.NoError(t, err)
		assert.Equal(t, PurchaseReturnStatusCancelled, pr.Status)
	})

	t.Run("cancel fails from shipped", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.Ship(uuid.New(), "", "")

		err := pr.Cancel("Try to cancel")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SHIPPED")
	})

	t.Run("cancel fails from completed", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.Ship(uuid.New(), "", "")
		pr.Complete()

		err := pr.Cancel("Try to cancel")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "COMPLETED")
	})

	t.Run("cannot approve from draft", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.Approve(uuid.New(), "")
		assert.Error(t, err)
	})

	t.Run("cannot ship from pending", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()

		err := pr.Ship(uuid.New(), "", "")
		assert.Error(t, err)
	})
}

func TestPurchaseReturnStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from     PurchaseReturnStatus
		to       PurchaseReturnStatus
		expected bool
	}{
		// From DRAFT
		{PurchaseReturnStatusDraft, PurchaseReturnStatusPending, true},
		{PurchaseReturnStatusDraft, PurchaseReturnStatusCancelled, true},
		{PurchaseReturnStatusDraft, PurchaseReturnStatusApproved, false},
		{PurchaseReturnStatusDraft, PurchaseReturnStatusShipped, false},
		{PurchaseReturnStatusDraft, PurchaseReturnStatusCompleted, false},

		// From PENDING
		{PurchaseReturnStatusPending, PurchaseReturnStatusApproved, true},
		{PurchaseReturnStatusPending, PurchaseReturnStatusRejected, true},
		{PurchaseReturnStatusPending, PurchaseReturnStatusCancelled, true},
		{PurchaseReturnStatusPending, PurchaseReturnStatusShipped, false},
		{PurchaseReturnStatusPending, PurchaseReturnStatusDraft, false},

		// From APPROVED
		{PurchaseReturnStatusApproved, PurchaseReturnStatusShipped, true},
		{PurchaseReturnStatusApproved, PurchaseReturnStatusCancelled, true},
		{PurchaseReturnStatusApproved, PurchaseReturnStatusCompleted, false},
		{PurchaseReturnStatusApproved, PurchaseReturnStatusPending, false},

		// From SHIPPED
		{PurchaseReturnStatusShipped, PurchaseReturnStatusCompleted, true},
		{PurchaseReturnStatusShipped, PurchaseReturnStatusCancelled, false},
		{PurchaseReturnStatusShipped, PurchaseReturnStatusApproved, false},

		// Terminal states
		{PurchaseReturnStatusRejected, PurchaseReturnStatusPending, false},
		{PurchaseReturnStatusCompleted, PurchaseReturnStatusCancelled, false},
		{PurchaseReturnStatusCancelled, PurchaseReturnStatusDraft, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPurchaseReturn_Events(t *testing.T) {
	t.Run("created event on new return", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		events := pr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		assert.Equal(t, EventTypePurchaseReturnCreated, events[0].EventType())
	})

	t.Run("submitted event on submit", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.ClearDomainEvents()

		pr.Submit()

		events := pr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		assert.Equal(t, EventTypePurchaseReturnSubmitted, events[0].EventType())
	})

	t.Run("approved event on approve", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.ClearDomainEvents()

		pr.Approve(uuid.New(), "OK")

		events := pr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		assert.Equal(t, EventTypePurchaseReturnApproved, events[0].EventType())
	})

	t.Run("shipped event on ship", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.ClearDomainEvents()

		pr.Ship(uuid.New(), "Notes", "TRACK")

		events := pr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		shippedEvent, ok := events[0].(*PurchaseReturnShippedEvent)
		require.True(t, ok)
		assert.Equal(t, "TRACK", shippedEvent.TrackingNumber)
		assert.Equal(t, 1, len(shippedEvent.Items))
	})

	t.Run("completed event includes items", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.Ship(uuid.New(), "", "")
		pr.ClearDomainEvents()

		pr.Complete()

		events := pr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		completedEvent, ok := events[0].(*PurchaseReturnCompletedEvent)
		require.True(t, ok)
		assert.Equal(t, 1, len(completedEvent.Items))
		assert.Equal(t, decimal.NewFromInt(3), completedEvent.Items[0].ReturnQuantity)
	})

	t.Run("cancelled event includes wasApproved flag", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.ClearDomainEvents()

		pr.Cancel("Cannot process")

		events := pr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		cancelledEvent, ok := events[0].(*PurchaseReturnCancelledEvent)
		require.True(t, ok)
		assert.True(t, cancelledEvent.WasApproved)
	})
}

func TestPurchaseReturn_HelperMethods(t *testing.T) {
	order := createTestPurchaseOrderForReturn(t)
	pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
	pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
	pr.AddItem(&order.Items[1], decimal.NewFromInt(2))

	t.Run("ItemCount", func(t *testing.T) {
		assert.Equal(t, 2, pr.ItemCount())
	})

	t.Run("TotalReturnQuantity", func(t *testing.T) {
		assert.Equal(t, decimal.NewFromInt(5), pr.TotalReturnQuantity())
	})

	t.Run("GetItem", func(t *testing.T) {
		item := pr.GetItem(pr.Items[0].ID)
		assert.NotNil(t, item)
		assert.Equal(t, pr.Items[0].ID, item.ID)
	})

	t.Run("GetItemByOrderItem", func(t *testing.T) {
		item := pr.GetItemByOrderItem(order.Items[0].ID)
		assert.NotNil(t, item)
		assert.Equal(t, order.Items[0].ID, item.PurchaseOrderItemID)
	})

	t.Run("GetTotalRefundMoney", func(t *testing.T) {
		money := pr.GetTotalRefundMoney()
		expected := decimal.NewFromInt(350) // 3*50 + 2*100
		assert.True(t, expected.Equal(money.Amount()))
	})

	t.Run("status predicates", func(t *testing.T) {
		assert.True(t, pr.IsDraft())
		assert.False(t, pr.IsPending())
		assert.True(t, pr.CanModify())
		assert.False(t, pr.IsTerminal())
		assert.False(t, pr.CanShip())
	})

	t.Run("TotalShippedQuantity", func(t *testing.T) {
		// Before shipping
		assert.True(t, pr.TotalShippedQuantity().IsZero())

		// After shipping
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.Ship(uuid.New(), "", "")
		assert.Equal(t, decimal.NewFromInt(5), pr.TotalShippedQuantity())
	})
}

func TestPurchaseReturnItem_SetReason(t *testing.T) {
	order := createTestPurchaseOrderForReturn(t)
	pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
	pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

	pr.Items[0].SetReason("Product defective")
	assert.Equal(t, "Product defective", pr.Items[0].Reason)
}

func TestPurchaseReturnItem_SetCondition(t *testing.T) {
	order := createTestPurchaseOrderForReturn(t)
	pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
	pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

	pr.Items[0].SetCondition("defective")
	assert.Equal(t, "defective", pr.Items[0].ConditionOnReturn)
}

func TestPurchaseReturnItem_SetBatchNumber(t *testing.T) {
	order := createTestPurchaseOrderForReturn(t)
	pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
	pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

	pr.Items[0].SetBatchNumber("BATCH-001")
	assert.Equal(t, "BATCH-001", pr.Items[0].BatchNumber)
}

func TestPurchaseReturnItem_MarkShipped(t *testing.T) {
	t.Run("marks shipped successfully", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.Items[0].MarkShipped(decimal.NewFromInt(3))
		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(3), pr.Items[0].ShippedQuantity)
		assert.NotNil(t, pr.Items[0].ShippedAt)
	})

	t.Run("fails with quantity exceeding return quantity", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := pr.Items[0].MarkShipped(decimal.NewFromInt(5))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceed return quantity")
	})
}

func TestPurchaseReturnItem_ConfirmSupplierReceived(t *testing.T) {
	t.Run("confirms received successfully", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Items[0].MarkShipped(decimal.NewFromInt(3))

		err := pr.Items[0].ConfirmSupplierReceived(decimal.NewFromInt(3))
		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(3), pr.Items[0].SupplierReceivedQty)
		assert.NotNil(t, pr.Items[0].SupplierReceivedAt)
	})

	t.Run("fails with quantity exceeding shipped", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Items[0].MarkShipped(decimal.NewFromInt(3))

		err := pr.Items[0].ConfirmSupplierReceived(decimal.NewFromInt(5))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceed shipped quantity")
	})
}

func TestPurchaseReturnItem_StatusHelpers(t *testing.T) {
	order := createTestPurchaseOrderForReturn(t)
	pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
	pr.AddItem(&order.Items[0], decimal.NewFromInt(3))

	item := &pr.Items[0]

	t.Run("IsFullyShipped", func(t *testing.T) {
		assert.False(t, item.IsFullyShipped())
		item.MarkShipped(decimal.NewFromInt(3))
		assert.True(t, item.IsFullyShipped())
	})

	t.Run("IsFullyReceived", func(t *testing.T) {
		assert.False(t, item.IsFullyReceived())
		item.ConfirmSupplierReceived(decimal.NewFromInt(3))
		assert.True(t, item.IsFullyReceived())
	})
}

func TestPurchaseReturn_SetWarehouse(t *testing.T) {
	t.Run("sets warehouse in draft", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.WarehouseID = nil

		warehouseID := uuid.New()
		err := pr.SetWarehouse(warehouseID)
		require.NoError(t, err)
		assert.Equal(t, &warehouseID, pr.WarehouseID)
	})

	t.Run("sets warehouse in pending", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.WarehouseID = nil

		warehouseID := uuid.New()
		err := pr.SetWarehouse(warehouseID)
		require.NoError(t, err)
		assert.Equal(t, &warehouseID, pr.WarehouseID)
	})

	t.Run("sets warehouse in approved status", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")

		warehouseID := uuid.New()
		err := pr.SetWarehouse(warehouseID)
		require.NoError(t, err)
		assert.Equal(t, &warehouseID, pr.WarehouseID)
	})

	t.Run("fails in shipped status", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.SetWarehouse(uuid.New())
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.Ship(uuid.New(), "", "")

		err := pr.SetWarehouse(uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "current status")
	})

	t.Run("fails with empty warehouse ID", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)

		err := pr.SetWarehouse(uuid.Nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestPurchaseReturn_CanShip(t *testing.T) {
	t.Run("can ship when approved with warehouse", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")

		assert.True(t, pr.CanShip())
	})

	t.Run("cannot ship when approved without warehouse", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()
		pr.Approve(uuid.New(), "")
		pr.WarehouseID = nil

		assert.False(t, pr.CanShip())
	})

	t.Run("cannot ship when pending", func(t *testing.T) {
		order := createTestPurchaseOrderForReturn(t)
		pr, _ := NewPurchaseReturn(order.TenantID, "PR-001", order)
		pr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		pr.Submit()

		assert.False(t, pr.CanShip())
	})
}
