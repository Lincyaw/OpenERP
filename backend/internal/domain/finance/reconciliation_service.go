package finance

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReconciliationService is a domain service that coordinates the allocation of
// receipt/payment vouchers to receivables/payables using reconciliation strategies.
// It ensures that:
// 1. Vouchers are confirmed before allocation
// 2. Allocations don't exceed outstanding amounts
// 3. Both voucher and receivable/payable states are updated consistently
//
// The service supports dependency injection of reconciliation strategies, allowing
// for configurable allocation behavior per tenant or use case.
type ReconciliationService struct {
	strategyFactory      *ReconciliationStrategyFactory
	defaultStrategyType  ReconciliationStrategyType
	strategyOverrideFunc StrategyOverrideFunc
}

// StrategyOverrideFunc is a function that can override the strategy type based on context.
// This allows for tenant-specific or context-specific strategy selection.
type StrategyOverrideFunc func(ctx context.Context, tenantID uuid.UUID) ReconciliationStrategyType

// ReconciliationServiceOption is a functional option for configuring ReconciliationService
type ReconciliationServiceOption func(*ReconciliationService)

// WithDefaultStrategy sets the default reconciliation strategy type
func WithDefaultStrategy(strategyType ReconciliationStrategyType) ReconciliationServiceOption {
	return func(s *ReconciliationService) {
		if strategyType.IsValid() {
			s.defaultStrategyType = strategyType
		}
	}
}

// WithStrategyOverride sets a function that can override strategy selection based on context
func WithStrategyOverride(fn StrategyOverrideFunc) ReconciliationServiceOption {
	return func(s *ReconciliationService) {
		s.strategyOverrideFunc = fn
	}
}

