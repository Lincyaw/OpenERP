package csvimport

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldRuleBuilder(t *testing.T) {
	t.Run("Build complete rule", func(t *testing.T) {
		minVal := decimal.NewFromInt(0)
		maxVal := decimal.NewFromInt(1000)

		rule := Field("price").
			Required().
			Decimal().
			MinValue(minVal).
			MaxValue(maxVal).
			Unique().
			Reference("product").
			Build()

		assert.Equal(t, "price", rule.Column)
		assert.True(t, rule.Required)
		assert.Equal(t, TypeDecimal, rule.Type)
		assert.Equal(t, &minVal, rule.MinValue)
		assert.Equal(t, &maxVal, rule.MaxValue)
		assert.True(t, rule.Unique)
		assert.Equal(t, "product", rule.Reference)
	})

	t.Run("String field with length", func(t *testing.T) {
		rule := Field("code").
			Required().
			String().
			MinLength(1).
			MaxLength(50).
			Build()

		assert.Equal(t, TypeString, rule.Type)
		assert.Equal(t, 1, rule.MinLength)
		assert.Equal(t, 50, rule.MaxLength)
	})

	t.Run("Pattern rule", func(t *testing.T) {
		rule := Field("phone").
			Pattern(`^\+?[\d\-]+$`, "phone number").
			Build()

		assert.NotNil(t, rule.Pattern)
		assert.Equal(t, "phone number", rule.PatternDesc)
	})

	t.Run("Date with format", func(t *testing.T) {
		rule := Field("birth_date").
			Date().
			DateFormat("02/01/2006").
			Build()

		assert.Equal(t, TypeDate, rule.Type)
		assert.Equal(t, "02/01/2006", rule.DateFormat)
	})

	t.Run("All types", func(t *testing.T) {
		testCases := []struct {
			name     string
			builder  *FieldRuleBuilder
			expected FieldType
		}{
			{"string", Field("f").String(), TypeString},
			{"int", Field("f").Int(), TypeInt},
			{"decimal", Field("f").Decimal(), TypeDecimal},
			{"date", Field("f").Date(), TypeDate},
			{"email", Field("f").Email(), TypeEmail},
			{"bool", Field("f").Bool(), TypeBool},
			{"uuid", Field("f").UUID(), TypeUUID},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := tc.builder.Build()
				assert.Equal(t, tc.expected, rule.Type)
			})
		}
	})

	t.Run("Custom validator", func(t *testing.T) {
		customFn := func(value string) error {
			return nil
		}

		rule := Field("custom").Custom(customFn).Build()
		assert.NotNil(t, rule.CustomFunc)
	})
}

