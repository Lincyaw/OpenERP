package partner

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Repositories
// =============================================================================

// MockCustomerRepository is a mock implementation of CustomerRepository
type MockCustomerRepository struct {
	mock.Mock
}

func (m *MockCustomerRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Customer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*partner.Customer, error) {
	args := m.Called(ctx, tenantID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByType(ctx context.Context, tenantID uuid.UUID, customerType partner.CustomerType, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, customerType, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByLevel(ctx context.Context, tenantID uuid.UUID, level partner.CustomerLevel, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, level, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.CustomerStatus, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindWithPositiveBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Customer, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Customer), args.Error(1)
}

func (m *MockCustomerRepository) Save(ctx context.Context, customer *partner.Customer) error {
	args := m.Called(ctx, customer)
	return args.Error(0)
}

func (m *MockCustomerRepository) SaveBatch(ctx context.Context, customers []*partner.Customer) error {
	args := m.Called(ctx, customers)
	return args.Error(0)
}

func (m *MockCustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCustomerRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockCustomerRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepository) CountByType(ctx context.Context, tenantID uuid.UUID, customerType partner.CustomerType) (int64, error) {
	args := m.Called(ctx, tenantID, customerType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepository) CountByLevel(ctx context.Context, tenantID uuid.UUID, level partner.CustomerLevel) (int64, error) {
	args := m.Called(ctx, tenantID, level)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.CustomerStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockCustomerRepository) ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error) {
	args := m.Called(ctx, tenantID, phone)
	return args.Bool(0), args.Error(1)
}

func (m *MockCustomerRepository) ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	args := m.Called(ctx, tenantID, email)
	return args.Bool(0), args.Error(1)
}

// =============================================================================
// Test Helper Functions
// =============================================================================

func newTestTenantID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func newTestCustomerID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func createTestCustomer(tenantID uuid.UUID) *partner.Customer {
	customer, _ := partner.NewIndividualCustomer(tenantID, "CUST-001", "Test Customer")
	return customer
}

// =============================================================================
// CustomerService Tests
// =============================================================================

func TestCustomerService_Create_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	req := CreateCustomerRequest{
		Code: "NEW-CUST-001",
		Name: "New Customer",
		Type: "individual",
	}

	mockRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "NEW-CUST-001", result.Code)
	assert.Equal(t, "New Customer", result.Name)
	assert.Equal(t, "individual", result.Type)
	assert.Equal(t, "active", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Create_WithAllFields(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	creditLimit := decimal.NewFromFloat(10000.00)
	sortOrder := 5

	req := CreateCustomerRequest{
		Code:        "FULL-CUST-001",
		Name:        "Full Customer",
		ShortName:   "FC",
		Type:        "organization",
		ContactName: "John Doe",
		Phone:       "13800138000",
		Email:       "john@example.com",
		Address:     "123 Main St",
		City:        "Shanghai",
		Province:    "Shanghai",
		PostalCode:  "200000",
		Country:     "中国",
		TaxID:       "1234567890",
		CreditLimit: &creditLimit,
		Notes:       "VIP customer",
		SortOrder:   &sortOrder,
		Attributes:  `{"industry": "tech"}`,
	}

	mockRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockRepo.On("ExistsByPhone", ctx, tenantID, req.Phone).Return(false, nil)
	mockRepo.On("ExistsByEmail", ctx, tenantID, req.Email).Return(false, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "FULL-CUST-001", result.Code)
	assert.Equal(t, "Full Customer", result.Name)
	assert.Equal(t, "organization", result.Type)
	assert.Equal(t, "John Doe", result.ContactName)
	assert.Equal(t, "13800138000", result.Phone)
	assert.True(t, result.CreditLimit.Equal(creditLimit))
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Create_DuplicateCode(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	req := CreateCustomerRequest{
		Code: "EXISTING-001",
		Name: "New Customer",
		Type: "individual",
	}

	mockRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(true, nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_EXISTS", domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Create_DuplicatePhone(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	req := CreateCustomerRequest{
		Code:  "NEW-001",
		Name:  "New Customer",
		Type:  "individual",
		Phone: "13800138000",
	}

	mockRepo.On("ExistsByCode", ctx, tenantID, req.Code).Return(false, nil)
	mockRepo.On("ExistsByPhone", ctx, tenantID, req.Phone).Return(true, nil)

	result, err := service.Create(ctx, tenantID, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "ALREADY_EXISTS", domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_GetByID_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)

	result, err := service.GetByID(ctx, tenantID, customerID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, customer.Code, result.Code)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_GetByID_NotFound(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(nil, shared.ErrNotFound)

	result, err := service.GetByID(ctx, tenantID, customerID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, shared.ErrNotFound)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_List_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	filter := CustomerListFilter{
		Page:     1,
		PageSize: 10,
	}

	customers := []partner.Customer{
		*createTestCustomer(tenantID),
	}

	mockRepo.On("FindAllForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(customers, nil)
	mockRepo.On("CountForTenant", ctx, tenantID, mock.AnythingOfType("shared.Filter")).Return(int64(1), nil)

	result, total, err := service.List(ctx, tenantID, filter)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), total)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Update_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)

	newName := "Updated Name"
	newNotes := "Updated Notes"
	req := UpdateCustomerRequest{
		Name:  &newName,
		Notes: &newNotes,
	}

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.Update(ctx, tenantID, customerID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_UpdateCode_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)
	newCode := "NEW-CODE-001"

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("ExistsByCode", ctx, tenantID, newCode).Return(false, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.UpdateCode(ctx, tenantID, customerID, newCode)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "NEW-CODE-001", result.Code)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Delete_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("DeleteForTenant", ctx, tenantID, customerID).Return(nil)

	err := service.Delete(ctx, tenantID, customerID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Delete_HasBalance(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)
	customer.AddBalance(decimal.NewFromFloat(100.00)) // Add balance

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)

	err := service.Delete(ctx, tenantID, customerID)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	assert.ErrorAs(t, err, &domainErr)
	assert.Equal(t, "CANNOT_DELETE", domainErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Activate_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)
	customer.Deactivate()

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.Activate(ctx, tenantID, customerID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "active", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Deactivate_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.Deactivate(ctx, tenantID, customerID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "inactive", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_Suspend_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.Suspend(ctx, tenantID, customerID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "suspended", result.Status)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_AddBalance_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)
	amount := decimal.NewFromFloat(100.00)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.AddBalance(ctx, tenantID, customerID, amount)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Balance.Equal(amount))
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_SetLevel_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()
	customerID := newTestCustomerID()
	customer := createTestCustomer(tenantID)

	mockRepo.On("FindByIDForTenant", ctx, tenantID, customerID).Return(customer, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)

	result, err := service.SetLevel(ctx, tenantID, customerID, "gold")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "gold", result.Level)
	mockRepo.AssertExpectations(t)
}

