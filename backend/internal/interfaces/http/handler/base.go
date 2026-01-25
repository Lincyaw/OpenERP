package handler

import (
	"errors"
	"net/http"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey is the context key for request ID
const RequestIDKey = "X-Request-ID"

// BaseHandler provides common handler utilities
type BaseHandler struct{}

// getRequestID extracts the request ID from the context
func getRequestID(c *gin.Context) string {
	if id := c.GetString(RequestIDKey); id != "" {
		return id
	}
	if id := c.GetHeader(RequestIDKey); id != "" {
		return id
	}
	return ""
}

// getUserID extracts user ID from JWT claims or returns error
func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr := middleware.GetJWTUserID(c)
	if userIDStr == "" {
		// Fallback to header for development (will be removed in production)
		userIDStr = c.GetHeader("X-User-ID")
	}
	if userIDStr == "" {
		return uuid.Nil, errors.New("user ID not found in context")
	}
	return uuid.Parse(userIDStr)
}

// getTenantID extracts tenant ID from JWT claims or returns error
func getTenantID(c *gin.Context) (uuid.UUID, error) {
	tenantIDStr := middleware.GetJWTTenantID(c)
	if tenantIDStr == "" {
		// Fallback to header for development
		tenantIDStr = c.GetHeader("X-Tenant-ID")
	}
	if tenantIDStr == "" {
		// Default development tenant for backwards compatibility
		return uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil
	}
	return uuid.Parse(tenantIDStr)
}

// Success sends a success response
func (h *BaseHandler) Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, dto.NewSuccessResponse(data))
}

// SuccessWithMeta sends a success response with pagination meta
func (h *BaseHandler) SuccessWithMeta(c *gin.Context, data any, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, dto.NewSuccessResponseWithMeta(data, total, page, pageSize))
}

// Created sends a 201 created response
func (h *BaseHandler) Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(data))
}

// NoContent sends a 204 no content response
func (h *BaseHandler) NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error sends an error response with the appropriate status code
func (h *BaseHandler) Error(c *gin.Context, statusCode int, code, message string) {
	requestID := getRequestID(c)
	c.JSON(statusCode, dto.NewErrorResponseWithRequestID(code, message, requestID))
}

// ErrorWithCode sends an error response, deriving status code from error code
func (h *BaseHandler) ErrorWithCode(c *gin.Context, code, message string) {
	statusCode := dto.GetHTTPStatus(code)
	requestID := getRequestID(c)
	c.JSON(statusCode, dto.NewErrorResponseWithRequestID(code, message, requestID))
}

// BadRequest sends a 400 bad request response
func (h *BaseHandler) BadRequest(c *gin.Context, message string) {
	h.Error(c, http.StatusBadRequest, dto.ErrCodeBadRequest, message)
}

// NotFound sends a 404 not found response
func (h *BaseHandler) NotFound(c *gin.Context, message string) {
	h.Error(c, http.StatusNotFound, dto.ErrCodeNotFound, message)
}

// Unauthorized sends a 401 unauthorized response
func (h *BaseHandler) Unauthorized(c *gin.Context, message string) {
	h.Error(c, http.StatusUnauthorized, dto.ErrCodeUnauthorized, message)
}

// Forbidden sends a 403 forbidden response
func (h *BaseHandler) Forbidden(c *gin.Context, message string) {
	h.Error(c, http.StatusForbidden, dto.ErrCodeForbidden, message)
}

// Conflict sends a 409 conflict response
func (h *BaseHandler) Conflict(c *gin.Context, message string) {
	h.Error(c, http.StatusConflict, dto.ErrCodeConflict, message)
}

// UnprocessableEntity sends a 422 unprocessable entity response
func (h *BaseHandler) UnprocessableEntity(c *gin.Context, code, message string) {
	h.Error(c, http.StatusUnprocessableEntity, code, message)
}

// InternalError sends a 500 internal server error response
func (h *BaseHandler) InternalError(c *gin.Context, message string) {
	h.Error(c, http.StatusInternalServerError, dto.ErrCodeInternal, message)
}

// TooManyRequests sends a 429 too many requests response
func (h *BaseHandler) TooManyRequests(c *gin.Context, message string) {
	h.Error(c, http.StatusTooManyRequests, dto.ErrCodeRateLimited, message)
}

// ValidationError sends a 400 validation error response with details
func (h *BaseHandler) ValidationError(c *gin.Context, details []dto.ValidationDetail) {
	requestID := getRequestID(c)
	c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
		"Request validation failed",
		requestID,
		details,
	))
}

// HandleDomainError converts domain errors to HTTP responses
func (h *BaseHandler) HandleDomainError(c *gin.Context, err error) {
	requestID := getRequestID(c)

	var domainErr *shared.DomainError
	if errors.As(err, &domainErr) {
		code := dto.NormalizeErrorCode(domainErr.Code)
		statusCode := dto.GetHTTPStatus(code)
		c.JSON(statusCode, dto.NewErrorResponseWithRequestID(code, domainErr.Message, requestID))
		return
	}

	// Unknown error type - return as internal error
	h.InternalError(c, "An unexpected error occurred")
}

// HandleError is a generic error handler that handles both domain and standard errors
func (h *BaseHandler) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	requestID := getRequestID(c)

	// Check for domain error using errors.As for wrapped error support
	var domainErr *shared.DomainError
	if errors.As(err, &domainErr) {
		code := dto.NormalizeErrorCode(domainErr.Code)
		statusCode := dto.GetHTTPStatus(code)
		c.JSON(statusCode, dto.NewErrorResponseWithRequestID(code, domainErr.Message, requestID))
		return
	}

	// Default to internal error for unknown error types
	c.JSON(http.StatusInternalServerError, dto.NewErrorResponseWithRequestID(
		dto.ErrCodeInternal,
		"An unexpected error occurred",
		requestID,
	))
}
