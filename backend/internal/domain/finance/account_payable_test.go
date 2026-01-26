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

// Test PayableStatus enum

func TestPayableStatus_String(t *testing.T) {
	tests := []struct {
		status   PayableStatus
		expected string
	}{
		{PayableStatusPending, "PENDING"},
		{PayableStatusPartial, "PARTIAL"},
		{PayableStatusPaid, "PAID"},
		{PayableStatusReversed, "REVERSED"},
		{PayableStatusCancelled, "CANCELLED"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.status.String())
		})
	}
}

func TestPayableStatus_IsValid(t *testing.T) {
	tests := []struct {
		status   PayableStatus
		expected bool
	}{
		{PayableStatusPending, true},
		{PayableStatusPartial, true},
		{PayableStatusPaid, true},
		{PayableStatusReversed, true},
		{PayableStatusCancelled, true},
		{PayableStatus("INVALID"), false},
		{PayableStatus(""), false},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.status.IsValid())
		})
	}
}

func TestPayableStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   PayableStatus
		expected bool
	}{
		{PayableStatusPending, false},
		{PayableStatusPartial, false},
		{PayableStatusPaid, true},
		{PayableStatusReversed, true},
		{PayableStatusCancelled, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.status.IsTerminal())
		})
	}
}

func TestPayableStatus_CanApplyPayment(t *testing.T) {
	tests := []struct {
		status   PayableStatus
		expected bool
	}{
		{PayableStatusPending, true},
		{PayableStatusPartial, true},
		{PayableStatusPaid, false},
		{PayableStatusReversed, false},
		{PayableStatusCancelled, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.status.CanApplyPayment())
		})
	}
}

// Test PayableSourceType enum

func TestPayableSourceType_IsValid(t *testing.T) {
	tests := []struct {
		sourceType PayableSourceType
		expected   bool
	}{
		{PayableSourceTypePurchaseOrder, true},
		{PayableSourceTypePurchaseReturn, true},
		{PayableSourceTypeManual, true},
		{PayableSourceType("INVALID"), false},
		{PayableSourceType(""), false},
	}

	for _, tc := range tests {
		t.Run(string(tc.sourceType), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.sourceType.IsValid())
		})
	}
}

// Test PayablePaymentRecord

func TestNewPayablePaymentRecord(t *testing.T) {
	payableID := uuid.New()
	voucherID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(100.50)

	record := NewPayablePaymentRecord(payableID, voucherID, amount, "Test payment")

	assert.NotEqual(t, uuid.Nil, record.ID)
	assert.Equal(t, payableID, record.PayableID)
	assert.Equal(t, voucherID, record.PaymentVoucherID)
	assert.True(t, record.Amount.Equal(decimal.NewFromFloat(100.50)))
	assert.Equal(t, "Test payment", record.Remark)
	assert.False(t, record.AppliedAt.IsZero())
}

func TestPayablePaymentRecord_GetAmountMoney(t *testing.T) {
	record := &PayablePaymentRecord{
		Amount: decimal.NewFromFloat(250.75),
	}

	money := record.GetAmountMoney()
	assert.True(t, money.Amount().Equal(decimal.NewFromFloat(250.75)))
}

// Test AccountPayable creation

func TestNewAccountPayable_ValidData(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"PO-2024-00001",
		amount,
		&dueDate,
	)

	require.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotEqual(t, uuid.Nil, ap.ID)
	assert.Equal(t, tenantID, ap.TenantID)
	assert.Equal(t, "AP-2024-00001", ap.PayableNumber)
	assert.Equal(t, supplierID, ap.SupplierID)
	assert.Equal(t, "Test Supplier", ap.SupplierName)
	assert.Equal(t, PayableSourceTypePurchaseOrder, ap.SourceType)
	assert.Equal(t, sourceID, ap.SourceID)
	assert.Equal(t, "PO-2024-00001", ap.SourceNumber)
	assert.True(t, ap.TotalAmount.Equal(decimal.NewFromFloat(1000.00)))
	assert.True(t, ap.PaidAmount.IsZero())
	assert.True(t, ap.OutstandingAmount.Equal(decimal.NewFromFloat(1000.00)))
	assert.Equal(t, PayableStatusPending, ap.Status)
	assert.NotNil(t, ap.DueDate)
	assert.Empty(t, ap.PaymentRecords)

	// Verify event was raised
	events := ap.GetDomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "AccountPayableCreated", events[0].EventType())
}

