package partner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBalanceTransactionType(t *testing.T) {
	t.Run("IsValid returns true for valid types", func(t *testing.T) {
		validTypes := []BalanceTransactionType{
			BalanceTransactionTypeRecharge,
			BalanceTransactionTypeConsume,
			BalanceTransactionTypeRefund,
			BalanceTransactionTypeAdjustment,
			BalanceTransactionTypeExpire,
		}

		for _, txType := range validTypes {
			assert.True(t, txType.IsValid(), "Expected %s to be valid", txType)
		}
	})

	t.Run("IsValid returns false for invalid type", func(t *testing.T) {
		invalid := BalanceTransactionType("INVALID")
		assert.False(t, invalid.IsValid())
	})

	t.Run("IsIncrease returns correct values", func(t *testing.T) {
		assert.True(t, BalanceTransactionTypeRecharge.IsIncrease())
		assert.True(t, BalanceTransactionTypeRefund.IsIncrease())
		assert.False(t, BalanceTransactionTypeConsume.IsIncrease())
		assert.False(t, BalanceTransactionTypeExpire.IsIncrease())
		assert.False(t, BalanceTransactionTypeAdjustment.IsIncrease()) // Adjustment can be either
	})

	t.Run("IsDecrease returns correct values", func(t *testing.T) {
		assert.True(t, BalanceTransactionTypeConsume.IsDecrease())
		assert.True(t, BalanceTransactionTypeExpire.IsDecrease())
		assert.False(t, BalanceTransactionTypeRecharge.IsDecrease())
		assert.False(t, BalanceTransactionTypeRefund.IsDecrease())
		assert.False(t, BalanceTransactionTypeAdjustment.IsDecrease()) // Adjustment can be either
	})

	t.Run("String returns correct value", func(t *testing.T) {
		assert.Equal(t, "RECHARGE", BalanceTransactionTypeRecharge.String())
		assert.Equal(t, "CONSUME", BalanceTransactionTypeConsume.String())
	})
}

func TestBalanceTransactionSourceType(t *testing.T) {
	t.Run("IsValid returns true for valid source types", func(t *testing.T) {
		validTypes := []BalanceTransactionSourceType{
			BalanceSourceTypeManual,
			BalanceSourceTypeSalesOrder,
			BalanceSourceTypeSalesReturn,
			BalanceSourceTypeReceiptVoucher,
			BalanceSourceTypeSystem,
		}

		for _, srcType := range validTypes {
			assert.True(t, srcType.IsValid(), "Expected %s to be valid", srcType)
		}
	})

	t.Run("IsValid returns false for invalid source type", func(t *testing.T) {
		invalid := BalanceTransactionSourceType("INVALID")
		assert.False(t, invalid.IsValid())
	})

	t.Run("String returns correct value", func(t *testing.T) {
		assert.Equal(t, "MANUAL", BalanceSourceTypeManual.String())
		assert.Equal(t, "SALES_ORDER", BalanceSourceTypeSalesOrder.String())
	})
}

func TestPaymentMethod(t *testing.T) {
	t.Run("IsValid returns true for valid payment methods", func(t *testing.T) {
		validMethods := []PaymentMethod{
			PaymentMethodCash,
			PaymentMethodWechat,
			PaymentMethodAlipay,
			PaymentMethodBank,
		}

		for _, method := range validMethods {
			assert.True(t, method.IsValid(), "Expected %s to be valid", method)
		}
	})

	t.Run("IsValid returns false for invalid payment method", func(t *testing.T) {
		invalid := PaymentMethod("INVALID")
		assert.False(t, invalid.IsValid())
	})

	t.Run("String returns correct value", func(t *testing.T) {
		assert.Equal(t, "CASH", PaymentMethodCash.String())
		assert.Equal(t, "WECHAT", PaymentMethodWechat.String())
		assert.Equal(t, "ALIPAY", PaymentMethodAlipay.String())
		assert.Equal(t, "BANK", PaymentMethodBank.String())
	})
}

