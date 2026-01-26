package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TrialBalanceService provides trial balance check functionality
type TrialBalanceService struct {
	receivableRepo     AccountReceivableRepository
	payableRepo        AccountPayableRepository
	receiptVoucherRepo ReceiptVoucherRepository
	paymentVoucherRepo PaymentVoucherRepository
	creditMemoRepo     CreditMemoRepository
	debitMemoRepo      DebitMemoRepository
	auditLogRepo       TrialBalanceAuditLogRepository
}

// TrialBalanceServiceOption is a functional option for configuring TrialBalanceService
type TrialBalanceServiceOption func(*TrialBalanceService)

// WithAuditLogRepository sets the audit log repository
func WithAuditLogRepository(repo TrialBalanceAuditLogRepository) TrialBalanceServiceOption {
	return func(s *TrialBalanceService) {
		s.auditLogRepo = repo
	}
}

// NewTrialBalanceService creates a new TrialBalanceService
func NewTrialBalanceService(
	receivableRepo AccountReceivableRepository,
	payableRepo AccountPayableRepository,
	receiptVoucherRepo ReceiptVoucherRepository,
	paymentVoucherRepo PaymentVoucherRepository,
	creditMemoRepo CreditMemoRepository,
	debitMemoRepo DebitMemoRepository,
	opts ...TrialBalanceServiceOption,
) *TrialBalanceService {
	s := &TrialBalanceService{
		receivableRepo:     receivableRepo,
		payableRepo:        payableRepo,
		receiptVoucherRepo: receiptVoucherRepo,
		paymentVoucherRepo: paymentVoucherRepo,
		creditMemoRepo:     creditMemoRepo,
		debitMemoRepo:      debitMemoRepo,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// PerformTrialBalanceCheck performs a comprehensive trial balance check
func (s *TrialBalanceService) PerformTrialBalanceCheck(
	ctx context.Context,
	tenantID uuid.UUID,
	checkedBy uuid.UUID,
	opts TrialBalanceCheckOptions,
) (*TrialBalanceResult, error) {
	startTime := time.Now()
	result := NewTrialBalanceResult(tenantID, checkedBy)
	result.SetPeriod(opts.PeriodStart, opts.PeriodEnd)

	// Aggregate totals
	var totalDebits, totalCredits decimal.Decimal

	// Check receivables
	if opts.CheckReceivables {
		receivableResults, err := s.checkReceivables(ctx, tenantID, opts)
		if err != nil {
			return nil, err
		}
		result.TotalReceivables = receivableResults.totalOutstanding
		result.ReceivableCount = receivableResults.count
		totalDebits = totalDebits.Add(receivableResults.totalOutstanding)
		for _, d := range receivableResults.discrepancies {
			result.AddDiscrepancy(&d)
		}
	}

	// Check payables
	if opts.CheckPayables {
		payableResults, err := s.checkPayables(ctx, tenantID, opts)
		if err != nil {
			return nil, err
		}
		result.TotalPayables = payableResults.totalOutstanding
		result.PayableCount = payableResults.count
		totalCredits = totalCredits.Add(payableResults.totalOutstanding)
		for _, d := range payableResults.discrepancies {
			result.AddDiscrepancy(&d)
		}
	}

	// Check receipt vouchers
	if opts.CheckReceipts {
		receiptResults, err := s.checkReceiptVouchers(ctx, tenantID, opts)
		if err != nil {
			return nil, err
		}
		result.TotalReceipts = receiptResults.totalUnallocated
		result.ReceiptCount = receiptResults.count
		totalCredits = totalCredits.Add(receiptResults.totalUnallocated)
		for _, d := range receiptResults.discrepancies {
			result.AddDiscrepancy(&d)
		}
	}

	// Check payment vouchers
	if opts.CheckPayments {
		paymentResults, err := s.checkPaymentVouchers(ctx, tenantID, opts)
		if err != nil {
			return nil, err
		}
		result.TotalPayments = paymentResults.totalUnallocated
		result.PaymentCount = paymentResults.count
		totalDebits = totalDebits.Add(paymentResults.totalUnallocated)
		for _, d := range paymentResults.discrepancies {
			result.AddDiscrepancy(&d)
		}
	}

	// Check credit memos
	if opts.CheckCreditMemos {
		creditMemoResults, err := s.checkCreditMemos(ctx, tenantID, opts)
		if err != nil {
			return nil, err
		}
		result.TotalCreditMemos = creditMemoResults.totalRemaining
		result.CreditMemoCount = creditMemoResults.count
		totalCredits = totalCredits.Add(creditMemoResults.totalRemaining)
		for _, d := range creditMemoResults.discrepancies {
			result.AddDiscrepancy(&d)
		}
	}

	// Check debit memos
	if opts.CheckDebitMemos {
		debitMemoResults, err := s.checkDebitMemos(ctx, tenantID, opts)
		if err != nil {
			return nil, err
		}
		result.TotalDebitMemos = debitMemoResults.totalRemaining
		result.DebitMemoCount = debitMemoResults.count
		totalDebits = totalDebits.Add(debitMemoResults.totalRemaining)
		for _, d := range debitMemoResults.discrepancies {
			result.AddDiscrepancy(&d)
		}
	}

	// Set totals and calculate net balance
	result.SetTotals(totalDebits, totalCredits)

	// Calculate execution duration
	result.SetExecutionDuration(time.Since(startTime).Milliseconds())

	// Determine final status
	if result.DiscrepancyCount == 0 {
		result.Status = TrialBalanceStatusBalanced
	} else {
		result.Status = TrialBalanceStatusUnbalanced
	}

	// Save audit log if repository is available
	if s.auditLogRepo != nil {
		auditLog := NewTrialBalanceAuditLog(result)
		if err := s.auditLogRepo.Save(ctx, auditLog); err != nil {
			// For financial compliance, audit logging failures should be logged
			// but not block the operation to avoid denying users visibility into their data
			// The error is recorded in the result notes for the caller to handle appropriately
			if result.Notes != "" {
				result.Notes = fmt.Sprintf("Warning: Failed to save audit log: %v. %s", err, result.Notes)
			} else {
				result.Notes = fmt.Sprintf("Warning: Failed to save audit log: %v", err)
			}
		}
	}

	return result, nil
}

// checkResult holds the results of checking a specific entity type
type checkResult struct {
	totalOutstanding decimal.Decimal // For receivables/payables
	totalUnallocated decimal.Decimal // For vouchers
	totalRemaining   decimal.Decimal // For memos
	count            int64
	discrepancies    []BalanceDiscrepancy
}

// checkReceivables validates all receivables for internal consistency
func (s *TrialBalanceService) checkReceivables(
	ctx context.Context,
	tenantID uuid.UUID,
	opts TrialBalanceCheckOptions,
) (*checkResult, error) {
	result := &checkResult{
		discrepancies: make([]BalanceDiscrepancy, 0),
	}

	// Get total outstanding for the tenant
	total, err := s.receivableRepo.SumOutstandingForTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receivables sum: %w", err)
	}
	result.totalOutstanding = total

	// Get count
	filter := AccountReceivableFilter{}
	count, err := s.receivableRepo.CountForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get receivables count: %w", err)
	}
	result.count = count

	// Validate internal consistency if enabled
	if opts.ValidateInternalConsistency {
		receivables, err := s.receivableRepo.FindAllForTenant(ctx, tenantID, AccountReceivableFilter{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch receivables: %w", err)
		}

		for _, r := range receivables {
			// Check that paid + outstanding = total
			expectedOutstanding := r.TotalAmount.Sub(r.PaidAmount)
			if !expectedOutstanding.Equal(r.OutstandingAmount) {
				diff := expectedOutstanding.Sub(r.OutstandingAmount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						ReceivableAmountMismatch,
						"AccountReceivable",
						r.ID,
						r.ReceivableNumber,
						expectedOutstanding,
						r.OutstandingAmount,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}
		}
	}

	return result, nil
}

// checkPayables validates all payables for internal consistency
func (s *TrialBalanceService) checkPayables(
	ctx context.Context,
	tenantID uuid.UUID,
	opts TrialBalanceCheckOptions,
) (*checkResult, error) {
	result := &checkResult{
		discrepancies: make([]BalanceDiscrepancy, 0),
	}

	// Get total outstanding for the tenant
	total, err := s.payableRepo.SumOutstandingForTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payables sum: %w", err)
	}
	result.totalOutstanding = total

	// Get count
	filter := AccountPayableFilter{}
	count, err := s.payableRepo.CountForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get payables count: %w", err)
	}
	result.count = count

	// Validate internal consistency if enabled
	if opts.ValidateInternalConsistency {
		payables, err := s.payableRepo.FindAllForTenant(ctx, tenantID, AccountPayableFilter{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch payables: %w", err)
		}

		for _, p := range payables {
			// Check that paid + outstanding = total
			expectedOutstanding := p.TotalAmount.Sub(p.PaidAmount)
			if !expectedOutstanding.Equal(p.OutstandingAmount) {
				diff := expectedOutstanding.Sub(p.OutstandingAmount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						PayableAmountMismatch,
						"AccountPayable",
						p.ID,
						p.PayableNumber,
						expectedOutstanding,
						p.OutstandingAmount,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}
		}
	}

	return result, nil
}

// checkReceiptVouchers validates all receipt vouchers for allocation consistency
func (s *TrialBalanceService) checkReceiptVouchers(
	ctx context.Context,
	tenantID uuid.UUID,
	opts TrialBalanceCheckOptions,
) (*checkResult, error) {
	result := &checkResult{
		discrepancies: make([]BalanceDiscrepancy, 0),
	}

	filter := ReceiptVoucherFilter{}
	vouchers, err := s.receiptVoucherRepo.FindAllForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch receipt vouchers: %w", err)
	}

	var totalUnallocated decimal.Decimal
	for _, v := range vouchers {
		totalUnallocated = totalUnallocated.Add(v.UnallocatedAmount)
		result.count++

		// Validate voucher allocations if enabled
		if opts.ValidateVoucherAllocations {
			// Sum up allocations
			var allocSum decimal.Decimal
			for _, a := range v.Allocations {
				allocSum = allocSum.Add(a.Amount)
			}

			// Check that allocations match allocated amount
			if !allocSum.Equal(v.AllocatedAmount) {
				diff := allocSum.Sub(v.AllocatedAmount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						VoucherAllocationMismatch,
						"ReceiptVoucher",
						v.ID,
						v.VoucherNumber,
						v.AllocatedAmount,
						allocSum,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}

			// Check that allocated + unallocated = total amount
			expectedTotal := v.AllocatedAmount.Add(v.UnallocatedAmount)
			if !expectedTotal.Equal(v.Amount) {
				diff := expectedTotal.Sub(v.Amount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						VoucherAllocationMismatch,
						"ReceiptVoucher",
						v.ID,
						v.VoucherNumber,
						v.Amount,
						expectedTotal,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}
		}
	}

	result.totalUnallocated = totalUnallocated
	return result, nil
}

// checkPaymentVouchers validates all payment vouchers for allocation consistency
func (s *TrialBalanceService) checkPaymentVouchers(
	ctx context.Context,
	tenantID uuid.UUID,
	opts TrialBalanceCheckOptions,
) (*checkResult, error) {
	result := &checkResult{
		discrepancies: make([]BalanceDiscrepancy, 0),
	}

	filter := PaymentVoucherFilter{}
	vouchers, err := s.paymentVoucherRepo.FindAllForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment vouchers: %w", err)
	}

	var totalUnallocated decimal.Decimal
	for _, v := range vouchers {
		totalUnallocated = totalUnallocated.Add(v.UnallocatedAmount)
		result.count++

		// Validate voucher allocations if enabled
		if opts.ValidateVoucherAllocations {
			// Sum up allocations
			var allocSum decimal.Decimal
			for _, a := range v.Allocations {
				allocSum = allocSum.Add(a.Amount)
			}

			// Check that allocations match allocated amount
			if !allocSum.Equal(v.AllocatedAmount) {
				diff := allocSum.Sub(v.AllocatedAmount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						VoucherAllocationMismatch,
						"PaymentVoucher",
						v.ID,
						v.VoucherNumber,
						v.AllocatedAmount,
						allocSum,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}

			// Check that allocated + unallocated = total amount
			expectedTotal := v.AllocatedAmount.Add(v.UnallocatedAmount)
			if !expectedTotal.Equal(v.Amount) {
				diff := expectedTotal.Sub(v.Amount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						VoucherAllocationMismatch,
						"PaymentVoucher",
						v.ID,
						v.VoucherNumber,
						v.Amount,
						expectedTotal,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}
		}
	}

	result.totalUnallocated = totalUnallocated
	return result, nil
}

// checkCreditMemos validates all credit memos for application consistency
func (s *TrialBalanceService) checkCreditMemos(
	ctx context.Context,
	tenantID uuid.UUID,
	opts TrialBalanceCheckOptions,
) (*checkResult, error) {
	result := &checkResult{
		discrepancies: make([]BalanceDiscrepancy, 0),
	}

	// Get total remaining credit for the tenant
	total, err := s.creditMemoRepo.SumRemainingForTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credit memos sum: %w", err)
	}
	result.totalRemaining = total

	// Get count
	filter := CreditMemoFilter{}
	count, err := s.creditMemoRepo.CountForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get credit memos count: %w", err)
	}
	result.count = count

	// Validate memo applications if enabled
	if opts.ValidateMemoApplications {
		memos, err := s.creditMemoRepo.FindAllForTenant(ctx, tenantID, CreditMemoFilter{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch credit memos: %w", err)
		}

		for _, m := range memos {
			// Check that applied + remaining = total
			expectedRemaining := m.TotalCredit.Sub(m.AppliedAmount)
			if !expectedRemaining.Equal(m.RemainingAmount) {
				diff := expectedRemaining.Sub(m.RemainingAmount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						CreditMemoImbalance,
						"CreditMemo",
						m.ID,
						m.MemoNumber,
						expectedRemaining,
						m.RemainingAmount,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}

			// Check that applied doesn't exceed total
			if m.AppliedAmount.GreaterThan(m.TotalCredit) {
				d := NewBalanceDiscrepancy(
					CreditMemoImbalance,
					"CreditMemo",
					m.ID,
					m.MemoNumber,
					m.TotalCredit,
					m.AppliedAmount,
				)
				result.discrepancies = append(result.discrepancies, *d)
			}
		}
	}

	return result, nil
}

// checkDebitMemos validates all debit memos for application consistency
func (s *TrialBalanceService) checkDebitMemos(
	ctx context.Context,
	tenantID uuid.UUID,
	opts TrialBalanceCheckOptions,
) (*checkResult, error) {
	result := &checkResult{
		discrepancies: make([]BalanceDiscrepancy, 0),
	}

	// Get total remaining debit for the tenant
	total, err := s.debitMemoRepo.SumRemainingForTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get debit memos sum: %w", err)
	}
	result.totalRemaining = total

	// Get count
	filter := DebitMemoFilter{}
	count, err := s.debitMemoRepo.CountForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get debit memos count: %w", err)
	}
	result.count = count

	// Validate memo applications if enabled
	if opts.ValidateMemoApplications {
		memos, err := s.debitMemoRepo.FindAllForTenant(ctx, tenantID, DebitMemoFilter{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch debit memos: %w", err)
		}

		for _, m := range memos {
			// Check that applied + remaining = total
			expectedRemaining := m.TotalDebit.Sub(m.AppliedAmount)
			if !expectedRemaining.Equal(m.RemainingAmount) {
				diff := expectedRemaining.Sub(m.RemainingAmount).Abs()
				if diff.GreaterThan(opts.Tolerance) {
					d := NewBalanceDiscrepancy(
						DebitMemoImbalance,
						"DebitMemo",
						m.ID,
						m.MemoNumber,
						expectedRemaining,
						m.RemainingAmount,
					)
					result.discrepancies = append(result.discrepancies, *d)
				}
			}

			// Check that applied doesn't exceed total
			if m.AppliedAmount.GreaterThan(m.TotalDebit) {
				d := NewBalanceDiscrepancy(
					DebitMemoImbalance,
					"DebitMemo",
					m.ID,
					m.MemoNumber,
					m.TotalDebit,
					m.AppliedAmount,
				)
				result.discrepancies = append(result.discrepancies, *d)
			}
		}
	}

	return result, nil
}

// CheckBalanceBeforeOperation performs a quick balance check before a financial operation
// This is the guard that prevents operations when balance is imbalanced
func (s *TrialBalanceService) CheckBalanceBeforeOperation(
	ctx context.Context,
	tenantID uuid.UUID,
) (*BalanceCheckGuardResult, error) {
	// Perform a quick check with minimal validation
	opts := TrialBalanceCheckOptions{
		CheckReceivables:            true,
		CheckPayables:               true,
		CheckReceipts:               true,
		CheckPayments:               true,
		CheckCreditMemos:            true,
		CheckDebitMemos:             true,
		ValidateInternalConsistency: false, // Skip for speed
		ValidateVoucherAllocations:  false, // Skip for speed
		ValidateMemoApplications:    false, // Skip for speed
		Tolerance:                   decimal.NewFromFloat(0.01),
	}

	result, err := s.PerformTrialBalanceCheck(ctx, tenantID, uuid.Nil, opts)
	if err != nil {
		return nil, err
	}

	if result.IsBalanced() {
		return NewAllowedGuardResult(), nil
	}

	return NewBlockedGuardResult(result.Discrepancies), nil
}

// EnforceBalanceCheck is a domain service method that checks balance and returns an error if unbalanced
// Use this to wrap financial operations that require balance verification
func (s *TrialBalanceService) EnforceBalanceCheck(
	ctx context.Context,
	tenantID uuid.UUID,
) error {
	guardResult, err := s.CheckBalanceBeforeOperation(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check balance: %w", err)
	}

	if !guardResult.Allowed {
		return shared.NewDomainError(
			"TRIAL_BALANCE_IMBALANCED",
			fmt.Sprintf("Cannot perform operation: %s", guardResult.Message),
		)
	}

	return nil
}

// GetLatestAuditLogs retrieves the most recent trial balance audit logs
func (s *TrialBalanceService) GetLatestAuditLogs(
	ctx context.Context,
	tenantID uuid.UUID,
	limit int,
) ([]TrialBalanceAuditLog, error) {
	if s.auditLogRepo == nil {
		return nil, shared.NewDomainError("NO_AUDIT_REPO", "Audit log repository not configured")
	}

	return s.auditLogRepo.FindLatestForTenant(ctx, tenantID, limit)
}

// TrialBalanceAuditLogRepository defines the interface for trial balance audit log persistence
type TrialBalanceAuditLogRepository interface {
	// Save persists an audit log entry
	Save(ctx context.Context, log *TrialBalanceAuditLog) error

	// FindByID finds an audit log by ID
	FindByID(ctx context.Context, id uuid.UUID) (*TrialBalanceAuditLog, error)

	// FindLatestForTenant finds the most recent audit logs for a tenant
	FindLatestForTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]TrialBalanceAuditLog, error)

	// FindByPeriod finds audit logs within a date range
	FindByPeriod(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]TrialBalanceAuditLog, error)

	// FindByStatus finds audit logs by status
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status TrialBalanceStatus) ([]TrialBalanceAuditLog, error)

	// CountForTenant counts all audit logs for a tenant
	CountForTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
}
