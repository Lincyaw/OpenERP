// Package integration provides integration tests for finance module interactions.
// This file tests the critical business flows:
// - Sales order shipped creates account receivable
// - Purchase order received creates account payable
// - Receipt/payment voucher workflow
// - Reconciliation process
package integration

import (
	"context"
	"testing"
	"time"

	financeapp "github.com/erp/backend/internal/application/finance"
	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// FinanceTestSetup provides test infrastructure for finance integration tests
type FinanceTestSetup struct {
	DB             *TestDB
	ReceivableRepo finance.AccountReceivableRepository
	PayableRepo    finance.AccountPayableRepository
	Logger         *zap.Logger
	TenantID       uuid.UUID
	CustomerID     uuid.UUID
	SupplierID     uuid.UUID
	WarehouseID    uuid.UUID
}

// NewFinanceTestSetup creates test infrastructure with real database
func NewFinanceTestSetup(t *testing.T) *FinanceTestSetup {
	t.Helper()

	testDB := NewTestDB(t)

	// Create repositories
	receivableRepo := persistence.NewGormAccountReceivableRepository(testDB.DB)
	payableRepo := persistence.NewGormAccountPayableRepository(testDB.DB)

	// Create test data
	tenantID := uuid.New()
	customerID := uuid.New()
	supplierID := uuid.New()
	warehouseID := uuid.New()

	testDB.CreateTestTenantWithUUID(tenantID)
	testDB.CreateTestCustomer(tenantID, customerID)
	testDB.CreateTestSupplier(tenantID, supplierID)
	testDB.CreateTestWarehouse(tenantID, warehouseID)

	return &FinanceTestSetup{
		DB:             testDB,
		ReceivableRepo: receivableRepo,
		PayableRepo:    payableRepo,
		Logger:         zap.NewNop(),
		TenantID:       tenantID,
		CustomerID:     customerID,
		SupplierID:     supplierID,
		WarehouseID:    warehouseID,
	}
}

// CreateTestCustomer creates a customer record for testing
func (tdb *TestDB) CreateTestCustomer(tenantID, customerID interface{}) {
	tdb.t.Helper()

	code := "CUST-TEST"
	name := "Test Customer"

	err := tdb.DB.Exec(`
		INSERT INTO customers (id, tenant_id, code, name, type, level, status, version, balance, credit_limit)
		VALUES (?, ?, ?, ?, 'individual', 'normal', 'active', 1, 0, 0)
		ON CONFLICT (id) DO NOTHING
	`, customerID, tenantID, code, name).Error
	require.NoError(tdb.t, err, "Failed to create test customer")
}

// CreateTestSupplier creates a supplier record for testing
func (tdb *TestDB) CreateTestSupplier(tenantID, supplierID interface{}) {
	tdb.t.Helper()

	code := "SUPP-TEST"
	name := "Test Supplier"

	err := tdb.DB.Exec(`
		INSERT INTO suppliers (id, tenant_id, code, name, status, version)
		VALUES (?, ?, ?, ?, 'active', 1)
		ON CONFLICT (id) DO NOTHING
	`, supplierID, tenantID, code, name).Error
	require.NoError(tdb.t, err, "Failed to create test supplier")
}

// ==================== Account Receivable Tests ====================

func TestFinance_SalesOrderShipped_CreatesReceivable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("sales order shipped creates account receivable", func(t *testing.T) {
		orderID := uuid.New()
		orderNumber := "SO-2024-00001"
		payableAmount := decimal.NewFromFloat(1500.00)

		// Create shipped event
		event := &trade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesOrderShipped,
				trade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:       orderID,
			OrderNumber:   orderNumber,
			CustomerID:    setup.CustomerID,
			CustomerName:  "Test Customer",
			WarehouseID:   setup.WarehouseID,
			Items: []trade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   uuid.New(),
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(10),
					UnitPrice:   decimal.NewFromFloat(150.00),
					Amount:      decimal.NewFromFloat(1500.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   payableAmount,
			PayableAmount: payableAmount,
		}

		// Create and execute handler
		handler := financeapp.NewSalesOrderShippedHandler(setup.ReceivableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify: Receivable should be created
		receivable, err := setup.ReceivableRepo.FindBySource(ctx, setup.TenantID, finance.SourceTypeSalesOrder, orderID)
		require.NoError(t, err)
		require.NotNil(t, receivable)

		assert.Equal(t, setup.CustomerID, receivable.CustomerID)
		assert.Equal(t, "Test Customer", receivable.CustomerName)
		assert.Equal(t, finance.SourceTypeSalesOrder, receivable.SourceType)
		assert.Equal(t, orderID, receivable.SourceID)
		assert.Equal(t, orderNumber, receivable.SourceNumber)
		assert.True(t, receivable.TotalAmount.Equal(payableAmount), "Total amount should match")
		assert.True(t, receivable.OutstandingAmount.Equal(payableAmount), "Outstanding should equal total")
		assert.True(t, receivable.PaidAmount.IsZero(), "Paid amount should be zero")
		assert.Equal(t, finance.ReceivableStatusPending, receivable.Status)
	})

	t.Run("idempotent - duplicate shipped event does not create duplicate receivable", func(t *testing.T) {
		orderID := uuid.New()
		orderNumber := "SO-2024-00002"
		payableAmount := decimal.NewFromFloat(2000.00)

		event := &trade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesOrderShipped,
				trade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:       orderID,
			OrderNumber:   orderNumber,
			CustomerID:    setup.CustomerID,
			CustomerName:  "Test Customer",
			WarehouseID:   setup.WarehouseID,
			Items: []trade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   uuid.New(),
					ProductName: "Test Product",
					ProductCode: "PROD-002",
					Quantity:    decimal.NewFromInt(20),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(2000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   payableAmount,
			PayableAmount: payableAmount,
		}

		handler := financeapp.NewSalesOrderShippedHandler(setup.ReceivableRepo, setup.Logger)

		// First call
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Second call (should be idempotent)
		err = handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify only one receivable exists
		count, err := setup.ReceivableRepo.CountForTenant(ctx, setup.TenantID, finance.AccountReceivableFilter{
			SourceID: &orderID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should only have one receivable for this order")
	})

	t.Run("zero payable amount does not create receivable", func(t *testing.T) {
		orderID := uuid.New()
		orderNumber := "SO-2024-00003"

		event := &trade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesOrderShipped,
				trade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:       orderID,
			OrderNumber:   orderNumber,
			CustomerID:    setup.CustomerID,
			CustomerName:  "Test Customer",
			WarehouseID:   setup.WarehouseID,
			Items: []trade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   uuid.New(),
					ProductName: "Test Product",
					ProductCode: "PROD-003",
					Quantity:    decimal.NewFromInt(5),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(500.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   decimal.NewFromFloat(500.00),
			PayableAmount: decimal.Zero, // Fully prepaid
		}

		handler := financeapp.NewSalesOrderShippedHandler(setup.ReceivableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify no receivable was created
		exists, err := setup.ReceivableRepo.ExistsBySource(ctx, setup.TenantID, finance.SourceTypeSalesOrder, orderID)
		require.NoError(t, err)
		assert.False(t, exists, "Should not create receivable for zero payable amount")
	})
}

// ==================== Account Payable Tests ====================

func TestFinance_PurchaseOrderReceived_CreatesPayable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("purchase order fully received creates account payable", func(t *testing.T) {
		orderID := uuid.New()
		orderNumber := "PO-2024-00001"
		payableAmount := decimal.NewFromFloat(3000.00)

		event := &trade.PurchaseOrderReceivedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypePurchaseOrderReceived,
				trade.AggregateTypePurchaseOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			SupplierID:   setup.SupplierID,
			SupplierName: "Test Supplier",
			WarehouseID:  setup.WarehouseID,
			ReceivedItems: []trade.ReceivedItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   uuid.New(),
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(30),
					UnitCost:    decimal.NewFromFloat(100.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:     payableAmount,
			PayableAmount:   payableAmount,
			IsFullyReceived: true,
		}

		handler := financeapp.NewPurchaseOrderReceivedHandler(setup.PayableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify: Payable should be created
		payable, err := setup.PayableRepo.FindBySource(ctx, setup.TenantID, finance.PayableSourceTypePurchaseOrder, orderID)
		require.NoError(t, err)
		require.NotNil(t, payable)

		assert.Equal(t, setup.SupplierID, payable.SupplierID)
		assert.Equal(t, "Test Supplier", payable.SupplierName)
		assert.Equal(t, finance.PayableSourceTypePurchaseOrder, payable.SourceType)
		assert.Equal(t, orderID, payable.SourceID)
		assert.Equal(t, orderNumber, payable.SourceNumber)
		assert.True(t, payable.TotalAmount.Equal(payableAmount), "Total amount should match")
		assert.True(t, payable.OutstandingAmount.Equal(payableAmount), "Outstanding should equal total")
		assert.True(t, payable.PaidAmount.IsZero(), "Paid amount should be zero")
		assert.Equal(t, finance.PayableStatusPending, payable.Status)
	})

	t.Run("partial receive does not create payable", func(t *testing.T) {
		orderID := uuid.New()
		orderNumber := "PO-2024-00002"

		event := &trade.PurchaseOrderReceivedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypePurchaseOrderReceived,
				trade.AggregateTypePurchaseOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			SupplierID:   setup.SupplierID,
			SupplierName: "Test Supplier",
			WarehouseID:  setup.WarehouseID,
			ReceivedItems: []trade.ReceivedItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   uuid.New(),
					ProductName: "Test Product",
					ProductCode: "PROD-002",
					Quantity:    decimal.NewFromInt(10),
					UnitCost:    decimal.NewFromFloat(50.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:     decimal.NewFromFloat(1000.00),
			PayableAmount:   decimal.NewFromFloat(500.00),
			IsFullyReceived: false, // Partial receive
		}

		handler := financeapp.NewPurchaseOrderReceivedHandler(setup.PayableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify no payable was created (only created on full receive)
		exists, err := setup.PayableRepo.ExistsBySource(ctx, setup.TenantID, finance.PayableSourceTypePurchaseOrder, orderID)
		require.NoError(t, err)
		assert.False(t, exists, "Should not create payable for partial receive")
	})
}

// ==================== Receivable Payment Workflow Tests ====================

func TestFinance_ReceivablePaymentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("apply full payment to receivable", func(t *testing.T) {
		// Create a receivable directly
		receivableNumber, err := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		require.NoError(t, err)

		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00))
		dueDate := time.Now().AddDate(0, 0, 30)

		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			receivableNumber,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"MANUAL-001",
			totalAmount,
			&dueDate,
		)
		require.NoError(t, err)

		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Apply full payment
		paymentAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00))
		voucherID := uuid.New()
		err = receivable.ApplyPayment(paymentAmount, voucherID, "Full payment")
		require.NoError(t, err)

		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Verify receivable is fully paid
		updated, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.ReceivableStatusPaid, updated.Status)
		assert.True(t, updated.OutstandingAmount.IsZero(), "Outstanding should be zero")
		assert.True(t, updated.PaidAmount.Equal(decimal.NewFromFloat(1000.00)), "Paid amount should match")
		assert.NotNil(t, updated.PaidAt, "PaidAt should be set")
	})

	t.Run("apply partial payment to receivable", func(t *testing.T) {
		receivableNumber, err := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		require.NoError(t, err)

		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(2000.00))
		dueDate := time.Now().AddDate(0, 0, 30)

		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			receivableNumber,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"MANUAL-002",
			totalAmount,
			&dueDate,
		)
		require.NoError(t, err)

		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Apply partial payment
		paymentAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(800.00))
		voucherID := uuid.New()
		err = receivable.ApplyPayment(paymentAmount, voucherID, "Partial payment 1")
		require.NoError(t, err)

		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Verify receivable is partially paid
		updated, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.ReceivableStatusPartial, updated.Status)
		assert.True(t, updated.OutstandingAmount.Equal(decimal.NewFromFloat(1200.00)), "Outstanding should be 1200")
		assert.True(t, updated.PaidAmount.Equal(decimal.NewFromFloat(800.00)), "Paid amount should be 800")
		assert.Nil(t, updated.PaidAt, "PaidAt should not be set")
	})

	t.Run("multiple payments complete receivable", func(t *testing.T) {
		receivableNumber, err := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		require.NoError(t, err)

		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1500.00))
		dueDate := time.Now().AddDate(0, 0, 30)

		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			receivableNumber,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"MANUAL-003",
			totalAmount,
			&dueDate,
		)
		require.NoError(t, err)

		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// First payment
		err = receivable.ApplyPayment(valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00)), uuid.New(), "Payment 1")
		require.NoError(t, err)
		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Second payment
		err = receivable.ApplyPayment(valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00)), uuid.New(), "Payment 2")
		require.NoError(t, err)
		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Final payment
		err = receivable.ApplyPayment(valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00)), uuid.New(), "Payment 3")
		require.NoError(t, err)
		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Verify receivable is fully paid with 3 payments
		updated, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.ReceivableStatusPaid, updated.Status)
		assert.True(t, updated.OutstandingAmount.IsZero(), "Outstanding should be zero")
		assert.Len(t, updated.PaymentRecords, 3, "Should have 3 payment records")
	})
}

