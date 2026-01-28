package pool

import (
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/stretchr/testify/assert"
)

func TestNewParameterValue(t *testing.T) {
	source := ValueSource{
		Endpoint:      "/customers",
		ResponseField: "$.data.id",
		RequestID:     "req-123",
	}

	pv := NewParameterValue("cust-123", circuit.EntityCustomerID, source)

	assert.Equal(t, "cust-123", pv.Data)
	assert.Equal(t, circuit.EntityCustomerID, pv.SemanticType)
	assert.Equal(t, source, pv.Source)
	assert.False(t, pv.CreatedAt.IsZero())
	assert.True(t, pv.ExpiresAt.IsZero()) // No TTL
}

func TestNewParameterValueWithTTL(t *testing.T) {
	source := ValueSource{Endpoint: "/customers"}

	pv := NewParameterValueWithTTL("cust-123", circuit.EntityCustomerID, source, time.Hour)

	assert.Equal(t, "cust-123", pv.Data)
	assert.False(t, pv.ExpiresAt.IsZero())
	assert.True(t, pv.ExpiresAt.After(pv.CreatedAt))
}

func TestValue_IsExpired(t *testing.T) {
	t.Run("no expiration", func(t *testing.T) {
		v := &Value{Data: "test"}
		assert.False(t, v.IsExpired())
	})

	t.Run("not expired", func(t *testing.T) {
		v := &Value{
			Data:      "test",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		assert.False(t, v.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		v := &Value{
			Data:      "test",
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		assert.True(t, v.IsExpired())
	})
}

func TestValue_IsExpiredAt(t *testing.T) {
	now := time.Now()
	v := &Value{
		Data:      "test",
		ExpiresAt: now,
	}

	assert.False(t, v.IsExpiredAt(now.Add(-time.Hour)))
	assert.True(t, v.IsExpiredAt(now.Add(time.Hour)))
}

func TestValue_TTL(t *testing.T) {
	t.Run("no expiration", func(t *testing.T) {
		v := &Value{Data: "test"}
		assert.Equal(t, time.Duration(0), v.TTL())
	})

	t.Run("has TTL remaining", func(t *testing.T) {
		v := &Value{
			Data:      "test",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		ttl := v.TTL()
		assert.True(t, ttl > 0)
		assert.True(t, ttl <= time.Hour)
	})

	t.Run("expired", func(t *testing.T) {
		v := &Value{
			Data:      "test",
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		assert.Equal(t, time.Duration(0), v.TTL())
	})
}

func TestValue_Clone(t *testing.T) {
	original := &Value{
		Data:         "test-data",
		SemanticType: circuit.EntityCustomerID,
		Source: ValueSource{
			Endpoint:      "/customers",
			ResponseField: "$.id",
			RequestID:     "req-1",
		},
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		UsageCount: 5,
	}

	clone := original.Clone()

	// Verify all fields are copied
	assert.Equal(t, original.Data, clone.Data)
	assert.Equal(t, original.SemanticType, clone.SemanticType)
	assert.Equal(t, original.Source, clone.Source)
	assert.Equal(t, original.CreatedAt, clone.CreatedAt)
	assert.Equal(t, original.ExpiresAt, clone.ExpiresAt)
	assert.Equal(t, original.UsageCount, clone.UsageCount)

	// Verify it's a different object
	assert.NotSame(t, original, clone)

	// Modify clone and verify original is unchanged
	clone.Data = "modified"
	clone.Source.Endpoint = "/modified"
	assert.Equal(t, "test-data", original.Data)
	assert.Equal(t, "/customers", original.Source.Endpoint)
}