func TestNewAccountPayable_EmptyPayableNumber(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ap, err := NewAccountPayable(
		tenantID,
		"",
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Payable number cannot be empty")
}

func TestNewAccountPayable_TooLongPayableNumber(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)
	longNumber := "AP-" + string(make([]byte, 50))

	ap, err := NewAccountPayable(
		tenantID,
		longNumber,
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "cannot exceed 50 characters")
}

func TestNewAccountPayable_EmptySupplierID(t *testing.T) {
	tenantID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		uuid.Nil,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Supplier ID cannot be empty")
}

func TestNewAccountPayable_EmptySupplierName(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		supplierID,
		"",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Supplier name cannot be empty")
}

func TestNewAccountPayable_InvalidSourceType(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		supplierID,
		"Test Supplier",
		PayableSourceType("INVALID"),
		sourceID,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Source type is not valid")
}

func TestNewAccountPayable_EmptySourceID(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		uuid.Nil,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Source ID cannot be empty")
}

func TestNewAccountPayable_EmptySourceNumber(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Source number cannot be empty")
}

func TestNewAccountPayable_ZeroAmount(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNY(decimal.Zero)

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Total amount must be positive")
}

func TestNewAccountPayable_NegativeAmount(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	sourceID := uuid.New()
	amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(-100.00))

	ap, err := NewAccountPayable(
		tenantID,
		"AP-2024-00001",
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		sourceID,
		"PO-2024-00001",
		amount,
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, ap)
	assert.Contains(t, err.Error(), "Total amount must be positive")
}

// Test ApplyPayment

func TestAccountPayable_ApplyPayment_FullPayment(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	voucherID := uuid.New()
	payment := valueobject.NewMoneyCNYFromFloat(1000.00)

	err := ap.ApplyPayment(payment, voucherID, "Full payment")

	require.NoError(t, err)
	assert.Equal(t, PayableStatusPaid, ap.Status)
	assert.True(t, ap.PaidAmount.Equal(decimal.NewFromFloat(1000.00)))
	assert.True(t, ap.OutstandingAmount.IsZero())
	assert.NotNil(t, ap.PaidAt)
	assert.Len(t, ap.PaymentRecords, 1)
	assert.Equal(t, "Full payment", ap.PaymentRecords[0].Remark)

	// Verify paid event
	events := ap.GetDomainEvents()
	require.Len(t, events, 2) // Created + Paid
	assert.Equal(t, "AccountPayablePaid", events[1].EventType())
}

func TestAccountPayable_ApplyPayment_PartialPayment(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	voucherID := uuid.New()
	payment := valueobject.NewMoneyCNYFromFloat(400.00)

	err := ap.ApplyPayment(payment, voucherID, "Partial payment")

	require.NoError(t, err)
	assert.Equal(t, PayableStatusPartial, ap.Status)
	assert.True(t, ap.PaidAmount.Equal(decimal.NewFromFloat(400.00)))
	assert.True(t, ap.OutstandingAmount.Equal(decimal.NewFromFloat(600.00)))
	assert.Nil(t, ap.PaidAt)
	assert.Len(t, ap.PaymentRecords, 1)

	// Verify partial payment event
	events := ap.GetDomainEvents()
	require.Len(t, events, 2) // Created + PartiallyPaid
	assert.Equal(t, "AccountPayablePartiallyPaid", events[1].EventType())
}

func TestAccountPayable_ApplyPayment_MultiplePartialPayments(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	voucherID1 := uuid.New()
	voucherID2 := uuid.New()
	voucherID3 := uuid.New()

	// First payment
	err := ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), voucherID1, "First payment")
	require.NoError(t, err)
	assert.Equal(t, PayableStatusPartial, ap.Status)

	// Second payment
	err = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(400.00), voucherID2, "Second payment")
	require.NoError(t, err)
	assert.Equal(t, PayableStatusPartial, ap.Status)

	// Final payment
	err = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), voucherID3, "Final payment")
	require.NoError(t, err)
	assert.Equal(t, PayableStatusPaid, ap.Status)
	assert.True(t, ap.OutstandingAmount.IsZero())
	assert.NotNil(t, ap.PaidAt)
	assert.Len(t, ap.PaymentRecords, 3)
}

func TestAccountPayable_ApplyPayment_ExceedsOutstanding(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	voucherID := uuid.New()
	payment := valueobject.NewMoneyCNYFromFloat(1500.00)

	err := ap.ApplyPayment(payment, voucherID, "Overpayment")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds outstanding amount")
	assert.Equal(t, PayableStatusPending, ap.Status)
	assert.True(t, ap.PaidAmount.IsZero())
}

func TestAccountPayable_ApplyPayment_ZeroAmount(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	voucherID := uuid.New()
	payment := valueobject.NewMoneyCNY(decimal.Zero)

	err := ap.ApplyPayment(payment, voucherID, "Zero payment")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Payment amount must be positive")
}

