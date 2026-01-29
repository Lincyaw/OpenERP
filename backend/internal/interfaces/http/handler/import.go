package handler

import (
	"net/http"
	"time"

	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	csvimport "github.com/erp/backend/internal/infrastructure/import"
)

const (
	// Maximum file size for imports (10MB)
	maxImportFileSize = 10 * 1024 * 1024
)

// ImportHandler handles import-related API endpoints
type ImportHandler struct {
	BaseHandler
	processor    *csvimport.ImportProcessor
	sessionStore csvimport.SessionStore
}

// NewImportHandler creates a new ImportHandler
func NewImportHandler() *ImportHandler {
	return &ImportHandler{
		processor:    csvimport.NewImportProcessor(),
		sessionStore: csvimport.NewInMemorySessionStore(15 * time.Minute),
	}
}

// ValidateImportRequest represents the request for import validation
type ValidateImportRequest struct {
	EntityType string `form:"entity_type" binding:"required"`
}

// ValidationResponse represents the response from import validation
// @Description Response from CSV import validation
type ValidationResponse struct {
	ValidationID string               `json:"validation_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TotalRows    int                  `json:"total_rows" example:"100"`
	ValidRows    int                  `json:"valid_rows" example:"98"`
	ErrorRows    int                  `json:"error_rows" example:"2"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	Preview      []map[string]any     `json:"preview,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty"`
	TotalErrors  int                  `json:"total_errors,omitempty"`
}

// Validate godoc
//
//	@Summary		Validate CSV file for import
//	@Description	Validates a CSV file for import without actually importing the data
//	@Tags			import
//	@ID				validateImport
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			file		formData	file	true	"CSV file to validate"
//	@Param			entity_type	formData	string	true	"Entity type"	Enums(products, customers, suppliers, inventory)
//	@Success		200			{object}	APIResponse[ValidationResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		413			{object}	dto.ErrorResponse
//	@Failure		415			{object}	dto.ErrorResponse
//	@Failure		422			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/validate [post]
func (h *ImportHandler) Validate(c *gin.Context) {
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

	// Parse entity type
	entityType := c.PostForm("entity_type")
	if entityType == "" {
		h.BadRequest(c, "entity_type is required")
		return
	}

	if !csvimport.IsValidEntityType(entityType) {
		h.BadRequest(c, "invalid entity_type, must be one of: products, customers, suppliers, inventory")
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
	session := csvimport.NewImportSession(tenantID, userID, csvimport.EntityType(entityType), header.Filename, header.Size)

	// Get validation rules for entity type
	rules := h.getValidationRules(csvimport.EntityType(entityType))

	// Run validation
	result, err := h.processor.Validate(c.Request.Context(), session, file, rules)
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

	// Save session for future reference
	if err := h.sessionStore.Save(session); err != nil {
		h.InternalError(c, "failed to save import session")
		return
	}

	// Build response
	response := ValidationResponse{
		ValidationID: result.ValidationID,
		TotalRows:    result.TotalRows,
		ValidRows:    result.ValidRows,
		ErrorRows:    result.ErrorRows,
		Errors:       result.Errors,
		Preview:      result.Preview,
		IsTruncated:  result.IsTruncated,
		TotalErrors:  result.TotalErrors,
	}

	h.Success(c, response)
}

// GetSession godoc
//
//	@Summary		Get import session
//	@Description	Retrieves the status and details of an import session
//	@Tags			import
//	@ID				getImportSession
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Session ID (UUID)"
//	@Success		200			{object}	APIResponse[csvimport.ImportSession]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/sessions/{id} [get]
func (h *ImportHandler) GetSession(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid session ID")
		return
	}

	session, err := h.sessionStore.Get(sessionID)
	if err != nil {
		h.InternalError(c, "failed to retrieve session")
		return
	}

	if session == nil {
		h.NotFound(c, "Import session not found or expired")
		return
	}

	// Verify tenant ownership to prevent cross-tenant access
	if session.TenantID != tenantID {
		h.NotFound(c, "Import session not found or expired")
		return
	}

	h.Success(c, session)
}

// getValidationRules returns validation rules for an entity type
func (h *ImportHandler) getValidationRules(entityType csvimport.EntityType) []csvimport.FieldRule {
	switch entityType {
	case csvimport.EntityProducts:
		return h.getProductRules()
	case csvimport.EntityCustomers:
		return h.getCustomerRules()
	case csvimport.EntitySuppliers:
		return h.getSupplierRules()
	case csvimport.EntityInventory:
		return h.getInventoryRules()
	case csvimport.EntityCategories:
		return h.getCategoryRules()
	default:
		return []csvimport.FieldRule{}
	}
}

