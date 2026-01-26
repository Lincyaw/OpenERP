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
func createTestReceiptVoucher(t *testing.T) *ReceiptVoucher {
	tenantID := uuid.New()
	customerID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1000.00)

	rv, err := NewReceiptVoucher(
		tenantID,
		"RV-2024-001",
		customerID,
		"Test Customer",
		amount,
		PaymentMethodCash,
		time.Now(),
	)
	require.NoError(t, err)
	return rv
}

func createConfirmedReceiptVoucher(t *testing.T) *ReceiptVoucher {
	rv := createTestReceiptVoucher(t)
	userID := uuid.New()
	err := rv.Confirm(userID)
	require.NoError(t, err)
	return rv
}

// ============================================
// VoucherStatus Tests
// ============================================

func TestVoucherStatus_IsValid(t *testing.T) {
	tests := []struct {
		status  VoucherStatus
		isValid bool
	}{
		{VoucherStatusDraft, true},
		{VoucherStatusConfirmed, true},
		{VoucherStatusAllocated, true},
		{VoucherStatusCancelled, true},
		{VoucherStatus("INVALID"), false},
		{VoucherStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.status.IsValid())
		})
	}
}

func TestVoucherStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status     VoucherStatus
		isTerminal bool
	}{
		{VoucherStatusDraft, false},
		{VoucherStatusConfirmed, false},
		{VoucherStatusAllocated, true},
		{VoucherStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.isTerminal, tt.status.IsTerminal())
		})
	}
}

func TestVoucherStatus_CanAllocate(t *testing.T) {
	tests := []struct {
		status      VoucherStatus
		canAllocate bool
	}{
		{VoucherStatusDraft, false},
		{VoucherStatusConfirmed, true},
		{VoucherStatusAllocated, false},
		{VoucherStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.canAllocate, tt.status.CanAllocate())
		})
	}
}

func TestVoucherStatus_CanConfirm(t *testing.T) {
	tests := []struct {
		status     VoucherStatus
		canConfirm bool
	}{
		{VoucherStatusDraft, true},
		{VoucherStatusConfirmed, false},
		{VoucherStatusAllocated, false},
		{VoucherStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.canConfirm, tt.status.CanConfirm())
		})
	}
}

func TestVoucherStatus_CanCancel(t *testing.T) {
	tests := []struct {
		status    VoucherStatus
		canCancel bool
	}{
		{VoucherStatusDraft, true},
		{VoucherStatusConfirmed, true},
		{VoucherStatusAllocated, false},
		{VoucherStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.canCancel, tt.status.CanCancel())
		})
	}
}

func TestVoucherStatus_String(t *testing.T) {
	assert.Equal(t, "DRAFT", VoucherStatusDraft.String())
	assert.Equal(t, "CONFIRMED", VoucherStatusConfirmed.String())
	assert.Equal(t, "ALLOCATED", VoucherStatusAllocated.String())
	assert.Equal(t, "CANCELLED", VoucherStatusCancelled.String())
}

// ============================================
// PaymentMethod Tests
// ============================================

func TestPaymentMethod_IsValid(t *testing.T) {
	tests := []struct {
		method  PaymentMethod
		isValid bool
	}{
		{PaymentMethodCash, true},
		{PaymentMethodBankTransfer, true},
		{PaymentMethodWechat, true},
		{PaymentMethodAlipay, true},
		{PaymentMethodCheck, true},
		{PaymentMethodBalance, true},
		{PaymentMethodOther, true},
		{PaymentMethod("INVALID"), false},
		{PaymentMethod(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.method.IsValid())
		})
	}
}

func TestPaymentMethod_String(t *testing.T) {
	assert.Equal(t, "CASH", PaymentMethodCash.String())
	assert.Equal(t, "BANK_TRANSFER", PaymentMethodBankTransfer.String())
	assert.Equal(t, "WECHAT", PaymentMethodWechat.String())
	assert.Equal(t, "ALIPAY", PaymentMethodAlipay.String())
}

// ============================================
// NewReceiptVoucher Tests
// ============================================

