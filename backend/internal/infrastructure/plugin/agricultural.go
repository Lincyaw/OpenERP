package plugin

import (
	"context"
	"regexp"
	"strings"

	"github.com/erp/backend/internal/domain/shared/plugin"
	"github.com/erp/backend/internal/domain/shared/strategy"
)

// AgriculturalPlugin is the reference implementation for agricultural industry
// It provides specific validation and batch management for agricultural products
type AgriculturalPlugin struct{}

// NewAgriculturalPlugin creates a new agricultural industry plugin
func NewAgriculturalPlugin() *AgriculturalPlugin {
	return &AgriculturalPlugin{}
}

// Name returns the unique identifier for the plugin
func (p *AgriculturalPlugin) Name() string {
	return "agricultural"
}

// DisplayName returns the human-readable name for the plugin
func (p *AgriculturalPlugin) DisplayName() string {
	return "农资行业"
}

// RegisterStrategies registers agricultural-specific strategies with the registry
func (p *AgriculturalPlugin) RegisterStrategies(registry plugin.StrategyRegistrar) {
	// Register agricultural product validation strategy
	validator := NewAgriculturalProductValidator()
	_ = registry.RegisterValidationStrategy(validator)
}

// GetRequiredProductAttributes returns the attribute definitions for agricultural products
func (p *AgriculturalPlugin) GetRequiredProductAttributes() []plugin.AttributeDefinition {
	return []plugin.AttributeDefinition{
		{
			Key:           "registration_number",
			Label:         "农药登记证号",
			Required:      false, // Only required for pesticide category
			Regex:         `^PD\d{8}$`,
			CategoryCodes: []string{"PESTICIDE"},
		},
		{
			Key:           "variety_approval_number",
			Label:         "品种审定编号",
			Required:      false, // Only required for seed category
			CategoryCodes: []string{"SEED"},
		},
		{
			Key:           "manufacturer",
			Label:         "生产厂家",
			Required:      true, // Required for all agricultural products
			CategoryCodes: []string{},
		},
		{
			Key:           "production_license",
			Label:         "生产许可证号",
			Required:      false,
			CategoryCodes: []string{"PESTICIDE", "FERTILIZER"},
		},
		{
			Key:           "validity_period",
			Label:         "有效期(月)",
			Required:      false,
			CategoryCodes: []string{"PESTICIDE"},
		},
	}
}

// AgriculturalProductValidator validates products for agricultural industry
type AgriculturalProductValidator struct {
	strategy.BaseStrategy
	registrationRegex *regexp.Regexp
}

// NewAgriculturalProductValidator creates a new agricultural product validator
func NewAgriculturalProductValidator() *AgriculturalProductValidator {
	return &AgriculturalProductValidator{
		BaseStrategy: strategy.NewBaseStrategy(
			"agricultural",
			strategy.StrategyTypeValidation,
			"Agricultural industry product validation with pesticide registration requirements",
		),
		registrationRegex: regexp.MustCompile(`^PD\d{8}$`),
	}
}

// Validate validates agricultural product data
func (v *AgriculturalProductValidator) Validate(
	ctx context.Context,
	valCtx strategy.ValidationContext,
	data strategy.ProductData,
) (strategy.ValidationResult, error) {
	result := strategy.ValidationResult{
		IsValid:  true,
		Errors:   make([]strategy.ValidationError, 0),
		Warnings: make([]strategy.ValidationWarning, 0),
	}

	// Validate basic required fields (same as standard validator)
	if strings.TrimSpace(data.SKU) == "" {
		result.AddError("sku", "REQUIRED", "SKU is required")
	}

	if strings.TrimSpace(data.Name) == "" {
		result.AddError("name", "REQUIRED", "Product name is required")
	}

	if strings.TrimSpace(data.CategoryID) == "" {
		result.AddError("categoryId", "REQUIRED", "Category is required")
	}

	// Validate price and cost
	if data.Price.IsNegative() {
		result.AddError("price", "INVALID", "Price cannot be negative")
	}

	if data.Cost.IsNegative() {
		result.AddError("cost", "INVALID", "Cost cannot be negative")
	}

	// Get category code from attributes for industry-specific validation
	categoryCode := ""
	if data.Attributes != nil {
		if code, ok := data.Attributes["category_code"].(string); ok {
			categoryCode = code
		}
	}

	// Validate manufacturer (required for all agricultural products)
	manufacturer := v.getAttributeString(data.Attributes, "manufacturer")
	if strings.TrimSpace(manufacturer) == "" {
		result.AddError("attributes.manufacturer", "REQUIRED", "生产厂家是必填项")
	}

	// Pesticide-specific validations
	if categoryCode == "PESTICIDE" {
		regNum := v.getAttributeString(data.Attributes, "registration_number")
		if strings.TrimSpace(regNum) == "" {
			result.AddError("attributes.registration_number", "REQUIRED", "农药产品必须填写农药登记证号")
		} else if !v.registrationRegex.MatchString(regNum) {
			result.AddError("attributes.registration_number", "INVALID_FORMAT", "农药登记证号格式不正确，应为PD+8位数字")
		}

		// Recommend setting validity period for pesticides
		if v.getAttributeString(data.Attributes, "validity_period") == "" {
			result.AddWarning("attributes.validity_period", "MISSING", "建议填写农药有效期以便批次管理")
		}
	}

	// Seed-specific validations
	if categoryCode == "SEED" {
		approvalNum := v.getAttributeString(data.Attributes, "variety_approval_number")
		if strings.TrimSpace(approvalNum) == "" {
			result.AddWarning("attributes.variety_approval_number", "MISSING", "种子产品建议填写品种审定编号")
		}
	}

	// Fertilizer-specific validations
	if categoryCode == "FERTILIZER" {
		license := v.getAttributeString(data.Attributes, "production_license")
		if strings.TrimSpace(license) == "" {
			result.AddWarning("attributes.production_license", "MISSING", "肥料产品建议填写生产许可证号")
		}
	}

	return result, nil
}

// ValidateField validates a single field
func (v *AgriculturalProductValidator) ValidateField(
	ctx context.Context,
	field string,
	value any,
) ([]strategy.ValidationError, error) {
	errors := make([]strategy.ValidationError, 0)

	switch field {
	case "sku", "name", "categoryId":
		if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
			errors = append(errors, strategy.ValidationError{
				Field:    field,
				Code:     "REQUIRED",
				Message:  field + " is required",
				Severity: strategy.ValidationSeverityError,
			})
		}
	case "attributes.registration_number":
		if str, ok := value.(string); ok {
			if strings.TrimSpace(str) != "" && !v.registrationRegex.MatchString(str) {
				errors = append(errors, strategy.ValidationError{
					Field:    field,
					Code:     "INVALID_FORMAT",
					Message:  "农药登记证号格式不正确，应为PD+8位数字",
					Severity: strategy.ValidationSeverityError,
				})
			}
		}
	case "attributes.manufacturer":
		if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
			errors = append(errors, strategy.ValidationError{
				Field:    field,
				Code:     "REQUIRED",
				Message:  "生产厂家是必填项",
				Severity: strategy.ValidationSeverityError,
			})
		}
	}

	return errors, nil
}

// getAttributeString safely gets a string attribute from the attributes map
func (v *AgriculturalProductValidator) getAttributeString(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	if val, ok := attrs[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Ensure AgriculturalPlugin implements IndustryPlugin interface
var _ plugin.IndustryPlugin = (*AgriculturalPlugin)(nil)

// Ensure AgriculturalProductValidator implements ProductValidationStrategy interface
var _ strategy.ProductValidationStrategy = (*AgriculturalProductValidator)(nil)
