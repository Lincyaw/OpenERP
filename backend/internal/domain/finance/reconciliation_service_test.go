package finance

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReconciliationService(t *testing.T) {
	service := NewReconciliationService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.strategyFactory)
	assert.Equal(t, ReconciliationStrategyTypeFIFO, service.defaultStrategyType)
}

func TestReconciliationServiceWithOptions(t *testing.T) {
	t.Run("WithDefaultStrategy sets custom default", func(t *testing.T) {
		service := NewReconciliationService(
			WithDefaultStrategy(ReconciliationStrategyTypeManual),
		)
		assert.Equal(t, ReconciliationStrategyTypeManual, service.GetDefaultStrategy())
	})

	t.Run("WithDefaultStrategy ignores invalid strategy", func(t *testing.T) {
		service := NewReconciliationService(
			WithDefaultStrategy("INVALID"),
		)
		// Should keep the default FIFO
		assert.Equal(t, ReconciliationStrategyTypeFIFO, service.GetDefaultStrategy())
	})

	t.Run("WithStrategyOverride allows context-based strategy selection", func(t *testing.T) {
		tenantID := uuid.New()
		overrideFunc := func(ctx context.Context, tid uuid.UUID) ReconciliationStrategyType {
			if tid == tenantID {
				return ReconciliationStrategyTypeManual
			}
			return ""
		}

		service := NewReconciliationService(
			WithStrategyOverride(overrideFunc),
		)

		// For matching tenant, should return override
		effective := service.GetEffectiveStrategy(context.Background(), tenantID)
		assert.Equal(t, ReconciliationStrategyTypeManual, effective)

		// For other tenant, should return default
		otherTenant := uuid.New()
		effective = service.GetEffectiveStrategy(context.Background(), otherTenant)
		assert.Equal(t, ReconciliationStrategyTypeFIFO, effective)
	})

	t.Run("Multiple options can be chained", func(t *testing.T) {
		tenantID := uuid.New()
		overrideFunc := func(ctx context.Context, tid uuid.UUID) ReconciliationStrategyType {
			return ReconciliationStrategyTypeManual
		}

		service := NewReconciliationService(
			WithDefaultStrategy(ReconciliationStrategyTypeFIFO),
			WithStrategyOverride(overrideFunc),
		)

		assert.Equal(t, ReconciliationStrategyTypeFIFO, service.GetDefaultStrategy())
		assert.Equal(t, ReconciliationStrategyTypeManual, service.GetEffectiveStrategy(context.Background(), tenantID))
	})
}

// Helper functions for creating test data with custom parameters
func createReceiptVoucherForReconciliation(t *testing.T, tenantID, customerID uuid.UUID, amount decimal.Decimal, confirmed bool) *ReceiptVoucher {
	rv, err := NewReceiptVoucher(
		tenantID,
		"RV-TEST-001",
		customerID,
		"Test Customer",
		valueobject.NewMoneyCNY(amount),
		PaymentMethodCash,
		time.Now(),
	)
	require.NoError(t, err)
	if confirmed {
		err = rv.Confirm(uuid.New())
		require.NoError(t, err)
	}
	return rv
}

func createPaymentVoucherForReconciliation(t *testing.T, tenantID, supplierID uuid.UUID, amount decimal.Decimal, confirmed bool) *PaymentVoucher {
	pv, err := NewPaymentVoucher(
		tenantID,
		"PV-TEST-001",
		supplierID,
		"Test Supplier",
		valueobject.NewMoneyCNY(amount),
		PaymentMethodBankTransfer,
		time.Now(),
	)
	require.NoError(t, err)
	if confirmed {
		err = pv.Confirm(uuid.New())
		require.NoError(t, err)
	}
	return pv
}

func createReceivableForReconciliation(t *testing.T, tenantID, customerID uuid.UUID, receivableNumber string, amount decimal.Decimal, dueDate *time.Time) AccountReceivable {
	ar, err := NewAccountReceivable(
		tenantID,
		receivableNumber,
		customerID,
		"Test Customer",
		SourceTypeSalesOrder,
		uuid.New(),
		"SO-001",
		valueobject.NewMoneyCNY(amount),
		dueDate,
	)
	require.NoError(t, err)
	return *ar
}

