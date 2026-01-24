package finance

import (
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func createTestReceivable(t *testing.T) *AccountReceivable {
	tenantID := uuid.New()
	customerID := uuid.New()
	sourceID := uuid.New()
	totalAmount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ar, err := NewAccountReceivable(
		tenantID,
		"AR-2024-001",
		customerID,
		"Test Customer",
		SourceTypeSalesOrder,
		sourceID,
		"SO-2024-001",
		totalAmount,
		nil,
	)
	require.NoError(t, err)
	return ar
}

func createTestReceivableWithDueDate(t *testing.T, daysFromNow int) *AccountReceivable {
	ar := createTestReceivable(t)
	dueDate := time.Now().AddDate(0, 0, daysFromNow)
	ar.DueDate = &dueDate
	return ar
}

// ============================================
// ReceivableStatus Tests
// ============================================

func TestReceivableStatus_IsValid(t *testing.T) {
	tests := []struct {
		status  ReceivableStatus
		isValid bool
	}{
		{ReceivableStatusPending, true},
		{ReceivableStatusPartial, true},
		{ReceivableStatusPaid, true},
		{ReceivableStatusReversed, true},
		{ReceivableStatusCancelled, true},
		{ReceivableStatus("INVALID"), false},
		{ReceivableStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.status.IsValid())
		})
	}
}

func TestReceivableStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status     ReceivableStatus
		isTerminal bool
	}{
		{ReceivableStatusPending, false},
		{ReceivableStatusPartial, false},
		{ReceivableStatusPaid, true},
		{ReceivableStatusReversed, true},
		{ReceivableStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isTerminal, tt.status.IsTerminal())
		})
	}
}

func TestReceivableStatus_CanApplyPayment(t *testing.T) {
	tests := []struct {
		status   ReceivableStatus
		canApply bool
	}{
		{ReceivableStatusPending, true},
		{ReceivableStatusPartial, true},
		{ReceivableStatusPaid, false},
		{ReceivableStatusReversed, false},
		{ReceivableStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.canApply, tt.status.CanApplyPayment())
		})
	}
}

func TestReceivableStatus_String(t *testing.T) {
	assert.Equal(t, "PENDING", ReceivableStatusPending.String())
	assert.Equal(t, "PAID", ReceivableStatusPaid.String())
}

// ============================================
// SourceType Tests
// ============================================

func TestSourceType_IsValid(t *testing.T) {
	tests := []struct {
		sourceType SourceType
		isValid    bool
	}{
		{SourceTypeSalesOrder, true},
		{SourceTypeSalesReturn, true},
		{SourceTypeManual, true},
		{SourceType("INVALID"), false},
		{SourceType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.sourceType), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.sourceType.IsValid())
		})
	}
}

// ============================================
// NewAccountReceivable Tests
// ============================================

