package featureflag

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/erp/backend/internal/application/featureflag/dto"
	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FlagService handles feature flag management operations
type FlagService struct {
	flagRepo     featureflag.FeatureFlagRepository
	auditLogRepo featureflag.FlagAuditLogRepository
	outboxRepo   shared.OutboxRepository
	logger       *zap.Logger
}

// NewFlagService creates a new flag service
func NewFlagService(
	flagRepo featureflag.FeatureFlagRepository,
	auditLogRepo featureflag.FlagAuditLogRepository,
	outboxRepo shared.OutboxRepository,
	logger *zap.Logger,
) *FlagService {
	return &FlagService{
		flagRepo:     flagRepo,
		auditLogRepo: auditLogRepo,
		outboxRepo:   outboxRepo,
		logger:       logger,
	}
}

// AuditContext contains contextual information for audit logging
type AuditContext struct {
	UserID    *uuid.UUID
	IPAddress string
	UserAgent string
}

// CreateFlag creates a new feature flag
func (s *FlagService) CreateFlag(ctx context.Context, req dto.CreateFlagRequest, auditCtx AuditContext) (*dto.FlagResponse, error) {
	s.logger.Info("Creating new feature flag",
		zap.String("key", req.Key),
		zap.String("name", req.Name),
		zap.String("type", req.Type))

	// Check if flag with key already exists
	exists, err := s.flagRepo.ExistsByKey(ctx, req.Key)
	if err != nil {
		s.logger.Error("Failed to check flag existence", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check flag key availability")
	}
	if exists {
		return nil, shared.NewDomainError("FLAG_EXISTS", "Flag with this key already exists")
	}

	// Create the flag
	flag, err := featureflag.NewFeatureFlag(
		req.Key,
		req.Name,
		featureflag.FlagType(req.Type),
		req.DefaultValue.ToDomain(),
		auditCtx.UserID,
	)
	if err != nil {
		return nil, err
	}

	// Set optional description
	if req.Description != "" {
		if err := flag.Update(req.Name, req.Description, auditCtx.UserID); err != nil {
			return nil, err
		}
	}

	// Add targeting rules
	for _, ruleDTO := range req.Rules {
		rule, err := ruleDTO.ToDomain()
		if err != nil {
			return nil, err
		}
		if err := flag.AddRule(rule, auditCtx.UserID); err != nil {
			return nil, err
		}
	}

	// Set tags
	if len(req.Tags) > 0 {
		if err := flag.SetTags(req.Tags, auditCtx.UserID); err != nil {
			return nil, err
		}
	}

	// Persist the flag
	if err := s.flagRepo.Create(ctx, flag); err != nil {
		s.logger.Error("Failed to create flag", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to create flag")
	}

	// Create audit log
	if err := s.createAuditLog(ctx, flag.Key, featureflag.AuditActionCreated, nil, flagToMap(flag), auditCtx); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
		// Non-blocking: don't fail the operation if audit log fails
	}

	// Publish domain events to outbox
	if err := s.publishEvents(ctx, flag); err != nil {
		s.logger.Error("Failed to publish domain events", zap.Error(err))
		// Non-blocking: events will be retried by the outbox processor
	}

	s.logger.Info("Feature flag created successfully",
		zap.String("id", flag.ID.String()),
		zap.String("key", flag.Key))

	return dto.ToFlagResponse(flag), nil
}

// UpdateFlag updates an existing feature flag
func (s *FlagService) UpdateFlag(ctx context.Context, key string, req dto.UpdateFlagRequest, auditCtx AuditContext) (*dto.FlagResponse, error) {
	s.logger.Info("Updating feature flag", zap.String("key", key))

	// Find the flag
	flag, err := s.flagRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
		}
		s.logger.Error("Failed to find flag", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find flag")
	}

	// Store old values for audit log
	oldValues := flagToMap(flag)

	// Update name and description if provided
	if req.Name != nil || req.Description != nil {
		name := flag.Name
		description := flag.Description
		if req.Name != nil {
			name = *req.Name
		}
		if req.Description != nil {
			description = *req.Description
		}
		if err := flag.Update(name, description, auditCtx.UserID); err != nil {
			return nil, err
		}
	}

	// Update default value if provided
	if req.DefaultValue != nil {
		if err := flag.SetDefault(req.DefaultValue.ToDomain(), auditCtx.UserID); err != nil {
			return nil, err
		}
	}

	// Update rules if provided
	if req.Rules != nil {
		// Clear existing rules
		if err := flag.ClearRules(auditCtx.UserID); err != nil {
			return nil, err
		}
		// Add new rules
		for _, ruleDTO := range *req.Rules {
			rule, err := ruleDTO.ToDomain()
			if err != nil {
				return nil, err
			}
			if err := flag.AddRule(rule, auditCtx.UserID); err != nil {
				return nil, err
			}
		}
	}

	// Update tags if provided
	if req.Tags != nil {
		if err := flag.SetTags(*req.Tags, auditCtx.UserID); err != nil {
			return nil, err
		}
	}

	// Persist the flag
	if err := s.flagRepo.Update(ctx, flag); err != nil {
		s.logger.Error("Failed to update flag", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update flag")
	}

	// Create audit log
	if err := s.createAuditLog(ctx, flag.Key, featureflag.AuditActionUpdated, oldValues, flagToMap(flag), auditCtx); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
	}

	// Publish domain events to outbox
	if err := s.publishEvents(ctx, flag); err != nil {
		s.logger.Error("Failed to publish domain events", zap.Error(err))
	}

	s.logger.Info("Feature flag updated", zap.String("key", key))

	return dto.ToFlagResponse(flag), nil
}

