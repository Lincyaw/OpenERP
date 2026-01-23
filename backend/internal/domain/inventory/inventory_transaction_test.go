package inventory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		txType   TransactionType
		expected bool
	}{
		{"INBOUND is valid", TransactionTypeInbound, true},
		{"OUTBOUND is valid", TransactionTypeOutbound, true},
		{"ADJUSTMENT_INCREASE is valid", TransactionTypeAdjustmentIncrease, true},
		{"ADJUSTMENT_DECREASE is valid", TransactionTypeAdjustmentDecrease, true},
		{"TRANSFER_IN is valid", TransactionTypeTransferIn, true},
		{"TRANSFER_OUT is valid", TransactionTypeTransferOut, true},
		{"RETURN is valid", TransactionTypeReturn, true},
		{"LOCK is valid", TransactionTypeLock, true},
		{"UNLOCK is valid", TransactionTypeUnlock, true},
		{"INVALID is not valid", TransactionType("INVALID"), false},
		{"empty is not valid", TransactionType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.txType.IsValid())
		})
	}
}

func TestTransactionType_IsIncrease(t *testing.T) {
	tests := []struct {
		name     string
		txType   TransactionType
		expected bool
	}{
		{"INBOUND is increase", TransactionTypeInbound, true},
		{"ADJUSTMENT_INCREASE is increase", TransactionTypeAdjustmentIncrease, true},
		{"TRANSFER_IN is increase", TransactionTypeTransferIn, true},
		{"RETURN is increase", TransactionTypeReturn, true},
		{"UNLOCK is increase", TransactionTypeUnlock, true},
		{"OUTBOUND is not increase", TransactionTypeOutbound, false},
		{"ADJUSTMENT_DECREASE is not increase", TransactionTypeAdjustmentDecrease, false},
		{"TRANSFER_OUT is not increase", TransactionTypeTransferOut, false},
		{"LOCK is not increase", TransactionTypeLock, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.txType.IsIncrease())
		})
	}
}

func TestTransactionType_IsDecrease(t *testing.T) {
	tests := []struct {
		name     string
		txType   TransactionType
		expected bool
	}{
		{"OUTBOUND is decrease", TransactionTypeOutbound, true},
		{"ADJUSTMENT_DECREASE is decrease", TransactionTypeAdjustmentDecrease, true},
		{"TRANSFER_OUT is decrease", TransactionTypeTransferOut, true},
		{"LOCK is decrease", TransactionTypeLock, true},
		{"INBOUND is not decrease", TransactionTypeInbound, false},
		{"ADJUSTMENT_INCREASE is not decrease", TransactionTypeAdjustmentIncrease, false},
		{"TRANSFER_IN is not decrease", TransactionTypeTransferIn, false},
		{"RETURN is not decrease", TransactionTypeReturn, false},
		{"UNLOCK is not decrease", TransactionTypeUnlock, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.txType.IsDecrease())
		})
	}
}

func TestTransactionType_String(t *testing.T) {
	assert.Equal(t, "INBOUND", TransactionTypeInbound.String())
	assert.Equal(t, "OUTBOUND", TransactionTypeOutbound.String())
}

func TestSourceType_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		sourceType SourceType
		expected   bool
	}{
		{"PURCHASE_ORDER is valid", SourceTypePurchaseOrder, true},
		{"SALES_ORDER is valid", SourceTypeSalesOrder, true},
		{"SALES_RETURN is valid", SourceTypeSalesReturn, true},
		{"PURCHASE_RETURN is valid", SourceTypePurchaseReturn, true},
		{"STOCK_TAKING is valid", SourceTypeStockTaking, true},
		{"MANUAL_ADJUSTMENT is valid", SourceTypeManualAdjustment, true},
		{"TRANSFER is valid", SourceTypeTransfer, true},
		{"INITIAL_STOCK is valid", SourceTypeInitialStock, true},
		{"INVALID is not valid", SourceType("INVALID"), false},
		{"empty is not valid", SourceType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.sourceType.IsValid())
		})
	}
}