func TestNewAccountReceivable(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	sourceID := uuid.New()
	totalAmount := valueobject.NewMoneyCNYFromFloat(1500.00)
	dueDate := time.Now().AddDate(0, 0, 30)

	t.Run("creates receivable with valid inputs", func(t *testing.T) {
		ar, err := NewAccountReceivable(
			tenantID,
			"AR-2024-001",
			customerID,
			"Test Customer",
			SourceTypeSalesOrder,
			sourceID,
			"SO-2024-001",
			totalAmount,
			&dueDate,
		)
		require.NoError(t, err)
		require.NotNil(t, ar)

		assert.Equal(t, tenantID, ar.TenantID)
		assert.Equal(t, "AR-2024-001", ar.ReceivableNumber)
		assert.Equal(t, customerID, ar.CustomerID)
		assert.Equal(t, "Test Customer", ar.CustomerName)
		assert.Equal(t, SourceTypeSalesOrder, ar.SourceType)
		assert.Equal(t, sourceID, ar.SourceID)
		assert.Equal(t, "SO-2024-001", ar.SourceNumber)
		assert.True(t, ar.TotalAmount.Equal(decimal.NewFromFloat(1500.00)))
		assert.True(t, ar.PaidAmount.IsZero())
		assert.True(t, ar.OutstandingAmount.Equal(decimal.NewFromFloat(1500.00)))
		assert.Equal(t, ReceivableStatusPending, ar.Status)
		assert.NotNil(t, ar.DueDate)
		assert.Empty(t, ar.PaymentRecords)
		assert.NotEmpty(t, ar.ID)
		assert.Equal(t, 1, ar.GetVersion())
	})

	t.Run("creates receivable without due date", func(t *testing.T) {
		ar, err := NewAccountReceivable(
			tenantID,
			"AR-2024-002",
			customerID,
			"Test Customer",
			SourceTypeSalesOrder,
			sourceID,
			"SO-2024-002",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		assert.Nil(t, ar.DueDate)
	})

	t.Run("publishes AccountReceivableCreated event", func(t *testing.T) {
		ar, err := NewAccountReceivable(
			tenantID,
			"AR-2024-003",
			customerID,
			"Test Customer",
			SourceTypeSalesOrder,
			sourceID,
			"SO-2024-003",
			totalAmount,
			nil,
		)
		require.NoError(t, err)

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "AccountReceivableCreated", events[0].EventType())

		event, ok := events[0].(*AccountReceivableCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, ar.ID, event.ReceivableID)
		assert.Equal(t, ar.ReceivableNumber, event.ReceivableNumber)
		assert.Equal(t, ar.CustomerID, event.CustomerID)
	})

	t.Run("fails with empty receivable number", func(t *testing.T) {
		_, err := NewAccountReceivable(
			tenantID, "", customerID, "Test Customer",
			SourceTypeSalesOrder, sourceID, "SO-2024-001", totalAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Receivable number cannot be empty")
	})

	t.Run("fails with receivable number too long", func(t *testing.T) {
		longNumber := string(make([]byte, 51))
		_, err := NewAccountReceivable(
			tenantID, longNumber, customerID, "Test Customer",
			SourceTypeSalesOrder, sourceID, "SO-2024-001", totalAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with nil customer ID", func(t *testing.T) {
		_, err := NewAccountReceivable(
			tenantID, "AR-2024-001", uuid.Nil, "Test Customer",
			SourceTypeSalesOrder, sourceID, "SO-2024-001", totalAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Customer ID cannot be empty")
	})

	t.Run("fails with empty customer name", func(t *testing.T) {
		_, err := NewAccountReceivable(
			tenantID, "AR-2024-001", customerID, "",
			SourceTypeSalesOrder, sourceID, "SO-2024-001", totalAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Customer name cannot be empty")
	})

	t.Run("fails with invalid source type", func(t *testing.T) {
		_, err := NewAccountReceivable(
			tenantID, "AR-2024-001", customerID, "Test Customer",
			SourceType("INVALID"), sourceID, "SO-2024-001", totalAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Source type is not valid")
	})

	t.Run("fails with nil source ID", func(t *testing.T) {
		_, err := NewAccountReceivable(
			tenantID, "AR-2024-001", customerID, "Test Customer",
			SourceTypeSalesOrder, uuid.Nil, "SO-2024-001", totalAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Source ID cannot be empty")
	})

	t.Run("fails with empty source number", func(t *testing.T) {
		_, err := NewAccountReceivable(
			tenantID, "AR-2024-001", customerID, "Test Customer",
			SourceTypeSalesOrder, sourceID, "", totalAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Source number cannot be empty")
	})

	t.Run("fails with zero total amount", func(t *testing.T) {
		zeroAmount := valueobject.NewMoneyCNY(decimal.Zero)
		_, err := NewAccountReceivable(
			tenantID, "AR-2024-001", customerID, "Test Customer",
			SourceTypeSalesOrder, sourceID, "SO-2024-001", zeroAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Total amount must be positive")
	})

	t.Run("fails with negative total amount", func(t *testing.T) {
		negativeAmount := valueobject.NewMoneyCNYFromFloat(-100.00)
		_, err := NewAccountReceivable(
			tenantID, "AR-2024-001", customerID, "Test Customer",
			SourceTypeSalesOrder, sourceID, "SO-2024-001", negativeAmount, nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Total amount must be positive")
	})
}

// ============================================
// ApplyPayment Tests
// ============================================

func TestAccountReceivable_ApplyPayment(t *testing.T) {
	t.Run("applies full payment and marks as paid", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		voucherID := uuid.New()
		payment := valueobject.NewMoneyCNYFromFloat(1000.00)

		err := ar.ApplyPayment(payment, voucherID, "Full payment")
		require.NoError(t, err)

		assert.Equal(t, ReceivableStatusPaid, ar.Status)
		assert.True(t, ar.PaidAmount.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, ar.OutstandingAmount.IsZero())
		assert.NotNil(t, ar.PaidAt)
		assert.Len(t, ar.PaymentRecords, 1)
		assert.Equal(t, "Full payment", ar.PaymentRecords[0].Remark)
	})

	t.Run("publishes AccountReceivablePaid event on full payment", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		voucherID := uuid.New()
		payment := valueobject.NewMoneyCNYFromFloat(1000.00)
		ar.ApplyPayment(payment, voucherID, "")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "AccountReceivablePaid", events[0].EventType())

		event, ok := events[0].(*AccountReceivablePaidEvent)
		require.True(t, ok)
		assert.Equal(t, ar.ID, event.ReceivableID)
		assert.True(t, event.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
	})

	t.Run("applies partial payment and marks as partial", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		voucherID := uuid.New()
		payment := valueobject.NewMoneyCNYFromFloat(300.00)

		err := ar.ApplyPayment(payment, voucherID, "Partial payment 1")
		require.NoError(t, err)

		assert.Equal(t, ReceivableStatusPartial, ar.Status)
		assert.True(t, ar.PaidAmount.Equal(decimal.NewFromFloat(300.00)))
		assert.True(t, ar.OutstandingAmount.Equal(decimal.NewFromFloat(700.00)))
		assert.Nil(t, ar.PaidAt)
	})

	t.Run("publishes AccountReceivablePartiallyPaid event", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		voucherID := uuid.New()
		payment := valueobject.NewMoneyCNYFromFloat(300.00)
		ar.ApplyPayment(payment, voucherID, "")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "AccountReceivablePartiallyPaid", events[0].EventType())

		event, ok := events[0].(*AccountReceivablePartiallyPaidEvent)
		require.True(t, ok)
		assert.True(t, event.PaymentAmount.Equal(decimal.NewFromFloat(300.00)))
		assert.True(t, event.OutstandingAmount.Equal(decimal.NewFromFloat(700.00)))
	})

	t.Run("applies multiple partial payments leading to full payment", func(t *testing.T) {
		ar := createTestReceivable(t)

		voucherID1 := uuid.New()
		voucherID2 := uuid.New()
		voucherID3 := uuid.New()

		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), voucherID1, "")
		assert.Equal(t, ReceivableStatusPartial, ar.Status)

		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(400.00), voucherID2, "")
		assert.Equal(t, ReceivableStatusPartial, ar.Status)

		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), voucherID3, "")
		assert.Equal(t, ReceivableStatusPaid, ar.Status)

		assert.Len(t, ar.PaymentRecords, 3)
		assert.True(t, ar.OutstandingAmount.IsZero())
	})

	t.Run("fails with payment exceeding outstanding amount", func(t *testing.T) {
		ar := createTestReceivable(t)

		voucherID := uuid.New()
		payment := valueobject.NewMoneyCNYFromFloat(1500.00) // Exceeds 1000

		err := ar.ApplyPayment(payment, voucherID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds outstanding amount")
	})

	t.Run("fails with zero payment amount", func(t *testing.T) {
		ar := createTestReceivable(t)

		voucherID := uuid.New()
		payment := valueobject.NewMoneyCNY(decimal.Zero)

		err := ar.ApplyPayment(payment, voucherID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payment amount must be positive")
	})

	t.Run("fails with negative payment amount", func(t *testing.T) {
		ar := createTestReceivable(t)

		voucherID := uuid.New()
		payment := valueobject.NewMoneyCNYFromFloat(-100.00)

		err := ar.ApplyPayment(payment, voucherID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payment amount must be positive")
	})

	t.Run("fails with nil voucher ID", func(t *testing.T) {
		ar := createTestReceivable(t)

		payment := valueobject.NewMoneyCNYFromFloat(100.00)

		err := ar.ApplyPayment(payment, uuid.Nil, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Receipt voucher ID cannot be empty")
	})

	t.Run("fails when receivable is already paid", func(t *testing.T) {
		ar := createTestReceivable(t)
		voucherID := uuid.New()
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), voucherID, "")

		err := ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(100.00), uuid.New(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PAID status")
	})

	t.Run("fails when receivable is reversed", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.Reverse("Order cancelled")

		err := ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(100.00), uuid.New(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REVERSED status")
	})

	t.Run("fails when receivable is cancelled", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.Cancel("Not needed")

		err := ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(100.00), uuid.New(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CANCELLED status")
	})
}

// ============================================
// Reverse Tests
// ============================================

func TestAccountReceivable_Reverse(t *testing.T) {
	t.Run("reverses pending receivable", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		err := ar.Reverse("Sales return processed")
		require.NoError(t, err)

		assert.Equal(t, ReceivableStatusReversed, ar.Status)
		assert.NotNil(t, ar.ReversedAt)
		assert.Equal(t, "Sales return processed", ar.ReversalReason)
	})

	t.Run("reverses partial receivable", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")
		ar.ClearDomainEvents()

		err := ar.Reverse("Customer dispute")
		require.NoError(t, err)

		assert.Equal(t, ReceivableStatusReversed, ar.Status)
	})

	t.Run("publishes AccountReceivableReversed event", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		ar.Reverse("Test reversal")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "AccountReceivableReversed", events[0].EventType())

		event, ok := events[0].(*AccountReceivableReversedEvent)
		require.True(t, ok)
		assert.Equal(t, ReceivableStatusPending, event.PreviousStatus)
		assert.Equal(t, "Test reversal", event.ReversalReason)
	})

	t.Run("fails without reason", func(t *testing.T) {
		ar := createTestReceivable(t)

		err := ar.Reverse("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Reversal reason is required")
	})

	t.Run("fails when already paid", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "")

		err := ar.Reverse("Try to reverse")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PAID status")
	})

	t.Run("fails when already reversed", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.Reverse("First reversal")

		err := ar.Reverse("Second reversal")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REVERSED status")
	})

	t.Run("fails when already cancelled", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.Cancel("Cancelled")

		err := ar.Reverse("Try to reverse")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CANCELLED status")
	})
}

