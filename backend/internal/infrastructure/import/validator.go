package csvimport

import (
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// FieldType represents the expected type of a field
type FieldType string

const (
	TypeString  FieldType = "string"
	TypeInt     FieldType = "int"
	TypeDecimal FieldType = "decimal"
	TypeDate    FieldType = "date"
	TypeEmail   FieldType = "email"
	TypeBool    FieldType = "bool"
	TypeUUID    FieldType = "uuid"
)

// FieldRule defines validation rules for a field
type FieldRule struct {
	Column      string
	Type        FieldType
	Required    bool
	MinLength   int
	MaxLength   int
	MinValue    *decimal.Decimal
	MaxValue    *decimal.Decimal
	Pattern     *regexp.Regexp
	PatternDesc string
	DateFormat  string
	Unique      bool
	Reference   string // Foreign key reference type (e.g., "category", "supplier")
	CustomFunc  func(value string) error
}

// FieldRuleBuilder helps build field rules fluently
type FieldRuleBuilder struct {
	rule FieldRule
}

// Field creates a new field rule builder
func Field(column string) *FieldRuleBuilder {
	return &FieldRuleBuilder{
		rule: FieldRule{
			Column:     column,
			Type:       TypeString,
			DateFormat: "2006-01-02", // Default date format
		},
	}
}

// Required marks the field as required
func (b *FieldRuleBuilder) Required() *FieldRuleBuilder {
	b.rule.Required = true
	return b
}

// String sets the field type to string
func (b *FieldRuleBuilder) String() *FieldRuleBuilder {
	b.rule.Type = TypeString
	return b
}

// Int sets the field type to integer
func (b *FieldRuleBuilder) Int() *FieldRuleBuilder {
	b.rule.Type = TypeInt
	return b
}

// Decimal sets the field type to decimal
func (b *FieldRuleBuilder) Decimal() *FieldRuleBuilder {
	b.rule.Type = TypeDecimal
	return b
}

// Date sets the field type to date
func (b *FieldRuleBuilder) Date() *FieldRuleBuilder {
	b.rule.Type = TypeDate
	return b
}

// DateFormat sets the expected date format
func (b *FieldRuleBuilder) DateFormat(format string) *FieldRuleBuilder {
	b.rule.DateFormat = format
	return b
}

// Email sets the field type to email
func (b *FieldRuleBuilder) Email() *FieldRuleBuilder {
	b.rule.Type = TypeEmail
	return b
}

// Bool sets the field type to boolean
func (b *FieldRuleBuilder) Bool() *FieldRuleBuilder {
	b.rule.Type = TypeBool
	return b
}

// UUID sets the field type to UUID
func (b *FieldRuleBuilder) UUID() *FieldRuleBuilder {
	b.rule.Type = TypeUUID
	return b
}

// MinLength sets the minimum length
func (b *FieldRuleBuilder) MinLength(n int) *FieldRuleBuilder {
	b.rule.MinLength = n
	return b
}

// MaxLength sets the maximum length
func (b *FieldRuleBuilder) MaxLength(n int) *FieldRuleBuilder {
	b.rule.MaxLength = n
	return b
}

// Length sets both min and max length to the same value
func (b *FieldRuleBuilder) Length(min, max int) *FieldRuleBuilder {
	b.rule.MinLength = min
	b.rule.MaxLength = max
	return b
}

// MinValue sets the minimum numeric value
func (b *FieldRuleBuilder) MinValue(v decimal.Decimal) *FieldRuleBuilder {
	b.rule.MinValue = &v
	return b
}

// MaxValue sets the maximum numeric value
func (b *FieldRuleBuilder) MaxValue(v decimal.Decimal) *FieldRuleBuilder {
	b.rule.MaxValue = &v
	return b
}

// Range sets both min and max values
func (b *FieldRuleBuilder) Range(min, max decimal.Decimal) *FieldRuleBuilder {
	b.rule.MinValue = &min
	b.rule.MaxValue = &max
	return b
}

// Pattern sets a regex pattern for validation
func (b *FieldRuleBuilder) Pattern(pattern, description string) *FieldRuleBuilder {
	b.rule.Pattern = regexp.MustCompile(pattern)
	b.rule.PatternDesc = description
	return b
}

// Unique marks the field as unique (within file)
func (b *FieldRuleBuilder) Unique() *FieldRuleBuilder {
	b.rule.Unique = true
	return b
}

// Reference sets the foreign key reference type
func (b *FieldRuleBuilder) Reference(refType string) *FieldRuleBuilder {
	b.rule.Reference = refType
	return b
}

// Custom sets a custom validation function
func (b *FieldRuleBuilder) Custom(fn func(value string) error) *FieldRuleBuilder {
	b.rule.CustomFunc = fn
	return b
}

// Build returns the built field rule
func (b *FieldRuleBuilder) Build() FieldRule {
	return b.rule
}

// FieldValidator validates fields according to rules
type FieldValidator struct {
	rules       map[string]FieldRule
	uniqueCheck map[string]map[string]int // column -> value -> first row number
	errors      *ErrorCollection
}

// NewFieldValidator creates a new field validator
func NewFieldValidator(rules []FieldRule, maxErrors int) *FieldValidator {
	ruleMap := make(map[string]FieldRule, len(rules))
	for _, r := range rules {
		ruleMap[r.Column] = r
	}

	return &FieldValidator{
		rules:       ruleMap,
		uniqueCheck: make(map[string]map[string]int),
		errors:      NewErrorCollection(maxErrors),
	}
}

// ValidateRow validates all fields in a row
func (v *FieldValidator) ValidateRow(row *Row) bool {
	hasError := false

	for column, rule := range v.rules {
		value := row.Get(column)

		// Required check
		if rule.Required && value == "" {
			v.errors.AddRequiredError(row.LineNumber, column)
			hasError = true
			continue
		}

		// Skip further validation for empty optional fields
		if value == "" {
			continue
		}

		// Type validation
		if err := v.validateType(value, rule.Type, rule.DateFormat); err != nil {
			v.errors.AddTypeError(row.LineNumber, column, string(rule.Type), value)
			hasError = true
			continue
		}

		// Length validation
		if rule.MaxLength > 0 && len(value) > rule.MaxLength {
			v.errors.AddLengthError(row.LineNumber, column, rule.MinLength, rule.MaxLength)
			hasError = true
		}
		if rule.MinLength > 0 && len(value) < rule.MinLength {
			v.errors.AddLengthError(row.LineNumber, column, rule.MinLength, rule.MaxLength)
			hasError = true
		}

		// Range validation for numeric types
		if rule.Type == TypeInt || rule.Type == TypeDecimal {
			if err := v.validateRange(value, rule.MinValue, rule.MaxValue); err != nil {
				if rule.MinValue != nil && rule.MaxValue != nil {
					minFloat, _ := rule.MinValue.Float64()
					maxFloat, _ := rule.MaxValue.Float64()
					v.errors.AddRangeError(row.LineNumber, column, minFloat, maxFloat)
				}
				hasError = true
			}
		}

		// Pattern validation
		if rule.Pattern != nil && !rule.Pattern.MatchString(value) {
			v.errors.AddPatternError(row.LineNumber, column, rule.PatternDesc, value)
			hasError = true
		}

		// Uniqueness check within file
		if rule.Unique {
			if v.uniqueCheck[column] == nil {
				v.uniqueCheck[column] = make(map[string]int)
			}
			if firstRow, exists := v.uniqueCheck[column][value]; exists {
				v.errors.Add(NewRowErrorWithValue(row.LineNumber, column, ErrCodeImportDuplicateInFile,
					fmt.Sprintf("duplicate value '%s' (first seen in row %d)", value, firstRow), value))
				hasError = true
			} else {
				v.uniqueCheck[column][value] = row.LineNumber
			}
		}

		// Custom validation
		if rule.CustomFunc != nil {
			if err := rule.CustomFunc(value); err != nil {
				v.errors.AddValidationError(row.LineNumber, column, ErrCodeImportValidation, err.Error())
				hasError = true
			}
		}
	}

	return !hasError
}

// validateType validates a value against expected type
func (v *FieldValidator) validateType(value string, fieldType FieldType, dateFormat string) error {
	switch fieldType {
	case TypeString:
		return nil
	case TypeInt:
		_, err := strconv.ParseInt(value, 10, 64)
		return err
	case TypeDecimal:
		_, err := decimal.NewFromString(value)
		return err
	case TypeDate:
		_, err := time.Parse(dateFormat, value)
		return err
	case TypeEmail:
		_, err := mail.ParseAddress(value)
		return err
	case TypeBool:
		lower := strings.ToLower(value)
		if lower == "true" || lower == "false" || lower == "1" || lower == "0" ||
			lower == "yes" || lower == "no" || lower == "y" || lower == "n" {
			return nil
		}
		return fmt.Errorf("invalid boolean value: %s", value)
	case TypeUUID:
		return validateUUID(value)
	}
	return nil
}

// validateUUID validates a UUID string
func validateUUID(s string) error {
	if len(s) != 36 {
		return fmt.Errorf("invalid UUID length")
	}
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return fmt.Errorf("invalid UUID format")
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("invalid UUID character")
		}
	}
	return nil
}

