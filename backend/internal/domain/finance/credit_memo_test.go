package finance

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreditMemo(t *testing.T) {
	tenantID := uuid.New()
	salesReturnID := uuid.New()
	salesOrderID := uuid.New()
	customerID := uuid.New()
	totalCredit := valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00))

	t.Run("successful creation", func(t *testing.T) {
		memo, err := NewCreditMemo(
			tenantID,
			"CM-001",
			salesReturnID,
			"SR-001",
			salesOrderID,
			"SO-001",
			customerID,
			"Test Customer",
			totalCredit,
			"Defective goods returned",
		)

		require.NoError(t, err)
		assert.NotNil(t, memo)
		assert.Equal(t, "CM-001", memo.MemoNumber)
		assert.Equal(t, salesReturnID, memo.SalesReturnID)
		assert.Equal(t, salesOrderID, memo.SalesOrderID)
		assert.Equal(t, customerID, memo.CustomerID)
		assert.Equal(t, "Test Customer", memo.CustomerName)
		assert.Equal(t, CreditMemoStatusPending, memo.Status)
		assert.True(t, memo.TotalCredit.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, memo.RemainingAmount.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, memo.AppliedAmount.IsZero())
		assert.Len(t, memo.GetDomainEvents(), 1)
	})

	t.Run("empty memo number", func(t *testing.T) {
		_, err := NewCreditMemo(
			tenantID,
			"",
			salesReturnID,
			"SR-001",
			salesOrderID,
			"SO-001",
			customerID,
			"Test Customer",
			totalCredit,
			"reason",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Memo number cannot be empty")
	})

	t.Run("empty sales return ID", func(t *testing.T) {
		_, err := NewCreditMemo(
			tenantID,
			"CM-001",
			uuid.Nil,
			"SR-001",
			salesOrderID,
			"SO-001",
			customerID,
			"Test Customer",
			totalCredit,
			"reason",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Sales return ID cannot be empty")
	})

	t.Run("zero total credit", func(t *testing.T) {
		_, err := NewCreditMemo(
			tenantID,
			"CM-001",
			salesReturnID,
			"SR-001",
			salesOrderID,
			"SO-001",
			customerID,
			"Test Customer",
			valueobject.NewMoneyCNY(decimal.Zero),
			"reason",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Total credit must be positive")
	})
}

func TestCreditMemo_AddItem(t *testing.T) {
	memo := createTestCreditMemo(t)

	t.Run("successful add item", func(t *testing.T) {
		item, err := memo.AddItem(
			memo.SalesReturnID,
			uuid.New(),
			uuid.New(),
			"Test Product",
			"PROD-001",
			"PCS",
			decimal.NewFromInt(5),
			valueobject.NewMoneyCNY(decimal.NewFromFloat(100.00)),
			"Defective",
		)

		require.NoError(t, err)
		assert.NotNil(t, item)
		assert.Equal(t, "Test Product", item.ProductName)
		assert.True(t, item.CreditAmount.Equal(decimal.NewFromFloat(500.00)))
		assert.Equal(t, 1, memo.ItemCount())
	})

	t.Run("cannot add to non-pending memo", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		memo.Status = CreditMemoStatusApplied

		_, err := memo.AddItem(
			memo.SalesReturnID,
			uuid.New(),
			uuid.New(),
			"Test Product",
			"PROD-001",
			"PCS",
			decimal.NewFromInt(5),
			valueobject.NewMoneyCNY(decimal.NewFromFloat(100.00)),
			"Defective",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot add items to a non-pending credit memo")
	})
}

func TestCreditMemo_ApplyToReceivable(t *testing.T) {
	t.Run("successful full application", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		receivableID := uuid.New()

		err := memo.ApplyToReceivable(receivableID, valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00)), "Apply to outstanding invoice")

		require.NoError(t, err)
		assert.Equal(t, CreditMemoStatusApplied, memo.Status)
		assert.True(t, memo.AppliedAmount.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, memo.RemainingAmount.IsZero())
		assert.Equal(t, 1, memo.ApplicationCount())
		assert.NotNil(t, memo.AppliedAt)
	})

	t.Run("successful partial application", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		receivableID := uuid.New()

		err := memo.ApplyToReceivable(receivableID, valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00)), "Partial application")

		require.NoError(t, err)
		assert.Equal(t, CreditMemoStatusPartial, memo.Status)
		assert.True(t, memo.AppliedAmount.Equal(decimal.NewFromFloat(500.00)))
		assert.True(t, memo.RemainingAmount.Equal(decimal.NewFromFloat(500.00)))
	})

	t.Run("multiple applications until fully applied", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		receivable1 := uuid.New()
		receivable2 := uuid.New()

		err := memo.ApplyToReceivable(receivable1, valueobject.NewMoneyCNY(decimal.NewFromFloat(600.00)), "First")
		require.NoError(t, err)
		assert.Equal(t, CreditMemoStatusPartial, memo.Status)

		err = memo.ApplyToReceivable(receivable2, valueobject.NewMoneyCNY(decimal.NewFromFloat(400.00)), "Second")
		require.NoError(t, err)
		assert.Equal(t, CreditMemoStatusApplied, memo.Status)
		assert.Equal(t, 2, memo.ApplicationCount())
	})

	t.Run("exceeds remaining amount", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		receivableID := uuid.New()

		err := memo.ApplyToReceivable(receivableID, valueobject.NewMoneyCNY(decimal.NewFromFloat(1500.00)), "Too much")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds remaining credit")
	})

	t.Run("cannot apply to voided memo", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		_ = memo.Void("Test void")

		err := memo.ApplyToReceivable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(100.00)), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot apply credit memo")
	})
}

