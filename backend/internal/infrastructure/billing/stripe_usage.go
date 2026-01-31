package billing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/usagerecord"
	"go.uber.org/zap"
)

// UsageReportInput contains input for reporting usage to Stripe
type UsageReportInput struct {
	TenantID           uuid.UUID // The tenant this usage belongs to
	SubscriptionItemID string    // Stripe subscription item ID (si_xxx)
	Quantity           int64     // Amount of usage to report
	Timestamp          time.Time // When the usage occurred (optional, defaults to now)
	Action             string    // "increment" (default) or "set"
	IdempotencyKey     string    // Optional idempotency key for deduplication
}

// UsageReportOutput contains the result of reporting usage to Stripe
type UsageReportOutput struct {
	UsageRecordID      string    // Stripe usage record ID
	SubscriptionItemID string    // Stripe subscription item ID
	Quantity           int64     // Reported quantity
	Timestamp          time.Time // When the usage was recorded
	Action             string    // Action taken ("increment" or "set")
}

// UsageReportBatchInput contains input for batch usage reporting
type UsageReportBatchInput struct {
	TenantID uuid.UUID
	Records  []UsageReportRecord
}

// UsageReportRecord represents a single usage record in a batch
type UsageReportRecord struct {
	SubscriptionItemID string
	Quantity           int64
	Timestamp          time.Time
	Action             string
	IdempotencyKey     string
}

// UsageReportBatchOutput contains the result of batch usage reporting
type UsageReportBatchOutput struct {
	Successful []UsageReportOutput
	Failed     []UsageReportError
}

// UsageReportError represents a failed usage report
type UsageReportError struct {
	SubscriptionItemID string
	Error              error
	RetryAfter         *time.Duration // Suggested retry delay if rate limited
}

// UsageReportLog represents a log entry for usage reporting (for reconciliation)
type UsageReportLog struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	SubscriptionItemID string
	UsageType          billing.UsageType
	Quantity           int64
	Timestamp          time.Time
	StripeRecordID     string
	Status             UsageReportStatus
	ErrorMessage       string
	RetryCount         int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// UsageReportStatus represents the status of a usage report
type UsageReportStatus string

const (
	UsageReportStatusPending   UsageReportStatus = "pending"
	UsageReportStatusSuccess   UsageReportStatus = "success"
	UsageReportStatusFailed    UsageReportStatus = "failed"
	UsageReportStatusRetrying  UsageReportStatus = "retrying"
	UsageReportStatusAbandoned UsageReportStatus = "abandoned"
)

// String returns the string representation of UsageReportStatus
func (s UsageReportStatus) String() string {
	return string(s)
}