func TestCustomerService_CountByStatus_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	service := NewCustomerService(mockRepo)

	ctx := context.Background()
	tenantID := newTestTenantID()

	mockRepo.On("CountByStatus", ctx, tenantID, partner.CustomerStatusActive).Return(int64(10), nil)
	mockRepo.On("CountByStatus", ctx, tenantID, partner.CustomerStatusInactive).Return(int64(5), nil)
	mockRepo.On("CountByStatus", ctx, tenantID, partner.CustomerStatusSuspended).Return(int64(2), nil)

	counts, err := service.CountByStatus(ctx, tenantID)

	assert.NoError(t, err)
	assert.NotNil(t, counts)
	assert.Equal(t, int64(10), counts["active"])
	assert.Equal(t, int64(5), counts["inactive"])
	assert.Equal(t, int64(2), counts["suspended"])
	assert.Equal(t, int64(17), counts["total"])
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// DTO Conversion Tests
// =============================================================================

func TestToCustomerResponse(t *testing.T) {
	tenantID := newTestTenantID()
	customer := createTestCustomer(tenantID)

	result := ToCustomerResponse(customer)

	assert.Equal(t, customer.ID, result.ID)
	assert.Equal(t, customer.TenantID, result.TenantID)
	assert.Equal(t, customer.Code, result.Code)
	assert.Equal(t, customer.Name, result.Name)
	assert.Equal(t, string(customer.Type), result.Type)
	assert.Equal(t, string(customer.Status), result.Status)
}

func TestToCustomerListResponses(t *testing.T) {
	tenantID := newTestTenantID()
	customers := []partner.Customer{
		*createTestCustomer(tenantID),
		*createTestCustomer(tenantID),
	}

	results := ToCustomerListResponses(customers)

	assert.Len(t, results, 2)
	assert.Equal(t, customers[0].Code, results[0].Code)
	assert.Equal(t, customers[1].Code, results[1].Code)
}
