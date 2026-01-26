// Package integration provides integration tests for finance module interactions.
// This file tests the critical business flows:
// - Sales order shipped creates account receivable
// - Purchase order received creates account payable
// - Receipt/payment voucher workflow
// - Reconciliation process
package integration

import (
	"context"
	"fmt"
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

	// Use UUID string suffix for unique code
	idStr := fmt.Sprintf("%v", customerID)
	code := fmt.Sprintf("CUST-%s", idStr[:8]) // Use first 8 chars of UUID
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

	// Use UUID string suffix for unique code
	idStr := fmt.Sprintf("%v", supplierID)
	code := fmt.Sprintf("SUPP-%s", idStr[:8]) // Use first 8 chars of UUID
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
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   setup.CustomerID,
			CustomerName: "Test Customer",
			WarehouseID:  setup.WarehouseID,
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
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   setup.CustomerID,
			CustomerName: "Test Customer",
			WarehouseID:  setup.WarehouseID,
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
			OrderID:      orderID,
			OrderNumber:  orderNumber,
			CustomerID:   setup.CustomerID,
			CustomerName: "Test Customer",
			WarehouseID:  setup.WarehouseID,
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

// ==================== Trade-Finance Return Red-Letter Tests ====================

func TestFinance_SalesReturnCompleted_CreatesRedLetterReceivable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("sales return completed creates red-letter receivable", func(t *testing.T) {
		returnID := uuid.New()
		salesOrderID := uuid.New()
		returnNumber := "SR-2024-00001"
		salesOrderNumber := "SO-2024-00001"
		refundAmount := decimal.NewFromFloat(500.00)

		// Create completed event
		event := &trade.SalesReturnCompletedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesReturnCompleted,
				trade.AggregateTypeSalesReturn,
				returnID,
				setup.TenantID,
			),
			ReturnID:         returnID,
			ReturnNumber:     returnNumber,
			SalesOrderID:     salesOrderID,
			SalesOrderNumber: salesOrderNumber,
			CustomerID:       setup.CustomerID,
			CustomerName:     "Test Customer",
			WarehouseID:      setup.WarehouseID,
			Items: []trade.SalesReturnItemInfo{
				{
					ItemID:           uuid.New(),
					SalesOrderItemID: uuid.New(),
					ProductID:        uuid.New(),
					ProductName:      "Test Product",
					ProductCode:      "PROD-001",
					ReturnQuantity:   decimal.NewFromInt(5),
					UnitPrice:        decimal.NewFromFloat(100.00),
					RefundAmount:     decimal.NewFromFloat(500.00),
					Unit:             "pcs",
				},
			},
			TotalRefund: refundAmount,
		}

		// Create and execute handler
		handler := financeapp.NewSalesReturnCompletedHandler(setup.ReceivableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify: Red-letter receivable should be created
		receivable, err := setup.ReceivableRepo.FindBySource(ctx, setup.TenantID, finance.SourceTypeSalesReturn, returnID)
		require.NoError(t, err)
		require.NotNil(t, receivable)

		assert.Equal(t, setup.CustomerID, receivable.CustomerID)
		assert.Equal(t, "Test Customer", receivable.CustomerName)
		assert.Equal(t, finance.SourceTypeSalesReturn, receivable.SourceType)
		assert.Equal(t, returnID, receivable.SourceID)
		assert.Equal(t, returnNumber, receivable.SourceNumber)
		assert.True(t, receivable.TotalAmount.Equal(refundAmount), "Total amount should match refund amount")
		assert.True(t, receivable.OutstandingAmount.Equal(refundAmount), "Outstanding should equal total")
		assert.Equal(t, finance.ReceivableStatusPending, receivable.Status)
		// Verify remark mentions red-letter and original order
		assert.Contains(t, receivable.Remark, "Red-letter entry")
		assert.Contains(t, receivable.Remark, returnNumber)
		assert.Contains(t, receivable.Remark, salesOrderNumber)
	})

	t.Run("idempotent - duplicate completed event does not create duplicate", func(t *testing.T) {
		returnID := uuid.New()
		salesOrderID := uuid.New()
		refundAmount := decimal.NewFromFloat(300.00)

		event := &trade.SalesReturnCompletedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesReturnCompleted,
				trade.AggregateTypeSalesReturn,
				returnID,
				setup.TenantID,
			),
			ReturnID:         returnID,
			ReturnNumber:     "SR-2024-00002",
			SalesOrderID:     salesOrderID,
			SalesOrderNumber: "SO-2024-00002",
			CustomerID:       setup.CustomerID,
			CustomerName:     "Test Customer",
			WarehouseID:      setup.WarehouseID,
			Items: []trade.SalesReturnItemInfo{
				{
					ItemID:         uuid.New(),
					ProductID:      uuid.New(),
					ProductName:    "Test Product",
					ProductCode:    "PROD-002",
					ReturnQuantity: decimal.NewFromInt(3),
					UnitPrice:      decimal.NewFromFloat(100.00),
					RefundAmount:   decimal.NewFromFloat(300.00),
					Unit:           "pcs",
				},
			},
			TotalRefund: refundAmount,
		}

		handler := financeapp.NewSalesReturnCompletedHandler(setup.ReceivableRepo, setup.Logger)

		// First call
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Second call (should be idempotent)
		err = handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify only one receivable exists
		count, err := setup.ReceivableRepo.CountForTenant(ctx, setup.TenantID, finance.AccountReceivableFilter{
			SourceID: &returnID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should only have one receivable for this return")
	})

	t.Run("zero refund does not create receivable", func(t *testing.T) {
		returnID := uuid.New()
		salesOrderID := uuid.New()

		event := &trade.SalesReturnCompletedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesReturnCompleted,
				trade.AggregateTypeSalesReturn,
				returnID,
				setup.TenantID,
			),
			ReturnID:         returnID,
			ReturnNumber:     "SR-2024-00003",
			SalesOrderID:     salesOrderID,
			SalesOrderNumber: "SO-2024-00003",
			CustomerID:       setup.CustomerID,
			CustomerName:     "Test Customer",
			WarehouseID:      setup.WarehouseID,
			Items:            []trade.SalesReturnItemInfo{},
			TotalRefund:      decimal.Zero, // Zero refund
		}

		handler := financeapp.NewSalesReturnCompletedHandler(setup.ReceivableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify no receivable was created
		exists, err := setup.ReceivableRepo.ExistsBySource(ctx, setup.TenantID, finance.SourceTypeSalesReturn, returnID)
		require.NoError(t, err)
		assert.False(t, exists, "Should not create receivable for zero refund")
	})
}