// GetFlag retrieves a feature flag by key
func (s *FlagService) GetFlag(ctx context.Context, key string) (*dto.FlagResponse, error) {
	flag, err := s.flagRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
		}
		s.logger.Error("Failed to find flag", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find flag")
	}

	return dto.ToFlagResponse(flag), nil
}

// ListFlags retrieves a paginated list of feature flags
func (s *FlagService) ListFlags(ctx context.Context, filter dto.FlagListFilter) (*dto.FlagListResponse, error) {
	// Build domain filter
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Search:   filter.Search,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}

	var flags []featureflag.FeatureFlag
	var total int64
	var err error

	// Apply status filter if provided
	if filter.Status != nil {
		status := featureflag.FlagStatus(*filter.Status)
		if status.IsValid() {
			flags, err = s.flagRepo.FindByStatus(ctx, status, domainFilter)
			if err == nil {
				total, err = s.flagRepo.CountByStatus(ctx, status)
			}
		} else {
			return nil, shared.NewDomainError("INVALID_STATUS", "Invalid status filter")
		}
	} else if filter.Type != nil {
		flagType := featureflag.FlagType(*filter.Type)
		if flagType.IsValid() {
			flags, err = s.flagRepo.FindByType(ctx, flagType, domainFilter)
			if err == nil {
				total, err = s.flagRepo.Count(ctx, domainFilter)
			}
		} else {
			return nil, shared.NewDomainError("INVALID_TYPE", "Invalid type filter")
		}
	} else if len(filter.Tags) > 0 {
		flags, err = s.flagRepo.FindByTags(ctx, filter.Tags, domainFilter)
		if err == nil {
			total, err = s.flagRepo.Count(ctx, domainFilter)
		}
	} else {
		flags, err = s.flagRepo.FindAll(ctx, domainFilter)
		if err == nil {
			total, err = s.flagRepo.Count(ctx, domainFilter)
		}
	}

	if err != nil {
		s.logger.Error("Failed to list flags", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list flags")
	}

	return dto.ToFlagListResponse(flags, total, filter.Page, filter.PageSize), nil
}

// EnableFlag enables a feature flag
func (s *FlagService) EnableFlag(ctx context.Context, key string, auditCtx AuditContext) error {
	s.logger.Info("Enabling feature flag", zap.String("key", key))

	flag, err := s.flagRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
		}
		s.logger.Error("Failed to find flag", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find flag")
	}

	oldValues := flagToMap(flag)

	if err := flag.Enable(auditCtx.UserID); err != nil {
		return err
	}

	if err := s.flagRepo.Update(ctx, flag); err != nil {
		s.logger.Error("Failed to update flag", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to enable flag")
	}

	// Create audit log
	if err := s.createAuditLog(ctx, flag.Key, featureflag.AuditActionEnabled, oldValues, flagToMap(flag), auditCtx); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
	}

	// Publish domain events
	if err := s.publishEvents(ctx, flag); err != nil {
		s.logger.Error("Failed to publish domain events", zap.Error(err))
	}

	s.logger.Info("Feature flag enabled", zap.String("key", key))

	return nil
}

