package importapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/bulk"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
)

// ImportHistoryService manages import history tracking and retrieval
type ImportHistoryService struct {
	historyRepo bulk.ImportHistoryRepository
}

// NewImportHistoryService creates a new ImportHistoryService
func NewImportHistoryService(historyRepo bulk.ImportHistoryRepository) *ImportHistoryService {
	return &ImportHistoryService{
		historyRepo: historyRepo,
	}
}

// CreateHistory creates a new import history record
func (s *ImportHistoryService) CreateHistory(
	ctx context.Context,
	tenantID uuid.UUID,
	entityType bulk.ImportEntityType,
	fileName string,
	fileSize int64,
	conflictMode bulk.ConflictMode,
	importedBy uuid.UUID,
) (*bulk.ImportHistory, error) {
	history, err := bulk.NewImportHistory(
		tenantID,
		entityType,
		fileName,
		fileSize,
		conflictMode,
		importedBy,
	)
	if err != nil {
		return nil, err
	}

	if err := s.historyRepo.Save(ctx, history); err != nil {
		return nil, fmt.Errorf("failed to save import history: %w", err)
	}

	return history, nil
}

// StartProcessing marks an import as started
func (s *ImportHistoryService) StartProcessing(
	ctx context.Context,
	tenantID uuid.UUID,
	historyID uuid.UUID,
	totalRows int,
) error {
	history, err := s.historyRepo.FindByID(ctx, tenantID, historyID)
	if err != nil {
		return err
	}

	if err := history.StartProcessing(totalRows); err != nil {
		return err
	}

	return s.historyRepo.Save(ctx, history)
}

// CompleteImport marks an import as completed with results
func (s *ImportHistoryService) CompleteImport(
	ctx context.Context,
	tenantID uuid.UUID,
	historyID uuid.UUID,
	successRows, errorRows, skippedRows, updatedRows int,
	errors []csvimport.RowError,
) error {
	history, err := s.historyRepo.FindByID(ctx, tenantID, historyID)
	if err != nil {
		return err
	}

	// Convert csvimport.RowError to bulk.ImportErrorDetail
	errorDetails := make([]bulk.ImportErrorDetail, len(errors))
	for i, e := range errors {
		errorDetails[i] = bulk.ImportErrorDetail{
			Row:     e.Row,
			Column:  e.Column,
			Code:    e.Code,
			Message: e.Message,
			Value:   e.Value,
		}
	}

	if err := history.Complete(successRows, errorRows, skippedRows, updatedRows, errorDetails); err != nil {
		return err
	}

	return s.historyRepo.Save(ctx, history)
}

// FailImport marks an import as failed
func (s *ImportHistoryService) FailImport(
	ctx context.Context,
	tenantID uuid.UUID,
	historyID uuid.UUID,
	errors []csvimport.RowError,
) error {
	history, err := s.historyRepo.FindByID(ctx, tenantID, historyID)
	if err != nil {
		return err
	}

	// Convert csvimport.RowError to bulk.ImportErrorDetail
	errorDetails := make([]bulk.ImportErrorDetail, len(errors))
	for i, e := range errors {
		errorDetails[i] = bulk.ImportErrorDetail{
			Row:     e.Row,
			Column:  e.Column,
			Code:    e.Code,
			Message: e.Message,
			Value:   e.Value,
		}
	}

	if err := history.Fail(errorDetails); err != nil {
		return err
	}

	return s.historyRepo.Save(ctx, history)
}

// CancelImport marks an import as cancelled
func (s *ImportHistoryService) CancelImport(
	ctx context.Context,
	tenantID uuid.UUID,
	historyID uuid.UUID,
) error {
	history, err := s.historyRepo.FindByID(ctx, tenantID, historyID)
	if err != nil {
		return err
	}

	if err := history.Cancel(); err != nil {
		return err
	}

	return s.historyRepo.Save(ctx, history)
}

// GetHistory retrieves a specific import history by ID
func (s *ImportHistoryService) GetHistory(
	ctx context.Context,
	tenantID, historyID uuid.UUID,
) (*bulk.ImportHistory, error) {
	return s.historyRepo.FindByID(ctx, tenantID, historyID)
}

// ListHistoryFilter defines the filter options for listing import histories
type ListHistoryFilter struct {
	EntityType  string     // Filter by entity type
	Status      string     // Filter by status
	ImportedBy  *uuid.UUID // Filter by user who imported
	StartedFrom *time.Time // Filter by start time (from)
	StartedTo   *time.Time // Filter by start time (to)
}

// ListHistory retrieves import history with pagination and filtering
func (s *ImportHistoryService) ListHistory(
	ctx context.Context,
	tenantID uuid.UUID,
	filter ListHistoryFilter,
	page, pageSize int,
) (*bulk.ImportHistoryListResult, error) {
	// Convert string filters to typed filters
	repoFilter := bulk.ImportHistoryFilter{
		ImportedBy:  filter.ImportedBy,
		StartedFrom: filter.StartedFrom,
		StartedTo:   filter.StartedTo,
	}

	if filter.EntityType != "" {
		entityType := bulk.ImportEntityType(filter.EntityType)
		if entityType.IsValid() {
			repoFilter.EntityType = &entityType
		}
	}

	if filter.Status != "" {
		status := bulk.ImportStatus(filter.Status)
		if status.IsValid() {
			repoFilter.Status = &status
		}
	}

	return s.historyRepo.FindAll(ctx, tenantID, repoFilter, page, pageSize)
}

// GetErrorsCSV generates a CSV string of error details for download
func (s *ImportHistoryService) GetErrorsCSV(
	ctx context.Context,
	tenantID, historyID uuid.UUID,
) (string, string, error) {
	history, err := s.historyRepo.FindByID(ctx, tenantID, historyID)
	if err != nil {
		return "", "", err
	}

	if len(history.ErrorDetails) == 0 {
		return "", "", fmt.Errorf("no errors to export")
	}

	// Generate CSV content
	var sb strings.Builder
	sb.WriteString("Row,Column,Error Code,Error Message,Value\n")

	for _, e := range history.ErrorDetails {
		// Escape CSV values
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s\n",
			e.Row,
			escapeCSV(e.Column),
			escapeCSV(e.Code),
			escapeCSV(e.Message),
			escapeCSV(e.Value),
		))
	}

	// Generate filename
	fileName := fmt.Sprintf("import_errors_%s_%s.csv",
		history.EntityType,
		history.ID.String()[:8],
	)

	return sb.String(), fileName, nil
}

// escapeCSV escapes a string for CSV output
func escapeCSV(s string) string {
	if s == "" {
		return ""
	}
	// If string contains comma, newline, or double quote, wrap in quotes
	if strings.ContainsAny(s, ",\"\n\r") {
		// Escape double quotes by doubling them
		escaped := strings.ReplaceAll(s, "\"", "\"\"")
		return "\"" + escaped + "\""
	}
	return s
}

// DeleteHistory deletes an import history record
func (s *ImportHistoryService) DeleteHistory(
	ctx context.Context,
	tenantID, historyID uuid.UUID,
) error {
	return s.historyRepo.Delete(ctx, tenantID, historyID)
}

// GetPendingImports retrieves all pending/processing imports for recovery
func (s *ImportHistoryService) GetPendingImports(
	ctx context.Context,
	tenantID uuid.UUID,
) ([]*bulk.ImportHistory, error) {
	return s.historyRepo.FindPending(ctx, tenantID)
}