func TestAccountPayable_ApplyPayment_NegativeAmount(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	voucherID := uuid.New()
	payment := valueobject.NewMoneyCNY(decimal.NewFromFloat(-100.00))

	err := ap.ApplyPayment(payment, voucherID, "Negative payment")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Payment amount must be positive")
}

func TestAccountPayable_ApplyPayment_EmptyVoucherID(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	payment := valueobject.NewMoneyCNYFromFloat(500.00)

	err := ap.ApplyPayment(payment, uuid.Nil, "No voucher")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Payment voucher ID cannot be empty")
}

func TestAccountPayable_ApplyPayment_ToPaidPayable(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	voucherID := uuid.New()

	// Pay in full
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), voucherID, "Full payment")

	// Try to pay again
	err := ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(100.00), uuid.New(), "Extra payment")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot apply payment")
}

func TestAccountPayable_ApplyPayment_ToReversedPayable(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.Reverse("Test reversal")

	err := ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(500.00), uuid.New(), "Payment")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot apply payment")
}

func TestAccountPayable_ApplyPayment_ToCancelledPayable(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.Cancel("Test cancellation")

	err := ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(500.00), uuid.New(), "Payment")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot apply payment")
}

// Test Reverse

func TestAccountPayable_Reverse_FromPending(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	err := ap.Reverse("Order cancelled")

	require.NoError(t, err)
	assert.Equal(t, PayableStatusReversed, ap.Status)
	assert.NotNil(t, ap.ReversedAt)
	assert.Equal(t, "Order cancelled", ap.ReversalReason)

	// Verify event
	events := ap.GetDomainEvents()
	require.Len(t, events, 2) // Created + Reversed
	reversedEvent := events[1].(*AccountPayableReversedEvent)
	assert.Equal(t, "AccountPayableReversed", reversedEvent.EventType())
	assert.Equal(t, PayableStatusPending, reversedEvent.PreviousStatus)
}

func TestAccountPayable_Reverse_FromPartial(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "Partial")

	err := ap.Reverse("Order returned")

	require.NoError(t, err)
	assert.Equal(t, PayableStatusReversed, ap.Status)

	events := ap.GetDomainEvents()
	reversedEvent := events[len(events)-1].(*AccountPayableReversedEvent)
	assert.Equal(t, PayableStatusPartial, reversedEvent.PreviousStatus)
}

func TestAccountPayable_Reverse_FromPaid(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "Full")

	err := ap.Reverse("Test reversal")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot reverse payable in PAID status")
}

func TestAccountPayable_Reverse_AlreadyReversed(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.Reverse("First reversal")

	err := ap.Reverse("Second reversal")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot reverse payable in REVERSED status")
}

func TestAccountPayable_Reverse_EmptyReason(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	err := ap.Reverse("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Reversal reason is required")
}

// Test Cancel

func TestAccountPayable_Cancel_FromPending(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	err := ap.Cancel("No longer needed")

	require.NoError(t, err)
	assert.Equal(t, PayableStatusCancelled, ap.Status)
	assert.NotNil(t, ap.CancelledAt)
	assert.Equal(t, "No longer needed", ap.CancelReason)
	assert.True(t, ap.OutstandingAmount.IsZero())

	// Verify event
	events := ap.GetDomainEvents()
	require.Len(t, events, 2) // Created + Cancelled
	assert.Equal(t, "AccountPayableCancelled", events[1].EventType())
}

func TestAccountPayable_Cancel_WithPartialPayment(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "Partial")

	err := ap.Cancel("Try to cancel")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot cancel payable with existing payments")
}

func TestAccountPayable_Cancel_AlreadyPaid(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "Full")

	err := ap.Cancel("Try to cancel")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot cancel payable in PAID status")
}

func TestAccountPayable_Cancel_EmptyReason(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	err := ap.Cancel("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cancel reason is required")
}

// Test SetDueDate

func TestAccountPayable_SetDueDate(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	newDueDate := time.Now().Add(60 * 24 * time.Hour)

	err := ap.SetDueDate(&newDueDate)

	require.NoError(t, err)
	assert.NotNil(t, ap.DueDate)
	assert.Equal(t, newDueDate.Unix(), ap.DueDate.Unix())
}

func TestAccountPayable_SetDueDate_ToNil(t *testing.T) {
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	ap := createTestPayableWithDueDate(t, 1000.00, &dueDate)

	err := ap.SetDueDate(nil)

	require.NoError(t, err)
	assert.Nil(t, ap.DueDate)
}

