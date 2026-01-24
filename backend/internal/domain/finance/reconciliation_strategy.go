package finance

import (
	"sort"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReconciliationStrategyType defines the type of reconciliation strategy
type ReconciliationStrategyType string

const (
	ReconciliationStrategyTypeFIFO   ReconciliationStrategyType = "FIFO"   // First In First Out by date
	ReconciliationStrategyTypeManual ReconciliationStrategyType = "MANUAL" // Manual allocation to specific items
)

// IsValid checks if the strategy type is valid
func (t ReconciliationStrategyType) IsValid() bool {
	switch t {
	case ReconciliationStrategyTypeFIFO, ReconciliationStrategyTypeManual:
		return true
	}
	return false
}

// String returns the string representation
func (t ReconciliationStrategyType) String() string {
	return string(t)
}

// AllReconciliationStrategyTypes returns all valid reconciliation strategy types
func AllReconciliationStrategyTypes() []ReconciliationStrategyType {
	return []ReconciliationStrategyType{
		ReconciliationStrategyTypeFIFO,
		ReconciliationStrategyTypeManual,
	}
}

// AllocationTarget represents a target receivable/payable for allocation
type AllocationTarget struct {
	ID                uuid.UUID       // ID of the receivable/payable
	Number            string          // Number for display purposes
	OutstandingAmount decimal.Decimal // Amount still outstanding
	DueDate           *time.Time      // Due date for FIFO ordering
	CreatedAt         time.Time       // Creation date as fallback ordering
}

// AllocationResult represents the result of a single allocation
type AllocationResult struct {
	TargetID     uuid.UUID       // ID of the receivable/payable
	TargetNumber string          // Number of the receivable/payable
	Amount       decimal.Decimal // Amount to allocate
}

// ReconciliationResult represents the complete result of a reconciliation strategy
type ReconciliationResult struct {
	Allocations          []AllocationResult // List of allocations to make
	TotalAllocated       decimal.Decimal    // Total amount allocated
	RemainingAmount      decimal.Decimal    // Amount left unallocated
	FullyReconciled      bool               // True if all amount was allocated
	TargetsFullyPaid     []uuid.UUID        // Targets that will be fully paid
	TargetsPartiallyPaid []uuid.UUID        // Targets that will be partially paid
}

// ReconciliationStrategy is the interface for reconciliation strategies
type ReconciliationStrategy interface {
	strategy.Strategy
	// StrategyType returns the reconciliation strategy type
	StrategyType() ReconciliationStrategyType
	// Allocate calculates how to allocate the given amount across targets
	Allocate(amount valueobject.Money, targets []AllocationTarget) (*ReconciliationResult, error)
}

// ReceivableReconciliationStrategy handles allocation of receipts to receivables
type ReceivableReconciliationStrategy interface {
	ReconciliationStrategy
	// AllocateReceipt calculates how to allocate a receipt voucher to receivables
	AllocateReceipt(voucher *ReceiptVoucher, receivables []AccountReceivable) (*ReconciliationResult, error)
}

// PayableReconciliationStrategy handles allocation of payments to payables
type PayableReconciliationStrategy interface {
	ReconciliationStrategy
	// AllocatePayment calculates how to allocate a payment voucher to payables
	AllocatePayment(voucher *PaymentVoucher, payables []AccountPayable) (*ReconciliationResult, error)
}

// FIFOReconciliationStrategy implements FIFO (First In First Out) reconciliation
// It allocates payments/receipts to the oldest outstanding items first
type FIFOReconciliationStrategy struct {
	strategy.BaseStrategy
}

// NewFIFOReconciliationStrategy creates a new FIFO reconciliation strategy
func NewFIFOReconciliationStrategy() *FIFOReconciliationStrategy {
	return &FIFOReconciliationStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"fifo_reconciliation",
			strategy.StrategyTypeAllocation,
			"FIFO reconciliation strategy - allocates to oldest outstanding items first by due date, then creation date",
		),
	}
}

// StrategyType returns the reconciliation strategy type
func (s *FIFOReconciliationStrategy) StrategyType() ReconciliationStrategyType {
	return ReconciliationStrategyTypeFIFO
}

