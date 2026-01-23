package strategy

import (
	"context"

	"github.com/shopspring/decimal"
)

// ValidationSeverity represents the severity of a validation issue
type ValidationSeverity string

const (
	ValidationSeverityError   ValidationSeverity = "error"
	ValidationSeverityWarning ValidationSeverity = "warning"
	ValidationSeverityInfo    ValidationSeverity = "info"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field    string
	Code     string
	Message  string
	Severity ValidationSeverity
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string
	Code    string
	Message string
}

// ProductData represents product data for validation
type ProductData struct {
	ID           string
	TenantID     string
	SKU          string
	Name         string
	CategoryID   string
	UnitID       string
	Price        decimal.Decimal
	Cost         decimal.Decimal
	MinStock     decimal.Decimal
	MaxStock     decimal.Decimal
	ReorderPoint decimal.Decimal
	IsActive     bool
	Attributes   map[string]any
}

// ValidationContext provides context for product validation
type ValidationContext struct {
	TenantID     string
	IsNew        bool // True if creating new product, false if updating
	ExistingData *ProductData
}

// ValidationResult contains the result of validation
type ValidationResult struct {
	IsValid  bool
	Errors   []ValidationError
	Warnings []ValidationWarning
}

// AddError adds an error to the validation result
func (r *ValidationResult) AddError(field, code, message string) {
	r.Errors = append(r.Errors, ValidationError{
		Field:    field,
		Code:     code,
		Message:  message,
		Severity: ValidationSeverityError,
	})
	r.IsValid = false
}

// AddWarning adds a warning to the validation result
func (r *ValidationResult) AddWarning(field, code, message string) {
	r.Warnings = append(r.Warnings, ValidationWarning{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

// ProductValidationStrategy defines the interface for product validation
type ProductValidationStrategy interface {
	Strategy
	// Validate validates the product data
	Validate(ctx context.Context, valCtx ValidationContext, data ProductData) (ValidationResult, error)
	// ValidateField validates a single field
	ValidateField(ctx context.Context, field string, value any) ([]ValidationError, error)
}