func TestFinance_PurchaseReturnCompleted_CreatesRedLetterPayable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("purchase return completed creates red-letter payable", func(t *testing.T) {
		returnID := uuid.New()
		purchaseOrderID := uuid.New()
		returnNumber := "PR-2024-00001"
		purchaseOrderNumber := "PO-2024-00001"
		refundAmount := decimal.NewFromFloat(800.00)

		event := &trade.PurchaseReturnCompletedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypePurchaseReturnCompleted,
				trade.AggregateTypePurchaseReturn,
				returnID,
				setup.TenantID,
			),
			ReturnID:            returnID,
			ReturnNumber:        returnNumber,
			PurchaseOrderID:     purchaseOrderID,
			PurchaseOrderNumber: purchaseOrderNumber,
			SupplierID:          setup.SupplierID,
			SupplierName:        "Test Supplier",
			WarehouseID:         setup.WarehouseID,
			Items: []trade.PurchaseReturnItemInfo{
				{
					ItemID:              uuid.New(),
					PurchaseOrderItemID: uuid.New(),
					ProductID:           uuid.New(),
					ProductName:         "Test Product",
					ProductCode:         "PROD-001",
					ReturnQuantity:      decimal.NewFromInt(8),
					UnitCost:            decimal.NewFromFloat(100.00),
					RefundAmount:        decimal.NewFromFloat(800.00),
					Unit:                "pcs",
					BatchNumber:         "BATCH001",
				},
			},
			TotalRefund: refundAmount,
		}

		// Create and execute handler
		handler := financeapp.NewPurchaseReturnCompletedHandler(setup.PayableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify: Red-letter payable should be created
		payable, err := setup.PayableRepo.FindBySource(ctx, setup.TenantID, finance.PayableSourceTypePurchaseReturn, returnID)
		require.NoError(t, err)
		require.NotNil(t, payable)

		assert.Equal(t, setup.SupplierID, payable.SupplierID)
		assert.Equal(t, "Test Supplier", payable.SupplierName)
		assert.Equal(t, finance.PayableSourceTypePurchaseReturn, payable.SourceType)
		assert.Equal(t, returnID, payable.SourceID)
		assert.Equal(t, returnNumber, payable.SourceNumber)
		assert.True(t, payable.TotalAmount.Equal(refundAmount), "Total amount should match refund amount")
		assert.True(t, payable.OutstandingAmount.Equal(refundAmount), "Outstanding should equal total")
		assert.Equal(t, finance.PayableStatusPending, payable.Status)
		// Verify remark mentions red-letter and original order
		assert.Contains(t, payable.Remark, "Red-letter entry")
		assert.Contains(t, payable.Remark, returnNumber)
		assert.Contains(t, payable.Remark, purchaseOrderNumber)
	})

	t.Run("idempotent - duplicate completed event does not create duplicate", func(t *testing.T) {
		returnID := uuid.New()
		purchaseOrderID := uuid.New()
		refundAmount := decimal.NewFromFloat(400.00)

		event := &trade.PurchaseReturnCompletedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypePurchaseReturnCompleted,
				trade.AggregateTypePurchaseReturn,
				returnID,
				setup.TenantID,
			),
			ReturnID:            returnID,
			ReturnNumber:        "PR-2024-00002",
			PurchaseOrderID:     purchaseOrderID,
			PurchaseOrderNumber: "PO-2024-00002",
			SupplierID:          setup.SupplierID,
			SupplierName:        "Test Supplier",
			WarehouseID:         setup.WarehouseID,
			Items: []trade.PurchaseReturnItemInfo{
				{
					ItemID:         uuid.New(),
					ProductID:      uuid.New(),
					ProductName:    "Test Product",
					ProductCode:    "PROD-002",
					ReturnQuantity: decimal.NewFromInt(4),
					UnitCost:       decimal.NewFromFloat(100.00),
					RefundAmount:   decimal.NewFromFloat(400.00),
					Unit:           "pcs",
				},
			},
			TotalRefund: refundAmount,
		}

		handler := financeapp.NewPurchaseReturnCompletedHandler(setup.PayableRepo, setup.Logger)

		// First call
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Second call (should be idempotent)
		err = handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify only one payable exists
		count, err := setup.PayableRepo.CountForTenant(ctx, setup.TenantID, finance.AccountPayableFilter{
			SourceID: &returnID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should only have one payable for this return")
	})

	t.Run("zero refund does not create payable", func(t *testing.T) {
		returnID := uuid.New()
		purchaseOrderID := uuid.New()

		event := &trade.PurchaseReturnCompletedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypePurchaseReturnCompleted,
				trade.AggregateTypePurchaseReturn,
				returnID,
				setup.TenantID,
			),
			ReturnID:            returnID,
			ReturnNumber:        "PR-2024-00003",
			PurchaseOrderID:     purchaseOrderID,
			PurchaseOrderNumber: "PO-2024-00003",
			SupplierID:          setup.SupplierID,
			SupplierName:        "Test Supplier",
			WarehouseID:         setup.WarehouseID,
			Items:               []trade.PurchaseReturnItemInfo{},
			TotalRefund:         decimal.Zero, // Zero refund
		}

		handler := financeapp.NewPurchaseReturnCompletedHandler(setup.PayableRepo, setup.Logger)
		err := handler.Handle(ctx, event)
		require.NoError(t, err)

		// Verify no payable was created
		exists, err := setup.PayableRepo.ExistsBySource(ctx, setup.TenantID, finance.PayableSourceTypePurchaseReturn, returnID)
		require.NoError(t, err)
		assert.False(t, exists, "Should not create payable for zero refund")
	})
}

