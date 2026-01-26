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
func createTestPaymentVoucher(t *testing.T) *PaymentVoucher {
	tenantID := uuid.New()
	supplierID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	pv, err := NewPaymentVoucher(
		tenantID,
		"PV-2024-001",
		supplierID,
		"Test Supplier",
		amount,
		PaymentMethodBankTransfer,
		time.Now(),
	)
	require.NoError(t, err)
	return pv
}

func createConfirmedPaymentVoucher(t *testing.T) *PaymentVoucher {
	pv := createTestPaymentVoucher(t)
	userID := uuid.New()
	err := pv.Confirm(userID)
	require.NoError(t, err)
	return pv
}

// ============================================
// NewPaymentVoucher Tests
// ============================================

func TestNewPaymentVoucher(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1500.00)
	paymentDate := time.Now()

	t.Run("creates voucher with valid inputs", func(t *testing.T) {
		pv, err := NewPaymentVoucher(
			tenantID,
			"PV-2024-001",
			supplierID,
			"Test Supplier",
			amount,
			PaymentMethodBankTransfer,
			paymentDate,
		)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, pv.ID)
		assert.Equal(t, tenantID, pv.TenantID)
		assert.Equal(t, "PV-2024-001", pv.VoucherNumber)
		assert.Equal(t, supplierID, pv.SupplierID)
		assert.Equal(t, "Test Supplier", pv.SupplierName)
		assert.True(t, amount.Amount().Equal(pv.Amount))
		assert.True(t, decimal.Zero.Equal(pv.AllocatedAmount))
		assert.True(t, amount.Amount().Equal(pv.UnallocatedAmount))
		assert.Equal(t, PaymentMethodBankTransfer, pv.PaymentMethod)
		assert.Equal(t, VoucherStatusDraft, pv.Status)
		assert.Equal(t, paymentDate.Unix(), pv.PaymentDate.Unix())
		assert.Empty(t, pv.Allocations)
		assert.NotEmpty(t, pv.GetDomainEvents())
	})

	t.Run("fails with empty voucher number", func(t *testing.T) {
		_, err := NewPaymentVoucher(tenantID, "", supplierID, "Test", amount, PaymentMethodCash, paymentDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Voucher number cannot be empty")
	})

	t.Run("fails with too long voucher number", func(t *testing.T) {
		longNumber := make([]byte, 51)
		for i := range longNumber {
			longNumber[i] = 'A'
		}
		_, err := NewPaymentVoucher(tenantID, string(longNumber), supplierID, "Test", amount, PaymentMethodCash, paymentDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with empty supplier ID", func(t *testing.T) {
		_, err := NewPaymentVoucher(tenantID, "PV-001", uuid.Nil, "Test", amount, PaymentMethodCash, paymentDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Supplier ID cannot be empty")
	})

	t.Run("fails with empty supplier name", func(t *testing.T) {
		_, err := NewPaymentVoucher(tenantID, "PV-001", supplierID, "", amount, PaymentMethodCash, paymentDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Supplier name cannot be empty")
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		zeroAmount := valueobject.NewMoneyCNY(decimal.Zero)
		_, err := NewPaymentVoucher(tenantID, "PV-001", supplierID, "Test", zeroAmount, PaymentMethodCash, paymentDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Amount must be positive")
	})

	t.Run("fails with negative amount", func(t *testing.T) {
		negativeAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(-100))
		_, err := NewPaymentVoucher(tenantID, "PV-001", supplierID, "Test", negativeAmount, PaymentMethodCash, paymentDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Amount must be positive")
	})

	t.Run("fails with invalid payment method", func(t *testing.T) {
		_, err := NewPaymentVoucher(tenantID, "PV-001", supplierID, "Test", amount, PaymentMethod("BITCOIN"), paymentDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payment method is not valid")
	})

	t.Run("fails with zero payment date", func(t *testing.T) {
		_, err := NewPaymentVoucher(tenantID, "PV-001", supplierID, "Test", amount, PaymentMethodCash, time.Time{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payment date is required")
	})
}

// ============================================
// Confirm Tests
// ============================================

func TestPaymentVoucher_Confirm(t *testing.T) {
	userID := uuid.New()

	t.Run("confirms draft voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		pv.ClearDomainEvents()

		err := pv.Confirm(userID)
		require.NoError(t, err)
		assert.Equal(t, VoucherStatusConfirmed, pv.Status)
		assert.NotNil(t, pv.ConfirmedAt)
		assert.Equal(t, userID, *pv.ConfirmedBy)
		assert.Len(t, pv.GetDomainEvents(), 1)
	})

	t.Run("fails to confirm already confirmed voucher", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)

		err := pv.Confirm(userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot confirm voucher")
	})

	t.Run("fails to confirm cancelled voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		_ = pv.Cancel(userID, "Test cancel")

		err := pv.Confirm(userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot confirm voucher")
	})

	t.Run("fails with empty user ID", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)

		err := pv.Confirm(uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Confirming user ID is required")
	})
}

// ============================================
// AllocateToPayable Tests
// ============================================

func TestPaymentVoucher_AllocateToPayable(t *testing.T) {
	payableID := uuid.New()

	t.Run("allocates partial amount to payable", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t) // 1000.00
		pv.ClearDomainEvents()
		allocAmount := valueobject.NewMoneyCNYFromFloat(400.00)

		allocation, err := pv.AllocateToPayable(payableID, "AP-001", allocAmount, "Partial payment")
		require.NoError(t, err)
		assert.NotNil(t, allocation)
		assert.Equal(t, pv.ID, allocation.PaymentVoucherID)
		assert.Equal(t, payableID, allocation.PayableID)
		assert.Equal(t, "AP-001", allocation.PayableNumber)
		assert.True(t, allocAmount.Amount().Equal(allocation.Amount))

		assert.True(t, decimal.NewFromFloat(400.00).Equal(pv.AllocatedAmount))
		assert.True(t, decimal.NewFromFloat(600.00).Equal(pv.UnallocatedAmount))
		assert.Equal(t, VoucherStatusConfirmed, pv.Status) // Still confirmed, not fully allocated
		assert.Len(t, pv.GetDomainEvents(), 1)
	})

	t.Run("allocates full amount and changes status to ALLOCATED", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t) // 1000.00
		allocAmount := valueobject.NewMoneyCNYFromFloat(1000.00)

		_, err := pv.AllocateToPayable(payableID, "AP-001", allocAmount, "Full payment")
		require.NoError(t, err)

		assert.True(t, decimal.NewFromFloat(1000.00).Equal(pv.AllocatedAmount))
		assert.True(t, decimal.Zero.Equal(pv.UnallocatedAmount))
		assert.Equal(t, VoucherStatusAllocated, pv.Status)
		assert.True(t, pv.IsFullyAllocated())
	})

	t.Run("allocates to multiple payables", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t) // 1000.00
		payableID2 := uuid.New()

		_, err := pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(400.00), "")
		require.NoError(t, err)

		_, err = pv.AllocateToPayable(payableID2, "AP-002", valueobject.NewMoneyCNYFromFloat(600.00), "")
		require.NoError(t, err)

		assert.Equal(t, 2, pv.AllocationCount())
		assert.True(t, pv.IsFullyAllocated())
		assert.Equal(t, VoucherStatusAllocated, pv.Status)
	})

	t.Run("fails to allocate draft voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t) // Draft status

		_, err := pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot allocate voucher")
	})

	t.Run("fails to allocate cancelled voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		userID := uuid.New()
		_ = pv.Cancel(userID, "Test")

		_, err := pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot allocate voucher")
	})

	t.Run("fails with empty payable ID", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)

		_, err := pv.AllocateToPayable(uuid.Nil, "AP-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payable ID cannot be empty")
	})

	t.Run("fails with empty payable number", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)

		_, err := pv.AllocateToPayable(payableID, "", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payable number is required")
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)

		_, err := pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNY(decimal.Zero), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Allocation amount must be positive")
	})

	t.Run("fails when allocation exceeds unallocated amount", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t) // 1000.00

		_, err := pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(1500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds unallocated amount")
	})

	t.Run("fails when allocating to same payable twice", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t) // 1000.00

		_, err := pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(300.00), "")
		require.NoError(t, err)

		_, err = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(300.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Already allocated to payable")
	})
}