// ============================================
// Cancel Tests
// ============================================

func TestAccountReceivable_Cancel(t *testing.T) {
	t.Run("cancels pending receivable", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		err := ar.Cancel("Order voided")
		require.NoError(t, err)

		assert.Equal(t, ReceivableStatusCancelled, ar.Status)
		assert.NotNil(t, ar.CancelledAt)
		assert.Equal(t, "Order voided", ar.CancelReason)
		assert.True(t, ar.OutstandingAmount.IsZero())
	})

	t.Run("publishes AccountReceivableCancelled event", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()

		ar.Cancel("Test cancel")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "AccountReceivableCancelled", events[0].EventType())

		event, ok := events[0].(*AccountReceivableCancelledEvent)
		require.True(t, ok)
		assert.Equal(t, "Test cancel", event.CancelReason)
	})

	t.Run("fails without reason", func(t *testing.T) {
		ar := createTestReceivable(t)

		err := ar.Cancel("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cancel reason is required")
	})

	t.Run("fails when partial payment exists", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")

		err := ar.Cancel("Try to cancel")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel receivable with existing payments")
	})

	t.Run("fails when already paid", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "")

		err := ar.Cancel("Try to cancel")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PAID status")
	})

	t.Run("fails when already cancelled", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.Cancel("First cancel")

		err := ar.Cancel("Second cancel")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CANCELLED status")
	})
}

