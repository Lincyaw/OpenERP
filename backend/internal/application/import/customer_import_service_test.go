package importapp

import (
	"context"
	"errors"
	"testing"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCustomerRepository is a mock implementation of partner.CustomerRepository
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

func (m *MockCustomerRepository) SaveWithLock(ctx context.Context, customer *partner.Customer) error {
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

// MockCustomerLevelRepository is a mock implementation of partner.CustomerLevelRepository
type MockCustomerLevelRepository struct {
	mock.Mock
}

func (m *MockCustomerLevelRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.CustomerLevelRecord, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.CustomerLevelRecord), args.Error(1)
}

func (m *MockCustomerLevelRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.CustomerLevelRecord, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.CustomerLevelRecord), args.Error(1)
}

func (m *MockCustomerLevelRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.CustomerLevelRecord, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.CustomerLevelRecord), args.Error(1)
}

func (m *MockCustomerLevelRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID) ([]*partner.CustomerLevelRecord, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*partner.CustomerLevelRecord), args.Error(1)
}

func (m *MockCustomerLevelRepository) FindActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]*partner.CustomerLevelRecord, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*partner.CustomerLevelRecord), args.Error(1)
}

func (m *MockCustomerLevelRepository) FindDefaultForTenant(ctx context.Context, tenantID uuid.UUID) (*partner.CustomerLevelRecord, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.CustomerLevelRecord), args.Error(1)
}

func (m *MockCustomerLevelRepository) Save(ctx context.Context, record *partner.CustomerLevelRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockCustomerLevelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCustomerLevelRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockCustomerLevelRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockCustomerLevelRepository) CountCustomersWithLevel(ctx context.Context, tenantID uuid.UUID, levelCode string) (int64, error) {
	args := m.Called(ctx, tenantID, levelCode)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCustomerLevelRepository) CountCustomersByLevelCodes(ctx context.Context, tenantID uuid.UUID, codes []string) (map[string]int64, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).(map[string]int64), args.Error(1)
}

func (m *MockCustomerLevelRepository) InitializeDefaultLevels(ctx context.Context, tenantID uuid.UUID) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

// Test helpers for customer import
func newCustomerValidatedSession(tenantID, userID uuid.UUID) *csvimport.ImportSession {
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityCustomers, "customers.csv", 1024)
	session.UpdateState(csvimport.StateValidating)
	session.TotalRows = 2
	session.ValidRows = 2
	session.ErrorRows = 0
	session.UpdateState(csvimport.StateValidated)
	return session
}

func newTestLevelRecord(tenantID uuid.UUID, code, name string, discountRate decimal.Decimal) *partner.CustomerLevelRecord {
	return &partner.CustomerLevelRecord{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Code:         code,
		Name:         name,
		DiscountRate: discountRate,
		IsActive:     true,
	}
}