// NewReconciliationService creates a new reconciliation service with optional configuration
func NewReconciliationService(opts ...ReconciliationServiceOption) *ReconciliationService {
	s := &ReconciliationService{
		strategyFactory:     NewReconciliationStrategyFactory(),
		defaultStrategyType: ReconciliationStrategyTypeFIFO, // Default to FIFO
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetDefaultStrategy returns the default strategy type
func (s *ReconciliationService) GetDefaultStrategy() ReconciliationStrategyType {
	return s.defaultStrategyType
}

// GetEffectiveStrategy returns the effective strategy type for a given context and tenant
func (s *ReconciliationService) GetEffectiveStrategy(ctx context.Context, tenantID uuid.UUID) ReconciliationStrategyType {
	if s.strategyOverrideFunc != nil {
		override := s.strategyOverrideFunc(ctx, tenantID)
		if override.IsValid() {
			return override
		}
	}
	return s.defaultStrategyType
}

// ReconcileReceiptRequest represents a request to reconcile a receipt voucher
type ReconcileReceiptRequest struct {
	ReceiptVoucher *ReceiptVoucher
	Receivables    []AccountReceivable
	StrategyType   ReconciliationStrategyType
	// ManualAllocations is only used when StrategyType is MANUAL
	ManualAllocations []ManualAllocationRequest
}

// ReconcileReceiptResult represents the result of reconciling a receipt voucher
type ReconcileReceiptResult struct {
	ReceiptVoucher       *ReceiptVoucher        // Updated voucher with new allocations
	UpdatedReceivables   []AccountReceivable    // Receivables that were updated
	Allocations          []ReceivableAllocation // Allocations that were made
	TotalReconciled      decimal.Decimal        // Total amount reconciled
	RemainingUnallocated decimal.Decimal        // Amount still unallocated
	FullyReconciled      bool                   // True if all voucher amount was allocated
}

// ReconcileReceipt reconciles a receipt voucher to receivables using the specified strategy
func (s *ReconciliationService) ReconcileReceipt(
	ctx context.Context,
	req ReconcileReceiptRequest,
) (*ReconcileReceiptResult, error) {
	// Validate voucher
	if req.ReceiptVoucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Receipt voucher cannot be nil")
	}
	if !req.ReceiptVoucher.Status.CanAllocate() {
		return nil, shared.NewDomainError("INVALID_STATE",
			fmt.Sprintf("Cannot allocate voucher in %s status, must be CONFIRMED", req.ReceiptVoucher.Status))
	}
	if req.ReceiptVoucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Receipt voucher has no unallocated amount")
	}

	// Validate strategy type
	if !req.StrategyType.IsValid() {
		return nil, shared.NewDomainError("INVALID_STRATEGY", "Invalid reconciliation strategy type")
	}

	// Create the appropriate strategy
	strategy, err := s.strategyFactory.GetStrategy(req.StrategyType, req.ManualAllocations)
	if err != nil {
		return nil, err
	}

	// Filter receivables for the same customer
	customerReceivables := make([]AccountReceivable, 0)
	for _, r := range req.Receivables {
		if r.CustomerID == req.ReceiptVoucher.CustomerID &&
			r.Status.CanApplyPayment() &&
			r.OutstandingAmount.GreaterThan(decimal.Zero) {
			customerReceivables = append(customerReceivables, r)
		}
	}

	if len(customerReceivables) == 0 {
		return &ReconcileReceiptResult{
			ReceiptVoucher:       req.ReceiptVoucher,
			UpdatedReceivables:   []AccountReceivable{},
			Allocations:          []ReceivableAllocation{},
			TotalReconciled:      decimal.Zero,
			RemainingUnallocated: req.ReceiptVoucher.UnallocatedAmount,
			FullyReconciled:      false,
		}, nil
	}

	// Get strategy implementation
	receivableStrategy, ok := strategy.(ReceivableReconciliationStrategy)
	if !ok {
		return nil, shared.NewDomainError("STRATEGY_ERROR", "Strategy does not support receivable reconciliation")
	}

	// Calculate allocations using the strategy
	reconciliationResult, err := receivableStrategy.AllocateReceipt(req.ReceiptVoucher, customerReceivables)
	if err != nil {
		return nil, err
	}

	// Apply allocations to voucher and receivables
	updatedReceivables := make([]AccountReceivable, 0)
	allocations := make([]ReceivableAllocation, 0)

	// Create a map for quick receivable lookup
	receivableMap := make(map[uuid.UUID]*AccountReceivable)
	for i := range customerReceivables {
		receivableMap[customerReceivables[i].ID] = &customerReceivables[i]
	}

	// Apply each allocation
	for _, alloc := range reconciliationResult.Allocations {
		receivable, exists := receivableMap[alloc.TargetID]
		if !exists {
			continue
		}

		allocAmount := valueobject.NewMoneyCNY(alloc.Amount)

		// Allocate on voucher
		allocation, err := req.ReceiptVoucher.AllocateToReceivable(
			receivable.ID,
			receivable.ReceivableNumber,
			allocAmount,
			fmt.Sprintf("Auto-reconciled via %s strategy", req.StrategyType),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate to receivable %s: %w", receivable.ReceivableNumber, err)
		}
		allocations = append(allocations, *allocation)

		// Apply payment to receivable
		err = receivable.ApplyPayment(allocAmount, req.ReceiptVoucher.ID,
			fmt.Sprintf("Payment from receipt voucher %s", req.ReceiptVoucher.VoucherNumber))
		if err != nil {
			return nil, fmt.Errorf("failed to apply payment to receivable %s: %w", receivable.ReceivableNumber, err)
		}
		updatedReceivables = append(updatedReceivables, *receivable)
	}

	return &ReconcileReceiptResult{
		ReceiptVoucher:       req.ReceiptVoucher,
		UpdatedReceivables:   updatedReceivables,
		Allocations:          allocations,
		TotalReconciled:      reconciliationResult.TotalAllocated,
		RemainingUnallocated: reconciliationResult.RemainingAmount,
		FullyReconciled:      reconciliationResult.FullyReconciled,
	}, nil
}

// ReconcilePaymentRequest represents a request to reconcile a payment voucher
type ReconcilePaymentRequest struct {
	PaymentVoucher *PaymentVoucher
	Payables       []AccountPayable
	StrategyType   ReconciliationStrategyType
	// ManualAllocations is only used when StrategyType is MANUAL
	ManualAllocations []ManualAllocationRequest
}

// ReconcilePaymentResult represents the result of reconciling a payment voucher
type ReconcilePaymentResult struct {
	PaymentVoucher       *PaymentVoucher     // Updated voucher with new allocations
	UpdatedPayables      []AccountPayable    // Payables that were updated
	Allocations          []PayableAllocation // Allocations that were made
	TotalReconciled      decimal.Decimal     // Total amount reconciled
	RemainingUnallocated decimal.Decimal     // Amount still unallocated
	FullyReconciled      bool                // True if all voucher amount was allocated
}

