package identity

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuditService handles audit log operations for admin actions
type AuditService struct {
	auditRepo identity.AuditLogRepository
	logger    *zap.Logger
}

// NewAuditService creates a new audit service
func NewAuditService(
	auditRepo identity.AuditLogRepository,
	logger *zap.Logger,
) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
		logger:    logger,
	}
}

// RecordAuditInput contains input for recording an audit log
type RecordAuditInput struct {
	AdminUserID uuid.UUID
	Action      identity.AuditAction
	TargetType  identity.AuditTargetType
	TargetID    *uuid.UUID
	OldValue    interface{}
	NewValue    interface{}
	IPAddress   string
	UserAgent   string
}

// AuditLogDTO represents audit log data transfer object
type AuditLogDTO struct {
	ID          uuid.UUID              `json:"id"`
	AdminUserID uuid.UUID              `json:"admin_user_id"`
	Action      string                 `json:"action"`
	TargetType  string                 `json:"target_type"`
	TargetID    *uuid.UUID             `json:"target_id,omitempty"`
	OldValue    map[string]interface{} `json:"old_value,omitempty"`
	NewValue    map[string]interface{} `json:"new_value,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// AuditLogListResult represents paginated audit log list result
type AuditLogListResult struct {
	Logs       []AuditLogDTO `json:"logs"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// AuditLogFilterInput contains filter options for querying audit logs
type AuditLogFilterInput struct {
	AdminUserID *uuid.UUID `json:"admin_user_id,omitempty"`
	Action      *string    `json:"action,omitempty"`
	TargetType  *string    `json:"target_type,omitempty"`
	TargetID    *uuid.UUID `json:"target_id,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	IPAddress   string     `json:"ip_address,omitempty"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
	SortBy      string     `json:"sort_by,omitempty"`
	SortOrder   string     `json:"sort_order,omitempty"`
}

// Record records a new audit log entry
func (s *AuditService) Record(ctx context.Context, input RecordAuditInput) error {
	s.logger.Info("Recording audit log",
		zap.String("action", string(input.Action)),
		zap.String("target_type", string(input.TargetType)),
		zap.String("admin_user_id", input.AdminUserID.String()))

	builder := identity.NewAuditLog(input.AdminUserID, input.Action, input.TargetType)

	if input.TargetID != nil {
		builder.WithTargetID(*input.TargetID)
	}

	if input.OldValue != nil {
		builder.WithOldValueFromEntity(input.OldValue)
	}

	if input.NewValue != nil {
		builder.WithNewValueFromEntity(input.NewValue)
	}

	if input.IPAddress != "" {
		builder.WithIPAddress(input.IPAddress)
	}

	if input.UserAgent != "" {
		builder.WithUserAgent(input.UserAgent)
	}

	log, err := builder.Build()
	if err != nil {
		s.logger.Error("Failed to build audit log", zap.Error(err))
		return err
	}

	if err := s.auditRepo.Create(ctx, log); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
		return err
	}

	s.logger.Info("Audit log recorded successfully",
		zap.String("audit_log_id", log.ID.String()))

	return nil
}

// RecordTenantCreate records a tenant creation audit log
func (s *AuditService) RecordTenantCreate(ctx context.Context, adminUserID uuid.UUID, tenant interface{}, ip, userAgent string) error {
	tenantID := extractID(tenant)
	return s.Record(ctx, RecordAuditInput{
		AdminUserID: adminUserID,
		Action:      identity.AuditActionTenantCreate,
		TargetType:  identity.AuditTargetTenant,
		TargetID:    tenantID,
		NewValue:    tenant,
		IPAddress:   ip,
		UserAgent:   userAgent,
	})
}

// RecordTenantUpdate records a tenant update audit log
func (s *AuditService) RecordTenantUpdate(ctx context.Context, adminUserID uuid.UUID, oldTenant, newTenant interface{}, ip, userAgent string) error {
	tenantID := extractID(newTenant)
	return s.Record(ctx, RecordAuditInput{
		AdminUserID: adminUserID,
		Action:      identity.AuditActionTenantUpdate,
		TargetType:  identity.AuditTargetTenant,
		TargetID:    tenantID,
		OldValue:    oldTenant,
		NewValue:    newTenant,
		IPAddress:   ip,
		UserAgent:   userAgent,
	})
}

// RecordTenantDelete records a tenant deletion audit log
func (s *AuditService) RecordTenantDelete(ctx context.Context, adminUserID uuid.UUID, tenant interface{}, ip, userAgent string) error {
	tenantID := extractID(tenant)
	return s.Record(ctx, RecordAuditInput{
		AdminUserID: adminUserID,
		Action:      identity.AuditActionTenantDelete,
		TargetType:  identity.AuditTargetTenant,
		TargetID:    tenantID,
		OldValue:    tenant,
		IPAddress:   ip,
		UserAgent:   userAgent,
	})
}

// RecordTenantSuspend records a tenant suspension audit log
func (s *AuditService) RecordTenantSuspend(ctx context.Context, adminUserID uuid.UUID, oldTenant, newTenant interface{}, ip, userAgent string) error {
	tenantID := extractID(newTenant)
	return s.Record(ctx, RecordAuditInput{
		AdminUserID: adminUserID,
		Action:      identity.AuditActionTenantSuspend,
		TargetType:  identity.AuditTargetTenant,
		TargetID:    tenantID,
		OldValue:    oldTenant,
		NewValue:    newTenant,
		IPAddress:   ip,
		UserAgent:   userAgent,
	})
}

