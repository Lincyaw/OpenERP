package handler

import (
	"net/http"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
)

// BaseHandler provides common handler utilities
type BaseHandler struct{}

// Success sends a success response
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.NewSuccessResponse(data))
}

// SuccessWithMeta sends a success response with pagination meta
func (h *BaseHandler) SuccessWithMeta(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, dto.NewSuccessResponseWithMeta(data, total, page, pageSize))
}

// Created sends a 201 created response
func (h *BaseHandler) Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, dto.NewSuccessResponse(data))
}

// NoContent sends a 204 no content response
func (h *BaseHandler) NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error sends an error response
func (h *BaseHandler) Error(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, dto.NewErrorResponse(code, message))
}

// BadRequest sends a 400 bad request response
func (h *BaseHandler) BadRequest(c *gin.Context, message string) {
	h.Error(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// NotFound sends a 404 not found response
func (h *BaseHandler) NotFound(c *gin.Context, message string) {
	h.Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

// InternalError sends a 500 internal server error response
func (h *BaseHandler) InternalError(c *gin.Context, message string) {
	h.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// HandleDomainError converts domain errors to HTTP responses
func (h *BaseHandler) HandleDomainError(c *gin.Context, err error) {
	if domainErr, ok := err.(*shared.DomainError); ok {
		switch domainErr.Code {
		case "NOT_FOUND":
			c.JSON(http.StatusNotFound, dto.NewErrorResponse(domainErr.Code, domainErr.Message))
		case "ALREADY_EXISTS":
			c.JSON(http.StatusConflict, dto.NewErrorResponse(domainErr.Code, domainErr.Message))
		case "INVALID_INPUT", "INVALID_STATE":
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse(domainErr.Code, domainErr.Message))
		case "UNAUTHORIZED":
			c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(domainErr.Code, domainErr.Message))
		case "FORBIDDEN":
			c.JSON(http.StatusForbidden, dto.NewErrorResponse(domainErr.Code, domainErr.Message))
		case "CONCURRENCY_CONFLICT":
			c.JSON(http.StatusConflict, dto.NewErrorResponse(domainErr.Code, domainErr.Message))
		default:
			c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(domainErr.Code, domainErr.Message))
		}
		return
	}
	h.InternalError(c, "An unexpected error occurred")
}
