package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
	"go.uber.org/zap"
)

// mockBackend implements stripe.Backend for testing
type mockBackend struct {
	handler func(method, path string, params stripe.ParamsContainer) ([]byte, error)
}

func (m *mockBackend) Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	data, err := m.handler(method, path, params)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (m *mockBackend) CallStreaming(method, path, key string, params stripe.ParamsContainer, v stripe.StreamingLastResponseSetter) error {
	return nil
}

func (m *mockBackend) CallRaw(method, path, key string, body *form.Values, params *stripe.Params, v stripe.LastResponseSetter) error {
	return nil
}

func (m *mockBackend) CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *stripe.Params, v stripe.LastResponseSetter) error {
	return nil
}

func (m *mockBackend) SetMaxNetworkRetries(maxNetworkRetries int64) {}

// testConfig returns a valid test configuration
func testConfig() *StripeConfig {
	return &StripeConfig{
		SecretKey:       "sk_test_123456789",
		PublishableKey:  "pk_test_123456789",
		WebhookSecret:   "whsec_test_123456789",
		IsTestMode:      true,
		DefaultCurrency: "usd",
		PriceIDs: map[string]string{
			"free":       "",
			"basic":      "price_basic_test",
			"pro":        "price_pro_test",
			"enterprise": "price_enterprise_test",
		},
	}
}

// testLogger returns a no-op logger for testing
func testLogger() *zap.Logger {
	return zap.NewNop()
}

// setupMockBackend sets up a mock Stripe backend for testing
func setupMockBackend(handler func(method, path string, params stripe.ParamsContainer) ([]byte, error)) func() {
	mock := &mockBackend{handler: handler}
	stripe.SetBackend(stripe.APIBackend, mock)
	return func() {
		// Reset to default backend after test
		stripe.SetBackend(stripe.APIBackend, nil)
	}
}

// ============================================================================
// NewStripeAdapter Tests
// ============================================================================

func TestNewStripeAdapter_Success(t *testing.T) {
	config := testConfig()
	logger := testLogger()

	adapter, err := NewStripeAdapter(config, logger)

	require.NoError(t, err)
	assert.NotNil(t, adapter)
}

func TestNewStripeAdapter_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *StripeConfig
		expectedErr string
	}{
		{
			name: "missing secret key",
			config: &StripeConfig{
				IsTestMode:      true,
				DefaultCurrency: "usd",
			},
			expectedErr: "secret key is required",
		},
		{
			name: "test mode with live key",
			config: &StripeConfig{
				SecretKey:       "sk_live_123456789",
				IsTestMode:      true,
				DefaultCurrency: "usd",
			},
			expectedErr: "test mode enabled but secret key is not a test key",
		},
		{
			name: "live mode with test key",
			config: &StripeConfig{
				SecretKey:       "sk_test_123456789",
				IsTestMode:      false,
				DefaultCurrency: "usd",
			},
			expectedErr: "live mode enabled but secret key is not a live key",
		},
		{
			name: "missing currency",
			config: &StripeConfig{
				SecretKey:  "sk_test_123456789",
				IsTestMode: true,
			},
			expectedErr: "default currency is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewStripeAdapter(tt.config, testLogger())

			assert.Error(t, err)
			assert.Nil(t, adapter)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// ============================================================================
// CreateCustomer Tests
// ============================================================================

func TestCreateCustomer_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/customers" {
			return json.Marshal(&stripe.Customer{
				ID:      "cus_test123",
				Email:   "test@example.com",
				Name:    "Test Customer",
				Created: time.Now().Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateCustomerInput{
		TenantID:    uuid.New(),
		Email:       "test@example.com",
		Name:        "Test Customer",
		Description: "Test description",
	}

	output, err := adapter.CreateCustomer(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "cus_test123", output.CustomerID)
	assert.Equal(t, "test@example.com", output.Email)
	assert.Equal(t, "Test Customer", output.Name)
}

func TestCreateCustomer_WithOptionalFields(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/customers" {
			return json.Marshal(&stripe.Customer{
				ID:      "cus_test456",
				Email:   "test@example.com",
				Name:    "Test Customer",
				Phone:   "+1234567890",
				Created: time.Now().Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateCustomerInput{
		TenantID:        uuid.New(),
		Email:           "test@example.com",
		Name:            "Test Customer",
		Phone:           "+1234567890",
		Description:     "Test description",
		TaxExempt:       true,
		PreferredLocale: "en-US",
		Metadata:        map[string]string{"custom_key": "custom_value"},
	}

	output, err := adapter.CreateCustomer(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "cus_test456", output.CustomerID)
}

func TestCreateCustomer_StripeError(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeCardDeclined,
			Msg:  "Your card was declined",
		}
	})
	defer cleanup()

	input := CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "test@example.com",
		Name:     "Test Customer",
	}

	output, err := adapter.CreateCustomer(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to create customer")
}

// ============================================================================
// GetCustomer Tests
// ============================================================================

func TestGetCustomer_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/customers/cus_test123" {
			return json.Marshal(&stripe.Customer{
				ID:      "cus_test123",
				Email:   "test@example.com",
				Name:    "Test Customer",
				Created: time.Now().Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	output, err := adapter.GetCustomer(context.Background(), "cus_test123")

	require.NoError(t, err)
	assert.Equal(t, "cus_test123", output.CustomerID)
	assert.Equal(t, "test@example.com", output.Email)
}

func TestGetCustomer_NotFound(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such customer: cus_nonexistent",
		}
	})
	defer cleanup()

	output, err := adapter.GetCustomer(context.Background(), "cus_nonexistent")

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to get customer")
}

