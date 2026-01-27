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

// OverrideService handles flag override management operations
type OverrideService struct {
	flagRepo     featureflag.FeatureFlagRepository
	overrideRepo featureflag.FlagOverrideRepository
	auditLogRepo featureflag.FlagAuditLogRepository
	outboxRepo   shared.OutboxRepository
	logger       *zap.Logger
}

// NewOverrideService creates a new override service
func NewOverrideService(
	flagRepo featureflag.FeatureFlagRepository,
	overrideRepo featureflag.FlagOverrideRepository,
	auditLogRepo featureflag.FlagAuditLogRepository,
	outboxRepo shared.OutboxRepository,
	logger *zap.Logger,
) *OverrideService {
	return &OverrideService{
		flagRepo:     flagRepo,
		overrideRepo: overrideRepo,
		auditLogRepo: auditLogRepo,
		outboxRepo:   outboxRepo,
		logger:       logger,
	}
}

// CreateOverride creates a new flag override
func (s *OverrideService) CreateOverride(ctx context.Context, flagKey string, req dto.CreateOverrideRequest, auditCtx AuditContext) (*dto.OverrideResponse, error) {
	s.logger.Info("Creating flag override",
		zap.String("flag_key", flagKey),
		zap.String("target_type", req.TargetType),
		zap.String("target_id", req.TargetID.String()))

	// Verify the flag exists
	flag, err := s.flagRepo.FindByKey(ctx, flagKey)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("FLAG_NOT_FOUND", "Feature flag not found")
		}
		s.logger.Error("Failed to find flag", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find flag")
	}

	// Verify the flag is not archived
	if flag.IsArchived() {
		return nil, shared.NewDomainError("FLAG_ARCHIVED", "Cannot create override for archived flag")
	}

	// Check if override already exists for this target
	existingOverride, err := s.overrideRepo.FindByFlagKeyAndTarget(
		ctx,
		flagKey,
		featureflag.OverrideTargetType(req.TargetType),
		req.TargetID,
	)
	if err == nil && existingOverride != nil {
		return nil, shared.NewDomainError("OVERRIDE_EXISTS", "Override already exists for this target")
	}

	// Create the override
	override, err := featureflag.NewFlagOverride(
		flagKey,
		featureflag.OverrideTargetType(req.TargetType),
		req.TargetID,
		req.Value.ToDomain(),
		req.Reason,
		req.ExpiresAt,
		auditCtx.UserID,
	)
	if err != nil {
		return nil, err
	}

	// Persist the override
	if err := s.overrideRepo.Create(ctx, override); err != nil {
		s.logger.Error("Failed to create override", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to create override")
	}

	// Create audit log
	if err := s.createOverrideAuditLog(ctx, flagKey, featureflag.AuditActionOverrideAdded, nil, overrideToMap(override), auditCtx); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
	}

	// Publish override created event
	event := featureflag.NewOverrideCreatedEvent(override, flag.ID)
	if err := s.publishEvent(ctx, event); err != nil {
		s.logger.Error("Failed to publish override created event", zap.Error(err))
	}

	s.logger.Info("Flag override created",
		zap.String("id", override.ID.String()),
		zap.String("flag_key", flagKey))

	return dto.ToOverrideResponse(override), nil
}

// ListOverrides retrieves all overrides for a flag
func (s *OverrideService) ListOverrides(ctx context.Context, flagKey string, filter dto.OverrideListFilter) (*dto.OverrideListResponse, error) {
	s.logger.Debug("Listing overrides for flag", zap.String("flag_key", flagKey))

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

	// Get overrides
	overrides, err := s.overrideRepo.FindByFlagKey(ctx, flagKey, domainFilter)
	if err != nil {
		s.logger.Error("Failed to list overrides", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list overrides")
	}

	// Get total count
	total, err := s.overrideRepo.CountByFlagKey(ctx, flagKey)
	if err != nil {
		s.logger.Error("Failed to count overrides", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to count overrides")
	}

	return dto.ToOverrideListResponse(overrides, total, filter.Page, filter.PageSize), nil
}

// GetOverride retrieves a single override by ID
func (s *OverrideService) GetOverride(ctx context.Context, id uuid.UUID) (*dto.OverrideResponse, error) {
	override, err := s.overrideRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil, shared.NewDomainError("OVERRIDE_NOT_FOUND", "Override not found")
		}
		s.logger.Error("Failed to find override", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find override")
	}

	return dto.ToOverrideResponse(override), nil
}

// DeleteOverride deletes a flag override
func (s *OverrideService) DeleteOverride(ctx context.Context, id uuid.UUID, auditCtx AuditContext) error {
	s.logger.Info("Deleting flag override", zap.String("id", id.String()))

	// Find the override
	override, err := s.overrideRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return shared.NewDomainError("OVERRIDE_NOT_FOUND", "Override not found")
		}
		s.logger.Error("Failed to find override", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find override")
	}

	// Find the flag for the event
	flag, err := s.flagRepo.FindByKey(ctx, override.FlagKey)
	if err != nil {
		s.logger.Warn("Flag not found for override, continuing with deletion", zap.Error(err))
	}

	oldValues := overrideToMap(override)

	// Delete the override
	if err := s.overrideRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete override", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to delete override")
	}

	// Create audit log
	if err := s.createOverrideAuditLog(ctx, override.FlagKey, featureflag.AuditActionOverrideRemoved, oldValues, nil, auditCtx); err != nil {
		s.logger.Error("Failed to create audit log", zap.Error(err))
	}

	// Publish override removed event
	if flag != nil {
		event := featureflag.NewOverrideRemovedEvent(override, flag.ID, auditCtx.UserID)
		if err := s.publishEvent(ctx, event); err != nil {
			s.logger.Error("Failed to publish override removed event", zap.Error(err))
		}
	}

	s.logger.Info("Flag override deleted", zap.String("id", id.String()))

	return nil
}

// CleanupExpiredOverrides removes all expired overrides
func (s *OverrideService) CleanupExpiredOverrides(ctx context.Context) (int64, error) {
	s.logger.Info("Cleaning up expired overrides")

	count, err := s.overrideRepo.DeleteExpired(ctx)
	if err != nil {
		s.logger.Error("Failed to cleanup expired overrides", zap.Error(err))
		return 0, shared.NewDomainError("INTERNAL_ERROR", "Failed to cleanup expired overrides")
	}

	s.logger.Info("Expired overrides cleaned up", zap.Int64("count", count))

	return count, nil
}

// createOverrideAuditLog creates an audit log entry for override operations
func (s *OverrideService) createOverrideAuditLog(ctx context.Context, flagKey string, action featureflag.AuditAction, oldValue, newValue map[string]any, auditCtx AuditContext) error {
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

// publishEvent publishes a single event to the outbox
func (s *OverrideService) publishEvent(ctx context.Context, event shared.DomainEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	entry := shared.NewOutboxEntry(uuid.Nil, event, payload)
	return s.outboxRepo.Save(ctx, entry)
}

// overrideToMap converts an override to a map for audit logging
func overrideToMap(override *featureflag.FlagOverride) map[string]any {
	return map[string]any{
		"id":          override.ID.String(),
		"flag_key":    override.FlagKey,
		"target_type": string(override.TargetType),
		"target_id":   override.TargetID.String(),
		"value":       override.Value,
		"reason":      override.Reason,
		"expires_at":  override.ExpiresAt,
	}
}
