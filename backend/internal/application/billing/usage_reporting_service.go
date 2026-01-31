package billing

import (
	"context"
	"fmt"
	"sync"
	"time"

	domainBilling "github.com/erp/backend/internal/domain/billing"
	infraBilling "github.com/erp/backend/internal/infrastructure/billing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UsageReportingService handles reporting usage to Stripe with retry logic
type UsageReportingService struct {
	stripeAdapter    *infraBilling.StripeAdapter
	usageRecordRepo  domainBilling.UsageRecordRepository
	reportLogRepo    infraBilling.UsageReportLogRepository
	subscriptionRepo SubscriptionRepository
	logger           *zap.Logger
	config           UsageReportingConfig
	mu               sync.Mutex
}

// UsageReportingConfig contains configuration for usage reporting
type UsageReportingConfig struct {
	// MaxRetries is the maximum number of retry attempts for failed reports
	MaxRetries int

	// RetryBaseDelay is the base delay between retries (exponential backoff)
	RetryBaseDelay time.Duration

	// RetryMaxDelay is the maximum delay between retries
	RetryMaxDelay time.Duration

	// BatchSize is the number of records to process in a single batch
	BatchSize int

	// ReportingInterval is how often to run the reporting job
	ReportingInterval time.Duration

	// UsageTypes specifies which usage types to report to Stripe
	UsageTypes []domainBilling.UsageType
}

// DefaultUsageReportingConfig returns default configuration
func DefaultUsageReportingConfig() UsageReportingConfig {
	return UsageReportingConfig{
		MaxRetries:        5,
		RetryBaseDelay:    time.Second,
		RetryMaxDelay:     5 * time.Minute,
		BatchSize:         100,
		ReportingInterval: time.Hour,
		UsageTypes: []domainBilling.UsageType{
			domainBilling.UsageTypeAPICalls,
			domainBilling.UsageTypeOrdersCreated,
			domainBilling.UsageTypeStorageBytes,
		},
	}
}

// SubscriptionRepository defines the interface for subscription data access
type SubscriptionRepository interface {
	// FindActiveByTenant finds active subscriptions for a tenant
	FindActiveByTenant(ctx context.Context, tenantID uuid.UUID) (*TenantSubscription, error)

	// FindAllActiveWithMeteredBilling finds all active subscriptions with metered billing
	FindAllActiveWithMeteredBilling(ctx context.Context) ([]*TenantSubscription, error)
}

// TenantSubscription represents a tenant's subscription information
type TenantSubscription struct {
	TenantID           uuid.UUID
	SubscriptionID     string // Stripe subscription ID
	SubscriptionItemID string // Stripe subscription item ID (for usage reporting)
	PlanID             string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
}

// NewUsageReportingService creates a new usage reporting service
func NewUsageReportingService(
	stripeAdapter *infraBilling.StripeAdapter,
	usageRecordRepo domainBilling.UsageRecordRepository,
	reportLogRepo infraBilling.UsageReportLogRepository,
	subscriptionRepo SubscriptionRepository,
	logger *zap.Logger,
	config UsageReportingConfig,
) *UsageReportingService {
	return &UsageReportingService{
		stripeAdapter:    stripeAdapter,
		usageRecordRepo:  usageRecordRepo,
		reportLogRepo:    reportLogRepo,
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
		config:           config,
	}
}

// ReportUsageForTenant reports usage for a specific tenant
func (s *UsageReportingService) ReportUsageForTenant(ctx context.Context, tenantID uuid.UUID) error {
	s.logger.Info("Starting usage reporting for tenant",
		zap.String("tenant_id", tenantID.String()))

	// Get tenant's active subscription
	subscription, err := s.subscriptionRepo.FindActiveByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to find subscription for tenant: %w", err)
	}

	if subscription == nil {
		s.logger.Debug("No active subscription found for tenant",
			zap.String("tenant_id", tenantID.String()))
		return nil
	}

	// Get current billing period
	periodStart := subscription.CurrentPeriodStart
	periodEnd := subscription.CurrentPeriodEnd

	// Report usage for each configured usage type
	for _, usageType := range s.config.UsageTypes {
		if err := s.reportUsageType(ctx, tenantID, subscription, usageType, periodStart, periodEnd); err != nil {
			s.logger.Error("Failed to report usage type",
				zap.String("tenant_id", tenantID.String()),
				zap.String("usage_type", usageType.String()),
				zap.Error(err))
			// Continue with other usage types
		}
	}

	return nil
}

