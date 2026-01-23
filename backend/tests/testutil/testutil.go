// Package testutil provides common test utilities for the ERP backend.
// It contains helper functions for setting up test environments, creating
// mock objects, and performing common test assertions.
package testutil

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// MockDB wraps a GORM database with sqlmock for testing.
type MockDB struct {
	DB     *gorm.DB
	Mock   sqlmock.Sqlmock
	SqlDB  *sql.DB
}

// NewMockDB creates a new mock database for testing.
// The caller is responsible for calling Close() when done.
func NewMockDB(t *testing.T) *MockDB {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create sqlmock")

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err, "Failed to open GORM connection")

	return &MockDB{
		DB:    gormDB,
		Mock:  mock,
		SqlDB: mockDB,
	}
}

// Close closes the mock database connection.
func (m *MockDB) Close() error {
	return m.SqlDB.Close()
}

// ExpectationsWereMet verifies that all expectations were met.
func (m *MockDB) ExpectationsWereMet(t *testing.T) {
	t.Helper()
	err := m.Mock.ExpectationsWereMet()
	require.NoError(t, err, "Unmet database expectations")
}

// TestContext wraps a Gin test context with HTTP recorder.
type TestContext struct {
	Context  *gin.Context
	Recorder *httptest.ResponseRecorder
	Engine   *gin.Engine
}

// NewTestContext creates a new Gin test context.
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	return &TestContext{
		Context:  c,
		Recorder: w,
		Engine:   engine,
	}
}

// NewTestContextWithRequest creates a Gin test context with a custom request.
func NewTestContextWithRequest(t *testing.T, method, path string, body *http.Request) *TestContext {
	t.Helper()

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)

	if body != nil {
		c.Request = body
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}

	return &TestContext{
		Context:  c,
		Recorder: w,
		Engine:   engine,
	}
}

// SetRequestID sets a request ID in the context.
func (tc *TestContext) SetRequestID(id string) {
	tc.Context.Set("X-Request-ID", id)
}

// SetTenantID sets a tenant ID in the context.
func (tc *TestContext) SetTenantID(id string) {
	tc.Context.Set("X-Tenant-ID", id)
}

// SetUserID sets a user ID in the context.
func (tc *TestContext) SetUserID(id string) {
	tc.Context.Set("X-User-ID", id)
}

// SetHeader sets a header on the request.
func (tc *TestContext) SetHeader(key, value string) {
	tc.Context.Request.Header.Set(key, value)
}

// ResponseBody returns the response body as bytes.
func (tc *TestContext) ResponseBody() []byte {
	return tc.Recorder.Body.Bytes()
}

// ResponseCode returns the HTTP status code.
func (tc *TestContext) ResponseCode() int {
	return tc.Recorder.Code
}

// NewTestUUID generates a deterministic UUID for testing.
// Uses the provided seed string to create a reproducible UUID.
func NewTestUUID(seed string) uuid.UUID {
	// Create a namespace UUID for testing
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	return uuid.NewSHA1(namespace, []byte(seed))
}

// NewRandomUUID generates a new random UUID.
func NewRandomUUID() uuid.UUID {
	return uuid.New()
}

// TestTenantID returns a standard tenant ID for tests.
func TestTenantID() uuid.UUID {
	return NewTestUUID("test-tenant")
}

// TestUserID returns a standard user ID for tests.
func TestUserID() uuid.UUID {
	return NewTestUUID("test-user")
}

// ContextWithTimeout creates a context with a timeout for tests.
func ContextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), timeout)
}

// ContextWithCancel creates a cancellable context for tests.
func ContextWithCancel(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithCancel(context.Background())
}

// AssertEventually retries an assertion function until it passes or times out.
func AssertEventually(t *testing.T, condition func() bool, timeout, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}

	t.Fatalf("Condition not met within %v: %v", timeout, msgAndArgs)
}

// RequireEventually is like AssertEventually but fails immediately.
func RequireEventually(t *testing.T, condition func() bool, timeout, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}

	require.Fail(t, "Condition not met within timeout", msgAndArgs...)
}

// AssertNever verifies a condition never becomes true within the duration.
func AssertNever(t *testing.T, condition func() bool, duration, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if condition() {
			t.Fatalf("Condition unexpectedly became true: %v", msgAndArgs)
		}
		time.Sleep(interval)
	}
}
