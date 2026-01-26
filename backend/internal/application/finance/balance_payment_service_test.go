package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Repositories for Balance Payment Service
// =============================================================================

// MockCustomerRepositoryForBalance is a mock implementation for balance payment tests
type MockCustomerRepositoryForBalance struct {
	mock.Mock
}

func (m *MockCustomerRepositoryForBalance) FindByID(ctx context.Context, id uuid.UUID) (*partner.Customer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByType(ctx context.Context, tenantID uuid.UUID, customerType partner.CustomerType, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, customerType, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByLevel(ctx context.Context, tenantID uuid.UUID, level partner.CustomerLevel, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, level, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.CustomerStatus, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) FindWithPositiveBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) Save(ctx context.Context, customer *partner.Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockCustomerRepositoryForBalance) SaveWithLock(ctx context.Context, customer *partner.Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockCustomerRepositoryForBalance) SaveBatch(ctx context.Context, customers []*partner.Customer) error {
	args := m.Called(ctx, customers)
	return args.Error(0)
}

func (m *MockCustomerRepositoryForBalance) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCustomerRepositoryForBalance) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockCustomerRepositoryForBalance) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) CountByType(ctx context.Context, tenantID uuid.UUID, customerType partner.CustomerType) (int64, error) {
	args := m.Called(ctx, tenantID, customerType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) CountByLevel(ctx context.Context, tenantID uuid.UUID, level partner.CustomerLevel) (int64, error) {
	args := m.Called(ctx, tenantID, level)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.CustomerStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error) {
	args := m.Called(ctx, tenantID, phone)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	args := m.Called(ctx, tenantID, email)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockCustomerRepositoryForBalance) GenerateCode(ctx context.Context, tenantID uuid.UUID, prefix string) (string, error) {
	args := m.Called(ctx, tenantID, prefix)
	return args.Get(0).(string), args.Error(1)
}

// MockBalanceTransactionRepository is a mock implementation of BalanceTransactionRepository
type MockBalanceTransactionRepository struct {
	mock.Mock
}

func (m *MockBalanceTransactionRepository) Create(ctx context.Context, transaction *partner.BalanceTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockBalanceTransactionRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*partner.BalanceTransaction, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.BalanceTransaction), args.Error(1)
}

func (m *MockBalanceTransactionRepository) FindByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID, filter partner.BalanceTransactionFilter) ([]*partner.BalanceTransaction, int64, error) {
	args := m.Called(ctx, tenantID, customerID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*partner.BalanceTransaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockBalanceTransactionRepository) FindBySourceID(ctx context.Context, tenantID uuid.UUID, sourceType partner.BalanceTransactionSourceType, sourceID string) ([]*partner.BalanceTransaction, error) {
	args := m.Called(ctx, tenantID, sourceType, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*partner.BalanceTransaction), args.Error(1)
}

func (m *MockBalanceTransactionRepository) List(ctx context.Context, tenantID uuid.UUID, filter partner.BalanceTransactionFilter) ([]*partner.BalanceTransaction, int64, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*partner.BalanceTransaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockBalanceTransactionRepository) GetLatestByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID) (*partner.BalanceTransaction, error) {
	args := m.Called(ctx, tenantID, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.BalanceTransaction), args.Error(1)
}

func (m *MockBalanceTransactionRepository) SumByCustomerIDAndType(ctx context.Context, tenantID, customerID uuid.UUID, txType partner.BalanceTransactionType, from, to time.Time) (float64, error) {
	args := m.Called(ctx, tenantID, customerID, txType, from, to)
	return args.Get(0).(float64), args.Error(1)
}

// =============================================================================
// Test Helper Functions
// =============================================================================

func createTestCustomerWithBalance(tenantID uuid.UUID, balance decimal.Decimal) *partner.Customer {
	customer, _ := partner.NewCustomer(tenantID, "CUST-001", "Test Customer", partner.CustomerTypeIndividual)
	customer.Balance = balance
	return customer
}

func createTestReceiptVoucherForBalance(tenantID, customerID uuid.UUID, amount decimal.Decimal, paymentMethod finance.PaymentMethod) *finance.ReceiptVoucher {
	money := valueobject.NewMoneyCNY(amount)
	voucher, _ := finance.NewReceiptVoucher(
		tenantID,
		"RCV-001",
		customerID,
		"Test Customer",
		money,
		paymentMethod,
		time.Now(),
	)
	return voucher
}

// =============================================================================
// Test Cases for ProcessBalancePayment
// =============================================================================

func TestBalancePaymentService_ProcessBalancePayment_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Create customer with balance of 1000
	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(1000.00))
	customer.ID = customerID

	// Mock expectations
	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
	balanceTxRepo.On("Create", ctx, mock.AnythingOfType("*partner.BalanceTransaction")).Return(nil)

	// Execute
	result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     decimal.NewFromFloat(200.00),
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
		SourceID:   "RCV-001",
		Reference:  "REF-001",
		Remark:     "Test payment",
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, customerID, result.CustomerID)
	assert.Equal(t, decimal.NewFromFloat(200.00).String(), result.Amount.String())
	assert.Equal(t, decimal.NewFromFloat(1000.00).String(), result.BalanceBefore.String())
	assert.Equal(t, decimal.NewFromFloat(800.00).String(), result.BalanceAfter.String())

	customerRepo.AssertExpectations(t)
	balanceTxRepo.AssertExpectations(t)
}