func createPayableForReconciliation(t *testing.T, tenantID, supplierID uuid.UUID, payableNumber string, amount decimal.Decimal, dueDate *time.Time) AccountPayable {
	ap, err := NewAccountPayable(
		tenantID,
		payableNumber,
		supplierID,
		"Test Supplier",
		PayableSourceTypePurchaseOrder,
		uuid.New(),
		"PO-001",
		valueobject.NewMoneyCNY(amount),
		dueDate,
	)
	require.NoError(t, err)
	return *ap
}

// ReconcileReceipt Tests

func TestReconciliationService_ReconcileReceipt_NilVoucher(t *testing.T) {
	service := NewReconciliationService()

	_, err := service.ReconcileReceipt(context.Background(), ReconcileReceiptRequest{
		ReceiptVoucher: nil,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Receipt voucher cannot be nil")
}

func TestReconciliationService_ReconcileReceipt_UnconfirmedVoucher(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), false)

	_, err := service.ReconcileReceipt(context.Background(), ReconcileReceiptRequest{
		ReceiptVoucher: voucher,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot allocate voucher in DRAFT status")
}

func TestReconciliationService_ReconcileReceipt_InvalidStrategy(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)

	_, err := service.ReconcileReceipt(context.Background(), ReconcileReceiptRequest{
		ReceiptVoucher: voucher,
		StrategyType:   "INVALID",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid reconciliation strategy type")
}

func TestReconciliationService_ReconcileReceipt_NoReceivables(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)

	result, err := service.ReconcileReceipt(context.Background(), ReconcileReceiptRequest{
		ReceiptVoucher: voucher,
		Receivables:    []AccountReceivable{},
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	require.NoError(t, err)
	assert.Len(t, result.UpdatedReceivables, 0)
	assert.Len(t, result.Allocations, 0)
	assert.True(t, result.TotalReconciled.IsZero())
	assert.Equal(t, voucher.UnallocatedAmount, result.RemainingUnallocated)
	assert.False(t, result.FullyReconciled)
}

func TestReconciliationService_AutoReconcileReceipt_FIFO_SingleReceivable(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	receivable := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable})

	require.NoError(t, err)
	assert.Len(t, result.UpdatedReceivables, 1)
	assert.Len(t, result.Allocations, 1)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(1000)))
	assert.True(t, result.RemainingUnallocated.IsZero())
	assert.True(t, result.FullyReconciled)

	// Verify receivable was updated
	assert.Equal(t, ReceivableStatusPaid, result.UpdatedReceivables[0].Status)
	assert.True(t, result.UpdatedReceivables[0].OutstandingAmount.IsZero())

	// Verify voucher was updated
	assert.Equal(t, VoucherStatusAllocated, result.ReceiptVoucher.Status)
	assert.True(t, result.ReceiptVoucher.UnallocatedAmount.IsZero())
}

func TestReconciliationService_AutoReconcileReceipt_FIFO_MultipleReceivables(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(2500), true)

	// Create receivables with different due dates - FIFO should allocate to oldest first
	dueDate1 := time.Now().Add(7 * 24 * time.Hour)  // Oldest
	dueDate2 := time.Now().Add(14 * 24 * time.Hour) // Middle
	dueDate3 := time.Now().Add(21 * 24 * time.Hour) // Newest

	receivables := []AccountReceivable{
		createReceivableForReconciliation(t, tenantID, customerID, "AR-003", decimal.NewFromInt(800), &dueDate3),  // Should be last
		createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate1), // Should be first
		createReceivableForReconciliation(t, tenantID, customerID, "AR-002", decimal.NewFromInt(1200), &dueDate2), // Should be second
	}

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, receivables)

	require.NoError(t, err)
	assert.Len(t, result.UpdatedReceivables, 3)
	assert.Len(t, result.Allocations, 3)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(2500)))
	assert.True(t, result.RemainingUnallocated.IsZero())
	assert.True(t, result.FullyReconciled)

	// Verify allocations were made in FIFO order (by due date)
	assert.Equal(t, "AR-001", result.Allocations[0].ReceivableNumber)
	assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(1000)))
	assert.Equal(t, "AR-002", result.Allocations[1].ReceivableNumber)
	assert.True(t, result.Allocations[1].Amount.Equal(decimal.NewFromInt(1200)))
	assert.Equal(t, "AR-003", result.Allocations[2].ReceivableNumber)
	assert.True(t, result.Allocations[2].Amount.Equal(decimal.NewFromInt(300))) // Partial allocation
}

