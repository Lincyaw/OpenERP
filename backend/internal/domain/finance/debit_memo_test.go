package finance

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDebitMemo(t *testing.T) {
	tenantID := uuid.New()
	purchaseReturnID := uuid.New()
	purchaseOrderID := uuid.New()
	supplierID := uuid.New()
	totalDebit := valueobject.NewMoneyCNY(decimal.NewFromFloat(2000.00))

	t.Run("successful creation", func(t *testing.T) {
		memo, err := NewDebitMemo(
			tenantID,
			"DM-001",
			purchaseReturnID,
			"PR-001",
			purchaseOrderID,
			"PO-001",
			supplierID,
			"Test Supplier",
			totalDebit,
			"Defective goods returned to supplier",
		)

		require.NoError(t, err)
		assert.NotNil(t, memo)
		assert.Equal(t, "DM-001", memo.MemoNumber)
		assert.Equal(t, purchaseReturnID, memo.PurchaseReturnID)
		assert.Equal(t, purchaseOrderID, memo.PurchaseOrderID)
		assert.Equal(t, supplierID, memo.SupplierID)
		assert.Equal(t, "Test Supplier", memo.SupplierName)
		assert.Equal(t, DebitMemoStatusPending, memo.Status)
		assert.True(t, memo.TotalDebit.Equal(decimal.NewFromFloat(2000.00)))
		assert.True(t, memo.RemainingAmount.Equal(decimal.NewFromFloat(2000.00)))
		assert.True(t, memo.AppliedAmount.IsZero())
		assert.Len(t, memo.GetDomainEvents(), 1)
	})

	t.Run("empty memo number", func(t *testing.T) {
		_, err := NewDebitMemo(
			tenantID,
			"",
			purchaseReturnID,
			"PR-001",
			purchaseOrderID,
			"PO-001",
			supplierID,
			"Test Supplier",
			totalDebit,
			"reason",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Memo number cannot be empty")
	})

	t.Run("empty purchase return ID", func(t *testing.T) {
		_, err := NewDebitMemo(
			tenantID,
			"DM-001",
			uuid.Nil,
			"PR-001",
			purchaseOrderID,
			"PO-001",
			supplierID,
			"Test Supplier",
			totalDebit,
			"reason",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Purchase return ID cannot be empty")
	})

	t.Run("empty supplier ID", func(t *testing.T) {
		_, err := NewDebitMemo(
			tenantID,
			"DM-001",
			purchaseReturnID,
			"PR-001",
			purchaseOrderID,
			"PO-001",
			uuid.Nil,
			"Test Supplier",
			totalDebit,
			"reason",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Supplier ID cannot be empty")
	})

	t.Run("zero total debit", func(t *testing.T) {
		_, err := NewDebitMemo(
			tenantID,
			"DM-001",
			purchaseReturnID,
			"PR-001",
			purchaseOrderID,
			"PO-001",
			supplierID,
			"Test Supplier",
			valueobject.NewMoneyCNY(decimal.Zero),
			"reason",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Total debit must be positive")
	})
}

func TestDebitMemo_AddItem(t *testing.T) {
	memo := createTestDebitMemo(t)

	t.Run("successful add item", func(t *testing.T) {
		item, err := memo.AddItem(
			memo.PurchaseReturnID,
			uuid.New(),
			uuid.New(),
			"Test Product",
			"PROD-001",
			"PCS",
			decimal.NewFromInt(10),
			valueobject.NewMoneyCNY(decimal.NewFromFloat(50.00)),
			"Wrong specification",
		)

		require.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "Test Product", item.ProductName)
		assert.True(t, item.DebitAmount.Equal(decimal.NewFromFloat(500.00)))
		assert.Equal(t, 1, memo.ItemCount())
	})

	t.Run("cannot add to non-pending memo", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		memo.Status = DebitMemoStatusApplied

		_, err := memo.AddItem(
			memo.PurchaseReturnID,
			uuid.New(),
			uuid.New(),
			"Test Product",
			"PROD-001",
			"PCS",
			decimal.NewFromInt(10),
			valueobject.NewMoneyCNY(decimal.NewFromFloat(50.00)),
			"Wrong specification",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot add items to a non-pending debit memo")
	})
}

func TestDebitMemo_ApplyToPayable(t *testing.T) {
	t.Run("successful full application", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		payableID := uuid.New()

		err := memo.ApplyToPayable(payableID, valueobject.NewMoneyCNY(decimal.NewFromFloat(2000.00)), "Apply to outstanding bill")

		require.NoError(t, err)
		assert.Equal(t, DebitMemoStatusApplied, memo.Status)
		assert.True(t, memo.AppliedAmount.Equal(decimal.NewFromFloat(2000.00)))
		assert.True(t, memo.RemainingAmount.IsZero())
		assert.Equal(t, 1, memo.ApplicationCount())
		assert.NotNil(t, memo.AppliedAt)
	})

	t.Run("successful partial application", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		payableID := uuid.New()

		err := memo.ApplyToPayable(payableID, valueobject.NewMoneyCNY(decimal.NewFromFloat(800.00)), "Partial application")

		require.NoError(t, err)
		assert.Equal(t, DebitMemoStatusPartial, memo.Status)
		assert.True(t, memo.AppliedAmount.Equal(decimal.NewFromFloat(800.00)))
		assert.True(t, memo.RemainingAmount.Equal(decimal.NewFromFloat(1200.00)))
	})

	t.Run("multiple applications until fully applied", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		payable1 := uuid.New()
		payable2 := uuid.New()
		payable3 := uuid.New()

		err := memo.ApplyToPayable(payable1, valueobject.NewMoneyCNY(decimal.NewFromFloat(700.00)), "First")
		require.NoError(t, err)

		err = memo.ApplyToPayable(payable2, valueobject.NewMoneyCNY(decimal.NewFromFloat(800.00)), "Second")
		require.NoError(t, err)
		assert.Equal(t, DebitMemoStatusPartial, memo.Status)

		err = memo.ApplyToPayable(payable3, valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00)), "Third")
		require.NoError(t, err)
		assert.Equal(t, DebitMemoStatusApplied, memo.Status)
		assert.Equal(t, 3, memo.ApplicationCount())
	})

	t.Run("exceeds remaining amount", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		payableID := uuid.New()

		err := memo.ApplyToPayable(payableID, valueobject.NewMoneyCNY(decimal.NewFromFloat(3000.00)), "Too much")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds remaining debit")
	})

	t.Run("cannot apply to voided memo", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		_ = memo.Void("Test void")

		err := memo.ApplyToPayable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(100.00)), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot apply debit memo")
	})
}