// RecordTenantActivate records a tenant activation audit log
func (s *AuditService) RecordTenantActivate(ctx context.Context, adminUserID uuid.UUID, oldTenant, newTenant interface{}, ip, userAgent string) error {
	tenantID := extractID(newTenant)
	return s.Record(ctx, RecordAuditInput{
		AdminUserID: adminUserID,
		Action:      identity.AuditActionTenantActivate,
		TargetType:  identity.AuditTargetTenant,
		TargetID:    tenantID,
		OldValue:    oldTenant,
		NewValue:    newTenant,
		IPAddress:   ip,
		UserAgent:   userAgent,
	})
}

// RecordSubscriptionChange records a subscription change audit log
func (s *AuditService) RecordSubscriptionChange(ctx context.Context, adminUserID uuid.UUID, tenantID uuid.UUID, oldValue, newValue interface{}, ip, userAgent string) error {
	return s.Record(ctx, RecordAuditInput{
		AdminUserID: adminUserID,
		Action:      identity.AuditActionSubscriptionChange,
		TargetType:  identity.AuditTargetSubscription,
		TargetID:    &tenantID,
		OldValue:    oldValue,
		NewValue:    newValue,
		IPAddress:   ip,
		UserAgent:   userAgent,
	})
}

// RecordQuotaUpdate records a quota update audit log
func (s *AuditService) RecordQuotaUpdate(ctx context.Context, adminUserID uuid.UUID, tenantID uuid.UUID, oldValue, newValue interface{}, ip, userAgent string) error {
	return s.Record(ctx, RecordAuditInput{
		AdminUserID: adminUserID,
		Action:      identity.AuditActionQuotaUpdate,
		TargetType:  identity.AuditTargetQuota,
		TargetID:    &tenantID,
		OldValue:    oldValue,
		NewValue:    newValue,
		IPAddress:   ip,
		UserAgent:   userAgent,
	})
}

// GetByID retrieves an audit log by ID
func (s *AuditService) GetByID(ctx context.Context, id uuid.UUID) (*AuditLogDTO, error) {
	log, err := s.auditRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find audit log", zap.Error(err))
		return nil, err
	}
	if log == nil {
		return nil, nil
	}

	return toAuditLogDTO(log), nil
}

// List retrieves audit logs with filtering and pagination
func (s *AuditService) List(ctx context.Context, input AuditLogFilterInput) (*AuditLogListResult, error) {
	filter := identity.NewAuditLogFilter()

	if input.AdminUserID != nil {
		filter = filter.WithAdminUserID(*input.AdminUserID)
	}

	if input.Action != nil {
		filter = filter.WithAction(identity.AuditAction(*input.Action))
	}

	if input.TargetType != nil {
		filter = filter.WithTargetType(identity.AuditTargetType(*input.TargetType))
	}

	if input.TargetID != nil {
		filter = filter.WithTargetID(*input.TargetID)
	}

	if input.StartTime != nil {
		filter = filter.WithStartTime(*input.StartTime)
	}

	if input.EndTime != nil {
		filter = filter.WithEndTime(*input.EndTime)
	}

	if input.IPAddress != "" {
		filter = filter.WithIPAddress(input.IPAddress)
	}

	if input.Page > 0 {
		filter.Page = input.Page
	}

	if input.PageSize > 0 {
		filter.PageSize = input.PageSize
	}

	if input.SortBy != "" {
		filter.SortBy = input.SortBy
	}

	if input.SortOrder != "" {
		filter.SortOrder = input.SortOrder
	}

	logs, total, err := s.auditRepo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list audit logs", zap.Error(err))
		return nil, err
	}

	dtos := make([]AuditLogDTO, len(logs))
	for i, log := range logs {
		dtos[i] = *toAuditLogDTO(log)
	}

	totalPages := int(total) / filter.Limit()
	if int(total)%filter.Limit() > 0 {
		totalPages++
	}

	return &AuditLogListResult{
		Logs:       dtos,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.Limit(),
		TotalPages: totalPages,
	}, nil
}

// ListByAdminUser retrieves audit logs for a specific admin user
func (s *AuditService) ListByAdminUser(ctx context.Context, adminUserID uuid.UUID, input AuditLogFilterInput) (*AuditLogListResult, error) {
	input.AdminUserID = &adminUserID
	return s.List(ctx, input)
}

// ListByTarget retrieves audit logs for a specific target entity
func (s *AuditService) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID, input AuditLogFilterInput) (*AuditLogListResult, error) {
	input.TargetType = &targetType
	input.TargetID = &targetID
	return s.List(ctx, input)
}

// toAuditLogDTO converts a domain AuditLog to DTO
func toAuditLogDTO(log *identity.AuditLog) *AuditLogDTO {
	return &AuditLogDTO{
		ID:          log.ID,
		AdminUserID: log.AdminUserID,
		Action:      string(log.Action),
		TargetType:  string(log.TargetType),
		TargetID:    log.TargetID,
		OldValue:    log.OldValue,
		NewValue:    log.NewValue,
		IPAddress:   log.IPAddress,
		UserAgent:   log.UserAgent,
		CreatedAt:   log.CreatedAt,
	}
}

// extractID extracts the ID from an entity if it has one
func extractID(entity interface{}) *uuid.UUID {
	if entity == nil {
		return nil
	}

	// Try to extract ID using type assertion for common patterns
	type hasID interface {
		GetID() uuid.UUID
	}

	if e, ok := entity.(hasID); ok {
		id := e.GetID()
		return &id
	}

	// Try map access
	if m, ok := entity.(map[string]interface{}); ok {
		if idVal, exists := m["id"]; exists {
			if idStr, ok := idVal.(string); ok {
				if id, err := uuid.Parse(idStr); err == nil {
					return &id
				}
			}
			if id, ok := idVal.(uuid.UUID); ok {
				return &id
			}
		}
	}

	return nil
}