func TestSourceType_String(t *testing.T) {
	assert.Equal(t, "PURCHASE_ORDER", SourceTypePurchaseOrder.String())
	assert.Equal(t, "SALES_ORDER", SourceTypeSalesOrder.String())
}

func TestNewInventoryTransaction_Success(t *testing.T) {
	tenantID := uuid.New()
	inventoryItemID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()
	quantity := decimal.NewFromInt(100)
	unitCost := decimal.NewFromFloat(10.50)
	balanceBefore := decimal.NewFromInt(200)
	balanceAfter := decimal.NewFromInt(300)

	tx, err := NewInventoryTransaction(
		tenantID,
		inventoryItemID,
		warehouseID,
		productID,
		TransactionTypeInbound,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		SourceTypePurchaseOrder,
		"PO-2024-001",
	)

	require.NoError(t, err)
	require.NotNil(t, tx)

	assert.NotEqual(t, uuid.Nil, tx.ID)
	assert.Equal(t, tenantID, tx.TenantID)
	assert.Equal(t, inventoryItemID, tx.InventoryItemID)
	assert.Equal(t, warehouseID, tx.WarehouseID)
	assert.Equal(t, productID, tx.ProductID)
	assert.Equal(t, TransactionTypeInbound, tx.TransactionType)
	assert.True(t, quantity.Equal(tx.Quantity))
	assert.True(t, unitCost.Equal(tx.UnitCost))
	assert.True(t, decimal.NewFromFloat(1050).Equal(tx.TotalCost))
	assert.True(t, balanceBefore.Equal(tx.BalanceBefore))
	assert.True(t, balanceAfter.Equal(tx.BalanceAfter))
	assert.Equal(t, SourceTypePurchaseOrder, tx.SourceType)
	assert.Equal(t, "PO-2024-001", tx.SourceID)
	assert.False(t, tx.TransactionDate.IsZero())
}

