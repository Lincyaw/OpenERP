package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/billing"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
	"go.uber.org/zap"
)

// StripeWebhookService handles Stripe webhook events
type StripeWebhookService struct {
	config     *billing.StripeConfig
	tenantRepo identity.TenantRepository
	eventBus   shared.EventBus
	logger     *zap.Logger
}

// StripeWebhookServiceConfig contains configuration for StripeWebhookService
type StripeWebhookServiceConfig struct {
	Config     *billing.StripeConfig
	TenantRepo identity.TenantRepository
	EventBus   shared.EventBus
	Logger     *zap.Logger
}

// NewStripeWebhookService creates a new StripeWebhookService
func NewStripeWebhookService(cfg StripeWebhookServiceConfig) *StripeWebhookService {
	return &StripeWebhookService{
		config:     cfg.Config,
		tenantRepo: cfg.TenantRepo,
		eventBus:   cfg.EventBus,
		logger:     cfg.Logger,
	}
}

// WebhookResult contains the result of processing a webhook
type WebhookResult struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
}

// ProcessWebhook processes a Stripe webhook event
func (s *StripeWebhookService) ProcessWebhook(ctx context.Context, payload []byte, signature string) (*WebhookResult, error) {
	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, signature, s.config.WebhookSecret)
	if err != nil {
		s.logger.Error("Failed to verify webhook signature",
			zap.Error(err))
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	s.logger.Info("Processing Stripe webhook event",
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)))

	result := &WebhookResult{
		EventID:   event.ID,
		EventType: string(event.Type),
		Processed: true,
	}

	// Handle different event types
	switch event.Type {
	case "customer.subscription.created":
		err = s.handleSubscriptionCreated(ctx, event)
	case "customer.subscription.updated":
		err = s.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		err = s.handleSubscriptionDeleted(ctx, event)
	case "invoice.paid":
		err = s.handleInvoicePaid(ctx, event)
	case "invoice.payment_failed":
		err = s.handleInvoicePaymentFailed(ctx, event)
	default:
		s.logger.Debug("Unhandled webhook event type",
			zap.String("event_type", string(event.Type)))
		result.Message = "Event type not handled"
	}

	if err != nil {
		s.logger.Error("Failed to process webhook event",
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.Type)),
			zap.Error(err))
		result.Processed = false
		result.Message = err.Error()
		return result, err
	}

	return result, nil
}

// handleSubscriptionCreated handles customer.subscription.created events
func (s *StripeWebhookService) handleSubscriptionCreated(ctx context.Context, event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	// Safely extract customer ID with nil check
	customerID := ""
	if subscription.Customer != nil {
		customerID = subscription.Customer.ID
	}
	if customerID == "" {
		s.logger.Warn("Subscription has no customer ID, skipping",
			zap.String("subscription_id", subscription.ID))
		return nil
	}

	s.logger.Info("Handling subscription created",
		zap.String("subscription_id", subscription.ID),
		zap.String("customer_id", customerID),
		zap.String("status", string(subscription.Status)))

	// Find tenant by Stripe customer ID
	tenant, err := s.tenantRepo.FindByStripeCustomerID(ctx, customerID)
	if err != nil {
		if err == shared.ErrNotFound {
			// Note: ErrNotFound is not treated as an error because webhooks may arrive
			// before tenant setup is complete, or for customers not in our system.
			// We acknowledge receipt to prevent Stripe retries.
			s.logger.Warn("Tenant not found for Stripe customer",
				zap.String("customer_id", customerID))
			return nil
		}
		return fmt.Errorf("failed to find tenant: %w", err)
	}

	// Update tenant with subscription ID
	tenant.SetStripeSubscriptionID(subscription.ID)

	// Update plan based on subscription metadata
	if planID, ok := subscription.Metadata["plan_id"]; ok {
		plan := identity.TenantPlan(planID)
		if err := tenant.SetPlan(plan); err != nil {
			s.logger.Warn("Failed to set tenant plan",
				zap.String("plan_id", planID),
				zap.Error(err))
		}
	}

	// Update expiration based on subscription period
	if subscription.CurrentPeriodEnd > 0 {
		expiresAt := time.Unix(subscription.CurrentPeriodEnd, 0)
		tenant.SetExpiration(expiresAt)
	}

	// Activate tenant if subscription is active
	if subscription.Status == stripe.SubscriptionStatusActive ||
		subscription.Status == stripe.SubscriptionStatusTrialing {
		if tenant.IsSuspended() || tenant.IsInactive() {
			if err := tenant.Activate(); err != nil {
				s.logger.Warn("Failed to activate tenant", zap.Error(err))
			}
		}
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}

	// Publish domain event
	s.publishSubscriptionEvent(ctx, tenant.ID, "subscription_created", subscription.ID)

	s.logger.Info("Subscription created processed successfully",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("subscription_id", subscription.ID))

	return nil
}