// ============================================
// Cancel Tests
// ============================================

func TestPaymentVoucher_Cancel(t *testing.T) {
	userID := uuid.New()

	t.Run("cancels draft voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		pv.ClearDomainEvents()

		err := pv.Cancel(userID, "Supplier requested cancellation")
		require.NoError(t, err)
		assert.Equal(t, VoucherStatusCancelled, pv.Status)
		assert.NotNil(t, pv.CancelledAt)
		assert.Equal(t, userID, *pv.CancelledBy)
		assert.Equal(t, "Supplier requested cancellation", pv.CancelReason)
		assert.Len(t, pv.GetDomainEvents(), 1)
	})

	t.Run("cancels confirmed voucher without allocations", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)
		pv.ClearDomainEvents()

		err := pv.Cancel(userID, "Duplicate entry")
		require.NoError(t, err)
		assert.Equal(t, VoucherStatusCancelled, pv.Status)
	})

	t.Run("fails to cancel voucher with allocations", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)
		payableID := uuid.New()
		_, _ = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(500.00), "")

		err := pv.Cancel(userID, "Test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel voucher with existing allocations")
	})

	t.Run("fails to cancel already cancelled voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		_ = pv.Cancel(userID, "First cancel")

		err := pv.Cancel(userID, "Second cancel")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel voucher")
	})

	t.Run("fails with empty user ID", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)

		err := pv.Cancel(uuid.Nil, "Test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cancelling user ID is required")
	})

	t.Run("fails with empty reason", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)

		err := pv.Cancel(userID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cancel reason is required")
	})
}