// Allocate allocates the amount to targets using FIFO order (oldest first)
func (s *FIFOReconciliationStrategy) Allocate(amount valueobject.Money, targets []AllocationTarget) (*ReconciliationResult, error) {
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Allocation amount must be positive")
	}
	if len(targets) == 0 {
		return &ReconciliationResult{
			Allocations:          make([]AllocationResult, 0),
			TotalAllocated:       decimal.Zero,
			RemainingAmount:      amount.Amount(),
			FullyReconciled:      false,
			TargetsFullyPaid:     make([]uuid.UUID, 0),
			TargetsPartiallyPaid: make([]uuid.UUID, 0),
		}, nil
	}

	// Sort targets by FIFO order: due date first, then creation date
	sortedTargets := make([]AllocationTarget, len(targets))
	copy(sortedTargets, targets)
	sort.Slice(sortedTargets, func(i, j int) bool {
		// Sort by due date first (nil due dates go to the end)
		if sortedTargets[i].DueDate != nil && sortedTargets[j].DueDate != nil {
			if !sortedTargets[i].DueDate.Equal(*sortedTargets[j].DueDate) {
				return sortedTargets[i].DueDate.Before(*sortedTargets[j].DueDate)
			}
		} else if sortedTargets[i].DueDate != nil {
			return true // i has due date, j doesn't - i comes first
		} else if sortedTargets[j].DueDate != nil {
			return false // j has due date, i doesn't - j comes first
		}
		// Fall back to creation date
		return sortedTargets[i].CreatedAt.Before(sortedTargets[j].CreatedAt)
	})

	allocations := make([]AllocationResult, 0)
	fullyPaid := make([]uuid.UUID, 0)
	partiallyPaid := make([]uuid.UUID, 0)
	remaining := amount.Amount()
	totalAllocated := decimal.Zero

	for _, target := range sortedTargets {
		if remaining.IsZero() {
			break
		}
		if target.OutstandingAmount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// Calculate allocation amount
		allocAmount := decimal.Min(remaining, target.OutstandingAmount)

		allocations = append(allocations, AllocationResult{
			TargetID:     target.ID,
			TargetNumber: target.Number,
			Amount:       allocAmount,
		})

		totalAllocated = totalAllocated.Add(allocAmount)
		remaining = remaining.Sub(allocAmount)

		// Track which targets will be fully vs partially paid
		if allocAmount.GreaterThanOrEqual(target.OutstandingAmount) {
			fullyPaid = append(fullyPaid, target.ID)
		} else {
			partiallyPaid = append(partiallyPaid, target.ID)
		}
	}

	return &ReconciliationResult{
		Allocations:          allocations,
		TotalAllocated:       totalAllocated,
		RemainingAmount:      remaining,
		FullyReconciled:      remaining.IsZero(),
		TargetsFullyPaid:     fullyPaid,
		TargetsPartiallyPaid: partiallyPaid,
	}, nil
}

// AllocateReceipt allocates a receipt voucher to receivables using FIFO
func (s *FIFOReconciliationStrategy) AllocateReceipt(voucher *ReceiptVoucher, receivables []AccountReceivable) (*ReconciliationResult, error) {
	if voucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Receipt voucher cannot be nil")
	}
	if voucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Receipt voucher has no unallocated amount")
	}

	// Convert receivables to allocation targets
	targets := make([]AllocationTarget, 0, len(receivables))
	for _, r := range receivables {
		// Only include receivables that can receive payments
		if r.Status.CanApplyPayment() && r.OutstandingAmount.GreaterThan(decimal.Zero) {
			targets = append(targets, AllocationTarget{
				ID:                r.ID,
				Number:            r.ReceivableNumber,
				OutstandingAmount: r.OutstandingAmount,
				DueDate:           r.DueDate,
				CreatedAt:         r.CreatedAt,
			})
		}
	}

	return s.Allocate(valueobject.NewMoneyCNY(voucher.UnallocatedAmount), targets)
}

