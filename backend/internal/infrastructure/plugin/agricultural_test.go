package plugin

import (
	"context"
	"testing"

	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAgriculturalProductValidator_Name(t *testing.T) {
	validator := NewAgriculturalProductValidator()
	assert.Equal(t, "agricultural", validator.Name())
}

func TestAgriculturalProductValidator_Type(t *testing.T) {
	validator := NewAgriculturalProductValidator()
	assert.Equal(t, strategy.StrategyTypeValidation, validator.Type())
}

func TestAgriculturalProductValidator_Validate_RequiredFields(t *testing.T) {
	validator := NewAgriculturalProductValidator()
	ctx := context.Background()
	valCtx := strategy.ValidationContext{TenantID: "tenant1", IsNew: true}

	tests := []struct {
		name          string
		data          strategy.ProductData
		expectValid   bool
		expectError   bool
		errorField    string
	}{
		{
			name: "valid product with all fields",
			data: strategy.ProductData{
				SKU:        "AGR001",
				Name:       "Test Product",
				CategoryID: "cat1",
				Price:      decimal.NewFromFloat(100),
				Cost:       decimal.NewFromFloat(50),
				Attributes: map[string]any{
					"manufacturer": "Test Manufacturer",
				},
			},
			expectValid: true,
		},
		{
			name: "missing SKU",
			data: strategy.ProductData{
				Name:       "Test Product",
				CategoryID: "cat1",
				Price:      decimal.NewFromFloat(100),
				Attributes: map[string]any{
					"manufacturer": "Test Manufacturer",
				},
			},
			expectValid: false,
			errorField:  "sku",
		},
		{
			name: "missing name",
			data: strategy.ProductData{
				SKU:        "AGR001",
				CategoryID: "cat1",
				Price:      decimal.NewFromFloat(100),
				Attributes: map[string]any{
					"manufacturer": "Test Manufacturer",
				},
			},
			expectValid: false,
			errorField:  "name",
		},
		{
			name: "missing manufacturer (required for agricultural)",
			data: strategy.ProductData{
				SKU:        "AGR001",
				Name:       "Test Product",
				CategoryID: "cat1",
				Price:      decimal.NewFromFloat(100),
				Attributes: map[string]any{},
			},
			expectValid: false,
			errorField:  "attributes.manufacturer",
		},
		{
			name: "negative price",
			data: strategy.ProductData{
				SKU:        "AGR001",
				Name:       "Test Product",
				CategoryID: "cat1",
				Price:      decimal.NewFromFloat(-100),
				Cost:       decimal.NewFromFloat(50),
				Attributes: map[string]any{
					"manufacturer": "Test Manufacturer",
				},
			},
			expectValid: false,
			errorField:  "price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(ctx, valCtx, tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectValid, result.IsValid)

			if !tt.expectValid && tt.errorField != "" {
				found := false
				for _, e := range result.Errors {
					if e.Field == tt.errorField {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error for field %s", tt.errorField)
			}
		})
	}
}

func TestAgriculturalProductValidator_Validate_PesticideCategory(t *testing.T) {
	validator := NewAgriculturalProductValidator()
	ctx := context.Background()
	valCtx := strategy.ValidationContext{TenantID: "tenant1", IsNew: true}

	tests := []struct {
		name         string
		attributes   map[string]any
		expectValid  bool
		expectErrors []string
	}{
		{
			name: "pesticide without registration number",
			attributes: map[string]any{
				"category_code": "PESTICIDE",
				"manufacturer":  "Test Manufacturer",
			},
			expectValid:  false,
			expectErrors: []string{"attributes.registration_number"},
		},
		{
			name: "pesticide with invalid registration number format",
			attributes: map[string]any{
				"category_code":       "PESTICIDE",
				"manufacturer":        "Test Manufacturer",
				"registration_number": "INVALID",
			},
			expectValid:  false,
			expectErrors: []string{"attributes.registration_number"},
		},
		{
			name: "pesticide with valid registration number",
			attributes: map[string]any{
				"category_code":       "PESTICIDE",
				"manufacturer":        "Test Manufacturer",
				"registration_number": "PD12345678",
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := strategy.ProductData{
				SKU:        "PEST001",
				Name:       "Test Pesticide",
				CategoryID: "cat1",
				Price:      decimal.NewFromFloat(100),
				Cost:       decimal.NewFromFloat(50),
				Attributes: tt.attributes,
			}

			result, err := validator.Validate(ctx, valCtx, data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectValid, result.IsValid)

			if !tt.expectValid {
				for _, expectedField := range tt.expectErrors {
					found := false
					for _, e := range result.Errors {
						if e.Field == expectedField {
							found = true
							break
						}
					}
					assert.True(t, found, "expected error for field %s", expectedField)
				}
			}
		})
	}
}

func TestAgriculturalProductValidator_Validate_SeedCategory(t *testing.T) {
	validator := NewAgriculturalProductValidator()
	ctx := context.Background()
	valCtx := strategy.ValidationContext{TenantID: "tenant1", IsNew: true}

	// Seed without variety approval number should have warning
	data := strategy.ProductData{
		SKU:        "SEED001",
		Name:       "Test Seed",
		CategoryID: "cat1",
		Price:      decimal.NewFromFloat(100),
		Cost:       decimal.NewFromFloat(50),
		Attributes: map[string]any{
			"category_code": "SEED",
			"manufacturer":  "Test Manufacturer",
		},
	}

	result, err := validator.Validate(ctx, valCtx, data)
	assert.NoError(t, err)
	assert.True(t, result.IsValid) // Valid but with warnings

	// Should have warning for missing variety approval number
	found := false
	for _, w := range result.Warnings {
		if w.Field == "attributes.variety_approval_number" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected warning for variety_approval_number")
}

func TestAgriculturalProductValidator_ValidateField(t *testing.T) {
	validator := NewAgriculturalProductValidator()
	ctx := context.Background()

	tests := []struct {
		name        string
		field       string
		value       any
		expectError bool
	}{
		{
			name:        "valid SKU",
			field:       "sku",
			value:       "SKU001",
			expectError: false,
		},
		{
			name:        "empty SKU",
			field:       "sku",
			value:       "",
			expectError: true,
		},
		{
			name:        "valid registration number",
			field:       "attributes.registration_number",
			value:       "PD12345678",
			expectError: false,
		},
		{
			name:        "invalid registration number format",
			field:       "attributes.registration_number",
			value:       "INVALID123",
			expectError: true,
		},
		{
			name:        "valid manufacturer",
			field:       "attributes.manufacturer",
			value:       "Test Manufacturer",
			expectError: false,
		},
		{
			name:        "empty manufacturer",
			field:       "attributes.manufacturer",
			value:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, err := validator.ValidateField(ctx, tt.field, tt.value)
			assert.NoError(t, err)

			if tt.expectError {
				assert.NotEmpty(t, errors, "expected validation errors")
			} else {
				assert.Empty(t, errors, "expected no validation errors")
			}
		})
	}
}