func TestReconciliationService_AutoReconcileReceipt_PartialAllocation(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(500), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	receivable := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable})

	require.NoError(t, err)
	assert.Len(t, result.UpdatedReceivables, 1)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(500)))
	assert.True(t, result.RemainingUnallocated.IsZero())
	assert.True(t, result.FullyReconciled)

	// Verify receivable is partially paid
	assert.Equal(t, ReceivableStatusPartial, result.UpdatedReceivables[0].Status)
	assert.True(t, result.UpdatedReceivables[0].OutstandingAmount.Equal(decimal.NewFromInt(500)))
}

func TestReconciliationService_AutoReconcileReceipt_VoucherExceedsReceivables(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(2000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	receivable := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable})

	require.NoError(t, err)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(1000)))
	assert.True(t, result.RemainingUnallocated.Equal(decimal.NewFromInt(1000)))
	assert.False(t, result.FullyReconciled)

	// Voucher should be CONFIRMED (not fully allocated)
	assert.Equal(t, VoucherStatusConfirmed, result.ReceiptVoucher.Status)
}

func TestReconciliationService_AutoReconcileReceipt_FiltersDifferentCustomer(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()
	otherCustomerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	// Receivable for different customer - should be filtered out
	receivable := createReceivableForReconciliation(t, tenantID, otherCustomerID, "AR-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable})

	require.NoError(t, err)
	assert.Len(t, result.UpdatedReceivables, 0)
	assert.Len(t, result.Allocations, 0)
	assert.False(t, result.FullyReconciled)
}

func TestReconciliationService_ManualReconcileReceipt(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1500), true)

	dueDate1 := time.Now().Add(7 * 24 * time.Hour)
	dueDate2 := time.Now().Add(14 * 24 * time.Hour)

	receivable1 := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate1)
	receivable2 := createReceivableForReconciliation(t, tenantID, customerID, "AR-002", decimal.NewFromInt(1000), &dueDate2)

	// Manually allocate to AR-002 first, then AR-001 (opposite of FIFO order)
	allocations := []ManualAllocationRequest{
		{TargetID: receivable2.ID, Amount: decimal.NewFromInt(800)},
		{TargetID: receivable1.ID, Amount: decimal.NewFromInt(700)},
	}

	result, err := service.ManualReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable1, receivable2}, allocations)

	require.NoError(t, err)
	assert.Len(t, result.UpdatedReceivables, 2)
	assert.Len(t, result.Allocations, 2)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(1500)))
	assert.True(t, result.FullyReconciled)

	// Verify allocations were made in manual order
	assert.Equal(t, "AR-002", result.Allocations[0].ReceivableNumber)
	assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(800)))
	assert.Equal(t, "AR-001", result.Allocations[1].ReceivableNumber)
	assert.True(t, result.Allocations[1].Amount.Equal(decimal.NewFromInt(700)))
}

