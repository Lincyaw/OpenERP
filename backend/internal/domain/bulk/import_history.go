package bulk

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// ImportEntityType represents the type of entity being imported
type ImportEntityType string

const (
	ImportEntityProducts   ImportEntityType = "products"
	ImportEntityCustomers  ImportEntityType = "customers"
	ImportEntitySuppliers  ImportEntityType = "suppliers"
	ImportEntityInventory  ImportEntityType = "inventory"
	ImportEntityCategories ImportEntityType = "categories"
)

// IsValid checks if the entity type is valid
func (e ImportEntityType) IsValid() bool {
	switch e {
	case ImportEntityProducts, ImportEntityCustomers, ImportEntitySuppliers,
		ImportEntityInventory, ImportEntityCategories:
		return true
	}
	return false
}

// ImportStatus represents the status of an import operation
type ImportStatus string

const (
	ImportStatusPending    ImportStatus = "pending"
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
	ImportStatusCancelled  ImportStatus = "cancelled"
)

// IsValid checks if the status is valid
func (s ImportStatus) IsValid() bool {
	switch s {
	case ImportStatusPending, ImportStatusProcessing, ImportStatusCompleted,
		ImportStatusFailed, ImportStatusCancelled:
		return true
	}
	return false
}

// IsTerminal returns true if this is a terminal state
func (s ImportStatus) IsTerminal() bool {
	return s == ImportStatusCompleted || s == ImportStatusFailed || s == ImportStatusCancelled
}

// ConflictMode defines how to handle conflicts during import
type ConflictMode string

const (
	ConflictModeSkip   ConflictMode = "skip"
	ConflictModeUpdate ConflictMode = "update"
	ConflictModeFail   ConflictMode = "fail"
)

// IsValid checks if the conflict mode is valid
func (c ConflictMode) IsValid() bool {
	switch c {
	case ConflictModeSkip, ConflictModeUpdate, ConflictModeFail:
		return true
	}
	return false
}