func TestNewReceiptVoucher(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(1500.00)
	receiptDate := time.Now()

	t.Run("creates voucher with valid inputs", func(t *testing.T) {
		rv, err := NewReceiptVoucher(
			tenantID,
			"RV-2024-001",
			customerID,
			"Test Customer",
			amount,
			PaymentMethodBankTransfer,
			receiptDate,
		)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, rv.ID)
		assert.Equal(t, tenantID, rv.TenantID)
		assert.Equal(t, "RV-2024-001", rv.VoucherNumber)
		assert.Equal(t, customerID, rv.CustomerID)
		assert.Equal(t, "Test Customer", rv.CustomerName)
		assert.True(t, amount.Amount().Equal(rv.Amount))
		assert.True(t, decimal.Zero.Equal(rv.AllocatedAmount))
		assert.True(t, amount.Amount().Equal(rv.UnallocatedAmount))
		assert.Equal(t, PaymentMethodBankTransfer, rv.PaymentMethod)
		assert.Equal(t, VoucherStatusDraft, rv.Status)
		assert.Equal(t, receiptDate.Unix(), rv.ReceiptDate.Unix())
		assert.Empty(t, rv.Allocations)
		assert.NotEmpty(t, rv.GetDomainEvents())
	})

	t.Run("fails with empty voucher number", func(t *testing.T) {
		_, err := NewReceiptVoucher(tenantID, "", customerID, "Test", amount, PaymentMethodCash, receiptDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Voucher number cannot be empty")
	})

	t.Run("fails with too long voucher number", func(t *testing.T) {
		longNumber := make([]byte, 51)
		for i := range longNumber {
			longNumber[i] = 'A'
		}
		_, err := NewReceiptVoucher(tenantID, string(longNumber), customerID, "Test", amount, PaymentMethodCash, receiptDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with empty customer ID", func(t *testing.T) {
		_, err := NewReceiptVoucher(tenantID, "RV-001", uuid.Nil, "Test", amount, PaymentMethodCash, receiptDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Customer ID cannot be empty")
	})

	t.Run("fails with empty customer name", func(t *testing.T) {
		_, err := NewReceiptVoucher(tenantID, "RV-001", customerID, "", amount, PaymentMethodCash, receiptDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Customer name cannot be empty")
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		zeroAmount := valueobject.NewMoneyCNY(decimal.Zero)
		_, err := NewReceiptVoucher(tenantID, "RV-001", customerID, "Test", zeroAmount, PaymentMethodCash, receiptDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Amount must be positive")
	})

	t.Run("fails with negative amount", func(t *testing.T) {
		negativeAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(-100))
		_, err := NewReceiptVoucher(tenantID, "RV-001", customerID, "Test", negativeAmount, PaymentMethodCash, receiptDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Amount must be positive")
	})

	t.Run("fails with invalid payment method", func(t *testing.T) {
		_, err := NewReceiptVoucher(tenantID, "RV-001", customerID, "Test", amount, PaymentMethod("BITCOIN"), receiptDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payment method is not valid")
	})

	t.Run("fails with zero receipt date", func(t *testing.T) {
		_, err := NewReceiptVoucher(tenantID, "RV-001", customerID, "Test", amount, PaymentMethodCash, time.Time{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Receipt date is required")
	})
}

// ============================================
// Confirm Tests
// ============================================

func TestReceiptVoucher_Confirm(t *testing.T) {
	userID := uuid.New()

	t.Run("confirms draft voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		rv.ClearDomainEvents()

		err := rv.Confirm(userID)
		require.NoError(t, err)
		assert.Equal(t, VoucherStatusConfirmed, rv.Status)
		assert.NotNil(t, rv.ConfirmedAt)
		assert.Equal(t, userID, *rv.ConfirmedBy)
		assert.Len(t, rv.GetDomainEvents(), 1)
	})

	t.Run("fails to confirm already confirmed voucher", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)

		err := rv.Confirm(userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot confirm voucher")
	})

	t.Run("fails to confirm cancelled voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		_ = rv.Cancel(userID, "Test cancel")

		err := rv.Confirm(userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot confirm voucher")
	})

	t.Run("fails with empty user ID", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)

		err := rv.Confirm(uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Confirming user ID is required")
	})
}

// ============================================
// AllocateToReceivable Tests
// ============================================

func TestReceiptVoucher_AllocateToReceivable(t *testing.T) {
	receivableID := uuid.New()

	t.Run("allocates partial amount to receivable", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t) // 1000.00
		rv.ClearDomainEvents()
		allocAmount := valueobject.NewMoneyCNYFromFloat(400.00)

		allocation, err := rv.AllocateToReceivable(receivableID, "AR-001", allocAmount, "Partial payment")
		require.NoError(t, err)
		assert.NotNil(t, allocation)
		assert.Equal(t, rv.ID, allocation.ReceiptVoucherID)
		assert.Equal(t, receivableID, allocation.ReceivableID)
		assert.Equal(t, "AR-001", allocation.ReceivableNumber)
		assert.True(t, allocAmount.Amount().Equal(allocation.Amount))

		assert.True(t, decimal.NewFromFloat(400.00).Equal(rv.AllocatedAmount))
		assert.True(t, decimal.NewFromFloat(600.00).Equal(rv.UnallocatedAmount))
		assert.Equal(t, VoucherStatusConfirmed, rv.Status) // Still confirmed, not fully allocated
		assert.Len(t, rv.GetDomainEvents(), 1)
	})

	t.Run("allocates full amount and changes status to ALLOCATED", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t) // 1000.00
		allocAmount := valueobject.NewMoneyCNYFromFloat(1000.00)

		_, err := rv.AllocateToReceivable(receivableID, "AR-001", allocAmount, "Full payment")
		require.NoError(t, err)

		assert.True(t, decimal.NewFromFloat(1000.00).Equal(rv.AllocatedAmount))
		assert.True(t, decimal.Zero.Equal(rv.UnallocatedAmount))
		assert.Equal(t, VoucherStatusAllocated, rv.Status)
		assert.True(t, rv.IsFullyAllocated())
	})

	t.Run("allocates to multiple receivables", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t) // 1000.00
		receivableID2 := uuid.New()

		_, err := rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(400.00), "")
		require.NoError(t, err)

		_, err = rv.AllocateToReceivable(receivableID2, "AR-002", valueobject.NewMoneyCNYFromFloat(600.00), "")
		require.NoError(t, err)

		assert.Equal(t, 2, rv.AllocationCount())
		assert.True(t, rv.IsFullyAllocated())
		assert.Equal(t, VoucherStatusAllocated, rv.Status)
	})

	t.Run("fails to allocate draft voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t) // Draft status

		_, err := rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot allocate voucher")
	})

	t.Run("fails to allocate cancelled voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		userID := uuid.New()
		_ = rv.Cancel(userID, "Test")

		_, err := rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot allocate voucher")
	})

	t.Run("fails with empty receivable ID", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)

		_, err := rv.AllocateToReceivable(uuid.Nil, "AR-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Receivable ID cannot be empty")
	})

	t.Run("fails with empty receivable number", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)

		_, err := rv.AllocateToReceivable(receivableID, "", valueobject.NewMoneyCNYFromFloat(500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Receivable number is required")
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)

		_, err := rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNY(decimal.Zero), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Allocation amount must be positive")
	})

	t.Run("fails when allocation exceeds unallocated amount", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t) // 1000.00

		_, err := rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(1500.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds unallocated amount")
	})

	t.Run("fails when allocating to same receivable twice", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t) // 1000.00

		_, err := rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(300.00), "")
		require.NoError(t, err)

		_, err = rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(300.00), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Already allocated to receivable")
	})
}

