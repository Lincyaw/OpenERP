package strategy

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Invoice represents an invoice for payment allocation
type Invoice struct {
	ID            string
	TenantID      string
	CustomerID    string
	InvoiceNumber string
	InvoiceDate   time.Time
	DueDate       time.Time
	TotalAmount   decimal.Decimal
	PaidAmount    decimal.Decimal
	Balance       decimal.Decimal
	Currency      string
	Priority      int
}

// Allocation represents a payment allocation to an invoice
type Allocation struct {
	InvoiceID       string
	InvoiceNumber   string
	AllocatedAmount decimal.Decimal
	BalanceBefore   decimal.Decimal
	BalanceAfter    decimal.Decimal
}

// AllocationContext provides context for payment allocation
type AllocationContext struct {
	TenantID      string
	CustomerID    string
	PaymentAmount decimal.Decimal
	PaymentDate   time.Time
	Currency      string
}

// AllocationResult contains the result of payment allocation
type AllocationResult struct {
	Allocations    []Allocation
	TotalAllocated decimal.Decimal
	Remaining      decimal.Decimal
}

// PaymentAllocationStrategy defines the interface for payment allocation
type PaymentAllocationStrategy interface {
	Strategy
	// Allocate allocates a payment amount to outstanding invoices
	Allocate(ctx context.Context, allocCtx AllocationContext, invoices []Invoice) (AllocationResult, error)
	// SupportsPartialAllocation returns true if the strategy supports partial allocation
	SupportsPartialAllocation() bool
}
