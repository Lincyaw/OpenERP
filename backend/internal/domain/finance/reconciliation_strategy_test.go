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

func TestReconciliationStrategyType(t *testing.T) {
	t.Run("IsValid returns true for valid types", func(t *testing.T) {
		assert.True(t, ReconciliationStrategyTypeFIFO.IsValid())
		assert.True(t, ReconciliationStrategyTypeManual.IsValid())
	})

	t.Run("IsValid returns false for invalid types", func(t *testing.T) {
		assert.False(t, ReconciliationStrategyType("INVALID").IsValid())
		assert.False(t, ReconciliationStrategyType("").IsValid())
	})

	t.Run("String returns correct string", func(t *testing.T) {
		assert.Equal(t, "FIFO", ReconciliationStrategyTypeFIFO.String())
		assert.Equal(t, "MANUAL", ReconciliationStrategyTypeManual.String())
	})

	t.Run("AllReconciliationStrategyTypes returns all types", func(t *testing.T) {
		types := AllReconciliationStrategyTypes()
		assert.Len(t, types, 2)
		assert.Contains(t, types, ReconciliationStrategyTypeFIFO)
		assert.Contains(t, types, ReconciliationStrategyTypeManual)
	})
}

func TestFIFOReconciliationStrategy(t *testing.T) {
	t.Run("NewFIFOReconciliationStrategy creates valid strategy", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		assert.NotNil(t, strategy)
		assert.Equal(t, "fifo_reconciliation", strategy.Name())
		assert.Equal(t, ReconciliationStrategyTypeFIFO, strategy.StrategyType())
	})

	t.Run("Allocate with zero amount returns error", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		targets := []AllocationTarget{
			{ID: uuid.New(), Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100)},
		}
		_, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.Zero), targets)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be positive")
	})

	t.Run("Allocate with negative amount returns error", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		targets := []AllocationTarget{
			{ID: uuid.New(), Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100)},
		}
		_, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(-100)), targets)
		assert.Error(t, err)
	})

	t.Run("Allocate with no targets returns empty result", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), []AllocationTarget{})
		require.NoError(t, err)
		assert.Empty(t, result.Allocations)
		assert.True(t, result.TotalAllocated.IsZero())
		assert.True(t, result.RemainingAmount.Equal(decimal.NewFromInt(100)))
		assert.False(t, result.FullyReconciled)
	})

	t.Run("Allocate sorts by due date FIFO", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		now := time.Now()
		earlier := now.Add(-24 * time.Hour)
		later := now.Add(24 * time.Hour)

		id1 := uuid.New()
		id2 := uuid.New()
		id3 := uuid.New()

		targets := []AllocationTarget{
			{ID: id2, Number: "AR-002", OutstandingAmount: decimal.NewFromInt(100), DueDate: &later, CreatedAt: now},
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), DueDate: &earlier, CreatedAt: now},
			{ID: id3, Number: "AR-003", OutstandingAmount: decimal.NewFromInt(100), DueDate: &now, CreatedAt: now},
		}

		// Allocate 150 - should go to earliest due date first
		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(150)), targets)
		require.NoError(t, err)

		assert.Len(t, result.Allocations, 2)
		// First allocation should be to AR-001 (earliest due date)
		assert.Equal(t, id1, result.Allocations[0].TargetID)
		assert.Equal(t, "AR-001", result.Allocations[0].TargetNumber)
		assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(100)))

		// Second allocation should be to AR-003 (middle due date)
		assert.Equal(t, id3, result.Allocations[1].TargetID)
		assert.True(t, result.Allocations[1].Amount.Equal(decimal.NewFromInt(50)))
	})

	t.Run("Allocate sorts by creation date when no due date", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		now := time.Now()
		earlier := now.Add(-24 * time.Hour)

		id1 := uuid.New()
		id2 := uuid.New()

		targets := []AllocationTarget{
			{ID: id2, Number: "AR-002", OutstandingAmount: decimal.NewFromInt(100), DueDate: nil, CreatedAt: now},
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), DueDate: nil, CreatedAt: earlier},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(150)), targets)
		require.NoError(t, err)

		assert.Len(t, result.Allocations, 2)
		// First allocation should be to AR-001 (earlier creation date)
		assert.Equal(t, id1, result.Allocations[0].TargetID)
	})

	t.Run("Allocate puts items with due date before items without", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		now := time.Now()
		dueDate := now.Add(7 * 24 * time.Hour)
		earlier := now.Add(-24 * time.Hour)

		id1 := uuid.New() // Has due date
		id2 := uuid.New() // No due date but created earlier

		targets := []AllocationTarget{
			{ID: id2, Number: "AR-002", OutstandingAmount: decimal.NewFromInt(100), DueDate: nil, CreatedAt: earlier},
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), DueDate: &dueDate, CreatedAt: now},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(50)), targets)
		require.NoError(t, err)

		// AR-001 should be first because it has a due date
		assert.Equal(t, id1, result.Allocations[0].TargetID)
	})

	t.Run("Allocate fully allocates amount when sufficient targets", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		now := time.Now()

		targets := []AllocationTarget{
			{ID: uuid.New(), Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: now},
			{ID: uuid.New(), Number: "AR-002", OutstandingAmount: decimal.NewFromInt(200), CreatedAt: now.Add(time.Hour)},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(250)), targets)
		require.NoError(t, err)

		assert.True(t, result.TotalAllocated.Equal(decimal.NewFromInt(250)))
		assert.True(t, result.RemainingAmount.IsZero())
		assert.True(t, result.FullyReconciled)
		assert.Len(t, result.TargetsFullyPaid, 1)    // AR-001 fully paid
		assert.Len(t, result.TargetsPartiallyPaid, 1) // AR-002 partially paid
	})

	t.Run("Allocate tracks fully and partially paid targets", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		now := time.Now()

		id1 := uuid.New()
		id2 := uuid.New()
		id3 := uuid.New()

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: now},
			{ID: id2, Number: "AR-002", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: now.Add(time.Hour)},
			{ID: id3, Number: "AR-003", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: now.Add(2 * time.Hour)},
		}

		// Allocate 150: fully pays AR-001, partially pays AR-002
		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(150)), targets)
		require.NoError(t, err)

		assert.Contains(t, result.TargetsFullyPaid, id1)
		assert.Contains(t, result.TargetsPartiallyPaid, id2)
		assert.NotContains(t, result.TargetsFullyPaid, id3)
		assert.NotContains(t, result.TargetsPartiallyPaid, id3)
	})

	t.Run("Allocate skips targets with zero outstanding", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		now := time.Now()

		id1 := uuid.New()
		id2 := uuid.New()

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.Zero, CreatedAt: now},
			{ID: id2, Number: "AR-002", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: now.Add(time.Hour)},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(50)), targets)
		require.NoError(t, err)

		assert.Len(t, result.Allocations, 1)
		assert.Equal(t, id2, result.Allocations[0].TargetID)
	})
}