// ============================================
// Cancel Tests
// ============================================

func TestReceiptVoucher_Cancel(t *testing.T) {
	userID := uuid.New()

	t.Run("cancels draft voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		rv.ClearDomainEvents()

		err := rv.Cancel(userID, "Customer requested cancellation")
		require.NoError(t, err)
		assert.Equal(t, VoucherStatusCancelled, rv.Status)
		assert.NotNil(t, rv.CancelledAt)
		assert.Equal(t, userID, *rv.CancelledBy)
		assert.Equal(t, "Customer requested cancellation", rv.CancelReason)
		assert.Len(t, rv.GetDomainEvents(), 1)
	})

	t.Run("cancels confirmed voucher without allocations", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)
		rv.ClearDomainEvents()

		err := rv.Cancel(userID, "Duplicate entry")
		require.NoError(t, err)
		assert.Equal(t, VoucherStatusCancelled, rv.Status)
	})

	t.Run("fails to cancel voucher with allocations", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)
		receivableID := uuid.New()
		_, _ = rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(500.00), "")

		err := rv.Cancel(userID, "Test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel voucher with existing allocations")
	})

	t.Run("fails to cancel already cancelled voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		_ = rv.Cancel(userID, "First cancel")

		err := rv.Cancel(userID, "Second cancel")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel voucher")
	})

	t.Run("fails with empty user ID", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)

		err := rv.Cancel(uuid.Nil, "Test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cancelling user ID is required")
	})

	t.Run("fails with empty reason", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)

		err := rv.Cancel(userID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cancel reason is required")
	})
}

