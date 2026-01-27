package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setJWTContext sets JWT context values for testing
// This simulates authenticated requests without actual JWT tokens
func setJWTContext(c *gin.Context, tenantID, userID uuid.UUID) {
	c.Set("jwt_tenant_id", tenantID.String())
	c.Set("jwt_user_id", userID.String())
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*gin.Context)
		expectedID string
	}{
		{
			name: "from context string",
			setup: func(c *gin.Context) {
				c.Set(RequestIDKey, "ctx-request-id")
			},
			expectedID: "ctx-request-id",
		},
		{
			name: "from header when context empty",
			setup: func(c *gin.Context) {
				c.Request.Header.Set(RequestIDKey, "header-request-id")
			},
			expectedID: "header-request-id",
		},
		{
			name:       "empty when not set",
			setup:      func(c *gin.Context) {},
			expectedID: "",
		},
		{
			name: "context takes precedence over header",
			setup: func(c *gin.Context) {
				c.Set(RequestIDKey, "ctx-id")
				c.Request.Header.Set(RequestIDKey, "header-id")
			},
			expectedID: "ctx-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			tt.setup(c)

			id := getRequestID(c)
			assert.Equal(t, tt.expectedID, id)
		})
	}
}

func TestBaseHandlerSuccess(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"key": "value"}
	h.Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestBaseHandlerSuccessWithMeta(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := []string{"item1", "item2"}
	h.SuccessWithMeta(c, data, 100, 1, 10)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, int64(100), resp.Meta.Total)
}

func TestBaseHandlerCreated(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.Created(c, map[string]string{"id": "123"})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestBaseHandlerNoContent(t *testing.T) {
	h := &BaseHandler{}

	router := gin.New()
	router.DELETE("/test", func(c *gin.Context) {
		h.NoContent(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.Bytes())
}

func TestBaseHandlerErrorMethods(t *testing.T) {
	tests := []struct {
		name         string
		method       func(*BaseHandler, *gin.Context)
		expectedCode int
		expectedErr  string
	}{
		{
			name: "BadRequest",
			method: func(h *BaseHandler, c *gin.Context) {
				h.BadRequest(c, "Invalid request")
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  dto.ErrCodeBadRequest,
		},
		{
			name: "NotFound",
			method: func(h *BaseHandler, c *gin.Context) {
				h.NotFound(c, "Resource not found")
			},
			expectedCode: http.StatusNotFound,
			expectedErr:  dto.ErrCodeNotFound,
		},
		{
			name: "Unauthorized",
			method: func(h *BaseHandler, c *gin.Context) {
				h.Unauthorized(c, "Not authenticated")
			},
			expectedCode: http.StatusUnauthorized,
			expectedErr:  dto.ErrCodeUnauthorized,
		},
		{
			name: "Forbidden",
			method: func(h *BaseHandler, c *gin.Context) {
				h.Forbidden(c, "Access denied")
			},
			expectedCode: http.StatusForbidden,
			expectedErr:  dto.ErrCodeForbidden,
		},
		{
			name: "Conflict",
			method: func(h *BaseHandler, c *gin.Context) {
				h.Conflict(c, "Resource conflict")
			},
			expectedCode: http.StatusConflict,
			expectedErr:  dto.ErrCodeConflict,
		},
		{
			name: "InternalError",
			method: func(h *BaseHandler, c *gin.Context) {
				h.InternalError(c, "Server error")
			},
			expectedCode: http.StatusInternalServerError,
			expectedErr:  dto.ErrCodeInternal,
		},
		{
			name: "TooManyRequests",
			method: func(h *BaseHandler, c *gin.Context) {
				h.TooManyRequests(c, "Rate limit exceeded")
			},
			expectedCode: http.StatusTooManyRequests,
			expectedErr:  dto.ErrCodeRateLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &BaseHandler{}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			tt.method(h, c)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp dto.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.False(t, resp.Success)
			assert.Equal(t, tt.expectedErr, resp.Error.Code)
		})
	}
}

func TestBaseHandlerErrorWithRequestID(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set(RequestIDKey, "test-request-123")

	h.BadRequest(c, "Invalid request")

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test-request-123", resp.Error.RequestID)
}

func TestBaseHandlerErrorWithCode(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	h.ErrorWithCode(c, dto.ErrCodeInsufficientStock, "Not enough items")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code) // Business rule errors -> 422

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, dto.ErrCodeInsufficientStock, resp.Error.Code)
}

func TestBaseHandlerValidationError(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set(RequestIDKey, "val-req-456")

	details := []dto.ValidationDetail{
		{Field: "email", Message: "Invalid format"},
		{Field: "name", Message: "Required"},
	}
	h.ValidationError(c, details)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, dto.ErrCodeValidation, resp.Error.Code)
	assert.Equal(t, "val-req-456", resp.Error.RequestID)
	assert.Len(t, resp.Error.Details, 2)
}

