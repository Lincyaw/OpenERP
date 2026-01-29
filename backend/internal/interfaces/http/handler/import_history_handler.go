package handler

import (
	"net/http"
	"time"

	importapp "github.com/erp/backend/internal/application/import"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ImportHistoryHandler handles import history related HTTP requests
type ImportHistoryHandler struct {
	BaseHandler
	historyService *importapp.ImportHistoryService
}

// NewImportHistoryHandler creates a new ImportHistoryHandler
func NewImportHistoryHandler(historyService *importapp.ImportHistoryService) *ImportHistoryHandler {
	return &ImportHistoryHandler{
		historyService: historyService,
	}
}

// ListHistory godoc
//
//	@Summary		List import histories
//	@Description	Returns a paginated list of import histories with optional filtering
//	@Tags			import
//	@ID				listImportHistory
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			entity_type	query		string	false	"Filter by entity type (products, customers, suppliers, inventory, categories)"
//	@Param			status		query		string	false	"Filter by status (pending, processing, completed, failed, cancelled)"
//	@Param			started_from	query	string	false	"Filter by start date (from), format: YYYY-MM-DD"
//	@Param			started_to	query		string	false	"Filter by start date (to), format: YYYY-MM-DD"
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			page_size	query		int		false	"Page size (default: 20, max: 100)"
//	@Success		200			{object}	APIResponse[dto.ImportHistoryListResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/history [get]
func (h *ImportHistoryHandler) ListHistory(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	// Parse query parameters
	var req dto.ImportHistoryListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// Build filter
	filter := importapp.ListHistoryFilter{
		EntityType: req.EntityType,
		Status:     req.Status,
	}

	// Parse date filters
	if req.StartedFrom != "" {
		t, err := time.Parse("2006-01-02", req.StartedFrom)
		if err == nil {
			filter.StartedFrom = &t
		}
	}
	if req.StartedTo != "" {
		t, err := time.Parse("2006-01-02", req.StartedTo)
		if err == nil {
			// Set to end of day
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.StartedTo = &endOfDay
		}
	}

	result, err := h.historyService.ListHistory(ctx, tenantID, filter, req.Page, req.PageSize)
	if err != nil {
		h.InternalError(c, "Failed to list import histories: "+err.Error())
		return
	}

	response := dto.NewImportHistoryListResponse(result)
	h.Success(c, response)
}

// GetHistory godoc
//
//	@Summary		Get import history details
//	@Description	Returns detailed information about a specific import operation
//	@Tags			import
//	@ID				getImportHistory
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Import history ID"
//	@Success		200			{object}	APIResponse[dto.ImportHistoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/history/{id} [get]
func (h *ImportHistoryHandler) GetHistory(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	// Parse history ID
	historyIDStr := c.Param("id")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid history ID")
		return
	}

	history, err := h.historyService.GetHistory(ctx, tenantID, historyID)
	if err != nil {
		if err == shared.ErrNotFound {
			h.NotFound(c, "Import history not found")
			return
		}
		h.InternalError(c, "Failed to get import history: "+err.Error())
		return
	}

	response := dto.NewImportHistoryResponse(history)
	h.Success(c, response)
}

// GetErrors godoc
//
//	@Summary		Download import errors as CSV
//	@Description	Downloads error details from an import operation as a CSV file
//	@Tags			import
//	@ID				getImportErrors
//	@Produce		text/csv
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Import history ID"
//	@Success		200			{string}	string	"CSV content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/history/{id}/errors [get]
func (h *ImportHistoryHandler) GetErrors(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	// Parse history ID
	historyIDStr := c.Param("id")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid history ID")
		return
	}

	csvContent, fileName, err := h.historyService.GetErrorsCSV(ctx, tenantID, historyID)
	if err != nil {
		if err == shared.ErrNotFound {
			h.NotFound(c, "Import history not found")
			return
		}
		if err.Error() == "no errors to export" {
			h.BadRequest(c, "No errors to export for this import")
			return
		}
		h.InternalError(c, "Failed to generate error report: "+err.Error())
		return
	}

	// Set response headers for CSV download
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	c.Header("Content-Length", string(rune(len(csvContent))))

	c.String(http.StatusOK, csvContent)
}

// DeleteHistory godoc
//
//	@Summary		Delete import history
//	@Description	Deletes an import history record
//	@Tags			import
//	@ID				deleteImportHistory
//	@Produce		json
//	@Param			X-Tenant-ID	header	string	false	"Tenant ID (optional for dev)"
//	@Param			id			path	string	true	"Import history ID"
//	@Success		204			"Successfully deleted"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/import/history/{id} [delete]
func (h *ImportHistoryHandler) DeleteHistory(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	// Parse history ID
	historyIDStr := c.Param("id")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		h.BadRequest(c, "Invalid history ID")
		return
	}

	err = h.historyService.DeleteHistory(ctx, tenantID, historyID)
	if err != nil {
		if err == shared.ErrNotFound {
			h.NotFound(c, "Import history not found")
			return
		}
		h.InternalError(c, "Failed to delete import history: "+err.Error())
		return
	}

	h.NoContent(c)
}

// RegisterRoutes registers all import history routes
func (h *ImportHistoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	history := rg.Group("/import/history")
	{
		history.GET("", h.ListHistory)
		history.GET("/:id", h.GetHistory)
		history.GET("/:id/errors", h.GetErrors)
		history.DELETE("/:id", h.DeleteHistory)
	}
}