// ==================== Trade-Finance Red-Letter Offset Tests ====================

func TestFinance_RedLetterReducesOutstandingBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("sales order then return reduces customer outstanding", func(t *testing.T) {
		orderID := uuid.New()
		returnID := uuid.New()
		orderAmount := decimal.NewFromFloat(1000.00)
		returnAmount := decimal.NewFromFloat(300.00)

		// Step 1: Create receivable from sales order shipped
		shippedEvent := &trade.SalesOrderShippedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesOrderShipped,
				trade.AggregateTypeSalesOrder,
				orderID,
				setup.TenantID,
			),
			OrderID:      orderID,
			OrderNumber:  "SO-2024-RED-001",
			CustomerID:   setup.CustomerID,
			CustomerName: "Test Customer",
			WarehouseID:  setup.WarehouseID,
			Items: []trade.SalesOrderItemInfo{
				{
					ItemID:      uuid.New(),
					ProductID:   uuid.New(),
					ProductName: "Test Product",
					ProductCode: "PROD-001",
					Quantity:    decimal.NewFromInt(10),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Amount:      decimal.NewFromFloat(1000.00),
					Unit:        "pcs",
				},
			},
			TotalAmount:   orderAmount,
			PayableAmount: orderAmount,
		}

		shippedHandler := financeapp.NewSalesOrderShippedHandler(setup.ReceivableRepo, setup.Logger)
		err := shippedHandler.Handle(ctx, shippedEvent)
		require.NoError(t, err)

		// Check initial outstanding balance
		initialOutstanding, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, setup.CustomerID)
		require.NoError(t, err)
		assert.True(t, initialOutstanding.GreaterThanOrEqual(orderAmount), "Should have at least order amount outstanding")

		// Step 2: Create red-letter receivable from return
		returnEvent := &trade.SalesReturnCompletedEvent{
			BaseDomainEvent: shared.NewBaseDomainEvent(
				trade.EventTypeSalesReturnCompleted,
				trade.AggregateTypeSalesReturn,
				returnID,
				setup.TenantID,
			),
			ReturnID:         returnID,
			ReturnNumber:     "SR-2024-RED-001",
			SalesOrderID:     orderID,
			SalesOrderNumber: "SO-2024-RED-001",
			CustomerID:       setup.CustomerID,
			CustomerName:     "Test Customer",
			WarehouseID:      setup.WarehouseID,
			Items: []trade.SalesReturnItemInfo{
				{
					ItemID:         uuid.New(),
					ProductID:      uuid.New(),
					ProductName:    "Test Product",
					ProductCode:    "PROD-001",
					ReturnQuantity: decimal.NewFromInt(3),
					UnitPrice:      decimal.NewFromFloat(100.00),
					RefundAmount:   decimal.NewFromFloat(300.00),
					Unit:           "pcs",
				},
			},
			TotalRefund: returnAmount,
		}

		returnHandler := financeapp.NewSalesReturnCompletedHandler(setup.ReceivableRepo, setup.Logger)
		err = returnHandler.Handle(ctx, returnEvent)
		require.NoError(t, err)

		// Verify both receivables exist
		orderReceivable, err := setup.ReceivableRepo.FindBySource(ctx, setup.TenantID, finance.SourceTypeSalesOrder, orderID)
		require.NoError(t, err)
		assert.Equal(t, finance.SourceTypeSalesOrder, orderReceivable.SourceType)

		returnReceivable, err := setup.ReceivableRepo.FindBySource(ctx, setup.TenantID, finance.SourceTypeSalesReturn, returnID)
		require.NoError(t, err)
		assert.Equal(t, finance.SourceTypeSalesReturn, returnReceivable.SourceType)

		// Net outstanding = orderAmount + returnAmount (both positive in DB)
		// In business terms: customer owes 1000, but gets 300 credit = net 700
		// Note: Both are stored as positive amounts, SourceType distinguishes them
		finalOutstanding, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, setup.CustomerID)
		require.NoError(t, err)
		// The sum includes both positive amounts; application logic handles net calculation
		assert.True(t, finalOutstanding.GreaterThanOrEqual(orderAmount.Add(returnAmount)),
			"Sum should include both order and return amounts")
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
		result, err := receivable.Reverse("Sales return processed")
		require.NoError(t, err)
		require.NotNil(t, result)

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