// ============================================
// SetDueDate Tests
// ============================================

func TestAccountReceivable_SetDueDate(t *testing.T) {
	t.Run("sets due date successfully", func(t *testing.T) {
		ar := createTestReceivable(t)
		newDate := time.Now().AddDate(0, 0, 60)

		err := ar.SetDueDate(&newDate)
		require.NoError(t, err)

		assert.NotNil(t, ar.DueDate)
		assert.Equal(t, newDate.Unix(), ar.DueDate.Unix())
	})

	t.Run("clears due date with nil", func(t *testing.T) {
		ar := createTestReceivableWithDueDate(t, 30)
		require.NotNil(t, ar.DueDate)

		err := ar.SetDueDate(nil)
		require.NoError(t, err)

		assert.Nil(t, ar.DueDate)
	})

	t.Run("fails when receivable is in terminal state", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "")

		newDate := time.Now().AddDate(0, 0, 60)
		err := ar.SetDueDate(&newDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "terminal state")
	})
}

// ============================================
// Helper Methods Tests
// ============================================

func TestAccountReceivable_HelperMethods(t *testing.T) {
	t.Run("GetTotalAmountMoney returns correct value", func(t *testing.T) {
		ar := createTestReceivable(t)
		money := ar.GetTotalAmountMoney()

		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(1000.00)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})

	t.Run("GetPaidAmountMoney returns correct value", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")

		money := ar.GetPaidAmountMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(300.00)))
	})

	t.Run("GetOutstandingAmountMoney returns correct value", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")

		money := ar.GetOutstandingAmountMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(700.00)))
	})

	t.Run("PaymentCount returns correct count", func(t *testing.T) {
		ar := createTestReceivable(t)
		assert.Equal(t, 0, ar.PaymentCount())

		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")
		assert.Equal(t, 1, ar.PaymentCount())

		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(200.00), uuid.New(), "")
		assert.Equal(t, 2, ar.PaymentCount())
	})

	t.Run("PaidPercentage returns correct percentage", func(t *testing.T) {
		ar := createTestReceivable(t)

		// No payment
		assert.True(t, ar.PaidPercentage().IsZero())

		// 30% paid
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")
		assert.True(t, ar.PaidPercentage().Equal(decimal.NewFromFloat(30.00)))

		// 50% paid
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(200.00), uuid.New(), "")
		assert.True(t, ar.PaidPercentage().Equal(decimal.NewFromFloat(50.00)))

		// 100% paid
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(500.00), uuid.New(), "")
		assert.True(t, ar.PaidPercentage().Equal(decimal.NewFromFloat(100.00)))
	})
}

