package allocation

import (
	"context"
	"sort"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
)

// FIFOAllocationStrategy implements First-In-First-Out payment allocation
type FIFOAllocationStrategy struct {
	strategy.BaseStrategy
}

// NewFIFOAllocationStrategy creates a new FIFO allocation strategy
func NewFIFOAllocationStrategy() *FIFOAllocationStrategy {
	return &FIFOAllocationStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"fifo",
			strategy.StrategyTypeAllocation,
			"Allocate payments to oldest invoices first",
		),
	}
}

// Allocate allocates payment to invoices in FIFO order
func (s *FIFOAllocationStrategy) Allocate(
	ctx context.Context,
	allocCtx strategy.AllocationContext,
	invoices []strategy.Invoice,
) (strategy.AllocationResult, error) {
	// Sort invoices by invoice date (oldest first)
	sortedInvoices := make([]strategy.Invoice, len(invoices))
	copy(sortedInvoices, invoices)
	sort.Slice(sortedInvoices, func(i, j int) bool {
		return sortedInvoices[i].InvoiceDate.Before(sortedInvoices[j].InvoiceDate)
	})

	remainingAmount := allocCtx.PaymentAmount
	allocations := make([]strategy.Allocation, 0)
	totalAllocated := decimal.Zero

	for _, invoice := range sortedInvoices {
		if remainingAmount.IsZero() || remainingAmount.IsNegative() {
			break
		}

		if invoice.Balance.IsZero() || invoice.Balance.IsNegative() {
			continue
		}

		allocatedAmount := decimal.Min(remainingAmount, invoice.Balance)
		balanceAfter := invoice.Balance.Sub(allocatedAmount)

		allocations = append(allocations, strategy.Allocation{
			InvoiceID:       invoice.ID,
			InvoiceNumber:   invoice.InvoiceNumber,
			AllocatedAmount: allocatedAmount,
			BalanceBefore:   invoice.Balance,
			BalanceAfter:    balanceAfter,
		})

		remainingAmount = remainingAmount.Sub(allocatedAmount)
		totalAllocated = totalAllocated.Add(allocatedAmount)
	}

	return strategy.AllocationResult{
		Allocations:    allocations,
		TotalAllocated: totalAllocated,
		Remaining:      remainingAmount,
	}, nil
}

// SupportsPartialAllocation returns true as FIFO supports partial allocation
func (s *FIFOAllocationStrategy) SupportsPartialAllocation() bool {
	return true
}