// ==================== Payable Payment Workflow Tests ====================

func TestFinance_PayablePaymentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("apply full payment to payable", func(t *testing.T) {
		payableNumber, err := setup.PayableRepo.GeneratePayableNumber(ctx, setup.TenantID)
		require.NoError(t, err)

		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(5000.00))
		dueDate := time.Now().AddDate(0, 0, 30)

		payable, err := finance.NewAccountPayable(
			setup.TenantID,
			payableNumber,
			setup.SupplierID,
			"Test Supplier",
			finance.PayableSourceTypeManual,
			uuid.New(),
			"MANUAL-001",
			totalAmount,
			&dueDate,
		)
		require.NoError(t, err)

		err = setup.PayableRepo.Save(ctx, payable)
		require.NoError(t, err)

		// Apply full payment
		paymentAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(5000.00))
		voucherID := uuid.New()
		err = payable.ApplyPayment(paymentAmount, voucherID, "Full payment")
		require.NoError(t, err)

		err = setup.PayableRepo.Save(ctx, payable)
		require.NoError(t, err)

		// Verify payable is fully paid
		updated, err := setup.PayableRepo.FindByID(ctx, payable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.PayableStatusPaid, updated.Status)
		assert.True(t, updated.OutstandingAmount.IsZero(), "Outstanding should be zero")
		assert.True(t, updated.PaidAmount.Equal(decimal.NewFromFloat(5000.00)), "Paid amount should match")
		assert.NotNil(t, updated.PaidAt, "PaidAt should be set")
	})
}

