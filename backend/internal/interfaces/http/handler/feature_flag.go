package handler

import (
	"time"

	featureflagapp "github.com/erp/backend/internal/application/featureflag"
	"github.com/erp/backend/internal/application/featureflag/dto"
	httpdto "github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FeatureFlagHandler handles feature flag related HTTP requests
type FeatureFlagHandler struct {
	BaseHandler
	flagService       *featureflagapp.FlagService
	evaluationService *featureflagapp.EvaluationService
	overrideService   *featureflagapp.OverrideService
}

// NewFeatureFlagHandler creates a new FeatureFlagHandler
func NewFeatureFlagHandler(
	flagService *featureflagapp.FlagService,
	evaluationService *featureflagapp.EvaluationService,
	overrideService *featureflagapp.OverrideService,
) *FeatureFlagHandler {
	return &FeatureFlagHandler{
		flagService:       flagService,
		evaluationService: evaluationService,
		overrideService:   overrideService,
	}
}

// ============================================================================
// Request/Response DTOs for HTTP layer
// ============================================================================

// CreateFlagHTTPRequest represents the HTTP request body for creating a flag
//
//	@Description	Request body for creating a new feature flag
type CreateFlagHTTPRequest struct {
	Key          string                 `json:"key" binding:"required,min=1,max=100" example:"new_checkout_flow"`
	Name         string                 `json:"name" binding:"required,min=1,max=200" example:"New Checkout Flow"`
	Description  string                 `json:"description,omitempty" example:"Enables the new checkout flow for users"`
	Type         string                 `json:"type" binding:"required,oneof=boolean percentage variant user_segment" example:"boolean"`
	DefaultValue dto.FlagValueDTO       `json:"default_value"`
	Rules        []dto.TargetingRuleDTO `json:"rules,omitempty"`
	Tags         []string               `json:"tags,omitempty" example:"checkout,experiment"`
}

// UpdateFlagHTTPRequest represents the HTTP request body for updating a flag
//
//	@Description	Request body for updating a feature flag
type UpdateFlagHTTPRequest struct {
	Name         *string                 `json:"name,omitempty" example:"Updated Name"`
	Description  *string                 `json:"description,omitempty" example:"Updated description"`
	DefaultValue *dto.FlagValueDTO       `json:"default_value,omitempty"`
	Rules        *[]dto.TargetingRuleDTO `json:"rules,omitempty"`
	Tags         *[]string               `json:"tags,omitempty"`
	Version      *int                    `json:"version,omitempty"` // For optimistic locking
}

// EvaluateFlagHTTPRequest represents the HTTP request body for evaluating a flag
//
//	@Description	Request body for evaluating a feature flag
type EvaluateFlagHTTPRequest struct {
	Context dto.EvaluationContextDTO `json:"context"`
}

// BatchEvaluateHTTPRequest represents the HTTP request body for batch evaluation
//
//	@Description	Request body for batch evaluating feature flags
type BatchEvaluateHTTPRequest struct {
	Keys    []string                 `json:"keys" binding:"required,min=1,max=100" example:"flag1,flag2"`
	Context dto.EvaluationContextDTO `json:"context"`
}

// ClientConfigHTTPRequest represents the HTTP request body for getting client config
//
//	@Description	Request body for getting client configuration
type ClientConfigHTTPRequest struct {
	Context dto.EvaluationContextDTO `json:"context"`
}

// CreateOverrideHTTPRequest represents the HTTP request body for creating an override
//
//	@Description	Request body for creating a flag override
type CreateOverrideHTTPRequest struct {
	TargetType string           `json:"target_type" binding:"required,oneof=user tenant" example:"user"`
	TargetID   string           `json:"target_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Value      dto.FlagValueDTO `json:"value"`
	Reason     string           `json:"reason,omitempty" binding:"max=500" example:"Testing for specific user"`
	ExpiresAt  *string          `json:"expires_at,omitempty" example:"2024-12-31T23:59:59Z"`
}

// ============================================================================
// Helper functions
// ============================================================================

// extractAuditContext extracts audit context from the request
func (h *FeatureFlagHandler) extractAuditContext(c *gin.Context) featureflagapp.AuditContext {
	var userID *uuid.UUID
	if userIDStr := middleware.GetJWTUserID(c); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			userID = &id
		}
	}

	return featureflagapp.AuditContext{
		UserID:    userID,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

// ============================================================================
// Flag Management Handlers
// ============================================================================

// ListFlags godoc
//
//	@Summary		List feature flags
//	@Description	Retrieve a paginated list of feature flags with optional filtering
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number"			default(1)
//	@Param			page_size	query		int		false	"Page size"				default(20)	maximum(100)
//	@Param			status		query		string	false	"Filter by status"		Enums(enabled, disabled, archived)
//	@Param			type		query		string	false	"Filter by type"		Enums(boolean, percentage, variant, user_segment)
//	@Param			tags		query		string	false	"Filter by tags (comma-separated)"
//	@Param			search		query		string	false	"Search term"
//	@Success		200			{object}	dto.Response{data=dto.FlagListResponse}
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags [get]
func (h *FeatureFlagHandler) ListFlags(c *gin.Context) {
	var filter dto.FlagListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := h.flagService.ListFlags(c.Request.Context(), filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// CreateFlag godoc
//
//	@Summary		Create a new feature flag
//	@Description	Create a new feature flag with the specified configuration
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateFlagHTTPRequest	true	"Flag creation request"
//	@Success		201		{object}	dto.Response{data=dto.FlagResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		409		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags [post]
func (h *FeatureFlagHandler) CreateFlag(c *gin.Context) {
	var req CreateFlagHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	auditCtx := h.extractAuditContext(c)

	// Convert to application DTO
	appReq := dto.CreateFlagRequest{
		Key:          req.Key,
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		DefaultValue: req.DefaultValue,
		Rules:        req.Rules,
		Tags:         req.Tags,
	}

	result, err := h.flagService.CreateFlag(c.Request.Context(), appReq, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, result)
}

// GetFlag godoc
//
//	@Summary		Get feature flag by key
//	@Description	Retrieve a feature flag by its unique key
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string	true	"Feature flag key"
//	@Success		200	{object}	dto.Response{data=dto.FlagResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key} [get]
func (h *FeatureFlagHandler) GetFlag(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	result, err := h.flagService.GetFlag(c.Request.Context(), key)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// UpdateFlag godoc
//
//	@Summary		Update a feature flag
//	@Description	Update an existing feature flag's configuration
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key		path		string					true	"Feature flag key"
//	@Param			request	body		UpdateFlagHTTPRequest	true	"Flag update request"
//	@Success		200		{object}	dto.Response{data=dto.FlagResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key} [put]
func (h *FeatureFlagHandler) UpdateFlag(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	var req UpdateFlagHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	auditCtx := h.extractAuditContext(c)

	// Convert to application DTO
	appReq := dto.UpdateFlagRequest{
		Name:         req.Name,
		Description:  req.Description,
		DefaultValue: req.DefaultValue,
		Rules:        req.Rules,
		Tags:         req.Tags,
		Version:      req.Version,
	}

	result, err := h.flagService.UpdateFlag(c.Request.Context(), key, appReq, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ArchiveFlag godoc
//
//	@Summary		Archive a feature flag
//	@Description	Archive a feature flag (soft delete). Archived flags cannot be evaluated.
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key	path	string	true	"Feature flag key"
//	@Success		204
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key} [delete]
func (h *FeatureFlagHandler) ArchiveFlag(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	auditCtx := h.extractAuditContext(c)

	if err := h.flagService.ArchiveFlag(c.Request.Context(), key, auditCtx); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// EnableFlag godoc
//
//	@Summary		Enable a feature flag
//	@Description	Enable a disabled feature flag
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string	true	"Feature flag key"
//	@Success		200	{object}	dto.Response{data=dto.MessageResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key}/enable [post]
func (h *FeatureFlagHandler) EnableFlag(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	auditCtx := h.extractAuditContext(c)

	if err := h.flagService.EnableFlag(c.Request.Context(), key, auditCtx); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, httpdto.MessageResponse{Message: "Feature flag enabled successfully"})
}

// DisableFlag godoc
//
//	@Summary		Disable a feature flag
//	@Description	Disable an enabled feature flag
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string	true	"Feature flag key"
//	@Success		200	{object}	dto.Response{data=dto.MessageResponse}
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		422	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key}/disable [post]
func (h *FeatureFlagHandler) DisableFlag(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	auditCtx := h.extractAuditContext(c)

	if err := h.flagService.DisableFlag(c.Request.Context(), key, auditCtx); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, httpdto.MessageResponse{Message: "Feature flag disabled successfully"})
}

// ============================================================================
// Evaluation Handlers
// ============================================================================

// EvaluateFlag godoc
//
//	@Summary		Evaluate a feature flag
//	@Description	Evaluate a single feature flag for the given context
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key		path		string					true	"Feature flag key"
//	@Param			request	body		EvaluateFlagHTTPRequest	true	"Evaluation context"
//	@Success		200		{object}	dto.Response{data=dto.EvaluateFlagResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key}/evaluate [post]
func (h *FeatureFlagHandler) EvaluateFlag(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	var req EvaluateFlagHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Enrich context with JWT information if available
	evalCtx := h.enrichEvaluationContext(c, req.Context)

	result, err := h.evaluationService.Evaluate(c.Request.Context(), key, evalCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// BatchEvaluate godoc
//
//	@Summary		Batch evaluate feature flags
//	@Description	Evaluate multiple feature flags at once for the given context
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			request	body		BatchEvaluateHTTPRequest	true	"Batch evaluation request"
//	@Success		200		{object}	dto.Response{data=dto.BatchEvaluateResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/evaluate-batch [post]
func (h *FeatureFlagHandler) BatchEvaluate(c *gin.Context) {
	var req BatchEvaluateHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Enrich context with JWT information if available
	evalCtx := h.enrichEvaluationContext(c, req.Context)

	result, err := h.evaluationService.BatchEvaluate(c.Request.Context(), req.Keys, evalCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// GetClientConfig godoc
//
//	@Summary		Get client configuration
//	@Description	Get all enabled feature flags and their values for client SDK initialization
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ClientConfigHTTPRequest	true	"Client config request"
//	@Success		200		{object}	dto.Response{data=dto.GetClientConfigResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/client-config [post]
func (h *FeatureFlagHandler) GetClientConfig(c *gin.Context) {
	var req ClientConfigHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Enrich context with JWT information if available
	evalCtx := h.enrichEvaluationContext(c, req.Context)

	result, err := h.evaluationService.GetClientConfig(c.Request.Context(), evalCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// enrichEvaluationContext adds JWT claims to the evaluation context if not already set
func (h *FeatureFlagHandler) enrichEvaluationContext(c *gin.Context, ctx dto.EvaluationContextDTO) dto.EvaluationContextDTO {
	// Add user ID from JWT if not provided
	if ctx.UserID == "" {
		if userID := middleware.GetJWTUserID(c); userID != "" {
			ctx.UserID = userID
		}
	}

	// Add tenant ID from JWT if not provided
	if ctx.TenantID == "" {
		if tenantID := middleware.GetJWTTenantID(c); tenantID != "" {
			ctx.TenantID = tenantID
		}
	}

	return ctx
}

// ============================================================================
// Override Handlers
// ============================================================================

// ListOverrides godoc
//
//	@Summary		List flag overrides
//	@Description	Retrieve all overrides for a specific feature flag
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key			path		string	true	"Feature flag key"
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size"		default(20)	maximum(100)
//	@Success		200			{object}	dto.Response{data=dto.OverrideListResponse}
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key}/overrides [get]
func (h *FeatureFlagHandler) ListOverrides(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	var filter dto.OverrideListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := h.overrideService.ListOverrides(c.Request.Context(), key, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// CreateOverride godoc
//
//	@Summary		Create a flag override
//	@Description	Create a new override for a feature flag to target specific users or tenants
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key		path		string						true	"Feature flag key"
//	@Param			request	body		CreateOverrideHTTPRequest	true	"Override creation request"
//	@Success		201		{object}	dto.Response{data=dto.OverrideResponse}
//	@Failure		400		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		409		{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500		{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key}/overrides [post]
func (h *FeatureFlagHandler) CreateOverride(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	var req CreateOverrideHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Parse target ID
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		h.BadRequest(c, "Invalid target ID format")
		return
	}

	auditCtx := h.extractAuditContext(c)

	// Convert to application DTO
	appReq := dto.CreateOverrideRequest{
		TargetType: req.TargetType,
		TargetID:   targetID,
		Value:      req.Value,
		Reason:     req.Reason,
	}

	// Parse expires_at if provided
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		expiresAt, err := parseTime(*req.ExpiresAt)
		if err != nil {
			h.BadRequest(c, "Invalid expires_at format. Use RFC3339 format (e.g., 2024-12-31T23:59:59Z)")
			return
		}
		appReq.ExpiresAt = &expiresAt
	}

	result, err := h.overrideService.CreateOverride(c.Request.Context(), key, appReq, auditCtx)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, result)
}

// DeleteOverride godoc
//
//	@Summary		Delete a flag override
//	@Description	Delete a specific override by its ID
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key	path	string	true	"Feature flag key"
//	@Param			id	path	string	true	"Override ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key}/overrides/{id} [delete]
func (h *FeatureFlagHandler) DeleteOverride(c *gin.Context) {
	// Note: We get the key param but don't use it directly for delete,
	// as the override ID is globally unique. However, we keep it in the route
	// for consistency and potential future validation.
	_ = c.Param("key")

	idStr := c.Param("id")
	if idStr == "" {
		h.BadRequest(c, "Override ID is required")
		return
	}

	overrideID, err := uuid.Parse(idStr)
	if err != nil {
		h.BadRequest(c, "Invalid override ID format")
		return
	}

	auditCtx := h.extractAuditContext(c)

	if err := h.overrideService.DeleteOverride(c.Request.Context(), overrideID, auditCtx); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// ============================================================================
// Audit Log Handlers
// ============================================================================

// GetAuditLogs godoc
//
//	@Summary		Get flag audit logs
//	@Description	Retrieve audit logs for a specific feature flag
//	@Tags			feature-flags
//	@Accept			json
//	@Produce		json
//	@Param			key			path		string	true	"Feature flag key"
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size"		default(20)	maximum(100)
//	@Param			action		query		string	false	"Filter by action"
//	@Success		200			{object}	dto.Response{data=dto.AuditLogListResponse}
//	@Failure		400			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		401			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		403			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		404			{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		500			{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/{key}/audit-logs [get]
func (h *FeatureFlagHandler) GetAuditLogs(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		h.BadRequest(c, "Feature flag key is required")
		return
	}

	var filter dto.AuditLogListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := h.flagService.GetAuditLogs(c.Request.Context(), key, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// ============================================================================
// Helper functions
// ============================================================================

// parseTime parses a time string in RFC3339 format
func parseTime(s string) (t time.Time, err error) {
	return time.Parse(time.RFC3339, s)
}