func TestDebitMemo_Void(t *testing.T) {
	t.Run("successful void", func(t *testing.T) {
		memo := createTestDebitMemo(t)

		err := memo.Void("Supplier rejected the return")

		require.NoError(t, err)
		assert.Equal(t, DebitMemoStatusVoided, memo.Status)
		assert.NotNil(t, memo.VoidedAt)
		assert.Equal(t, "Supplier rejected the return", memo.VoidReason)
		assert.True(t, memo.RemainingAmount.IsZero())
	})

	t.Run("cannot void with applications", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		_ = memo.ApplyToPayable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(100.00)), "")

		err := memo.Void("Try to void")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot void debit memo with existing applications")
	})

	t.Run("cannot void already applied memo", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		_ = memo.ApplyToPayable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(2000.00)), "")

		err := memo.Void("Try to void")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot void debit memo")
	})

	t.Run("empty void reason", func(t *testing.T) {
		memo := createTestDebitMemo(t)

		err := memo.Void("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Void reason is required")
	})
}

func TestDebitMemo_ReceiveRefund(t *testing.T) {
	t.Run("successful full refund", func(t *testing.T) {
		memo := createTestDebitMemo(t)

		err := memo.ReceiveRefund("BANK_TRANSFER")

		require.NoError(t, err)
		assert.Equal(t, DebitMemoStatusRefunded, memo.Status)
		assert.NotNil(t, memo.RefundReceivedAt)
		assert.Equal(t, "BANK_TRANSFER", memo.RefundMethod)
		assert.True(t, memo.RemainingAmount.IsZero())
	})

	t.Run("refund remaining after partial application", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		_ = memo.ApplyToPayable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(1500.00)), "")

		err := memo.ReceiveRefund("CHECK")

		require.NoError(t, err)
		assert.Equal(t, DebitMemoStatusRefunded, memo.Status)
		assert.True(t, memo.AppliedAmount.Equal(memo.TotalDebit))
	})

	t.Run("cannot receive refund for already applied memo", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		_ = memo.ApplyToPayable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(2000.00)), "")

		err := memo.ReceiveRefund("CASH")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot receive refund for debit memo")
	})

	t.Run("empty refund method", func(t *testing.T) {
		memo := createTestDebitMemo(t)

		err := memo.ReceiveRefund("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Refund method is required")
	})
}