func TestFieldValidator(t *testing.T) {
	t.Run("Required field validation", func(t *testing.T) {
		rules := []FieldRule{
			Field("code").Required().Build(),
			Field("name").Required().Build(),
			Field("description").Build(), // Optional
		}
		validator := NewFieldValidator(rules, 10)

		// Valid row
		row1 := &Row{
			LineNumber: 2,
			Data:       map[string]string{"code": "001", "name": "Widget", "description": ""},
		}
		assert.True(t, validator.ValidateRow(row1))

		// Missing required field
		row2 := &Row{
			LineNumber: 3,
			Data:       map[string]string{"code": "", "name": "Widget"},
		}
		assert.False(t, validator.ValidateRow(row2))

		errors := validator.Errors().Errors()
		assert.Len(t, errors, 1)
		assert.Equal(t, ErrCodeImportRequiredField, errors[0].Code)
		assert.Equal(t, "code", errors[0].Column)
	})

	t.Run("Type validation - integer", func(t *testing.T) {
		rules := []FieldRule{
			Field("quantity").Int().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Valid integer
		row1 := &Row{LineNumber: 2, Data: map[string]string{"quantity": "100"}}
		assert.True(t, validator.ValidateRow(row1))

		// Invalid integer
		row2 := &Row{LineNumber: 3, Data: map[string]string{"quantity": "abc"}}
		assert.False(t, validator.ValidateRow(row2))
	})

	t.Run("Type validation - decimal", func(t *testing.T) {
		rules := []FieldRule{
			Field("price").Decimal().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Valid decimals
		validCases := []string{"100.50", "0.01", "-50.00", "1000000.999"}
		for _, val := range validCases {
			validator.Reset()
			row := &Row{LineNumber: 2, Data: map[string]string{"price": val}}
			assert.True(t, validator.ValidateRow(row), "should accept: %s", val)
		}

		// Invalid decimal
		validator.Reset()
		row := &Row{LineNumber: 2, Data: map[string]string{"price": "not-a-number"}}
		assert.False(t, validator.ValidateRow(row))
	})

	t.Run("Type validation - date", func(t *testing.T) {
		rules := []FieldRule{
			Field("expiry_date").Date().DateFormat("2006-01-02").Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Valid date
		row1 := &Row{LineNumber: 2, Data: map[string]string{"expiry_date": "2024-12-31"}}
		assert.True(t, validator.ValidateRow(row1))

		// Invalid date
		row2 := &Row{LineNumber: 3, Data: map[string]string{"expiry_date": "31/12/2024"}}
		assert.False(t, validator.ValidateRow(row2))
	})

	t.Run("Type validation - email", func(t *testing.T) {
		rules := []FieldRule{
			Field("email").Email().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Valid email
		row1 := &Row{LineNumber: 2, Data: map[string]string{"email": "test@example.com"}}
		assert.True(t, validator.ValidateRow(row1))

		// Invalid email
		row2 := &Row{LineNumber: 3, Data: map[string]string{"email": "not-an-email"}}
		assert.False(t, validator.ValidateRow(row2))
	})

	t.Run("Type validation - boolean", func(t *testing.T) {
		rules := []FieldRule{
			Field("active").Bool().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		validBools := []string{"true", "false", "1", "0", "yes", "no", "y", "n", "TRUE", "FALSE"}
		for _, val := range validBools {
			validator.Reset()
			row := &Row{LineNumber: 2, Data: map[string]string{"active": val}}
			assert.True(t, validator.ValidateRow(row), "should accept boolean: %s", val)
		}

		validator.Reset()
		row := &Row{LineNumber: 2, Data: map[string]string{"active": "maybe"}}
		assert.False(t, validator.ValidateRow(row))
	})

	t.Run("Type validation - UUID", func(t *testing.T) {
		rules := []FieldRule{
			Field("id").UUID().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Valid UUID
		row1 := &Row{LineNumber: 2, Data: map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"}}
		assert.True(t, validator.ValidateRow(row1))

		// Invalid UUID
		row2 := &Row{LineNumber: 3, Data: map[string]string{"id": "not-a-uuid"}}
		assert.False(t, validator.ValidateRow(row2))

		// Wrong length
		validator.Reset()
		row3 := &Row{LineNumber: 4, Data: map[string]string{"id": "550e8400-e29b-41d4"}}
		assert.False(t, validator.ValidateRow(row3))
	})

	t.Run("Length validation", func(t *testing.T) {
		rules := []FieldRule{
			Field("code").MinLength(3).MaxLength(10).Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Too short
		row1 := &Row{LineNumber: 2, Data: map[string]string{"code": "AB"}}
		assert.False(t, validator.ValidateRow(row1))

		// Too long
		validator.Reset()
		row2 := &Row{LineNumber: 3, Data: map[string]string{"code": "ABCDEFGHIJK"}}
		assert.False(t, validator.ValidateRow(row2))

		// Valid length
		validator.Reset()
		row3 := &Row{LineNumber: 4, Data: map[string]string{"code": "ABCDE"}}
		assert.True(t, validator.ValidateRow(row3))
	})

	t.Run("Range validation", func(t *testing.T) {
		minVal := decimal.NewFromInt(0)
		maxVal := decimal.NewFromInt(100)
		rules := []FieldRule{
			Field("quantity").Decimal().MinValue(minVal).MaxValue(maxVal).Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Below min
		row1 := &Row{LineNumber: 2, Data: map[string]string{"quantity": "-1"}}
		assert.False(t, validator.ValidateRow(row1))

		// Above max
		validator.Reset()
		row2 := &Row{LineNumber: 3, Data: map[string]string{"quantity": "101"}}
		assert.False(t, validator.ValidateRow(row2))

		// Valid range
		validator.Reset()
		row3 := &Row{LineNumber: 4, Data: map[string]string{"quantity": "50"}}
		assert.True(t, validator.ValidateRow(row3))
	})

	t.Run("Pattern validation", func(t *testing.T) {
		rules := []FieldRule{
			Field("phone").Pattern(`^[\d\-]+$`, "phone number").Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Valid pattern
		row1 := &Row{LineNumber: 2, Data: map[string]string{"phone": "123-456-7890"}}
		assert.True(t, validator.ValidateRow(row1))

		// Invalid pattern
		row2 := &Row{LineNumber: 3, Data: map[string]string{"phone": "abc-def-ghij"}}
		assert.False(t, validator.ValidateRow(row2))
	})

	t.Run("Uniqueness within file", func(t *testing.T) {
		rules := []FieldRule{
			Field("code").Unique().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		row1 := &Row{LineNumber: 2, Data: map[string]string{"code": "SKU-001"}}
		row2 := &Row{LineNumber: 3, Data: map[string]string{"code": "SKU-002"}}
		row3 := &Row{LineNumber: 4, Data: map[string]string{"code": "SKU-001"}} // Duplicate

		assert.True(t, validator.ValidateRow(row1))
		assert.True(t, validator.ValidateRow(row2))
		assert.False(t, validator.ValidateRow(row3))

		errors := validator.Errors().Errors()
		// Should have duplicate error
		hasDuplicateError := false
		for _, err := range errors {
			if err.Code == ErrCodeImportDuplicateInFile {
				hasDuplicateError = true
				break
			}
		}
		assert.True(t, hasDuplicateError)
	})

	t.Run("Custom validation", func(t *testing.T) {
		customValidator := func(value string) error {
			if len(value) > 0 && value[0] != 'A' {
				return assert.AnError
			}
			return nil
		}

		rules := []FieldRule{
			Field("code").Custom(customValidator).Build(),
		}
		validator := NewFieldValidator(rules, 10)

		// Passes custom validation
		row1 := &Row{LineNumber: 2, Data: map[string]string{"code": "ABC"}}
		assert.True(t, validator.ValidateRow(row1))

		// Fails custom validation
		row2 := &Row{LineNumber: 3, Data: map[string]string{"code": "XYZ"}}
		assert.False(t, validator.ValidateRow(row2))
	})

	t.Run("Skip validation for empty optional fields", func(t *testing.T) {
		rules := []FieldRule{
			Field("email").Email().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		row := &Row{LineNumber: 2, Data: map[string]string{"email": ""}}
		assert.True(t, validator.ValidateRow(row))
	})

	t.Run("Reset clears state", func(t *testing.T) {
		rules := []FieldRule{
			Field("code").Unique().Build(),
		}
		validator := NewFieldValidator(rules, 10)

		row1 := &Row{LineNumber: 2, Data: map[string]string{"code": "SKU-001"}}
		validator.ValidateRow(row1)

		validator.Reset()

		// After reset, same code should be valid again
		row2 := &Row{LineNumber: 3, Data: map[string]string{"code": "SKU-001"}}
		assert.True(t, validator.ValidateRow(row2))
	})
}

func TestReferenceValidator(t *testing.T) {
	t.Run("Valid reference", func(t *testing.T) {
		lookupFn := func(refType, value string) (bool, error) {
			validRefs := map[string][]string{
				"category": {"CAT-001", "CAT-002"},
				"supplier": {"SUP-001"},
			}
			for _, v := range validRefs[refType] {
				if v == value {
					return true, nil
				}
			}
			return false, nil
		}

		validator := NewReferenceValidator(lookupFn, 10)

		// Valid reference
		assert.True(t, validator.ValidateReference(2, "category_id", "category", "CAT-001"))

		// Invalid reference
		assert.False(t, validator.ValidateReference(3, "category_id", "category", "CAT-999"))

		errors := validator.Errors().Errors()
		require.Len(t, errors, 1)
		assert.Equal(t, ErrCodeImportReferenceNotFound, errors[0].Code)
	})

	t.Run("Caching references", func(t *testing.T) {
		callCount := 0
		lookupFn := func(refType, value string) (bool, error) {
			callCount++
			return value == "VALID", nil
		}

		validator := NewReferenceValidator(lookupFn, 10)

		// First call - should lookup
		validator.ValidateReference(2, "col", "type", "VALID")
		assert.Equal(t, 1, callCount)

		// Second call - should use cache
		validator.ValidateReference(3, "col", "type", "VALID")
		assert.Equal(t, 1, callCount)

		// Different value - should lookup again
		validator.ValidateReference(4, "col", "type", "INVALID")
		assert.Equal(t, 2, callCount)
	})

	t.Run("Empty value skipped", func(t *testing.T) {
		callCount := 0
		lookupFn := func(refType, value string) (bool, error) {
			callCount++
			return true, nil
		}

		validator := NewReferenceValidator(lookupFn, 10)
		assert.True(t, validator.ValidateReference(2, "col", "type", ""))
		assert.Equal(t, 0, callCount)
	})

	t.Run("PreloadReferences", func(t *testing.T) {
		lookupFn := func(refType, value string) (bool, error) {
			return value == "VALID1" || value == "VALID2", nil
		}

		validator := NewReferenceValidator(lookupFn, 10)
		err := validator.PreloadReferences("type", []string{"VALID1", "VALID2", "INVALID"})
		require.NoError(t, err)

		// Preloaded values should use cache
		assert.True(t, validator.ValidateReference(2, "col", "type", "VALID1"))
		assert.True(t, validator.ValidateReference(3, "col", "type", "VALID2"))
		assert.False(t, validator.ValidateReference(4, "col", "type", "INVALID"))
	})

	t.Run("Reset clears cache", func(t *testing.T) {
		callCount := 0
		lookupFn := func(refType, value string) (bool, error) {
			callCount++
			return true, nil
		}

		validator := NewReferenceValidator(lookupFn, 10)

		validator.ValidateReference(2, "col", "type", "VALUE")
		assert.Equal(t, 1, callCount)

		validator.Reset()

		// After reset, should lookup again
		validator.ValidateReference(3, "col", "type", "VALUE")
		assert.Equal(t, 2, callCount)
	})
}

func TestUniquenessValidator(t *testing.T) {
	t.Run("Value does not exist in DB", func(t *testing.T) {
		lookupFn := func(entityType, field, value string) (bool, error) {
			return false, nil
		}

		validator := NewUniquenessValidator(lookupFn, 10)
		assert.True(t, validator.ValidateUnique(2, "code", "products", "SKU-001"))
	})

	t.Run("Value exists in DB", func(t *testing.T) {
		lookupFn := func(entityType, field, value string) (bool, error) {
			return value == "EXISTING", nil
		}

		validator := NewUniquenessValidator(lookupFn, 10)
		assert.False(t, validator.ValidateUnique(2, "code", "products", "EXISTING"))

		errors := validator.Errors().Errors()
		require.Len(t, errors, 1)
		assert.Equal(t, ErrCodeImportDuplicateInDB, errors[0].Code)
	})

	t.Run("Empty value skipped", func(t *testing.T) {
		lookupFn := func(entityType, field, value string) (bool, error) {
			return true, nil // Would fail if called
		}

		validator := NewUniquenessValidator(lookupFn, 10)
		assert.True(t, validator.ValidateUnique(2, "code", "products", ""))
	})
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		uuid    string
		wantErr bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID lowercase", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"valid UUID mixed case", "550e8400-E29B-41d4-A716-446655440000", false},
		{"too short", "550e8400-e29b-41d4", true},
		{"too long", "550e8400-e29b-41d4-a716-446655440000-extra", true},
		{"wrong format - missing dashes", "550e8400e29b41d4a716446655440000", true},
		{"wrong format - wrong dash positions", "550e-8400-e29b-41d4-a716-446655440000", true},
		{"invalid characters", "550g8400-e29b-41d4-a716-446655440000", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUUID(tt.uuid)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
