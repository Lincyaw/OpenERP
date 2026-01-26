package trade

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test sales order
func createTestSalesOrderForReturn(t *testing.T) *SalesOrder {
	tenantID := uuid.New()
	customerID := uuid.New()

	order, err := NewSalesOrder(tenantID, "SO-20260124-001", customerID, "Test Customer")
	require.NoError(t, err)

	// Add items
	_, err = order.AddItem(
		uuid.New(), "Product A", "PROD-A", "个", "个", decimal.NewFromInt(10), decimal.NewFromInt(1), valueobject.NewMoneyCNY(decimal.NewFromInt(100)),
	)
	require.NoError(t, err)

	_, err = order.AddItem(
		uuid.New(), "Product B", "PROD-B", "箱", "箱", decimal.NewFromInt(5), decimal.NewFromInt(1), valueobject.NewMoneyCNY(decimal.NewFromInt(200)),
	)
	require.NoError(t, err)

	// Set warehouse and confirm
	warehouseID := uuid.New()
	require.NoError(t, order.SetWarehouse(warehouseID))
	require.NoError(t, order.Confirm())
	require.NoError(t, order.Ship())

	return order
}

func TestNewSalesReturn(t *testing.T) {
	t.Run("creates sales return from shipped order", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)

		sr, err := NewSalesReturn(order.TenantID, "SR-20260124-001", order)
		require.NoError(t, err)
		assert.NotNil(t, sr)
		assert.Equal(t, "SR-20260124-001", sr.ReturnNumber)
		assert.Equal(t, order.ID, sr.SalesOrderID)
		assert.Equal(t, order.OrderNumber, sr.SalesOrderNumber)
		assert.Equal(t, order.CustomerID, sr.CustomerID)
		assert.Equal(t, order.CustomerName, sr.CustomerName)
		assert.Equal(t, ReturnStatusDraft, sr.Status)
		assert.Equal(t, 0, len(sr.Items))
		assert.True(t, sr.TotalRefund.IsZero())
	})

	t.Run("fails with empty return number", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)

		sr, err := NewSalesReturn(order.TenantID, "", order)
		assert.Nil(t, sr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Return number cannot be empty")
	})

	t.Run("fails with nil order", func(t *testing.T) {
		sr, err := NewSalesReturn(uuid.New(), "SR-001", nil)
		assert.Nil(t, sr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Sales order cannot be nil")
	})

	t.Run("fails with draft order", func(t *testing.T) {
		tenantID := uuid.New()
		order, _ := NewSalesOrder(tenantID, "SO-001", uuid.New(), "Customer")

		sr, err := NewSalesReturn(tenantID, "SR-001", order)
		assert.Nil(t, sr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shipped or completed orders")
	})
}

func TestSalesReturn_AddItem(t *testing.T) {
	t.Run("adds item successfully", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)

		item, err := sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		require.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, 1, len(sr.Items))
		assert.Equal(t, decimal.NewFromInt(3), item.ReturnQuantity)
		assert.Equal(t, decimal.NewFromInt(300), item.RefundAmount) // 3 * 100
		assert.Equal(t, decimal.NewFromInt(300), sr.TotalRefund)
	})

	t.Run("fails with duplicate item", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)

		_, err := sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		require.NoError(t, err)

		_, err = sr.AddItem(&order.Items[0], decimal.NewFromInt(2))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("fails when return quantity exceeds original", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)

		_, err := sr.AddItem(&order.Items[0], decimal.NewFromInt(15)) // Original is 10
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceed original quantity")
	})

	t.Run("fails in non-draft status", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()

		_, err := sr.AddItem(&order.Items[1], decimal.NewFromInt(2))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-draft")
	})
}