// DisableFlag disables a feature flag
func (s *FlagService) DisableFlag(ctx context.Context, key string, auditCtx AuditContext) error {
	s.logger.Info("Disabling feature flag", zap.String("key", key))

	flag, err := s.flagRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
		}
		s.logger.Error("Failed to find flag", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find flag")
	}

	oldValues := flagToMap(flag)

	if err := flag.Disable(auditCtx.UserID); err != nil {
		return err
	}

	if err := s.flagRepo.Update(ctx, flag); err != nil {
		s.logger.Error("Failed to update flag", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to disable flag")
	}

	// Create audit log
	if err := s.createAuditLog(ctx, flag.Key, featureflag.AuditActionDisabled, oldValues, flagToMap(flag), auditCtx); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
	}

	// Publish domain events
	if err := s.publishEvents(ctx, flag); err != nil {
		s.logger.Error("Failed to publish domain events", zap.Error(err))
	}

	s.logger.Info("Feature flag disabled", zap.String("key", key))

	return nil
}

// ArchiveFlag archives a feature flag
func (s *FlagService) ArchiveFlag(ctx context.Context, key string, auditCtx AuditContext) error {
	s.logger.Info("Archiving feature flag", zap.String("key", key))

	flag, err := s.flagRepo.FindByKey(ctx, key)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
		}
		s.logger.Error("Failed to find flag", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find flag")
	}

	oldValues := flagToMap(flag)

	if err := flag.Archive(auditCtx.UserID); err != nil {
		return err
	}

	if err := s.flagRepo.Update(ctx, flag); err != nil {
		s.logger.Error("Failed to update flag", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to archive flag")
	}

	// Create audit log
	if err := s.createAuditLog(ctx, flag.Key, featureflag.AuditActionArchived, oldValues, flagToMap(flag), auditCtx); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
	}

	// Publish domain events
	if err := s.publishEvents(ctx, flag); err != nil {
		s.logger.Error("Failed to publish domain events", zap.Error(err))
	}

	s.logger.Info("Feature flag archived", zap.String("key", key))

	return nil
}

// createAuditLog creates an audit log entry
func (s *FlagService) createAuditLog(ctx context.Context, flagKey string, action featureflag.AuditAction, oldValue, newValue map[string]any, auditCtx AuditContext) error {
	auditLog, err := featureflag.NewFlagAuditLog(
		flagKey,
		action,
		oldValue,
		newValue,
		auditCtx.UserID,
		auditCtx.IPAddress,
		auditCtx.UserAgent,
	)
	if err != nil {
		return err
	}

	return s.auditLogRepo.Create(ctx, auditLog)
}

// publishEvents publishes domain events to the outbox
func (s *FlagService) publishEvents(ctx context.Context, flag *featureflag.FeatureFlag) error {
	events := flag.GetDomainEvents()
	if len(events) == 0 {
		return nil
	}

	outboxEntries := make([]*shared.OutboxEntry, 0, len(events))
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			s.logger.Error("Failed to marshal event", zap.Error(err))
			continue
		}
		// Feature flags are global (no tenant), use zero UUID
		entry := shared.NewOutboxEntry(uuid.Nil, event, payload)
		outboxEntries = append(outboxEntries, entry)
	}

	if len(outboxEntries) > 0 {
		if err := s.outboxRepo.Save(ctx, outboxEntries...); err != nil {
			return err
		}
	}

	flag.ClearDomainEvents()
	return nil
}

// GetAuditLogs retrieves audit logs for a feature flag
func (s *FlagService) GetAuditLogs(ctx context.Context, flagKey string, filter dto.AuditLogListFilter) (*dto.AuditLogListResponse, error) {
	s.logger.Debug("Getting audit logs for flag", zap.String("flag_key", flagKey))

	// Verify the flag exists
	_, err := s.flagRepo.FindByKey(ctx, flagKey)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
		}
		s.logger.Error("Failed to find flag", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find flag")
	}

	// Build domain filter
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}

	// Get audit logs
	logs, err := s.auditLogRepo.FindByFlagKey(ctx, flagKey, domainFilter)
	if err != nil {
		s.logger.Error("Failed to get audit logs", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to get audit logs")
	}

	// Get total count
	total, err := s.auditLogRepo.CountByFlagKey(ctx, flagKey)
	if err != nil {
		s.logger.Error("Failed to count audit logs", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to count audit logs")
	}

	return dto.ToAuditLogListResponse(logs, total, filter.Page, filter.PageSize), nil
}

// flagToMap converts a flag to a map for audit logging
func flagToMap(flag *featureflag.FeatureFlag) map[string]any {
	return map[string]any{
		"id":            flag.ID.String(),
		"key":           flag.Key,
		"name":          flag.Name,
		"description":   flag.Description,
		"type":          string(flag.Type),
		"status":        string(flag.Status),
		"default_value": flag.DefaultValue,
		"version":       flag.Version,
	}
}