// AllocatePayment allocates a payment voucher to payables using FIFO
func (s *FIFOReconciliationStrategy) AllocatePayment(voucher *PaymentVoucher, payables []AccountPayable) (*ReconciliationResult, error) {
	if voucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Payment voucher cannot be nil")
	}
	if voucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Payment voucher has no unallocated amount")
	}

	// Convert payables to allocation targets
	targets := make([]AllocationTarget, 0, len(payables))
	for _, p := range payables {
		// Only include payables that can receive payments
		if p.Status.CanApplyPayment() && p.OutstandingAmount.GreaterThan(decimal.Zero) {
			targets = append(targets, AllocationTarget{
				ID:                p.ID,
				Number:            p.PayableNumber,
				OutstandingAmount: p.OutstandingAmount,
				DueDate:           p.DueDate,
				CreatedAt:         p.CreatedAt,
			})
		}
	}

	return s.Allocate(valueobject.NewMoneyCNY(voucher.UnallocatedAmount), targets)
}

// ManualAllocationRequest represents a request to manually allocate to a specific target
type ManualAllocationRequest struct {
	TargetID uuid.UUID       // ID of the receivable/payable
	Amount   decimal.Decimal // Amount to allocate (if zero, allocate full outstanding)
}

// ManualReconciliationStrategy implements manual allocation to specified targets
type ManualReconciliationStrategy struct {
	strategy.BaseStrategy
	allocations []ManualAllocationRequest
}

// NewManualReconciliationStrategy creates a new manual reconciliation strategy
func NewManualReconciliationStrategy(allocations []ManualAllocationRequest) *ManualReconciliationStrategy {
	return &ManualReconciliationStrategy{
		BaseStrategy: strategy.NewBaseStrategy(
			"manual_reconciliation",
			strategy.StrategyTypeAllocation,
			"Manual reconciliation strategy - allocates to user-specified targets in order",
		),
		allocations: allocations,
	}
}

// StrategyType returns the reconciliation strategy type
func (s *ManualReconciliationStrategy) StrategyType() ReconciliationStrategyType {
	return ReconciliationStrategyTypeManual
}

// GetAllocations returns the configured manual allocations
func (s *ManualReconciliationStrategy) GetAllocations() []ManualAllocationRequest {
	return s.allocations
}

// Allocate allocates the amount to targets based on manual allocation requests
func (s *ManualReconciliationStrategy) Allocate(amount valueobject.Money, targets []AllocationTarget) (*ReconciliationResult, error) {
	if amount.Amount().LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Allocation amount must be positive")
	}
	if len(targets) == 0 {
		return &ReconciliationResult{
			Allocations:          make([]AllocationResult, 0),
			TotalAllocated:       decimal.Zero,
			RemainingAmount:      amount.Amount(),
			FullyReconciled:      false,
			TargetsFullyPaid:     make([]uuid.UUID, 0),
			TargetsPartiallyPaid: make([]uuid.UUID, 0),
		}, nil
	}

	// Build target map for quick lookup
	targetMap := make(map[uuid.UUID]*AllocationTarget)
	for i := range targets {
		targetMap[targets[i].ID] = &targets[i]
	}

	allocations := make([]AllocationResult, 0)
	fullyPaid := make([]uuid.UUID, 0)
	partiallyPaid := make([]uuid.UUID, 0)
	remaining := amount.Amount()
	totalAllocated := decimal.Zero

	// Process manual allocation requests in order
	for _, req := range s.allocations {
		if remaining.IsZero() {
			break
		}

		target, exists := targetMap[req.TargetID]
		if !exists {
			// Skip invalid targets
			continue
		}
		if target.OutstandingAmount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// Determine allocation amount
		var allocAmount decimal.Decimal
		if req.Amount.IsZero() {
			// If amount not specified, allocate as much as possible
			allocAmount = decimal.Min(remaining, target.OutstandingAmount)
		} else {
			// Use specified amount, but cap at available/outstanding
			allocAmount = decimal.Min(req.Amount, remaining)
			allocAmount = decimal.Min(allocAmount, target.OutstandingAmount)
		}

		if allocAmount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		allocations = append(allocations, AllocationResult{
			TargetID:     target.ID,
			TargetNumber: target.Number,
			Amount:       allocAmount,
		})

		totalAllocated = totalAllocated.Add(allocAmount)
		remaining = remaining.Sub(allocAmount)

		// Track which targets will be fully vs partially paid
		if allocAmount.GreaterThanOrEqual(target.OutstandingAmount) {
			fullyPaid = append(fullyPaid, target.ID)
		} else {
			partiallyPaid = append(partiallyPaid, target.ID)
		}

		// Update target's available amount for next iteration
		target.OutstandingAmount = target.OutstandingAmount.Sub(allocAmount)
	}

	return &ReconciliationResult{
		Allocations:          allocations,
		TotalAllocated:       totalAllocated,
		RemainingAmount:      remaining,
		FullyReconciled:      remaining.IsZero(),
		TargetsFullyPaid:     fullyPaid,
		TargetsPartiallyPaid: partiallyPaid,
	}, nil
}