// ============================================
// Status Helper Tests
// ============================================

func TestAccountReceivable_StatusHelpers(t *testing.T) {
	ar := createTestReceivable(t)

	assert.True(t, ar.IsPending())
	assert.False(t, ar.IsPartial())
	assert.False(t, ar.IsPaid())
	assert.False(t, ar.IsReversed())
	assert.False(t, ar.IsCancelled())

	// Partial payment
	ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")
	assert.False(t, ar.IsPending())
	assert.True(t, ar.IsPartial())
	assert.False(t, ar.IsPaid())

	// Full payment
	ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(700.00), uuid.New(), "")
	assert.False(t, ar.IsPartial())
	assert.True(t, ar.IsPaid())
}

// ============================================
// Overdue Tests
// ============================================

func TestAccountReceivable_Overdue(t *testing.T) {
	t.Run("IsOverdue returns false for no due date", func(t *testing.T) {
		ar := createTestReceivable(t)
		assert.Nil(t, ar.DueDate)
		assert.False(t, ar.IsOverdue())
	})

	t.Run("IsOverdue returns false for future due date", func(t *testing.T) {
		ar := createTestReceivableWithDueDate(t, 30) // 30 days in future
		assert.False(t, ar.IsOverdue())
	})

	t.Run("IsOverdue returns true for past due date", func(t *testing.T) {
		ar := createTestReceivableWithDueDate(t, -10) // 10 days ago
		assert.True(t, ar.IsOverdue())
	})

	t.Run("IsOverdue returns false for paid receivable", func(t *testing.T) {
		ar := createTestReceivableWithDueDate(t, -10)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "")
		assert.False(t, ar.IsOverdue())
	})

	t.Run("DaysOverdue returns correct count", func(t *testing.T) {
		ar := createTestReceivableWithDueDate(t, -5)  // 5 days ago
		assert.GreaterOrEqual(t, ar.DaysOverdue(), 4) // At least 4 days
		assert.LessOrEqual(t, ar.DaysOverdue(), 6)    // At most 6 days
	})

	t.Run("DaysOverdue returns 0 for non-overdue", func(t *testing.T) {
		ar := createTestReceivableWithDueDate(t, 30)
		assert.Equal(t, 0, ar.DaysOverdue())
	})
}

// ============================================
// SetRemark Tests
// ============================================

func TestAccountReceivable_SetRemark(t *testing.T) {
	ar := createTestReceivable(t)
	originalVersion := ar.GetVersion()

	ar.SetRemark("Important customer")

	assert.Equal(t, "Important customer", ar.Remark)
	assert.Equal(t, originalVersion+1, ar.GetVersion())
}

// ============================================
// PaymentRecord Tests
// ============================================