// ============================================
// SetPaymentReference Tests
// ============================================

func TestPaymentVoucher_SetPaymentReference(t *testing.T) {
	t.Run("sets payment reference on draft voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)

		err := pv.SetPaymentReference("TXN-123456")
		require.NoError(t, err)
		assert.Equal(t, "TXN-123456", pv.PaymentReference)
	})

	t.Run("sets payment reference on confirmed voucher", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)

		err := pv.SetPaymentReference("BANK-REF-789")
		require.NoError(t, err)
		assert.Equal(t, "BANK-REF-789", pv.PaymentReference)
	})

	t.Run("fails on cancelled voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		userID := uuid.New()
		_ = pv.Cancel(userID, "Test")

		err := pv.SetPaymentReference("REF-001")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot modify voucher in terminal state")
	})

	t.Run("fails with too long reference", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		longRef := make([]byte, 101)
		for i := range longRef {
			longRef[i] = 'A'
		}

		err := pv.SetPaymentReference(string(longRef))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payment reference cannot exceed 100 characters")
	})
}

// ============================================
// SetRemark Tests
// ============================================

func TestPaymentVoucher_SetRemark(t *testing.T) {
	t.Run("sets remark on draft voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)

		err := pv.SetRemark("Payment for invoice 12345")
		require.NoError(t, err)
		assert.Equal(t, "Payment for invoice 12345", pv.Remark)
	})

	t.Run("fails on cancelled voucher", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		userID := uuid.New()
		_ = pv.Cancel(userID, "Test")

		err := pv.SetRemark("Note")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot modify voucher in terminal state")
	})
}

// ============================================
// Helper Methods Tests
// ============================================

func TestPaymentVoucher_GetAmountMoney(t *testing.T) {
	pv := createTestPaymentVoucher(t)
	amount := pv.GetAmountMoney()
	assert.True(t, decimal.NewFromFloat(1000.00).Equal(amount.Amount()))
}

func TestPaymentVoucher_GetAllocatedAmountMoney(t *testing.T) {
	pv := createConfirmedPaymentVoucher(t)
	payableID := uuid.New()
	_, _ = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(400.00), "")

	allocated := pv.GetAllocatedAmountMoney()
	assert.True(t, decimal.NewFromFloat(400.00).Equal(allocated.Amount()))
}

func TestPaymentVoucher_GetUnallocatedAmountMoney(t *testing.T) {
	pv := createConfirmedPaymentVoucher(t)
	payableID := uuid.New()
	_, _ = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(400.00), "")

	unallocated := pv.GetUnallocatedAmountMoney()
	assert.True(t, decimal.NewFromFloat(600.00).Equal(unallocated.Amount()))
}