func TestCreditMemo_Void(t *testing.T) {
	t.Run("successful void", func(t *testing.T) {
		memo := createTestCreditMemo(t)

		err := memo.Void("Customer changed mind")

		require.NoError(t, err)
		assert.Equal(t, CreditMemoStatusVoided, memo.Status)
		assert.NotNil(t, memo.VoidedAt)
		assert.Equal(t, "Customer changed mind", memo.VoidReason)
		assert.True(t, memo.RemainingAmount.IsZero())
	})

	t.Run("cannot void with applications", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		_ = memo.ApplyToReceivable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(100.00)), "")

		err := memo.Void("Try to void")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot void credit memo with existing applications")
	})

	t.Run("cannot void already applied memo", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		_ = memo.ApplyToReceivable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00)), "")

		err := memo.Void("Try to void")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot void credit memo")
	})

	t.Run("empty void reason", func(t *testing.T) {
		memo := createTestCreditMemo(t)

		err := memo.Void("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Void reason is required")
	})
}

func TestCreditMemo_Refund(t *testing.T) {
	t.Run("successful full refund", func(t *testing.T) {
		memo := createTestCreditMemo(t)

		err := memo.Refund("BANK_TRANSFER")

		require.NoError(t, err)
		assert.Equal(t, CreditMemoStatusRefunded, memo.Status)
		assert.NotNil(t, memo.RefundedAt)
		assert.Equal(t, "BANK_TRANSFER", memo.RefundMethod)
		assert.True(t, memo.RemainingAmount.IsZero())
	})

	t.Run("refund remaining after partial application", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		_ = memo.ApplyToReceivable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(600.00)), "")

		err := memo.Refund("CASH")

		require.NoError(t, err)
		assert.Equal(t, CreditMemoStatusRefunded, memo.Status)
		// AppliedAmount should now equal TotalCredit since remaining is refunded
		assert.True(t, memo.AppliedAmount.Equal(memo.TotalCredit))
	})

	t.Run("cannot refund already applied memo", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		_ = memo.ApplyToReceivable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00)), "")

		err := memo.Refund("CASH")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot refund credit memo")
	})

	t.Run("empty refund method", func(t *testing.T) {
		memo := createTestCreditMemo(t)

		err := memo.Refund("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Refund method is required")
	})
}

func TestCreditMemo_HelperMethods(t *testing.T) {
	memo := createTestCreditMemo(t)

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
		_ = memo.ApplyToReceivable(uuid.New(), valueobject.NewMoneyCNY(decimal.NewFromFloat(250.00)), "")
		pct := memo.AppliedPercentage()
		assert.True(t, pct.Equal(decimal.NewFromFloat(25.00)))
	})

	t.Run("money getters", func(t *testing.T) {
		memo := createTestCreditMemo(t)
		assert.True(t, memo.GetTotalCreditMoney().Amount().Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, memo.GetAppliedAmountMoney().Amount().IsZero())
		assert.True(t, memo.GetRemainingAmountMoney().Amount().Equal(decimal.NewFromFloat(1000.00)))
	})
}

func TestCreditMemoStatus(t *testing.T) {
	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, CreditMemoStatusPending.IsValid())
		assert.True(t, CreditMemoStatusApplied.IsValid())
		assert.True(t, CreditMemoStatusPartial.IsValid())
		assert.True(t, CreditMemoStatusVoided.IsValid())
		assert.True(t, CreditMemoStatusRefunded.IsValid())
		assert.False(t, CreditMemoStatus("INVALID").IsValid())
	})

	t.Run("IsTerminal", func(t *testing.T) {
		assert.False(t, CreditMemoStatusPending.IsTerminal())
		assert.False(t, CreditMemoStatusPartial.IsTerminal())
		assert.True(t, CreditMemoStatusApplied.IsTerminal())
		assert.True(t, CreditMemoStatusVoided.IsTerminal())
		assert.True(t, CreditMemoStatusRefunded.IsTerminal())
	})

	t.Run("CanApply", func(t *testing.T) {
		assert.True(t, CreditMemoStatusPending.CanApply())
		assert.True(t, CreditMemoStatusPartial.CanApply())
		assert.False(t, CreditMemoStatusApplied.CanApply())
		assert.False(t, CreditMemoStatusVoided.CanApply())
		assert.False(t, CreditMemoStatusRefunded.CanApply())
	})
}

// Helper function to create a test credit memo
func createTestCreditMemo(t *testing.T) *CreditMemo {
	memo, err := NewCreditMemo(
		uuid.New(),
		"CM-TEST-001",
		uuid.New(),
		"SR-TEST-001",
		uuid.New(),
		"SO-TEST-001",
		uuid.New(),
		"Test Customer",
		valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00)),
		"Test return reason",
	)
	require.NoError(t, err)
	memo.ClearDomainEvents() // Clear events for cleaner tests
	return memo
}