func TestFIFOReconciliationStrategy_AllocateReceipt(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("AllocateReceipt with nil voucher returns error", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		_, err := strategy.AllocateReceipt(nil, []AccountReceivable{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("AllocateReceipt with no unallocated amount returns error", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		voucher, _ := NewReceiptVoucher(
			tenantID, "RV-001", customerID, "Customer",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)),
			PaymentMethodCash, time.Now(),
		)
		voucher.AllocatedAmount = decimal.NewFromInt(100)
		voucher.UnallocatedAmount = decimal.Zero

		_, err := strategy.AllocateReceipt(voucher, []AccountReceivable{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no unallocated")
	})

	t.Run("AllocateReceipt filters and allocates to valid receivables", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		voucher, _ := NewReceiptVoucher(
			tenantID, "RV-001", customerID, "Customer",
			valueobject.NewMoneyCNY(decimal.NewFromInt(150)),
			PaymentMethodCash, time.Now(),
		)

		now := time.Now()
		earlier := now.Add(-24 * time.Hour)

		// Create receivables
		ar1, _ := NewAccountReceivable(tenantID, "AR-001", customerID, "Customer",
			SourceTypeSalesOrder, uuid.New(), "SO-001",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)), &earlier)
		ar1.CreatedAt = now

		ar2, _ := NewAccountReceivable(tenantID, "AR-002", customerID, "Customer",
			SourceTypeSalesOrder, uuid.New(), "SO-002",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)), nil)
		ar2.CreatedAt = now.Add(time.Hour)

		// One paid receivable (should be skipped)
		ar3, _ := NewAccountReceivable(tenantID, "AR-003", customerID, "Customer",
			SourceTypeSalesOrder, uuid.New(), "SO-003",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)), nil)
		ar3.Status = ReceivableStatusPaid
		ar3.OutstandingAmount = decimal.Zero
		ar3.CreatedAt = now.Add(2 * time.Hour)

		receivables := []AccountReceivable{*ar1, *ar2, *ar3}

		result, err := strategy.AllocateReceipt(voucher, receivables)
		require.NoError(t, err)

		// Should allocate to AR-001 and AR-002, skip AR-003
		assert.Len(t, result.Allocations, 2)
		assert.True(t, result.TotalAllocated.Equal(decimal.NewFromInt(150)))
	})
}

