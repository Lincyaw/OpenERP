package finance

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrialBalanceStatus(t *testing.T) {
	t.Run("IsValid returns true for valid statuses", func(t *testing.T) {
		assert.True(t, TrialBalanceStatusBalanced.IsValid())
		assert.True(t, TrialBalanceStatusUnbalanced.IsValid())
	})

	t.Run("IsValid returns false for invalid status", func(t *testing.T) {
		assert.False(t, TrialBalanceStatus("INVALID").IsValid())
	})

	t.Run("String returns correct string representation", func(t *testing.T) {
		assert.Equal(t, "BALANCED", TrialBalanceStatusBalanced.String())
		assert.Equal(t, "UNBALANCED", TrialBalanceStatusUnbalanced.String())
	})

	t.Run("IsBalanced returns correct boolean", func(t *testing.T) {
		assert.True(t, TrialBalanceStatusBalanced.IsBalanced())
		assert.False(t, TrialBalanceStatusUnbalanced.IsBalanced())
	})
}

func TestBalanceDiscrepancyType(t *testing.T) {
	t.Run("IsValid returns true for valid types", func(t *testing.T) {
		validTypes := []BalanceDiscrepancyType{
			ReceivablePaymentMismatch,
			PayablePaymentMismatch,
			VoucherAllocationMismatch,
			CreditMemoImbalance,
			DebitMemoImbalance,
			ReceivableAmountMismatch,
			PayableAmountMismatch,
		}
		for _, dt := range validTypes {
			assert.True(t, dt.IsValid(), "Expected %s to be valid", dt)
		}
	})

	t.Run("IsValid returns false for invalid type", func(t *testing.T) {
		assert.False(t, BalanceDiscrepancyType("INVALID").IsValid())
	})

	t.Run("Description returns non-empty description for valid types", func(t *testing.T) {
		validTypes := []BalanceDiscrepancyType{
			ReceivablePaymentMismatch,
			PayablePaymentMismatch,
			VoucherAllocationMismatch,
			CreditMemoImbalance,
			DebitMemoImbalance,
			ReceivableAmountMismatch,
			PayableAmountMismatch,
		}
		for _, dt := range validTypes {
			desc := dt.Description()
			assert.NotEmpty(t, desc, "Expected non-empty description for %s", dt)
		}
	})

	t.Run("Description returns default for unknown type", func(t *testing.T) {
		desc := BalanceDiscrepancyType("UNKNOWN").Description()
		assert.Equal(t, "Unknown discrepancy", desc)
	})
}

func TestNewBalanceDiscrepancy(t *testing.T) {
	t.Run("creates discrepancy with correct values", func(t *testing.T) {
		entityID := uuid.New()
		expected := decimal.NewFromFloat(1000.00)
		actual := decimal.NewFromFloat(900.00)

		d := NewBalanceDiscrepancy(
			ReceivableAmountMismatch,
			"AccountReceivable",
			entityID,
			"AR-2026-00001",
			expected,
			actual,
		)

		assert.NotEqual(t, uuid.Nil, d.ID)
		assert.Equal(t, ReceivableAmountMismatch, d.Type)
		assert.Equal(t, "AccountReceivable", d.EntityType)
		assert.Equal(t, entityID, d.EntityID)
		assert.Equal(t, "AR-2026-00001", d.EntityNumber)
		assert.True(t, expected.Equal(d.ExpectedAmount))
		assert.True(t, actual.Equal(d.ActualAmount))
		assert.True(t, decimal.NewFromFloat(100.00).Equal(d.Difference))
		assert.Equal(t, "CRITICAL", d.Severity)
		assert.NotZero(t, d.DetectedAt)
	})

	t.Run("sets severity to WARNING for small differences", func(t *testing.T) {
		expected := decimal.NewFromFloat(100.00)
		actual := decimal.NewFromFloat(100.005)

		d := NewBalanceDiscrepancy(
			ReceivableAmountMismatch,
			"AccountReceivable",
			uuid.New(),
			"AR-2026-00001",
			expected,
			actual,
		)

		assert.Equal(t, "WARNING", d.Severity)
	})

	t.Run("handles zero difference", func(t *testing.T) {
		expected := decimal.NewFromFloat(100.00)
		actual := decimal.NewFromFloat(100.00)

		d := NewBalanceDiscrepancy(
			ReceivableAmountMismatch,
			"AccountReceivable",
			uuid.New(),
			"AR-2026-00001",
			expected,
			actual,
		)

		assert.True(t, d.Difference.IsZero())
		assert.Equal(t, "WARNING", d.Severity) // Zero diff is warning level
	})
}