func TestSalesReturn_UpdateItemQuantity(t *testing.T) {
	t.Run("updates quantity successfully", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		item, _ := sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := sr.UpdateItemQuantity(item.ID, decimal.NewFromInt(5))
		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(5), sr.Items[0].ReturnQuantity)
		assert.Equal(t, decimal.NewFromInt(500), sr.TotalRefund)
	})

	t.Run("fails with invalid quantity", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		item, _ := sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := sr.UpdateItemQuantity(item.ID, decimal.NewFromInt(0))
		assert.Error(t, err)
	})

	t.Run("fails with item not found", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)

		err := sr.UpdateItemQuantity(uuid.New(), decimal.NewFromInt(5))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSalesReturn_RemoveItem(t *testing.T) {
	t.Run("removes item successfully", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		item, _ := sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := sr.RemoveItem(item.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, len(sr.Items))
		assert.True(t, sr.TotalRefund.IsZero())
	})

	t.Run("fails with item not found", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)

		err := sr.RemoveItem(uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSalesReturn_StatusTransitions(t *testing.T) {
	t.Run("submit transitions to pending", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := sr.Submit()
		require.NoError(t, err)
		assert.Equal(t, ReturnStatusPending, sr.Status)
		assert.NotNil(t, sr.SubmittedAt)
	})

	t.Run("submit fails without items", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)

		err := sr.Submit()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "without items")
	})

	t.Run("approve transitions to approved", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()

		approverID := uuid.New()
		err := sr.Approve(approverID, "Approved - valid return")
		require.NoError(t, err)
		assert.Equal(t, ReturnStatusApproved, sr.Status)
		assert.NotNil(t, sr.ApprovedAt)
		assert.Equal(t, &approverID, sr.ApprovedBy)
		assert.Equal(t, "Approved - valid return", sr.ApprovalNote)
	})

	t.Run("reject transitions to rejected", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()

		rejecterID := uuid.New()
		err := sr.Reject(rejecterID, "Invalid return request")
		require.NoError(t, err)
		assert.Equal(t, ReturnStatusRejected, sr.Status)
		assert.NotNil(t, sr.RejectedAt)
		assert.Equal(t, &rejecterID, sr.RejectedBy)
		assert.Equal(t, "Invalid return request", sr.RejectionReason)
	})

	t.Run("reject fails without reason", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()

		err := sr.Reject(uuid.New(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reason is required")
	})

	t.Run("complete transitions to completed", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()
		sr.Approve(uuid.New(), "")
		sr.Receive() // Must receive before completing

		err := sr.Complete()
		require.NoError(t, err)
		assert.Equal(t, ReturnStatusCompleted, sr.Status)
		assert.NotNil(t, sr.CompletedAt)
	})

	t.Run("complete fails without warehouse", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.WarehouseID = nil // Clear warehouse
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()
		sr.Approve(uuid.New(), "")

		err := sr.Receive() // Must receive before completing
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse must be set")
	})

	t.Run("cancel works from draft", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := sr.Cancel("Changed mind")
		require.NoError(t, err)
		assert.Equal(t, ReturnStatusCancelled, sr.Status)
	})

	t.Run("cancel works from pending", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()

		err := sr.Cancel("Customer withdrew request")
		require.NoError(t, err)
		assert.Equal(t, ReturnStatusCancelled, sr.Status)
	})

	t.Run("cancel works from approved", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()
		sr.Approve(uuid.New(), "")

		err := sr.Cancel("Cannot process return")
		require.NoError(t, err)
		assert.Equal(t, ReturnStatusCancelled, sr.Status)
	})

	t.Run("cancel fails from completed", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()
		sr.Approve(uuid.New(), "")
		sr.Receive() // Must receive before completing
		sr.Complete()

		err := sr.Cancel("Try to cancel")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "COMPLETED")
	})

	t.Run("cannot approve from draft", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

		err := sr.Approve(uuid.New(), "")
		assert.Error(t, err)
	})
}

func TestReturnStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from     ReturnStatus
		to       ReturnStatus
		expected bool
	}{
		// From DRAFT
		{ReturnStatusDraft, ReturnStatusPending, true},
		{ReturnStatusDraft, ReturnStatusCancelled, true},
		{ReturnStatusDraft, ReturnStatusApproved, false},
		{ReturnStatusDraft, ReturnStatusRejected, false},
		{ReturnStatusDraft, ReturnStatusCompleted, false},

		// From PENDING
		{ReturnStatusPending, ReturnStatusApproved, true},
		{ReturnStatusPending, ReturnStatusRejected, true},
		{ReturnStatusPending, ReturnStatusCancelled, true},
		{ReturnStatusPending, ReturnStatusCompleted, false},
		{ReturnStatusPending, ReturnStatusDraft, false},

		// From APPROVED - must go through RECEIVING first
		{ReturnStatusApproved, ReturnStatusReceiving, true},
		{ReturnStatusApproved, ReturnStatusCompleted, false}, // Direct transition not allowed
		{ReturnStatusApproved, ReturnStatusCancelled, true},
		{ReturnStatusApproved, ReturnStatusRejected, false},
		{ReturnStatusApproved, ReturnStatusPending, false},

		// From RECEIVING
		{ReturnStatusReceiving, ReturnStatusCompleted, true},
		{ReturnStatusReceiving, ReturnStatusCancelled, true},
		{ReturnStatusReceiving, ReturnStatusApproved, false},
		{ReturnStatusReceiving, ReturnStatusPending, false},
		{ReturnStatusReceiving, ReturnStatusDraft, false},

		// Terminal states
		{ReturnStatusRejected, ReturnStatusPending, false},
		{ReturnStatusCompleted, ReturnStatusCancelled, false},
		{ReturnStatusCancelled, ReturnStatusDraft, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSalesReturn_Events(t *testing.T) {
	t.Run("created event on new return", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)

		events := sr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		assert.Equal(t, EventTypeSalesReturnCreated, events[0].EventType())
	})

	t.Run("submitted event on submit", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.ClearDomainEvents()

		sr.Submit()

		events := sr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		assert.Equal(t, EventTypeSalesReturnSubmitted, events[0].EventType())
	})

	t.Run("approved event on approve", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()
		sr.ClearDomainEvents()

		sr.Approve(uuid.New(), "OK")

		events := sr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		assert.Equal(t, EventTypeSalesReturnApproved, events[0].EventType())
	})

	t.Run("completed event includes items", func(t *testing.T) {
		order := createTestSalesOrderForReturn(t)
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
		sr.Submit()
		sr.Approve(uuid.New(), "")
		sr.Receive() // Must receive before completing
		sr.ClearDomainEvents()

		sr.Complete()

		events := sr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		completedEvent, ok := events[0].(*SalesReturnCompletedEvent)
		require.True(t, ok)
		assert.Equal(t, 1, len(completedEvent.Items))
		assert.Equal(t, decimal.NewFromInt(3), completedEvent.Items[0].ReturnQuantity)
	})
}

func TestSalesReturn_HelperMethods(t *testing.T) {
	order := createTestSalesOrderForReturn(t)
	sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
	sr.AddItem(&order.Items[0], decimal.NewFromInt(3))
	sr.AddItem(&order.Items[1], decimal.NewFromInt(2))

	t.Run("ItemCount", func(t *testing.T) {
		assert.Equal(t, 2, sr.ItemCount())
	})

	t.Run("TotalReturnQuantity", func(t *testing.T) {
		assert.Equal(t, decimal.NewFromInt(5), sr.TotalReturnQuantity())
	})

	t.Run("GetItem", func(t *testing.T) {
		item := sr.GetItem(sr.Items[0].ID)
		assert.NotNil(t, item)
		assert.Equal(t, sr.Items[0].ID, item.ID)
	})

	t.Run("GetItemByOrderItem", func(t *testing.T) {
		item := sr.GetItemByOrderItem(order.Items[0].ID)
		assert.NotNil(t, item)
		assert.Equal(t, order.Items[0].ID, item.SalesOrderItemID)
	})

	t.Run("GetTotalRefundMoney", func(t *testing.T) {
		money := sr.GetTotalRefundMoney()
		expected := decimal.NewFromInt(700) // 3*100 + 2*200
		assert.True(t, expected.Equal(money.Amount()))
	})

	t.Run("status predicates", func(t *testing.T) {
		assert.True(t, sr.IsDraft())
		assert.False(t, sr.IsPending())
		assert.True(t, sr.CanModify())
		assert.False(t, sr.IsTerminal())
	})
}