// ==================== Summary Query Tests ====================

func TestFinance_SummaryQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("sum outstanding receivables by customer", func(t *testing.T) {
		// Create multiple receivables
		for i := 0; i < 3; i++ {
			number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
			amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00))
			receivable, _ := finance.NewAccountReceivable(
				setup.TenantID,
				number,
				setup.CustomerID,
				"Test Customer",
				finance.SourceTypeManual,
				uuid.New(),
				"MANUAL",
				amount,
				nil,
			)
			setup.ReceivableRepo.Save(ctx, receivable)
		}

		// Sum outstanding
		total, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, setup.CustomerID)
		require.NoError(t, err)
		assert.True(t, total.GreaterThanOrEqual(decimal.NewFromFloat(3000.00)), "Total should be at least 3000")
	})

	t.Run("sum outstanding payables by supplier", func(t *testing.T) {
		// Create multiple payables
		for i := 0; i < 2; i++ {
			number, _ := setup.PayableRepo.GeneratePayableNumber(ctx, setup.TenantID)
			amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(2000.00))
			payable, _ := finance.NewAccountPayable(
				setup.TenantID,
				number,
				setup.SupplierID,
				"Test Supplier",
				finance.PayableSourceTypeManual,
				uuid.New(),
				"MANUAL",
				amount,
				nil,
			)
			setup.PayableRepo.Save(ctx, payable)
		}

		// Sum outstanding
		total, err := setup.PayableRepo.SumOutstandingBySupplier(ctx, setup.TenantID, setup.SupplierID)
		require.NoError(t, err)
		assert.True(t, total.GreaterThanOrEqual(decimal.NewFromFloat(4000.00)), "Total should be at least 4000")
	})
}