// ============================================
// SetPaymentReference Tests
// ============================================

func TestReceiptVoucher_SetPaymentReference(t *testing.T) {
	t.Run("sets payment reference on draft voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)

		err := rv.SetPaymentReference("TXN-123456")
		require.NoError(t, err)
		assert.Equal(t, "TXN-123456", rv.PaymentReference)
	})

	t.Run("sets payment reference on confirmed voucher", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)

		err := rv.SetPaymentReference("BANK-REF-789")
		require.NoError(t, err)
		assert.Equal(t, "BANK-REF-789", rv.PaymentReference)
	})

	t.Run("fails on cancelled voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		userID := uuid.New()
		_ = rv.Cancel(userID, "Test")

		err := rv.SetPaymentReference("REF-001")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot modify voucher in terminal state")
	})

	t.Run("fails with too long reference", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		longRef := make([]byte, 101)
		for i := range longRef {
			longRef[i] = 'A'
		}

		err := rv.SetPaymentReference(string(longRef))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Payment reference cannot exceed 100 characters")
	})
}

// ============================================
// SetRemark Tests
// ============================================

func TestReceiptVoucher_SetRemark(t *testing.T) {
	t.Run("sets remark on draft voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)

		err := rv.SetRemark("Payment for invoice 12345")
		require.NoError(t, err)
		assert.Equal(t, "Payment for invoice 12345", rv.Remark)
	})

	t.Run("fails on cancelled voucher", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		userID := uuid.New()
		_ = rv.Cancel(userID, "Test")

		err := rv.SetRemark("Note")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot modify voucher in terminal state")
	})
}

// ============================================
// Helper Methods Tests
// ============================================

func TestReceiptVoucher_GetAmountMoney(t *testing.T) {
	rv := createTestReceiptVoucher(t)
	amount := rv.GetAmountMoney()
	assert.True(t, decimal.NewFromFloat(1000.00).Equal(amount.Amount()))
}

func TestReceiptVoucher_GetAllocatedAmountMoney(t *testing.T) {
	rv := createConfirmedReceiptVoucher(t)
	receivableID := uuid.New()
	_, _ = rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(400.00), "")

	allocated := rv.GetAllocatedAmountMoney()
	assert.True(t, decimal.NewFromFloat(400.00).Equal(allocated.Amount()))
}

func TestReceiptVoucher_GetUnallocatedAmountMoney(t *testing.T) {
	rv := createConfirmedReceiptVoucher(t)
	receivableID := uuid.New()
	_, _ = rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(400.00), "")

	unallocated := rv.GetUnallocatedAmountMoney()
	assert.True(t, decimal.NewFromFloat(600.00).Equal(unallocated.Amount()))
}

func TestReceiptVoucher_StatusMethods(t *testing.T) {
	t.Run("IsDraft", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		assert.True(t, rv.IsDraft())
		assert.False(t, rv.IsConfirmed())
		assert.False(t, rv.IsAllocated())
		assert.False(t, rv.IsCancelled())
	})

	t.Run("IsConfirmed", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)
		assert.False(t, rv.IsDraft())
		assert.True(t, rv.IsConfirmed())
		assert.False(t, rv.IsAllocated())
		assert.False(t, rv.IsCancelled())
	})

	t.Run("IsAllocated", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)
		receivableID := uuid.New()
		_, _ = rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(1000.00), "")

		assert.False(t, rv.IsDraft())
		assert.False(t, rv.IsConfirmed())
		assert.True(t, rv.IsAllocated())
		assert.False(t, rv.IsCancelled())
	})

	t.Run("IsCancelled", func(t *testing.T) {
		rv := createTestReceiptVoucher(t)
		userID := uuid.New()
		_ = rv.Cancel(userID, "Test")

		assert.False(t, rv.IsDraft())
		assert.False(t, rv.IsConfirmed())
		assert.False(t, rv.IsAllocated())
		assert.True(t, rv.IsCancelled())
	})
}