func TestBalanceDiscrepancy_AddRelatedEntity(t *testing.T) {
	d := NewBalanceDiscrepancy(
		ReceivableAmountMismatch,
		"AccountReceivable",
		uuid.New(),
		"AR-2026-00001",
		decimal.NewFromFloat(1000.00),
		decimal.NewFromFloat(900.00),
	)

	relatedID := uuid.New()
	d.AddRelatedEntity("ReceiptVoucher", relatedID, "RV-2026-00001")

	assert.Len(t, d.RelatedEntities, 1)
	assert.Equal(t, "ReceiptVoucher", d.RelatedEntities[0].EntityType)
	assert.Equal(t, relatedID, d.RelatedEntities[0].EntityID)
	assert.Equal(t, "RV-2026-00001", d.RelatedEntities[0].Reference)
}

func TestNewTrialBalanceResult(t *testing.T) {
	tenantID := uuid.New()
	checkedBy := uuid.New()

	result := NewTrialBalanceResult(tenantID, checkedBy)

	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, tenantID, result.TenantID)
	assert.Equal(t, checkedBy, result.CheckedBy)
	assert.NotZero(t, result.CheckedAt)
	assert.Equal(t, TrialBalanceStatusBalanced, result.Status)
	assert.True(t, result.TotalDebits.IsZero())
	assert.True(t, result.TotalCredits.IsZero())
	assert.True(t, result.NetBalance.IsZero())
	assert.Empty(t, result.Discrepancies)
}

func TestTrialBalanceResult_AddDiscrepancy(t *testing.T) {
	result := NewTrialBalanceResult(uuid.New(), uuid.New())

	// Add critical discrepancy
	d1 := NewBalanceDiscrepancy(
		ReceivableAmountMismatch,
		"AccountReceivable",
		uuid.New(),
		"AR-2026-00001",
		decimal.NewFromFloat(1000.00),
		decimal.NewFromFloat(900.00),
	)
	result.AddDiscrepancy(d1)

	assert.Len(t, result.Discrepancies, 1)
	assert.Equal(t, 1, result.DiscrepancyCount)
	assert.Equal(t, 1, result.CriticalCount)
	assert.Equal(t, 0, result.WarningCount)
	assert.Equal(t, TrialBalanceStatusUnbalanced, result.Status)

	// Add warning discrepancy
	d2 := NewBalanceDiscrepancy(
		ReceivableAmountMismatch,
		"AccountReceivable",
		uuid.New(),
		"AR-2026-00002",
		decimal.NewFromFloat(100.00),
		decimal.NewFromFloat(100.005),
	)
	result.AddDiscrepancy(d2)

	assert.Len(t, result.Discrepancies, 2)
	assert.Equal(t, 2, result.DiscrepancyCount)
	assert.Equal(t, 1, result.CriticalCount)
	assert.Equal(t, 1, result.WarningCount)
}

func TestTrialBalanceResult_SetTotals(t *testing.T) {
	result := NewTrialBalanceResult(uuid.New(), uuid.New())

	debits := decimal.NewFromFloat(10000.00)
	credits := decimal.NewFromFloat(8000.00)

	result.SetTotals(debits, credits)

	assert.True(t, debits.Equal(result.TotalDebits))
	assert.True(t, credits.Equal(result.TotalCredits))
	assert.True(t, decimal.NewFromFloat(2000.00).Equal(result.NetBalance))
}

func TestTrialBalanceResult_SetPeriod(t *testing.T) {
	result := NewTrialBalanceResult(uuid.New(), uuid.New())

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	result.SetPeriod(&start, &end)

	require.NotNil(t, result.PeriodStart)
	require.NotNil(t, result.PeriodEnd)
	assert.Equal(t, start, *result.PeriodStart)
	assert.Equal(t, end, *result.PeriodEnd)
}

func TestTrialBalanceResult_IsBalanced(t *testing.T) {
	t.Run("returns true when balanced and no discrepancies", func(t *testing.T) {
		result := NewTrialBalanceResult(uuid.New(), uuid.New())
		result.SetTotals(decimal.NewFromFloat(1000), decimal.NewFromFloat(1000))

		assert.True(t, result.IsBalanced())
	})

	t.Run("returns false when net balance is non-zero", func(t *testing.T) {
		result := NewTrialBalanceResult(uuid.New(), uuid.New())
		result.SetTotals(decimal.NewFromFloat(1000), decimal.NewFromFloat(800))

		assert.False(t, result.IsBalanced())
	})

	t.Run("returns false when discrepancies exist", func(t *testing.T) {
		result := NewTrialBalanceResult(uuid.New(), uuid.New())
		result.SetTotals(decimal.NewFromFloat(1000), decimal.NewFromFloat(1000))

		d := NewBalanceDiscrepancy(
			ReceivableAmountMismatch,
			"AccountReceivable",
			uuid.New(),
			"AR-2026-00001",
			decimal.NewFromFloat(1000.00),
			decimal.NewFromFloat(900.00),
		)
		result.AddDiscrepancy(d)

		assert.False(t, result.IsBalanced())
	})
}