func TestAccountPayable_SetDueDate_OnTerminalState(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	_ = ap.Cancel("Cancelled")
	newDueDate := time.Now().Add(60 * 24 * time.Hour)

	err := ap.SetDueDate(&newDueDate)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot modify due date")
}

// Test SetRemark

func TestAccountPayable_SetRemark(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	originalVersion := ap.Version

	ap.SetRemark("New remark")

	assert.Equal(t, "New remark", ap.Remark)
	assert.Greater(t, ap.Version, originalVersion)
}

// Test Helper methods

func TestAccountPayable_GetAmountMethods(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	assert.True(t, ap.GetTotalAmountMoney().Amount().Equal(decimal.NewFromFloat(1000.00)))
	assert.True(t, ap.GetPaidAmountMoney().Amount().IsZero())
	assert.True(t, ap.GetOutstandingAmountMoney().Amount().Equal(decimal.NewFromFloat(1000.00)))
}

func TestAccountPayable_StatusChecks(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	assert.True(t, ap.IsPending())
	assert.False(t, ap.IsPartial())
	assert.False(t, ap.IsPaid())
	assert.False(t, ap.IsReversed())
	assert.False(t, ap.IsCancelled())

	// Make partial
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(500.00), uuid.New(), "Partial")
	assert.False(t, ap.IsPending())
	assert.True(t, ap.IsPartial())

	// Make paid
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(500.00), uuid.New(), "Final")
	assert.True(t, ap.IsPaid())
}

func TestAccountPayable_IsOverdue_NotOverdue(t *testing.T) {
	dueDate := time.Now().Add(30 * 24 * time.Hour) // 30 days in future
	ap := createTestPayableWithDueDate(t, 1000.00, &dueDate)

	assert.False(t, ap.IsOverdue())
	assert.Equal(t, 0, ap.DaysOverdue())
}

func TestAccountPayable_IsOverdue_Overdue(t *testing.T) {
	dueDate := time.Now().Add(-10 * 24 * time.Hour) // 10 days ago
	ap := createTestPayableWithDueDate(t, 1000.00, &dueDate)

	assert.True(t, ap.IsOverdue())
	assert.GreaterOrEqual(t, ap.DaysOverdue(), 9)
}

func TestAccountPayable_IsOverdue_NoDueDate(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	assert.False(t, ap.IsOverdue())
	assert.Equal(t, 0, ap.DaysOverdue())
}

func TestAccountPayable_IsOverdue_PaidPayableNotOverdue(t *testing.T) {
	dueDate := time.Now().Add(-10 * 24 * time.Hour) // 10 days ago
	ap := createTestPayableWithDueDate(t, 1000.00, &dueDate)
	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(1000.00), uuid.New(), "Full")

	assert.False(t, ap.IsOverdue()) // Paid payables are not overdue
	assert.Equal(t, 0, ap.DaysOverdue())
}

func TestAccountPayable_PaymentCount(t *testing.T) {
	ap := createTestPayable(t, 1000.00)
	assert.Equal(t, 0, ap.PaymentCount())

	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "First")
	assert.Equal(t, 1, ap.PaymentCount())

	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(300.00), uuid.New(), "Second")
	assert.Equal(t, 2, ap.PaymentCount())
}

func TestAccountPayable_PaidPercentage(t *testing.T) {
	ap := createTestPayable(t, 1000.00)

	assert.True(t, ap.PaidPercentage().Equal(decimal.NewFromInt(0)))

	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(250.00), uuid.New(), "25%")
	assert.True(t, ap.PaidPercentage().Equal(decimal.NewFromInt(25)))

	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(500.00), uuid.New(), "50% more")
	assert.True(t, ap.PaidPercentage().Equal(decimal.NewFromInt(75)))

	_ = ap.ApplyPayment(valueobject.NewMoneyCNYFromFloat(250.00), uuid.New(), "Final 25%")
	assert.True(t, ap.PaidPercentage().Equal(decimal.NewFromInt(100)))
}

// Helper functions

func createTestPayable(t *testing.T, amount float64) *AccountPayable {
	t.Helper()
	return createTestPayableWithDueDate(t, amount, nil)
}

func createTestPayableWithDueDate(t *testing.T, amount float64, dueDate *time.Time) *AccountPayable {
	t.Helper()
	ap, err := NewAccountPayable(
		uuid.New(),
		"AP-TEST-00001",
		uuid.New(),
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		uuid.New(),
		"PO-TEST-00001",
		valueobject.NewMoneyCNYFromFloat(amount),
		dueDate,
	)
	require.NoError(t, err)
	return ap
}