func TestFIFOReconciliationStrategy_AllocatePayment(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()

	t.Run("AllocatePayment with nil voucher returns error", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		_, err := strategy.AllocatePayment(nil, []AccountPayable{})
		assert.Error(t, err)
	})

	t.Run("AllocatePayment filters and allocates to valid payables", func(t *testing.T) {
		strategy := NewFIFOReconciliationStrategy()
		voucher := createReconciliationTestPaymentVoucher(tenantID, supplierID, decimal.NewFromInt(150))

		now := time.Now()

		// Create payables
		ap1, _ := NewAccountPayable(tenantID, "AP-001", supplierID, "Supplier",
			PayableSourceTypePurchaseOrder, uuid.New(), "PO-001",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)), nil)
		ap1.CreatedAt = now

		ap2, _ := NewAccountPayable(tenantID, "AP-002", supplierID, "Supplier",
			PayableSourceTypePurchaseOrder, uuid.New(), "PO-002",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)), nil)
		ap2.CreatedAt = now.Add(time.Hour)

		payables := []AccountPayable{*ap1, *ap2}

		result, err := strategy.AllocatePayment(voucher, payables)
		require.NoError(t, err)

		assert.Len(t, result.Allocations, 2)
		assert.True(t, result.TotalAllocated.Equal(decimal.NewFromInt(150)))
	})
}

func TestManualReconciliationStrategy(t *testing.T) {
	t.Run("NewManualReconciliationStrategy creates valid strategy", func(t *testing.T) {
		allocations := []ManualAllocationRequest{
			{TargetID: uuid.New(), Amount: decimal.NewFromInt(100)},
		}
		strategy := NewManualReconciliationStrategy(allocations)
		assert.NotNil(t, strategy)
		assert.Equal(t, "manual_reconciliation", strategy.Name())
		assert.Equal(t, ReconciliationStrategyTypeManual, strategy.StrategyType())
		assert.Len(t, strategy.GetAllocations(), 1)
	})

	t.Run("Allocate with zero amount returns error", func(t *testing.T) {
		strategy := NewManualReconciliationStrategy(nil)
		targets := []AllocationTarget{
			{ID: uuid.New(), Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100)},
		}
		_, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.Zero), targets)
		assert.Error(t, err)
	})

	t.Run("Allocate follows manual order not FIFO", func(t *testing.T) {
		now := time.Now()
		earlier := now.Add(-24 * time.Hour)

		id1 := uuid.New()
		id2 := uuid.New()

		// Request allocation to id2 first, then id1
		allocations := []ManualAllocationRequest{
			{TargetID: id2, Amount: decimal.NewFromInt(50)},
			{TargetID: id1, Amount: decimal.NewFromInt(50)},
		}
		strategy := NewManualReconciliationStrategy(allocations)

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: earlier},
			{ID: id2, Number: "AR-002", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: now},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), targets)
		require.NoError(t, err)

		// Should follow manual order: id2 first
		assert.Equal(t, id2, result.Allocations[0].TargetID)
		assert.Equal(t, id1, result.Allocations[1].TargetID)
	})

	t.Run("Allocate respects specified amounts", func(t *testing.T) {
		id1 := uuid.New()

		allocations := []ManualAllocationRequest{
			{TargetID: id1, Amount: decimal.NewFromInt(30)}, // Request only 30
		}
		strategy := NewManualReconciliationStrategy(allocations)

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: time.Now()},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), targets)
		require.NoError(t, err)

		// Should only allocate 30 as requested
		assert.Len(t, result.Allocations, 1)
		assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(30)))
		assert.True(t, result.RemainingAmount.Equal(decimal.NewFromInt(70)))
	})

	t.Run("Allocate with zero request amount allocates full outstanding", func(t *testing.T) {
		id1 := uuid.New()

		allocations := []ManualAllocationRequest{
			{TargetID: id1, Amount: decimal.Zero}, // Zero means allocate as much as possible
		}
		strategy := NewManualReconciliationStrategy(allocations)

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(80), CreatedAt: time.Now()},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), targets)
		require.NoError(t, err)

		// Should allocate full outstanding (80)
		assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(80)))
	})

	t.Run("Allocate caps at outstanding amount", func(t *testing.T) {
		id1 := uuid.New()

		allocations := []ManualAllocationRequest{
			{TargetID: id1, Amount: decimal.NewFromInt(200)}, // Request more than outstanding
		}
		strategy := NewManualReconciliationStrategy(allocations)

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(80), CreatedAt: time.Now()},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), targets)
		require.NoError(t, err)

		// Should cap at outstanding (80)
		assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(80)))
	})

	t.Run("Allocate caps at available amount", func(t *testing.T) {
		id1 := uuid.New()

		allocations := []ManualAllocationRequest{
			{TargetID: id1, Amount: decimal.NewFromInt(200)}, // Request more than available
		}
		strategy := NewManualReconciliationStrategy(allocations)

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(500), CreatedAt: time.Now()},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), targets)
		require.NoError(t, err)

		// Should cap at available (100)
		assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(100)))
	})

	t.Run("Allocate skips invalid target IDs", func(t *testing.T) {
		id1 := uuid.New()
		invalidID := uuid.New()

		allocations := []ManualAllocationRequest{
			{TargetID: invalidID, Amount: decimal.NewFromInt(50)},
			{TargetID: id1, Amount: decimal.NewFromInt(50)},
		}
		strategy := NewManualReconciliationStrategy(allocations)

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: time.Now()},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), targets)
		require.NoError(t, err)

		// Should only have allocation to id1
		assert.Len(t, result.Allocations, 1)
		assert.Equal(t, id1, result.Allocations[0].TargetID)
	})

	t.Run("Allocate handles multiple allocations to different targets", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()

		allocations := []ManualAllocationRequest{
			{TargetID: id1, Amount: decimal.NewFromInt(60)},
			{TargetID: id2, Amount: decimal.NewFromInt(40)},
		}
		strategy := NewManualReconciliationStrategy(allocations)

		targets := []AllocationTarget{
			{ID: id1, Number: "AR-001", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: time.Now()},
			{ID: id2, Number: "AR-002", OutstandingAmount: decimal.NewFromInt(100), CreatedAt: time.Now()},
		}

		result, err := strategy.Allocate(valueobject.NewMoneyCNY(decimal.NewFromInt(100)), targets)
		require.NoError(t, err)

		assert.Len(t, result.Allocations, 2)
		assert.True(t, result.TotalAllocated.Equal(decimal.NewFromInt(100)))
		assert.True(t, result.RemainingAmount.IsZero())
		assert.True(t, result.FullyReconciled)
	})
}

