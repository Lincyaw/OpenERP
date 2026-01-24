package inventory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStockTaking(t *testing.T) {
	tenantID := uuid.New()
	warehouseID := uuid.New()
	creatorID := uuid.New()
	takingDate := time.Now()

	t.Run("creates stock taking with valid inputs", func(t *testing.T) {
		st, err := NewStockTaking(tenantID, warehouseID, "Main Warehouse", "ST-20260124-001", takingDate, creatorID, "John Doe")

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, st.ID)
		assert.Equal(t, tenantID, st.TenantID)
		assert.Equal(t, warehouseID, st.WarehouseID)
		assert.Equal(t, "Main Warehouse", st.WarehouseName)
		assert.Equal(t, "ST-20260124-001", st.TakingNumber)
		assert.Equal(t, StockTakingStatusDraft, st.Status)
		assert.Equal(t, creatorID, st.CreatedByID)
		assert.Equal(t, "John Doe", st.CreatedByName)
		assert.Equal(t, 0, st.TotalItems)
		assert.Equal(t, 0, st.CountedItems)
		assert.True(t, st.TotalDifference.IsZero())
		assert.Len(t, st.GetDomainEvents(), 1)
	})

	t.Run("fails with empty warehouse ID", func(t *testing.T) {
		_, err := NewStockTaking(tenantID, uuid.Nil, "Main Warehouse", "ST-001", takingDate, creatorID, "John")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse ID cannot be empty")
	})

	t.Run("fails with empty warehouse name", func(t *testing.T) {
		_, err := NewStockTaking(tenantID, warehouseID, "", "ST-001", takingDate, creatorID, "John")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Warehouse name cannot be empty")
	})

	t.Run("fails with empty taking number", func(t *testing.T) {
		_, err := NewStockTaking(tenantID, warehouseID, "Main Warehouse", "", takingDate, creatorID, "John")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Taking number cannot be empty")
	})

	t.Run("fails with empty creator ID", func(t *testing.T) {
		_, err := NewStockTaking(tenantID, warehouseID, "Main Warehouse", "ST-001", takingDate, uuid.Nil, "John")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Creator ID cannot be empty")
	})
}