// ReconcilePayment reconciles a payment voucher to payables using the specified strategy
func (s *ReconciliationService) ReconcilePayment(
	ctx context.Context,
	req ReconcilePaymentRequest,
) (*ReconcilePaymentResult, error) {
	// Validate voucher
	if req.PaymentVoucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Payment voucher cannot be nil")
	}
	if !req.PaymentVoucher.Status.CanAllocate() {
		return nil, shared.NewDomainError("INVALID_STATE",
			fmt.Sprintf("Cannot allocate voucher in %s status, must be CONFIRMED", req.PaymentVoucher.Status))
	}
	if req.PaymentVoucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Payment voucher has no unallocated amount")
	}

	// Validate strategy type
	if !req.StrategyType.IsValid() {
		return nil, shared.NewDomainError("INVALID_STRATEGY", "Invalid reconciliation strategy type")
	}

	// Create the appropriate strategy
	strategy, err := s.strategyFactory.GetStrategy(req.StrategyType, req.ManualAllocations)
	if err != nil {
		return nil, err
	}

	// Filter payables for the same supplier
	supplierPayables := make([]AccountPayable, 0)
	for _, p := range req.Payables {
		if p.SupplierID == req.PaymentVoucher.SupplierID &&
			p.Status.CanApplyPayment() &&
			p.OutstandingAmount.GreaterThan(decimal.Zero) {
			supplierPayables = append(supplierPayables, p)
		}
	}

	if len(supplierPayables) == 0 {
		return &ReconcilePaymentResult{
			PaymentVoucher:       req.PaymentVoucher,
			UpdatedPayables:      []AccountPayable{},
			Allocations:          []PayableAllocation{},
			TotalReconciled:      decimal.Zero,
			RemainingUnallocated: req.PaymentVoucher.UnallocatedAmount,
			FullyReconciled:      false,
		}, nil
	}

	// Get strategy implementation
	payableStrategy, ok := strategy.(PayableReconciliationStrategy)
	if !ok {
		return nil, shared.NewDomainError("STRATEGY_ERROR", "Strategy does not support payable reconciliation")
	}

	// Calculate allocations using the strategy
	reconciliationResult, err := payableStrategy.AllocatePayment(req.PaymentVoucher, supplierPayables)
	if err != nil {
		return nil, err
	}

	// Apply allocations to voucher and payables
	updatedPayables := make([]AccountPayable, 0)
	allocations := make([]PayableAllocation, 0)

	// Create a map for quick payable lookup
	payableMap := make(map[uuid.UUID]*AccountPayable)
	for i := range supplierPayables {
		payableMap[supplierPayables[i].ID] = &supplierPayables[i]
	}

	// Apply each allocation
	for _, alloc := range reconciliationResult.Allocations {
		payable, exists := payableMap[alloc.TargetID]
		if !exists {
			continue
		}

		allocAmount := valueobject.NewMoneyCNY(alloc.Amount)

		// Allocate on voucher
		allocation, err := req.PaymentVoucher.AllocateToPayable(
			payable.ID,
			payable.PayableNumber,
			allocAmount,
			fmt.Sprintf("Auto-reconciled via %s strategy", req.StrategyType),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate to payable %s: %w", payable.PayableNumber, err)
		}
		allocations = append(allocations, *allocation)

		// Apply payment to payable
		err = payable.ApplyPayment(allocAmount, req.PaymentVoucher.ID,
			fmt.Sprintf("Payment from payment voucher %s", req.PaymentVoucher.VoucherNumber))
		if err != nil {
			return nil, fmt.Errorf("failed to apply payment to payable %s: %w", payable.PayableNumber, err)
		}
		updatedPayables = append(updatedPayables, *payable)
	}

	return &ReconcilePaymentResult{
		PaymentVoucher:       req.PaymentVoucher,
		UpdatedPayables:      updatedPayables,
		Allocations:          allocations,
		TotalReconciled:      reconciliationResult.TotalAllocated,
		RemainingUnallocated: reconciliationResult.RemainingAmount,
		FullyReconciled:      reconciliationResult.FullyReconciled,
	}, nil
}

// AutoReconcileReceipt reconciles a receipt voucher using FIFO strategy
func (s *ReconciliationService) AutoReconcileReceipt(
	ctx context.Context,
	voucher *ReceiptVoucher,
	receivables []AccountReceivable,
) (*ReconcileReceiptResult, error) {
	return s.ReconcileReceipt(ctx, ReconcileReceiptRequest{
		ReceiptVoucher: voucher,
		Receivables:    receivables,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})
}

