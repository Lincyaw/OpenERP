package handler

import (
	"github.com/erp/backend/internal/application/event"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OutboxHandler handles outbox management HTTP requests
type OutboxHandler struct {
	BaseHandler
	outboxService *event.OutboxService
}

// NewOutboxHandler creates a new outbox handler
func NewOutboxHandler(outboxService *event.OutboxService) *OutboxHandler {
	return &OutboxHandler{
		outboxService: outboxService,
	}
}


// GetDeadLetterEntries godoc
// @ID           getOutboxDeadLetterEntries
// @Summary      List dead letter entries
// @Description  Get a paginated list of dead letter queue entries
// @Tags         outbox
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Items per page" default(20) maximum(100)
// @Success      200 {object} APIResponse[OutboxListResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /system/outbox/dead [get]
func (h *OutboxHandler) GetDeadLetterEntries(c *gin.Context) {
	var filter event.OutboxFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, "Invalid query parameters")
		return
	}

	result, err := h.outboxService.GetDeadLetterEntries(c.Request.Context(), filter)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toOutboxListResponse(result))
}


// GetEntry godoc
// @ID           getOutboxEntry
// @Summary      Get an outbox entry by ID
// @Description  Retrieve a single outbox entry by its ID
// @Tags         outbox
// @Produce      json
// @Param        id path string true "Outbox Entry ID" format(uuid)
// @Success      200 {object} APIResponse[OutboxEntryResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /system/outbox/{id} [get]
func (h *OutboxHandler) GetEntry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid entry ID")
		return
	}

	entry, err := h.outboxService.GetEntry(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toOutboxEntryResponse(entry))
}


// RetryDeadEntry godoc
// @ID           retryDeadEntryOutbox
// @Summary      Retry a dead letter entry
// @Description  Reset a dead letter entry for retry processing
// @Tags         outbox
// @Accept       json
// @Produce      json
// @Param        id path string true "Outbox Entry ID" format(uuid)
// @Success      200 {object} APIResponse[OutboxEntryResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      422 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /system/outbox/{id}/retry [post]
func (h *OutboxHandler) RetryDeadEntry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid entry ID")
		return
	}

	entry, err := h.outboxService.RetryDeadEntry(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toOutboxEntryResponse(entry))
}


// RetryAllDeadEntries godoc
// @ID           retryAllDeadEntriesOutbox
// @Summary      Retry all dead letter entries
// @Description  Reset all dead letter entries for retry processing
// @Tags         outbox
// @Accept       json
// @Produce      json
// @Success      200 {object} APIResponse[RetryAllResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /system/outbox/dead/retry-all [post]
func (h *OutboxHandler) RetryAllDeadEntries(c *gin.Context) {
	count, err := h.outboxService.RetryAllDeadEntries(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, RetryAllResponse{Count: count})
}


// GetStats godoc
// @ID           getOutboxStats
// @Summary      Get outbox statistics
// @Description  Get statistics about outbox entries by status
// @Tags         outbox
// @Produce      json
// @Success      200 {object} APIResponse[OutboxStatsResponse]
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     BearerAuth
// @Router       /system/outbox/stats [get]
func (h *OutboxHandler) GetStats(c *gin.Context) {
	stats, err := h.outboxService.GetStats(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, toOutboxStatsResponse(stats))
}

// Request/Response types for swagger

// OutboxEntryResponse represents an outbox entry in API response
type OutboxEntryResponse struct {
	ID            string  `json:"id"`
	TenantID      string  `json:"tenant_id"`
	EventID       string  `json:"event_id"`
	EventType     string  `json:"event_type"`
	AggregateID   string  `json:"aggregate_id"`
	AggregateType string  `json:"aggregate_type"`
	Status        string  `json:"status"`
	RetryCount    int     `json:"retry_count"`
	MaxRetries    int     `json:"max_retries"`
	LastError     string  `json:"last_error,omitempty"`
	NextRetryAt   *string `json:"next_retry_at,omitempty"`
	ProcessedAt   *string `json:"processed_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// OutboxListResponse represents paginated outbox list response
type OutboxListResponse struct {
	Entries    []OutboxEntryResponse `json:"entries"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// OutboxStatsResponse represents outbox statistics response
type OutboxStatsResponse struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Sent       int64 `json:"sent"`
	Failed     int64 `json:"failed"`
	Dead       int64 `json:"dead"`
	Total      int64 `json:"total"`
}

// RetryAllResponse represents the response for retry all operation
type RetryAllResponse struct {
	Count int64 `json:"count"`
}

// Conversion functions

func toOutboxEntryResponse(dto *event.OutboxEntryDTO) OutboxEntryResponse {
	resp := OutboxEntryResponse{
		ID:            dto.ID.String(),
		TenantID:      dto.TenantID.String(),
		EventID:       dto.EventID.String(),
		EventType:     dto.EventType,
		AggregateID:   dto.AggregateID.String(),
		AggregateType: dto.AggregateType,
		Status:        dto.Status,
		RetryCount:    dto.RetryCount,
		MaxRetries:    dto.MaxRetries,
		LastError:     dto.LastError,
		CreatedAt:     dto.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     dto.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if dto.NextRetryAt != nil {
		t := dto.NextRetryAt.Format("2006-01-02T15:04:05Z07:00")
		resp.NextRetryAt = &t
	}
	if dto.ProcessedAt != nil {
		t := dto.ProcessedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ProcessedAt = &t
	}
	return resp
}

func toOutboxListResponse(result *event.OutboxListResult) OutboxListResponse {
	entries := make([]OutboxEntryResponse, len(result.Entries))
	for i, entry := range result.Entries {
		entries[i] = toOutboxEntryResponse(&entry)
	}
	return OutboxListResponse{
		Entries:    entries,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}
}

func toOutboxStatsResponse(dto *event.OutboxStatsDTO) OutboxStatsResponse {
	return OutboxStatsResponse{
		Pending:    dto.Pending,
		Processing: dto.Processing,
		Sent:       dto.Sent,
		Failed:     dto.Failed,
		Dead:       dto.Dead,
		Total:      dto.Total,
	}
}
