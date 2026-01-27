package finance

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BalancePaymentService handles payment processing using customer balance
type BalancePaymentService struct {
	customerRepo  partner.CustomerRepository
	balanceTxRepo partner.BalanceTransactionRepository
}

// NewBalancePaymentService creates a new BalancePaymentService
func NewBalancePaymentService(
	customerRepo partner.CustomerRepository,
	balanceTxRepo partner.BalanceTransactionRepository,
) *BalancePaymentService {
	return &BalancePaymentService{
		customerRepo:  customerRepo,
		balanceTxRepo: balanceTxRepo,
	}
}

// BalancePaymentRequest represents a request to process a balance payment
type BalancePaymentRequest struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	Amount     decimal.Decimal
	SourceType partner.BalanceTransactionSourceType
	SourceID   string // The voucher number or ID
	Reference  string // Optional reference
	Remark     string // Optional remark
	OperatorID *uuid.UUID
}

// BalancePaymentResult represents the result of a balance payment
type BalancePaymentResult struct {
	TransactionID uuid.UUID       `json:"transaction_id"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	Amount        decimal.Decimal `json:"amount"`
	BalanceBefore decimal.Decimal `json:"balance_before"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
	Success       bool            `json:"success"`
}

// ProcessBalancePayment deducts from customer balance and creates a transaction record
// This is the core method for balance payment functionality
func (s *BalancePaymentService) ProcessBalancePayment(
	ctx context.Context,
	req BalancePaymentRequest,
) (*BalancePaymentResult, error) {
	// Start tracing span for balance payment processing
	ctx, span := telemetry.StartServiceSpan(ctx, "payment", "process_balance_payment")
	defer span.End()

	telemetry.SetAttributes(span,
		telemetry.SpanAttrCustomerID, req.CustomerID.String(),
		telemetry.SpanAttrAmount, req.Amount.String(),
		telemetry.SpanAttrSourceType, string(req.SourceType),
		telemetry.SpanAttrSourceID, req.SourceID,
	)

	// Wrap in profiling labels for performance analysis
	var result *BalancePaymentResult
	var operationErr error
	telemetry.WithProfilingLabels(ctx, telemetry.FinanceOperationLabels(telemetry.OperationProcessPayment, "balance"), func(c context.Context) {
		// Validate amount
		if req.Amount.IsNegative() || req.Amount.IsZero() {
			err := shared.NewDomainError("INVALID_AMOUNT", "Payment amount must be positive")
			telemetry.RecordError(span, err)
			operationErr = err
			return
		}

		// Get customer
		customer, err := s.customerRepo.FindByIDForTenant(c, req.TenantID, req.CustomerID)
		if err != nil {
			telemetry.RecordError(span, err)
			operationErr = fmt.Errorf("failed to get customer: %w", err)
			return
		}
		if customer == nil {
			err := shared.NewDomainError("CUSTOMER_NOT_FOUND", "Customer not found")
			telemetry.RecordError(span, err)
			operationErr = err
			return
		}

		// Check sufficient balance
		if customer.Balance.LessThan(req.Amount) {
			err := shared.NewDomainError(
				"INSUFFICIENT_BALANCE",
				fmt.Sprintf("Insufficient balance: available %.2f, required %.2f",
					customer.Balance.InexactFloat64(), req.Amount.InexactFloat64()),
			)
			telemetry.RecordError(span, err)
			operationErr = err
			return
		}

		balanceBefore := customer.Balance
		telemetry.SetAttribute(span, "balance_before", balanceBefore.String())

		// Deduct balance from customer
		if err := customer.DeductBalance(req.Amount); err != nil {
			telemetry.RecordError(span, err)
			operationErr = fmt.Errorf("failed to deduct balance: %w", err)
			return
		}

		// Create balance transaction record
		transaction, err := partner.CreateConsumeTransaction(
			req.TenantID,
			req.CustomerID,
			req.Amount,
			balanceBefore,
			req.SourceType,
		)
		if err != nil {
			telemetry.RecordError(span, err)
			operationErr = fmt.Errorf("failed to create transaction: %w", err)
			return
		}

		// Set optional fields
		if req.SourceID != "" {
			transaction.WithSourceID(req.SourceID)
		}
		if req.Reference != "" {
			transaction.WithReference(req.Reference)
		}
		if req.Remark != "" {
			transaction.WithRemark(req.Remark)
		}
		if req.OperatorID != nil {
			transaction.WithOperatorID(*req.OperatorID)
		}

		// Save customer (with updated balance) using optimistic locking
		// This prevents concurrent payments from causing balance overdraft
		if err := s.customerRepo.SaveWithLock(c, customer); err != nil {
			telemetry.RecordError(span, err)
			operationErr = fmt.Errorf("failed to save customer: %w", err)
			return
		}

		// Save balance transaction
		if err := s.balanceTxRepo.Create(c, transaction); err != nil {
			telemetry.RecordError(span, err)
			operationErr = fmt.Errorf("failed to save transaction: %w", err)
			return
		}

		// Add success event to span
		telemetry.AddEvent(span, "balance_payment_completed",
			"transaction_id", transaction.ID.String(),
			"balance_after", customer.Balance.String(),
		)
		telemetry.SetAttribute(span, "balance_after", customer.Balance.String())

		result = &BalancePaymentResult{
			TransactionID: transaction.ID,
			CustomerID:    req.CustomerID,
			Amount:        req.Amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  customer.Balance,
			Success:       true,
		}
	})

	return result, operationErr
}