// ManualReconcileReceipt reconciles a receipt voucher using manual allocations
func (s *ReconciliationService) ManualReconcileReceipt(
	ctx context.Context,
	voucher *ReceiptVoucher,
	receivables []AccountReceivable,
	allocations []ManualAllocationRequest,
) (*ReconcileReceiptResult, error) {
	return s.ReconcileReceipt(ctx, ReconcileReceiptRequest{
		ReceiptVoucher:    voucher,
		Receivables:       receivables,
		StrategyType:      ReconciliationStrategyTypeManual,
		ManualAllocations: allocations,
	})
}

// AutoReconcilePayment reconciles a payment voucher using FIFO strategy
func (s *ReconciliationService) AutoReconcilePayment(
	ctx context.Context,
	voucher *PaymentVoucher,
	payables []AccountPayable,
) (*ReconcilePaymentResult, error) {
	return s.ReconcilePayment(ctx, ReconcilePaymentRequest{
		PaymentVoucher: voucher,
		Payables:       payables,
		StrategyType:   ReconciliationStrategyTypeFIFO,
	})
}

// ManualReconcilePayment reconciles a payment voucher using manual allocations
func (s *ReconciliationService) ManualReconcilePayment(
	ctx context.Context,
	voucher *PaymentVoucher,
	payables []AccountPayable,
	allocations []ManualAllocationRequest,
) (*ReconcilePaymentResult, error) {
	return s.ReconcilePayment(ctx, ReconcilePaymentRequest{
		PaymentVoucher:    voucher,
		Payables:          payables,
		StrategyType:      ReconciliationStrategyTypeManual,
		ManualAllocations: allocations,
	})
}

// PreviewReconcileReceipt calculates what allocations would be made without applying them
// This is useful for showing the user what would happen before they confirm
func (s *ReconciliationService) PreviewReconcileReceipt(
	ctx context.Context,
	req ReconcileReceiptRequest,
) (*ReconciliationResult, error) {
	// Validate voucher
	if req.ReceiptVoucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Receipt voucher cannot be nil")
	}
	if req.ReceiptVoucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Receipt voucher has no unallocated amount")
	}

	// Create strategy
	strategy, err := s.strategyFactory.GetStrategy(req.StrategyType, req.ManualAllocations)
	if err != nil {
		return nil, err
	}

	// Filter receivables for the same customer
	customerReceivables := make([]AccountReceivable, 0)
	for _, r := range req.Receivables {
		if r.CustomerID == req.ReceiptVoucher.CustomerID &&
			r.Status.CanApplyPayment() &&
			r.OutstandingAmount.GreaterThan(decimal.Zero) {
			customerReceivables = append(customerReceivables, r)
		}
	}

	receivableStrategy, ok := strategy.(ReceivableReconciliationStrategy)
	if !ok {
		return nil, shared.NewDomainError("STRATEGY_ERROR", "Strategy does not support receivable reconciliation")
	}

	return receivableStrategy.AllocateReceipt(req.ReceiptVoucher, customerReceivables)
}

// PreviewReconcilePayment calculates what allocations would be made without applying them
func (s *ReconciliationService) PreviewReconcilePayment(
	ctx context.Context,
	req ReconcilePaymentRequest,
) (*ReconciliationResult, error) {
	// Validate voucher
	if req.PaymentVoucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Payment voucher cannot be nil")
	}
	if req.PaymentVoucher.UnallocatedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("NO_UNALLOCATED", "Payment voucher has no unallocated amount")
	}

	// Create strategy
	strategy, err := s.strategyFactory.GetStrategy(req.StrategyType, req.ManualAllocations)
	if err != nil {
		return nil, err
	}

	// Filter payables for the same supplier
	supplierPayables := make([]AccountPayable, 0)
	for _, p := range req.Payables {
		if p.SupplierID == req.PaymentVoucher.SupplierID &&
			p.Status.CanApplyPayment() &&
			p.OutstandingAmount.GreaterThan(decimal.Zero) {
			supplierPayables = append(supplierPayables, p)
		}
	}

	payableStrategy, ok := strategy.(PayableReconciliationStrategy)
	if !ok {
		return nil, shared.NewDomainError("STRATEGY_ERROR", "Strategy does not support payable reconciliation")
	}

	return payableStrategy.AllocatePayment(req.PaymentVoucher, supplierPayables)
}