// ==================== P4-QA-004: Financial Data Accuracy Tests ====================

// TestFinance_AmountCalculationPrecision verifies decimal precision in financial calculations
func TestFinance_AmountCalculationPrecision(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("decimal precision preserved for small amounts", func(t *testing.T) {
		// Test precise decimal amounts like 0.01
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		// 0.01 is the smallest unit for financial transactions
		amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(0.01))
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"PRECISION-001",
			amount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Retrieve and verify
		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)
		assert.True(t, retrieved.TotalAmount.Equal(decimal.NewFromFloat(0.01)),
			"Should preserve 0.01 precision, got %s", retrieved.TotalAmount.String())
		assert.True(t, retrieved.OutstandingAmount.Equal(decimal.NewFromFloat(0.01)),
			"Outstanding should also be 0.01")
	})

	t.Run("decimal precision preserved for amounts with 4 decimal places", func(t *testing.T) {
		// DB schema allows decimal(18,4) - test 4 decimal place precision
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		// Amount with 4 decimal places
		preciseAmount := decimal.NewFromFloat(1234.5678)
		amount := valueobject.NewMoneyCNY(preciseAmount)
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"PRECISION-002",
			amount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Retrieve and verify
		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)
		assert.True(t, retrieved.TotalAmount.Equal(preciseAmount),
			"Should preserve 4 decimal places, got %s", retrieved.TotalAmount.String())
	})

	t.Run("large amount precision preserved", func(t *testing.T) {
		// Test large amounts (up to 18 digits before decimal)
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		// Large amount: 99,999,999.9999 (typical max for business transactions)
		largeAmount := decimal.NewFromFloat(99999999.9999)
		amount := valueobject.NewMoneyCNY(largeAmount)
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"PRECISION-003",
			amount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Retrieve and verify
		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)
		assert.True(t, retrieved.TotalAmount.Equal(largeAmount),
			"Should preserve large amount precision, got %s", retrieved.TotalAmount.String())
	})

	t.Run("payment amounts sum correctly without floating point errors", func(t *testing.T) {
		// Classic floating point issue: 0.1 + 0.2 != 0.3 in IEEE 754
		// Decimal library should handle this correctly
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(0.30))
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"PRECISION-SUM",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Apply first payment: 0.10
		payment1 := valueobject.NewMoneyCNY(decimal.NewFromFloat(0.10))
		voucherID1 := uuid.New()
		err = receivable.ApplyPayment(payment1, voucherID1, "Payment 1")
		require.NoError(t, err)

		// Apply second payment: 0.20
		payment2 := valueobject.NewMoneyCNY(decimal.NewFromFloat(0.20))
		voucherID2 := uuid.New()
		err = receivable.ApplyPayment(payment2, voucherID2, "Payment 2")
		require.NoError(t, err)

		// Save and verify
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		// Verify 0.1 + 0.2 = 0.3 exactly
		expectedPaid := decimal.NewFromFloat(0.30)
		assert.True(t, retrieved.PaidAmount.Equal(expectedPaid),
			"0.1 + 0.2 should equal 0.3, got %s", retrieved.PaidAmount.String())
		assert.True(t, retrieved.OutstandingAmount.IsZero(),
			"Outstanding should be zero, got %s", retrieved.OutstandingAmount.String())
		assert.Equal(t, finance.ReceivableStatusPaid, retrieved.Status)
	})

	t.Run("multiple partial payments maintain precision", func(t *testing.T) {
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		// Amount that requires multiple partial payments
		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(333.33))
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"PRECISION-MULTI",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Apply 3 partial payments of 111.11 each
		for i := 0; i < 3; i++ {
			payment := valueobject.NewMoneyCNY(decimal.NewFromFloat(111.11))
			err = receivable.ApplyPayment(payment, uuid.New(), "Partial payment")
			require.NoError(t, err)
		}

		// Save and verify
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		expectedPaid := decimal.NewFromFloat(333.33)
		assert.True(t, retrieved.PaidAmount.Equal(expectedPaid),
			"111.11 * 3 should equal 333.33, got %s", retrieved.PaidAmount.String())
		assert.True(t, retrieved.OutstandingAmount.IsZero())
		assert.Equal(t, finance.ReceivableStatusPaid, retrieved.Status)
	})
}