func TestNewBalanceTransaction(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("creates transaction successfully", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			customerID,
			BalanceTransactionTypeRecharge,
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(0.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeManual,
		)

		require.NoError(t, err)
		assert.NotNil(t, tx)
		assert.Equal(t, tenantID, tx.TenantID)
		assert.Equal(t, customerID, tx.CustomerID)
		assert.Equal(t, BalanceTransactionTypeRecharge, tx.TransactionType)
		assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, tx.BalanceBefore.Equal(decimal.NewFromFloat(0.00)))
		assert.True(t, tx.BalanceAfter.Equal(decimal.NewFromFloat(100.00)))
		assert.Equal(t, BalanceSourceTypeManual, tx.SourceType)
		assert.NotEmpty(t, tx.ID)
		assert.False(t, tx.TransactionDate.IsZero())
	})

	t.Run("fails with nil tenant ID", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			uuid.Nil,
			customerID,
			BalanceTransactionTypeRecharge,
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(0.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeManual,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "Tenant ID")
	})

	t.Run("fails with nil customer ID", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			uuid.Nil,
			BalanceTransactionTypeRecharge,
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(0.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeManual,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "Customer ID")
	})

	t.Run("fails with invalid transaction type", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			customerID,
			BalanceTransactionType("INVALID"),
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(0.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeManual,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "transaction type")
	})

	t.Run("fails with zero amount", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			customerID,
			BalanceTransactionTypeRecharge,
			decimal.Zero,
			decimal.NewFromFloat(0.00),
			decimal.NewFromFloat(0.00),
			BalanceSourceTypeManual,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("fails with negative amount", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			customerID,
			BalanceTransactionTypeRecharge,
			decimal.NewFromFloat(-100.00),
			decimal.NewFromFloat(0.00),
			decimal.NewFromFloat(0.00),
			BalanceSourceTypeManual,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "positive")
	})

	t.Run("fails with negative balance before", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			customerID,
			BalanceTransactionTypeRecharge,
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(-10.00),
			decimal.NewFromFloat(90.00),
			BalanceSourceTypeManual,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "Balance before")
	})

	t.Run("fails with negative balance after", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			customerID,
			BalanceTransactionTypeConsume,
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(50.00),
			decimal.NewFromFloat(-50.00),
			BalanceSourceTypeSalesOrder,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "Balance after")
	})

	t.Run("fails with invalid source type", func(t *testing.T) {
		tx, err := NewBalanceTransaction(
			tenantID,
			customerID,
			BalanceTransactionTypeRecharge,
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(0.00),
			decimal.NewFromFloat(100.00),
			BalanceTransactionSourceType("INVALID"),
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "source type")
	})
}

func TestCreateRechargeTransaction(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("creates recharge transaction successfully", func(t *testing.T) {
		tx, err := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(500.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeManual,
		)

		require.NoError(t, err)
		assert.Equal(t, BalanceTransactionTypeRecharge, tx.TransactionType)
		assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(500.00)))
		assert.True(t, tx.BalanceBefore.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, tx.BalanceAfter.Equal(decimal.NewFromFloat(600.00)))
	})
}

func TestCreateConsumeTransaction(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("creates consume transaction successfully", func(t *testing.T) {
		tx, err := CreateConsumeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(50.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeSalesOrder,
		)

		require.NoError(t, err)
		assert.Equal(t, BalanceTransactionTypeConsume, tx.TransactionType)
		assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(50.00)))
		assert.True(t, tx.BalanceBefore.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, tx.BalanceAfter.Equal(decimal.NewFromFloat(50.00)))
	})

	t.Run("fails with insufficient balance", func(t *testing.T) {
		tx, err := CreateConsumeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(150.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeSalesOrder,
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "Insufficient balance")
	})
}

func TestCreateRefundTransaction(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("creates refund transaction successfully", func(t *testing.T) {
		tx, err := CreateRefundTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(30.00),
			decimal.NewFromFloat(70.00),
			BalanceSourceTypeSalesReturn,
		)

		require.NoError(t, err)
		assert.Equal(t, BalanceTransactionTypeRefund, tx.TransactionType)
		assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(30.00)))
		assert.True(t, tx.BalanceBefore.Equal(decimal.NewFromFloat(70.00)))
		assert.True(t, tx.BalanceAfter.Equal(decimal.NewFromFloat(100.00)))
	})
}