func TestStockTaking_AddItem(t *testing.T) {
	st := createTestStockTaking(t)
	productID := uuid.New()
	systemQty := decimal.NewFromInt(100)
	unitCost := decimal.NewFromFloat(10.5)

	t.Run("adds item successfully in draft status", func(t *testing.T) {
		err := st.AddItem(productID, "Test Product", "PRD-001", "个", systemQty, unitCost)

		require.NoError(t, err)
		assert.Equal(t, 1, st.TotalItems)
		assert.Len(t, st.Items, 1)
		assert.Equal(t, productID, st.Items[0].ProductID)
		assert.Equal(t, "Test Product", st.Items[0].ProductName)
		assert.True(t, st.Items[0].SystemQuantity.Equal(systemQty))
		assert.False(t, st.Items[0].Counted)
	})

	t.Run("fails to add duplicate product", func(t *testing.T) {
		err := st.AddItem(productID, "Same Product", "PRD-001", "个", systemQty, unitCost)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("fails to add item after counting started", func(t *testing.T) {
		st2 := createTestStockTaking(t)
		_ = st2.AddItem(uuid.New(), "Product", "PRD-002", "个", systemQty, unitCost)
		_ = st2.StartCounting()

		err := st2.AddItem(uuid.New(), "New Product", "PRD-003", "个", systemQty, unitCost)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "DRAFT")
	})
}

func TestStockTaking_RemoveItem(t *testing.T) {
	st := createTestStockTaking(t)
	productID := uuid.New()
	_ = st.AddItem(productID, "Test Product", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10.5))

	t.Run("removes item successfully", func(t *testing.T) {
		err := st.RemoveItem(productID)

		require.NoError(t, err)
		assert.Equal(t, 0, st.TotalItems)
		assert.Len(t, st.Items, 0)
	})

	t.Run("fails to remove non-existent item", func(t *testing.T) {
		err := st.RemoveItem(uuid.New())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestStockTaking_StartCounting(t *testing.T) {
	t.Run("starts counting with items", func(t *testing.T) {
		st := createTestStockTaking(t)
		_ = st.AddItem(uuid.New(), "Product", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		st.ClearDomainEvents()

		err := st.StartCounting()

		require.NoError(t, err)
		assert.Equal(t, StockTakingStatusCounting, st.Status)
		assert.NotNil(t, st.StartedAt)
		assert.Len(t, st.GetDomainEvents(), 1)
	})

	t.Run("fails without items", func(t *testing.T) {
		st := createTestStockTaking(t)

		err := st.StartCounting()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no items")
	})

	t.Run("fails from wrong status", func(t *testing.T) {
		st := createTestStockTaking(t)
		_ = st.AddItem(uuid.New(), "Product", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		_ = st.StartCounting()

		err := st.StartCounting()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot transition")
	})
}

func TestStockTaking_RecordItemCount(t *testing.T) {
	st := createTestStockTaking(t)
	productID := uuid.New()
	_ = st.AddItem(productID, "Product", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
	_ = st.StartCounting()

	t.Run("records count successfully", func(t *testing.T) {
		err := st.RecordItemCount(productID, decimal.NewFromInt(95), "Missing 5 units")

		require.NoError(t, err)
		assert.Equal(t, 1, st.CountedItems)
		assert.True(t, st.Items[0].Counted)
		assert.True(t, st.Items[0].ActualQuantity.Equal(decimal.NewFromInt(95)))
		assert.True(t, st.Items[0].DifferenceQty.Equal(decimal.NewFromInt(-5)))
		assert.Equal(t, 1, st.DifferenceItems)
		// Total difference: -5 * 10 = -50
		assert.True(t, st.TotalDifference.Equal(decimal.NewFromFloat(-50)))
	})

	t.Run("fails for non-existent product", func(t *testing.T) {
		err := st.RecordItemCount(uuid.New(), decimal.NewFromInt(95), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("fails with negative quantity", func(t *testing.T) {
		st2 := createTestStockTaking(t)
		productID2 := uuid.New()
		_ = st2.AddItem(productID2, "Product", "PRD-002", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		_ = st2.StartCounting()

		err := st2.RecordItemCount(productID2, decimal.NewFromInt(-5), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("fails when not in counting status", func(t *testing.T) {
		st2 := createTestStockTaking(t)
		productID2 := uuid.New()
		_ = st2.AddItem(productID2, "Product", "PRD-002", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))

		err := st2.RecordItemCount(productID2, decimal.NewFromInt(95), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "COUNTING")
	})
}

func TestStockTaking_SubmitForApproval(t *testing.T) {
	t.Run("submits when all items counted", func(t *testing.T) {
		st := createTestStockTaking(t)
		productID := uuid.New()
		_ = st.AddItem(productID, "Product", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		_ = st.StartCounting()
		_ = st.RecordItemCount(productID, decimal.NewFromInt(100), "")
		st.ClearDomainEvents()

		err := st.SubmitForApproval()

		require.NoError(t, err)
		assert.Equal(t, StockTakingStatusPendingApproval, st.Status)
		assert.NotNil(t, st.CompletedAt)
		assert.Len(t, st.GetDomainEvents(), 1)
	})

	t.Run("fails with incomplete counts", func(t *testing.T) {
		st := createTestStockTaking(t)
		_ = st.AddItem(uuid.New(), "Product 1", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		_ = st.AddItem(uuid.New(), "Product 2", "PRD-002", "个", decimal.NewFromInt(50), decimal.NewFromFloat(20))
		_ = st.StartCounting()
		_ = st.RecordItemCount(st.Items[0].ProductID, decimal.NewFromInt(100), "")
		// Not counting the second item

		err := st.SubmitForApproval()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Not all items have been counted")
	})
}

func TestStockTaking_Approve(t *testing.T) {
	st := createStockTakingPendingApproval(t)
	approverID := uuid.New()

	t.Run("approves successfully", func(t *testing.T) {
		st.ClearDomainEvents()

		err := st.Approve(approverID, "Admin User", "Approved after verification")

		require.NoError(t, err)
		assert.Equal(t, StockTakingStatusApproved, st.Status)
		assert.NotNil(t, st.ApprovedAt)
		assert.Equal(t, &approverID, st.ApprovedByID)
		assert.Equal(t, "Admin User", st.ApprovedByName)
		assert.Equal(t, "Approved after verification", st.ApprovalNote)
		assert.Len(t, st.GetDomainEvents(), 1)
	})

	t.Run("fails with empty approver", func(t *testing.T) {
		st2 := createStockTakingPendingApproval(t)

		err := st2.Approve(uuid.Nil, "Admin", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Approver ID cannot be empty")
	})
}

func TestStockTaking_Reject(t *testing.T) {
	st := createStockTakingPendingApproval(t)
	approverID := uuid.New()

	t.Run("rejects with reason", func(t *testing.T) {
		st.ClearDomainEvents()

		err := st.Reject(approverID, "Admin User", "Counts need to be redone")

		require.NoError(t, err)
		assert.Equal(t, StockTakingStatusRejected, st.Status)
		assert.NotNil(t, st.ApprovedAt)
		assert.Equal(t, "Counts need to be redone", st.ApprovalNote)
		assert.Len(t, st.GetDomainEvents(), 1)
	})

	t.Run("fails without reason", func(t *testing.T) {
		st2 := createStockTakingPendingApproval(t)

		err := st2.Reject(approverID, "Admin", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reason is required")
	})
}

func TestStockTaking_Cancel(t *testing.T) {
	t.Run("cancels from draft", func(t *testing.T) {
		st := createTestStockTaking(t)
		st.ClearDomainEvents()

		err := st.Cancel("No longer needed")

		require.NoError(t, err)
		assert.Equal(t, StockTakingStatusCancelled, st.Status)
		assert.Equal(t, "No longer needed", st.Remark)
		assert.Len(t, st.GetDomainEvents(), 1)
	})

	t.Run("cancels from counting", func(t *testing.T) {
		st := createTestStockTaking(t)
		_ = st.AddItem(uuid.New(), "Product", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		_ = st.StartCounting()

		err := st.Cancel("Interrupted")

		require.NoError(t, err)
		assert.Equal(t, StockTakingStatusCancelled, st.Status)
	})

	t.Run("fails from approved status", func(t *testing.T) {
		st := createStockTakingPendingApproval(t)
		_ = st.Approve(uuid.New(), "Admin", "")

		err := st.Cancel("Should not work")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot transition")
	})
}

func TestStockTaking_StatusTransitions(t *testing.T) {
	t.Run("DRAFT can transition to COUNTING or CANCELLED", func(t *testing.T) {
		assert.True(t, StockTakingStatusDraft.CanTransitionTo(StockTakingStatusCounting))
		assert.True(t, StockTakingStatusDraft.CanTransitionTo(StockTakingStatusCancelled))
		assert.False(t, StockTakingStatusDraft.CanTransitionTo(StockTakingStatusPendingApproval))
		assert.False(t, StockTakingStatusDraft.CanTransitionTo(StockTakingStatusApproved))
	})

	t.Run("COUNTING can transition to PENDING_APPROVAL or CANCELLED", func(t *testing.T) {
		assert.True(t, StockTakingStatusCounting.CanTransitionTo(StockTakingStatusPendingApproval))
		assert.True(t, StockTakingStatusCounting.CanTransitionTo(StockTakingStatusCancelled))
		assert.False(t, StockTakingStatusCounting.CanTransitionTo(StockTakingStatusDraft))
		assert.False(t, StockTakingStatusCounting.CanTransitionTo(StockTakingStatusApproved))
	})

	t.Run("PENDING_APPROVAL can transition to APPROVED or REJECTED", func(t *testing.T) {
		assert.True(t, StockTakingStatusPendingApproval.CanTransitionTo(StockTakingStatusApproved))
		assert.True(t, StockTakingStatusPendingApproval.CanTransitionTo(StockTakingStatusRejected))
		assert.False(t, StockTakingStatusPendingApproval.CanTransitionTo(StockTakingStatusCancelled))
		assert.False(t, StockTakingStatusPendingApproval.CanTransitionTo(StockTakingStatusDraft))
	})

	t.Run("terminal states cannot transition", func(t *testing.T) {
		terminals := []StockTakingStatus{StockTakingStatusApproved, StockTakingStatusRejected, StockTakingStatusCancelled}
		allStatuses := []StockTakingStatus{
			StockTakingStatusDraft, StockTakingStatusCounting, StockTakingStatusPendingApproval,
			StockTakingStatusApproved, StockTakingStatusRejected, StockTakingStatusCancelled,
		}

		for _, terminal := range terminals {
			for _, target := range allStatuses {
				assert.False(t, terminal.CanTransitionTo(target), "%s should not transition to %s", terminal, target)
			}
		}
	})
}

func TestStockTaking_GetProgress(t *testing.T) {
	t.Run("returns 0 when no items", func(t *testing.T) {
		st := createTestStockTaking(t)
		assert.Equal(t, float64(0), st.GetProgress())
	})

	t.Run("returns correct percentage", func(t *testing.T) {
		st := createTestStockTaking(t)
		_ = st.AddItem(uuid.New(), "P1", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		_ = st.AddItem(uuid.New(), "P2", "PRD-002", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
		_ = st.StartCounting()
		_ = st.RecordItemCount(st.Items[0].ProductID, decimal.NewFromInt(100), "")

		assert.Equal(t, float64(50), st.GetProgress())
	})
}

func TestStockTaking_GetItemsWithDifference(t *testing.T) {
	st := createTestStockTaking(t)
	productID1 := uuid.New()
	productID2 := uuid.New()
	_ = st.AddItem(productID1, "P1", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
	_ = st.AddItem(productID2, "P2", "PRD-002", "个", decimal.NewFromInt(50), decimal.NewFromFloat(20))
	_ = st.StartCounting()
	_ = st.RecordItemCount(productID1, decimal.NewFromInt(95), "")  // Difference: -5
	_ = st.RecordItemCount(productID2, decimal.NewFromInt(50), "")  // No difference

	items := st.GetItemsWithDifference()

	assert.Len(t, items, 1)
	assert.Equal(t, productID1, items[0].ProductID)
}

func TestStockTaking_GetUncountedItems(t *testing.T) {
	st := createTestStockTaking(t)
	productID1 := uuid.New()
	productID2 := uuid.New()
	_ = st.AddItem(productID1, "P1", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
	_ = st.AddItem(productID2, "P2", "PRD-002", "个", decimal.NewFromInt(50), decimal.NewFromFloat(20))
	_ = st.StartCounting()
	_ = st.RecordItemCount(productID1, decimal.NewFromInt(100), "")

	items := st.GetUncountedItems()

	assert.Len(t, items, 1)
	assert.Equal(t, productID2, items[0].ProductID)
}

// Helper functions

func createTestStockTaking(t *testing.T) *StockTaking {
	t.Helper()
	st, err := NewStockTaking(
		uuid.New(),
		uuid.New(),
		"Test Warehouse",
		"ST-TEST-001",
		time.Now(),
		uuid.New(),
		"Test User",
	)
	require.NoError(t, err)
	return st
}

func createStockTakingPendingApproval(t *testing.T) *StockTaking {
	t.Helper()
	st := createTestStockTaking(t)
	productID := uuid.New()
	_ = st.AddItem(productID, "Test Product", "PRD-001", "个", decimal.NewFromInt(100), decimal.NewFromFloat(10))
	_ = st.StartCounting()
	_ = st.RecordItemCount(productID, decimal.NewFromInt(100), "")
	_ = st.SubmitForApproval()
	return st
}
