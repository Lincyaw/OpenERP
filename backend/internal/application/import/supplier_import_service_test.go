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

// MockSupplierRepository is a mock implementation of partner.SupplierRepository
type MockSupplierRepository struct {
	mock.Mock
}

func (m *MockSupplierRepository) FindByID(ctx context.Context, id uuid.UUID) (*partner.Supplier, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*partner.Supplier, error) {
	args := m.Called(ctx, tenantID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindAll(ctx context.Context, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByType(ctx context.Context, tenantID uuid.UUID, supplierType partner.SupplierType, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, supplierType, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status partner.SupplierStatus, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, status, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindActive(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, ids)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindByCodes(ctx context.Context, tenantID uuid.UUID, codes []string) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, codes)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindWithOutstandingBalance(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) FindOverCreditLimit(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) ([]partner.Supplier, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]partner.Supplier), args.Error(1)
}

func (m *MockSupplierRepository) Save(ctx context.Context, supplier *partner.Supplier) error {
	args := m.Called(ctx, supplier)
	return args.Error(0)
}

func (m *MockSupplierRepository) SaveBatch(ctx context.Context, suppliers []*partner.Supplier) error {
	args := m.Called(ctx, suppliers)
	return args.Error(0)
}

func (m *MockSupplierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSupplierRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockSupplierRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) CountByType(ctx context.Context, tenantID uuid.UUID, supplierType partner.SupplierType) (int64, error) {
	args := m.Called(ctx, tenantID, supplierType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status partner.SupplierStatus) (int64, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSupplierRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockSupplierRepository) ExistsByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (bool, error) {
	args := m.Called(ctx, tenantID, phone)
	return args.Bool(0), args.Error(1)
}

func (m *MockSupplierRepository) ExistsByEmail(ctx context.Context, tenantID uuid.UUID, email string) (bool, error) {
	args := m.Called(ctx, tenantID, email)
	return args.Bool(0), args.Error(1)
}

// Test helpers for supplier import
func newSupplierValidatedSession(tenantID, userID uuid.UUID) *csvimport.ImportSession {
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntitySuppliers, "suppliers.csv", 1024)
	session.UpdateState(csvimport.StateValidating)
	session.TotalRows = 2
	session.ValidRows = 2
	session.ErrorRows = 0
	session.UpdateState(csvimport.StateValidated)
	return session
}

// Tests for validateSupplierType
func TestValidateSupplierType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"manufacturer is valid", "manufacturer", false},
		{"distributor is valid", "distributor", false},
		{"retailer is valid", "retailer", false},
		{"service is valid", "service", false},
		{"MANUFACTURER uppercase is valid", "MANUFACTURER", false},
		{"Distributor mixed case is valid", "Distributor", false},
		{"unknown is invalid", "unknown", true},
		{"supplier is invalid", "supplier", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSupplierType(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Tests for normalizeSupplierType
func TestNormalizeSupplierType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected partner.SupplierType
	}{
		{"manufacturer", "manufacturer", partner.SupplierTypeManufacturer},
		{"MANUFACTURER", "MANUFACTURER", partner.SupplierTypeManufacturer},
		{"distributor", "distributor", partner.SupplierTypeDistributor},
		{"Distributor", "Distributor", partner.SupplierTypeDistributor},
		{"retailer", "retailer", partner.SupplierTypeRetailer},
		{"service", "service", partner.SupplierTypeService},
		{"unknown defaults to distributor", "unknown", partner.SupplierTypeDistributor},
		{"empty defaults to distributor", "", partner.SupplierTypeDistributor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSupplierType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for GetValidationRules
func TestSupplierImportService_GetValidationRules(t *testing.T) {
	supplierRepo := new(MockSupplierRepository)
	service := NewSupplierImportService(supplierRepo, nil)

	rules := service.GetValidationRules()

	// Verify required fields
	requiredFields := map[string]bool{
		"name": false,
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

	// Verify all expected columns are present
	expectedColumns := []string{
		"name", "code", "type", "contact_person", "phone", "email",
		"credit_days", "credit_limit", "address_province", "address_city",
		"address_district", "address_detail", "bank_name", "bank_account", "notes",
	}

	columnMap := make(map[string]bool)
	for _, rule := range rules {
		columnMap[rule.Column] = true
	}

	for _, col := range expectedColumns {
		assert.True(t, columnMap[col], "column %s should be present in rules", col)
	}
}

// Tests for LookupUnique
func TestSupplierImportService_LookupUnique(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()

	t.Run("empty value returns false", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "code", "")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("existing code returns true", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		supplierRepo.On("ExistsByCode", ctx, tenantID, "SUPP001").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "code", "SUPP001")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("existing email returns true", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		supplierRepo.On("ExistsByEmail", ctx, tenantID, "supplier@example.com").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "email", "supplier@example.com")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("existing phone returns true", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		supplierRepo.On("ExistsByPhone", ctx, tenantID, "13800138000").Return(true, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "phone", "13800138000")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("unknown field returns false", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		exists, err := service.LookupUnique(ctx, tenantID, "unknown", "value")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("database error returns error", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		dbErr := errors.New("database connection failed")
		supplierRepo.On("ExistsByCode", ctx, tenantID, "SUPP001").Return(false, dbErr)

		_, err := service.LookupUnique(ctx, tenantID, "code", "SUPP001")
		assert.Error(t, err)
		assert.Equal(t, dbErr, err)
	})
}

// Tests for Import
func TestSupplierImportService_Import(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("invalid session state returns error", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := csvimport.NewImportSession(tenantID, userID, csvimport.EntitySuppliers, "suppliers.csv", 1024)
		// Session is in "created" state, not "validated"

		_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validated state")
	})

	t.Run("session with errors returns error", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)
		session.ErrorRows = 1 // Has errors

		_, err := service.Import(ctx, tenantID, userID, session, nil, ConflictModeSkip)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation errors")
	})

	t.Run("cancels import when context is cancelled", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Supplier One",
				"code": "SUPP001",
			}),
		}

		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := service.Import(cancelledCtx, tenantID, userID, session, rows, ConflictModeSkip)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, csvimport.StateCancelled, session.State)
	})

	t.Run("successful import creates suppliers", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":             "Supplier One",
				"type":             "manufacturer",
				"code":             "SUPP001",
				"contact_person":   "John Doe",
				"phone":            "13800138001",
				"email":            "john@supplier.com",
				"credit_days":      "30",
				"credit_limit":     "100000",
				"address_city":     "Shanghai",
				"address_province": "Shanghai",
				"bank_name":        "Bank of China",
				"bank_account":     "1234567890",
			}),
		}

		// No existing supplier
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(nil)
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
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)
		service.ResetCodeSequence()

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Supplier Without Code",
				// code is empty - should be auto-generated
			}),
		}

		// Match any generated code
		supplierRepo.On("FindByCode", ctx, tenantID, mock.MatchedBy(func(code string) bool {
			return len(code) > 0 && code[:4] == "SUPP"
		})).Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("skip mode skips existing suppliers", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Existing Supplier",
				"code": "SUPP001",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Existing Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.SkippedRows)
	})

	t.Run("fail mode reports error on existing suppliers", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Existing Supplier",
				"code": "SUPP001",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Existing Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeFail)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "already exists")
	})

	t.Run("update mode updates existing suppliers", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":         "Updated Supplier",
				"code":         "SUPP001",
				"credit_days":  "60",
				"credit_limit": "200000",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Original Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalRows)
		assert.Equal(t, 0, result.ImportedRows)
		assert.Equal(t, 1, result.UpdatedRows)
	})

	t.Run("sets payment terms correctly", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":         "Supplier with Credit",
				"code":         "SUPP001",
				"credit_days":  "45",
				"credit_limit": "500000",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.CreditDays == 45 && s.CreditLimit.Equal(decimal.NewFromInt(500000))
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("sets bank info correctly", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":         "Supplier with Bank",
				"code":         "SUPP001",
				"bank_name":    "Industrial Bank",
				"bank_account": "9876543210",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.BankName == "Industrial Bank" && s.BankAccount == "9876543210"
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("handles manufacturer supplier type", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Manufacturer Inc.",
				"type": "manufacturer",
				"code": "MANU001",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "MANU001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.Type == partner.SupplierTypeManufacturer
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("handles retailer supplier type", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Retailer Shop",
				"type": "retailer",
				"code": "RET001",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "RET001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.Type == partner.SupplierTypeRetailer
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("handles service supplier type", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Service Provider",
				"type": "service",
				"code": "SVC001",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SVC001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.Type == partner.SupplierTypeService
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("reports error for invalid credit_days", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":        "Supplier",
				"code":        "SUPP001",
				"credit_days": "invalid",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Contains(t, result.Errors[0].Message, "invalid integer")
	})

	t.Run("reports error for invalid credit_limit", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":         "Supplier",
				"code":         "SUPP001",
				"credit_limit": "not-a-number",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Contains(t, result.Errors[0].Message, "invalid decimal")
	})
}

// Tests for code generation
func TestSupplierImportService_GenerateCode(t *testing.T) {
	service := &SupplierImportService{}
	service.ResetCodeSequence()

	t.Run("generates unique codes", func(t *testing.T) {
		code1, err := service.generateCode()
		require.NoError(t, err)
		assert.True(t, len(code1) > 0)
		assert.Contains(t, code1, "SUPP-")

		code2, err := service.generateCode()
		require.NoError(t, err)
		assert.NotEqual(t, code1, code2)
	})

	t.Run("code format is correct", func(t *testing.T) {
		service.ResetCodeSequence()
		code, err := service.generateCode()
		require.NoError(t, err)
		// Format: SUPP-YYYYMMDD-NNNNNN
		assert.Regexp(t, `^SUPP-\d{8}-\d{6}$`, code)
	})
}

// Tests for ValidateWithWarnings
func TestSupplierImportService_ValidateWithWarnings(t *testing.T) {
	service := &SupplierImportService{}

	t.Run("warns about very high credit limit", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"credit_limit": "20000000",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "credit limit")
		assert.Contains(t, warnings[0], "high")
	})

	t.Run("warns about very high credit days", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"credit_days": "365",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "credit days")
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
			"email": "supplier@example.com",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "test address")
	})

	t.Run("no warnings for valid data", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"credit_limit": "100000",
			"credit_days":  "30",
			"email":        "supplier@company.com",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 0)
	})

	t.Run("invalid credit values do not warn", func(t *testing.T) {
		row := newTestRow(2, map[string]string{
			"credit_limit": "invalid",
			"credit_days":  "invalid",
		})

		warnings := service.ValidateWithWarnings(row)
		assert.Len(t, warnings, 0)
	})
}