func TestReceiptVoucher_IsFullyAllocated(t *testing.T) {
	t.Run("not fully allocated when unallocated amount remains", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)
		receivableID := uuid.New()
		_, _ = rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(500.00), "")

		assert.False(t, rv.IsFullyAllocated())
	})

	t.Run("fully allocated when all amount is allocated", func(t *testing.T) {
		rv := createConfirmedReceiptVoucher(t)
		receivableID := uuid.New()
		_, _ = rv.AllocateToReceivable(receivableID, "AR-001", valueobject.NewMoneyCNYFromFloat(1000.00), "")

		assert.True(t, rv.IsFullyAllocated())
	})
}

func TestReceiptVoucher_AllocationCount(t *testing.T) {
	rv := createConfirmedReceiptVoucher(t)
	assert.Equal(t, 0, rv.AllocationCount())

	_, _ = rv.AllocateToReceivable(uuid.New(), "AR-001", valueobject.NewMoneyCNYFromFloat(300.00), "")
	assert.Equal(t, 1, rv.AllocationCount())

	_, _ = rv.AllocateToReceivable(uuid.New(), "AR-002", valueobject.NewMoneyCNYFromFloat(300.00), "")
	assert.Equal(t, 2, rv.AllocationCount())
}

func TestReceiptVoucher_AllocatedPercentage(t *testing.T) {
	rv := createConfirmedReceiptVoucher(t)

	// 0% allocated
	pct := rv.AllocatedPercentage()
	assert.True(t, decimal.Zero.Equal(pct))

	// 50% allocated
	_, _ = rv.AllocateToReceivable(uuid.New(), "AR-001", valueobject.NewMoneyCNYFromFloat(500.00), "")
	pct = rv.AllocatedPercentage()
	assert.True(t, decimal.NewFromFloat(50.00).Equal(pct))

	// 100% allocated
	_, _ = rv.AllocateToReceivable(uuid.New(), "AR-002", valueobject.NewMoneyCNYFromFloat(500.00), "")
	pct = rv.AllocatedPercentage()
	assert.True(t, decimal.NewFromFloat(100.00).Equal(pct))
}

func TestReceiptVoucher_GetAllocationByReceivableID(t *testing.T) {
	rv := createConfirmedReceiptVoucher(t)
	receivableID1 := uuid.New()
	receivableID2 := uuid.New()

	_, _ = rv.AllocateToReceivable(receivableID1, "AR-001", valueobject.NewMoneyCNYFromFloat(400.00), "")
	_, _ = rv.AllocateToReceivable(receivableID2, "AR-002", valueobject.NewMoneyCNYFromFloat(600.00), "")

	// Find existing allocation
	allocation := rv.GetAllocationByReceivableID(receivableID1)
	require.NotNil(t, allocation)
	assert.Equal(t, receivableID1, allocation.ReceivableID)
	assert.True(t, decimal.NewFromFloat(400.00).Equal(allocation.Amount))

	// Find non-existing allocation
	allocation = rv.GetAllocationByReceivableID(uuid.New())
	assert.Nil(t, allocation)
}

// ============================================
// ReceivableAllocation Tests
// ============================================

func TestNewReceivableAllocation(t *testing.T) {
	voucherID := uuid.New()
	receivableID := uuid.New()
	amount := valueobject.NewMoneyCNYFromFloat(500.00)

	allocation := NewReceivableAllocation(voucherID, receivableID, "AR-001", amount, "Test remark")

	assert.NotEqual(t, uuid.Nil, allocation.ID)
	assert.Equal(t, voucherID, allocation.ReceiptVoucherID)
	assert.Equal(t, receivableID, allocation.ReceivableID)
	assert.Equal(t, "AR-001", allocation.ReceivableNumber)
	assert.True(t, amount.Amount().Equal(allocation.Amount))
	assert.Equal(t, "Test remark", allocation.Remark)
	assert.False(t, allocation.AllocatedAt.IsZero())
}

func TestReceivableAllocation_GetAmountMoney(t *testing.T) {
	allocation := NewReceivableAllocation(uuid.New(), uuid.New(), "AR-001", valueobject.NewMoneyCNYFromFloat(750.50), "")

	money := allocation.GetAmountMoney()
	assert.True(t, decimal.NewFromFloat(750.50).Equal(money.Amount()))
}
