package handler

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	importapp "github.com/erp/backend/internal/application/import"
	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InventoryImportHandler handles inventory-specific import operations
type InventoryImportHandler struct {
	BaseHandler
	importService *importapp.InventoryImportService
	sessionStore  csvimport.SessionStore
	// validRows stores validated rows for import
	validRowsStore     map[uuid.UUID][]*csvimport.Row
	validRowsStoreMu   sync.RWMutex
	validRowsStoreTTL  time.Duration
	validRowsCleanupCh chan struct{}
}

// NewInventoryImportHandler creates a new InventoryImportHandler
func NewInventoryImportHandler(
	inventoryRepo inventory.InventoryItemRepository,
	transactionRepo inventory.InventoryTransactionRepository,
	productRepo catalog.ProductRepository,
	warehouseRepo partner.WarehouseRepository,
	eventBus shared.EventPublisher,
) *InventoryImportHandler {
	importService := importapp.NewInventoryImportService(
		inventoryRepo,
		transactionRepo,
		productRepo,
		warehouseRepo,
		eventBus,
	)
	sessionStore := csvimport.NewInMemorySessionStore(15 * time.Minute)

	h := &InventoryImportHandler{
		importService:      importService,
		sessionStore:       sessionStore,
		validRowsStore:     make(map[uuid.UUID][]*csvimport.Row),
		validRowsStoreTTL:  15 * time.Minute,
		validRowsCleanupCh: make(chan struct{}),
	}

	// Start background cleanup
	go h.cleanupValidRowsStore()

	return h
}

// cleanupValidRowsStore periodically removes expired valid rows
func (h *InventoryImportHandler) cleanupValidRowsStore() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.validRowsStoreMu.Lock()
			// Clean up entries where session no longer exists
			for sessionID := range h.validRowsStore {
				session, _ := h.sessionStore.Get(sessionID)
				if session == nil {
					delete(h.validRowsStore, sessionID)
				}
			}
			h.validRowsStoreMu.Unlock()
		case <-h.validRowsCleanupCh:
			return
		}
	}
}

// Stop stops the background cleanup goroutine
func (h *InventoryImportHandler) Stop() {
	close(h.validRowsCleanupCh)
}

// InventoryImportRequest represents the request to import inventory
type InventoryImportRequest struct {
	ValidationID string `json:"validation_id" binding:"required"`
	ConflictMode string `json:"conflict_mode" binding:"required,oneof=skip update fail"`
}

// InventoryImportResponse represents the response from inventory import
// @Description Response from inventory bulk import operation
type InventoryImportResponse struct {
	TotalRows    int                  `json:"total_rows" example:"100"`
	ImportedRows int                  `json:"imported_rows" example:"95"`
	UpdatedRows  int                  `json:"updated_rows" example:"3"`
	SkippedRows  int                  `json:"skipped_rows" example:"2"`
	ErrorRows    int                  `json:"error_rows" example:"0"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty" example:"false"`
	TotalErrors  int                  `json:"total_errors,omitempty" example:"0"`
}

// InventoryValidationResponse represents the response from inventory import validation
// @Description Response from inventory CSV validation
type InventoryValidationResponse struct {
	ValidationID string               `json:"validation_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TotalRows    int                  `json:"total_rows" example:"100"`
	ValidRows    int                  `json:"valid_rows" example:"98"`
	ErrorRows    int                  `json:"error_rows" example:"2"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	Preview      []map[string]any     `json:"preview,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty"`
	TotalErrors  int                  `json:"total_errors,omitempty"`
}

