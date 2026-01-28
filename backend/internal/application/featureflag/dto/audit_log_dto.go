package dto

import (
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/google/uuid"
)

// AuditLogResponse represents a feature flag audit log entry in API responses
// @name FeatureFlagAuditLogResponse
type AuditLogResponse struct {
	ID        uuid.UUID      `json:"id"`
	FlagKey   string         `json:"flag_key"`
	Action    string         `json:"action"`
	OldValue  map[string]any `json:"old_value,omitempty"`
	NewValue  map[string]any `json:"new_value,omitempty"`
	UserID    *uuid.UUID     `json:"user_id,omitempty"`
	IPAddress string         `json:"ip_address,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ToAuditLogResponse converts domain FlagAuditLog to AuditLogResponse
// @name FeatureFlagAuditLogListFilter
func ToAuditLogResponse(log *featureflag.FlagAuditLog) *AuditLogResponse {
	if log == nil {
		return nil
	}
	return &AuditLogResponse{
		ID:        log.ID,
		FlagKey:   log.GetFlagKey(),
		Action:    string(log.GetAction()),
		OldValue:  log.GetOldValue(),
		NewValue:  log.GetNewValue(),
		UserID:    log.GetUserID(),
		IPAddress: log.GetIPAddress(),
		UserAgent: log.GetUserAgent(),
		CreatedAt: log.GetTimestamp(),
	}
}

// AuditLogListFilter represents filter options for listing audit logs
type AuditLogListFilter struct {
	Page     int     `form:"page,default=1" binding:"min=1"`
	PageSize int     `form:"page_size,default=20" binding:"min=1,max=100"`
	Action   *string `form:"action,omitempty"`
}

// AuditLogListResponse represents a paginated list of audit logs
// @name FeatureFlagAuditLogListFilter
type AuditLogListResponse struct {
	AuditLogs  []AuditLogResponse `json:"audit_logs"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// ToAuditLogListResponse creates an AuditLogListResponse from domain audit logs
func ToAuditLogListResponse(logs []featureflag.FlagAuditLog, total int64, page, pageSize int) *AuditLogListResponse {
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	logResponses := make([]AuditLogResponse, len(logs))
	for i := range logs {
		logResponses[i] = *ToAuditLogResponse(&logs[i])
	}

	return &AuditLogListResponse{
		AuditLogs:  logResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