// ============================================================================
// UpdateCustomer Tests
// ============================================================================

func TestUpdateCustomer_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/customers/cus_test123" {
			return json.Marshal(&stripe.Customer{
				ID:      "cus_test123",
				Email:   "updated@example.com",
				Name:    "Updated Customer",
				Created: time.Now().Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateCustomerInput{
		TenantID:    uuid.New(),
		Email:       "updated@example.com",
		Name:        "Updated Customer",
		Description: "Updated description",
	}

	output, err := adapter.UpdateCustomer(context.Background(), "cus_test123", input)

	require.NoError(t, err)
	assert.Equal(t, "cus_test123", output.CustomerID)
	assert.Equal(t, "updated@example.com", output.Email)
	assert.Equal(t, "Updated Customer", output.Name)
}

func TestUpdateCustomer_WithMetadata(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/customers/cus_test123" {
			return json.Marshal(&stripe.Customer{
				ID:      "cus_test123",
				Email:   "test@example.com",
				Name:    "Test Customer",
				Created: time.Now().Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "test@example.com",
		Name:     "Test Customer",
		Metadata: map[string]string{"key1": "value1", "key2": "value2"},
	}

	output, err := adapter.UpdateCustomer(context.Background(), "cus_test123", input)

	require.NoError(t, err)
	assert.NotNil(t, output)
}

func TestUpdateCustomer_Error(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such customer",
		}
	})
	defer cleanup()

	input := CreateCustomerInput{
		TenantID: uuid.New(),
		Email:    "test@example.com",
		Name:     "Test Customer",
	}

	output, err := adapter.UpdateCustomer(context.Background(), "cus_nonexistent", input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to update customer")
}

// ============================================================================
// DeleteCustomer Tests
// ============================================================================

func TestDeleteCustomer_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "DELETE" && path == "/v1/customers/cus_test123" {
			return json.Marshal(&stripe.Customer{
				ID:      "cus_test123",
				Deleted: true,
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	err = adapter.DeleteCustomer(context.Background(), "cus_test123")

	assert.NoError(t, err)
}

func TestDeleteCustomer_NotFound(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such customer",
		}
	})
	defer cleanup()

	err = adapter.DeleteCustomer(context.Background(), "cus_nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete customer")
}

// ============================================================================
// CreateSubscription Tests
// ============================================================================

func TestCreateSubscription_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	periodEnd := now.Add(30 * 24 * time.Hour)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscriptions" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_test123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   periodEnd.Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_basic_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: "cus_test123",
		PlanID:     "basic",
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_test123", output.SubscriptionID)
	assert.Equal(t, "cus_test123", output.CustomerID)
	assert.Equal(t, SubscriptionStatusActive, output.Status)
}

func TestCreateSubscription_WithTrialPeriod(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	trialEnd := now.Add(14 * 24 * time.Hour)
	periodEnd := now.Add(44 * 24 * time.Hour)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscriptions" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_trial123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusTrialing,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   periodEnd.Unix(),
				TrialStart:         now.Unix(),
				TrialEnd:           trialEnd.Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_pro_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: "cus_test123",
		PlanID:     "pro",
		TrialDays:  14,
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_trial123", output.SubscriptionID)
	assert.Equal(t, SubscriptionStatusTrialing, output.Status)
	assert.NotNil(t, output.TrialStart)
	assert.NotNil(t, output.TrialEnd)
}