// reportUsageType reports a specific usage type for a tenant
func (s *UsageReportingService) reportUsageType(
	ctx context.Context,
	tenantID uuid.UUID,
	subscription *TenantSubscription,
	usageType domainBilling.UsageType,
	periodStart, periodEnd time.Time,
) error {
	// Get aggregated usage for the period
	totalUsage, err := s.usageRecordRepo.SumByTenantAndType(ctx, tenantID, usageType, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to get usage sum: %w", err)
	}

	if totalUsage == 0 {
		s.logger.Debug("No usage to report",
			zap.String("tenant_id", tenantID.String()),
			zap.String("usage_type", usageType.String()))
		return nil
	}

	// Create usage report log for tracking
	reportLog := infraBilling.NewUsageReportLog(
		tenantID,
		subscription.SubscriptionItemID,
		usageType,
		totalUsage,
		time.Now(),
	)

	// Save the log entry
	if err := s.reportLogRepo.Save(ctx, reportLog); err != nil {
		s.logger.Error("Failed to save usage report log",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		// Continue anyway - we don't want to block reporting
	}

	// Generate idempotency key
	idempotencyKey := infraBilling.GenerateIdempotencyKey(
		tenantID,
		subscription.SubscriptionItemID,
		usageType,
		time.Now().Truncate(time.Hour), // Truncate to hour for deduplication
	)

	// Report to Stripe
	output, err := s.stripeAdapter.ReportUsage(ctx, infraBilling.UsageReportInput{
		TenantID:           tenantID,
		SubscriptionItemID: subscription.SubscriptionItemID,
		Quantity:           totalUsage,
		Timestamp:          time.Now(),
		Action:             "set", // Use "set" to report absolute usage
		IdempotencyKey:     idempotencyKey,
	})

	if err != nil {
		// Mark as failed and schedule for retry
		if reportLog.ID != uuid.Nil {
			if markErr := s.reportLogRepo.MarkAsFailed(ctx, reportLog.ID, err.Error()); markErr != nil {
				s.logger.Error("Failed to mark report as failed",
					zap.String("report_id", reportLog.ID.String()),
					zap.Error(markErr))
			}
		}
		return fmt.Errorf("failed to report usage to Stripe: %w", err)
	}

	// Mark as successful
	if reportLog.ID != uuid.Nil {
		if markErr := s.reportLogRepo.MarkAsSuccess(ctx, reportLog.ID, output.UsageRecordID); markErr != nil {
			s.logger.Error("Failed to mark report as successful",
				zap.String("report_id", reportLog.ID.String()),
				zap.Error(markErr))
		}
	}

	s.logger.Info("Successfully reported usage to Stripe",
		zap.String("tenant_id", tenantID.String()),
		zap.String("usage_type", usageType.String()),
		zap.Int64("quantity", totalUsage),
		zap.String("stripe_record_id", output.UsageRecordID))

	return nil
}

// ReportUsageForAllTenants reports usage for all tenants with metered billing
func (s *UsageReportingService) ReportUsageForAllTenants(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting usage reporting for all tenants")

	// Get all active subscriptions with metered billing
	subscriptions, err := s.subscriptionRepo.FindAllActiveWithMeteredBilling(ctx)
	if err != nil {
		return fmt.Errorf("failed to find active subscriptions: %w", err)
	}

	s.logger.Info("Found subscriptions to process",
		zap.Int("count", len(subscriptions)))

	var errors []error
	for _, sub := range subscriptions {
		if err := s.ReportUsageForTenant(ctx, sub.TenantID); err != nil {
			s.logger.Error("Failed to report usage for tenant",
				zap.String("tenant_id", sub.TenantID.String()),
				zap.Error(err))
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to report usage for %d tenants", len(errors))
	}

	s.logger.Info("Completed usage reporting for all tenants")
	return nil
}

// RetryFailedReports retries failed usage reports
func (s *UsageReportingService) RetryFailedReports(ctx context.Context) error {
	s.logger.Info("Starting retry of failed usage reports")

	// Find pending reports that haven't exceeded max retries
	pendingReports, err := s.reportLogRepo.FindPending(ctx, s.config.MaxRetries)
	if err != nil {
		return fmt.Errorf("failed to find pending reports: %w", err)
	}

	s.logger.Info("Found pending reports to retry",
		zap.Int("count", len(pendingReports)))

	for _, report := range pendingReports {
		if err := s.retryReport(ctx, report); err != nil {
			s.logger.Error("Failed to retry report",
				zap.String("report_id", report.ID.String()),
				zap.Error(err))
		}
	}

	return nil
}

// retryReport retries a single failed report
func (s *UsageReportingService) retryReport(ctx context.Context, report *infraBilling.UsageReportLog) error {
	// Calculate backoff delay
	delay := s.calculateBackoff(report.RetryCount)

	// Check if enough time has passed since last attempt
	if time.Since(report.UpdatedAt) < delay {
		return nil // Not ready for retry yet
	}

	// Increment retry count
	if err := s.reportLogRepo.IncrementRetryCount(ctx, report.ID); err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	// Generate idempotency key
	idempotencyKey := infraBilling.GenerateIdempotencyKey(
		report.TenantID,
		report.SubscriptionItemID,
		report.UsageType,
		report.Timestamp,
	)

	// Retry the report
	output, err := s.stripeAdapter.ReportUsage(ctx, infraBilling.UsageReportInput{
		TenantID:           report.TenantID,
		SubscriptionItemID: report.SubscriptionItemID,
		Quantity:           report.Quantity,
		Timestamp:          report.Timestamp,
		Action:             "set",
		IdempotencyKey:     idempotencyKey,
	})

	if err != nil {
		// Check if we've exceeded max retries
		if report.RetryCount >= s.config.MaxRetries {
			// Mark as abandoned
			if markErr := s.reportLogRepo.MarkAsFailed(ctx, report.ID, "max retries exceeded: "+err.Error()); markErr != nil {
				s.logger.Error("Failed to mark report as abandoned",
					zap.String("report_id", report.ID.String()),
					zap.Error(markErr))
			}
		}
		return fmt.Errorf("retry failed: %w", err)
	}

	// Mark as successful
	if err := s.reportLogRepo.MarkAsSuccess(ctx, report.ID, output.UsageRecordID); err != nil {
		return fmt.Errorf("failed to mark report as successful: %w", err)
	}

	s.logger.Info("Successfully retried usage report",
		zap.String("report_id", report.ID.String()),
		zap.String("stripe_record_id", output.UsageRecordID))

	return nil
}

// calculateBackoff calculates the backoff delay for a retry attempt
func (s *UsageReportingService) calculateBackoff(retryCount int) time.Duration {
	// Prevent overflow - cap at reasonable exponent
	if retryCount > 30 {
		return s.config.RetryMaxDelay
	}

	// Exponential backoff: base * 2^retryCount
	delay := s.config.RetryBaseDelay * time.Duration(1<<uint(retryCount))

	// Cap at max delay
	if delay > s.config.RetryMaxDelay {
		delay = s.config.RetryMaxDelay
	}

	return delay
}

// GetReportingStats returns statistics about usage reporting
func (s *UsageReportingService) GetReportingStats(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*ReportingStats, error) {
	logs, err := s.reportLogRepo.FindByTenant(ctx, tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get report logs: %w", err)
	}

	stats := &ReportingStats{
		TenantID:    tenantID,
		PeriodStart: start,
		PeriodEnd:   end,
	}

	for _, log := range logs {
		stats.TotalReports++
		switch log.Status {
		case infraBilling.UsageReportStatusSuccess:
			stats.SuccessfulReports++
		case infraBilling.UsageReportStatusFailed, infraBilling.UsageReportStatusAbandoned:
			stats.FailedReports++
		case infraBilling.UsageReportStatusPending, infraBilling.UsageReportStatusRetrying:
			stats.PendingReports++
		}
	}

	return stats, nil
}

// ReportingStats contains statistics about usage reporting
type ReportingStats struct {
	TenantID          uuid.UUID
	PeriodStart       time.Time
	PeriodEnd         time.Time
	TotalReports      int
	SuccessfulReports int
	FailedReports     int
	PendingReports    int
}