func TestBalancePaymentService_ProcessBalancePayment_InsufficientBalance(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Create customer with balance of 100
	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(100.00))
	customer.ID = customerID

	// Mock expectations
	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)

	// Execute - trying to pay 200 with only 100 balance
	result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     decimal.NewFromFloat(200.00),
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
		SourceID:   "RCV-001",
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Insufficient balance")

	customerRepo.AssertExpectations(t)
}

func TestBalancePaymentService_ProcessBalancePayment_CustomerNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Mock expectations - customer not found
	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(nil, nil)

	// Execute
	result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     decimal.NewFromFloat(200.00),
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Customer not found")

	customerRepo.AssertExpectations(t)
}

func TestBalancePaymentService_ProcessBalancePayment_InvalidAmount(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	testCases := []struct {
		name   string
		amount decimal.Decimal
	}{
		{"zero amount", decimal.Zero},
		{"negative amount", decimal.NewFromFloat(-100.00)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
				TenantID:   tenantID,
				CustomerID: customerID,
				Amount:     tc.amount,
				SourceType: partner.BalanceSourceTypeReceiptVoucher,
			})

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "Payment amount must be positive")
		})
	}
}

func TestBalancePaymentService_ProcessBalancePayment_SaveCustomerFails(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(1000.00))
	customer.ID = customerID

	// Mock expectations
	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(errors.New("database error"))

	// Execute
	result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     decimal.NewFromFloat(200.00),
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
	})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save customer")

	customerRepo.AssertExpectations(t)
}

// =============================================================================
// Test Cases for ValidateBalancePayment
// =============================================================================

func TestBalancePaymentService_ValidateBalancePayment_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(500.00))
	customer.ID = customerID

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)

	// Validate for 300 when customer has 500
	err := service.ValidateBalancePayment(ctx, tenantID, customerID, decimal.NewFromFloat(300.00))

	assert.NoError(t, err)
	customerRepo.AssertExpectations(t)
}

func TestBalancePaymentService_ValidateBalancePayment_InsufficientBalance(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(100.00))
	customer.ID = customerID

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)

	// Validate for 300 when customer has only 100
	err := service.ValidateBalancePayment(ctx, tenantID, customerID, decimal.NewFromFloat(300.00))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Insufficient balance")
	customerRepo.AssertExpectations(t)
}

// =============================================================================
// Test Cases for GetCustomerBalance
// =============================================================================

func TestBalancePaymentService_GetCustomerBalance_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(750.50))
	customer.ID = customerID

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)

	balance, err := service.GetCustomerBalance(ctx, tenantID, customerID)

	assert.NoError(t, err)
	assert.Equal(t, decimal.NewFromFloat(750.50).String(), balance.String())
	customerRepo.AssertExpectations(t)
}

// =============================================================================
// Test Cases for HasSufficientBalance
// =============================================================================

func TestBalancePaymentService_HasSufficientBalance(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(500.00))
	customer.ID = customerID

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)

	testCases := []struct {
		amount   decimal.Decimal
		expected bool
	}{
		{decimal.NewFromFloat(300.00), true},  // 300 < 500
		{decimal.NewFromFloat(500.00), true},  // 500 == 500
		{decimal.NewFromFloat(600.00), false}, // 600 > 500
	}

	for _, tc := range testCases {
		result, err := service.HasSufficientBalance(ctx, tenantID, customerID, tc.amount)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, result, "amount: %s", tc.amount.String())
	}
}

// =============================================================================
// Test Cases for ProcessReceiptVoucherBalancePayment
// =============================================================================

func TestBalancePaymentService_ProcessReceiptVoucherBalancePayment_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()
	operatorID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(1000.00))
	customer.ID = customerID

	voucher := createTestReceiptVoucherForBalance(tenantID, customerID, decimal.NewFromFloat(300.00), finance.PaymentMethodBalance)

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
	balanceTxRepo.On("Create", ctx, mock.AnythingOfType("*partner.BalanceTransaction")).Return(nil)

	result, err := service.ProcessReceiptVoucherBalancePayment(ctx, tenantID, voucher, &operatorID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, decimal.NewFromFloat(300.00).String(), result.Amount.String())
	assert.Equal(t, decimal.NewFromFloat(1000.00).String(), result.BalanceBefore.String())
	assert.Equal(t, decimal.NewFromFloat(700.00).String(), result.BalanceAfter.String())

	customerRepo.AssertExpectations(t)
	balanceTxRepo.AssertExpectations(t)
}