func TestCreateSubscription_FreePlan(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	// Free plan should not create a Stripe subscription
	input := CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: "cus_test123",
		PlanID:     "free",
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "", output.SubscriptionID)
	assert.Equal(t, "cus_test123", output.CustomerID)
	assert.Equal(t, SubscriptionStatusActive, output.Status)
}

func TestCreateSubscription_WithPaymentMethod(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscriptions" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_pm123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_basic_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateSubscriptionInput{
		TenantID:         uuid.New(),
		CustomerID:       "cus_test123",
		PlanID:           "basic",
		PaymentMethod:    "pm_card_visa",
		CollectionMethod: "charge_automatically",
		Metadata:         map[string]string{"source": "web"},
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_pm123", output.SubscriptionID)
}

func TestCreateSubscription_WithDirectPriceID(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscriptions" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_direct123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_custom_123"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: "cus_test123",
		PriceID:    "price_custom_123", // Direct price ID instead of plan
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_direct123", output.SubscriptionID)
}

func TestCreateSubscription_WithIncompleteStatus(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscriptions" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_incomplete123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusIncomplete,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_basic_test"},
						},
					},
				},
				LatestInvoice: &stripe.Invoice{
					ID: "in_test123",
					PaymentIntent: &stripe.PaymentIntent{
						ClientSecret: "pi_secret_123",
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: "cus_test123",
		PlanID:     "basic",
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusIncomplete, output.Status)
	assert.Equal(t, "in_test123", output.LatestInvoiceID)
	assert.Equal(t, "pi_secret_123", output.ClientSecret)
}

func TestCreateSubscription_InvalidPlan(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	input := CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: "cus_test123",
		PlanID:     "nonexistent_plan",
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "no price ID configured for plan")
}

func TestCreateSubscription_StripeError(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeCardDeclined,
			Msg:  "Your card was declined",
		}
	})
	defer cleanup()

	input := CreateSubscriptionInput{
		TenantID:   uuid.New(),
		CustomerID: "cus_test123",
		PlanID:     "basic",
	}

	output, err := adapter.CreateSubscription(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to create subscription")
}

// ============================================================================
// UpdateSubscription Tests
// ============================================================================

func TestUpdateSubscription_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		// First call: GET subscription
		if method == "GET" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:       "sub_test123",
				Customer: &stripe.Customer{ID: "cus_test123"},
				Status:   stripe.SubscriptionStatusActive,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_basic_test"},
						},
					},
				},
			})
		}
		// Second call: POST update subscription
		if method == "POST" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_test123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_pro_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UpdateSubscriptionInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_test123",
		NewPlanID:      "pro",
	}

	output, err := adapter.UpdateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_test123", output.SubscriptionID)
	assert.Equal(t, SubscriptionStatusActive, output.Status)
	assert.Equal(t, "price_basic_test", output.PreviousPriceID)
	assert.Equal(t, "price_pro_test", output.NewPriceID)
}

func TestUpdateSubscription_WithProration(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:       "sub_test123",
				Customer: &stripe.Customer{ID: "cus_test123"},
				Status:   stripe.SubscriptionStatusActive,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_basic_test"},
						},
					},
				},
			})
		}
		if method == "POST" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_test123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_enterprise_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UpdateSubscriptionInput{
		TenantID:          uuid.New(),
		SubscriptionID:    "sub_test123",
		NewPlanID:         "enterprise",
		ProrationBehavior: "always_invoice",
		Metadata:          map[string]string{"upgrade_reason": "more_users"},
	}

	output, err := adapter.UpdateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_test123", output.SubscriptionID)
}

func TestUpdateSubscription_CancelAtPeriodEnd(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:       "sub_test123",
				Customer: &stripe.Customer{ID: "cus_test123"},
				Status:   stripe.SubscriptionStatusActive,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_pro_test"},
						},
					},
				},
			})
		}
		if method == "POST" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_test123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
				CancelAtPeriodEnd:  true,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_pro_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UpdateSubscriptionInput{
		TenantID:          uuid.New(),
		SubscriptionID:    "sub_test123",
		NewPriceID:        "price_pro_test",
		CancelAtPeriodEnd: true,
	}

	output, err := adapter.UpdateSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.True(t, output.CancelAtPeriodEnd)
}

