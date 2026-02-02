package handler

import (
	"net/http"

	printingapp "github.com/erp/backend/internal/application/printing"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AutoPrintRuleHandler handles auto print rule API endpoints
type AutoPrintRuleHandler struct {
	BaseHandler
	service *printingapp.AutoPrintRuleService
}

// NewAutoPrintRuleHandler creates a new AutoPrintRuleHandler
func NewAutoPrintRuleHandler(service *printingapp.AutoPrintRuleService) *AutoPrintRuleHandler {
	return &AutoPrintRuleHandler{
		service: service,
	}
}

// =============================================================================
// Request/Response Types for Swagger
// =============================================================================

// CreateAutoPrintRuleHTTPRequest represents a request to create an auto print rule
//
//	@Description	Request body for creating an auto print rule
type CreateAutoPrintRuleHTTPRequest struct {
	DocumentType string  `json:"document_type" binding:"required" example:"SALES_ORDER"`
	TriggerEvent string  `json:"trigger_event" binding:"required" example:"CONFIRMED"`
	TemplateID   *string `json:"template_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AutoPrint    bool    `json:"auto_print" example:"true"`
	Copies       *int    `json:"copies" binding:"omitempty,min=1,max=100" example:"1"`
	PrinterName  string  `json:"printer_name" example:"Office Printer"`
}

// UpdateAutoPrintRuleHTTPRequest represents a request to update an auto print rule
//
//	@Description	Request body for updating an auto print rule
type UpdateAutoPrintRuleHTTPRequest struct {
	TemplateID  *string `json:"template_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AutoPrint   bool    `json:"auto_print" example:"true"`
	Copies      *int    `json:"copies" binding:"omitempty,min=1,max=100" example:"1"`
	PrinterName string  `json:"printer_name" example:"Office Printer"`
}

// AutoPrintRuleHTTPResponse represents an auto print rule response
//
//	@Description	Auto print rule response
type AutoPrintRuleHTTPResponse struct {
	ID           string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID     string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	DocumentType string  `json:"document_type" example:"SALES_ORDER"`
	TriggerEvent string  `json:"trigger_event" example:"CONFIRMED"`
	TemplateID   *string `json:"template_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	AutoPrint    bool    `json:"auto_print" example:"true"`
	Copies       int     `json:"copies" example:"1"`
	PrinterName  string  `json:"printer_name,omitempty" example:"Office Printer"`
	Enabled      bool    `json:"enabled" example:"true"`
	CreatedAt    string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt    string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// TriggerEventHTTPResponse represents a trigger event response
//
//	@Description	Trigger event response
type TriggerEventHTTPResponse struct {
	Code        string `json:"code" example:"CONFIRMED"`
	DisplayName string `json:"display_name" example:"确认时"`
}

// =============================================================================
// API Endpoints
// =============================================================================

// CreateRule godoc
//
//	@ID				createAutoPrintRule
//
//	@Summary		Create auto print rule
//	@Description	Create a new auto print rule for automatic printing on business events
//	@Tags			auto-print-rules
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateAutoPrintRuleHTTPRequest	true	"Create rule request"
//	@Success		201		{object}	APIResponse[AutoPrintRuleHTTPResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse	"Rule already exists for this document type and trigger event"
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules [post]
func (h *AutoPrintRuleHandler) CreateRule(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateAutoPrintRuleHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := printingapp.CreateAutoPrintRuleRequest{
		DocumentType: req.DocumentType,
		TriggerEvent: req.TriggerEvent,
		TemplateID:   req.TemplateID,
		AutoPrint:    req.AutoPrint,
		Copies:       req.Copies,
		PrinterName:  req.PrinterName,
	}

	result, err := h.service.CreateRule(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, result)
}

// GetRule godoc
//
//	@ID				getAutoPrintRule
//
//	@Summary		Get auto print rule by ID
//	@Description	Retrieve an auto print rule by its ID
//	@Tags			auto-print-rules
//	@Produce		json
//	@Param			id	path		string	true	"Rule ID"	format(uuid)
//	@Success		200	{object}	APIResponse[AutoPrintRuleHTTPResponse]
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules/{id} [get]
func (h *AutoPrintRuleHandler) GetRule(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid rule ID format")
		return
	}

	result, err := h.service.GetRule(c.Request.Context(), tenantID, ruleID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ListRules godoc
//
//	@ID				listAutoPrintRules
//
//	@Summary		List auto print rules
//	@Description	Retrieve a paginated list of auto print rules
//	@Tags			auto-print-rules
//	@Produce		json
//	@Param			page			query		int		false	"Page number"			default(1)
//	@Param			page_size		query		int		false	"Page size"				default(20)
//	@Param			order_by		query		string	false	"Order by field"		default(created_at)
//	@Param			order_dir		query		string	false	"Order direction"		Enums(asc, desc)	default(desc)
//	@Param			document_type	query		string	false	"Filter by document type"
//	@Param			trigger_event	query		string	false	"Filter by trigger event"
//	@Param			enabled			query		bool	false	"Filter by enabled status"
//	@Success		200				{object}	APIResponse[[]AutoPrintRuleHTTPResponse]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		403				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules [get]
func (h *AutoPrintRuleHandler) ListRules(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	req := printingapp.ListAutoPrintRulesRequest{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// Validate OrderBy to prevent SQL injection
	allowedOrderBy := map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"document_type": true,
		"trigger_event": true,
		"enabled":       true,
	}
	if req.OrderBy != "" && !allowedOrderBy[req.OrderBy] {
		h.BadRequest(c, "Invalid order_by field")
		return
	}

	// Validate OrderDir
	if req.OrderDir != "" && req.OrderDir != "asc" && req.OrderDir != "desc" {
		h.BadRequest(c, "Invalid order_dir field, must be 'asc' or 'desc'")
		return
	}

	result, err := h.service.ListRules(c.Request.Context(), tenantID, req)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, result.Items, result.Total, result.Page, result.Size)
}

// UpdateRule godoc
//
//	@ID				updateAutoPrintRule
//
//	@Summary		Update auto print rule
//	@Description	Update an existing auto print rule
//	@Tags			auto-print-rules
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Rule ID"	format(uuid)
//	@Param			request	body		UpdateAutoPrintRuleHTTPRequest	true	"Update rule request"
//	@Success		200		{object}	APIResponse[AutoPrintRuleHTTPResponse]
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules/{id} [put]
func (h *AutoPrintRuleHandler) UpdateRule(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid rule ID format")
		return
	}

	var req UpdateAutoPrintRuleHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := printingapp.UpdateAutoPrintRuleRequest{
		TemplateID:  req.TemplateID,
		AutoPrint:   req.AutoPrint,
		Copies:      req.Copies,
		PrinterName: req.PrinterName,
	}

	result, err := h.service.UpdateRule(c.Request.Context(), tenantID, ruleID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// DeleteRule godoc
//
//	@ID				deleteAutoPrintRule
//
//	@Summary		Delete auto print rule
//	@Description	Delete an auto print rule by ID
//	@Tags			auto-print-rules
//	@Produce		json
//	@Param			id	path	string	true	"Rule ID"	format(uuid)
//	@Success		204	"No Content"
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules/{id} [delete]
func (h *AutoPrintRuleHandler) DeleteRule(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid rule ID format")
		return
	}

	if err := h.service.DeleteRule(c.Request.Context(), tenantID, ruleID); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// EnableRule godoc
//
//	@ID				enableAutoPrintRule
//
//	@Summary		Enable auto print rule
//	@Description	Enable an auto print rule
//	@Tags			auto-print-rules
//	@Produce		json
//	@Param			id	path		string	true	"Rule ID"	format(uuid)
//	@Success		200	{object}	APIResponse[AutoPrintRuleHTTPResponse]
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules/{id}/enable [post]
func (h *AutoPrintRuleHandler) EnableRule(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid rule ID format")
		return
	}

	result, err := h.service.EnableRule(c.Request.Context(), tenantID, ruleID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// DisableRule godoc
//
//	@ID				disableAutoPrintRule
//
//	@Summary		Disable auto print rule
//	@Description	Disable an auto print rule
//	@Tags			auto-print-rules
//	@Produce		json
//	@Param			id	path		string	true	"Rule ID"	format(uuid)
//	@Success		200	{object}	APIResponse[AutoPrintRuleHTTPResponse]
//	@Failure		400	{object}	dto.ErrorResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules/{id}/disable [post]
func (h *AutoPrintRuleHandler) DisableRule(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid rule ID format")
		return
	}

	result, err := h.service.DisableRule(c.Request.Context(), tenantID, ruleID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// LookupRule godoc
//
//	@ID				lookupAutoPrintRule
//
//	@Summary		Lookup auto print rule
//	@Description	Find an auto print rule by document type and trigger event
//	@Tags			auto-print-rules
//	@Produce		json
//	@Param			document_type	query		string	true	"Document type"
//	@Param			trigger_event	query		string	true	"Trigger event"
//	@Success		200				{object}	APIResponse[AutoPrintRuleHTTPResponse]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		403				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules/lookup [get]
func (h *AutoPrintRuleHandler) LookupRule(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	documentType := c.Query("document_type")
	if documentType == "" {
		h.BadRequest(c, "document_type is required")
		return
	}

	triggerEvent := c.Query("trigger_event")
	if triggerEvent == "" {
		h.BadRequest(c, "trigger_event is required")
		return
	}

	result, err := h.service.LookupRule(c.Request.Context(), tenantID, documentType, triggerEvent)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// GetTriggerEvents godoc
//
//	@ID				getTriggerEventsForDocType
//
//	@Summary		Get trigger events for document type
//	@Description	Get all trigger events applicable to a specific document type
//	@Tags			auto-print-rules
//	@Produce		json
//	@Param			document_type	query		string	true	"Document type"
//	@Success		200				{object}	APIResponse[[]TriggerEventHTTPResponse]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/printing/auto-rules/trigger-events [get]
func (h *AutoPrintRuleHandler) GetTriggerEvents(c *gin.Context) {
	documentType := c.Query("document_type")
	if documentType == "" {
		h.BadRequest(c, "document_type is required")
		return
	}

	result, err := h.service.GetTriggerEvents(documentType)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// =============================================================================
// Permission Check Helper
// =============================================================================

// RequireAdminRole is a middleware that checks if the user has admin role
// This is used to restrict auto print rule management to administrators
func RequireAdminRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user roles from context (set by JWT middleware)
		roles, exists := c.Get("user_roles")
		if !exists {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(dto.ErrCodeForbidden, "Access denied: admin role required"))
			c.Abort()
			return
		}

		// Check if user has admin role
		roleList, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(dto.ErrCodeForbidden, "Access denied: admin role required"))
			c.Abort()
			return
		}

		hasAdmin := false
		for _, role := range roleList {
			if role == "admin" || role == "super_admin" {
				hasAdmin = true
				break
			}
		}

		if !hasAdmin {
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(dto.ErrCodeForbidden, "Access denied: admin role required"))
			c.Abort()
			return
		}

		c.Next()
	}
}
