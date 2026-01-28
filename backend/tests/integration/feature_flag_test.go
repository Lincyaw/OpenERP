package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFeatureFlagIntegration tests the complete feature flag functionality
func TestFeatureFlagIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a simple test server
	ts := NewTestServer(t)
	tenantID := uuid.New()
	ts.DB.CreateTestTenantWithUUID(tenantID)

	// Create a test user
	userID := uuid.New()

	t.Run("CRUD Operations", func(t *testing.T) {
		// Create a feature flag
		flagKey := fmt.Sprintf("test_flag_%d", time.Now().Unix())
		createPayload := map[string]interface{}{
			"key":         flagKey,
			"name":        "Test Flag",
			"description": "Test flag for integration",
			"type":        "boolean",
			"default_value": map[string]interface{}{
				"enabled": false,
			},
			"tags": []string{"test", "integration"},
		}

		w := ts.Request(http.MethodPost, "/api/v1/feature-flags", createPayload, tenantID, userID)
		assert.Equal(t, http.StatusCreated, w.Code)

		var createResp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)
		assert.True(t, createResp.Success)
		assert.Equal(t, flagKey, createResp.Data.(map[string]interface{})["key"])

		// List feature flags
		w = ts.Request(http.MethodGet, "/api/v1/feature-flags", nil, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		var listResp APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &listResp)
		require.NoError(t, err)
		assert.True(t, listResp.Success)
		assert.NotNil(t, listResp.Data.(map[string]interface{})["items"])

		// Get single flag
		w = ts.Request(http.MethodGet, fmt.Sprintf("/api/v1/feature-flags/%s", flagKey), nil, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		var getResp APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &getResp)
		require.NoError(t, err)
		assert.True(t, getResp.Success)
		assert.Equal(t, flagKey, getResp.Data.(map[string]interface{})["key"])

		// Update flag
		updatePayload := map[string]interface{}{
			"name":        "Updated Test Flag",
			"description": "Updated description",
		}
		w = ts.Request(http.MethodPut, fmt.Sprintf("/api/v1/feature-flags/%s", flagKey), updatePayload, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		var updateResp APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &updateResp)
		require.NoError(t, err)
		assert.True(t, updateResp.Success)
		assert.Equal(t, "Updated Test Flag", updateResp.Data.(map[string]interface{})["name"])

		// Delete flag
		w = ts.Request(http.MethodDelete, fmt.Sprintf("/api/v1/feature-flags/%s", flagKey), nil, tenantID, userID)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Flag Evaluation", func(t *testing.T) {
		// Create test flags
		enabledFlag := fmt.Sprintf("enabled_flag_%d", time.Now().Unix())
		disabledFlag := fmt.Sprintf("disabled_flag_%d", time.Now().Unix())

		// Create enabled flag
		createPayload := map[string]interface{}{
			"key":         enabledFlag,
			"name":        "Enabled Flag",
			"description": "Flag that is enabled",
			"type":        "boolean",
			"default_value": map[string]interface{}{
				"enabled": true,
			},
		}
		w := ts.Request(http.MethodPost, "/api/v1/feature-flags", createPayload, tenantID, userID)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Enable the flag
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/enable", enabledFlag), nil, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		// Create disabled flag
		createPayload["key"] = disabledFlag
		createPayload["name"] = "Disabled Flag"
		createPayload["default_value"] = map[string]interface{}{"enabled": false}
		w = ts.Request(http.MethodPost, "/api/v1/feature-flags", createPayload, tenantID, userID)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Evaluate enabled flag
		evalPayload := map[string]interface{}{
			"context": map[string]interface{}{},
		}
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/evaluate", enabledFlag), evalPayload, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		var evalResp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &evalResp)
		require.NoError(t, err)
		assert.True(t, evalResp.Success)
		assert.True(t, evalResp.Data.(map[string]interface{})["enabled"].(bool))

		// Evaluate disabled flag
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/evaluate", disabledFlag), evalPayload, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &evalResp)
		require.NoError(t, err)
		assert.True(t, evalResp.Success)
		assert.False(t, evalResp.Data.(map[string]interface{})["enabled"].(bool))

		// Batch evaluation
		batchPayload := map[string]interface{}{
			"flags":   []string{enabledFlag, disabledFlag},
			"context": map[string]interface{}{},
		}
		w = ts.Request(http.MethodPost, "/api/v1/feature-flags/evaluate-batch", batchPayload, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		var batchResp APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &batchResp)
		require.NoError(t, err)
		assert.True(t, batchResp.Success)
		flags := batchResp.Data.(map[string]interface{})["flags"].(map[string]interface{})
		assert.True(t, flags[enabledFlag].(map[string]interface{})["enabled"].(bool))
		assert.False(t, flags[disabledFlag].(map[string]interface{})["enabled"].(bool))
	})

	t.Run("Percentage Rollout", func(t *testing.T) {
		// Create percentage flag
		percentFlag := fmt.Sprintf("percent_flag_%d", time.Now().Unix())
		createPayload := map[string]interface{}{
			"key":         percentFlag,
			"name":        "Percentage Flag",
			"description": "Flag with percentage rollout",
			"type":        "percentage",
			"default_value": map[string]interface{}{
				"enabled":    false,
				"percentage": 50,
			},
		}
		w := ts.Request(http.MethodPost, "/api/v1/feature-flags", createPayload, tenantID, userID)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Enable the flag
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/enable", percentFlag), nil, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test percentage distribution
		trueCount := 0
		totalEvaluations := 100
		for i := 0; i < totalEvaluations; i++ {
			evalPayload := map[string]interface{}{
				"context": map[string]interface{}{
					"user_id": fmt.Sprintf("user_%d", i),
				},
			}
			w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/evaluate", percentFlag), evalPayload, tenantID, userID)
			assert.Equal(t, http.StatusOK, w.Code)

			var evalResp APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &evalResp)
			require.NoError(t, err)
			if evalResp.Data.(map[string]interface{})["enabled"].(bool) {
				trueCount++
			}
		}

		// Should be approximately 50%
		percentage := float64(trueCount) / float64(totalEvaluations) * 100
		assert.Greater(t, percentage, 30.0) // At least 30%
		assert.Less(t, percentage, 70.0)      // At most 70%
	})

	t.Run("User Overrides", func(t *testing.T) {
		// Create flag
		overrideFlag := fmt.Sprintf("override_flag_%d", time.Now().Unix())
		createPayload := map[string]interface{}{
			"key":         overrideFlag,
			"name":        "Override Flag",
			"description": "Flag for testing overrides",
			"type":        "boolean",
			"default_value": map[string]interface{}{
				"enabled": false,
			},
		}
		w := ts.Request(http.MethodPost, "/api/v1/feature-flags", createPayload, tenantID, userID)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Enable the flag
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/enable", overrideFlag), nil, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		// Create user override
		overridePayload := map[string]interface{}{
			"target_type": "user",
			"target_id":   userID.String(),
			"value": map[string]interface{}{
				"enabled": true,
			},
		}
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/overrides", overrideFlag), overridePayload, tenantID, userID)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Evaluate with user context - should get override value
		evalPayload := map[string]interface{}{
			"context": map[string]interface{}{
				"user_id": userID.String(),
			},
		}
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/evaluate", overrideFlag), evalPayload, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		var evalResp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &evalResp)
		require.NoError(t, err)
		assert.True(t, evalResp.Success)
		assert.True(t, evalResp.Data.(map[string]interface{})["enabled"].(bool)) // Override value
	})

	t.Run("Audit Logging", func(t *testing.T) {
		// Create flag
		auditFlag := fmt.Sprintf("audit_flag_%d", time.Now().Unix())
		createPayload := map[string]interface{}{
			"key":         auditFlag,
			"name":        "Audit Flag",
			"description": "Flag for audit testing",
			"type":        "boolean",
			"default_value": map[string]interface{}{
				"enabled": false,
			},
		}
		w := ts.Request(http.MethodPost, "/api/v1/feature-flags", createPayload, tenantID, userID)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Enable flag
		w = ts.Request(http.MethodPost, fmt.Sprintf("/api/v1/feature-flags/%s/enable", auditFlag), nil, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		// Get audit logs
		w = ts.Request(http.MethodGet, fmt.Sprintf("/api/v1/feature-flags/%s/audit", auditFlag), nil, tenantID, userID)
		assert.Equal(t, http.StatusOK, w.Code)

		var auditResp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &auditResp)
		require.NoError(t, err)
		assert.True(t, auditResp.Success)
		assert.NotNil(t, auditResp.Data.(map[string]interface{})["items"])
	})
}