// validateRange validates numeric value against min/max
func (v *FieldValidator) validateRange(value string, min, max *decimal.Decimal) error {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return err
	}

	if min != nil && d.LessThan(*min) {
		return fmt.Errorf("value %s is less than minimum %s", value, min.String())
	}
	if max != nil && d.GreaterThan(*max) {
		return fmt.Errorf("value %s is greater than maximum %s", value, max.String())
	}
	return nil
}

// Errors returns the error collection
func (v *FieldValidator) Errors() *ErrorCollection {
	return v.errors
}

// Reset clears the validator state for reuse
func (v *FieldValidator) Reset() {
	v.uniqueCheck = make(map[string]map[string]int)
	v.errors.Clear()
}

// ReferenceValidator validates foreign key references
type ReferenceValidator struct {
	cache      map[string]map[string]bool // refType -> value -> exists
	lookupFunc func(refType, value string) (bool, error)
	errors     *ErrorCollection
}

// NewReferenceValidator creates a new reference validator
func NewReferenceValidator(lookupFunc func(refType, value string) (bool, error), maxErrors int) *ReferenceValidator {
	return &ReferenceValidator{
		cache:      make(map[string]map[string]bool),
		lookupFunc: lookupFunc,
		errors:     NewErrorCollection(maxErrors),
	}
}