// Tests for validateCustomerType
func TestValidateCustomerType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"individual is valid", "individual", false},
		{"company is valid", "company", false},
		{"organization is valid", "organization", false},
		{"INDIVIDUAL uppercase is valid", "INDIVIDUAL", false},
		{"Company mixed case is valid", "Company", false},
		{"unknown is invalid", "unknown", true},
		{"business is invalid", "business", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCustomerType(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Tests for normalizeCustomerType
func TestNormalizeCustomerType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected partner.CustomerType
	}{
		{"individual", "individual", partner.CustomerTypeIndividual},
		{"INDIVIDUAL", "INDIVIDUAL", partner.CustomerTypeIndividual},
		{"company", "company", partner.CustomerTypeOrganization},
		{"Company", "Company", partner.CustomerTypeOrganization},
		{"organization", "organization", partner.CustomerTypeOrganization},
		{"unknown defaults to individual", "unknown", partner.CustomerTypeIndividual},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCustomerType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for GetValidationRules
func TestCustomerImportService_GetValidationRules(t *testing.T) {
	customerRepo := new(MockCustomerRepository)
	levelRepo := new(MockCustomerLevelRepository)
	service := NewCustomerImportService(customerRepo, levelRepo, nil)

	rules := service.GetValidationRules()

	// Verify required fields
	requiredFields := map[string]bool{
		"name": false,
		"type": false,
	}

	for _, rule := range rules {
		if _, ok := requiredFields[rule.Column]; ok {
			requiredFields[rule.Column] = rule.Required
		}
	}

	for field, required := range requiredFields {
		assert.True(t, required, "field %s should be required", field)
	}

	// Verify unique fields
	uniqueFields := map[string]bool{
		"code": false,
	}

	for _, rule := range rules {
		if _, ok := uniqueFields[rule.Column]; ok {
			uniqueFields[rule.Column] = rule.Unique
		}
	}

	for field, unique := range uniqueFields {
		assert.True(t, unique, "field %s should be unique", field)
	}

	// Verify reference fields
	referenceFields := map[string]string{
		"level_code": "customer_level",
	}

	for _, rule := range rules {
		if expectedRef, ok := referenceFields[rule.Column]; ok {
			assert.Equal(t, expectedRef, rule.Reference, "field %s should have reference %s", rule.Column, expectedRef)
		}
	}
}

// Tests for LookupCustomerLevel
func TestCustomerImportService_LookupCustomerLevel(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()

	t.Run("empty code returns true", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		exists, err := service.LookupCustomerLevel(ctx, tenantID, "")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("existing level returns true", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		levelRepo.On("ExistsByCode", ctx, tenantID, "gold").Return(true, nil)

		exists, err := service.LookupCustomerLevel(ctx, tenantID, "gold")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existing level returns false", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		levelRepo.On("ExistsByCode", ctx, tenantID, "unknown").Return(false, nil)

		exists, err := service.LookupCustomerLevel(ctx, tenantID, "unknown")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("database error returns error", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		dbErr := errors.New("database connection failed")
		levelRepo.On("ExistsByCode", ctx, tenantID, "gold").Return(false, dbErr)

		_, err := service.LookupCustomerLevel(ctx, tenantID, "gold")
		assert.Error(t, err)
		assert.Equal(t, dbErr, err)
	})
}

// Tests for LookupUnique
func TestCustomerImportService_LookupUnique(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()

	t.Run("empty value returns false", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "code", "")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("existing code returns true", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		customerRepo.On("ExistsByCode", ctx, tenantID, "CUST001").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "code", "CUST001")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("existing email returns true", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		customerRepo.On("ExistsByEmail", ctx, tenantID, "test@example.com").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "email", "test@example.com")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("existing phone returns true", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		customerRepo.On("ExistsByPhone", ctx, tenantID, "13800138000").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "phone", "13800138000")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("unknown field returns false", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "unknown", "value")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

// Tests for Import
func TestCustomerImportService_Import(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("invalid session state returns error", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityCustomers, "customers.csv", 1024)
		// Session is in "created" state, not "validated"

		_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validated state")
	})

	t.Run("session with errors returns error", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		session := newCustomerValidatedSession(tenantID, userID)
		session.ErrorRows = 1 // Has errors

		_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation errors")
	})

	t.Run("cancels import when context is cancelled", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Customer One",
				"type": "individual",
				"code": "CUST001",
			}),
		}

		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := service.Import(cancelledCtx, tenantID, userID, session, rows, ConflictModeSkip)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, csvimport.StateCancelled, session.State)
	})

	t.Run("successful import creates customers", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		eventBus := new(MockEventPublisher)
		service := NewCustomerImportService(customerRepo, levelRepo, eventBus)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":             "Customer One",
				"type":             "individual",
				"code":             "CUST001",
				"contact_person":   "John Doe",
				"phone":            "13800138001",
				"email":            "john@example.com",
				"credit_limit":     "10000",
				"address_city":     "Shanghai",
				"address_province": "Shanghai",
			}),
		}

		// No existing customer
		customerRepo.On("FindByCode", ctx, tenantID, "CUST001").Return(nil, shared.ErrNotFound)
		customerRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 1, result.ImportedRows)
		assert.Equal(t, 0, result.UpdatedRows)
		assert.Equal(t, 0, result.SkippedRows)
		assert.Equal(t, 0, result.ErrorRows)
	})

	t.Run("auto-generates code when not provided", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		eventBus := new(MockEventPublisher)
		service := NewCustomerImportService(customerRepo, levelRepo, eventBus)
		service.ResetCodeSequence()

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Customer Without Code",
				"type": "individual",
				// code is empty - should be auto-generated
			}),
		}

		// Match any generated code
		customerRepo.On("FindByCode", ctx, tenantID, mock.MatchedBy(func(code string) bool {
			return len(code) > 0 && code[:4] == "CUST"
		})).Return(nil, shared.ErrNotFound)
		customerRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("skip mode skips existing customers", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Existing Customer",
				"type": "individual",
				"code": "CUST001",
			}),
		}

		existingCustomer, _ := partner.NewCustomer(tenantID, "CUST001", "Existing Customer", partner.CustomerTypeIndividual)
		customerRepo.On("FindByCode", ctx, tenantID, "CUST001").Return(existingCustomer, nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.SkippedRows)
	})

	t.Run("fail mode reports error on existing customers", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Existing Customer",
				"type": "individual",
				"code": "CUST001",
			}),
		}

		existingCustomer, _ := partner.NewCustomer(tenantID, "CUST001", "Existing Customer", partner.CustomerTypeIndividual)
		customerRepo.On("FindByCode", ctx, tenantID, "CUST001").Return(existingCustomer, nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeFail)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "already exists")
	})

	t.Run("update mode updates existing customers", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		eventBus := new(MockEventPublisher)
		service := NewCustomerImportService(customerRepo, levelRepo, eventBus)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":         "Updated Customer",
				"type":         "individual",
				"code":         "CUST001",
				"credit_limit": "50000",
			}),
		}

		existingCustomer, _ := partner.NewCustomer(tenantID, "CUST001", "Original Customer", partner.CustomerTypeIndividual)
		customerRepo.On("FindByCode", ctx, tenantID, "CUST001").Return(existingCustomer, nil)
		customerRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.UpdatedRows)
	})

	t.Run("sets customer level when provided", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		eventBus := new(MockEventPublisher)
		service := NewCustomerImportService(customerRepo, levelRepo, eventBus)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":       "VIP Customer",
				"type":       "company",
				"code":       "CUST001",
				"level_code": "gold",
			}),
		}

		customerRepo.On("FindByCode", ctx, tenantID, "CUST001").Return(nil, shared.ErrNotFound)
		levelRecord := newTestLevelRecord(tenantID, "gold", "Gold Level", decimal.NewFromFloat(0.05))
		levelRepo.On("FindByCode", ctx, tenantID, "gold").Return(levelRecord, nil)
		customerRepo.On("Save", ctx, mock.AnythingOfType("*partner.Customer")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("reports error for invalid customer level", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		service := NewCustomerImportService(customerRepo, levelRepo, nil)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":       "Customer",
				"type":       "individual",
				"code":       "CUST001",
				"level_code": "nonexistent",
			}),
		}

		customerRepo.On("FindByCode", ctx, tenantID, "CUST001").Return(nil, shared.ErrNotFound)
		levelRepo.On("FindByCode", ctx, tenantID, "nonexistent").Return(nil, shared.ErrNotFound)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Contains(t, result.Errors[0].Message, "not found")
	})

	t.Run("handles organization customer type", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		eventBus := new(MockEventPublisher)
		service := NewCustomerImportService(customerRepo, levelRepo, eventBus)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Company Inc.",
				"type": "company",
				"code": "COMP001",
			}),
		}

		customerRepo.On("FindByCode", ctx, tenantID, "COMP001").Return(nil, shared.ErrNotFound)
		customerRepo.On("Save", ctx, mock.MatchedBy(func(c *partner.Customer) bool {
			return c.Type == partner.CustomerTypeOrganization
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})
}