func TestManualReconciliationStrategy_AllocateReceipt(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("AllocateReceipt with nil voucher returns error", func(t *testing.T) {
		strategy := NewManualReconciliationStrategy(nil)
		_, err := strategy.AllocateReceipt(nil, []AccountReceivable{})
		assert.Error(t, err)
	})

	t.Run("AllocateReceipt with no unallocated amount returns error", func(t *testing.T) {
		strategy := NewManualReconciliationStrategy(nil)
		voucher, _ := NewReceiptVoucher(
			tenantID, "RV-001", customerID, "Customer",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)),
			PaymentMethodCash, time.Now(),
		)
		voucher.AllocatedAmount = decimal.NewFromInt(100)
		voucher.UnallocatedAmount = decimal.Zero

		_, err := strategy.AllocateReceipt(voucher, []AccountReceivable{})
		assert.Error(t, err)
	})

	t.Run("AllocateReceipt allocates to specified receivables", func(t *testing.T) {
		ar, _ := NewAccountReceivable(tenantID, "AR-001", customerID, "Customer",
			SourceTypeSalesOrder, uuid.New(), "SO-001",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)), nil)

		allocations := []ManualAllocationRequest{
			{TargetID: ar.ID, Amount: decimal.NewFromInt(50)},
		}
		strategy := NewManualReconciliationStrategy(allocations)

		voucher, _ := NewReceiptVoucher(
			tenantID, "RV-001", customerID, "Customer",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)),
			PaymentMethodCash, time.Now(),
		)

		result, err := strategy.AllocateReceipt(voucher, []AccountReceivable{*ar})
		require.NoError(t, err)

		assert.Len(t, result.Allocations, 1)
		assert.Equal(t, ar.ID, result.Allocations[0].TargetID)
		assert.True(t, result.Allocations[0].Amount.Equal(decimal.NewFromInt(50)))
	})
}