// AllocateReceipt allocates a receipt voucher to receivables using manual allocations
func (s *ManualReconciliationStrategy) AllocateReceipt(voucher *ReceiptVoucher, receivables []AccountReceivable) (*ReconciliationResult, error) {
	if voucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Receipt voucher cannot be nil")
	}
	if voucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Receipt voucher has no unallocated amount")
	}

	// Convert receivables to allocation targets
	targets := make([]AllocationTarget, 0, len(receivables))
	for _, r := range receivables {
		if r.Status.CanApplyPayment() && r.OutstandingAmount.GreaterThan(decimal.Zero) {
			targets = append(targets, AllocationTarget{
				ID:                r.ID,
				Number:            r.ReceivableNumber,
				OutstandingAmount: r.OutstandingAmount,
				DueDate:           r.DueDate,
				CreatedAt:         r.CreatedAt,
			})
		}
	}

	return s.Allocate(valueobject.NewMoneyCNY(voucher.UnallocatedAmount), targets)
}

// AllocatePayment allocates a payment voucher to payables using manual allocations
func (s *ManualReconciliationStrategy) AllocatePayment(voucher *PaymentVoucher, payables []AccountPayable) (*ReconciliationResult, error) {
	if voucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Payment voucher cannot be nil")
	}
	if voucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Payment voucher has no unallocated amount")
	}

	// Convert payables to allocation targets
	targets := make([]AllocationTarget, 0, len(payables))
	for _, p := range payables {
		if p.Status.CanApplyPayment() && p.OutstandingAmount.GreaterThan(decimal.Zero) {
			targets = append(targets, AllocationTarget{
				ID:                p.ID,
				Number:            p.PayableNumber,
				OutstandingAmount: p.OutstandingAmount,
				DueDate:           p.DueDate,
				CreatedAt:         p.CreatedAt,
			})
		}
	}

	return s.Allocate(valueobject.NewMoneyCNY(voucher.UnallocatedAmount), targets)
}

// ReconciliationStrategyFactory creates reconciliation strategies
type ReconciliationStrategyFactory struct{}

// NewReconciliationStrategyFactory creates a new factory
func NewReconciliationStrategyFactory() *ReconciliationStrategyFactory {
	return &ReconciliationStrategyFactory{}
}

// CreateFIFOStrategy creates a FIFO reconciliation strategy
func (f *ReconciliationStrategyFactory) CreateFIFOStrategy() *FIFOReconciliationStrategy {
	return NewFIFOReconciliationStrategy()
}

// CreateManualStrategy creates a manual reconciliation strategy with specified allocations
func (f *ReconciliationStrategyFactory) CreateManualStrategy(allocations []ManualAllocationRequest) *ManualReconciliationStrategy {
	return NewManualReconciliationStrategy(allocations)
}

// GetStrategy returns a strategy by type
func (f *ReconciliationStrategyFactory) GetStrategy(strategyType ReconciliationStrategyType, allocations []ManualAllocationRequest) (ReconciliationStrategy, error) {
	switch strategyType {
	case ReconciliationStrategyTypeFIFO:
		return f.CreateFIFOStrategy(), nil
	case ReconciliationStrategyTypeManual:
		if len(allocations) == 0 {
			return nil, shared.NewDomainError("INVALID_ALLOCATIONS", "Manual strategy requires allocation requests")
		}
		return f.CreateManualStrategy(allocations), nil
	default:
		return nil, shared.NewDomainError("INVALID_STRATEGY", "Unknown reconciliation strategy type")
	}
}