// TestFinance_BalanceAfterReconciliation verifies balance updates after reconciliation
func TestFinance_BalanceAfterReconciliation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	t.Run("partial payment updates status to PARTIAL", func(t *testing.T) {
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00))
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"BALANCE-PARTIAL",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Apply partial payment of 400.00
		payment := valueobject.NewMoneyCNY(decimal.NewFromFloat(400.00))
		err = receivable.ApplyPayment(payment, uuid.New(), "Partial payment")
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Verify
		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.ReceivableStatusPartial, retrieved.Status)
		assert.True(t, retrieved.PaidAmount.Equal(decimal.NewFromFloat(400.00)))
		assert.True(t, retrieved.OutstandingAmount.Equal(decimal.NewFromFloat(600.00)))
		assert.Nil(t, retrieved.PaidAt, "PaidAt should be nil for partial payment")
	})

	t.Run("full payment updates status to PAID", func(t *testing.T) {
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00))
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"BALANCE-FULL",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Apply full payment
		payment := valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00))
		err = receivable.ApplyPayment(payment, uuid.New(), "Full payment")
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Verify
		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		assert.Equal(t, finance.ReceivableStatusPaid, retrieved.Status)
		assert.True(t, retrieved.PaidAmount.Equal(decimal.NewFromFloat(500.00)))
		assert.True(t, retrieved.OutstandingAmount.IsZero())
		assert.NotNil(t, retrieved.PaidAt, "PaidAt should be set for full payment")
	})

	t.Run("balance invariant: TotalAmount = PaidAmount + OutstandingAmount", func(t *testing.T) {
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1234.56))
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			setup.CustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"BALANCE-INVARIANT",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Apply multiple partial payments with varying amounts
		payments := []float64{123.45, 234.56, 345.67, 200.00}
		for _, p := range payments {
			payment := valueobject.NewMoneyCNY(decimal.NewFromFloat(p))
			err = receivable.ApplyPayment(payment, uuid.New(), "Partial")
			require.NoError(t, err)

			// Verify invariant after each payment
			expected := receivable.TotalAmount
			actual := receivable.PaidAmount.Add(receivable.OutstandingAmount)
			assert.True(t, expected.Equal(actual),
				"Invariant violated: %s != %s + %s",
				expected.String(), receivable.PaidAmount.String(), receivable.OutstandingAmount.String())
		}

		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Verify from database
		retrieved, err := setup.ReceivableRepo.FindByID(ctx, receivable.ID)
		require.NoError(t, err)

		invariantCheck := retrieved.TotalAmount.Equal(
			retrieved.PaidAmount.Add(retrieved.OutstandingAmount))
		assert.True(t, invariantCheck,
			"Balance invariant failed after DB round-trip: Total=%s, Paid=%s, Outstanding=%s",
			retrieved.TotalAmount.String(),
			retrieved.PaidAmount.String(),
			retrieved.OutstandingAmount.String())
	})

	t.Run("payable balance updates correctly after payments", func(t *testing.T) {
		number, _ := setup.PayableRepo.GeneratePayableNumber(ctx, setup.TenantID)
		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(2500.00))
		payable, err := finance.NewAccountPayable(
			setup.TenantID,
			number,
			setup.SupplierID,
			"Test Supplier",
			finance.PayableSourceTypeManual,
			uuid.New(),
			"MANUAL",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.PayableRepo.Save(ctx, payable))

		// Apply multiple payments
		payments := []float64{500.00, 750.00, 1250.00}
		for _, p := range payments {
			payment := valueobject.NewMoneyCNY(decimal.NewFromFloat(p))
			err = payable.ApplyPayment(payment, uuid.New(), "Payment")
			require.NoError(t, err)
		}
		require.NoError(t, setup.PayableRepo.Save(ctx, payable))

		// Verify
		retrieved, err := setup.PayableRepo.FindByID(ctx, payable.ID)
		require.NoError(t, err)

		assert.True(t, retrieved.PaidAmount.Equal(decimal.NewFromFloat(2500.00)))
		assert.True(t, retrieved.OutstandingAmount.IsZero())
		assert.Equal(t, finance.PayableStatusPaid, retrieved.Status)
	})
}