func TestReconciliationService_ManualReconcileReceipt_NoAllocationsProvided(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)

	_, err := service.ManualReconcileReceipt(context.Background(), voucher, []AccountReceivable{}, []ManualAllocationRequest{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Manual strategy requires allocation requests")
}

// ReconcilePayment Tests

func TestReconciliationService_ReconcilePayment_NilVoucher(t *testing.T) {
	service := NewReconciliationService()

	_, err := service.ReconcilePayment(context.Background(), ReconcilePaymentRequest{
		PaymentVoucher: nil,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Payment voucher cannot be nil")
}

func TestReconciliationService_ReconcilePayment_UnconfirmedVoucher(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(1000), false)

	_, err := service.ReconcilePayment(context.Background(), ReconcilePaymentRequest{
		PaymentVoucher: voucher,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot allocate voucher in DRAFT status")
}

func TestReconciliationService_AutoReconcilePayment_FIFO_SinglePayable(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(1000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	payable := createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcilePayment(context.Background(), voucher, []AccountPayable{payable})

	require.NoError(t, err)
	assert.Len(t, result.UpdatedPayables, 1)
	assert.Len(t, result.Allocations, 1)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(1000)))
	assert.True(t, result.RemainingUnallocated.IsZero())
	assert.True(t, result.FullyReconciled)

	// Verify payable was updated
	assert.Equal(t, PayableStatusPaid, result.UpdatedPayables[0].Status)
	assert.True(t, result.UpdatedPayables[0].OutstandingAmount.IsZero())

	// Verify voucher was updated
	assert.Equal(t, VoucherStatusAllocated, result.PaymentVoucher.Status)
	assert.True(t, result.PaymentVoucher.UnallocatedAmount.IsZero())
}

func TestReconciliationService_AutoReconcilePayment_FIFO_MultiplePayables(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(2500), true)

	// Create payables with different due dates
	dueDate1 := time.Now().Add(7 * 24 * time.Hour)
	dueDate2 := time.Now().Add(14 * 24 * time.Hour)
	dueDate3 := time.Now().Add(21 * 24 * time.Hour)

	payables := []AccountPayable{
		createPayableForReconciliation(t, tenantID, supplierID, "AP-003", decimal.NewFromInt(800), &dueDate3),
		createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate1),
		createPayableForReconciliation(t, tenantID, supplierID, "AP-002", decimal.NewFromInt(1200), &dueDate2),
	}

	result, err := service.AutoReconcilePayment(context.Background(), voucher, payables)

	require.NoError(t, err)
	assert.Len(t, result.UpdatedPayables, 3)
	assert.Len(t, result.Allocations, 3)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(2500)))
	assert.True(t, result.FullyReconciled)

	// Verify allocations were made in FIFO order
	assert.Equal(t, "AP-001", result.Allocations[0].PayableNumber)
	assert.Equal(t, "AP-002", result.Allocations[1].PayableNumber)
	assert.Equal(t, "AP-003", result.Allocations[2].PayableNumber)
}

func TestReconciliationService_AutoReconcilePayment_FiltersDifferentSupplier(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()
	otherSupplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(1000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	// Payable for different supplier - should be filtered out
	payable := createPayableForReconciliation(t, tenantID, otherSupplierID, "AP-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcilePayment(context.Background(), voucher, []AccountPayable{payable})

	require.NoError(t, err)
	assert.Len(t, result.UpdatedPayables, 0)
	assert.Len(t, result.Allocations, 0)
	assert.False(t, result.FullyReconciled)
}

func TestReconciliationService_ManualReconcilePayment(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(1500), true)

	dueDate1 := time.Now().Add(7 * 24 * time.Hour)
	dueDate2 := time.Now().Add(14 * 24 * time.Hour)

	payable1 := createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate1)
	payable2 := createPayableForReconciliation(t, tenantID, supplierID, "AP-002", decimal.NewFromInt(1000), &dueDate2)

	// Manually allocate to AP-002 first
	allocations := []ManualAllocationRequest{
		{TargetID: payable2.ID, Amount: decimal.NewFromInt(800)},
		{TargetID: payable1.ID, Amount: decimal.NewFromInt(700)},
	}

	result, err := service.ManualReconcilePayment(context.Background(), voucher, []AccountPayable{payable1, payable2}, allocations)

	require.NoError(t, err)
	assert.Len(t, result.UpdatedPayables, 2)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(1500)))
	assert.True(t, result.FullyReconciled)

	// Verify allocations were made in manual order
	assert.Equal(t, "AP-002", result.Allocations[0].PayableNumber)
	assert.Equal(t, "AP-001", result.Allocations[1].PayableNumber)
}