// Tests for address handling
func TestSupplierImportService_AddressHandling(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("imports supplier with full address", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":             "Supplier With Address",
				"code":             "SUPP001",
				"address_province": "Guangdong",
				"address_city":     "Shenzhen",
				"address_district": "Nanshan",
				"address_detail":   "Tech Park Road 100",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			// Verify address was set correctly
			return s.Province == "Guangdong" &&
				s.City == "Shenzhen" &&
				s.Address == "Nanshan Tech Park Road 100"
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})
}

// Tests for SupplierCreated event publishing
func TestSupplierImportService_EventPublishing(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("publishes SupplierCreated event on successful import", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "New Supplier",
				"code": "SUPP001",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(nil)
		eventBus.On("Publish", ctx, mock.MatchedBy(func(events []shared.DomainEvent) bool {
			if len(events) == 0 {
				return false
			}
			for _, e := range events {
				if e.EventType() == partner.EventTypeSupplierCreated {
					return true
				}
			}
			return false
		})).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
		eventBus.AssertCalled(t, "Publish", ctx, mock.Anything)
	})
}

// Tests for update mode with various field combinations
func TestSupplierImportService_UpdateMode(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("update mode updates contact info", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":           "Updated Supplier",
				"code":           "SUPP001",
				"contact_person": "Jane Doe",
				"phone":          "13900139001",
				"email":          "jane@supplier.com",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Original Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.ContactName == "Jane Doe" &&
				s.Phone == "13900139001" &&
				s.Email == "jane@supplier.com"
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.UpdatedRows)
	})

	t.Run("update mode updates bank info", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":         "Updated Supplier",
				"code":         "SUPP001",
				"bank_name":    "New Bank",
				"bank_account": "1111222233334444",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Original Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.BankName == "New Bank" && s.BankAccount == "1111222233334444"
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.UpdatedRows)
	})

	t.Run("update mode updates address", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":             "Updated Supplier",
				"code":             "SUPP001",
				"address_province": "Beijing",
				"address_city":     "Beijing",
				"address_detail":   "CBD Tower 888",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Original Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.Province == "Beijing" && s.City == "Beijing"
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.UpdatedRows)
	})

	t.Run("update mode updates notes", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name":  "Updated Supplier",
				"code":  "SUPP001",
				"notes": "Updated notes content",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Original Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)
		supplierRepo.On("Save", ctx, mock.MatchedBy(func(s *partner.Supplier) bool {
			return s.Notes == "Updated notes content"
		})).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.UpdatedRows)
	})
}