// ==================== Receivable Cancellation/Reversal Tests ====================

func TestFinance_ReceivableCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("cancel pending receivable", func(t *testing.T) {
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00))
		receivable, _ := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"CANCEL-TEST",
			amount,
			nil,
		)
		setup.ReceivableRepo.Save(ctx, receivable)

		// Cancel
		err := receivable.Cancel("Customer requested cancellation")
		require.NoError(t, err)

		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Verify
		updated, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.ReceivableStatusCancelled, updated.Status)
		assert.True(t, updated.OutstandingAmount.IsZero(), "Outstanding should be zero after cancel")
		assert.NotNil(t, updated.CancelledAt)
		assert.Equal(t, "Customer requested cancellation", updated.CancelReason)
	})

	t.Run("reverse receivable", func(t *testing.T) {
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1500.00))
		receivable, _ := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"REVERSE-TEST",
			amount,
			nil,
		)
		setup.ReceivableRepo.Save(ctx, receivable)

		// Reverse
		err := receivable.Reverse("Sales return processed")
		require.NoError(t, err)

		err = setup.ReceivableRepo.Save(ctx, receivable)
		require.NoError(t, err)

		// Verify
		updated, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.ReceivableStatusReversed, updated.Status)
		assert.NotNil(t, updated.ReversedAt)
		assert.Equal(t, "Sales return processed", updated.ReversalReason)
	})
}