func TestNewInventoryTransaction_Validation(t *testing.T) {
	validTenantID := uuid.New()
	validInventoryItemID := uuid.New()
	validWarehouseID := uuid.New()
	validProductID := uuid.New()
	validQuantity := decimal.NewFromInt(100)
	validUnitCost := decimal.NewFromFloat(10.50)
	validBalanceBefore := decimal.NewFromInt(200)
	validBalanceAfter := decimal.NewFromInt(300)

	tests := []struct {
		name            string
		tenantID        uuid.UUID
		inventoryItemID uuid.UUID
		warehouseID     uuid.UUID
		productID       uuid.UUID
		txType          TransactionType
		quantity        decimal.Decimal
		unitCost        decimal.Decimal
		sourceType      SourceType
		sourceID        string
		expectedError   string
	}{
		{
			name:            "empty tenant ID",
			tenantID:        uuid.Nil,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        validQuantity,
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Tenant ID cannot be empty",
		},
		{
			name:            "empty inventory item ID",
			tenantID:        validTenantID,
			inventoryItemID: uuid.Nil,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        validQuantity,
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Inventory item ID cannot be empty",
		},
		{
			name:            "empty warehouse ID",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     uuid.Nil,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        validQuantity,
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Warehouse ID cannot be empty",
		},
		{
			name:            "empty product ID",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       uuid.Nil,
			txType:          TransactionTypeInbound,
			quantity:        validQuantity,
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Product ID cannot be empty",
		},
		{
			name:            "invalid transaction type",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionType("INVALID"),
			quantity:        validQuantity,
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Invalid transaction type",
		},
		{
			name:            "zero quantity",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        decimal.Zero,
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Quantity must be positive",
		},
		{
			name:            "negative quantity",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        decimal.NewFromInt(-100),
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Quantity must be positive",
		},
		{
			name:            "negative unit cost",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        validQuantity,
			unitCost:        decimal.NewFromInt(-10),
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "PO-001",
			expectedError:   "Unit cost cannot be negative",
		},
		{
			name:            "invalid source type",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        validQuantity,
			unitCost:        validUnitCost,
			sourceType:      SourceType("INVALID"),
			sourceID:        "PO-001",
			expectedError:   "Invalid source type",
		},
		{
			name:            "empty source ID",
			tenantID:        validTenantID,
			inventoryItemID: validInventoryItemID,
			warehouseID:     validWarehouseID,
			productID:       validProductID,
			txType:          TransactionTypeInbound,
			quantity:        validQuantity,
			unitCost:        validUnitCost,
			sourceType:      SourceTypePurchaseOrder,
			sourceID:        "",
			expectedError:   "Source ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := NewInventoryTransaction(
				tt.tenantID,
				tt.inventoryItemID,
				tt.warehouseID,
				tt.productID,
				tt.txType,
				tt.quantity,
				tt.unitCost,
				validBalanceBefore,
				validBalanceAfter,
				tt.sourceType,
				tt.sourceID,
			)

			assert.Nil(t, tx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestInventoryTransaction_WithMethods(t *testing.T) {
	tx, err := NewInventoryTransaction(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		TransactionTypeInbound,
		decimal.NewFromInt(100),
		decimal.NewFromFloat(10.50),
		decimal.NewFromInt(200),
		decimal.NewFromInt(300),
		SourceTypePurchaseOrder,
		"PO-2024-001",
	)
	require.NoError(t, err)

	batchID := uuid.New()
	lockID := uuid.New()
	operatorID := uuid.New()
	txDate := time.Now().Add(-24 * time.Hour)

	tx.WithBatchID(batchID).
		WithLockID(lockID).
		WithSourceLineID("LINE-001").
		WithReference("REF-001").
		WithReason("Test reason").
		WithOperatorID(operatorID).
		WithTransactionDate(txDate)

	assert.NotNil(t, tx.BatchID)
	assert.Equal(t, batchID, *tx.BatchID)
	assert.NotNil(t, tx.LockID)
	assert.Equal(t, lockID, *tx.LockID)
	assert.Equal(t, "LINE-001", tx.SourceLineID)
	assert.Equal(t, "REF-001", tx.Reference)
	assert.Equal(t, "Test reason", tx.Reason)
	assert.NotNil(t, tx.OperatorID)
	assert.Equal(t, operatorID, *tx.OperatorID)
	assert.Equal(t, txDate, tx.TransactionDate)
}

func TestInventoryTransaction_GetSignedQuantity(t *testing.T) {
	quantity := decimal.NewFromInt(100)

	tests := []struct {
		name         string
		txType       TransactionType
		expectedSign int
	}{
		{"INBOUND is positive", TransactionTypeInbound, 1},
		{"OUTBOUND is negative", TransactionTypeOutbound, -1},
		{"ADJUSTMENT_INCREASE is positive", TransactionTypeAdjustmentIncrease, 1},
		{"ADJUSTMENT_DECREASE is negative", TransactionTypeAdjustmentDecrease, -1},
		{"TRANSFER_IN is positive", TransactionTypeTransferIn, 1},
		{"TRANSFER_OUT is negative", TransactionTypeTransferOut, -1},
		{"RETURN is positive", TransactionTypeReturn, 1},
		{"LOCK is negative", TransactionTypeLock, -1},
		{"UNLOCK is positive", TransactionTypeUnlock, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := NewInventoryTransaction(
				uuid.New(),
				uuid.New(),
				uuid.New(),
				uuid.New(),
				tt.txType,
				quantity,
				decimal.NewFromFloat(10.0),
				decimal.NewFromInt(200),
				decimal.NewFromInt(200),
				SourceTypePurchaseOrder,
				"PO-001",
			)
			require.NoError(t, err)

			signedQty := tx.GetSignedQuantity()
			if tt.expectedSign > 0 {
				assert.True(t, signedQty.IsPositive(), "Expected positive quantity for %s", tt.txType)
			} else {
				assert.True(t, signedQty.IsNegative(), "Expected negative quantity for %s", tt.txType)
			}
			assert.True(t, signedQty.Abs().Equal(quantity))
		})
	}
}

func TestInventoryTransaction_GetSignedTotalCost(t *testing.T) {
	quantity := decimal.NewFromInt(100)
	unitCost := decimal.NewFromFloat(10.0)
	expectedTotalCost := decimal.NewFromInt(1000)

	// Test increase transaction
	inboundTx, err := NewInventoryTransaction(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		TransactionTypeInbound,
		quantity, unitCost,
		decimal.NewFromInt(0), decimal.NewFromInt(100),
		SourceTypePurchaseOrder, "PO-001",
	)
	require.NoError(t, err)

	signedCost := inboundTx.GetSignedTotalCost()
	assert.True(t, signedCost.IsPositive())
	assert.True(t, signedCost.Equal(expectedTotalCost))

	// Test decrease transaction
	outboundTx, err := NewInventoryTransaction(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		TransactionTypeOutbound,
		quantity, unitCost,
		decimal.NewFromInt(100), decimal.NewFromInt(0),
		SourceTypeSalesOrder, "SO-001",
	)
	require.NoError(t, err)

	signedCost = outboundTx.GetSignedTotalCost()
	assert.True(t, signedCost.IsNegative())
	assert.True(t, signedCost.Abs().Equal(expectedTotalCost))
}

func TestInventoryTransaction_IsInboundAndOutbound(t *testing.T) {
	inboundTx, err := NewInventoryTransaction(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		TransactionTypeInbound,
		decimal.NewFromInt(100), decimal.NewFromFloat(10.0),
		decimal.NewFromInt(0), decimal.NewFromInt(100),
		SourceTypePurchaseOrder, "PO-001",
	)
	require.NoError(t, err)

	assert.True(t, inboundTx.IsInbound())
	assert.False(t, inboundTx.IsOutbound())

	outboundTx, err := NewInventoryTransaction(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		TransactionTypeOutbound,
		decimal.NewFromInt(50), decimal.NewFromFloat(10.0),
		decimal.NewFromInt(100), decimal.NewFromInt(50),
		SourceTypeSalesOrder, "SO-001",
	)
	require.NoError(t, err)

	assert.False(t, outboundTx.IsInbound())
	assert.True(t, outboundTx.IsOutbound())
}

func TestInventoryTransaction_QuantityChange(t *testing.T) {
	tests := []struct {
		name           string
		balanceBefore  decimal.Decimal
		balanceAfter   decimal.Decimal
		expectedChange decimal.Decimal
	}{
		{
			name:           "positive change",
			balanceBefore:  decimal.NewFromInt(100),
			balanceAfter:   decimal.NewFromInt(150),
			expectedChange: decimal.NewFromInt(50),
		},
		{
			name:           "negative change",
			balanceBefore:  decimal.NewFromInt(100),
			balanceAfter:   decimal.NewFromInt(70),
			expectedChange: decimal.NewFromInt(-30),
		},
		{
			name:           "no change",
			balanceBefore:  decimal.NewFromInt(100),
			balanceAfter:   decimal.NewFromInt(100),
			expectedChange: decimal.NewFromInt(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := NewInventoryTransaction(
				uuid.New(), uuid.New(), uuid.New(), uuid.New(),
				TransactionTypeInbound,
				decimal.NewFromInt(50), decimal.NewFromFloat(10.0),
				tt.balanceBefore, tt.balanceAfter,
				SourceTypePurchaseOrder, "PO-001",
			)
			require.NoError(t, err)

			assert.True(t, tt.expectedChange.Equal(tx.QuantityChange()))
		})
	}
}

func TestInventoryTransaction_TableName(t *testing.T) {
	tx := InventoryTransaction{}
	assert.Equal(t, "inventory_transactions", tx.TableName())
}

func TestTransactionBuilder_Success(t *testing.T) {
	tenantID := uuid.New()
	inventoryItemID := uuid.New()
	warehouseID := uuid.New()
	productID := uuid.New()
	batchID := uuid.New()
	lockID := uuid.New()
	operatorID := uuid.New()

	tx, err := NewTransactionBuilder(
		tenantID,
		inventoryItemID,
		warehouseID,
		productID,
		TransactionTypeInbound,
		decimal.NewFromInt(100),
		decimal.NewFromFloat(10.0),
		decimal.NewFromInt(0),
		decimal.NewFromInt(100),
		SourceTypePurchaseOrder,
		"PO-001",
	).
		WithBatchID(batchID).
		WithLockID(lockID).
		WithSourceLineID("LINE-001").
		WithReference("REF-001").
		WithReason("Initial stock").
		WithOperatorID(operatorID).
		Build()

	require.NoError(t, err)
	require.NotNil(t, tx)

	assert.Equal(t, batchID, *tx.BatchID)
	assert.Equal(t, lockID, *tx.LockID)
	assert.Equal(t, "LINE-001", tx.SourceLineID)
	assert.Equal(t, "REF-001", tx.Reference)
	assert.Equal(t, "Initial stock", tx.Reason)
	assert.Equal(t, operatorID, *tx.OperatorID)
}

func TestTransactionBuilder_Error(t *testing.T) {
	// Builder with invalid parameters should propagate error
	tx, err := NewTransactionBuilder(
		uuid.Nil, // Invalid tenant ID
		uuid.New(),
		uuid.New(),
		uuid.New(),
		TransactionTypeInbound,
		decimal.NewFromInt(100),
		decimal.NewFromFloat(10.0),
		decimal.NewFromInt(0),
		decimal.NewFromInt(100),
		SourceTypePurchaseOrder,
		"PO-001",
	).
		WithBatchID(uuid.New()). // This should be ignored
		Build()

	assert.Nil(t, tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Tenant ID cannot be empty")
}

func TestCreateInboundTransaction(t *testing.T) {
	tx, err := CreateInboundTransaction(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(100), decimal.NewFromFloat(10.0),
		decimal.NewFromInt(0), decimal.NewFromInt(100),
		SourceTypePurchaseOrder, "PO-001",
	)

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, TransactionTypeInbound, tx.TransactionType)
}

func TestCreateOutboundTransaction(t *testing.T) {
	tx, err := CreateOutboundTransaction(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		decimal.NewFromInt(50), decimal.NewFromFloat(10.0),
		decimal.NewFromInt(100), decimal.NewFromInt(50),
		SourceTypeSalesOrder, "SO-001",
	)

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, TransactionTypeOutbound, tx.TransactionType)
}

func TestCreateAdjustmentTransaction(t *testing.T) {
	t.Run("positive adjustment", func(t *testing.T) {
		tx, err := CreateAdjustmentTransaction(
			uuid.New(), uuid.New(), uuid.New(), uuid.New(),
			decimal.NewFromInt(10), decimal.NewFromFloat(10.0),
			decimal.NewFromInt(100), decimal.NewFromInt(110),
			SourceTypeStockTaking, "ST-001",
			"Stock count adjustment",
		)

		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.Equal(t, TransactionTypeAdjustmentIncrease, tx.TransactionType)
		assert.Equal(t, "Stock count adjustment", tx.Reason)
	})

	t.Run("negative adjustment", func(t *testing.T) {
		tx, err := CreateAdjustmentTransaction(
			uuid.New(), uuid.New(), uuid.New(), uuid.New(),
			decimal.NewFromInt(10), decimal.NewFromFloat(10.0),
			decimal.NewFromInt(100), decimal.NewFromInt(90),
			SourceTypeStockTaking, "ST-001",
			"Stock count adjustment",
		)

		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.Equal(t, TransactionTypeAdjustmentDecrease, tx.TransactionType)
	})
}

func TestInventoryTransaction_ZeroUnitCost(t *testing.T) {
	// Zero unit cost should be valid (e.g., free items, samples)
	tx, err := NewInventoryTransaction(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		TransactionTypeInbound,
		decimal.NewFromInt(100),
		decimal.Zero, // Zero unit cost
		decimal.NewFromInt(0),
		decimal.NewFromInt(100),
		SourceTypeManualAdjustment,
		"ADJ-001",
	)

	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.True(t, tx.UnitCost.IsZero())
	assert.True(t, tx.TotalCost.IsZero())
}