// Tests for code generation
func TestCustomerImportService_GenerateCode(t *testing.T) {
	service := &CustomerImportService{}
	service.ResetCodeSequence()

	t.Run("generates unique codes", func(t *testing.T) {
		code1, err := service.generateCode()
		require.NoError(t, err)
		assert.True(t, len(code1) > 0)
		assert.Contains(t, code1, "CUST-")

		code2, err := service.generateCode()
		require.NoError(t, err)
		assert.NotEqual(t, code1, code2)
	})

	t.Run("code format is correct", func(t *testing.T) {
		service.ResetCodeSequence()
		code, err := service.generateCode()
		require.NoError(t, err)
		// Format: CUST-YYYYMMDD-NNNNNN
		assert.Regexp(t, `^CUST-\d{8}-\d{6}$`, code)
	})
}

// Tests for ValidateWithWarnings
func TestCustomerImportService_ValidateWithWarnings(t *testing.T) {
	service := &CustomerImportService{}

	t.Run("warns about very high credit limit", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"credit_limit": "2000000",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "credit limit")
		assert.Contains(t, warnings[0], "high")
	})

	t.Run("warns about test email", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"email": "test@test.com",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "test address")
	})

	t.Run("warns about example email", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"email": "user@example.com",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "test address")
	})

	t.Run("no warnings for valid data", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"credit_limit": "10000",
			"email":        "john.doe@company.com",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 0)
	})

	t.Run("invalid credit limit does not warn", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"credit_limit": "invalid",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 0)
	})
}

// Tests for address handling
func TestCustomerImportService_AddressHandling(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("imports customer with full address", func(t *testing.T) {
		customerRepo := new(MockCustomerRepository)
		levelRepo := new(MockCustomerLevelRepository)
		eventBus := new(MockEventPublisher)
		service := NewCustomerImportService(customerRepo, levelRepo, eventBus)

		session := newCustomerValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":             "Customer With Address",
				"type":             "individual",
				"code":             "CUST001",
				"address_province": "Shanghai",
				"address_city":     "Shanghai",
				"address_district": "Pudong",
				"address_detail":   "Century Avenue 100",
			}),
		}

		customerRepo.On("FindByCode", ctx, tenantID, "CUST001").Return(nil, shared.ErrNotFound)
		customerRepo.On("Save", ctx, mock.MatchedBy(func(c *partner.Customer) bool {
			// Verify address was set correctly
			return c.Province == "Shanghai" &&
				c.City == "Shanghai" &&
				c.Address == "Pudong Century Avenue 100"
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})
}