func TestUpdateSubscription_NoItems(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:       "sub_test123",
				Customer: &stripe.Customer{ID: "cus_test123"},
				Status:   stripe.SubscriptionStatusActive,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{}, // Empty items
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UpdateSubscriptionInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_test123",
		NewPlanID:      "pro",
	}

	output, err := adapter.UpdateSubscription(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "subscription has no items")
}

func TestUpdateSubscription_GetError(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such subscription",
		}
	})
	defer cleanup()

	input := UpdateSubscriptionInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_nonexistent",
		NewPlanID:      "pro",
	}

	output, err := adapter.UpdateSubscription(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to get subscription")
}

func TestUpdateSubscription_UpdateError(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	callCount := 0
	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		callCount++
		if callCount == 1 && method == "GET" {
			return json.Marshal(&stripe.Subscription{
				ID:       "sub_test123",
				Customer: &stripe.Customer{ID: "cus_test123"},
				Status:   stripe.SubscriptionStatusActive,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_basic_test"},
						},
					},
				},
			})
		}
		// Second call fails
		return nil, &stripe.Error{
			Code: stripe.ErrorCode("invalid_request_error"),
			Msg:  "Invalid price",
		}
	})
	defer cleanup()

	input := UpdateSubscriptionInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_test123",
		NewPlanID:      "pro",
	}

	output, err := adapter.UpdateSubscription(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to update subscription")
}

// ============================================================================
// CancelSubscription Tests
// ============================================================================

func TestCancelSubscription_Immediately(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "DELETE" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:               "sub_test123",
				Customer:         &stripe.Customer{ID: "cus_test123"},
				Status:           stripe.SubscriptionStatusCanceled,
				CurrentPeriodEnd: now.Add(30 * 24 * time.Hour).Unix(),
				CanceledAt:       now.Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CancelSubscriptionInput{
		TenantID:          uuid.New(),
		SubscriptionID:    "sub_test123",
		CancelAtPeriodEnd: false,
		Reason:            "Customer requested cancellation",
	}

	output, err := adapter.CancelSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_test123", output.SubscriptionID)
	assert.Equal(t, SubscriptionStatusCanceled, output.Status)
	assert.NotNil(t, output.CanceledAt)
	assert.False(t, output.CancelAtPeriodEnd)
}

func TestCancelSubscription_AtPeriodEnd(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	periodEnd := now.Add(30 * 24 * time.Hour)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:                "sub_test123",
				Customer:          &stripe.Customer{ID: "cus_test123"},
				Status:            stripe.SubscriptionStatusActive,
				CurrentPeriodEnd:  periodEnd.Unix(),
				CancelAtPeriodEnd: true,
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := CancelSubscriptionInput{
		TenantID:          uuid.New(),
		SubscriptionID:    "sub_test123",
		CancelAtPeriodEnd: true,
		Reason:            "Downgrading to free plan",
	}

	output, err := adapter.CancelSubscription(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_test123", output.SubscriptionID)
	assert.Equal(t, SubscriptionStatusActive, output.Status)
	assert.True(t, output.CancelAtPeriodEnd)
	assert.Nil(t, output.CanceledAt) // Not canceled yet
}

func TestCancelSubscription_Error(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such subscription",
		}
	})
	defer cleanup()

	input := CancelSubscriptionInput{
		TenantID:          uuid.New(),
		SubscriptionID:    "sub_nonexistent",
		CancelAtPeriodEnd: false,
	}

	output, err := adapter.CancelSubscription(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to cancel subscription")
}

// ============================================================================
// GetSubscriptionStatus Tests
// ============================================================================

func TestGetSubscriptionStatus_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	periodEnd := now.Add(30 * 24 * time.Hour)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_test123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   periodEnd.Unix(),
				StartDate:          now.Add(-60 * 24 * time.Hour).Unix(),
				DaysUntilDue:       0,
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID: "si_test123",
							Price: &stripe.Price{
								ID:      "price_pro_test",
								Product: &stripe.Product{ID: "prod_test123"},
							},
						},
					},
				},
				LatestInvoice: &stripe.Invoice{ID: "in_test123"},
				DefaultPaymentMethod: &stripe.PaymentMethod{
					ID: "pm_test123",
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := GetSubscriptionStatusInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_test123",
	}

	output, err := adapter.GetSubscriptionStatus(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "sub_test123", output.SubscriptionID)
	assert.Equal(t, "cus_test123", output.CustomerID)
	assert.Equal(t, SubscriptionStatusActive, output.Status)
	assert.Equal(t, "price_pro_test", output.PriceID)
	assert.Equal(t, "prod_test123", output.ProductID)
	assert.Equal(t, "in_test123", output.LatestInvoiceID)
	assert.Equal(t, "pm_test123", output.DefaultPaymentMethod)
}

