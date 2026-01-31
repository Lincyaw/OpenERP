package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
)

// ============================================================================
// ReportUsage Tests
// ============================================================================

func TestReportUsage_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscription_items/si_test123/usage_records" {
			return json.Marshal(&stripe.UsageRecord{
				ID:               "mbur_test123",
				SubscriptionItem: "si_test123",
				Quantity:         100,
				Timestamp:        now.Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UsageReportInput{
		TenantID:           uuid.New(),
		SubscriptionItemID: "si_test123",
		Quantity:           100,
		Timestamp:          now,
		Action:             "increment",
	}

	output, err := adapter.ReportUsage(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "mbur_test123", output.UsageRecordID)
	assert.Equal(t, "si_test123", output.SubscriptionItemID)
	assert.Equal(t, int64(100), output.Quantity)
	assert.Equal(t, "increment", output.Action)
}

func TestReportUsage_WithSetAction(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscription_items/si_test123/usage_records" {
			return json.Marshal(&stripe.UsageRecord{
				ID:               "mbur_set123",
				SubscriptionItem: "si_test123",
				Quantity:         500,
				Timestamp:        now.Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UsageReportInput{
		TenantID:           uuid.New(),
		SubscriptionItemID: "si_test123",
		Quantity:           500,
		Timestamp:          now,
		Action:             "set",
	}

	output, err := adapter.ReportUsage(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "mbur_set123", output.UsageRecordID)
	assert.Equal(t, int64(500), output.Quantity)
	assert.Equal(t, "set", output.Action)
}

func TestReportUsage_WithIdempotencyKey(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscription_items/si_test123/usage_records" {
			return json.Marshal(&stripe.UsageRecord{
				ID:               "mbur_idem123",
				SubscriptionItem: "si_test123",
				Quantity:         50,
				Timestamp:        now.Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	tenantID := uuid.New()
	idempotencyKey := GenerateIdempotencyKey(tenantID, "si_test123", billing.UsageTypeAPICalls, now)

	input := UsageReportInput{
		TenantID:           tenantID,
		SubscriptionItemID: "si_test123",
		Quantity:           50,
		Timestamp:          now,
		IdempotencyKey:     idempotencyKey,
	}

	output, err := adapter.ReportUsage(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "mbur_idem123", output.UsageRecordID)
}

func TestReportUsage_MissingSubscriptionItemID(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	input := UsageReportInput{
		TenantID:           uuid.New(),
		SubscriptionItemID: "", // Missing
		Quantity:           100,
	}

	output, err := adapter.ReportUsage(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "subscription item ID is required")
}

func TestReportUsage_NegativeQuantity(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	input := UsageReportInput{
		TenantID:           uuid.New(),
		SubscriptionItemID: "si_test123",
		Quantity:           -100, // Negative
	}

	output, err := adapter.ReportUsage(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "quantity cannot be negative")
}

func TestReportUsage_StripeError(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		return nil, &stripe.Error{
			Code: stripe.ErrorCodeResourceMissing,
			Msg:  "No such subscription item: si_nonexistent",
		}
	})
	defer cleanup()

	input := UsageReportInput{
		TenantID:           uuid.New(),
		SubscriptionItemID: "si_nonexistent",
		Quantity:           100,
	}

	output, err := adapter.ReportUsage(context.Background(), input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to report usage")
}

func TestReportUsage_DefaultAction(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" && path == "/v1/subscription_items/si_test123/usage_records" {
			return json.Marshal(&stripe.UsageRecord{
				ID:               "mbur_default123",
				SubscriptionItem: "si_test123",
				Quantity:         25,
				Timestamp:        now.Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UsageReportInput{
		TenantID:           uuid.New(),
		SubscriptionItemID: "si_test123",
		Quantity:           25,
		// Action not specified - should default to "increment"
	}

	output, err := adapter.ReportUsage(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "increment", output.Action)
}

// ============================================================================
// ReportUsageBatch Tests
// ============================================================================

func TestReportUsageBatch_AllSuccess(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	callCount := 0

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		if method == "POST" {
			callCount++
			return json.Marshal(&stripe.UsageRecord{
				ID:               fmt.Sprintf("mbur_batch%d", callCount),
				SubscriptionItem: fmt.Sprintf("si_test%d", callCount),
				Quantity:         int64(callCount * 10),
				Timestamp:        now.Unix(),
			})
		}
		return nil, fmt.Errorf("unexpected call: %s %s", method, path)
	})
	defer cleanup()

	input := UsageReportBatchInput{
		TenantID: uuid.New(),
		Records: []UsageReportRecord{
			{SubscriptionItemID: "si_test1", Quantity: 10},
			{SubscriptionItemID: "si_test2", Quantity: 20},
			{SubscriptionItemID: "si_test3", Quantity: 30},
		},
	}

	output, err := adapter.ReportUsageBatch(context.Background(), input)

	require.NoError(t, err)
	assert.Len(t, output.Successful, 3)
	assert.Empty(t, output.Failed)
}

func TestReportUsageBatch_PartialFailure(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()
	callCount := 0

	cleanup := setupMockBackend(func(method, path string, params stripe.ParamsContainer) ([]byte, error) {
		callCount++
		if callCount == 2 {
			// Second call fails
			return nil, &stripe.Error{
				Code: stripe.ErrorCodeResourceMissing,
				Msg:  "No such subscription item",
			}
		}
		return json.Marshal(&stripe.UsageRecord{
			ID:               fmt.Sprintf("mbur_batch%d", callCount),
			SubscriptionItem: fmt.Sprintf("si_test%d", callCount),
			Quantity:         int64(callCount * 10),
			Timestamp:        now.Unix(),
		})
	})
	defer cleanup()

	input := UsageReportBatchInput{
		TenantID: uuid.New(),
		Records: []UsageReportRecord{
			{SubscriptionItemID: "si_test1", Quantity: 10},
			{SubscriptionItemID: "si_test2", Quantity: 20}, // This will fail
			{SubscriptionItemID: "si_test3", Quantity: 30},
		},
	}

	output, err := adapter.ReportUsageBatch(context.Background(), input)

	require.NoError(t, err) // Batch doesn't return error, it collects failures
	assert.Len(t, output.Successful, 2)
	assert.Len(t, output.Failed, 1)
	assert.Equal(t, "si_test2", output.Failed[0].SubscriptionItemID)
}

func TestReportUsageBatch_RateLimited(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	_, cleanup := setupHTTPMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"type":    "rate_limit_error",
				"message": "Rate limit exceeded",
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer cleanup()

	input := UsageReportBatchInput{
		TenantID: uuid.New(),
		Records: []UsageReportRecord{
			{SubscriptionItemID: "si_test1", Quantity: 10},
		},
	}

	output, err := adapter.ReportUsageBatch(context.Background(), input)

	require.NoError(t, err)
	assert.Empty(t, output.Successful)
	assert.Len(t, output.Failed, 1)
	// Rate limited errors should have RetryAfter set
	// Note: The actual RetryAfter depends on how the error is parsed
}

// ============================================================================
// GetSubscriptionItemID Tests
// ============================================================================

func TestGetSubscriptionItemID_Success(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	_, cleanup := setupHTTPMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/subscriptions/sub_test123" {
			response := map[string]interface{}{
				"id":       "sub_test123",
				"object":   "subscription",
				"customer": "cus_test123",
				"status":   "active",
				"items": map[string]interface{}{
					"object": "list",
					"data": []map[string]interface{}{
						{
							"id":     "si_item123",
							"object": "subscription_item",
							"price": map[string]interface{}{
								"id": "price_test123",
							},
						},
					},
				},
				"current_period_start": now.Unix(),
				"current_period_end":   now.Add(30 * 24 * time.Hour).Unix(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer cleanup()

	itemID, err := adapter.GetSubscriptionItemID(context.Background(), "sub_test123")

	require.NoError(t, err)
	assert.Equal(t, "si_item123", itemID)
}

func TestGetSubscriptionItemID_NoItems(t *testing.T) {
	config := testConfig()
	adapter, err := NewStripeAdapter(config, testLogger())
	require.NoError(t, err)

	now := time.Now()

	_, cleanup := setupHTTPMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/subscriptions/sub_test123" {
			response := map[string]interface{}{
				"id":       "sub_test123",
				"object":   "subscription",
				"customer": "cus_test123",
				"status":   "active",
				"items": map[string]interface{}{
					"object": "list",
					"data":   []map[string]interface{}{}, // Empty items
				},
				"current_period_start": now.Unix(),
				"current_period_end":   now.Add(30 * 24 * time.Hour).Unix(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer cleanup()

	itemID, err := adapter.GetSubscriptionItemID(context.Background(), "sub_test123")

	assert.Error(t, err)
	assert.Empty(t, itemID)
	assert.Contains(t, err.Error(), "no price/item")
}

// ============================================================================
// IdempotencyKey Tests
// ============================================================================

func TestGenerateIdempotencyKey(t *testing.T) {
	tenantID := uuid.MustParse("12345678-1234-1234-1234-123456789012")
	subscriptionItemID := "si_test123"
	usageType := billing.UsageTypeAPICalls
	timestamp := time.Unix(1704067200, 0) // 2024-01-01 00:00:00 UTC

	key := GenerateIdempotencyKey(tenantID, subscriptionItemID, usageType, timestamp)

	expected := "12345678-1234-1234-1234-123456789012:si_test123:API_CALLS:1704067200"
	assert.Equal(t, expected, key)
}

func TestParseIdempotencyKey_Success(t *testing.T) {
	key := "12345678-1234-1234-1234-123456789012:si_test123:API_CALLS:1704067200"

	tenantID, subscriptionItemID, usageType, timestamp, err := ParseIdempotencyKey(key)

	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse("12345678-1234-1234-1234-123456789012"), tenantID)
	assert.Equal(t, "si_test123", subscriptionItemID)
	assert.Equal(t, billing.UsageTypeAPICalls, usageType)
	assert.Equal(t, time.Unix(1704067200, 0), timestamp)
}

func TestParseIdempotencyKey_InvalidFormat(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectedErr string
	}{
		{"too few parts", "tenant:item:type", "invalid idempotency key format"},
		{"too many parts", "tenant:item:type:time:extra", "invalid tenant ID"}, // Extra parts end up in timestamp
		{"empty", "", "invalid idempotency key format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, err := ParseIdempotencyKey(tt.key)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestParseIdempotencyKey_InvalidTenantID(t *testing.T) {
	key := "invalid-uuid:si_test123:API_CALLS:1704067200"

	_, _, _, _, err := ParseIdempotencyKey(key)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tenant ID")
}

func TestParseIdempotencyKey_InvalidUsageType(t *testing.T) {
	key := "12345678-1234-1234-1234-123456789012:si_test123:INVALID_TYPE:1704067200"

	_, _, _, _, err := ParseIdempotencyKey(key)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid usage type")
}

func TestParseIdempotencyKey_InvalidTimestamp(t *testing.T) {
	key := "12345678-1234-1234-1234-123456789012:si_test123:API_CALLS:not-a-number"

	_, _, _, _, err := ParseIdempotencyKey(key)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

// ============================================================================
// UsageReportLog Tests
// ============================================================================

func TestNewUsageReportLog(t *testing.T) {
	tenantID := uuid.New()
	subscriptionItemID := "si_test123"
	usageType := billing.UsageTypeAPICalls
	quantity := int64(100)
	timestamp := time.Now()

	log := NewUsageReportLog(tenantID, subscriptionItemID, usageType, quantity, timestamp)

	assert.NotEqual(t, uuid.Nil, log.ID)
	assert.Equal(t, tenantID, log.TenantID)
	assert.Equal(t, subscriptionItemID, log.SubscriptionItemID)
	assert.Equal(t, usageType, log.UsageType)
	assert.Equal(t, quantity, log.Quantity)
	assert.Equal(t, timestamp, log.Timestamp)
	assert.Equal(t, UsageReportStatusPending, log.Status)
	assert.Equal(t, 0, log.RetryCount)
	assert.Empty(t, log.StripeRecordID)
	assert.Empty(t, log.ErrorMessage)
}

func TestUsageReportStatus_String(t *testing.T) {
	tests := []struct {
		status   UsageReportStatus
		expected string
	}{
		{UsageReportStatusPending, "pending"},
		{UsageReportStatusSuccess, "success"},
		{UsageReportStatusFailed, "failed"},
		{UsageReportStatusRetrying, "retrying"},
		{UsageReportStatusAbandoned, "abandoned"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}