// handleSubscriptionUpdated handles customer.subscription.updated events
func (s *StripeWebhookService) handleSubscriptionUpdated(ctx context.Context, event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	s.logger.Info("Handling subscription updated",
		zap.String("subscription_id", subscription.ID),
		zap.String("status", string(subscription.Status)))

	// Find tenant by subscription ID
	tenant, err := s.tenantRepo.FindByStripeSubscriptionID(ctx, subscription.ID)
	if err != nil {
		if err == shared.ErrNotFound {
			// Try finding by customer ID (with nil check)
			customerID := ""
			if subscription.Customer != nil {
				customerID = subscription.Customer.ID
			}
			if customerID == "" {
				s.logger.Warn("Tenant not found for subscription and no customer ID available",
					zap.String("subscription_id", subscription.ID))
				return nil
			}
			tenant, err = s.tenantRepo.FindByStripeCustomerID(ctx, customerID)
			if err != nil {
				if err == shared.ErrNotFound {
					s.logger.Warn("Tenant not found for subscription",
						zap.String("subscription_id", subscription.ID))
					return nil
				}
				return fmt.Errorf("failed to find tenant: %w", err)
			}
		} else {
			return fmt.Errorf("failed to find tenant: %w", err)
		}
	}

	// Update subscription ID if changed
	if tenant.StripeSubscriptionID != subscription.ID {
		tenant.SetStripeSubscriptionID(subscription.ID)
	}

	// Update plan if changed
	if planID, ok := subscription.Metadata["plan_id"]; ok {
		plan := identity.TenantPlan(planID)
		if string(tenant.Plan) != planID {
			if err := tenant.SetPlan(plan); err != nil {
				s.logger.Warn("Failed to set tenant plan",
					zap.String("plan_id", planID),
					zap.Error(err))
			}
		}
	}

	// Update expiration
	if subscription.CurrentPeriodEnd > 0 {
		expiresAt := time.Unix(subscription.CurrentPeriodEnd, 0)
		tenant.SetExpiration(expiresAt)
	}

	// Handle status changes
	switch subscription.Status {
	case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusTrialing:
		if tenant.IsSuspended() || tenant.IsInactive() {
			if err := tenant.Activate(); err != nil {
				s.logger.Warn("Failed to activate tenant", zap.Error(err))
			}
		}
	case stripe.SubscriptionStatusPastDue, stripe.SubscriptionStatusUnpaid:
		// Keep active but log warning
		s.logger.Warn("Subscription payment issue",
			zap.String("tenant_id", tenant.ID.String()),
			zap.String("status", string(subscription.Status)))
	case stripe.SubscriptionStatusCanceled:
		// Will be handled by subscription.deleted event
		s.logger.Info("Subscription canceled",
			zap.String("tenant_id", tenant.ID.String()))
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}

	// Publish domain event
	s.publishSubscriptionEvent(ctx, tenant.ID, "subscription_updated", subscription.ID)

	s.logger.Info("Subscription updated processed successfully",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("subscription_id", subscription.ID))

	return nil
}

// handleSubscriptionDeleted handles customer.subscription.deleted events
func (s *StripeWebhookService) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	s.logger.Info("Handling subscription deleted",
		zap.String("subscription_id", subscription.ID))

	// Find tenant by subscription ID
	tenant, err := s.tenantRepo.FindByStripeSubscriptionID(ctx, subscription.ID)
	if err != nil {
		if err == shared.ErrNotFound {
			s.logger.Warn("Tenant not found for subscription",
				zap.String("subscription_id", subscription.ID))
			return nil
		}
		return fmt.Errorf("failed to find tenant: %w", err)
	}

	// Clear subscription ID
	tenant.ClearStripeSubscription()

	// Downgrade to free plan
	if err := tenant.SetPlan(identity.TenantPlanFree); err != nil {
		s.logger.Warn("Failed to set tenant to free plan", zap.Error(err))
	}

	// Clear expiration for free plan
	tenant.ClearExpiration()

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}

	// Publish domain event
	s.publishSubscriptionEvent(ctx, tenant.ID, "subscription_deleted", subscription.ID)

	s.logger.Info("Subscription deleted processed successfully",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("subscription_id", subscription.ID))

	return nil
}

