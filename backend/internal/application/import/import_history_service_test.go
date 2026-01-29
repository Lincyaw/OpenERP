package importapp

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/bulk"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockImportHistoryRepository is a mock implementation of ImportHistoryRepository
type MockImportHistoryRepository struct {
	mock.Mock
}

func (m *MockImportHistoryRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*bulk.ImportHistory, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bulk.ImportHistory), args.Error(1)
}

func (m *MockImportHistoryRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter bulk.ImportHistoryFilter, page, pageSize int) (*bulk.ImportHistoryListResult, error) {
	args := m.Called(ctx, tenantID, filter, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bulk.ImportHistoryListResult), args.Error(1)
}

func (m *MockImportHistoryRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status bulk.ImportStatus) ([]*bulk.ImportHistory, error) {
	args := m.Called(ctx, tenantID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*bulk.ImportHistory), args.Error(1)
}

func (m *MockImportHistoryRepository) FindPending(ctx context.Context, tenantID uuid.UUID) ([]*bulk.ImportHistory, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*bulk.ImportHistory), args.Error(1)
}

func (m *MockImportHistoryRepository) Save(ctx context.Context, history *bulk.ImportHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockImportHistoryRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func TestImportHistoryService_CreateHistory(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		repo.On("Save", ctx, mock.AnythingOfType("*bulk.ImportHistory")).Return(nil)

		history, err := service.CreateHistory(
			ctx,
			tenantID,
			bulk.ImportEntityProducts,
			"products.csv",
			1024,
			bulk.ConflictModeSkip,
			userID,
		)

		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.Equal(t, bulk.ImportEntityProducts, history.EntityType)
		assert.Equal(t, "products.csv", history.FileName)
		assert.Equal(t, bulk.ImportStatusPending, history.Status)
		repo.AssertExpectations(t)
	})

	t.Run("invalid entity type", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		_, err := service.CreateHistory(
			ctx,
			tenantID,
			bulk.ImportEntityType("invalid"),
			"products.csv",
			1024,
			bulk.ConflictModeSkip,
			userID,
		)

		require.Error(t, err)
		repo.AssertNotCalled(t, "Save")
	})
}

func TestImportHistoryService_StartProcessing(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)

		repo.On("FindByID", ctx, tenantID, history.ID).Return(history, nil)
		repo.On("Save", ctx, mock.AnythingOfType("*bulk.ImportHistory")).Return(nil)

		err := service.StartProcessing(ctx, tenantID, history.ID, 100)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestImportHistoryService_CompleteImport(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)
		_ = history.StartProcessing(100)

		repo.On("FindByID", ctx, tenantID, history.ID).Return(history, nil)
		repo.On("Save", ctx, mock.AnythingOfType("*bulk.ImportHistory")).Return(nil)

		errors := []csvimport.RowError{
			{Row: 1, Column: "name", Code: "ERR_REQUIRED", Message: "Field is required"},
		}

		err := service.CompleteImport(ctx, tenantID, history.ID, 95, 5, 0, 0, errors)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestImportHistoryService_FailImport(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)

		repo.On("FindByID", ctx, tenantID, history.ID).Return(history, nil)
		repo.On("Save", ctx, mock.AnythingOfType("*bulk.ImportHistory")).Return(nil)

		errors := []csvimport.RowError{
			{Row: 0, Code: "ERR_FILE", Message: "Invalid file format"},
		}

		err := service.FailImport(ctx, tenantID, history.ID, errors)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestImportHistoryService_CancelImport(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)

		repo.On("FindByID", ctx, tenantID, history.ID).Return(history, nil)
		repo.On("Save", ctx, mock.AnythingOfType("*bulk.ImportHistory")).Return(nil)

		err := service.CancelImport(ctx, tenantID, history.ID)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestImportHistoryService_ListHistory(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success with filters", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)
		result := &bulk.ImportHistoryListResult{
			Items:      []*bulk.ImportHistory{history},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		}

		repo.On("FindAll", ctx, tenantID, mock.AnythingOfType("bulk.ImportHistoryFilter"), 1, 20).Return(result, nil)

		filter := ListHistoryFilter{
			EntityType: "products",
			Status:     "completed",
		}

		res, err := service.ListHistory(ctx, tenantID, filter, 1, 20)

		require.NoError(t, err)
		assert.Equal(t, int64(1), res.TotalCount)
		assert.Len(t, res.Items, 1)
		repo.AssertExpectations(t)
	})

	t.Run("with date filters", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		result := &bulk.ImportHistoryListResult{
			Items:      []*bulk.ImportHistory{},
			TotalCount: 0,
			Page:       1,
			PageSize:   20,
		}

		repo.On("FindAll", ctx, tenantID, mock.AnythingOfType("bulk.ImportHistoryFilter"), 1, 20).Return(result, nil)

		startFrom := time.Now().Add(-24 * time.Hour)
		startTo := time.Now()
		filter := ListHistoryFilter{
			StartedFrom: &startFrom,
			StartedTo:   &startTo,
		}

		res, err := service.ListHistory(ctx, tenantID, filter, 1, 20)

		require.NoError(t, err)
		assert.NotNil(t, res)
		repo.AssertExpectations(t)
	})
}