// getProductRules returns validation rules for products
func (h *ImportHandler) getProductRules() []csvimport.FieldRule {
	zero := decimal.Zero
	return []csvimport.FieldRule{
		csvimport.Field("code").Required().String().MinLength(1).MaxLength(50).Unique().Build(),
		csvimport.Field("name").Required().String().MinLength(1).MaxLength(200).Build(),
		csvimport.Field("description").String().MaxLength(2000).Build(),
		csvimport.Field("barcode").String().MaxLength(50).Build(),
		csvimport.Field("category_code").String().MaxLength(50).Reference("category").Build(),
		csvimport.Field("unit").Required().String().MinLength(1).MaxLength(20).Build(),
		csvimport.Field("purchase_price").Decimal().MinValue(zero).Build(),
		csvimport.Field("selling_price").Decimal().MinValue(zero).Build(),
		csvimport.Field("min_stock").Decimal().MinValue(zero).Build(),
	}
}

// getCustomerRules returns validation rules for customers
func (h *ImportHandler) getCustomerRules() []csvimport.FieldRule {
	return []csvimport.FieldRule{
		csvimport.Field("code").Required().String().MinLength(1).MaxLength(50).Unique().Build(),
		csvimport.Field("name").Required().String().MinLength(1).MaxLength(200).Build(),
		csvimport.Field("contact_name").String().MaxLength(100).Build(),
		csvimport.Field("phone").String().MaxLength(20).Pattern(`^[\d\-\+\s\(\)]+$`, "phone number").Build(),
		csvimport.Field("email").Email().Build(),
		csvimport.Field("address").String().MaxLength(500).Build(),
		csvimport.Field("city").String().MaxLength(100).Build(),
		csvimport.Field("province").String().MaxLength(100).Build(),
		csvimport.Field("postal_code").String().MaxLength(20).Build(),
		csvimport.Field("tax_id").String().MaxLength(50).Build(),
		csvimport.Field("credit_limit").Decimal().Build(),
	}
}

// getSupplierRules returns validation rules for suppliers
func (h *ImportHandler) getSupplierRules() []csvimport.FieldRule {
	return []csvimport.FieldRule{
		csvimport.Field("code").Required().String().MinLength(1).MaxLength(50).Unique().Build(),
		csvimport.Field("name").Required().String().MinLength(1).MaxLength(200).Build(),
		csvimport.Field("contact_name").String().MaxLength(100).Build(),
		csvimport.Field("phone").String().MaxLength(20).Pattern(`^[\d\-\+\s\(\)]+$`, "phone number").Build(),
		csvimport.Field("email").Email().Build(),
		csvimport.Field("address").String().MaxLength(500).Build(),
		csvimport.Field("city").String().MaxLength(100).Build(),
		csvimport.Field("province").String().MaxLength(100).Build(),
		csvimport.Field("postal_code").String().MaxLength(20).Build(),
		csvimport.Field("tax_id").String().MaxLength(50).Build(),
		csvimport.Field("bank_name").String().MaxLength(100).Build(),
		csvimport.Field("bank_account").String().MaxLength(50).Build(),
	}
}

// getInventoryRules returns validation rules for inventory adjustments
func (h *ImportHandler) getInventoryRules() []csvimport.FieldRule {
	return []csvimport.FieldRule{
		csvimport.Field("product_code").Required().String().MaxLength(50).Reference("product").Build(),
		csvimport.Field("warehouse_code").Required().String().MaxLength(50).Reference("warehouse").Build(),
		csvimport.Field("quantity").Required().Decimal().Build(),
		csvimport.Field("unit_cost").Decimal().Build(),
		csvimport.Field("batch_number").String().MaxLength(50).Build(),
		csvimport.Field("expiry_date").Date().Build(),
		csvimport.Field("note").String().MaxLength(500).Build(),
	}
}

// getCategoryRules returns validation rules for categories
func (h *ImportHandler) getCategoryRules() []csvimport.FieldRule {
	return []csvimport.FieldRule{
		csvimport.Field("code").Required().String().MinLength(1).MaxLength(50).Unique().Build(),
		csvimport.Field("name").Required().String().MinLength(1).MaxLength(200).Build(),
		csvimport.Field("description").String().MaxLength(2000).Build(),
		csvimport.Field("parent_code").String().MaxLength(50).Reference("category").Build(),
		csvimport.Field("sort_order").Int().Build(),
	}
}

// RegisterRoutes registers all import routes
func (h *ImportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	imports := rg.Group("/import")
	{
		imports.POST("/validate", h.Validate)
		imports.GET("/sessions/:id", h.GetSession)
	}
}
