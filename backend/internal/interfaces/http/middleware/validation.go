package middleware

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RequestIDKey is the context key for request ID
const RequestIDKey = "X-Request-ID"

// SetupValidator configures the validator with custom tags
func SetupValidator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// Use JSON tag names for field names in errors
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name == "" {
				name = strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
			}
			return name
		})
	}
}

// FormatValidationErrors formats validation errors into a standard response
func FormatValidationErrors(err error, requestID string) dto.Response {
	var details []dto.ValidationDetail

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			details = append(details, dto.ValidationDetail{
				Field:   e.Field(),
				Message: getValidationMessage(e),
			})
		}
	}

	return dto.NewValidationErrorResponse(
		"Request validation failed",
		requestID,
		details,
	)
}

// HandleValidationError returns a validation error response
func HandleValidationError(c *gin.Context, err error) {
	requestID := getRequestIDFromContext(c)
	c.JSON(http.StatusBadRequest, FormatValidationErrors(err, requestID))
}

// getRequestIDFromContext extracts request ID from gin context
func getRequestIDFromContext(c *gin.Context) string {
	if id := c.GetString(RequestIDKey); id != "" {
		return id
	}
	if id := c.GetHeader(RequestIDKey); id != "" {
		return id
	}
	return ""
}

// getValidationMessage returns a human-readable validation message
func getValidationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		if e.Type().Kind() == reflect.String {
			return "Must be at least " + e.Param() + " characters"
		}
		return "Must be at least " + e.Param()
	case "max":
		if e.Type().Kind() == reflect.String {
			return "Must be at most " + e.Param() + " characters"
		}
		return "Must be at most " + e.Param()
	case "len":
		return "Must be exactly " + e.Param() + " characters"
	case "uuid":
		return "Invalid UUID format"
	case "oneof":
		return "Must be one of: " + e.Param()
	case "gte":
		return "Must be greater than or equal to " + e.Param()
	case "lte":
		return "Must be less than or equal to " + e.Param()
	case "gt":
		return "Must be greater than " + e.Param()
	case "lt":
		return "Must be less than " + e.Param()
	case "url":
		return "Invalid URL format"
	case "numeric":
		return "Must be numeric"
	case "alphanum":
		return "Must be alphanumeric"
	case "alpha":
		return "Must contain only letters"
	default:
		return "Invalid value"
	}
}