func TestImportHistoryService_GetErrorsCSV(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)
		_ = history.StartProcessing(100)
		errors := []bulk.ImportErrorDetail{
			{Row: 1, Column: "name", Code: "ERR_REQUIRED", Message: "Field is required"},
			{Row: 2, Column: "price", Code: "ERR_TYPE", Message: "Invalid number", Value: "abc"},
		}
		_ = history.Complete(98, 2, 0, 0, errors)

		repo.On("FindByID", ctx, tenantID, history.ID).Return(history, nil)

		csv, fileName, err := service.GetErrorsCSV(ctx, tenantID, history.ID)

		require.NoError(t, err)
		assert.Contains(t, csv, "Row,Column,Error Code,Error Message,Value")
		assert.Contains(t, csv, "1,name,ERR_REQUIRED,Field is required")
		assert.Contains(t, csv, "2,price,ERR_TYPE,Invalid number,abc")
		assert.Contains(t, fileName, "import_errors_products_")
		repo.AssertExpectations(t)
	})

	t.Run("no errors to export", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)
		_ = history.StartProcessing(100)
		_ = history.Complete(100, 0, 0, 0, nil)

		repo.On("FindByID", ctx, tenantID, history.ID).Return(history, nil)

		_, _, err := service.GetErrorsCSV(ctx, tenantID, history.ID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no errors to export")
		repo.AssertExpectations(t)
	})

	t.Run("csv escaping", func(t *testing.T) {
		repo := new(MockImportHistoryRepository)
		service := NewImportHistoryService(repo)

		history := createTestHistoryEntity(tenantID, userID)
		_ = history.StartProcessing(100)
		errors := []bulk.ImportErrorDetail{
			{Row: 1, Column: "description", Code: "ERR", Message: "Message with \"quotes\" and,comma", Value: "value\nnewline"},
		}
		_ = history.Complete(99, 1, 0, 0, errors)

		repo.On("FindByID", ctx, tenantID, history.ID).Return(history, nil)

		csv, _, err := service.GetErrorsCSV(ctx, tenantID, history.ID)

		require.NoError(t, err)
		// Verify CSV escaping works
		assert.Contains(t, csv, "\"Message with \"\"quotes\"\" and,comma\"")
		repo.AssertExpectations(t)
	})
}

func TestEscapeCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"simple", "hello", "hello"},
		{"with comma", "hello,world", "\"hello,world\""},
		{"with newline", "hello\nworld", "\"hello\nworld\""},
		{"with quotes", "say \"hello\"", "\"say \"\"hello\"\"\""},
		{"with all", "say \"hello\",\nworld", "\"say \"\"hello\"\",\nworld\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeCSV(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create a test history entity
func createTestHistoryEntity(tenantID, userID uuid.UUID) *bulk.ImportHistory {
	history, _ := bulk.NewImportHistory(
		tenantID,
		bulk.ImportEntityProducts,
		"test.csv",
		1024,
		bulk.ConflictModeSkip,
		userID,
	)
	return history
}