// handleInvoicePaid handles invoice.paid events
func (s *StripeWebhookService) handleInvoicePaid(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to unmarshal invoice: %w", err)
	}

	// Safely extract customer ID with nil check
	customerID := ""
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	}
	if customerID == "" {
		s.logger.Warn("Invoice has no customer ID, skipping",
			zap.String("invoice_id", invoice.ID))
		return nil
	}

	s.logger.Info("Handling invoice paid",
		zap.String("invoice_id", invoice.ID),
		zap.String("customer_id", customerID))

	// Skip if not a subscription invoice
	if invoice.Subscription == nil {
		s.logger.Debug("Invoice is not for a subscription, skipping")
		return nil
	}

	// Find tenant by customer ID
	tenant, err := s.tenantRepo.FindByStripeCustomerID(ctx, customerID)
	if err != nil {
		if err == shared.ErrNotFound {
			s.logger.Warn("Tenant not found for customer",
				zap.String("customer_id", customerID))
			return nil
		}
		return fmt.Errorf("failed to find tenant: %w", err)
	}

	// Ensure tenant is active after successful payment
	if tenant.IsSuspended() {
		if err := tenant.Activate(); err != nil {
			s.logger.Warn("Failed to activate tenant after payment", zap.Error(err))
		}
	}

	// Update expiration based on invoice period
	if invoice.PeriodEnd > 0 {
		expiresAt := time.Unix(invoice.PeriodEnd, 0)
		tenant.SetExpiration(expiresAt)
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}

	// Publish domain event
	s.publishPaymentEvent(ctx, tenant.ID, "invoice_paid", invoice.ID)

	s.logger.Info("Invoice paid processed successfully",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("invoice_id", invoice.ID))

	return nil
}

// handleInvoicePaymentFailed handles invoice.payment_failed events
func (s *StripeWebhookService) handleInvoicePaymentFailed(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to unmarshal invoice: %w", err)
	}

	// Safely extract customer ID with nil check
	customerID := ""
	if invoice.Customer != nil {
		customerID = invoice.Customer.ID
	}
	if customerID == "" {
		s.logger.Warn("Invoice has no customer ID, skipping",
			zap.String("invoice_id", invoice.ID))
		return nil
	}

	s.logger.Info("Handling invoice payment failed",
		zap.String("invoice_id", invoice.ID),
		zap.String("customer_id", customerID))

	// Skip if not a subscription invoice
	if invoice.Subscription == nil {
		s.logger.Debug("Invoice is not for a subscription, skipping")
		return nil
	}

	// Find tenant by customer ID
	tenant, err := s.tenantRepo.FindByStripeCustomerID(ctx, customerID)
	if err != nil {
		if err == shared.ErrNotFound {
			s.logger.Warn("Tenant not found for customer",
				zap.String("customer_id", customerID))
			return nil
		}
		return fmt.Errorf("failed to find tenant: %w", err)
	}

	// Suspend tenant due to payment failure
	if !tenant.IsSuspended() {
		if err := tenant.Suspend(); err != nil {
			s.logger.Warn("Failed to suspend tenant after payment failure", zap.Error(err))
		}
	}

	// Save tenant
	if err := s.tenantRepo.Save(ctx, tenant); err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}

	// Publish domain event
	s.publishPaymentEvent(ctx, tenant.ID, "payment_failed", invoice.ID)

	s.logger.Warn("Invoice payment failed - tenant suspended",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("invoice_id", invoice.ID))

	return nil
}

// publishSubscriptionEvent publishes a subscription-related domain event
func (s *StripeWebhookService) publishSubscriptionEvent(ctx context.Context, tenantID uuid.UUID, eventType, subscriptionID string) {
	if s.eventBus == nil {
		return
	}

	event := NewStripeSubscriptionEvent(tenantID, eventType, subscriptionID)
	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Error("Failed to publish subscription event",
			zap.String("event_type", eventType),
			zap.Error(err))
	}
}

// publishPaymentEvent publishes a payment-related domain event
func (s *StripeWebhookService) publishPaymentEvent(ctx context.Context, tenantID uuid.UUID, eventType, invoiceID string) {
	if s.eventBus == nil {
		return
	}

	event := NewStripePaymentEvent(tenantID, eventType, invoiceID)
	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Error("Failed to publish payment event",
			zap.String("event_type", eventType),
			zap.Error(err))
	}
}