func TestGetSubscriptionStatus_WithTrialPeriod(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	trialEnd := now.Add(14 * 24 * time.Hour)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/subscriptions/sub_trial123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_trial123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusTrialing,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   trialEnd.Unix(),
				TrialStart:         now.Unix(),
				TrialEnd:           trialEnd.Unix(),
				StartDate:          now.Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_pro_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := GetSubscriptionStatusInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_trial123",
	}

	output, err := adapter.GetSubscriptionStatus(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusTrialing, output.Status)
	assert.NotNil(t, output.TrialStart)
	assert.NotNil(t, output.TrialEnd)
}

func TestGetSubscriptionStatus_Canceled(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	canceledAt := now.Add(-24 * time.Hour)
	endedAt := now

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/subscriptions/sub_canceled123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_canceled123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusCanceled,
				CurrentPeriodStart: now.Add(-30 * 24 * time.Hour).Unix(),
				CurrentPeriodEnd:   now.Unix(),
				CanceledAt:         canceledAt.Unix(),
				EndedAt:            endedAt.Unix(),
				StartDate:          now.Add(-60 * 24 * time.Hour).Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_basic_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := GetSubscriptionStatusInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_canceled123",
	}

	output, err := adapter.GetSubscriptionStatus(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusCanceled, output.Status)
	assert.NotNil(t, output.CanceledAt)
	assert.NotNil(t, output.EndedAt)
}

func TestGetSubscriptionStatus_ScheduledCancellation(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	cancelAt := now.Add(30 * 24 * time.Hour)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "GET" && path == "/v1/subscriptions/sub_scheduled123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_scheduled123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   cancelAt.Unix(),
				CancelAt:           cancelAt.Unix(),
				CancelAtPeriodEnd:  true,
				StartDate:          now.Add(-30 * 24 * time.Hour).Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							ID:    "si_test123",
							Price: &stripe.Price{ID: "price_pro_test"},
						},
					},
				},
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := GetSubscriptionStatusInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_scheduled123",
	}

	output, err := adapter.GetSubscriptionStatus(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusActive, output.Status)
	assert.True(t, output.CancelAtPeriodEnd)
	assert.NotNil(t, output.CancelAt)
}

func TestGetSubscriptionStatus_Error(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such subscription",
		}
	})
	defer cleanup()

	input := GetSubscriptionStatusInput{
		TenantID:       uuid.New(),
		SubscriptionID: "sub_nonexistent",
	}

	output, err := adapter.GetSubscriptionStatus(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to get subscription")
}

// ============================================================================
// ListSubscriptions Tests
// ============================================================================

// setupHTTPMockServer creates a mock HTTP server for Stripe API
func setupHTTPMockServer(handler http.HandlerFunc) (*httptest.Server, func()) {
	server := httptest.NewServer(handler)

	// Create a custom backend config pointing to our test server
	backendConfig := &stripe.BackendConfig{
		URL: stripe.String(server.URL),
	}
	backend := stripe.GetBackendWithConfig(stripe.APIBackend, backendConfig)
	stripe.SetBackend(stripe.APIBackend, backend)

	return server, func() {
		server.Close()
		stripe.SetBackend(stripe.APIBackend, nil)
	}
}