func TestDebitMemo_SetOriginalPayable(t *testing.T) {
	t.Run("successful set", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		payableID := uuid.New()

		err := memo.SetOriginalPayable(payableID)

		require.NoError(t, err)
		assert.NotNil(t, memo.OriginalPayableID)
		assert.Equal(t, payableID, *memo.OriginalPayableID)
	})

	t.Run("empty payable ID", func(t *testing.T) {
		memo := createTestDebitMemo(t)

		err := memo.SetOriginalPayable(uuid.Nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payable ID cannot be empty")
	})
}

func TestDebitMemo_HelperMethods(t *testing.T) {
	memo := createTestDebitMemo(t)

	t.Run("status checks", func(t *testing.T) {
		assert.True(t, memo.IsPending())
		assert.False(t, memo.IsPartial())
		assert.False(t, memo.IsApplied())
		assert.False(t, memo.IsVoided())
		assert.False(t, memo.IsRefunded())
	})

	t.Run("applied percentage - zero applied", func(t *testing.T) {
		pct := memo.AppliedPercentage()
		assert.True(t, pct.IsZero())
	})

	t.Run("applied percentage - partial", func(t *testing.T) {
		_ = memo.ApplyToPayable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00)), "")
		pct := memo.AppliedPercentage()
		assert.True(t, pct.Equal(decimal.NewFromFloat(50.00)))
	})

	t.Run("money getters", func(t *testing.T) {
		memo := createTestDebitMemo(t)
		assert.True(t, memo.GetTotalDebitMoney().Amount().Equal(decimal.NewFromFloat(2000.00)))
		assert.True(t, memo.GetAppliedAmountMoney().Amount().IsZero())
		assert.True(t, memo.GetRemainingAmountMoney().Amount().Equal(decimal.NewFromFloat(2000.00)))
	})
}

func TestDebitMemoStatus(t *testing.T) {
	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, DebitMemoStatusPending.IsValid())
		assert.True(t, DebitMemoStatusApplied.IsValid())
		assert.True(t, DebitMemoStatusPartial.IsValid())
		assert.True(t, DebitMemoStatusVoided.IsValid())
		assert.True(t, DebitMemoStatusRefunded.IsValid())
		assert.False(t, DebitMemoStatus("INVALID").IsValid())
	})

	t.Run("IsTerminal", func(t *testing.T) {
		assert.False(t, DebitMemoStatusPending.IsTerminal())
		assert.False(t, DebitMemoStatusPartial.IsTerminal())
		assert.True(t, DebitMemoStatusApplied.IsTerminal())
		assert.True(t, DebitMemoStatusVoided.IsTerminal())
		assert.True(t, DebitMemoStatusRefunded.IsTerminal())
	})

	t.Run("CanApply", func(t *testing.T) {
		assert.True(t, DebitMemoStatusPending.CanApply())
		assert.True(t, DebitMemoStatusPartial.CanApply())
		assert.False(t, DebitMemoStatusApplied.CanApply())
		assert.False(t, DebitMemoStatusVoided.CanApply())
		assert.False(t, DebitMemoStatusRefunded.CanApply())
	})
}

// Helper function to create a test debit memo
func createTestDebitMemo(t *testing.T) *DebitMemo {
	memo, err := NewDebitMemo(
		uuid.New(),
		"DM-TEST-001",
		uuid.New(),
		"PR-TEST-001",
		uuid.New(),
		"PO-TEST-001",
		uuid.New(),
		"Test Supplier",
		valueobject.NewMoneyCNY(decimal.NewFromFloat(2000.00)),
		"Test return reason",
	)
	require.NoError(t, err)
	memo.ClearDomainEvents() // Clear events for cleaner tests
	return memo
}
