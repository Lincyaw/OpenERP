package bulk

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportEntityType_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		entityType ImportEntityType
		want       bool
	}{
		{"products", ImportEntityProducts, true},
		{"customers", ImportEntityCustomers, true},
		{"suppliers", ImportEntitySuppliers, true},
		{"inventory", ImportEntityInventory, true},
		{"categories", ImportEntityCategories, true},
		{"invalid", ImportEntityType("invalid"), false},
		{"empty", ImportEntityType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.entityType.IsValid())
		})
	}
}

func TestImportStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status ImportStatus
		want   bool
	}{
		{"pending", ImportStatusPending, true},
		{"processing", ImportStatusProcessing, true},
		{"completed", ImportStatusCompleted, true},
		{"failed", ImportStatusFailed, true},
		{"cancelled", ImportStatusCancelled, true},
		{"invalid", ImportStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestImportStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status ImportStatus
		want   bool
	}{
		{"pending", ImportStatusPending, false},
		{"processing", ImportStatusProcessing, false},
		{"completed", ImportStatusCompleted, true},
		{"failed", ImportStatusFailed, true},
		{"cancelled", ImportStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsTerminal())
		})
	}
}

func TestConflictMode_IsValid(t *testing.T) {
	tests := []struct {
		name string
		mode ConflictMode
		want bool
	}{
		{"skip", ConflictModeSkip, true},
		{"update", ConflictModeUpdate, true},
		{"fail", ConflictModeFail, true},
		{"invalid", ConflictMode("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.IsValid())
		})
	}
}

func TestNewImportHistory(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		history, err := NewImportHistory(
			tenantID,
			ImportEntityProducts,
			"products.csv",
			1024,
			ConflictModeSkip,
			userID,
		)

		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.Equal(t, tenantID, history.TenantID)
		assert.Equal(t, ImportEntityProducts, history.EntityType)
		assert.Equal(t, "products.csv", history.FileName)
		assert.Equal(t, int64(1024), history.FileSize)
		assert.Equal(t, ConflictModeSkip, history.ConflictMode)
		assert.Equal(t, ImportStatusPending, history.Status)
		assert.Equal(t, &userID, history.ImportedBy)
		assert.NotEqual(t, uuid.Nil, history.ID)
	})

	t.Run("invalid entity type", func(t *testing.T) {
		_, err := NewImportHistory(
			tenantID,
			ImportEntityType("invalid"),
			"products.csv",
			1024,
			ConflictModeSkip,
			userID,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid entity type")
	})

	t.Run("empty file name", func(t *testing.T) {
		_, err := NewImportHistory(
			tenantID,
			ImportEntityProducts,
			"",
			1024,
			ConflictModeSkip,
			userID,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "File name cannot be empty")
	})

	t.Run("negative file size", func(t *testing.T) {
		_, err := NewImportHistory(
			tenantID,
			ImportEntityProducts,
			"products.csv",
			-1,
			ConflictModeSkip,
			userID,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "File size cannot be negative")
	})

	t.Run("invalid conflict mode", func(t *testing.T) {
		_, err := NewImportHistory(
			tenantID,
			ImportEntityProducts,
			"products.csv",
			1024,
			ConflictMode("invalid"),
			userID,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid conflict mode")
	})
}

func TestImportHistory_StartProcessing(t *testing.T) {
	history := createTestHistory(t)

	t.Run("success", func(t *testing.T) {
		err := history.StartProcessing(100)

		require.NoError(t, err)
		assert.Equal(t, ImportStatusProcessing, history.Status)
		assert.Equal(t, 100, history.TotalRows)
		assert.NotNil(t, history.StartedAt)
	})

	t.Run("invalid state", func(t *testing.T) {
		history2 := createTestHistory(t)
		_ = history2.StartProcessing(100)

		err := history2.StartProcessing(100)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot start processing from state")
	})

	t.Run("negative total rows", func(t *testing.T) {
		history2 := createTestHistory(t)

		err := history2.StartProcessing(-1)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Total rows cannot be negative")
	})
}

func TestImportHistory_Complete(t *testing.T) {
	t.Run("success with no errors", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)

		err := history.Complete(95, 0, 5, 0, nil)

		require.NoError(t, err)
		assert.Equal(t, ImportStatusCompleted, history.Status)
		assert.Equal(t, 95, history.SuccessRows)
		assert.Equal(t, 0, history.ErrorRows)
		assert.Equal(t, 5, history.SkippedRows)
		assert.NotNil(t, history.CompletedAt)
	})

	t.Run("success with errors", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)

		errors := []ImportErrorDetail{
			{Row: 1, Column: "name", Code: "ERR_REQUIRED", Message: "Field is required"},
		}
		err := history.Complete(90, 5, 5, 0, errors)

		require.NoError(t, err)
		assert.Equal(t, ImportStatusCompleted, history.Status)
		assert.Len(t, history.ErrorDetails, 1)
	})

	t.Run("failed when all rows have errors", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)

		errors := []ImportErrorDetail{
			{Row: 1, Column: "name", Code: "ERR_REQUIRED", Message: "Field is required"},
		}
		err := history.Complete(0, 100, 0, 0, errors)

		require.NoError(t, err)
		assert.Equal(t, ImportStatusFailed, history.Status)
	})

	t.Run("invalid state", func(t *testing.T) {
		history := createTestHistory(t)

		err := history.Complete(90, 10, 0, 0, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot complete from state")
	})
}