// ValidateBalancePayment validates if a balance payment can be processed
// without actually processing it
func (s *BalancePaymentService) ValidateBalancePayment(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
) error {
	if amount.IsNegative() || amount.IsZero() {
		return shared.NewDomainError("INVALID_AMOUNT", "Payment amount must be positive")
	}

	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return shared.NewDomainError("CUSTOMER_NOT_FOUND", "Customer not found")
	}

	if customer.Balance.LessThan(amount) {
		return shared.NewDomainError(
			"INSUFFICIENT_BALANCE",
			fmt.Sprintf("Insufficient balance: available %.2f, required %.2f",
				customer.Balance.InexactFloat64(), amount.InexactFloat64()),
		)
	}

	return nil
}

// GetCustomerBalance returns the current balance for a customer
func (s *BalancePaymentService) GetCustomerBalance(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
) (decimal.Decimal, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return decimal.Zero, shared.NewDomainError("CUSTOMER_NOT_FOUND", "Customer not found")
	}

	return customer.Balance, nil
}

// HasSufficientBalance checks if a customer has sufficient balance for a payment
func (s *BalancePaymentService) HasSufficientBalance(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
) (bool, error) {
	balance, err := s.GetCustomerBalance(ctx, tenantID, customerID)
	if err != nil {
		return false, err
	}
	return balance.GreaterThanOrEqual(amount), nil
}

// ProcessReceiptVoucherBalancePayment processes a balance payment for a receipt voucher
// This is a convenience method specifically for receipt voucher scenarios
func (s *BalancePaymentService) ProcessReceiptVoucherBalancePayment(
	ctx context.Context,
	tenantID uuid.UUID,
	voucher *finance.ReceiptVoucher,
	operatorID *uuid.UUID,
) (*BalancePaymentResult, error) {
	if voucher == nil {
		return nil, shared.NewDomainError("INVALID_VOUCHER", "Receipt voucher cannot be nil")
	}

	// Only process if payment method is BALANCE
	if voucher.PaymentMethod != finance.PaymentMethodBalance {
		return nil, shared.NewDomainError(
			"INVALID_PAYMENT_METHOD",
			"Payment method must be BALANCE for balance payment processing",
		)
	}

	return s.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: voucher.CustomerID,
		Amount:     voucher.Amount,
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
		SourceID:   voucher.ID.String(),
		Reference:  voucher.VoucherNumber,
		Remark:     fmt.Sprintf("Balance payment for receipt voucher %s", voucher.VoucherNumber),
		OperatorID: operatorID,
	})
}

// RefundBalancePayment refunds a balance payment back to customer
// Used when a receipt voucher is cancelled after balance was deducted
func (s *BalancePaymentService) RefundBalancePayment(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
	sourceID, reference, remark string,
	operatorID *uuid.UUID,
) (*BalancePaymentResult, error) {
	if amount.IsNegative() || amount.IsZero() {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Refund amount must be positive")
	}

	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if customer == nil {
		return nil, shared.NewDomainError("CUSTOMER_NOT_FOUND", "Customer not found")
	}

	balanceBefore := customer.Balance

	// Refund balance to customer
	if err := customer.RefundBalance(amount); err != nil {
		return nil, fmt.Errorf("failed to refund balance: %w", err)
	}

	// Create refund transaction record
	transaction, err := partner.CreateRefundTransaction(
		tenantID,
		customerID,
		amount,
		balanceBefore,
		partner.BalanceSourceTypeReceiptVoucher,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund transaction: %w", err)
	}

	if sourceID != "" {
		transaction.WithSourceID(sourceID)
	}
	if reference != "" {
		transaction.WithReference(reference)
	}
	if remark != "" {
		transaction.WithRemark(remark)
	}
	if operatorID != nil {
		transaction.WithOperatorID(*operatorID)
	}

	// Save customer using optimistic locking to prevent concurrent modification issues
	if err := s.customerRepo.SaveWithLock(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	// Save transaction
	if err := s.balanceTxRepo.Create(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	return &BalancePaymentResult{
		TransactionID: transaction.ID,
		CustomerID:    customerID,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  customer.Balance,
		Success:       true,
	}, nil
}