func TestListSubscriptions_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	_, cleanup := setupHTTPMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/subscriptions" {
			response := map[string]interface{}{
				"object":   "list",
				"has_more": false,
				"data": []map[string]interface{}{
					{
						"id":                   "sub_test1",
						"object":               "subscription",
						"customer":             "cus_test123",
						"status":               "active",
						"current_period_start": now.Unix(),
						"current_period_end":   now.Add(30 * 24 * time.Hour).Unix(),
						"start_date":           now.Add(-30 * 24 * time.Hour).Unix(),
						"items": map[string]interface{}{
							"object": "list",
							"data": []map[string]interface{}{
								{
									"id": "si_test1",
									"price": map[string]interface{}{
										"id": "price_basic_test",
									},
								},
							},
						},
					},
					{
						"id":                   "sub_test2",
						"object":               "subscription",
						"customer":             "cus_test123",
						"status":               "trialing",
						"current_period_start": now.Unix(),
						"current_period_end":   now.Add(14 * 24 * time.Hour).Unix(),
						"start_date":           now.Unix(),
						"items": map[string]interface{}{
							"object": "list",
							"data": []map[string]interface{}{
								{
									"id": "si_test2",
									"price": map[string]interface{}{
										"id": "price_pro_test",
									},
								},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer cleanup()

	subscriptions, err := adapter.ListSubscriptions(context.Background(), "cus_test123")

	require.NoError(t, err)
	assert.Len(t, subscriptions, 2)
	assert.Equal(t, "sub_test1", subscriptions[0].SubscriptionID)
	assert.Equal(t, SubscriptionStatusActive, subscriptions[0].Status)
	assert.Equal(t, "sub_test2", subscriptions[1].SubscriptionID)
	assert.Equal(t, SubscriptionStatusTrialing, subscriptions[1].Status)
}

func TestListSubscriptions_Empty(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	_, cleanup := setupHTTPMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/subscriptions" {
			response := map[string]interface{}{
				"object":   "list",
				"has_more": false,
				"data":     []map[string]interface{}{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer cleanup()

	subscriptions, err := adapter.ListSubscriptions(context.Background(), "cus_no_subs")

	require.NoError(t, err)
	assert.Empty(t, subscriptions)
}

func TestListSubscriptions_Error(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	_, cleanup := setupHTTPMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"type":    "invalid_request_error",
				"code":    "resource_missing",
				"message": "No such customer: cus_nonexistent",
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer cleanup()

	subscriptions, err := adapter.ListSubscriptions(context.Background(), "cus_nonexistent")

	assert.Error(t, err)
	assert.Nil(t, subscriptions)
	assert.Contains(t, err.Error(), "failed to list subscriptions")
}

// ============================================================================
// ResumeSubscription Tests
// ============================================================================

func TestResumeSubscription_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscriptions/sub_test123" {
			return json.Marshal(&stripe.Subscription{
				ID:                 "sub_test123",
				Customer:           &stripe.Customer{ID: "cus_test123"},
				Status:             stripe.SubscriptionStatusActive,
				CurrentPeriodStart: now.Unix(),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour).Unix(),
				CancelAtPeriodEnd:  false,
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	output, err := adapter.ResumeSubscription(context.Background(), uuid.New(), "sub_test123")

	require.NoError(t, err)
	assert.Equal(t, "sub_test123", output.SubscriptionID)
	assert.Equal(t, "cus_test123", output.CustomerID)
	assert.Equal(t, SubscriptionStatusActive, output.Status)
	assert.False(t, output.CancelAtPeriodEnd)
}

func TestResumeSubscription_Error(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such subscription",
		}
	})
	defer cleanup()

	output, err := adapter.ResumeSubscription(context.Background(), uuid.New(), "sub_nonexistent")

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to resume subscription")
}

func TestResumeSubscription_AlreadyCanceled(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCode("invalid_request_error"),
			Msg:  "Cannot resume a canceled subscription",
		}
	})
	defer cleanup()

	output, err := adapter.ResumeSubscription(context.Background(), uuid.New(), "sub_canceled123")

	assert.Error(t, err)
	assert.Nil(t, output)
}

// ============================================================================
// mapStripeSubscriptionStatus Tests
// ============================================================================

func TestMapStripeSubscriptionStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    stripe.SubscriptionStatus
		expected SubscriptionStatus
	}{
		{
			name:     "active",
			input:    stripe.SubscriptionStatusActive,
			expected: SubscriptionStatusActive,
		},
		{
			name:     "past_due",
			input:    stripe.SubscriptionStatusPastDue,
			expected: SubscriptionStatusPastDue,
		},
		{
			name:     "canceled",
			input:    stripe.SubscriptionStatusCanceled,
			expected: SubscriptionStatusCanceled,
		},
		{
			name:     "incomplete",
			input:    stripe.SubscriptionStatusIncomplete,
			expected: SubscriptionStatusIncomplete,
		},
		{
			name:     "incomplete_expired",
			input:    stripe.SubscriptionStatusIncompleteExpired,
			expected: SubscriptionStatusIncompleteExpired,
		},
		{
			name:     "trialing",
			input:    stripe.SubscriptionStatusTrialing,
			expected: SubscriptionStatusTrialing,
		},
		{
			name:     "unpaid",
			input:    stripe.SubscriptionStatusUnpaid,
			expected: SubscriptionStatusUnpaid,
		},
		{
			name:     "paused",
			input:    stripe.SubscriptionStatusPaused,
			expected: SubscriptionStatusPaused,
		},
		{
			name:     "unknown status",
			input:    stripe.SubscriptionStatus("unknown"),
			expected: SubscriptionStatus("unknown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapStripeSubscriptionStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// SubscriptionStatus Tests
// ============================================================================

func TestSubscriptionStatus_String(t *testing.T) {
	tests := []struct {
		status   SubscriptionStatus
		expected string
	}{
		{SubscriptionStatusActive, "active"},
		{SubscriptionStatusPastDue, "past_due"},
		{SubscriptionStatusCanceled, "canceled"},
		{SubscriptionStatusTrialing, "trialing"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestSubscriptionStatus_IsActive(t *testing.T) {
	tests := []struct {
		status   SubscriptionStatus
		expected bool
	}{
		{SubscriptionStatusActive, true},
		{SubscriptionStatusTrialing, true},
		{SubscriptionStatusPastDue, false},
		{SubscriptionStatusCanceled, false},
		{SubscriptionStatusIncomplete, false},
		{SubscriptionStatusUnpaid, false},
		{SubscriptionStatusPaused, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsActive())
		})
	}
}

// ============================================================================
// StripeConfig Tests
// ============================================================================

func TestStripeConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *StripeConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid test config",
			config:      testConfig(),
			expectError: false,
		},
		{
			name: "valid live config",
			config: &StripeConfig{
				SecretKey:       "sk_live_123456789",
				IsTestMode:      false,
				DefaultCurrency: "usd",
			},
			expectError: false,
		},
		{
			name: "missing secret key",
			config: &StripeConfig{
				IsTestMode:      true,
				DefaultCurrency: "usd",
			},
			expectError: true,
			errorMsg:    "secret key is required",
		},
		{
			name: "missing currency",
			config: &StripeConfig{
				SecretKey:  "sk_test_123456789",
				IsTestMode: true,
			},
			expectError: true,
			errorMsg:    "default currency is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStripeConfig_GetPriceID(t *testing.T) {
	config := testConfig()

	tests := []struct {
		name        string
		plan        string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "free plan",
			plan:        "free",
			expected:    "",
			expectError: false,
		},
		{
			name:        "basic plan",
			plan:        "basic",
			expected:    "price_basic_test",
			expectError: false,
		},
		{
			name:        "pro plan",
			plan:        "pro",
			expected:    "price_pro_test",
			expectError: false,
		},
		{
			name:        "enterprise plan",
			plan:        "enterprise",
			expected:    "price_enterprise_test",
			expectError: false,
		},
		{
			name:        "nonexistent plan",
			plan:        "nonexistent",
			expectError: true,
			errorMsg:    "no price ID configured for plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priceID, err := config.GetPriceID(tt.plan)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, priceID)
			}
		})
	}
}

func TestStripeConfig_GetPriceID_EmptyPriceForNonFreePlan(t *testing.T) {
	config := &StripeConfig{
		SecretKey:       "sk_test_123456789",
		IsTestMode:      true,
		DefaultCurrency: "usd",
		PriceIDs: map[string]string{
			"free":  "",
			"basic": "", // Empty price for non-free plan
		},
	}

	priceID, err := config.GetPriceID("basic")

	assert.Error(t, err)
	assert.Empty(t, priceID)
	assert.Contains(t, err.Error(), "price ID not set for plan")
}

func TestDefaultStripeConfig(t *testing.T) {
	config := DefaultStripeConfig()

	assert.True(t, config.IsTestMode)
	assert.Equal(t, "cny", config.DefaultCurrency)
	assert.Contains(t, config.PriceIDs, "free")
	assert.Contains(t, config.PriceIDs, "basic")
	assert.Contains(t, config.PriceIDs, "pro")
	assert.Contains(t, config.PriceIDs, "enterprise")
}