func TestPaymentVoucher_StatusMethods(t *testing.T) {
	t.Run("IsDraft", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		assert.True(t, pv.IsDraft())
		assert.False(t, pv.IsConfirmed())
		assert.False(t, pv.IsAllocated())
		assert.False(t, pv.IsCancelled())
	})

	t.Run("IsConfirmed", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)
		assert.False(t, pv.IsDraft())
		assert.True(t, pv.IsConfirmed())
		assert.False(t, pv.IsAllocated())
		assert.False(t, pv.IsCancelled())
	})

	t.Run("IsAllocated", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)
		payableID := uuid.New()
		_, _ = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(1000.00), "")

		assert.False(t, pv.IsDraft())
		assert.False(t, pv.IsConfirmed())
		assert.True(t, pv.IsAllocated())
		assert.False(t, pv.IsCancelled())
	})

	t.Run("IsCancelled", func(t *testing.T) {
		pv := createTestPaymentVoucher(t)
		userID := uuid.New()
		_ = pv.Cancel(userID, "Test")

		assert.False(t, pv.IsDraft())
		assert.False(t, pv.IsConfirmed())
		assert.False(t, pv.IsAllocated())
		assert.True(t, pv.IsCancelled())
	})
}

func TestPaymentVoucher_IsFullyAllocated(t *testing.T) {
	t.Run("not fully allocated when unallocated amount remains", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)
		payableID := uuid.New()
		_, _ = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(500.00), "")

		assert.False(t, pv.IsFullyAllocated())
	})

	t.Run("fully allocated when all amount is allocated", func(t *testing.T) {
		pv := createConfirmedPaymentVoucher(t)
		payableID := uuid.New()
		_, _ = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(1000.00), "")

		assert.True(t, pv.IsFullyAllocated())
	})
}

func TestPaymentVoucher_AllocationCount(t *testing.T) {
	pv := createConfirmedPaymentVoucher(t)
	assert.Equal(t, 0, pv.AllocationCount())

	_, _ = pv.AllocateToPayable(uuid.New(), "AP-001", valueobject.NewMoneyCNYFromFloat(300.00), "")
	assert.Equal(t, 1, pv.AllocationCount())

	_, _ = pv.AllocateToPayable(uuid.New(), "AP-002", valueobject.NewMoneyCNYFromFloat(300.00), "")
	assert.Equal(t, 2, pv.AllocationCount())
}

