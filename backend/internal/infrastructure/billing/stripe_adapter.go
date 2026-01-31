package billing

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/subscription"
	"go.uber.org/zap"
)

// StripeAdapter implements Stripe billing operations for subscription management
type StripeAdapter struct {
	config *StripeConfig
	logger *zap.Logger
}

// NewStripeAdapter creates a new Stripe adapter
func NewStripeAdapter(config *StripeConfig, logger *zap.Logger) (*StripeAdapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Initialize Stripe client
	config.InitStripeClient()

	return &StripeAdapter{
		config: config,
		logger: logger,
	}, nil
}

// CreateCustomer creates a new customer in Stripe
func (a *StripeAdapter) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	a.logger.Debug("Creating Stripe customer",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("email", input.Email))

	// Build customer params
	params := &stripe.CustomerParams{
		Email:       stripe.String(input.Email),
		Name:        stripe.String(input.Name),
		Description: stripe.String(input.Description),
	}

	if input.Phone != "" {
		params.Phone = stripe.String(input.Phone)
	}

	if input.PreferredLocale != "" {
		params.PreferredLocales = stripe.StringSlice([]string{input.PreferredLocale})
	}

	if input.TaxExempt {
		params.TaxExempt = stripe.String("exempt")
	}

	// Add metadata
	params.Metadata = map[string]string{
		"tenant_id": input.TenantID.String(),
	}
	maps.Copy(params.Metadata, input.Metadata)

	// Create customer
	cust, err := customer.New(params)
	if err != nil {
		a.logger.Error("Failed to create Stripe customer",
			zap.String("tenant_id", input.TenantID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to create customer: %w", err)
	}

	a.logger.Info("Created Stripe customer",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("customer_id", cust.ID))

	return &CreateCustomerOutput{
		CustomerID: cust.ID,
		Email:      cust.Email,
		Name:       cust.Name,
		CreatedAt:  time.Unix(cust.Created, 0),
	}, nil
}

// GetCustomer retrieves a customer from Stripe
func (a *StripeAdapter) GetCustomer(ctx context.Context, customerID string) (*CreateCustomerOutput, error) {
	a.logger.Debug("Getting Stripe customer", zap.String("customer_id", customerID))

	cust, err := customer.Get(customerID, nil)
	if err != nil {
		a.logger.Error("Failed to get Stripe customer",
			zap.String("customer_id", customerID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to get customer: %w", err)
	}

	return &CreateCustomerOutput{
		CustomerID: cust.ID,
		Email:      cust.Email,
		Name:       cust.Name,
		CreatedAt:  time.Unix(cust.Created, 0),
	}, nil
}

// UpdateCustomer updates a customer in Stripe
func (a *StripeAdapter) UpdateCustomer(ctx context.Context, customerID string, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	a.logger.Debug("Updating Stripe customer",
		zap.String("customer_id", customerID),
		zap.String("email", input.Email))

	params := &stripe.CustomerParams{
		Email:       stripe.String(input.Email),
		Name:        stripe.String(input.Name),
		Description: stripe.String(input.Description),
	}

	if input.Phone != "" {
		params.Phone = stripe.String(input.Phone)
	}

	// Update metadata
	if len(input.Metadata) > 0 {
		params.Metadata = input.Metadata
	}

	cust, err := customer.Update(customerID, params)
	if err != nil {
		a.logger.Error("Failed to update Stripe customer",
			zap.String("customer_id", customerID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to update customer: %w", err)
	}

	return &CreateCustomerOutput{
		CustomerID: cust.ID,
		Email:      cust.Email,
		Name:       cust.Name,
		CreatedAt:  time.Unix(cust.Created, 0),
	}, nil
}

// DeleteCustomer deletes a customer from Stripe
func (a *StripeAdapter) DeleteCustomer(ctx context.Context, customerID string) error {
	a.logger.Debug("Deleting Stripe customer", zap.String("customer_id", customerID))

	_, err := customer.Del(customerID, nil)
	if err != nil {
		a.logger.Error("Failed to delete Stripe customer",
			zap.String("customer_id", customerID),
			zap.Error(err))
		return fmt.Errorf("stripe: failed to delete customer: %w", err)
	}

	a.logger.Info("Deleted Stripe customer", zap.String("customer_id", customerID))
	return nil
}

// CreateSubscription creates a new subscription in Stripe
func (a *StripeAdapter) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*CreateSubscriptionOutput, error) {
	a.logger.Debug("Creating Stripe subscription",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("customer_id", input.CustomerID),
		zap.String("plan_id", input.PlanID))

	// Get price ID from config if not provided
	priceID := input.PriceID
	if priceID == "" {
		var err error
		priceID, err = a.config.GetPriceID(input.PlanID)
		if err != nil {
			return nil, err
		}
	}

	// Free plan doesn't need a Stripe subscription
	if priceID == "" && input.PlanID == "free" {
		return &CreateSubscriptionOutput{
			SubscriptionID: "",
			CustomerID:     input.CustomerID,
			Status:         SubscriptionStatusActive,
		}, nil
	}

	// Build subscription params
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(input.CustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
	}

	// Set trial period
	if input.TrialDays > 0 {
		params.TrialPeriodDays = stripe.Int64(int64(input.TrialDays))
	}

	// Set payment behavior
	params.PaymentBehavior = stripe.String("default_incomplete")
	params.AddExpand("latest_invoice.payment_intent")

	// Set collection method
	if input.CollectionMethod != "" {
		params.CollectionMethod = stripe.String(input.CollectionMethod)
	}

	// Set default payment method
	if input.PaymentMethod != "" {
		params.DefaultPaymentMethod = stripe.String(input.PaymentMethod)
	}

	// Add metadata
	params.Metadata = map[string]string{
		"tenant_id": input.TenantID.String(),
		"plan_id":   input.PlanID,
	}
	for k, v := range input.Metadata {
		params.Metadata[k] = v
	}

	// Create subscription
	sub, err := subscription.New(params)
	if err != nil {
		a.logger.Error("Failed to create Stripe subscription",
			zap.String("tenant_id", input.TenantID.String()),
			zap.String("customer_id", input.CustomerID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to create subscription: %w", err)
	}

	a.logger.Info("Created Stripe subscription",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("subscription_id", sub.ID),
		zap.String("status", string(sub.Status)))

	output := &CreateSubscriptionOutput{
		SubscriptionID:     sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             mapStripeSubscriptionStatus(sub.Status),
		CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(sub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
	}

	// Set trial dates if present
	if sub.TrialStart > 0 {
		t := time.Unix(sub.TrialStart, 0)
		output.TrialStart = &t
	}
	if sub.TrialEnd > 0 {
		t := time.Unix(sub.TrialEnd, 0)
		output.TrialEnd = &t
	}

	// Get client secret for incomplete subscriptions
	if sub.LatestInvoice != nil {
		output.LatestInvoiceID = sub.LatestInvoice.ID
		if sub.LatestInvoice.PaymentIntent != nil {
			output.ClientSecret = sub.LatestInvoice.PaymentIntent.ClientSecret
		}
	}

	return output, nil
}

// UpdateSubscription updates an existing subscription (upgrade/downgrade)
func (a *StripeAdapter) UpdateSubscription(ctx context.Context, input UpdateSubscriptionInput) (*UpdateSubscriptionOutput, error) {
	a.logger.Debug("Updating Stripe subscription",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("subscription_id", input.SubscriptionID),
		zap.String("new_plan_id", input.NewPlanID))

	// Get current subscription to find the item ID
	sub, err := subscription.Get(input.SubscriptionID, nil)
	if err != nil {
		a.logger.Error("Failed to get Stripe subscription",
			zap.String("subscription_id", input.SubscriptionID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to get subscription: %w", err)
	}

	// Get the subscription item ID (assuming single item subscription)
	if len(sub.Items.Data) == 0 {
		return nil, fmt.Errorf("stripe: subscription has no items")
	}
	itemID := sub.Items.Data[0].ID
	previousPriceID := sub.Items.Data[0].Price.ID

	// Get new price ID
	newPriceID := input.NewPriceID
	if newPriceID == "" && input.NewPlanID != "" {
		newPriceID, err = a.config.GetPriceID(input.NewPlanID)
		if err != nil {
			return nil, err
		}
	}

	// Build update params
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(itemID),
				Price: stripe.String(newPriceID),
			},
		},
		CancelAtPeriodEnd: stripe.Bool(input.CancelAtPeriodEnd),
	}

	// Set proration behavior
	if input.ProrationBehavior != "" {
		params.ProrationBehavior = stripe.String(input.ProrationBehavior)
	} else {
		params.ProrationBehavior = stripe.String("create_prorations")
	}

	// Update metadata
	if len(input.Metadata) > 0 {
		params.Metadata = input.Metadata
	}
	if input.NewPlanID != "" {
		if params.Metadata == nil {
			params.Metadata = make(map[string]string)
		}
		params.Metadata["plan_id"] = input.NewPlanID
	}

	// Update subscription
	updatedSub, err := subscription.Update(input.SubscriptionID, params)
	if err != nil {
		a.logger.Error("Failed to update Stripe subscription",
			zap.String("subscription_id", input.SubscriptionID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to update subscription: %w", err)
	}

	a.logger.Info("Updated Stripe subscription",
		zap.String("subscription_id", updatedSub.ID),
		zap.String("previous_price", previousPriceID),
		zap.String("new_price", newPriceID))

	return &UpdateSubscriptionOutput{
		SubscriptionID:     updatedSub.ID,
		Status:             mapStripeSubscriptionStatus(updatedSub.Status),
		CurrentPeriodStart: time.Unix(updatedSub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(updatedSub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:  updatedSub.CancelAtPeriodEnd,
		PreviousPriceID:    previousPriceID,
		NewPriceID:         newPriceID,
	}, nil
}

// CancelSubscription cancels a subscription
func (a *StripeAdapter) CancelSubscription(ctx context.Context, input CancelSubscriptionInput) (*CancelSubscriptionOutput, error) {
	a.logger.Debug("Canceling Stripe subscription",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("subscription_id", input.SubscriptionID),
		zap.Bool("cancel_at_period_end", input.CancelAtPeriodEnd))

	var sub *stripe.Subscription
	var err error

	if input.CancelAtPeriodEnd {
		// Cancel at end of billing period
		params := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}
		if input.Reason != "" {
			params.CancellationDetails = &stripe.SubscriptionCancellationDetailsParams{
				Comment: stripe.String(input.Reason),
			}
		}
		sub, err = subscription.Update(input.SubscriptionID, params)
	} else {
		// Cancel immediately
		params := &stripe.SubscriptionCancelParams{}
		if input.Reason != "" {
			params.CancellationDetails = &stripe.SubscriptionCancelCancellationDetailsParams{
				Comment: stripe.String(input.Reason),
			}
		}
		sub, err = subscription.Cancel(input.SubscriptionID, params)
	}

	if err != nil {
		a.logger.Error("Failed to cancel Stripe subscription",
			zap.String("subscription_id", input.SubscriptionID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to cancel subscription: %w", err)
	}

	a.logger.Info("Canceled Stripe subscription",
		zap.String("subscription_id", sub.ID),
		zap.String("status", string(sub.Status)),
		zap.Bool("cancel_at_period_end", sub.CancelAtPeriodEnd))

	output := &CancelSubscriptionOutput{
		SubscriptionID:    sub.ID,
		Status:            mapStripeSubscriptionStatus(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
		CurrentPeriodEnd:  time.Unix(sub.CurrentPeriodEnd, 0),
	}

	if sub.CanceledAt > 0 {
		t := time.Unix(sub.CanceledAt, 0)
		output.CanceledAt = &t
	}

	return output, nil
}

// GetSubscriptionStatus retrieves the current status of a subscription
func (a *StripeAdapter) GetSubscriptionStatus(ctx context.Context, input GetSubscriptionStatusInput) (*GetSubscriptionStatusOutput, error) {
	a.logger.Debug("Getting Stripe subscription status",
		zap.String("tenant_id", input.TenantID.String()),
		zap.String("subscription_id", input.SubscriptionID))

	params := &stripe.SubscriptionParams{}
	params.AddExpand("default_payment_method")
	params.AddExpand("latest_invoice")

	sub, err := subscription.Get(input.SubscriptionID, params)
	if err != nil {
		a.logger.Error("Failed to get Stripe subscription",
			zap.String("subscription_id", input.SubscriptionID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to get subscription: %w", err)
	}

	output := &GetSubscriptionStatusOutput{
		SubscriptionID:     sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             mapStripeSubscriptionStatus(sub.Status),
		CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(sub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
		StartDate:          time.Unix(sub.StartDate, 0),
		DaysUntilDue:       int(sub.DaysUntilDue),
	}

	// Get price and product info
	if len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		output.PriceID = item.Price.ID
		if item.Price.Product != nil {
			output.ProductID = item.Price.Product.ID
		}
	}

	// Set trial dates
	if sub.TrialStart > 0 {
		t := time.Unix(sub.TrialStart, 0)
		output.TrialStart = &t
	}
	if sub.TrialEnd > 0 {
		t := time.Unix(sub.TrialEnd, 0)
		output.TrialEnd = &t
	}

	// Set cancel dates
	if sub.CancelAt > 0 {
		t := time.Unix(sub.CancelAt, 0)
		output.CancelAt = &t
	}
	if sub.CanceledAt > 0 {
		t := time.Unix(sub.CanceledAt, 0)
		output.CanceledAt = &t
	}
	if sub.EndedAt > 0 {
		t := time.Unix(sub.EndedAt, 0)
		output.EndedAt = &t
	}

	// Get latest invoice ID
	if sub.LatestInvoice != nil {
		output.LatestInvoiceID = sub.LatestInvoice.ID
	}

	// Get default payment method
	if sub.DefaultPaymentMethod != nil {
		output.DefaultPaymentMethod = sub.DefaultPaymentMethod.ID
	}

	return output, nil
}

// ListSubscriptions lists all subscriptions for a customer
func (a *StripeAdapter) ListSubscriptions(ctx context.Context, customerID string) ([]*GetSubscriptionStatusOutput, error) {
	a.logger.Debug("Listing Stripe subscriptions", zap.String("customer_id", customerID))

	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}
	params.AddExpand("data.default_payment_method")

	var subscriptions []*GetSubscriptionStatusOutput
	iter := subscription.List(params)
	for iter.Next() {
		sub := iter.Subscription()
		output := &GetSubscriptionStatusOutput{
			SubscriptionID:     sub.ID,
			CustomerID:         sub.Customer.ID,
			Status:             mapStripeSubscriptionStatus(sub.Status),
			CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
			CurrentPeriodEnd:   time.Unix(sub.CurrentPeriodEnd, 0),
			CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
			StartDate:          time.Unix(sub.StartDate, 0),
		}

		if len(sub.Items.Data) > 0 {
			output.PriceID = sub.Items.Data[0].Price.ID
		}

		subscriptions = append(subscriptions, output)
	}

	if err := iter.Err(); err != nil {
		a.logger.Error("Failed to list Stripe subscriptions",
			zap.String("customer_id", customerID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to list subscriptions: %w", err)
	}

	return subscriptions, nil
}

// ResumeSubscription resumes a paused or canceled-at-period-end subscription
func (a *StripeAdapter) ResumeSubscription(ctx context.Context, tenantID uuid.UUID, subscriptionID string) (*GetSubscriptionStatusOutput, error) {
	a.logger.Debug("Resuming Stripe subscription",
		zap.String("tenant_id", tenantID.String()),
		zap.String("subscription_id", subscriptionID))

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}

	sub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		a.logger.Error("Failed to resume Stripe subscription",
			zap.String("subscription_id", subscriptionID),
			zap.Error(err))
		return nil, fmt.Errorf("stripe: failed to resume subscription: %w", err)
	}

	a.logger.Info("Resumed Stripe subscription",
		zap.String("subscription_id", sub.ID),
		zap.String("status", string(sub.Status)))

	return &GetSubscriptionStatusOutput{
		SubscriptionID:     sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             mapStripeSubscriptionStatus(sub.Status),
		CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(sub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
	}, nil
}

// mapStripeSubscriptionStatus maps Stripe subscription status to our internal status
func mapStripeSubscriptionStatus(status stripe.SubscriptionStatus) SubscriptionStatus {
	switch status {
	case stripe.SubscriptionStatusActive:
		return SubscriptionStatusActive
	case stripe.SubscriptionStatusPastDue:
		return SubscriptionStatusPastDue
	case stripe.SubscriptionStatusCanceled:
		return SubscriptionStatusCanceled
	case stripe.SubscriptionStatusIncomplete:
		return SubscriptionStatusIncomplete
	case stripe.SubscriptionStatusIncompleteExpired:
		return SubscriptionStatusIncompleteExpired
	case stripe.SubscriptionStatusTrialing:
		return SubscriptionStatusTrialing
	case stripe.SubscriptionStatusUnpaid:
		return SubscriptionStatusUnpaid
	case stripe.SubscriptionStatusPaused:
		return SubscriptionStatusPaused
	default:
		return SubscriptionStatus(status)
	}
}