// Preview Tests

func TestReconciliationService_PreviewReconcileReceipt(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)
	originalUnallocated := voucher.UnallocatedAmount

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	receivable := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate)
	originalOutstanding := receivable.OutstandingAmount

	result, err := service.PreviewReconcileReceipt(context.Background(), ReconcileReceiptRequest{
		ReceiptVoucher: voucher,
		Receivables:    []AccountReceivable{receivable},
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	require.NoError(t, err)
	assert.Len(t, result.Allocations, 1)
	assert.True(t, result.TotalAllocated.Equal(decimal.NewFromInt(1000)))
	assert.True(t, result.FullyReconciled)

	// Verify original entities were NOT modified
	assert.True(t, voucher.UnallocatedAmount.Equal(originalUnallocated))
	assert.True(t, receivable.OutstandingAmount.Equal(originalOutstanding))
}

func TestReconciliationService_PreviewReconcileReceipt_NilVoucher(t *testing.T) {
	service := NewReconciliationService()

	_, err := service.PreviewReconcileReceipt(context.Background(), ReconcileReceiptRequest{
		ReceiptVoucher: nil,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Receipt voucher cannot be nil")
}

func TestReconciliationService_PreviewReconcilePayment(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(1000), true)
	originalUnallocated := voucher.UnallocatedAmount

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	payable := createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate)
	originalOutstanding := payable.OutstandingAmount

	result, err := service.PreviewReconcilePayment(context.Background(), ReconcilePaymentRequest{
		PaymentVoucher: voucher,
		Payables:       []AccountPayable{payable},
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	require.NoError(t, err)
	assert.Len(t, result.Allocations, 1)
	assert.True(t, result.TotalAllocated.Equal(decimal.NewFromInt(1000)))

	// Verify original entities were NOT modified
	assert.True(t, voucher.UnallocatedAmount.Equal(originalUnallocated))
	assert.True(t, payable.OutstandingAmount.Equal(originalOutstanding))
}

func TestReconciliationService_PreviewReconcilePayment_NilVoucher(t *testing.T) {
	service := NewReconciliationService()

	_, err := service.PreviewReconcilePayment(context.Background(), ReconcilePaymentRequest{
		PaymentVoucher: nil,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Payment voucher cannot be nil")
}

// Edge Cases

func TestReconciliationService_ReconcileReceipt_ReceivableAlreadyCancelled(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	receivable := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate)
	// Mark receivable as cancelled
	_ = receivable.Cancel("Test cancellation")

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable})

	require.NoError(t, err)
	// Should not allocate to cancelled receivable
	assert.Len(t, result.UpdatedReceivables, 0)
	assert.Len(t, result.Allocations, 0)
	assert.False(t, result.FullyReconciled)
}

func TestReconciliationService_ReconcilePayment_PayableAlreadyCancelled(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(1000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	payable := createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate)
	// Mark payable as cancelled
	_ = payable.Cancel("Test cancellation")

	result, err := service.AutoReconcilePayment(context.Background(), voucher, []AccountPayable{payable})

	require.NoError(t, err)
	// Should not allocate to cancelled payable
	assert.Len(t, result.UpdatedPayables, 0)
	assert.Len(t, result.Allocations, 0)
	assert.False(t, result.FullyReconciled)
}

func TestReconciliationService_ReconcileReceipt_ZeroOutstandingReceivable(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)

	// Create two receivables: one will be paid, one with zero outstanding (already paid in previous reconciliation)
	dueDate := time.Now().Add(7 * 24 * time.Hour)
	receivable1 := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(500), &dueDate)
	receivable2 := createReceivableForReconciliation(t, tenantID, customerID, "AR-002", decimal.NewFromInt(500), &dueDate)

	// Simulate receivable2 being already paid
	_ = receivable2.ApplyPayment(valueobject.NewMoneyCNY(decimal.NewFromInt(500)), uuid.New(), "Previous payment")

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable1, receivable2})

	require.NoError(t, err)
	// Should only allocate to receivable1
	assert.Len(t, result.UpdatedReceivables, 1)
	assert.Equal(t, "AR-001", result.Allocations[0].ReceivableNumber)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(500)))
}

