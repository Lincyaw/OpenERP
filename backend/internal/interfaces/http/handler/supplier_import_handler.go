package handler

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	importapp "github.com/erp/backend/internal/application/import"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SupplierImportHandler handles supplier-specific import operations
type SupplierImportHandler struct {
	BaseHandler
	importService *importapp.SupplierImportService
	sessionStore  csvimport.SessionStore
	// validRows stores validated rows for import
	validRowsStore     map[uuid.UUID][]*csvimport.Row
	validRowsStoreMu   sync.RWMutex
	validRowsStoreTTL  time.Duration
	validRowsCleanupCh chan struct{}
}

// NewSupplierImportHandler creates a new SupplierImportHandler
func NewSupplierImportHandler(
	supplierRepo partner.SupplierRepository,
	eventBus shared.EventPublisher,
) *SupplierImportHandler {
	importService := importapp.NewSupplierImportService(supplierRepo, eventBus)
	sessionStore := csvimport.NewInMemorySessionStore(15 * time.Minute)

	h := &SupplierImportHandler{
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
func (h *SupplierImportHandler) cleanupValidRowsStore() {
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
func (h *SupplierImportHandler) Stop() {
	close(h.validRowsCleanupCh)
}

// SupplierImportRequest represents the request to import suppliers
type SupplierImportRequest struct {
	ValidationID string `json:"validation_id" binding:"required"`
	ConflictMode string `json:"conflict_mode" binding:"required,oneof=skip update fail"`
}

// SupplierImportResponse represents the response from supplier import
// @Description Response from supplier bulk import operation
type SupplierImportResponse struct {
	TotalRows    int                  `json:"total_rows" example:"100"`
	ImportedRows int                  `json:"imported_rows" example:"95"`
	UpdatedRows  int                  `json:"updated_rows" example:"3"`
	SkippedRows  int                  `json:"skipped_rows" example:"2"`
	ErrorRows    int                  `json:"error_rows" example:"0"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty" example:"false"`
	TotalErrors  int                  `json:"total_errors,omitempty" example:"0"`
}

// SupplierValidationResponse represents the response from supplier import validation
// @Description Response from supplier CSV validation
type SupplierValidationResponse struct {
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

// ValidateSuppliers godoc
//
//	@Summary		Validate supplier CSV file for import
//	@Description	Validates a supplier CSV file for import without actually importing the data
//	@Tags			import
//	@ID				validateSupplierImport
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			file		formData	file	true	"CSV file to validate"
//	@Success		200			{object}	APIResponse[SupplierValidationResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		413			{object}	dto.ErrorResponse
//	@Failure		415			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/suppliers/validate [post]
func (h *SupplierImportHandler) ValidateSuppliers(c *gin.Context) {
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
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntitySuppliers, header.Filename, header.Size)

	// Get validation rules from service
	rules := h.importService.GetValidationRules()

	// Create processor with uniqueness lookup
	processor := csvimport.NewImportProcessor(
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
	response := SupplierValidationResponse{
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

// ImportSuppliers godoc
//
//	@Summary		Import suppliers from validated CSV
//	@Description	Imports suppliers from a previously validated CSV file
//	@Tags			import
//	@ID				importSuppliers
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			request		body		SupplierImportRequest	true	"Import request"
//	@Success		200			{object}	APIResponse[SupplierImportResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/suppliers [post]
func (h *SupplierImportHandler) ImportSuppliers(c *gin.Context) {
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

	var req SupplierImportRequest
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

	// Import suppliers
	result, err := h.importService.Import(ctx, tenantID, userID, session, validRows, conflictMode)
	if err != nil {
		if domainErr, ok := err.(*shared.DomainError); ok {
			h.Error(c, http.StatusUnprocessableEntity, domainErr.Code, domainErr.Message)
			return
		}
		h.InternalError(c, "failed to import suppliers: "+err.Error())
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
	response := SupplierImportResponse{
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

// RegisterRoutes registers all supplier import routes
func (h *SupplierImportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	imports := rg.Group("/import")
	{
		imports.POST("/suppliers/validate", h.ValidateSuppliers)
		imports.POST("/suppliers", h.ImportSuppliers)
	}
}