func TestBaseHandlerHandleDomainError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedErr  string
	}{
		{
			name:         "NOT_FOUND error",
			err:          shared.ErrNotFound,
			expectedCode: http.StatusNotFound,
			expectedErr:  dto.ErrCodeNotFound,
		},
		{
			name:         "ALREADY_EXISTS error",
			err:          shared.ErrAlreadyExists,
			expectedCode: http.StatusConflict,
			expectedErr:  dto.ErrCodeAlreadyExists,
		},
		{
			name:         "INVALID_INPUT error",
			err:          shared.ErrInvalidInput,
			expectedCode: http.StatusBadRequest,
			expectedErr:  dto.ErrCodeInvalidInput,
		},
		{
			name:         "UNAUTHORIZED error",
			err:          shared.ErrUnauthorized,
			expectedCode: http.StatusUnauthorized,
			expectedErr:  dto.ErrCodeUnauthorized,
		},
		{
			name:         "FORBIDDEN error",
			err:          shared.ErrForbidden,
			expectedCode: http.StatusForbidden,
			expectedErr:  dto.ErrCodeForbidden,
		},
		{
			name:         "INVALID_STATE error",
			err:          shared.ErrInvalidState,
			expectedCode: http.StatusUnprocessableEntity,
			expectedErr:  dto.ErrCodeInvalidState,
		},
		{
			name:         "CONCURRENCY_CONFLICT error",
			err:          shared.ErrConcurrencyConflict,
			expectedCode: http.StatusConflict,
			expectedErr:  dto.ErrCodeConcurrencyConflict,
		},
		{
			name:         "INSUFFICIENT_STOCK error",
			err:          shared.ErrInsufficientStock,
			expectedCode: http.StatusUnprocessableEntity,
			expectedErr:  dto.ErrCodeInsufficientStock,
		},
		{
			name:         "INSUFFICIENT_BALANCE error",
			err:          shared.ErrInsufficientBalance,
			expectedCode: http.StatusUnprocessableEntity,
			expectedErr:  dto.ErrCodeInsufficientBalance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &BaseHandler{}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			h.HandleDomainError(c, tt.err)

			assert.Equal(t, tt.expectedCode, w.Code)

			var resp dto.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.False(t, resp.Success)
			assert.Equal(t, tt.expectedErr, resp.Error.Code)
		})
	}
}

func TestBaseHandlerHandleDomainErrorWithRequestID(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set(RequestIDKey, "domain-err-req")

	h.HandleDomainError(c, shared.ErrNotFound)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "domain-err-req", resp.Error.RequestID)
}

func TestBaseHandlerHandleNonDomainError(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	// Standard error (not DomainError)
	h.HandleDomainError(c, assert.AnError)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, dto.ErrCodeInternal, resp.Error.Code)
	assert.Equal(t, "An unexpected error occurred", resp.Error.Message)
}

func TestBaseHandlerHandleError(t *testing.T) {
	h := &BaseHandler{}

	t.Run("handles nil error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		h.HandleError(c, nil)

		// Should not write anything for nil error
		assert.Equal(t, http.StatusOK, w.Code) // Default status
	})

	t.Run("handles domain error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		h.HandleError(c, shared.ErrNotFound)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles standard error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		h.HandleError(c, assert.AnError)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles wrapped domain error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		// Wrap a domain error
		wrappedErr := fmt.Errorf("additional context: %w", shared.ErrNotFound)
		h.HandleError(c, wrappedErr)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp dto.Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, dto.ErrCodeNotFound, resp.Error.Code)
	})
}

func TestBaseHandlerUnprocessableEntity(t *testing.T) {
	h := &BaseHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	h.UnprocessableEntity(c, dto.ErrCodeBusinessRule, "Business rule violated")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, dto.ErrCodeBusinessRule, resp.Error.Code)
}