func TestCreateAdjustmentTransaction(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("creates positive adjustment transaction successfully", func(t *testing.T) {
		tx, err := CreateAdjustmentTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(25.00),
			true, // isIncrease
			decimal.NewFromFloat(100.00),
		)

		require.NoError(t, err)
		assert.Equal(t, BalanceTransactionTypeAdjustment, tx.TransactionType)
		assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(25.00)))
		assert.True(t, tx.BalanceBefore.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, tx.BalanceAfter.Equal(decimal.NewFromFloat(125.00)))
	})

	t.Run("creates negative adjustment transaction successfully", func(t *testing.T) {
		tx, err := CreateAdjustmentTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(25.00),
			false, // isIncrease
			decimal.NewFromFloat(100.00),
		)

		require.NoError(t, err)
		assert.Equal(t, BalanceTransactionTypeAdjustment, tx.TransactionType)
		assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(25.00)))
		assert.True(t, tx.BalanceBefore.Equal(decimal.NewFromFloat(100.00)))
		assert.True(t, tx.BalanceAfter.Equal(decimal.NewFromFloat(75.00)))
	})

	t.Run("fails with insufficient balance for decrease", func(t *testing.T) {
		tx, err := CreateAdjustmentTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(150.00),
			false, // isIncrease
			decimal.NewFromFloat(100.00),
		)

		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "Insufficient balance")
	})
}

func TestBalanceTransactionHelpers(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("WithSourceID sets source ID", func(t *testing.T) {
		tx, _ := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(100.00),
			decimal.Zero,
			BalanceSourceTypeManual,
		)

		sourceID := "SRC-001"
		tx.WithSourceID(sourceID)

		assert.NotNil(t, tx.SourceID)
		assert.Equal(t, sourceID, *tx.SourceID)
	})

	t.Run("WithReference sets reference", func(t *testing.T) {
		tx, _ := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(100.00),
			decimal.Zero,
			BalanceSourceTypeManual,
		)

		tx.WithReference("REF-001")
		assert.Equal(t, "REF-001", tx.Reference)
	})

	t.Run("WithRemark sets remark", func(t *testing.T) {
		tx, _ := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(100.00),
			decimal.Zero,
			BalanceSourceTypeManual,
		)

		tx.WithRemark("Test remark")
		assert.Equal(t, "Test remark", tx.Remark)
	})

	t.Run("WithOperatorID sets operator ID", func(t *testing.T) {
		tx, _ := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(100.00),
			decimal.Zero,
			BalanceSourceTypeManual,
		)

		operatorID := uuid.New()
		tx.WithOperatorID(operatorID)

		assert.NotNil(t, tx.OperatorID)
		assert.Equal(t, operatorID, *tx.OperatorID)
	})
}

func TestBalanceTransactionMethods(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	t.Run("GetSignedAmount returns correct value for recharge", func(t *testing.T) {
		tx, _ := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(100.00),
			decimal.Zero,
			BalanceSourceTypeManual,
		)

		signed := tx.GetSignedAmount()
		assert.True(t, signed.Equal(decimal.NewFromFloat(100.00)))
	})

	t.Run("GetSignedAmount returns negative value for consume", func(t *testing.T) {
		tx, _ := CreateConsumeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(50.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeSalesOrder,
		)

		signed := tx.GetSignedAmount()
		assert.True(t, signed.Equal(decimal.NewFromFloat(-50.00)))
	})

	t.Run("IsIncrease and IsDecrease work correctly", func(t *testing.T) {
		rechargeTx, _ := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(100.00),
			decimal.Zero,
			BalanceSourceTypeManual,
		)
		assert.True(t, rechargeTx.IsIncrease())
		assert.False(t, rechargeTx.IsDecrease())

		consumeTx, _ := CreateConsumeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(50.00),
			decimal.NewFromFloat(100.00),
			BalanceSourceTypeSalesOrder,
		)
		assert.False(t, consumeTx.IsIncrease())
		assert.True(t, consumeTx.IsDecrease())
	})

	t.Run("BalanceChange returns correct value", func(t *testing.T) {
		tx, _ := CreateRechargeTransaction(
			tenantID,
			customerID,
			decimal.NewFromFloat(100.00),
			decimal.NewFromFloat(50.00),
			BalanceSourceTypeManual,
		)

		change := tx.BalanceChange()
		assert.True(t, change.Equal(decimal.NewFromFloat(100.00)))
	})
}