func TestImportHistory_Fail(t *testing.T) {
	t.Run("success from pending", func(t *testing.T) {
		history := createTestHistory(t)

		errors := []ImportErrorDetail{
			{Row: 0, Code: "ERR_FILE", Message: "Invalid file format"},
		}
		err := history.Fail(errors)

		require.NoError(t, err)
		assert.Equal(t, ImportStatusFailed, history.Status)
		assert.Len(t, history.ErrorDetails, 1)
		assert.NotNil(t, history.CompletedAt)
	})

	t.Run("success from processing", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)

		err := history.Fail(nil)

		require.NoError(t, err)
		assert.Equal(t, ImportStatusFailed, history.Status)
	})

	t.Run("invalid state - already completed", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)
		_ = history.Complete(100, 0, 0, 0, nil)

		err := history.Fail(nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot fail from terminal state")
	})
}

func TestImportHistory_Cancel(t *testing.T) {
	t.Run("success from pending", func(t *testing.T) {
		history := createTestHistory(t)

		err := history.Cancel()

		require.NoError(t, err)
		assert.Equal(t, ImportStatusCancelled, history.Status)
		assert.NotNil(t, history.CompletedAt)
	})

	t.Run("success from processing", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)

		err := history.Cancel()

		require.NoError(t, err)
		assert.Equal(t, ImportStatusCancelled, history.Status)
	})

	t.Run("invalid state - already completed", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)
		_ = history.Complete(100, 0, 0, 0, nil)

		err := history.Cancel()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel from terminal state")
	})
}

func TestImportHistory_ErrorDetailsJSON(t *testing.T) {
	history := createTestHistory(t)

	t.Run("empty errors", func(t *testing.T) {
		json, err := history.ErrorDetailsJSON()

		require.NoError(t, err)
		assert.Equal(t, "[]", json)
	})

	t.Run("with errors", func(t *testing.T) {
		_ = history.StartProcessing(100)
		errors := []ImportErrorDetail{
			{Row: 1, Column: "name", Code: "ERR_REQUIRED", Message: "Field is required"},
		}
		_ = history.Complete(99, 1, 0, 0, errors)

		json, err := history.ErrorDetailsJSON()

		require.NoError(t, err)
		assert.Contains(t, json, "ERR_REQUIRED")
		assert.Contains(t, json, "name")
	})
}

func TestImportHistory_SetErrorDetailsFromJSON(t *testing.T) {
	history := createTestHistory(t)

	t.Run("empty string", func(t *testing.T) {
		err := history.SetErrorDetailsFromJSON("")

		require.NoError(t, err)
		assert.Empty(t, history.ErrorDetails)
	})

	t.Run("empty array", func(t *testing.T) {
		err := history.SetErrorDetailsFromJSON("[]")

		require.NoError(t, err)
		assert.Empty(t, history.ErrorDetails)
	})

	t.Run("valid JSON", func(t *testing.T) {
		json := `[{"row":1,"column":"name","code":"ERR_REQUIRED","message":"Field is required"}]`
		err := history.SetErrorDetailsFromJSON(json)

		require.NoError(t, err)
		require.Len(t, history.ErrorDetails, 1)
		assert.Equal(t, 1, history.ErrorDetails[0].Row)
		assert.Equal(t, "name", history.ErrorDetails[0].Column)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		err := history.SetErrorDetailsFromJSON("invalid")

		require.Error(t, err)
	})
}

func TestImportHistory_SuccessRate(t *testing.T) {
	t.Run("zero total rows", func(t *testing.T) {
		history := createTestHistory(t)

		assert.Equal(t, float64(0), history.SuccessRate())
	})

	t.Run("all successful", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)
		_ = history.Complete(100, 0, 0, 0, nil)

		assert.Equal(t, float64(100), history.SuccessRate())
	})

	t.Run("partial success", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)
		_ = history.Complete(80, 10, 10, 0, nil)

		assert.Equal(t, float64(80), history.SuccessRate())
	})

	t.Run("with updates", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)
		_ = history.Complete(50, 0, 0, 50, nil)

		assert.Equal(t, float64(100), history.SuccessRate())
	})
}

func TestImportHistory_Duration(t *testing.T) {
	t.Run("not started", func(t *testing.T) {
		history := createTestHistory(t)

		assert.Equal(t, time.Duration(0), history.Duration())
	})

	t.Run("completed", func(t *testing.T) {
		history := createTestHistory(t)
		_ = history.StartProcessing(100)

		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		_ = history.Complete(100, 0, 0, 0, nil)

		duration := history.Duration()
		assert.True(t, duration >= 10*time.Millisecond)
	})
}

func TestImportHistory_StatusChecks(t *testing.T) {
	t.Run("IsCompleted", func(t *testing.T) {
		history := createTestHistory(t)
		assert.False(t, history.IsCompleted())

		_ = history.StartProcessing(100)
		assert.False(t, history.IsCompleted())

		_ = history.Complete(100, 0, 0, 0, nil)
		assert.True(t, history.IsCompleted())
	})

	t.Run("IsFailed", func(t *testing.T) {
		history := createTestHistory(t)
		assert.False(t, history.IsFailed())

		_ = history.Fail(nil)
		assert.True(t, history.IsFailed())
	})

	t.Run("HasErrors", func(t *testing.T) {
		history := createTestHistory(t)
		assert.False(t, history.HasErrors())

		_ = history.StartProcessing(100)
		errors := []ImportErrorDetail{{Row: 1, Message: "Error"}}
		_ = history.Complete(99, 1, 0, 0, errors)
		assert.True(t, history.HasErrors())
	})
}

// Helper function to create a test history
func createTestHistory(t *testing.T) *ImportHistory {
	t.Helper()
	history, err := NewImportHistory(
		uuid.New(),
		ImportEntityProducts,
		"test.csv",
		1024,
		ConflictModeSkip,
		uuid.New(),
	)
	require.NoError(t, err)
	return history
}
