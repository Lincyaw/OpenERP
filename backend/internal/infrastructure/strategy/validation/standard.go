package validation

import (
	"context"
	"strings"

	"github.com/erp/backend/internal/domain/shared/strategy"
)

// StandardProductValidator implements basic product validation
type StandardProductValidator struct {
	strategy.BaseStrategy
}

// NewStandardProductValidator creates a new standard product validator
func NewStandardProductValidator() *StandardProductValidator {
	return &StandardProductValidator{
		BaseStrategy: strategy.NewBaseStrategy(
			"standard",
			strategy.StrategyTypeValidation,
			"Standard product validation for required fields",
		),
	}
}

// Validate validates product data
func (s *StandardProductValidator) Validate(
	ctx context.Context,
	valCtx strategy.ValidationContext,
	data strategy.ProductData,
) (strategy.ValidationResult, error) {
	result := strategy.ValidationResult{
		IsValid:  true,
		Errors:   make([]strategy.ValidationError, 0),
		Warnings: make([]strategy.ValidationWarning, 0),
	}

	// Validate required fields
	if strings.TrimSpace(data.SKU) == "" {
		result.AddError("sku", "REQUIRED", "SKU is required")
	}

	if strings.TrimSpace(data.Name) == "" {
		result.AddError("name", "REQUIRED", "Product name is required")
	}

	if strings.TrimSpace(data.CategoryID) == "" {
		result.AddError("categoryId", "REQUIRED", "Category is required")
	}

	if strings.TrimSpace(data.UnitID) == "" {
		result.AddError("unitId", "REQUIRED", "Unit is required")
	}

	// Validate price and cost
	if data.Price.IsNegative() {
		result.AddError("price", "INVALID", "Price cannot be negative")
	}

	if data.Cost.IsNegative() {
		result.AddError("cost", "INVALID", "Cost cannot be negative")
	}

	// Validate stock levels
	if data.MinStock.IsNegative() {
		result.AddError("minStock", "INVALID", "Minimum stock cannot be negative")
	}

	if data.MaxStock.IsNegative() {
		result.AddError("maxStock", "INVALID", "Maximum stock cannot be negative")
	}

	if !data.MaxStock.IsZero() && data.MinStock.GreaterThan(data.MaxStock) {
		result.AddError("minStock", "INVALID", "Minimum stock cannot exceed maximum stock")
	}

	// Add warnings
	if data.Price.IsZero() {
		result.AddWarning("price", "ZERO_PRICE", "Product has zero price")
	}

	if data.Cost.IsZero() {
		result.AddWarning("cost", "ZERO_COST", "Product has zero cost")
	}

	if !data.Cost.IsZero() && !data.Price.IsZero() {
		if data.Price.LessThan(data.Cost) {
			result.AddWarning("price", "LOW_MARGIN", "Price is lower than cost")
		}
	}

	return result, nil
}

// ValidateField validates a single field
func (s *StandardProductValidator) ValidateField(
	ctx context.Context,
	field string,
	value any,
) ([]strategy.ValidationError, error) {
	errors := make([]strategy.ValidationError, 0)

	switch field {
	case "sku", "name", "categoryId", "unitId":
		if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
			errors = append(errors, strategy.ValidationError{
				Field:    field,
				Code:     "REQUIRED",
				Message:  field + " is required",
				Severity: strategy.ValidationSeverityError,
			})
		}
	}

	return errors, nil
}
