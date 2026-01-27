package pool

import (
	"testing"
	"time"
)

func TestNewParameterValue(t *testing.T) {
	tests := []struct {
		name         string
		value        any
		semanticType SemanticType
		ttl          time.Duration
		checkExpiry  bool
	}{
		{
			name:         "string value with TTL",
			value:        "test-value",
			semanticType: SemanticTypeCustomerID,
			ttl:          time.Hour,
			checkExpiry:  true,
		},
		{
			name:         "int value without TTL",
			value:        12345,
			semanticType: SemanticTypeProductID,
			ttl:          0,
			checkExpiry:  false,
		},
		{
			name:         "struct value",
			value:        struct{ ID string }{ID: "test"},
			semanticType: SemanticTypeSalesOrderID,
			ttl:          time.Minute,
			checkExpiry:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := NewParameterValue(tt.value, tt.semanticType, tt.ttl)

			if pv.Value != tt.value {
				t.Errorf("Value = %v, want %v", pv.Value, tt.value)
			}

			if pv.SemanticType != tt.semanticType {
				t.Errorf("SemanticType = %v, want %v", pv.SemanticType, tt.semanticType)
			}

			if pv.CreatedAt.IsZero() {
				t.Error("CreatedAt should not be zero")
			}

			if tt.checkExpiry {
				if pv.ExpiresAt.IsZero() {
					t.Error("ExpiresAt should not be zero when TTL is set")
				}
				if pv.ExpiresAt.Before(pv.CreatedAt) {
					t.Error("ExpiresAt should be after CreatedAt")
				}
			} else {
				if !pv.ExpiresAt.IsZero() {
					t.Error("ExpiresAt should be zero when TTL is not set")
				}
			}
		})
	}
}

func TestParameterValueWithSource(t *testing.T) {
	pv := NewParameterValue("test", SemanticTypeCustomerID, 0)
	pv.WithSource("POST /customers", "$.data.id")

	if pv.SourceEndpoint != "POST /customers" {
		t.Errorf("SourceEndpoint = %v, want POST /customers", pv.SourceEndpoint)
	}

	if pv.ResponsePath != "$.data.id" {
		t.Errorf("ResponsePath = %v, want $.data.id", pv.ResponsePath)
	}
}

func TestParameterValueIsExpired(t *testing.T) {
	// Value without expiry
	pv1 := NewParameterValue("test", SemanticTypeCustomerID, 0)
	if pv1.IsExpired() {
		t.Error("Value without TTL should not be expired")
	}

	// Value with future expiry
	pv2 := NewParameterValue("test", SemanticTypeCustomerID, time.Hour)
	if pv2.IsExpired() {
		t.Error("Value with future expiry should not be expired")
	}

	// Value with past expiry
	pv3 := NewParameterValue("test", SemanticTypeCustomerID, time.Nanosecond)
	time.Sleep(2 * time.Millisecond)
	if !pv3.IsExpired() {
		t.Error("Value with past expiry should be expired")
	}
}

func TestParameterValueTouch(t *testing.T) {
	pv := NewParameterValue("test", SemanticTypeCustomerID, 0)
	initialAccess := pv.LastAccessedAt()
	initialCount := pv.AccessCount()

	time.Sleep(time.Millisecond)
	pv.Touch()

	if pv.AccessCount() != initialCount+1 {
		t.Errorf("AccessCount = %v, want %v", pv.AccessCount(), initialCount+1)
	}

	if !pv.LastAccessedAt().After(initialAccess) {
		t.Error("LastAccessedAt should be updated after Touch")
	}
}

func TestParameterValueClone(t *testing.T) {
	pv := NewParameterValue("test", SemanticTypeCustomerID, time.Hour)
	pv.WithSource("POST /customers", "$.data.id")
	pv.Touch()

	clone := pv.Clone()

	if clone.Value != pv.Value {
		t.Errorf("Clone Value = %v, want %v", clone.Value, pv.Value)
	}

	if clone.SemanticType != pv.SemanticType {
		t.Errorf("Clone SemanticType = %v, want %v", clone.SemanticType, pv.SemanticType)
	}

	if clone.SourceEndpoint != pv.SourceEndpoint {
		t.Errorf("Clone SourceEndpoint = %v, want %v", clone.SourceEndpoint, pv.SourceEndpoint)
	}

	if clone.AccessCount() != pv.AccessCount() {
		t.Errorf("Clone AccessCount = %v, want %v", clone.AccessCount(), pv.AccessCount())
	}

	// Verify it's a separate instance
	if clone == pv {
		t.Error("Clone should be a different instance")
	}
}
