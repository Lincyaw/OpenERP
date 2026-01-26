package acl

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CustomerID Tests
// =============================================================================

func TestNewCustomerID_Success(t *testing.T) {
	id := uuid.New()
	customerID, err := NewCustomerID(id)

	require.NoError(t, err)
	assert.Equal(t, id, customerID.UUID())
	assert.Equal(t, id.String(), customerID.String())
	assert.False(t, customerID.IsEmpty())
}

func TestNewCustomerID_NilUUID(t *testing.T) {
	customerID, err := NewCustomerID(uuid.Nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
	assert.True(t, customerID.IsEmpty())
}

func TestMustNewCustomerID_Success(t *testing.T) {
	id := uuid.New()
	customerID := MustNewCustomerID(id)

	assert.Equal(t, id, customerID.UUID())
}

func TestMustNewCustomerID_Panics(t *testing.T) {
	assert.Panics(t, func() {
		MustNewCustomerID(uuid.Nil)
	})
}

func TestParseCustomerID_Success(t *testing.T) {
	id := uuid.New()
	customerID, err := ParseCustomerID(id.String())

	require.NoError(t, err)
	assert.Equal(t, id, customerID.UUID())
}

func TestParseCustomerID_EmptyString(t *testing.T) {
	customerID, err := ParseCustomerID("")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
	assert.True(t, customerID.IsEmpty())
}

func TestParseCustomerID_InvalidUUID(t *testing.T) {
	customerID, err := ParseCustomerID("not-a-uuid")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid UUID")
	assert.True(t, customerID.IsEmpty())
}

func TestCustomerID_Equals(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	customerID1 := MustNewCustomerID(id1)
	customerID1Copy := MustNewCustomerID(id1)
	customerID2 := MustNewCustomerID(id2)

	assert.True(t, customerID1.Equals(customerID1Copy), "Same UUIDs should be equal")
	assert.False(t, customerID1.Equals(customerID2), "Different UUIDs should not be equal")
}

// =============================================================================
// CustomerReference Tests
// =============================================================================

func TestNewCustomerReference_Success(t *testing.T) {
	id := MustNewCustomerID(uuid.New())
	name := "Test Customer"
	code := "CUST-001"

	ref, err := NewCustomerReference(id, name, code)

	require.NoError(t, err)
	assert.Equal(t, id, ref.ID())
	assert.Equal(t, id.UUID(), ref.UUID())
	assert.Equal(t, name, ref.Name())
	assert.Equal(t, code, ref.Code())
	assert.False(t, ref.IsEmpty())
}

func TestNewCustomerReference_EmptyID(t *testing.T) {
	ref, err := NewCustomerReference(CustomerID{}, "Test", "CODE")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Customer ID cannot be empty")
	assert.True(t, ref.IsEmpty())
}

func TestNewCustomerReference_EmptyName(t *testing.T) {
	id := MustNewCustomerID(uuid.New())
	_, err := NewCustomerReference(id, "", "CODE")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Customer name cannot be empty")
}

func TestNewCustomerReference_EmptyCode_Allowed(t *testing.T) {
	id := MustNewCustomerID(uuid.New())
	ref, err := NewCustomerReference(id, "Test Customer", "")

	require.NoError(t, err)
	assert.Equal(t, "", ref.Code())
}

func TestNewCustomerReferenceFromUUID_Success(t *testing.T) {
	id := uuid.New()
	name := "Test Customer"
	code := "CUST-001"

	ref, err := NewCustomerReferenceFromUUID(id, name, code)

	require.NoError(t, err)
	assert.Equal(t, id, ref.UUID())
	assert.Equal(t, name, ref.Name())
	assert.Equal(t, code, ref.Code())
}

func TestNewCustomerReferenceFromUUID_NilUUID(t *testing.T) {
	ref, err := NewCustomerReferenceFromUUID(uuid.Nil, "Test", "CODE")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Customer ID cannot be empty")
	assert.True(t, ref.IsEmpty())
}

func TestMustNewCustomerReference_Success(t *testing.T) {
	id := uuid.New()
	ref := MustNewCustomerReference(id, "Test", "CODE")

	assert.Equal(t, id, ref.UUID())
}

func TestMustNewCustomerReference_Panics(t *testing.T) {
	assert.Panics(t, func() {
		MustNewCustomerReference(uuid.Nil, "Test", "CODE")
	})

	assert.Panics(t, func() {
		MustNewCustomerReference(uuid.New(), "", "CODE")
	})
}

func TestCustomerReference_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		custName string
		expected string
	}{
		{
			name:     "With code",
			code:     "CUST-001",
			custName: "Test Customer",
			expected: "CUST-001 - Test Customer",
		},
		{
			name:     "Without code",
			code:     "",
			custName: "Test Customer",
			expected: "Test Customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := MustNewCustomerReference(uuid.New(), tt.custName, tt.code)
			assert.Equal(t, tt.expected, ref.DisplayName())
		})
	}
}