func TestTrialBalanceResult_HasCriticalDiscrepancies(t *testing.T) {
	t.Run("returns false when no discrepancies", func(t *testing.T) {
		result := NewTrialBalanceResult(uuid.New(), uuid.New())
		assert.False(t, result.HasCriticalDiscrepancies())
	})

	t.Run("returns true when critical discrepancies exist", func(t *testing.T) {
		result := NewTrialBalanceResult(uuid.New(), uuid.New())

		d := NewBalanceDiscrepancy(
			ReceivableAmountMismatch,
			"AccountReceivable",
			uuid.New(),
			"AR-2026-00001",
			decimal.NewFromFloat(1000.00),
			decimal.NewFromFloat(900.00),
		)
		result.AddDiscrepancy(d)

		assert.True(t, result.HasCriticalDiscrepancies())
	})
}

func TestTrialBalanceResult_GetCriticalDiscrepancies(t *testing.T) {
	result := NewTrialBalanceResult(uuid.New(), uuid.New())

	// Add critical discrepancy
	d1 := NewBalanceDiscrepancy(
		ReceivableAmountMismatch,
		"AccountReceivable",
		uuid.New(),
		"AR-2026-00001",
		decimal.NewFromFloat(1000.00),
		decimal.NewFromFloat(900.00),
	)
	result.AddDiscrepancy(d1)

	// Add warning discrepancy
	d2 := NewBalanceDiscrepancy(
		ReceivableAmountMismatch,
		"AccountReceivable",
		uuid.New(),
		"AR-2026-00002",
		decimal.NewFromFloat(100.00),
		decimal.NewFromFloat(100.005),
	)
	result.AddDiscrepancy(d2)

	critical := result.GetCriticalDiscrepancies()

	assert.Len(t, critical, 1)
	assert.Equal(t, "CRITICAL", critical[0].Severity)
}

func TestDefaultTrialBalanceCheckOptions(t *testing.T) {
	opts := DefaultTrialBalanceCheckOptions()

	assert.True(t, opts.CheckReceivables)
	assert.True(t, opts.CheckPayables)
	assert.True(t, opts.CheckReceipts)
	assert.True(t, opts.CheckPayments)
	assert.True(t, opts.CheckCreditMemos)
	assert.True(t, opts.CheckDebitMemos)
	assert.True(t, opts.ValidateInternalConsistency)
	assert.True(t, opts.ValidateVoucherAllocations)
	assert.True(t, opts.ValidateMemoApplications)
	assert.True(t, opts.Tolerance.Equal(decimal.NewFromFloat(0.01)))
}

func TestNewTrialBalanceAuditLog(t *testing.T) {
	result := NewTrialBalanceResult(uuid.New(), uuid.New())
	result.SetTotals(decimal.NewFromFloat(10000), decimal.NewFromFloat(8000))
	result.SetExecutionDuration(150)

	d := NewBalanceDiscrepancy(
		ReceivableAmountMismatch,
		"AccountReceivable",
		uuid.New(),
		"AR-2026-00001",
		decimal.NewFromFloat(1000.00),
		decimal.NewFromFloat(900.00),
	)
	result.AddDiscrepancy(d)

	log := NewTrialBalanceAuditLog(result)

	assert.NotEqual(t, uuid.Nil, log.ID)
	assert.Equal(t, result.TenantID, log.TenantID)
	assert.Equal(t, result.CheckedAt, log.CheckedAt)
	assert.Equal(t, result.CheckedBy, log.CheckedBy)
	assert.Equal(t, result.Status, log.Status)
	assert.True(t, result.TotalDebits.Equal(log.TotalDebits))
	assert.True(t, result.TotalCredits.Equal(log.TotalCredits))
	assert.True(t, result.NetBalance.Equal(log.NetBalance))
	assert.Equal(t, result.DiscrepancyCount, log.DiscrepancyCount)
	assert.Equal(t, result.CriticalCount, log.CriticalCount)
	assert.Equal(t, result.WarningCount, log.WarningCount)
	assert.Equal(t, result.ExecutionDurationMs, log.DurationMs)
}

func TestBalanceCheckGuardResult(t *testing.T) {
	t.Run("NewAllowedGuardResult creates allowed result", func(t *testing.T) {
		result := NewAllowedGuardResult()

		assert.True(t, result.Allowed)
		assert.Equal(t, TrialBalanceStatusBalanced, result.Status)
		assert.Contains(t, result.Message, "allowed")
		assert.Empty(t, result.Discrepancies)
	})

	t.Run("NewBlockedGuardResult creates blocked result with discrepancies", func(t *testing.T) {
		d := NewBalanceDiscrepancy(
			ReceivableAmountMismatch,
			"AccountReceivable",
			uuid.New(),
			"AR-2026-00001",
			decimal.NewFromFloat(1000.00),
			decimal.NewFromFloat(900.00),
		)

		result := NewBlockedGuardResult([]BalanceDiscrepancy{*d})

		assert.False(t, result.Allowed)
		assert.Equal(t, TrialBalanceStatusUnbalanced, result.Status)
		assert.Contains(t, result.Message, "blocked")
		assert.Len(t, result.Discrepancies, 1)
	})
}