func TestBalancePaymentService_ProcessReceiptVoucherBalancePayment_WrongPaymentMethod(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Create voucher with CASH payment method instead of BALANCE
	voucher := createTestReceiptVoucherForBalance(tenantID, customerID, decimal.NewFromFloat(300.00), finance.PaymentMethodCash)

	result, err := service.ProcessReceiptVoucherBalancePayment(ctx, tenantID, voucher, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Payment method must be BALANCE")
}

func TestBalancePaymentService_ProcessReceiptVoucherBalancePayment_NilVoucher(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	result, err := service.ProcessReceiptVoucherBalancePayment(ctx, tenantID, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Receipt voucher cannot be nil")
}

// =============================================================================
// Test Cases for RefundBalancePayment
// =============================================================================

func TestBalancePaymentService_RefundBalancePayment_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()
	operatorID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Customer has 500 balance, refunding 200
	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(500.00))
	customer.ID = customerID

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
	balanceTxRepo.On("Create", ctx, mock.AnythingOfType("*partner.BalanceTransaction")).Return(nil)

	result, err := service.RefundBalancePayment(
		ctx,
		tenantID,
		customerID,
		decimal.NewFromFloat(200.00),
		"RCV-001",
		"REF-001",
		"Refund for cancelled voucher",
		&operatorID,
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, decimal.NewFromFloat(200.00).String(), result.Amount.String())
	assert.Equal(t, decimal.NewFromFloat(500.00).String(), result.BalanceBefore.String())
	assert.Equal(t, decimal.NewFromFloat(700.00).String(), result.BalanceAfter.String())

	customerRepo.AssertExpectations(t)
	balanceTxRepo.AssertExpectations(t)
}

func TestBalancePaymentService_RefundBalancePayment_InvalidAmount(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	result, err := service.RefundBalancePayment(
		ctx,
		tenantID,
		customerID,
		decimal.Zero,
		"RCV-001",
		"REF-001",
		"Test",
		nil,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Refund amount must be positive")
}

// =============================================================================
// Test Cases for Edge Cases
// =============================================================================

func TestBalancePaymentService_ProcessBalancePayment_ExactBalance(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Customer balance exactly equals payment amount
	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(500.00))
	customer.ID = customerID

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
	balanceTxRepo.On("Create", ctx, mock.AnythingOfType("*partner.BalanceTransaction")).Return(nil)

	result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     decimal.NewFromFloat(500.00),
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.True(t, result.BalanceAfter.IsZero())

	customerRepo.AssertExpectations(t)
	balanceTxRepo.AssertExpectations(t)
}

func TestBalancePaymentService_ProcessBalancePayment_SmallDecimalAmount(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Test with small decimal amounts
	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(100.0025))
	customer.ID = customerID

	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
	balanceTxRepo.On("Create", ctx, mock.AnythingOfType("*partner.BalanceTransaction")).Return(nil)

	result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     decimal.NewFromFloat(0.0001),
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	// 100.0025 - 0.0001 = 100.0024
	assert.Equal(t, decimal.NewFromFloat(100.0024).String(), result.BalanceAfter.String())

	customerRepo.AssertExpectations(t)
	balanceTxRepo.AssertExpectations(t)
}

// =============================================================================
// Test Cases for Optimistic Locking
// =============================================================================

func TestBalancePaymentService_ProcessBalancePayment_OptimisticLockError(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Create customer with balance
	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(1000.00))
	customer.ID = customerID

	// Mock expectations - SaveWithLock fails due to concurrent modification
	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(
		shared.NewDomainError("OPTIMISTIC_LOCK_ERROR", "The customer record has been modified by another transaction"),
	)

	// Execute
	result, err := service.ProcessBalancePayment(ctx, BalancePaymentRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		Amount:     decimal.NewFromFloat(200.00),
		SourceType: partner.BalanceSourceTypeReceiptVoucher,
		SourceID:   "RCV-001",
	})

	// Assert - should fail with optimistic lock error
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save customer")

	customerRepo.AssertExpectations(t)
}

func TestBalancePaymentService_RefundBalancePayment_OptimisticLockError(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	customerID := uuid.New()

	customerRepo := new(MockCustomerRepositoryForBalance)
	balanceTxRepo := new(MockBalanceTransactionRepository)
	service := NewBalancePaymentService(customerRepo, balanceTxRepo)

	// Customer has balance
	customer := createTestCustomerWithBalance(tenantID, decimal.NewFromFloat(500.00))
	customer.ID = customerID

	// Mock expectations - SaveWithLock fails due to concurrent modification
	customerRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	customerRepo.On("SaveWithLock", ctx, mock.AnythingOfType("*partner.Customer")).Return(
		shared.NewDomainError("OPTIMISTIC_LOCK_ERROR", "The customer record has been modified by another transaction"),
	)

	// Execute
	result, err := service.RefundBalancePayment(
		ctx,
		tenantID,
		customerID,
		decimal.NewFromFloat(200.00),
		"RCV-001",
		"REF-001",
		"Refund for cancelled voucher",
		nil,
	)

	// Assert - should fail with optimistic lock error
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save customer")

	customerRepo.AssertExpectations(t)
}