// ImportErrorDetail represents a detailed error for a specific row
type ImportErrorDetail struct {
	Row     int    `json:"row"`
	Column  string `json:"column,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ImportHistory tracks the history and result of bulk import operations
type ImportHistory struct {
	shared.TenantAggregateRoot
	EntityType   ImportEntityType    `json:"entity_type"`
	FileName     string              `json:"file_name"`
	FileSize     int64               `json:"file_size"`
	TotalRows    int                 `json:"total_rows"`
	SuccessRows  int                 `json:"success_rows"`
	ErrorRows    int                 `json:"error_rows"`
	SkippedRows  int                 `json:"skipped_rows"`
	UpdatedRows  int                 `json:"updated_rows"`
	ConflictMode ConflictMode        `json:"conflict_mode"`
	Status       ImportStatus        `json:"status"`
	ErrorDetails []ImportErrorDetail `json:"error_details,omitempty"`
	ImportedBy   *uuid.UUID          `json:"imported_by,omitempty"`
	StartedAt    *time.Time          `json:"started_at,omitempty"`
	CompletedAt  *time.Time          `json:"completed_at,omitempty"`
}

// NewImportHistory creates a new import history record
func NewImportHistory(
	tenantID uuid.UUID,
	entityType ImportEntityType,
	fileName string,
	fileSize int64,
	conflictMode ConflictMode,
	importedBy uuid.UUID,
) (*ImportHistory, error) {
	if !entityType.IsValid() {
		return nil, shared.NewDomainError("INVALID_ENTITY_TYPE", fmt.Sprintf("Invalid entity type: %s", entityType))
	}
	if fileName == "" {
		return nil, shared.NewDomainError("INVALID_FILE_NAME", "File name cannot be empty")
	}
	if fileSize < 0 {
		return nil, shared.NewDomainError("INVALID_FILE_SIZE", "File size cannot be negative")
	}
	if !conflictMode.IsValid() {
		return nil, shared.NewDomainError("INVALID_CONFLICT_MODE", fmt.Sprintf("Invalid conflict mode: %s", conflictMode))
	}

	history := &ImportHistory{
		TenantAggregateRoot: shared.NewTenantAggregateRootWithCreator(tenantID, importedBy),
		EntityType:          entityType,
		FileName:            fileName,
		FileSize:            fileSize,
		ConflictMode:        conflictMode,
		Status:              ImportStatusPending,
		ErrorDetails:        make([]ImportErrorDetail, 0),
		ImportedBy:          &importedBy,
	}

	return history, nil
}

// StartProcessing marks the import as started
func (h *ImportHistory) StartProcessing(totalRows int) error {
	if h.Status != ImportStatusPending {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot start processing from state: %s", h.Status))
	}
	if totalRows < 0 {
		return shared.NewDomainError("INVALID_TOTAL_ROWS", "Total rows cannot be negative")
	}

	h.Status = ImportStatusProcessing
	h.TotalRows = totalRows
	now := time.Now()
	h.StartedAt = &now
	h.UpdatedAt = now
	h.IncrementVersion()

	return nil
}

// Complete marks the import as successfully completed
func (h *ImportHistory) Complete(successRows, errorRows, skippedRows, updatedRows int, errors []ImportErrorDetail) error {
	if h.Status != ImportStatusProcessing {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot complete from state: %s", h.Status))
	}

	// Calculate status based on errors
	status := ImportStatusCompleted
	if errorRows > 0 && successRows == 0 && updatedRows == 0 {
		status = ImportStatusFailed
	}

	h.Status = status
	h.SuccessRows = successRows
	h.ErrorRows = errorRows
	h.SkippedRows = skippedRows
	h.UpdatedRows = updatedRows
	h.ErrorDetails = errors
	now := time.Now()
	h.CompletedAt = &now
	h.UpdatedAt = now
	h.IncrementVersion()

	return nil
}

// Fail marks the import as failed
func (h *ImportHistory) Fail(errors []ImportErrorDetail) error {
	if h.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot fail from terminal state: %s", h.Status))
	}

	h.Status = ImportStatusFailed
	h.ErrorDetails = errors
	now := time.Now()
	h.CompletedAt = &now
	h.UpdatedAt = now
	h.IncrementVersion()

	return nil
}

// Cancel marks the import as cancelled
func (h *ImportHistory) Cancel() error {
	if h.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot cancel from terminal state: %s", h.Status))
	}

	h.Status = ImportStatusCancelled
	now := time.Now()
	h.CompletedAt = &now
	h.UpdatedAt = now
	h.IncrementVersion()

	return nil
}

// IsCompleted returns true if the import is completed (successful or with partial errors)
func (h *ImportHistory) IsCompleted() bool {
	return h.Status == ImportStatusCompleted
}

// IsFailed returns true if the import failed completely
func (h *ImportHistory) IsFailed() bool {
	return h.Status == ImportStatusFailed
}

// HasErrors returns true if there are any errors
func (h *ImportHistory) HasErrors() bool {
	return len(h.ErrorDetails) > 0
}

// ErrorDetailsJSON returns the error details as a JSON string
func (h *ImportHistory) ErrorDetailsJSON() (string, error) {
	if len(h.ErrorDetails) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(h.ErrorDetails)
	if err != nil {
		return "", fmt.Errorf("failed to marshal error details: %w", err)
	}
	return string(data), nil
}

// SetErrorDetailsFromJSON parses error details from a JSON string
func (h *ImportHistory) SetErrorDetailsFromJSON(jsonStr string) error {
	if jsonStr == "" || jsonStr == "[]" {
		h.ErrorDetails = make([]ImportErrorDetail, 0)
		return nil
	}
	var errors []ImportErrorDetail
	if err := json.Unmarshal([]byte(jsonStr), &errors); err != nil {
		return fmt.Errorf("failed to unmarshal error details: %w", err)
	}
	h.ErrorDetails = errors
	return nil
}

// SuccessRate returns the success rate as a percentage (0-100)
func (h *ImportHistory) SuccessRate() float64 {
	if h.TotalRows == 0 {
		return 0
	}
	return float64(h.SuccessRows+h.UpdatedRows) / float64(h.TotalRows) * 100
}

// Duration returns the duration of the import operation
func (h *ImportHistory) Duration() time.Duration {
	if h.StartedAt == nil {
		return 0
	}
	endTime := h.CompletedAt
	if endTime == nil {
		endTime = &time.Time{}
		*endTime = time.Now()
	}
	return endTime.Sub(*h.StartedAt)
}