// ValidateInventory godoc
//
//	@Summary		Validate inventory CSV file for import
//	@Description	Validates an inventory CSV file for import without actually importing the data. CRITICAL: This is for initial data migration only, not for regular stock operations.
//	@Tags			import
//	@ID				validateInventoryImport
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			file		formData	file	true	"CSV file to validate"
//	@Success		200			{object}	APIResponse[InventoryValidationResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		413			{object}	dto.ErrorResponse
//	@Failure		415			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/inventory/validate [post]
func (h *InventoryImportHandler) ValidateInventory(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > maxImportFileSize {
		h.Error(c, http.StatusRequestEntityTooLarge, dto.ErrCodeValidation, "file exceeds maximum size of 10MB")
		return
	}

	// Check content type
	contentType := header.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/csv" && contentType != "application/octet-stream" &&
		contentType != "text/plain" && contentType != "application/vnd.ms-excel" {
		h.Error(c, http.StatusUnsupportedMediaType, dto.ErrCodeValidation, "file must be a CSV file")
		return
	}

	// Create import session
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityInventory, header.Filename, header.Size)

	// Get validation rules from service
	rules := h.importService.GetValidationRules()

	// Create processor with reference lookup for product and warehouse validation
	processor := csvimport.NewImportProcessor(
		csvimport.WithReferenceLookup(func(refType, value string) (bool, error) {
			return h.importService.LookupReference(ctx, tenantID, refType, value)
		}),
		csvimport.WithUniqueLookup(func(entityType, field, value string) (bool, error) {
			return h.importService.LookupUnique(ctx, tenantID, field, value)
		}),
	)

	// Run validation
	result, err := processor.Validate(ctx, session, file, rules)
	if err != nil {
		if err == csvimport.ErrEmptyFile {
			h.BadRequest(c, "CSV file is empty")
			return
		}
		if err == csvimport.ErrInvalidEncoding {
			h.BadRequest(c, "CSV file has invalid encoding, must be UTF-8")
			return
		}
		if err == csvimport.ErrMissingHeader {
			h.BadRequest(c, "CSV file is missing header row")
			return
		}
		h.InternalError(c, "failed to validate file: "+err.Error())
		return
	}

	// Parse file again to get valid rows (since we consumed the reader)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Printf("failed to seek file: %v", err)
		h.InternalError(c, "Failed to process file")
		return
	}
	parser, err := csvimport.NewCSVParser(file)
	var warnings []string
	const maxWarnings = 100
	if err == nil {
		if err := parser.ParseHeader(); err == nil {
			var validRows []*csvimport.Row
			// Build error row index for O(1) lookup
			errorRows := make(map[int]bool)
			for _, e := range result.Errors {
				errorRows[e.Row] = true
			}

			for {
				row, err := parser.ReadRow()
				if err == io.EOF {
					break
				}
				if err != nil {
					continue
				}
				if row.IsEmpty() {
					continue
				}

				// Only add rows without errors
				if !errorRows[row.LineNumber] {
					validRows = append(validRows, row)
					// Check for warnings (with limit)
					if len(warnings) < maxWarnings {
						rowWarnings := h.importService.ValidateWithWarnings(row)
						for _, w := range rowWarnings {
							if len(warnings) < maxWarnings {
								warnings = append(warnings, w)
							}
						}
					}
				}
			}

			// Store valid rows for import
			if len(validRows) > 0 {
				h.validRowsStoreMu.Lock()
				h.validRowsStore[session.ID] = validRows
				h.validRowsStoreMu.Unlock()
			}
		}
	}

	// Save session
	if err := h.sessionStore.Save(session); err != nil {
		h.InternalError(c, "failed to save import session")
		return
	}

	// Build response
	response := InventoryValidationResponse{
		ValidationID: result.ValidationID,
		TotalRows:    result.TotalRows,
		ValidRows:    result.ValidRows,
		ErrorRows:    result.ErrorRows,
		Errors:       result.Errors,
		Preview:      result.Preview,
		Warnings:     warnings,
		IsTruncated:  result.IsTruncated,
		TotalErrors:  result.TotalErrors,
	}

	h.Success(c, response)
}

// ImportInventory godoc
//
//	@Summary		Import inventory from validated CSV
//	@Description	Imports inventory from a previously validated CSV file. CRITICAL: This is for initial data migration only, not for regular stock operations. Requires admin permission.
//	@Tags			import
//	@ID				importInventory
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					true	"Tenant ID"
//	@Param			request		body		InventoryImportRequest	true	"Import request"
//	@Success		200			{object}	APIResponse[InventoryImportResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		403			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/inventory [post]
func (h *InventoryImportHandler) ImportInventory(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		h.BadRequest(c, "Invalid user ID")
		return
	}

	var req InventoryImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Parse validation ID
	validationID, err := uuid.Parse(req.ValidationID)
	if err != nil {
		h.BadRequest(c, "Invalid validation_id")
		return
	}

	// Parse conflict mode
	conflictMode := importapp.ConflictMode(req.ConflictMode)
	if !conflictMode.IsValid() {
		h.BadRequest(c, "Invalid conflict_mode, must be one of: skip, update, fail")
		return
	}

	// Get session
	session, err := h.sessionStore.Get(validationID)
	if err != nil {
		h.InternalError(c, "failed to retrieve session")
		return
	}

	if session == nil {
		h.NotFound(c, "Import session not found or expired")
		return
	}

	// Verify tenant ownership
	if session.TenantID != tenantID {
		h.NotFound(c, "Import session not found or expired")
		return
	}

	// Verify session state
	if session.State != csvimport.StateValidated {
		h.BadRequest(c, "Session must be validated before import. Current state: "+string(session.State))
		return
	}

	// Get valid rows
	h.validRowsStoreMu.RLock()
	validRows := h.validRowsStore[validationID]
	h.validRowsStoreMu.RUnlock()

	if len(validRows) == 0 {
		h.BadRequest(c, "No valid rows found for import. Please re-validate the file.")
		return
	}

	// Import inventory
	result, err := h.importService.Import(ctx, tenantID, userID, session, validRows, conflictMode)
	if err != nil {
		if domainErr, ok := err.(*shared.DomainError); ok {
			h.Error(c, http.StatusUnprocessableEntity, domainErr.Code, domainErr.Message)
			return
		}
		h.InternalError(c, "failed to import inventory: "+err.Error())
		return
	}

	// Clean up valid rows
	h.validRowsStoreMu.Lock()
	delete(h.validRowsStore, validationID)
	h.validRowsStoreMu.Unlock()

	// Update session in store
	if err := h.sessionStore.Save(session); err != nil {
		log.Printf("ERROR: failed to update session %s after import: %v", session.ID, err)
	}

	// Build response
	response := InventoryImportResponse{
		TotalRows:    result.TotalRows,
		ImportedRows: result.ImportedRows,
		UpdatedRows:  result.UpdatedRows,
		SkippedRows:  result.SkippedRows,
		ErrorRows:    result.ErrorRows,
		Errors:       result.Errors,
		IsTruncated:  result.IsTruncated,
		TotalErrors:  result.TotalErrors,
	}

	h.Success(c, response)
}

// RegisterRoutes registers all inventory import routes
func (h *InventoryImportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	imports := rg.Group("/import")
	{
		imports.POST("/inventory/validate", h.ValidateInventory)
		imports.POST("/inventory", h.ImportInventory)
	}
}
