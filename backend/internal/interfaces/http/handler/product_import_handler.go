package handler

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	importapp "github.com/erp/backend/internal/application/import"
	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProductImportHandler handles product-specific import operations
type ProductImportHandler struct {
	BaseHandler
	importService *importapp.ProductImportService
	processor     *csvimport.ImportProcessor
	sessionStore  csvimport.SessionStore
	// validRows stores validated rows for import
	validRowsStore     map[uuid.UUID][]*csvimport.Row
	validRowsStoreMu   sync.RWMutex
	validRowsStoreTTL  time.Duration
	validRowsCleanupCh chan struct{}
}

// NewProductImportHandler creates a new ProductImportHandler
func NewProductImportHandler(
	productRepo catalog.ProductRepository,
	categoryRepo catalog.CategoryRepository,
	eventBus shared.EventPublisher,
) *ProductImportHandler {
	importService := importapp.NewProductImportService(productRepo, categoryRepo, eventBus)
	sessionStore := csvimport.NewInMemorySessionStore(15 * time.Minute)

	// Create processor with reference lookup
	processor := csvimport.NewImportProcessor(
		csvimport.WithReferenceLookup(func(refType, value string) (bool, error) {
			// This is a stub - actual lookup requires context
			// The real lookup happens in ValidateProducts
			return true, nil
		}),
	)

	h := &ProductImportHandler{
		importService:      importService,
		processor:          processor,
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
func (h *ProductImportHandler) cleanupValidRowsStore() {
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
func (h *ProductImportHandler) Stop() {
	close(h.validRowsCleanupCh)
}

// ProductImportRequest represents the request to import products
type ProductImportRequest struct {
	ValidationID string `json:"validation_id" binding:"required"`
	ConflictMode string `json:"conflict_mode" binding:"required,oneof=skip update fail"`
}

// ProductImportResponse represents the response from product import
// @Description Response from product bulk import operation
type ProductImportResponse struct {
	TotalRows    int                  `json:"total_rows" example:"100"`
	ImportedRows int                  `json:"imported_rows" example:"95"`
	UpdatedRows  int                  `json:"updated_rows" example:"3"`
	SkippedRows  int                  `json:"skipped_rows" example:"2"`
	ErrorRows    int                  `json:"error_rows" example:"0"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty" example:"false"`
	TotalErrors  int                  `json:"total_errors,omitempty" example:"0"`
}

// ProductValidationResponse represents the response from product import validation
// @Description Response from product CSV validation
type ProductValidationResponse struct {
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

// ValidateProducts godoc
//
//	@Summary		Validate product CSV file for import
//	@Description	Validates a product CSV file for import without actually importing the data
//	@Tags			import
//	@ID				validateProductImport
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			file		formData	file	true	"CSV file to validate"
//	@Success		200			{object}	APIResponse[ProductValidationResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		413			{object}	ErrorResponse
//	@Failure		415			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/products/validate [post]
func (h *ProductImportHandler) ValidateProducts(c *gin.Context) {
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
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityProducts, header.Filename, header.Size)

	// Get validation rules from service
	rules := h.importService.GetValidationRules()

	// Create processor with reference and uniqueness lookup
	processor := csvimport.NewImportProcessor(
		csvimport.WithReferenceLookup(func(refType, value string) (bool, error) {
			if refType == "category" {
				return h.importService.LookupCategory(ctx, tenantID, value)
			}
			return true, nil
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
	response := ProductValidationResponse{
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

// ImportProducts godoc
//
//	@Summary		Import products from validated CSV
//	@Description	Imports products from a previously validated CSV file
//	@Tags			import
//	@ID				importProducts
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			request		body		ProductImportRequest	true	"Import request"
//	@Success		200			{object}	APIResponse[ProductImportResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/products [post]
func (h *ProductImportHandler) ImportProducts(c *gin.Context) {
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

	var req ProductImportRequest
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

	// Import products
	result, err := h.importService.Import(ctx, tenantID, userID, session, validRows, conflictMode)
	if err != nil {
		if domainErr, ok := err.(*shared.DomainError); ok {
			h.Error(c, http.StatusUnprocessableEntity, domainErr.Code, domainErr.Message)
			return
		}
		h.InternalError(c, "failed to import products: "+err.Error())
		return
	}

	// Clean up valid rows
	h.validRowsStoreMu.Lock()
	delete(h.validRowsStore, validationID)
	h.validRowsStoreMu.Unlock()

	// Update session in store
	if err := h.sessionStore.Save(session); err != nil {
		// Log error but don't fail the import
	}

	// Build response
	response := ProductImportResponse{
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

// RegisterRoutes registers all product import routes
func (h *ProductImportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	imports := rg.Group("/import")
	{
		imports.POST("/products/validate", h.ValidateProducts)
		imports.POST("/products", h.ImportProducts)
	}
}