// PreloadReferences preloads reference data for efficiency
func (v *ReferenceValidator) PreloadReferences(refType string, values []string) error {
	if v.cache[refType] == nil {
		v.cache[refType] = make(map[string]bool)
	}

	for _, value := range values {
		exists, err := v.lookupFunc(refType, value)
		if err != nil {
			return err
		}
		v.cache[refType][value] = exists
	}

	return nil
}

// ValidateReference validates a single reference
func (v *ReferenceValidator) ValidateReference(row int, column, refType, value string) bool {
	if value == "" {
		return true
	}

	// Check cache first
	if v.cache[refType] != nil {
		if exists, ok := v.cache[refType][value]; ok {
			if !exists {
				v.errors.AddReferenceError(row, column, value, refType)
				return false
			}
			return true
		}
	}

	// Lookup if not cached
	exists, err := v.lookupFunc(refType, value)
	if err != nil {
		v.errors.AddValidationError(row, column, ErrCodeImportValidation,
			fmt.Sprintf("error checking %s reference: %v", refType, err))
		return false
	}

	// Cache the result
	if v.cache[refType] == nil {
		v.cache[refType] = make(map[string]bool)
	}
	v.cache[refType][value] = exists

	if !exists {
		v.errors.AddReferenceError(row, column, value, refType)
		return false
	}

	return true
}

// Errors returns the error collection
func (v *ReferenceValidator) Errors() *ErrorCollection {
	return v.errors
}

// Reset clears the validator cache
func (v *ReferenceValidator) Reset() {
	v.cache = make(map[string]map[string]bool)
	v.errors.Clear()
}

// UniquenessValidator validates uniqueness constraints against the database
type UniquenessValidator struct {
	lookupFunc func(entityType, field, value string) (bool, error)
	errors     *ErrorCollection
}

// NewUniquenessValidator creates a new uniqueness validator
func NewUniquenessValidator(lookupFunc func(entityType, field, value string) (bool, error), maxErrors int) *UniquenessValidator {
	return &UniquenessValidator{
		lookupFunc: lookupFunc,
		errors:     NewErrorCollection(maxErrors),
	}
}

// ValidateUnique checks if a value is unique in the database
func (v *UniquenessValidator) ValidateUnique(row int, column, entityType, value string) bool {
	if value == "" {
		return true
	}

	exists, err := v.lookupFunc(entityType, column, value)
	if err != nil {
		v.errors.AddValidationError(row, column, ErrCodeImportValidation,
			fmt.Sprintf("error checking uniqueness: %v", err))
		return false
	}

	if exists {
		v.errors.AddDuplicateError(row, column, value, true)
		return false
	}

	return true
}

// Errors returns the error collection
func (v *UniquenessValidator) Errors() *ErrorCollection {
	return v.errors
}