// Test that events are added properly during reconciliation

func TestReconciliationService_ReconcileReceipt_GeneratesEvents(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	customerID := uuid.New()

	voucher := createReceiptVoucherForReconciliation(t, tenantID, customerID, decimal.NewFromInt(1000), true)
	// Clear events from creation/confirmation
	voucher.ClearDomainEvents()

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	receivable := createReceivableForReconciliation(t, tenantID, customerID, "AR-001", decimal.NewFromInt(1000), &dueDate)
	receivable.ClearDomainEvents()

	result, err := service.AutoReconcileReceipt(context.Background(), voucher, []AccountReceivable{receivable})

	require.NoError(t, err)

	// Voucher should have allocation event
	voucherEvents := result.ReceiptVoucher.GetDomainEvents()
	assert.GreaterOrEqual(t, len(voucherEvents), 1)

	// Receivable should have payment event
	receivableEvents := result.UpdatedReceivables[0].GetDomainEvents()
	assert.GreaterOrEqual(t, len(receivableEvents), 1)
}

func TestReconciliationService_ReconcilePayment_GeneratesEvents(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(1000), true)
	voucher.ClearDomainEvents()

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	payable := createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate)
	payable.ClearDomainEvents()

	result, err := service.AutoReconcilePayment(context.Background(), voucher, []AccountPayable{payable})

	require.NoError(t, err)

	// Voucher should have allocation event
	voucherEvents := result.PaymentVoucher.GetDomainEvents()
	assert.GreaterOrEqual(t, len(voucherEvents), 1)

	// Payable should have payment event
	payableEvents := result.UpdatedPayables[0].GetDomainEvents()
	assert.GreaterOrEqual(t, len(payableEvents), 1)
}

func TestReconciliationService_AutoReconcilePayment_PartialAllocation(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(500), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	payable := createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcilePayment(context.Background(), voucher, []AccountPayable{payable})

	require.NoError(t, err)
	assert.Len(t, result.UpdatedPayables, 1)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(500)))
	assert.True(t, result.RemainingUnallocated.IsZero())
	assert.True(t, result.FullyReconciled)

	// Verify payable is partially paid
	assert.Equal(t, PayableStatusPartial, result.UpdatedPayables[0].Status)
	assert.True(t, result.UpdatedPayables[0].OutstandingAmount.Equal(decimal.NewFromInt(500)))
}

func TestReconciliationService_AutoReconcilePayment_VoucherExceedsPayables(t *testing.T) {
	service := NewReconciliationService()
	tenantID := uuid.New()
	supplierID := uuid.New()

	voucher := createPaymentVoucherForReconciliation(t, tenantID, supplierID, decimal.NewFromInt(2000), true)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	payable := createPayableForReconciliation(t, tenantID, supplierID, "AP-001", decimal.NewFromInt(1000), &dueDate)

	result, err := service.AutoReconcilePayment(context.Background(), voucher, []AccountPayable{payable})

	require.NoError(t, err)
	assert.True(t, result.TotalReconciled.Equal(decimal.NewFromInt(1000)))
	assert.True(t, result.RemainingUnallocated.Equal(decimal.NewFromInt(1000)))
	assert.False(t, result.FullyReconciled)

	// Voucher should be CONFIRMED (not fully allocated)
	assert.Equal(t, VoucherStatusConfirmed, result.PaymentVoucher.Status)
}