func TestCustomerReference_Equals(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	ref1 := MustNewCustomerReference(id1, "Customer 1", "C001")
	ref1Copy := MustNewCustomerReference(id1, "Different Name", "C999") // Same ID
	ref2 := MustNewCustomerReference(id2, "Customer 2", "C002")

	// Equals is based on ID only
	assert.True(t, ref1.Equals(ref1Copy), "Same IDs should be equal regardless of other fields")
	assert.False(t, ref1.Equals(ref2), "Different IDs should not be equal")
}

func TestCustomerReference_WithUpdatedInfo(t *testing.T) {
	id := uuid.New()
	original := MustNewCustomerReference(id, "Original Name", "C001")

	updated, err := original.WithUpdatedInfo("New Name", "C002")

	require.NoError(t, err)
	assert.Equal(t, id, updated.UUID(), "ID should remain the same")
	assert.Equal(t, "New Name", updated.Name())
	assert.Equal(t, "C002", updated.Code())
}

func TestCustomerReference_WithUpdatedInfo_EmptyName(t *testing.T) {
	original := MustNewCustomerReference(uuid.New(), "Original", "C001")

	_, err := original.WithUpdatedInfo("", "C002")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Customer name cannot be empty")
}

func TestEmptyCustomerReference(t *testing.T) {
	ref := EmptyCustomerReference()

	assert.True(t, ref.IsEmpty())
	assert.Equal(t, uuid.Nil, ref.UUID())
	assert.Equal(t, "", ref.Name())
	assert.Equal(t, "", ref.Code())
}

// =============================================================================
// Event DTO Tests
// =============================================================================

func TestCustomerCreatedEventDTO(t *testing.T) {
	dto := CustomerCreatedEventDTO{
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Code:       "CUST-001",
		Name:       "Test Customer",
	}

	assert.NotEqual(t, uuid.Nil, dto.TenantID)
	assert.NotEqual(t, uuid.Nil, dto.CustomerID)
	assert.Equal(t, "CUST-001", dto.Code)
	assert.Equal(t, "Test Customer", dto.Name)
}

func TestCustomerUpdatedEventDTO(t *testing.T) {
	dto := CustomerUpdatedEventDTO{
		TenantID:    uuid.New(),
		CustomerID:  uuid.New(),
		Code:        "CUST-001",
		Name:        "Updated Customer",
		ContactName: "John Doe",
		Phone:       "123-456-7890",
		Email:       "john@example.com",
	}

	assert.Equal(t, "Updated Customer", dto.Name)
	assert.Equal(t, "John Doe", dto.ContactName)
	assert.Equal(t, "123-456-7890", dto.Phone)
	assert.Equal(t, "john@example.com", dto.Email)
}

func TestCustomerDeletedEventDTO(t *testing.T) {
	dto := CustomerDeletedEventDTO{
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Code:       "CUST-001",
		Name:       "Deleted Customer",
	}

	assert.Equal(t, "CUST-001", dto.Code)
	assert.Equal(t, "Deleted Customer", dto.Name)
}

func TestCustomerStatusChangedEventDTO(t *testing.T) {
	dto := CustomerStatusChangedEventDTO{
		TenantID:   uuid.New(),
		CustomerID: uuid.New(),
		Code:       "CUST-001",
		OldStatus:  "active",
		NewStatus:  "suspended",
	}

	assert.Equal(t, "active", dto.OldStatus)
	assert.Equal(t, "suspended", dto.NewStatus)
}

// =============================================================================
// Integration Tests (Value Object Immutability)
// =============================================================================

func TestCustomerID_Immutability(t *testing.T) {
	id := uuid.New()
	customerID := MustNewCustomerID(id)
	originalUUID := customerID.UUID()

	// The underlying UUID should not change
	assert.Equal(t, id, customerID.UUID())
	assert.Equal(t, originalUUID, customerID.UUID())
}

func TestCustomerReference_Immutability(t *testing.T) {
	id := uuid.New()
	original := MustNewCustomerReference(id, "Original Name", "C001")
	originalName := original.Name()
	originalCode := original.Code()

	// WithUpdatedInfo should return a NEW reference, not mutate the original
	updated, _ := original.WithUpdatedInfo("New Name", "C002")

	// Original should be unchanged
	assert.Equal(t, originalName, original.Name())
	assert.Equal(t, originalCode, original.Code())

	// Updated should have new values
	assert.Equal(t, "New Name", updated.Name())
	assert.Equal(t, "C002", updated.Code())

	// They should be equal by ID
	assert.True(t, original.Equals(updated))
}