func TestSalesReturnItem_SetReason(t *testing.T) {
	order := createTestSalesOrderForReturn(t)
	sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
	sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

	// Get item from slice directly since it's stored by value
	sr.Items[0].SetReason("Product defective")
	assert.Equal(t, "Product defective", sr.Items[0].Reason)
}

func TestSalesReturnItem_SetCondition(t *testing.T) {
	order := createTestSalesOrderForReturn(t)
	sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
	sr.AddItem(&order.Items[0], decimal.NewFromInt(3))

	// Get item from slice directly since it's stored by value
	sr.Items[0].SetCondition("damaged")
	assert.Equal(t, "damaged", sr.Items[0].ConditionOnReturn)
}

func TestSalesReturnItem_UnitConversion(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	// Create an order with auxiliary unit (箱=box) that converts to base unit (个=pieces)
	order, err := NewSalesOrder(tenantID, "SO-20260126-001", customerID, "Test Customer")
	require.NoError(t, err)

	// Add item: 5 boxes @ 12 pieces per box = 60 pieces total
	productID := uuid.New()
	_, err = order.AddItem(
		productID, "Product C", "PROD-C",
		"箱",                    // order unit (box)
		"个",                    // base unit (pieces)
		decimal.NewFromInt(5),  // quantity in boxes
		decimal.NewFromInt(12), // conversion rate: 1 box = 12 pieces
		valueobject.NewMoneyCNY(decimal.NewFromInt(1200)), // price per box
	)
	require.NoError(t, err)

	warehouseID := uuid.New()
	require.NoError(t, order.SetWarehouse(warehouseID))
	require.NoError(t, order.Confirm())
	require.NoError(t, order.Ship())

	t.Run("unit conversion fields are set correctly on add", func(t *testing.T) {
		sr, _ := NewSalesReturn(order.TenantID, "SR-001", order)
		item, err := sr.AddItem(&order.Items[0], decimal.NewFromInt(2)) // return 2 boxes
		require.NoError(t, err)

		assert.Equal(t, "箱", item.Unit)
		assert.Equal(t, "个", item.BaseUnit)
		assert.True(t, item.ConversionRate.Equal(decimal.NewFromInt(12)))
		assert.True(t, item.ReturnQuantity.Equal(decimal.NewFromInt(2)))
		// 2 boxes * 12 pieces/box = 24 pieces
		assert.True(t, item.BaseQuantity.Equal(decimal.NewFromInt(24)))
	})

	t.Run("base quantity recalculated on update", func(t *testing.T) {
		sr, _ := NewSalesReturn(order.TenantID, "SR-002", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(2))

		err := sr.UpdateItemQuantity(sr.Items[0].ID, decimal.NewFromInt(3)) // update to 3 boxes
		require.NoError(t, err)

		assert.True(t, sr.Items[0].ReturnQuantity.Equal(decimal.NewFromInt(3)))
		// 3 boxes * 12 pieces/box = 36 pieces
		assert.True(t, sr.Items[0].BaseQuantity.Equal(decimal.NewFromInt(36)))
	})

	t.Run("event includes base quantity", func(t *testing.T) {
		sr, _ := NewSalesReturn(order.TenantID, "SR-003", order)
		sr.AddItem(&order.Items[0], decimal.NewFromInt(2))
		sr.Submit()
		sr.Approve(uuid.New(), "")
		sr.Receive()
		sr.ClearDomainEvents()

		sr.Complete()

		events := sr.GetDomainEvents()
		require.Equal(t, 1, len(events))
		completedEvent, ok := events[0].(*SalesReturnCompletedEvent)
		require.True(t, ok)
		require.Equal(t, 1, len(completedEvent.Items))

		// Verify event includes unit conversion fields
		eventItem := completedEvent.Items[0]
		assert.True(t, eventItem.ReturnQuantity.Equal(decimal.NewFromInt(2)))
		assert.True(t, eventItem.BaseQuantity.Equal(decimal.NewFromInt(24)))
		assert.True(t, eventItem.ConversionRate.Equal(decimal.NewFromInt(12)))
		assert.Equal(t, "箱", eventItem.Unit)
		assert.Equal(t, "个", eventItem.BaseUnit)
	})
}
