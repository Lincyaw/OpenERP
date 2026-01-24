package partner

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BalanceTransactionService handles balance transaction operations
type BalanceTransactionService struct {
	balanceTxRepo partner.BalanceTransactionRepository
	customerRepo  partner.CustomerRepository
}

// NewBalanceTransactionService creates a new BalanceTransactionService
func NewBalanceTransactionService(
	balanceTxRepo partner.BalanceTransactionRepository,
	customerRepo partner.CustomerRepository,
) *BalanceTransactionService {
	return &BalanceTransactionService{
		balanceTxRepo: balanceTxRepo,
		customerRepo:  customerRepo,
	}
}

// Recharge adds balance to a customer with transaction record
func (s *BalanceTransactionService) Recharge(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
	reference, remark string,
	operatorID *uuid.UUID,
) (*BalanceTransactionResponse, error) {
	// Get customer
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	balanceBefore := customer.Balance

	// Add balance to customer
	if err := customer.AddBalance(amount); err != nil {
		return nil, err
	}

	// Create transaction record
	transaction, err := partner.CreateRechargeTransaction(
		tenantID,
		customerID,
		amount,
		balanceBefore,
		partner.BalanceSourceTypeManual,
	)
	if err != nil {
		return nil, err
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

	// Save customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	// Save transaction
	if err := s.balanceTxRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	response := ToBalanceTransactionResponse(transaction)
	return &response, nil
}

// Consume deducts balance from a customer with transaction record
func (s *BalanceTransactionService) Consume(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
	sourceType partner.BalanceTransactionSourceType,
	sourceID *string,
	reference, remark string,
	operatorID *uuid.UUID,
) (*BalanceTransactionResponse, error) {
	// Get customer
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	balanceBefore := customer.Balance

	// Deduct balance from customer
	if err := customer.DeductBalance(amount); err != nil {
		return nil, err
	}

	// Create transaction record
	transaction, err := partner.CreateConsumeTransaction(
		tenantID,
		customerID,
		amount,
		balanceBefore,
		sourceType,
	)
	if err != nil {
		return nil, err
	}

	if sourceID != nil {
		transaction.WithSourceID(*sourceID)
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

	// Save customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	// Save transaction
	if err := s.balanceTxRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	response := ToBalanceTransactionResponse(transaction)
	return &response, nil
}

// Refund adds balance back to customer with transaction record
func (s *BalanceTransactionService) Refund(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
	sourceType partner.BalanceTransactionSourceType,
	sourceID *string,
	reference, remark string,
	operatorID *uuid.UUID,
) (*BalanceTransactionResponse, error) {
	// Get customer
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	balanceBefore := customer.Balance

	// Refund balance to customer
	if err := customer.RefundBalance(amount); err != nil {
		return nil, err
	}

	// Create transaction record
	transaction, err := partner.CreateRefundTransaction(
		tenantID,
		customerID,
		amount,
		balanceBefore,
		sourceType,
	)
	if err != nil {
		return nil, err
	}

	if sourceID != nil {
		transaction.WithSourceID(*sourceID)
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

	// Save customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	// Save transaction
	if err := s.balanceTxRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	response := ToBalanceTransactionResponse(transaction)
	return &response, nil
}

// Adjust adjusts customer balance with transaction record (for manual corrections)
func (s *BalanceTransactionService) Adjust(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	amount decimal.Decimal,
	isIncrease bool,
	reference, remark string,
	operatorID *uuid.UUID,
) (*BalanceTransactionResponse, error) {
	// Get customer
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	balanceBefore := customer.Balance

	// Adjust customer balance
	if isIncrease {
		if err := customer.AddBalance(amount); err != nil {
			return nil, err
		}
	} else {
		if err := customer.DeductBalance(amount); err != nil {
			return nil, err
		}
	}

	// Create transaction record
	transaction, err := partner.CreateAdjustmentTransaction(
		tenantID,
		customerID,
		amount,
		isIncrease,
		balanceBefore,
	)
	if err != nil {
		return nil, err
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

	// Save customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	// Save transaction
	if err := s.balanceTxRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	response := ToBalanceTransactionResponse(transaction)
	return &response, nil
}

// GetByID retrieves a balance transaction by ID
func (s *BalanceTransactionService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*BalanceTransactionResponse, error) {
	transaction, err := s.balanceTxRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	response := ToBalanceTransactionResponse(transaction)
	return &response, nil
}

// ListByCustomer retrieves balance transactions for a customer
func (s *BalanceTransactionService) ListByCustomer(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
	filter BalanceTransactionListFilter,
) ([]BalanceTransactionResponse, int64, error) {
	// Verify customer exists
	_, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, 0, err
	}

	// Build domain filter
	domainFilter := partner.BalanceTransactionFilter{
		CustomerID: &customerID,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}

	if filter.TransactionType != "" {
		txType := partner.BalanceTransactionType(filter.TransactionType)
		domainFilter.TransactionType = &txType
	}

	if filter.SourceType != "" {
		srcType := partner.BalanceTransactionSourceType(filter.SourceType)
		domainFilter.SourceType = &srcType
	}

	if filter.DateFrom != "" {
		if t, err := time.Parse("2006-01-02", filter.DateFrom); err == nil {
			domainFilter.DateFrom = &t
		}
	}

	if filter.DateTo != "" {
		if t, err := time.Parse("2006-01-02", filter.DateTo); err == nil {
			// Add 1 day to include the end date
			t = t.Add(24 * time.Hour)
			domainFilter.DateTo = &t
		}
	}

	// Set defaults
	if domainFilter.Page <= 0 {
		domainFilter.Page = 1
	}
	if domainFilter.PageSize <= 0 {
		domainFilter.PageSize = 20
	}

	transactions, total, err := s.balanceTxRepo.FindByCustomerID(ctx, tenantID, customerID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToBalanceTransactionResponses(transactions), total, nil
}

// GetBalanceSummary retrieves balance summary for a customer
func (s *BalanceTransactionService) GetBalanceSummary(
	ctx context.Context,
	tenantID, customerID uuid.UUID,
) (*BalanceSummaryResponse, error) {
	// Get customer
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	// Get all-time sums (using a very old date as start)
	startDate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Now().Add(24 * time.Hour)

	totalRecharge, err := s.balanceTxRepo.SumByCustomerIDAndType(
		ctx, tenantID, customerID,
		partner.BalanceTransactionTypeRecharge,
		startDate, endDate,
	)
	if err != nil {
		return nil, err
	}

	totalConsume, err := s.balanceTxRepo.SumByCustomerIDAndType(
		ctx, tenantID, customerID,
		partner.BalanceTransactionTypeConsume,
		startDate, endDate,
	)
	if err != nil {
		return nil, err
	}

	totalRefund, err := s.balanceTxRepo.SumByCustomerIDAndType(
		ctx, tenantID, customerID,
		partner.BalanceTransactionTypeRefund,
		startDate, endDate,
	)
	if err != nil {
		return nil, err
	}

	return &BalanceSummaryResponse{
		CustomerID:     customerID,
		CurrentBalance: customer.Balance,
		TotalRecharge:  decimal.NewFromFloat(totalRecharge),
		TotalConsume:   decimal.NewFromFloat(totalConsume),
		TotalRefund:    decimal.NewFromFloat(totalRefund),
	}, nil
}

// GetBalance retrieves current balance for a customer
func (s *BalanceTransactionService) GetBalance(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return decimal.Zero, err
	}

	return customer.Balance, nil
}

// HasSufficientBalance checks if customer has sufficient balance
func (s *BalanceTransactionService) HasSufficientBalance(ctx context.Context, tenantID, customerID uuid.UUID, amount decimal.Decimal) (bool, error) {
	balance, err := s.GetBalance(ctx, tenantID, customerID)
	if err != nil {
		if err == shared.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	return balance.GreaterThanOrEqual(amount), nil
}