func TestManualReconciliationStrategy_AllocatePayment(t *testing.T) {
	tenantID := uuid.New()
	supplierID := uuid.New()

	t.Run("AllocatePayment with nil voucher returns error", func(t *testing.T) {
		strategy := NewManualReconciliationStrategy(nil)
		_, err := strategy.AllocatePayment(nil, []AccountPayable{})
		assert.Error(t, err)
	})

	t.Run("AllocatePayment allocates to specified payables", func(t *testing.T) {
		ap, _ := NewAccountPayable(tenantID, "AP-001", supplierID, "Supplier",
			PayableSourceTypePurchaseOrder, uuid.New(), "PO-001",
			valueobject.NewMoneyCNY(decimal.NewFromInt(100)), nil)

		allocations := []ManualAllocationRequest{
			{TargetID: ap.ID, Amount: decimal.NewFromInt(50)},
		}
		strategy := NewManualReconciliationStrategy(allocations)

		voucher := createReconciliationTestPaymentVoucher(tenantID, supplierID, decimal.NewFromInt(100))

		result, err := strategy.AllocatePayment(voucher, []AccountPayable{*ap})
		require.NoError(t, err)

		assert.Len(t, result.Allocations, 1)
		assert.Equal(t, ap.ID, result.Allocations[0].TargetID)
	})
}

func TestReconciliationStrategyFactory(t *testing.T) {
	factory := NewReconciliationStrategyFactory()

	t.Run("CreateFIFOStrategy creates FIFO strategy", func(t *testing.T) {
		strategy := factory.CreateFIFOStrategy()
		assert.NotNil(t, strategy)
		assert.Equal(t, ReconciliationStrategyTypeFIFO, strategy.StrategyType())
	})

	t.Run("CreateManualStrategy creates manual strategy", func(t *testing.T) {
		allocations := []ManualAllocationRequest{
			{TargetID: uuid.New(), Amount: decimal.NewFromInt(100)},
		}
		strategy := factory.CreateManualStrategy(allocations)
		assert.NotNil(t, strategy)
		assert.Equal(t, ReconciliationStrategyTypeManual, strategy.StrategyType())
	})

	t.Run("GetStrategy returns FIFO strategy", func(t *testing.T) {
		strategy, err := factory.GetStrategy(ReconciliationStrategyTypeFIFO, nil)
		require.NoError(t, err)
		assert.Equal(t, ReconciliationStrategyTypeFIFO, strategy.StrategyType())
	})

	t.Run("GetStrategy returns manual strategy with allocations", func(t *testing.T) {
		allocations := []ManualAllocationRequest{
			{TargetID: uuid.New(), Amount: decimal.NewFromInt(100)},
		}
		strategy, err := factory.GetStrategy(ReconciliationStrategyTypeManual, allocations)
		require.NoError(t, err)
		assert.Equal(t, ReconciliationStrategyTypeManual, strategy.StrategyType())
	})

	t.Run("GetStrategy returns error for manual without allocations", func(t *testing.T) {
		_, err := factory.GetStrategy(ReconciliationStrategyTypeManual, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires allocation")
	})

	t.Run("GetStrategy returns error for invalid type", func(t *testing.T) {
		_, err := factory.GetStrategy(ReconciliationStrategyType("INVALID"), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Unknown")
	})
}

func TestAllocationResult(t *testing.T) {
	t.Run("AllocationResult fields are set correctly", func(t *testing.T) {
		id := uuid.New()
		result := AllocationResult{
			TargetID:     id,
			TargetNumber: "AR-001",
			Amount:       decimal.NewFromInt(100),
		}
		assert.Equal(t, id, result.TargetID)
		assert.Equal(t, "AR-001", result.TargetNumber)
		assert.True(t, result.Amount.Equal(decimal.NewFromInt(100)))
	})
}

func TestReconciliationResult(t *testing.T) {
	t.Run("ReconciliationResult fields are set correctly", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		result := ReconciliationResult{
			Allocations: []AllocationResult{
				{TargetID: id1, TargetNumber: "AR-001", Amount: decimal.NewFromInt(100)},
			},
			TotalAllocated:      decimal.NewFromInt(100),
			RemainingAmount:     decimal.NewFromInt(50),
			FullyReconciled:     false,
			TargetsFullyPaid:    []uuid.UUID{id1},
			TargetsPartiallyPaid: []uuid.UUID{id2},
		}
		assert.Len(t, result.Allocations, 1)
		assert.True(t, result.TotalAllocated.Equal(decimal.NewFromInt(100)))
		assert.True(t, result.RemainingAmount.Equal(decimal.NewFromInt(50)))
		assert.False(t, result.FullyReconciled)
		assert.Contains(t, result.TargetsFullyPaid, id1)
		assert.Contains(t, result.TargetsPartiallyPaid, id2)
	})
}

// Helper function to create test payment voucher for reconciliation tests
func createReconciliationTestPaymentVoucher(tenantID, supplierID uuid.UUID, amount decimal.Decimal) *PaymentVoucher {
	voucher, _ := NewPaymentVoucher(
		tenantID, "PV-001", supplierID, "Supplier",
		valueobject.NewMoneyCNY(amount),
		PaymentMethodBankTransfer, time.Now(),
	)
	return voucher
}
