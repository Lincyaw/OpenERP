package billing

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/billing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v81"
	"go.uber.org/zap"
)

// MockTenantRepository is a mock implementation of identity.TenantRepository
type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByCode(ctx context.Context, code string) (*identity.Tenant, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByDomain(ctx context.Context, domain string) (*identity.Tenant, error) {
	args := m.Called(ctx, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindAll(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByStatus(ctx context.Context, status identity.TenantStatus, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, status, filter)
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByPlan(ctx context.Context, plan identity.TenantPlan, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, plan, filter)
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindActive(ctx context.Context, filter shared.Filter) ([]identity.Tenant, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindTrialExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindSubscriptionExpiring(ctx context.Context, withinDays int) ([]identity.Tenant, error) {
	args := m.Called(ctx, withinDays)
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]identity.Tenant, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) Save(ctx context.Context, tenant *identity.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantRepository) Count(ctx context.Context, filter shared.Filter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTenantRepository) CountByStatus(ctx context.Context, status identity.TenantStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTenantRepository) CountByPlan(ctx context.Context, plan identity.TenantPlan) (int64, error) {
	args := m.Called(ctx, plan)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTenantRepository) ExistsByCode(ctx context.Context, code string) (bool, error) {
	args := m.Called(ctx, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockTenantRepository) ExistsByDomain(ctx context.Context, domain string) (bool, error) {
	args := m.Called(ctx, domain)
	return args.Bool(0), args.Error(1)
}

func (m *MockTenantRepository) FindByStripeCustomerID(ctx context.Context, customerID string) (*identity.Tenant, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

func (m *MockTenantRepository) FindByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*identity.Tenant, error) {
	args := m.Called(ctx, subscriptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Tenant), args.Error(1)
}

// Helper function to create a test tenant for webhook tests
func createWebhookTestTenant(t *testing.T) *identity.Tenant {
	tenant, err := identity.NewTenant("TEST001", "Test Tenant")
	assert.NoError(t, err)
	tenant.SetStripeCustomerID("cus_test123")
	tenant.SetStripeSubscriptionID("sub_test123")
	return tenant
}

// Helper function to create a test service
func createWebhookTestService(t *testing.T, mockRepo *MockTenantRepository) *StripeWebhookService {
	logger, _ := zap.NewDevelopment()
	config := &billing.StripeConfig{
		SecretKey:       "sk_test_xxx",
		WebhookSecret:   "whsec_test_xxx",
		IsTestMode:      true,
		DefaultCurrency: "cny",
		PriceIDs: map[string]string{
			"free":       "",
			"basic":      "price_basic",
			"pro":        "price_pro",
			"enterprise": "price_ent",
		},
	}

	return NewStripeWebhookService(StripeWebhookServiceConfig{
		Config:     config,
		TenantRepo: mockRepo,
		EventBus:   nil,
		Logger:     logger,
	})
}

func TestStripeWebhookService_ProcessWebhook_InvalidSignature(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)

	// Test with invalid signature
	payload := []byte(`{"type": "customer.subscription.created"}`)
	signature := "invalid_signature"

	result, err := service.ProcessWebhook(context.Background(), payload, signature)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "webhook signature verification failed")
}

func TestStripeWebhookService_handleSubscriptionCreated(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)

	// Create subscription event data
	subscription := stripe.Subscription{
		ID: "sub_new123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
		Metadata: map[string]string{
			"plan_id": "pro",
		},
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.created",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	// Setup expectations
	mockRepo.On("FindByStripeCustomerID", ctx, "cus_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	// Call the handler directly
	err := service.handleSubscriptionCreated(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionCreated_TenantNotFound(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	subscription := stripe.Subscription{
		ID: "sub_new123",
		Customer: &stripe.Customer{
			ID: "cus_unknown",
		},
		Status: stripe.SubscriptionStatusActive,
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.created",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	// Setup expectations - tenant not found
	mockRepo.On("FindByStripeCustomerID", ctx, "cus_unknown").Return(nil, shared.ErrNotFound)

	// Call the handler directly - should not error, just skip
	err := service.handleSubscriptionCreated(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionUpdated(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)

	subscription := stripe.Subscription{
		ID: "sub_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
		Metadata: map[string]string{
			"plan_id": "enterprise",
		},
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	// Setup expectations
	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	err := service.handleSubscriptionUpdated(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionDeleted(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)
	// Set a paid plan before deletion
	_ = tenant.SetPlan(identity.TenantPlanPro)

	subscription := stripe.Subscription{
		ID: "sub_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Status: stripe.SubscriptionStatusCanceled,
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	// Setup expectations
	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	err := service.handleSubscriptionDeleted(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaid(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)
	// Suspend tenant to test reactivation
	_ = tenant.Suspend()

	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test123",
		},
		PeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.paid",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	// Setup expectations
	mockRepo.On("FindByStripeCustomerID", ctx, "cus_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	err := service.handleInvoicePaid(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaid_NonSubscriptionInvoice(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	// Invoice without subscription (one-time payment)
	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Subscription: nil, // No subscription
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.paid",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	// Should skip without calling repo
	err := service.handleInvoicePaid(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "FindByStripeCustomerID")
}

func TestStripeWebhookService_handleInvoicePaymentFailed(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)

	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test123",
		},
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.payment_failed",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	// Setup expectations
	mockRepo.On("FindByStripeCustomerID", ctx, "cus_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	err := service.handleInvoicePaymentFailed(ctx, event)

	assert.NoError(t, err)
	// Verify tenant was suspended
	assert.True(t, tenant.IsSuspended())
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaymentFailed_AlreadySuspended(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)
	// Already suspended
	_ = tenant.Suspend()

	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test123",
		},
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.payment_failed",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	// Setup expectations
	mockRepo.On("FindByStripeCustomerID", ctx, "cus_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	err := service.handleInvoicePaymentFailed(ctx, event)

	assert.NoError(t, err)
	assert.True(t, tenant.IsSuspended())
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionUpdated_StatusChanges(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	t.Run("activates suspended tenant on active status", func(t *testing.T) {
		tenant := createWebhookTestTenant(t)
		_ = tenant.Suspend()

		subscription := stripe.Subscription{
			ID: "sub_test123",
			Customer: &stripe.Customer{
				ID: "cus_test123",
			},
			Status:             stripe.SubscriptionStatusActive,
			CurrentPeriodStart: time.Now().Unix(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
		}

		subscriptionJSON, _ := json.Marshal(subscription)
		event := stripe.Event{
			ID:   "evt_test123",
			Type: "customer.subscription.updated",
			Data: &stripe.EventData{
				Raw: subscriptionJSON,
			},
		}

		mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(tenant, nil)
		mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

		err := service.handleSubscriptionUpdated(ctx, event)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles past_due status", func(t *testing.T) {
		mockRepo := new(MockTenantRepository)
		service := createWebhookTestService(t, mockRepo)
		tenant := createWebhookTestTenant(t)

		subscription := stripe.Subscription{
			ID: "sub_test123",
			Customer: &stripe.Customer{
				ID: "cus_test123",
			},
			Status:             stripe.SubscriptionStatusPastDue,
			CurrentPeriodStart: time.Now().Unix(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
		}

		subscriptionJSON, _ := json.Marshal(subscription)
		event := stripe.Event{
			ID:   "evt_test123",
			Type: "customer.subscription.updated",
			Data: &stripe.EventData{
				Raw: subscriptionJSON,
			},
		}

		mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(tenant, nil)
		mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

		err := service.handleSubscriptionUpdated(ctx, event)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles canceled status", func(t *testing.T) {
		mockRepo := new(MockTenantRepository)
		service := createWebhookTestService(t, mockRepo)
		tenant := createWebhookTestTenant(t)

		subscription := stripe.Subscription{
			ID: "sub_test123",
			Customer: &stripe.Customer{
				ID: "cus_test123",
			},
			Status:             stripe.SubscriptionStatusCanceled,
			CurrentPeriodStart: time.Now().Unix(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
		}

		subscriptionJSON, _ := json.Marshal(subscription)
		event := stripe.Event{
			ID:   "evt_test123",
			Type: "customer.subscription.updated",
			Data: &stripe.EventData{
				Raw: subscriptionJSON,
			},
		}

		mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(tenant, nil)
		mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

		err := service.handleSubscriptionUpdated(ctx, event)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestStripeWebhookService_handleSubscriptionUpdated_TenantNotFound(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	subscription := stripe.Subscription{
		ID: "sub_unknown",
		Customer: &stripe.Customer{
			ID: "cus_unknown",
		},
		Status: stripe.SubscriptionStatusActive,
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	// First lookup by subscription ID fails
	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_unknown").Return(nil, shared.ErrNotFound)
	// Then lookup by customer ID also fails
	mockRepo.On("FindByStripeCustomerID", ctx, "cus_unknown").Return(nil, shared.ErrNotFound)

	err := service.handleSubscriptionUpdated(ctx, event)

	// Should not error, just skip
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionDeleted_TenantNotFound(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	subscription := stripe.Subscription{
		ID: "sub_unknown",
		Customer: &stripe.Customer{
			ID: "cus_unknown",
		},
		Status: stripe.SubscriptionStatusCanceled,
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_unknown").Return(nil, shared.ErrNotFound)

	err := service.handleSubscriptionDeleted(ctx, event)

	// Should not error, just skip
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaymentFailed_TenantNotFound(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_unknown",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test123",
		},
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.payment_failed",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	mockRepo.On("FindByStripeCustomerID", ctx, "cus_unknown").Return(nil, shared.ErrNotFound)

	err := service.handleInvoicePaymentFailed(ctx, event)

	// Should not error, just skip
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaid_TenantNotFound(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_unknown",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test123",
		},
		PeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.paid",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	mockRepo.On("FindByStripeCustomerID", ctx, "cus_unknown").Return(nil, shared.ErrNotFound)

	err := service.handleInvoicePaid(ctx, event)

	// Should not error, just skip
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionUpdated_WithPlanChange(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)
	_ = tenant.SetPlan(identity.TenantPlanBasic)

	subscription := stripe.Subscription{
		ID: "sub_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
		Metadata: map[string]string{
			"plan_id": "pro",
		},
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	err := service.handleSubscriptionUpdated(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionUpdated_FallbackToCustomerID(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	tenant := createWebhookTestTenant(t)

	subscription := stripe.Subscription{
		ID: "sub_new123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	// First lookup by subscription ID fails
	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_new123").Return(nil, shared.ErrNotFound)
	// Then lookup by customer ID succeeds
	mockRepo.On("FindByStripeCustomerID", ctx, "cus_test123").Return(tenant, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*identity.Tenant")).Return(nil)

	err := service.handleSubscriptionUpdated(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionUpdated_NoCustomerID(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	subscription := stripe.Subscription{
		ID:                 "sub_new123",
		Customer:           nil, // No customer
		Status:             stripe.SubscriptionStatusActive,
		CurrentPeriodStart: time.Now().Unix(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	// First lookup by subscription ID fails
	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_new123").Return(nil, shared.ErrNotFound)

	err := service.handleSubscriptionUpdated(ctx, event)

	// Should not error, just skip
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionUpdated_DatabaseError(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	subscription := stripe.Subscription{
		ID: "sub_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Status: stripe.SubscriptionStatusActive,
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.updated",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(nil, errors.New("database error"))

	err := service.handleSubscriptionUpdated(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find tenant")
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleSubscriptionDeleted_DatabaseError(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	subscription := stripe.Subscription{
		ID: "sub_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Status: stripe.SubscriptionStatusCanceled,
	}

	subscriptionJSON, _ := json.Marshal(subscription)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "customer.subscription.deleted",
		Data: &stripe.EventData{
			Raw: subscriptionJSON,
		},
	}

	mockRepo.On("FindByStripeSubscriptionID", ctx, "sub_test123").Return(nil, errors.New("database error"))

	err := service.handleSubscriptionDeleted(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find tenant")
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaid_DatabaseError(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test123",
		},
		PeriodEnd: time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.paid",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	mockRepo.On("FindByStripeCustomerID", ctx, "cus_test123").Return(nil, errors.New("database error"))

	err := service.handleInvoicePaid(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find tenant")
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaymentFailed_DatabaseError(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Subscription: &stripe.Subscription{
			ID: "sub_test123",
		},
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.payment_failed",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	mockRepo.On("FindByStripeCustomerID", ctx, "cus_test123").Return(nil, errors.New("database error"))

	err := service.handleInvoicePaymentFailed(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find tenant")
	mockRepo.AssertExpectations(t)
}

func TestStripeWebhookService_handleInvoicePaymentFailed_NonSubscriptionInvoice(t *testing.T) {
	mockRepo := new(MockTenantRepository)
	service := createWebhookTestService(t, mockRepo)
	ctx := context.Background()

	// Invoice without subscription (one-time payment)
	invoice := stripe.Invoice{
		ID: "in_test123",
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		Subscription: nil, // No subscription
	}

	invoiceJSON, _ := json.Marshal(invoice)
	event := stripe.Event{
		ID:   "evt_test123",
		Type: "invoice.payment_failed",
		Data: &stripe.EventData{
			Raw: invoiceJSON,
		},
	}

	// Should skip without calling repo
	err := service.handleInvoicePaymentFailed(ctx, event)

	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "FindByStripeCustomerID")
}

func TestStripeSubscriptionEvent(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		action       string
		expectedType string
	}{
		{"subscription_created", EventTypeStripeSubscriptionCreated},
		{"subscription_updated", EventTypeStripeSubscriptionUpdated},
		{"subscription_deleted", EventTypeStripeSubscriptionDeleted},
		{"unknown_action", "StripeSubscriptionunknown_action"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			event := NewStripeSubscriptionEvent(tenantID, tt.action, "sub_123")
			assert.Equal(t, tt.expectedType, event.EventType())
			assert.Equal(t, "sub_123", event.SubscriptionID)
			assert.Equal(t, tt.action, event.Action)
			assert.Equal(t, tenantID, event.AggregateID())
		})
	}
}

func TestStripePaymentEvent(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		action       string
		expectedType string
	}{
		{"invoice_paid", EventTypeStripeInvoicePaid},
		{"payment_failed", EventTypeStripePaymentFailed},
		{"unknown_action", "StripePaymentunknown_action"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			event := NewStripePaymentEvent(tenantID, tt.action, "in_123")
			assert.Equal(t, tt.expectedType, event.EventType())
			assert.Equal(t, "in_123", event.InvoiceID)
			assert.Equal(t, tt.action, event.Action)
			assert.Equal(t, tenantID, event.AggregateID())
		})
	}
}