func TestPaymentRecord(t *testing.T) {
	t.Run("creates payment record correctly", func(t *testing.T) {
		receivableID := uuid.New()
		voucherID := uuid.New()
		amount := valueobject.NewMoneyCNYFromFloat(500.00)

		record := NewPaymentRecord(receivableID, voucherID, amount, "Test payment")

		assert.NotEqual(t, uuid.Nil, record.ID)
		assert.Equal(t, receivableID, record.ReceivableID)
		assert.Equal(t, voucherID, record.ReceiptVoucherID)
		assert.True(t, record.Amount.Equal(decimal.NewFromFloat(500.00)))
		assert.Equal(t, "Test payment", record.Remark)
		assert.False(t, record.AppliedAt.IsZero())
	})

	t.Run("GetAmountMoney returns correct value", func(t *testing.T) {
		record := NewPaymentRecord(uuid.New(), uuid.New(), valueobject.NewMoneyCNYFromFloat(750.50), "")

		money := record.GetAmountMoney()
		assert.True(t, money.Amount().Equal(decimal.NewFromFloat(750.50)))
		assert.Equal(t, valueobject.CNY, money.Currency())
	})
}

// ============================================
// Event Tests
// ============================================

func TestAccountReceivableEvents(t *testing.T) {
	t.Run("AccountReceivableCreatedEvent has correct fields", func(t *testing.T) {
		ar := createTestReceivable(t)
		events := ar.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*AccountReceivableCreatedEvent)
		require.True(t, ok)

		assert.Equal(t, ar.ID, event.ReceivableID)
		assert.Equal(t, ar.ReceivableNumber, event.ReceivableNumber)
		assert.Equal(t, ar.CustomerID, event.CustomerID)
		assert.Equal(t, ar.CustomerName, event.CustomerName)
		assert.Equal(t, ar.SourceType, event.SourceType)
		assert.Equal(t, ar.SourceID, event.SourceID)
		assert.Equal(t, ar.SourceNumber, event.SourceNumber)
		assert.True(t, event.TotalAmount.Equal(ar.TotalAmount))
		assert.Equal(t, ar.TenantID, event.TenantID())
		assert.Equal(t, "AccountReceivableCreated", event.EventType())
		assert.Equal(t, "AccountReceivable", event.AggregateType())
	})

	t.Run("AccountReceivablePaidEvent has correct fields", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*AccountReceivablePaidEvent)
		require.True(t, ok)

		assert.Equal(t, ar.ID, event.ReceivableID)
		assert.True(t, event.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
		assert.True(t, event.PaidAmount.Equal(decimal.NewFromFloat(1000.00)))
		assert.False(t, event.PaidAt.IsZero())
	})

	t.Run("AccountReceivablePartiallyPaidEvent has correct fields", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*AccountReceivablePartiallyPaidEvent)
		require.True(t, ok)

		assert.True(t, event.PaymentAmount.Equal(decimal.NewFromFloat(300.00)))
		assert.True(t, event.PaidAmount.Equal(decimal.NewFromFloat(300.00)))
		assert.True(t, event.OutstandingAmount.Equal(decimal.NewFromFloat(700.00)))
	})

	t.Run("AccountReceivableReversedEvent has correct fields", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "")
		ar.ClearDomainEvents()
		ar.Reverse("Return processed")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*AccountReceivableReversedEvent)
		require.True(t, ok)

		assert.Equal(t, ReceivableStatusPartial, event.PreviousStatus)
		assert.Equal(t, "Return processed", event.ReversalReason)
		assert.True(t, event.PaidAmount.Equal(decimal.NewFromFloat(300.00)))
		assert.True(t, event.OutstandingAmount.Equal(decimal.NewFromFloat(700.00)))
	})

	t.Run("AccountReceivableCancelledEvent has correct fields", func(t *testing.T) {
		ar := createTestReceivable(t)
		ar.ClearDomainEvents()
		ar.Cancel("Order voided")

		events := ar.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*AccountReceivableCancelledEvent)
		require.True(t, ok)

		assert.Equal(t, ar.ID, event.ReceivableID)
		assert.Equal(t, "Order voided", event.CancelReason)
		assert.True(t, event.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
	})
}

// ============================================
// Table Names Tests
// ============================================

func TestTableNames(t *testing.T) {
	t.Run("AccountReceivable table name", func(t *testing.T) {
		ar := &AccountReceivable{}
		assert.Equal(t, "account_receivables", ar.TableName())
	})

	t.Run("PaymentRecord table name", func(t *testing.T) {
		pr := &PaymentRecord{}
		assert.Equal(t, "receivable_payment_records", pr.TableName())
	})
}
