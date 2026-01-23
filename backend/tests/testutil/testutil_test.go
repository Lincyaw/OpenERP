package testutil

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockDB(t *testing.T) {
	mockDB := NewMockDB(t)
	defer mockDB.Close()

	assert.NotNil(t, mockDB.DB)
	assert.NotNil(t, mockDB.Mock)
	assert.NotNil(t, mockDB.SqlDB)
}

func TestMockDB_ExpectationsWereMet(t *testing.T) {
	mockDB := NewMockDB(t)
	defer mockDB.Close()

	// No expectations set, should pass
	mockDB.ExpectationsWereMet(t)
}

func TestNewTestContext(t *testing.T) {
	tc := NewTestContext(t)

	assert.NotNil(t, tc.Context)
	assert.NotNil(t, tc.Recorder)
	assert.NotNil(t, tc.Engine)
	assert.Equal(t, http.MethodGet, tc.Context.Request.Method)
}

func TestTestContext_SetRequestID(t *testing.T) {
	tc := NewTestContext(t)

	tc.SetRequestID("req-123")

	val, exists := tc.Context.Get("X-Request-ID")
	assert.True(t, exists)
	assert.Equal(t, "req-123", val)
}

func TestTestContext_SetTenantID(t *testing.T) {
	tc := NewTestContext(t)

	tc.SetTenantID("tenant-456")

	val, exists := tc.Context.Get("X-Tenant-ID")
	assert.True(t, exists)
	assert.Equal(t, "tenant-456", val)
}

func TestTestContext_SetUserID(t *testing.T) {
	tc := NewTestContext(t)

	tc.SetUserID("user-789")

	val, exists := tc.Context.Get("X-User-ID")
	assert.True(t, exists)
	assert.Equal(t, "user-789", val)
}

func TestTestContext_SetHeader(t *testing.T) {
	tc := NewTestContext(t)

	tc.SetHeader("Authorization", "Bearer token")

	assert.Equal(t, "Bearer token", tc.Context.Request.Header.Get("Authorization"))
}

func TestTestContext_ResponseCode(t *testing.T) {
	tc := NewTestContext(t)
	tc.Recorder.WriteHeader(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, tc.ResponseCode())
}

func TestNewTestUUID(t *testing.T) {
	uuid1 := NewTestUUID("test-seed")
	uuid2 := NewTestUUID("test-seed")
	uuid3 := NewTestUUID("different-seed")

	// Same seed should produce same UUID
	assert.Equal(t, uuid1, uuid2)

	// Different seed should produce different UUID
	assert.NotEqual(t, uuid1, uuid3)
}

func TestNewRandomUUID(t *testing.T) {
	uuid1 := NewRandomUUID()
	uuid2 := NewRandomUUID()

	// Random UUIDs should be different
	assert.NotEqual(t, uuid1, uuid2)
}

func TestTestTenantID(t *testing.T) {
	tenantID := TestTenantID()

	assert.NotEqual(t, tenantID.String(), "00000000-0000-0000-0000-000000000000")

	// Should be deterministic
	assert.Equal(t, TestTenantID(), tenantID)
}

func TestTestUserID(t *testing.T) {
	userID := TestUserID()

	assert.NotEqual(t, userID.String(), "00000000-0000-0000-0000-000000000000")

	// Should be deterministic
	assert.Equal(t, TestUserID(), userID)
}

func TestContextWithTimeout(t *testing.T) {
	ctx, cancel := ContextWithTimeout(t, 100*time.Millisecond)
	defer cancel()

	require.NotNil(t, ctx)

	// Context should have deadline
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))
}

func TestContextWithCancel(t *testing.T) {
	ctx, cancel := ContextWithCancel(t)

	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled yet")
	default:
		// Expected
	}

	cancel()

	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Fatal("Context should be cancelled")
	}
}

func TestAssertEventually(t *testing.T) {
	counter := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		counter = 1
	}()

	AssertEventually(t, func() bool {
		return counter == 1
	}, 200*time.Millisecond, 10*time.Millisecond)
}

func TestAssertNever(t *testing.T) {
	value := false

	AssertNever(t, func() bool {
		return value
	}, 50*time.Millisecond, 10*time.Millisecond)
}

func TestRunHTTPTestCase(t *testing.T) {
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "hello",
		})
	}

	tc := HTTPTestCase{
		Name:           "simple test",
		Method:         http.MethodGet,
		Path:           "/test",
		ExpectedStatus: http.StatusOK,
		ExpectedBody: map[string]any{
			"success": true,
		},
	}

	RunHTTPTestCase(t, handler, tc)
}

func TestRunHTTPTestCases(t *testing.T) {
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	}

	cases := []HTTPTestCase{
		{
			Name:           "case 1",
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "case 2",
			ExpectedStatus: http.StatusOK,
		},
	}

	RunHTTPTestCases(t, handler, cases)
}

func TestJSONResponse(t *testing.T) {
	tc := NewTestContext(t)
	tc.Context.JSON(http.StatusOK, gin.H{"key": "value"})

	resp := JSONResponse(t, tc)
	assert.Equal(t, "value", resp["key"])
}

func TestJSONResponseAs(t *testing.T) {
	type Response struct {
		Key string `json:"key"`
	}

	tc := NewTestContext(t)
	tc.Context.JSON(http.StatusOK, gin.H{"key": "value"})

	resp := JSONResponseAs[Response](t, tc)
	assert.Equal(t, "value", resp.Key)
}

func TestAssertSuccessResponse(t *testing.T) {
	tc := NewTestContext(t)
	tc.Context.JSON(http.StatusOK, gin.H{"success": true})

	AssertSuccessResponse(t, tc)
}

func TestToJSONReader(t *testing.T) {
	data := map[string]string{"key": "value"}
	reader := ToJSONReader(t, data)

	require.NotNil(t, reader)
}