// ReportUsage reports usage to Stripe for metered billing
func (a *StripeAdapter) ReportUsage(ctx context.Context, input UsageReportInput) (*UsageReportOutput, error) {
	a.logger.Debug("Reporting usage to Stripe",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("subscription_item_id", input.SubscriptionItemID),
		zap.Int64("quantity", input.Quantity))

	// Validate input
	if input.SubscriptionItemID == "" {
		return nil, fmt.Errorf("stripe: subscription item ID is required")
	}
	if input.Quantity < 0 {
		return nil, fmt.Errorf("stripe: quantity cannot be negative")
	}

	// Build usage record params
	params := &stripe.UsageRecordParams{
		SubscriptionItem: stripe.String(input.SubscriptionItemID),
		Quantity:         stripe.Int64(input.Quantity),
	}

	// Set timestamp if provided
	if !input.Timestamp.IsZero() {
		params.Timestamp = stripe.Int64(input.Timestamp.Unix())
	}

	// Set action (default to increment)
	action := input.Action
	if action == "" {
		action = "increment"
	}
	params.Action = stripe.String(action)

	// Set idempotency key if provided
	if input.IdempotencyKey != "" {
		params.SetIdempotencyKey(input.IdempotencyKey)
	}

	// Create usage record
	record, err := usagerecord.New(params)
	if err != nil {
		a.logger.Error("Failed to report usage to Stripe",
			zap.String("tenant_id", input.TenantID.String()),
			zap.String("subscription_item_id", input.SubscriptionItemID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to report usage: %w", err)
	}

	a.logger.Info("Reported usage to Stripe",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("usage_record_id", record.ID),
		zap.String("subscription_item_id", record.SubscriptionItem),
		zap.Int64("quantity", record.Quantity))

	return &UsageReportOutput{
		UsageRecordID:      record.ID,
		SubscriptionItemID: record.SubscriptionItem,
		Quantity:           record.Quantity,
		Timestamp:          time.Unix(record.Timestamp, 0),
		Action:             action,
	}, nil
}

// ReportUsageBatch reports multiple usage records to Stripe
// Note: Stripe doesn't have a native batch API for usage records,
// so this method reports them sequentially with error handling
func (a *StripeAdapter) ReportUsageBatch(ctx context.Context, input UsageReportBatchInput) (*UsageReportBatchOutput, error) {
	a.logger.Debug("Reporting batch usage to Stripe",
		zap.String("tenant_id", input.TenantID.String()),
		zap.Int("record_count", len(input.Records)))

	output := &UsageReportBatchOutput{
		Successful: make([]UsageReportOutput, 0, len(input.Records)),
		Failed:     make([]UsageReportError, 0),
	}

	for _, record := range input.Records {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return output, ctx.Err()
		default:
		}

		result, err := a.ReportUsage(ctx, UsageReportInput{
			TenantID:           input.TenantID,
			SubscriptionItemID: record.SubscriptionItemID,
			Quantity:           record.Quantity,
			Timestamp:          record.Timestamp,
			Action:             record.Action,
			IdempotencyKey:     record.IdempotencyKey,
		})

		if err != nil {
			reportErr := UsageReportError{
				SubscriptionItemID: record.SubscriptionItemID,
				Error:              err,
			}

			// Check if rate limited and extract retry-after
			if stripeErr, ok := err.(*stripe.Error); ok {
				if stripeErr.HTTPStatusCode == 429 {
					// Default retry after 1 second for rate limits
					retryAfter := time.Second
					reportErr.RetryAfter = &retryAfter
				}
			}

			output.Failed = append(output.Failed, reportErr)
			continue
		}

		output.Successful = append(output.Successful, *result)
	}

	a.logger.Info("Completed batch usage reporting",
		zap.String("tenant_id", input.TenantID.String()),
		zap.Int("successful", len(output.Successful)),
		zap.Int("failed", len(output.Failed)))

	return output, nil
}

// GetSubscriptionItemID retrieves the subscription item ID for a given subscription
// This is needed because usage records are reported against subscription items, not subscriptions
func (a *StripeAdapter) GetSubscriptionItemID(ctx context.Context, subscriptionID string) (string, error) {
	a.logger.Debug("Getting subscription item ID",
		zap.String("subscription_id", subscriptionID))

	// Get subscription to find the item ID
	input := GetSubscriptionStatusInput{
		SubscriptionID: subscriptionID,
	}

	status, err := a.GetSubscriptionStatus(ctx, input)
	if err != nil {
		return "", fmt.Errorf("stripe: failed to get subscription: %w", err)
	}

	// For metered billing, we need the subscription item ID
	// This assumes a single-item subscription (most common case)
	// For multi-item subscriptions, you'd need to identify the correct item
	if status.PriceID == "" {
		return "", fmt.Errorf("stripe: subscription has no price/item")
	}

	// The subscription item ID is not directly available from GetSubscriptionStatus
	// We need to get it from the subscription directly
	sub, err := a.getSubscriptionWithItems(ctx, subscriptionID)
	if err != nil {
		return "", err
	}

	if len(sub.Items.Data) == 0 {
		return "", fmt.Errorf("stripe: subscription has no items")
	}

	return sub.Items.Data[0].ID, nil
}

// getSubscriptionWithItems retrieves a subscription with its items
func (a *StripeAdapter) getSubscriptionWithItems(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{}
	params.AddExpand("items")

	sub, err := subscription.Get(subscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: failed to get subscription: %w", err)
	}

	return sub, nil
}

// UsageReportLogRepository defines the interface for persisting usage report logs
type UsageReportLogRepository interface {
	// Save persists a usage report log
	Save(ctx context.Context, log *UsageReportLog) error

	// Update updates an existing usage report log
	Update(ctx context.Context, log *UsageReportLog) error

	// FindByID retrieves a usage report log by ID
	FindByID(ctx context.Context, id uuid.UUID) (*UsageReportLog, error)

	// FindPending retrieves all pending usage reports for retry
	FindPending(ctx context.Context, maxRetries int) ([]*UsageReportLog, error)

	// FindByTenant retrieves usage report logs for a tenant
	FindByTenant(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]*UsageReportLog, error)

	// FindByStatus retrieves usage report logs by status
	FindByStatus(ctx context.Context, status UsageReportStatus, limit int) ([]*UsageReportLog, error)

	// MarkAsSuccess marks a usage report as successful
	MarkAsSuccess(ctx context.Context, id uuid.UUID, stripeRecordID string) error

	// MarkAsFailed marks a usage report as failed
	MarkAsFailed(ctx context.Context, id uuid.UUID, errorMessage string) error

	// IncrementRetryCount increments the retry count for a usage report
	IncrementRetryCount(ctx context.Context, id uuid.UUID) error
}

// NewUsageReportLog creates a new usage report log entry
func NewUsageReportLog(
	tenantID uuid.UUID,
	subscriptionItemID string,
	usageType billing.UsageType,
	quantity int64,
	timestamp time.Time,
) *UsageReportLog {
	now := time.Now()
	return &UsageReportLog{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		SubscriptionItemID: subscriptionItemID,
		UsageType:          usageType,
		Quantity:           quantity,
		Timestamp:          timestamp,
		Status:             UsageReportStatusPending,
		RetryCount:         0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// GenerateIdempotencyKey generates a unique idempotency key for a usage report
func GenerateIdempotencyKey(tenantID uuid.UUID, subscriptionItemID string, usageType billing.UsageType, timestamp time.Time) string {
	// Format: tenant_id:subscription_item_id:usage_type:timestamp_unix
	return fmt.Sprintf("%s:%s:%s:%d",
		tenantID.String(),
		subscriptionItemID,
		usageType.String(),
		timestamp.Unix(),
	)
}

// ParseIdempotencyKey parses an idempotency key back to its components
func ParseIdempotencyKey(key string) (tenantID uuid.UUID, subscriptionItemID string, usageType billing.UsageType, timestamp time.Time, err error) {
	parts := strings.SplitN(key, ":", 4)

	if len(parts) != 4 {
		return uuid.Nil, "", "", time.Time{}, fmt.Errorf("invalid idempotency key format")
	}

	tenantID, err = uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, "", "", time.Time{}, fmt.Errorf("invalid tenant ID in idempotency key: %w", err)
	}

	subscriptionItemID = parts[1]

	usageType, err = billing.ParseUsageType(parts[2])
	if err != nil {
		return uuid.Nil, "", "", time.Time{}, fmt.Errorf("invalid usage type in idempotency key: %w", err)
	}

	unixTime, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return uuid.Nil, "", "", time.Time{}, fmt.Errorf("invalid timestamp in idempotency key: %w", err)
	}
	timestamp = time.Unix(unixTime, 0)

	return tenantID, subscriptionItemID, usageType, timestamp, nil
}
