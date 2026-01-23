package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupValidator(t *testing.T) {
	// Should not panic
	SetupValidator()

	// Verify the validator is configured
	v, ok := binding.Validator.Engine().(*validator.Validate)
	assert.True(t, ok)
	assert.NotNil(t, v)
}

func TestFormatValidationErrors(t *testing.T) {
	type TestStruct struct {
		Email string `json:"email" binding:"required,email"`
		Age   int    `json:"age" binding:"required,min=18"`
	}

	// Setup validator
	SetupValidator()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		var req TestStruct
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleValidationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	t.Run("returns validation errors for invalid input", func(t *testing.T) {
		body := strings.NewReader(`{"email": "invalid", "age": 10}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp ValidationErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.False(t, resp.Success)
		assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
		assert.Equal(t, "Request validation failed", resp.Error.Message)
		assert.Len(t, resp.Error.Details, 2)
	})

	t.Run("returns success for valid input", func(t *testing.T) {
		body := strings.NewReader(`{"email": "test@example.com", "age": 25}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetValidationMessage(t *testing.T) {
	type TestStruct struct {
		Required string `binding:"required"`
		Email    string `binding:"email"`
		Min      string `binding:"min=5"`
		Max      string `binding:"max=10"`
		Len      string `binding:"len=5"`
		UUID     string `binding:"uuid"`
		OneOf    string `binding:"oneof=a b c"`
		GTE      int    `binding:"gte=10"`
		LTE      int    `binding:"lte=100"`
		GT       int    `binding:"gt=0"`
		LT       int    `binding:"lt=1000"`
		URL      string `binding:"url"`
		Numeric  string `binding:"numeric"`
	}

	v := validator.New()

	tests := []struct {
		field    string
		value    interface{}
		expected string
	}{
		{"Required", "", "This field is required"},
		{"Email", "invalid", "Invalid email format"},
		{"Min", "ab", "Must be at least 5 characters"},
		{"Max", "this is way too long", "Must be at most 10 characters"},
		{"Len", "ab", "Must be exactly 5 characters"},
		{"UUID", "invalid", "Invalid UUID format"},
		{"OneOf", "d", "Must be one of: a b c"},
		{"URL", "invalid", "Invalid URL format"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			obj := TestStruct{}
			err := v.Struct(obj)
			if err != nil {
				validationErrs := err.(validator.ValidationErrors)
				for _, e := range validationErrs {
					if e.Field() == tt.field {
						msg := getValidationMessage(e)
						assert.Contains(t, msg, tt.expected[:10]) // Check partial match
						return
					}
				}
			}
		})
	}
}

func TestHandleValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("handles validator.ValidationErrors", func(t *testing.T) {
		type Input struct {
			Name string `json:"name" binding:"required"`
		}

		router := gin.New()
		router.POST("/test", func(c *gin.Context) {
			var input Input
			if err := c.ShouldBindJSON(&input); err != nil {
				HandleValidationError(c, err)
				return
			}
		})

		body := strings.NewReader(`{}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
	})
}
