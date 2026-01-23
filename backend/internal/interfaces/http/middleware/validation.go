package middleware

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse represents the validation error response
type ValidationErrorResponse struct {
	Success bool              `json:"success"`
	Error   ValidationErrInfo `json:"error"`
}

// ValidationErrInfo contains validation error details
type ValidationErrInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details []ValidationError `json:"details,omitempty"`
}

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
func FormatValidationErrors(err error) ValidationErrorResponse {
	var errors []ValidationError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   e.Field(),
				Message: getValidationMessage(e),
			})
		}
	}

	return ValidationErrorResponse{
		Success: false,
		Error: ValidationErrInfo{
			Code:    "VALIDATION_ERROR",
			Message: "Request validation failed",
			Details: errors,
		},
	}
}

// HandleValidationError returns a validation error response
func HandleValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, FormatValidationErrors(err))
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