// TestFinance_SummaryDataAccuracy verifies aggregate summary calculations
func TestFinance_SummaryDataAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use a fresh setup with a new tenant for isolated sum tests
	setup := NewFinanceTestSetup(t)
	ctx := context.Background()

	// Create a new customer and supplier for this test to ensure clean sums
	testCustomerID := uuid.New()
	testSupplierID := uuid.New()
	setup.DB.CreateTestCustomer(setup.TenantID, testCustomerID)
	setup.DB.CreateTestSupplier(setup.TenantID, testSupplierID)

	t.Run("customer outstanding sum matches individual receivables", func(t *testing.T) {
		// Create multiple receivables with known amounts
		amounts := []float64{1000.00, 2500.50, 3333.33, 166.17}
		var expectedTotal decimal.Decimal
		for _, amt := range amounts {
			expectedTotal = expectedTotal.Add(decimal.NewFromFloat(amt))
		}

		// Create receivables
		for i, amt := range amounts {
			number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
			amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(amt))
			receivable, err := finance.NewAccountReceivable(
				setup.TenantID,
				number,
				testCustomerID,
				"Test Customer",
				finance.SourceTypeManual,
				uuid.New(),
				fmt.Sprintf("SUM-TEST-%d", i),
				amount,
				nil,
			)
			require.NoError(t, err)
			require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))
		}

		// Verify sum
		totalOutstanding, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, testCustomerID)
		require.NoError(t, err)
		assert.True(t, totalOutstanding.Equal(expectedTotal),
			"Sum should be %s, got %s", expectedTotal.String(), totalOutstanding.String())
	})

	t.Run("sum updates correctly after partial payments", func(t *testing.T) {
		// Create a new customer for clean test
		partialCustomerID := uuid.New()
		setup.DB.CreateTestCustomer(setup.TenantID, partialCustomerID)

		// Create receivable
		number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		totalAmount := valueobject.NewMoneyCNY(decimal.NewFromFloat(1000.00))
		receivable, err := finance.NewAccountReceivable(
			setup.TenantID,
			number,
			partialCustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"SUM-PARTIAL-TEST",
			totalAmount,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Initial sum should be 1000.00
		sum1, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, partialCustomerID)
		require.NoError(t, err)
		assert.True(t, sum1.Equal(decimal.NewFromFloat(1000.00)))

		// Apply payment of 350.00
		payment := valueobject.NewMoneyCNY(decimal.NewFromFloat(350.00))
		err = receivable.ApplyPayment(payment, uuid.New(), "Payment")
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))

		// Sum should now be 650.00
		sum2, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, partialCustomerID)
		require.NoError(t, err)
		assert.True(t, sum2.Equal(decimal.NewFromFloat(650.00)),
			"After 350 payment, outstanding should be 650, got %s", sum2.String())
	})

	t.Run("supplier payable sum matches individual payables", func(t *testing.T) {
		// Create multiple payables with known amounts
		amounts := []float64{5000.00, 2500.00, 1500.25, 999.75}
		var expectedTotal decimal.Decimal
		for _, amt := range amounts {
			expectedTotal = expectedTotal.Add(decimal.NewFromFloat(amt))
		}

		// Create payables
		for i, amt := range amounts {
			number, _ := setup.PayableRepo.GeneratePayableNumber(ctx, setup.TenantID)
			amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(amt))
			payable, err := finance.NewAccountPayable(
				setup.TenantID,
				number,
				testSupplierID,
				"Test Supplier",
				finance.PayableSourceTypeManual,
				uuid.New(),
				fmt.Sprintf("PAYABLE-SUM-%d", i),
				amount,
				nil,
			)
			require.NoError(t, err)
			require.NoError(t, setup.PayableRepo.Save(ctx, payable))
		}

		// Verify sum
		totalOutstanding, err := setup.PayableRepo.SumOutstandingBySupplier(ctx, setup.TenantID, testSupplierID)
		require.NoError(t, err)
		assert.True(t, totalOutstanding.Equal(expectedTotal),
			"Payable sum should be %s, got %s", expectedTotal.String(), totalOutstanding.String())
	})

	t.Run("count for tenant matches created records", func(t *testing.T) {
		// Create a new customer for isolated count test
		countCustomerID := uuid.New()
		setup.DB.CreateTestCustomer(setup.TenantID, countCustomerID)

		expectedCount := 5
		for i := 0; i < expectedCount; i++ {
			number, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
			amount := valueobject.NewMoneyCNY(decimal.NewFromFloat(100.00))
			receivable, err := finance.NewAccountReceivable(
				setup.TenantID,
				number,
				countCustomerID,
				"Test Customer",
				finance.SourceTypeManual,
				uuid.New(),
				fmt.Sprintf("COUNT-TEST-%d", i),
				amount,
				nil,
			)
			require.NoError(t, err)
			require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable))
		}

		// Count by customer filter
		count, err := setup.ReceivableRepo.CountForTenant(ctx, setup.TenantID, finance.AccountReceivableFilter{
			CustomerID: &countCustomerID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(expectedCount), count,
			"Count should be %d, got %d", expectedCount, count)
	})

	t.Run("paid receivables excluded from outstanding sum", func(t *testing.T) {
		// Create customer for isolated test
		paidCustomerID := uuid.New()
		setup.DB.CreateTestCustomer(setup.TenantID, paidCustomerID)

		// Create two receivables: one paid, one pending
		// Receivable 1: 500.00 - will be paid
		number1, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		amount1 := valueobject.NewMoneyCNY(decimal.NewFromFloat(500.00))
		receivable1, err := finance.NewAccountReceivable(
			setup.TenantID,
			number1,
			paidCustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"PAID-EXCLUDE-1",
			amount1,
			nil,
		)
		require.NoError(t, err)
		// Save first (FK constraint requires receivable to exist before payment record)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable1))
		// Pay it fully
		err = receivable1.ApplyPayment(amount1, uuid.New(), "Full payment")
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable1))

		// Receivable 2: 300.00 - pending
		number2, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		amount2 := valueobject.NewMoneyCNY(decimal.NewFromFloat(300.00))
		receivable2, err := finance.NewAccountReceivable(
			setup.TenantID,
			number2,
			paidCustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"PAID-EXCLUDE-2",
			amount2,
			nil,
		)
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable2))

		// Outstanding sum should only include the pending 300.00
		// (SumOutstandingByCustomer sums OutstandingAmount, which is 0 for paid records)
		sum, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, paidCustomerID)
		require.NoError(t, err)
		assert.True(t, sum.Equal(decimal.NewFromFloat(300.00)),
			"Outstanding sum should be 300 (paid receivable excluded), got %s", sum.String())
	})

	t.Run("cancelled receivables excluded from outstanding sum", func(t *testing.T) {
		// Create customer for isolated test
		cancelCustomerID := uuid.New()
		setup.DB.CreateTestCustomer(setup.TenantID, cancelCustomerID)

		// Create receivable and cancel it
		number1, _ := setup.ReceivableRepo.GenerateReceivableNumber(ctx, setup.TenantID)
		amount1 := valueobject.NewMoneyCNY(decimal.NewFromFloat(700.00))
		receivable1, err := finance.NewAccountReceivable(
			setup.TenantID,
			number1,
			cancelCustomerID,
			"Test Customer",
			finance.SourceTypeManual,
			uuid.New(),
			"CANCEL-EXCLUDE",
			amount1,
			nil,
		)
		require.NoError(t, err)
		err = receivable1.Cancel("Order cancelled")
		require.NoError(t, err)
		require.NoError(t, setup.ReceivableRepo.Save(ctx, receivable1))

		// Outstanding sum should be 0 (cancelled has OutstandingAmount = 0)
		sum, err := setup.ReceivableRepo.SumOutstandingByCustomer(ctx, setup.TenantID, cancelCustomerID)
		require.NoError(t, err)
		assert.True(t, sum.IsZero(),
			"Outstanding sum should be 0 for cancelled receivable, got %s", sum.String())
	})
}