func TestPaymentVoucher_AllocatedPercentage(t *testing.T) {
	pv := createConfirmedPaymentVoucher(t)

	// 0% allocated
	pct := pv.AllocatedPercentage()
	assert.True(t, decimal.Zero.Equal(pct))

	// 50% allocated
	_, _ = pv.AllocateToPayable(uuid.New(), "AP-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
	pct = pv.AllocatedPercentage()
	assert.True(t, decimal.NewFromFloat(50.00).Equal(pct))

	// 100% allocated
	_, _ = pv.AllocateToPayable(uuid.New(), "AP-002", valueobject.NewMoneyCNYFromFloat(500.00), "")
	pct = pv.AllocatedPercentage()
	assert.True(t, decimal.NewFromFloat(100.00).Equal(pct))
}

func TestPaymentVoucher_GetAllocationByPayableID(t *testing.T) {
	pv := createConfirmedPaymentVoucher(t)
	payableID1 := uuid.New()
	payableID2 := uuid.New()

	_, _ = pv.AllocateToPayable(payableID1, "AP-001", valueobject.NewMoneyCNYFromFloat(400.00), "")
	_, _ = pv.AllocateToPayable(payableID2, "AP-002", valueobject.NewMoneyCNYFromFloat(600.00), "")

	// Find existing allocation
	allocation := pv.GetAllocationByPayableID(payableID1)
	require.NotNil(t, allocation)
	assert.Equal(t, payableID1, allocation.PayableID)
	assert.True(t, decimal.NewFromFloat(400.00).Equal(allocation.Amount))

	// Find non-existing allocation
	allocation = pv.GetAllocationByPayableID(uuid.New())
	assert.Nil(t, allocation)
}

// ============================================
// PayableAllocation Tests
// ============================================

func TestNewPayableAllocation(t *testing.T) {
	voucherID := uuid.New()
	payableID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(500.00)

	allocation := NewPayableAllocation(voucherID, payableID, "AP-001", amount, "Test remark")

	assert.NotEqual(t, uuid.Nil, allocation.ID)
	assert.Equal(t, voucherID, allocation.PaymentVoucherID)
	assert.Equal(t, payableID, allocation.PayableID)
	assert.Equal(t, "AP-001", allocation.PayableNumber)
	assert.True(t, amount.Amount().Equal(allocation.Amount))
	assert.Equal(t, "Test remark", allocation.Remark)
	assert.False(t, allocation.AllocatedAt.IsZero())
}

func TestPayableAllocation_GetAmountMoney(t *testing.T) {
	allocation := NewPayableAllocation(uuid.New(), uuid.New(), "AP-001", valueobject.NewMoneyCNYFromFloat(750.50), "")

	money := allocation.GetAmountMoney()
	assert.True(t, decimal.NewFromFloat(750.50).Equal(money.Amount()))
}

// ============================================
// Domain Events Tests
// ============================================

func TestPaymentVoucherCreatedEvent(t *testing.T) {
	pv := createTestPaymentVoucher(t)
	events := pv.GetDomainEvents()

	require.Len(t, events, 1)
	event, ok := events[0].(*PaymentVoucherCreatedEvent)
	require.True(t, ok)

	assert.Equal(t, "PaymentVoucherCreated", event.EventType())
	assert.Equal(t, pv.ID, event.VoucherID)
	assert.Equal(t, pv.VoucherNumber, event.VoucherNumber)
	assert.Equal(t, pv.SupplierID, event.SupplierID)
	assert.Equal(t, pv.SupplierName, event.SupplierName)
	assert.True(t, pv.Amount.Equal(event.Amount))
	assert.Equal(t, pv.PaymentMethod, event.PaymentMethod)
}

func TestPaymentVoucherConfirmedEvent(t *testing.T) {
	pv := createTestPaymentVoucher(t)
	pv.ClearDomainEvents()
	userID := uuid.New()

	_ = pv.Confirm(userID)
	events := pv.GetDomainEvents()

	require.Len(t, events, 1)
	event, ok := events[0].(*PaymentVoucherConfirmedEvent)
	require.True(t, ok)

	assert.Equal(t, "PaymentVoucherConfirmed", event.EventType())
	assert.Equal(t, pv.ID, event.VoucherID)
	assert.Equal(t, userID, event.ConfirmedBy)
}

func TestPaymentVoucherAllocatedEvent(t *testing.T) {
	pv := createConfirmedPaymentVoucher(t)
	pv.ClearDomainEvents()
	payableID := uuid.New()

	_, _ = pv.AllocateToPayable(payableID, "AP-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
	events := pv.GetDomainEvents()

	require.Len(t, events, 1)
	event, ok := events[0].(*PaymentVoucherAllocatedEvent)
	require.True(t, ok)

	assert.Equal(t, "PaymentVoucherAllocated", event.EventType())
	assert.Equal(t, pv.ID, event.VoucherID)
	assert.Equal(t, payableID, event.PayableID)
	assert.Equal(t, "AP-001", event.PayableNumber)
	assert.True(t, decimal.NewFromFloat(500.00).Equal(event.AllocationAmount))
	assert.False(t, event.IsFullyAllocated)
}

func TestPaymentVoucherCancelledEvent(t *testing.T) {
	pv := createTestPaymentVoucher(t)
	pv.ClearDomainEvents()
	userID := uuid.New()

	_ = pv.Cancel(userID, "Test cancellation")
	events := pv.GetDomainEvents()

	require.Len(t, events, 1)
	event, ok := events[0].(*PaymentVoucherCancelledEvent)
	require.True(t, ok)

	assert.Equal(t, "PaymentVoucherCancelled", event.EventType())
	assert.Equal(t, pv.ID, event.VoucherID)
	assert.Equal(t, userID, event.CancelledBy)
	assert.Equal(t, "Test cancellation", event.CancelReason)
	assert.Equal(t, VoucherStatusDraft, event.PreviousStatus)
}