// Tests for error scenarios
func TestSupplierImportService_ErrorScenarios(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("database error on FindByCode returns error", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Supplier",
				"code": "SUPP001",
			}),
		}

		dbErr := errors.New("database connection failed")
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, dbErr)

		_, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection failed")
	})

	t.Run("save error on create records row error", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Supplier",
				"code": "SUPP001",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(errors.New("save failed"))

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Contains(t, result.Errors[0].Message, "failed to save supplier")
	})

	t.Run("save error on update records row error", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Updated Supplier",
				"code": "SUPP001",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Original Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(errors.New("save failed"))

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
		assert.Contains(t, result.Errors[0].Message, "failed to save supplier")
	})

	t.Run("event publish error logs warning but continues", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		eventBus := new(MockEventPublisher)
		service := NewSupplierImportService(supplierRepo, eventBus)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Supplier",
				"code": "SUPP001",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(nil)
		eventBus.On("Publish", ctx, mock.Anything).Return(errors.New("event bus error"))

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		// Import should succeed even if event publishing fails
		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})
}

// Tests for import without event bus
func TestSupplierImportService_NoEventBus(t *testing.T) {
	ctx := context.Background()
	tenantID := newTestTenantID()
	userID := newTestUserID()

	t.Run("import succeeds without event bus", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		// Pass nil event bus
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Supplier Without Events",
				"code": "SUPP001",
			}),
		}

		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(nil, shared.ErrNotFound)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeSkip)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ImportedRows)
	})

	t.Run("update succeeds without event bus", func(t *testing.T) {
		supplierRepo := new(MockSupplierRepository)
		// Pass nil event bus
		service := NewSupplierImportService(supplierRepo, nil)

		session := newSupplierValidatedSession(tenantID, userID)

		rows := []*csvimport.Row{
			newTestRow(2, map[string]string{
				"name": "Updated Supplier",
				"code": "SUPP001",
			}),
		}

		existingSupplier, _ := partner.NewSupplier(tenantID, "SUPP001", "Original Supplier", partner.SupplierTypeDistributor)
		supplierRepo.On("FindByCode", ctx, tenantID, "SUPP001").Return(existingSupplier, nil)
		supplierRepo.On("Save", ctx, mock.AnythingOfType("*partner.Supplier")).Return(nil)

		result, err := service.Import(ctx, tenantID, userID, session, rows, ConflictModeUpdate)

		require.NoError(t, err)
		assert.Equal(t, 1, result.UpdatedRows)
	})
}
