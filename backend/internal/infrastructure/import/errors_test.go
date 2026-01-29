package csvimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRowError(t *testing.T) {
	t.Run("Error with column", func(t *testing.T) {
		err := NewRowError(5, "email", ErrCodeImportInvalidFormat, "invalid email format")
		assert.Equal(t, "row 5, column 'email': invalid email format", err.Error())
	})

	t.Run("Error without column", func(t *testing.T) {
		err := NewRowError(10, "", ErrCodeImportCSVParsing, "malformed row")
		assert.Equal(t, "row 10: malformed row", err.Error())
	})

	t.Run("Error with value", func(t *testing.T) {
		err := NewRowErrorWithValue(3, "phone", ErrCodeImportPatternMismatch, "invalid phone", "abc123")
		assert.Equal(t, "abc123", err.Value)
		assert.Equal(t, 3, err.Row)
	})
}

func TestErrorCollection(t *testing.T) {
	t.Run("Add errors within limit", func(t *testing.T) {
		ec := NewErrorCollection(10)

		ec.Add(NewRowError(1, "col1", ErrCodeImportValidation, "error 1"))
		ec.Add(NewRowError(2, "col2", ErrCodeImportValidation, "error 2"))
		ec.Add(NewRowError(3, "col3", ErrCodeImportValidation, "error 3"))

		assert.Equal(t, 3, ec.Count())
		assert.Equal(t, 3, ec.TotalCount())
		assert.True(t, ec.HasErrors())
		assert.False(t, ec.IsTruncated())
	})

	t.Run("Add errors exceeding limit", func(t *testing.T) {
		ec := NewErrorCollection(3)

		for i := 1; i <= 5; i++ {
			ec.Add(NewRowError(i, "col", ErrCodeImportValidation, "error"))
		}

		assert.Equal(t, 3, ec.Count())
		assert.Equal(t, 5, ec.TotalCount())
		assert.True(t, ec.IsTruncated())
	})

	t.Run("Helper methods", func(t *testing.T) {
		ec := NewErrorCollection(10)

		ec.AddRequiredError(1, "name")
		ec.AddTypeError(2, "price", "decimal", "abc")
		ec.AddFormatError(3, "email", "email@domain.com", "invalid")
		ec.AddLengthError(4, "code", 1, 50)
		ec.AddRangeError(5, "quantity", 0, 100)
		ec.AddPatternError(6, "phone", "phone number", "xyz")
		ec.AddDuplicateError(7, "sku", "SKU-001", false)
		ec.AddDuplicateError(8, "sku", "SKU-002", true)
		ec.AddReferenceError(9, "category", "CAT-999", "category")

		assert.Equal(t, 9, ec.Count())

		// Check error codes
		errors := ec.Errors()
		assert.Equal(t, ErrCodeImportRequiredField, errors[0].Code)
		assert.Equal(t, ErrCodeImportInvalidType, errors[1].Code)
		assert.Equal(t, ErrCodeImportInvalidFormat, errors[2].Code)
		assert.Equal(t, ErrCodeImportInvalidLength, errors[3].Code)
		assert.Equal(t, ErrCodeImportInvalidRange, errors[4].Code)
		assert.Equal(t, ErrCodeImportPatternMismatch, errors[5].Code)
		assert.Equal(t, ErrCodeImportDuplicateInFile, errors[6].Code)
		assert.Equal(t, ErrCodeImportDuplicateInDB, errors[7].Code)
		assert.Equal(t, ErrCodeImportReferenceNotFound, errors[8].Code)
	})

	t.Run("Error summary", func(t *testing.T) {
		ec := NewErrorCollection(10)

		ec.Add(NewRowError(1, "col", ErrCodeImportValidation, "err1"))
		ec.Add(NewRowError(2, "col", ErrCodeImportValidation, "err2"))
		ec.Add(NewRowError(3, "col", ErrCodeImportRequiredField, "err3"))

		summary := ec.ErrorSummary()
		assert.Equal(t, 2, summary[ErrCodeImportValidation])
		assert.Equal(t, 1, summary[ErrCodeImportRequiredField])
	})

	t.Run("Clear", func(t *testing.T) {
		ec := NewErrorCollection(10)
		ec.Add(NewRowError(1, "col", ErrCodeImportValidation, "err"))

		ec.Clear()

		assert.False(t, ec.HasErrors())
		assert.Equal(t, 0, ec.Count())
	})

	t.Run("String representation", func(t *testing.T) {
		ec := NewErrorCollection(10)
		ec.Add(NewRowError(1, "name", ErrCodeImportRequiredField, "field is required"))
		ec.Add(NewRowError(2, "email", ErrCodeImportInvalidFormat, "invalid email"))

		s := ec.String()
		assert.Contains(t, s, "2 error(s) found")
		assert.Contains(t, s, "row 1, column 'name'")
		assert.Contains(t, s, "row 2, column 'email'")
	})
}

func TestValidationResult(t *testing.T) {
	t.Run("New validation result", func(t *testing.T) {
		vr := NewValidationResult("test-id")

		assert.Equal(t, "test-id", vr.ValidationID)
		assert.Empty(t, vr.Errors)
		assert.Empty(t, vr.Preview)
	})

	t.Run("Set counts", func(t *testing.T) {
		vr := NewValidationResult("test-id")
		vr.SetCounts(100, 95, 5)

		assert.Equal(t, 100, vr.TotalRows)
		assert.Equal(t, 95, vr.ValidRows)
		assert.Equal(t, 5, vr.ErrorRows)
	})

	t.Run("Add preview", func(t *testing.T) {
		vr := NewValidationResult("test-id")

		for i := 0; i < 10; i++ {
			vr.AddPreview(map[string]any{"row": i})
		}

		// Should only keep first 5
		assert.Equal(t, 5, len(vr.Preview))
	})

	t.Run("Set errors from collection", func(t *testing.T) {
		vr := NewValidationResult("test-id")
		ec := NewErrorCollection(5)

		for i := 0; i < 10; i++ {
			ec.Add(NewRowError(i, "col", ErrCodeImportValidation, "error"))
		}

		vr.SetErrors(ec)

		assert.Equal(t, 5, len(vr.Errors))
		assert.True(t, vr.IsTruncated)
		assert.Equal(t, 10, vr.TotalErrors)
	})

	t.Run("IsValid", func(t *testing.T) {
		vr := NewValidationResult("test-id")
		vr.SetCounts(100, 100, 0)
		assert.True(t, vr.IsValid())

		vr.SetCounts(100, 95, 5)
		assert.False(t, vr.IsValid())
	})
}

func TestLengthErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		minLen   int
		maxLen   int
		expected string
	}{
		{"both limits", 1, 50, "length must be between 1 and 50"},
		{"only max", 0, 100, "length must be at most 100"},
		{"only min", 5, 0, "length must be at least 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec := NewErrorCollection(10)
			ec.AddLengthError(1, "field", tt.minLen, tt.maxLen)

			errors := ec.Errors()
			require.Len(t, errors, 1)
			assert.Contains(t, errors[0].Message, strings.Split(tt.expected, " ")[2])
		})
	}
}
